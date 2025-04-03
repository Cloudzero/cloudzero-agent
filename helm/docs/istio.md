# Installing `cloudzero-agent` in Istio-Enabled Clusters

When installing the `cloudzero-agent` Helm chart in a Kubernetes cluster that uses Istio, additional steps may be required to ensure proper functionality. This is because Istio automatically configures sidecars to use **mutual TLS (mTLS)**, which can interfere with the agent’s existing TLS communication.

## Why These Steps Are Needed

The `cloudzero-agent` includes a **webhook server** component responsible for handling admission review requests from the Kubernetes API server. These requests use TLS, and when intercepted by an Istio sidecar, Istio may attempt to apply its mTLS policies. These policies are not always compatible with the webhook’s TLS configuration.

While this does not block pod deployments, it **prevents the `insightsController` from collecting critical pod labels**, which are necessary for accurate cost allocation.

To ensure `cloudzero-agent` works correctly in Istio-enabled clusters, you can choose from the following options:

- [**Disable sidecar injection for `cloudzero-agent` webhook-server pods only**](#option-1-disable-sidecar-injection-for-cloudzero-agent-webhook-server-pods-only) — Completely removes the Istio sidecar from the webhook server, avoiding any interference.
- [**Disable envoy for webhook ports only**](#option-2-disable-envoy-for-webhook-ports-only) — Keeps the sidecar but excludes webhook traffic, preserving Istio functionality for all other traffic.
- [**Disable mTLS for `cloudzero-agent` webhook-server pods**](#option-3-disable-mtls-for-cloudzero-agent) — Keeps the sidecar but disables mTLS enforcement specifically for webhook-server traffic.

---

## **Option 1: Disable Sidecar Injection for `cloudzero-agent` webhook-server Pods Only**

To disable the sidecar injection **only for a subset of the pods deployed by the cluster**, update the chart input values with the following annotation:

```yaml
insightsController:
  server:
    podAnnotations:
      sidecar.istio.io/inject: "false"
```

This annotation prevents the Istio sidecar from being injected into the `webhook-server` pods, while the rest of the cluster retains normal Istio functionality.

---

## **Option 2: Disable Envoy for Webhook Ports Only**

To prevent only requests to a single port on the webhook-server pods from being routed through envoy, apply the following annotation:

```yaml
insightsController:
  server:
    podAnnotations:
      traffic.sidecar.istio.io/excludeInboundPorts: "8443"
```

In this case, the pods will still have an Istio sidecar injected, but traffic to port 8443 (the webhook port) will bypass envoy.

For more details, see [Istio Documentation](https://istio.io/latest/docs/reference/config/annotations/#SidecarTrafficExcludeInboundPorts).

---

## **Option 3: Disable mTLS for `cloudzero-agent`**

To disable mTLS for the `cloudzero-agent` service, apply the following `PeerAuthentication` resource:

```yaml
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: cloudzero-agent-mtls
  namespace: <cloudzero-namespace>
spec:
  selector:
    matchLabels:
      app.kubernetes.io/component: webhook-server
  mtls:
    mode: DISABLE
```

### **Steps to Apply:**

1. Replace `<your-namespace>` with the namespace where `cloudzero-agent` is deployed.
2. Apply the resource:
   ```sh
   kubectl apply -f cloudzero-agent-mtls.yaml
   ```
3. Deploy the `cloudzero-agent` chart as instructed in the chart README.md

This configuration **disables mTLS for `cloudzero-agent` webhook-server pods only**, while keeping it enabled for the rest of the cluster.

---
