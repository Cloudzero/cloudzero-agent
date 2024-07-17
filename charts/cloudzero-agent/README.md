# Cloudzero Agent Helm Chart

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE-OF-CONDUCT.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
![GitHub release](https://img.shields.io/github/release/Cloudzero/cloudzero-charts.svg)

A Helm chart for deploying Prometheus in agent mode to send cluster metrics to the CloudZero platform.

## Prerequisites

- Kubernetes 1.23+
- Helm 3+
- A CloudZero API key

## Installation

### Get Helm Repository Info

```console
helm repo add cloudzero https://cloudzero.github.io/cloudzero-charts
helm repo update
```

_See [`helm repo`](https://helm.sh/docs/helm/helm_repo/) for command documentation._

### Install Helm Chart

The chart can be installed directly with Helm or any other common Kubernetes deployment tools.

If installing with Helm directly, the following command will install the chart:

```console
helm install <RELEASE_NAME> cloudzero/cloudzero-agent \
    --set existingSecretName=<NAME_OF_SECRET> \
    --set clusterName=<CLUSTER_NAME> \
    --set cloudAccountId=<CLOUD_ACCOUNT_ID> \
    --set region=<REGION>
```

Alternatively if you are updating an existing installation, you can upgrade the chart with:

```console
helm upgrade <RELEASE_NAME> cloudzero/cloudzero-agent \
    --set existingSecretName=<NAME_OF_SECRET> \
    --set clusterName=<CLUSTER_NAME> \
    --set cloudAccountId=<CLOUD_ACCOUNT_ID> \
    --set region=<REGION>
```

### Mandatory Values

There are several mandatory values that must be specified for the chart to install properly. Below are the required settings along with strategies for providing custom values during installation:

| Key               | Type   | Default               | Description                                                                                                             |
|-------------------|--------|-----------------------|-------------------------------------------------------------------------------------------------------------------------|
| cloudAccountId    | string | `nil`                 | Account ID in AWS or Subscription ID in Azure where the cluster is running. Must be a string due to Helm limitations.  |
| clusterName       | string | `nil`                 | Name of the cluster. Must be RFC 1123 compliant.                                                                         |
| host              | string | `"api.cloudzero.com"` | CloudZero host to send metrics to.                                                                                      |
| apiKey            | string | `nil`                 | The CloudZero API key to use for exporting metrics. Only used if `existingSecretName` is not set.                       |
| existingSecretName| string | `nil`                 | Name of the secret that contains the CloudZero API key. Required if not providing the API key via `apiKey`.             |
| region            | string | `nil`                 | Region where the cluster is running (e.g., `us-east-1`, `eastus`). For more information, see AWS or Azure documentation. |

#### Overriding Default Values

Default values are specified in the chart's `values.yaml` file. If you need to change any of these values, it is recommended to create a `values-override.yaml` file for your customizations.

##### Using the `--values` Flag

You can use the `--values` (or short form `-f`) flag in your Helm commands to override values in the chart with a new file. Specify the name of the file after the `--values` flag:

```console
helm install <RELEASE_NAME> cloudzero/cloudzero-agent \
    --set existingSecretName=<NAME_OF_SECRET> \
    --set clusterName=<CLUSTER_NAME> \
    --set cloudAccountId=<CLOUD_ACCOUNT_ID> \
    --set region=<REGION> \
    -f values-override.yaml
```

Ensure `values-override.yaml` contains only the values you wish to override from `values.yaml`.

> Note it is possible to save values for different environments, or based on other criteria into seperate values files and multiple files using the `-f` helm parameters.

##### Using the `--set` Flag

You can use the `--set` flag in Helm commands to directly set or override specific values from `values.yaml`. Use dot notation to specify nested values:

```console
helm install <RELEASE_NAME> cloudzero/cloudzero-agent \
    --set existingSecretName=<NAME_OF_SECRET> \
    --set clusterName=<CLUSTER_NAME> \
    --set cloudAccountId=<CLOUD_ACCOUNT_ID> \
    --set region=<REGION> \
    --set server.resources.limits.memory=2048Mi \
    -f values-override.yaml
```

### Secret Management

The chart requires a CloudZero API key to send metric data. Admins can retrieve API keys [here](https://app.cloudzero.com/organization/api-keys).

The API key can be supplied as an existing secret (default) or created by the chart. Ensure the Secret is in the same namespace as the chart and follows this format:

**values-override.yaml**
```yaml
data:
  value: <API_KEY>
```

Example of creating a secret:
```console
kubectl create secret -n example-namespace generic example-secret-name --from-literal=value=<example-api-key-value>
```

The secret can then be used with `existingSecretName`.

### Memory Sizing

Please see the [sizing guide](./docs/sizing-guide.md) in the docs directory.

### Metric Exporters

This chart depends on metrics from [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics) and [node-exporter](https://github.com/prometheus/node_exporter) projects as subcharts.

By default, these subcharts are disabled to allow scraping from existing instances. Configure the `cloudzero-agent` to use existing service endpoint addresses in `values.yaml`:

```yaml
validator:
  serviceEndpoints:
     kubeStateMetrics: <kube-state-metrics>.<example-namespace>.svc.cluster.local:8080
     prometheusNodeExporter: <node-exporter>.<example-namespace>.svc.cluster.local:9100
```

Alternatively, deploy them automatically by enabling settings in `values-override.yaml`:

```yaml
kube-state-metrics:
  enabled: true
prometheus-node-exporter:
  enabled: true
```

#### Passing Values to Subcharts

Values can be passed to subcharts like [kube-state-metrics](https://github.com/prometheus-community/helm-charts/blob/main/charts/kube-state-metrics/values.yaml) and [prometheus-node-exporter](https://github.com/prometheus-community/helm-charts/blob/main/charts/prometheus-node-exporter/values.yaml) by adding entries in `values-override.yaml` as per their specifications.

A common addition may be to pull the container images from custom image registries/repositories:

`values-override.yaml`
```yaml
kube-state-metrics:
  enabled: true
  image:
    registry: my-custom-registry.io
    repository: my-custom-kube-state-metrics/kube-state-metrics

prometheus-node-exporter:
  enabled: true
  image:
    registry: my-custom-registry.io
    repository: my-custom-prometheus/node-exporter
```

### Custom Scrape Configs

If running without the default exporters, adjust Prometheus scrape configs:

`values-override.yaml`
```yaml
prometheusConfig:
  scrapeJobs:
    kubeStateMetrics:
      enabled: false # this disables the default kube-state-metrics scrape job, which will be replaced by an entry in additionalScrapeJobs
    additionalScrapeJobs:
    - job_name: custom-kube-state-metrics
      honor_timestamps: true
      scrape_interval: 1m
      scrape_timeout: 10s
      metrics_path: /metrics
      static_configs:
        - targets:
          - 'my-kube-state-metrics-service.default.svc.cluster.local:8080'
          - 'my-node-exporter.default.svc.cluster.local:9100'
      relabel_configs:
      - separator: ;
        regex: __meta_kubernetes_service_label_(.+)
        replacement: $1
        action: labelmap
      - source_labels: [__meta_kubernetes_namespace]
        separator: ;
        regex: (.*)
        target_label: namespace
        replacement: $1
        action: replace
      - source_labels: [__meta_kubernetes_service_name]
        separator: ;
        regex: (.*)
        target_label: service
        replacement: $1
        action: replace
      - source_labels: [__meta_kubernetes_pod_node_name]
        separator: ;
        regex: (.*)
        target_label: node
        replacement: $1
        action: replace
      kubernetes_sd_configs:
        - role: endpoints
          kubeconfig_file: ""
          follow_redirects: true
          enable_http2: true
```

### Exporting Pod Labels

Pod labels can be exported as metrics using `kube-state-metrics`. Customize the labels to export in `values-override.yaml`:

Note a subset of relevant pod labels can be included -- for example only exporting the pod labels named `foo` and `bar` -- can be achieved with the following:

```yaml
kube-state-metrics:
  extraArgs:
    - --metric-labels-allowlist=pods=[foo,bar]
```

> This is preferable to including all labels with `*` because the performance and memory impact is reduced. Regular expression matching is not currently supported. See the `kube-state-metrics` [documentation](https://github.com/kubernetes/kube-state-metrics/blob/main/docs/developer/cli-arguments.md) for more details.


## Dependencies

| Repository                                         | Name                     | Version |
|----------------------------------------------------|--------------------------|---------|
| https://prometheus-community.github.io/helm-charts | kube-state-metrics       | 5.15.*  |
| https://prometheus-community.github.io/helm-charts | prometheus-node-exporter | 4.24.*  |

## Useful References

- [Memory Sizing Guide](./docs/sizing-guide.md)
- [Deployment Validation Guide](./docs/deploy-validation.md)

---