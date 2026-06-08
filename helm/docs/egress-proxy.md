# Routing Agent Traffic Through an HTTP(S) Proxy

Some Kubernetes clusters cannot reach the public internet directly —
egress goes through a forward HTTP(S) proxy, usually paired with a
firewall that denies everything else. To run the CloudZero Agent in
such a cluster, you have to tell each of the agent's containers:

1. **The proxy to use** for traffic destined for `api.cloudzero.com`.
2. **Which destinations to bypass** the proxy for: in-cluster Services,
   the kube-apiserver, the pod CIDR Prometheus scrapes against, cloud
   instance metadata, and so on.

The chart's `defaults.env` value injects environment variables into
every chart-managed container. This document covers how to use it for
proxy configuration. The same mechanism works for any other env you
need to push to every container.

> [!IMPORTANT]
> The chart cannot auto-discover your proxy URL, pod CIDR, service
> CIDR, or apiserver IP — those vary per cluster. You have to inspect
> your own cluster to assemble `NO_PROXY`. Discovery commands are in
> the [Finding your cluster's values](#finding-your-clusters-values)
> section.

## How the chart picks up proxy settings

Every binary the chart ships is built in Go (this includes the bundled
Prometheus, taken unmodified from upstream releases, and Alloy, our
fork of Grafana Alloy). All of them honor three environment variables
via Go's standard
[`http.ProxyFromEnvironment`](https://pkg.go.dev/net/http#ProxyFromEnvironment):

| Variable      | Effect                                                    |
| ------------- | --------------------------------------------------------- |
| `HTTPS_PROXY` | Proxy URL for HTTPS destinations.                         |
| `HTTP_PROXY`  | Proxy URL for HTTP destinations.                          |
| `NO_PROXY`    | Comma-separated list of destinations to bypass the proxy. |

There is no separate "proxy URL" knob in the chart — setting these
three env vars is the entire interface.

To set them, list them under `defaults.env`:

```yaml
defaults:
  env:
    - name: HTTPS_PROXY
      value: "http://proxy.example.com:8080"
    - name: HTTP_PROXY
      value: "http://proxy.example.com:8080"
    - name: NO_PROXY
      value: "localhost,127.0.0.1,169.254.169.254,cluster.local,10.0.0.0/8"
```

### Precedence

`defaults.env` is the **lowest**-priority env source. The chart's
[`generateEnv`](../templates/_helpers.tpl) helper iterates sources in
order, and later sources overwrite earlier ones by `name`. The order
used at every call site is:

1. `.Values.defaults.env` (this value)
2. `.Values.server.env` (Prometheus-only)
3. Validator-lifecycle env (`K8S_NAMESPACE`, `K8S_POD_NAME`, `ISTIO_*`,
   etc.)
4. Hardcoded literals (`HOSTNAME`, `NODE_NAME`, `SERVER_PORT`,
   fieldRefs)

In practice you will never collide with `HTTPS_PROXY` / `HTTP_PROXY` /
`NO_PROXY` — the chart never sets those itself — so the precedence
question only matters if you try to use `defaults.env` to override a
chart-emitted name. Don't.

## NO_PROXY syntax

`NO_PROXY` is parsed by
[`golang.org/x/net/http/httpproxy`](https://pkg.go.dev/golang.org/x/net/http/httpproxy#Config),
which `net/http` delegates to. The package's
[`Config.NoProxy` doc comment](https://pkg.go.dev/golang.org/x/net/http/httpproxy#Config)
is the authoritative reference. The notable points:

- **Comma-separated.** Whitespace around values is stripped — `"a, b"`
  works the same as `"a,b"`.
- **Case-insensitive.** Both the request host and each entry are
  lowercased before matching.
- **Plain hostname** (`foo.com`) matches the bare domain **and** all
  subdomains (`foo.com`, `bar.foo.com`, `a.b.foo.com`).
- **Leading-dot hostname** (`.foo.com`) matches subdomains **only**, not
  the bare domain.
- **`*.foo.com`** is normalized to `.foo.com` — same semantics as the
  leading-dot form.
- **CIDR blocks** (`10.0.0.0/16`, `2001:db8::/64`) match any IP in
  range.
- **Single IPs** match exactly.
- **`:port` suffix** on an entry restricts it to that port; an entry
  without a port matches any port.
- **`*` alone** disables the proxy entirely.

Note in particular that `cluster.local` covers `foo.svc.cluster.local`,
`bar.cluster.local`, and the bare `cluster.local` — you don't need to
list `.svc.cluster.local` or `.svc` separately. (`.svc` on its own
would only match strings ending in `.svc`, e.g. `foo.svc`, which is
not what cluster DNS produces.)

## What you have to cover

The agent's components talk to several categories of destination.
Anything not in `NO_PROXY` goes through the proxy — which usually means
either an outright failure (the proxy rejects in-cluster destinations)
or an expensive hairpin out to your egress edge and back.

### Loopback

```text
localhost,127.0.0.1
```

Containers talk to themselves for health checks and pprof endpoints.

### Cluster DNS suffix

```text
cluster.local
```

The aggregator, webhook server, and Prometheus federation endpoint are
reached by Service DNS names like
`cloudzero-agent-server.cloudzero-agent.svc.cluster.local`. The bare
`cluster.local` entry suffix-matches all of those.

If your cluster uses a non-default DNS domain (configured via
`--cluster-domain` on kubelet), use that instead.

### kube-apiserver ClusterIP

```text
10.96.0.1
```

The agent calls the API server for pod, node, and namespace metadata.
`kubernetes.default.svc.cluster.local` is covered by the entry above,
but the apiserver's **ClusterIP is a routed IP, not a name** — and
client libraries often connect by IP. List it explicitly.

The default ClusterIP for kubeadm clusters is `10.96.0.1`, but
cloud-managed clusters use different ranges (`172.20.0.1` is common on
EKS, GKE uses an IP from `34.118.224.0/20` on this author's test
cluster, etc.).

### Pod CIDR (critical for Prometheus)

```text
10.244.0.0/16
```

This is the one that bites. Prometheus's
[`endpointslice` SD](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#kubernetes_sd_config)
resolves each target to its **pod IP**, not the Service DNS name. So
even with `cluster.local` in `NO_PROXY`, scrape requests still go to
addresses like `10.244.5.32:9090`.

Add your cluster's pod CIDR.

### Service CIDR

```text
10.96.0.0/12
```

Some clients address Services via their ClusterIP directly. Listing
the whole service CIDR makes those bypass the proxy and also covers
the apiserver ClusterIP, so you can drop the explicit apiserver entry
if you list the full range.

### Cloud instance metadata

```text
169.254.169.254,metadata.google.internal
```

The agent's
[Scout](https://github.com/Cloudzero/cloudzero-agent/tree/develop/app/utils/scout)
component queries cloud-provider metadata to identify the cluster
(account/project/subscription ID, region). On AWS, Azure, and GCP the
endpoint is link-local at `169.254.169.254`. GCP also accepts the DNS
name `metadata.google.internal` (which resolves to the same address).

If `169.254.169.254` goes through the proxy, the metadata call will
hang or 4xx and the validator will refuse to start past the
"cloud provider detection failed" stage unless your values set the
relevant identity fields explicitly.

## Finding your cluster's values

The values above are placeholders — you have to look up the real
numbers. The commands below were tested against a GKE test cluster
unless otherwise noted.

### apiserver ClusterIP

```sh
kubectl get svc kubernetes -n default -o jsonpath='{.spec.clusterIP}'
```

Returns a single IP.

### Pod CIDR (cluster-wide)

`kubectl get nodes -o jsonpath='{.items[*].spec.podCIDR}'` returns the
**per-node** slice (a `/24` per node on most platforms), not the
cluster-wide parent. For the parent CIDR, prefer your cloud provider's
CLI:

```sh
# GKE (works for both zonal and regional clusters via --location):
gcloud container clusters describe CLUSTER --location LOCATION \
  --format='value(clusterIpv4Cidr,servicesIpv4Cidr)'

# EKS (pod IPs come from VPC subnets when using the AWS VPC CNI):
aws eks describe-cluster --name CLUSTER \
  --query 'cluster.{podCidr:resourcesVpcConfig.cidrs,svcCidr:kubernetesNetworkConfig.serviceIpv4Cidr}'

# AKS:
az aks show -n CLUSTER -g RESOURCE_GROUP \
  --query 'networkProfile.{podCidr:podCidr,svcCidr:serviceCidr}' -o tsv
```

On many self-managed clusters the kube-proxy `--cluster-cidr` flag is
visible in `kubectl cluster-info dump`:

```sh
kubectl cluster-info dump | grep -m1 cluster-cidr
```

The cloud CLI invocations above were not exhaustively tested across
account types — confirm against your cloud-provider documentation if
the values look off.

### Service CIDR (via cloud CLI)

The cloud CLIs above also return the service CIDR. On clusters where
the control plane is visible to `kubectl cluster-info dump`,
`grep service-cluster-ip-range` will pull it out; on cloud-managed
clusters the control plane is hidden and that doesn't work, so use the
cloud CLI.

## Worked examples

These are **starting points**, not finished configurations. The CIDRs
came from this author's recollection of typical defaults — verify each
value against your cluster before deploying.

### AWS EKS

```yaml
defaults:
  env:
    - name: HTTPS_PROXY
      value: "http://proxy.internal.example.com:3128"
    - name: HTTP_PROXY
      value: "http://proxy.internal.example.com:3128"
    - name: NO_PROXY
      value: "localhost,127.0.0.1,169.254.169.254,cluster.local,10.0.0.0/8,172.20.0.0/16"
```

`10.0.0.0/8` is a typical VPC CIDR (EKS pod IPs live in your VPC under
the AWS VPC CNI). `172.20.0.0/16` is the EKS default service CIDR.
Both vary by cluster.

### GKE

```yaml
defaults:
  env:
    - name: HTTPS_PROXY
      value: "http://proxy.internal.example.com:3128"
    - name: HTTP_PROXY
      value: "http://proxy.internal.example.com:3128"
    - name: NO_PROXY
      value: "localhost,127.0.0.1,169.254.169.254,metadata.google.internal,cluster.local,10.0.0.0/14,10.4.0.0/19"
```

`10.0.0.0/14` is the GKE default pod CIDR, `10.4.0.0/19` the default
service CIDR. Confirm with `gcloud container clusters describe`.

### Azure AKS

```yaml
defaults:
  env:
    - name: HTTPS_PROXY
      value: "http://proxy.internal.example.com:3128"
    - name: HTTP_PROXY
      value: "http://proxy.internal.example.com:3128"
    - name: NO_PROXY
      value: "localhost,127.0.0.1,169.254.169.254,cluster.local,10.244.0.0/16,10.0.0.0/16"
```

`10.244.0.0/16` is the AKS kubenet pod CIDR. With Azure CNI the pod
CIDR is your subnet, often much smaller than `/16`. `10.0.0.0/16` is
a typical service CIDR default.

## Validating the configuration

After deploying, spot-check that the env vars reached the containers:

```sh
kubectl exec -n cloudzero-agent deploy/cloudzero-agent-server \
  -c server -- env | grep -i proxy
```

…and, if you control the proxy, watch its access log while the agent
starts. Expected behavior:

- Calls to `api.cloudzero.com` (and `app.cloudzero.com` for Replicated
  installs) appear in the proxy log.
- Calls to your apiserver, in-cluster Services, pod IPs, and the
  metadata endpoint do **not** appear.

If you see in-cluster destinations in the proxy log, your `NO_PROXY`
is incomplete — find the missing entry and reinstall.
