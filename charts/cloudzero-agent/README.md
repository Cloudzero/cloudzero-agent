# cloudzero-agent

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE-OF-CONDUCT.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
![GitHub release](https://img.shields.io/github/release/cloudzero/template-cloudzero-open-source.svg)

A chart for using Prometheus in agent mode to send cluster metrics to the CloudZero platform.

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
    --set cloudAccountId=<CLOUD_ACCOUNT_ID>
```

### Secret Management

The chart requires an CloudZero API key in order to send metric data to the CloudZero platform. Admins can retrieve API keys [here](https://app.cloudzero.com/organization/api-keys).

The Deployment running Prometheus ingests the API key via a Secret; this Secret can be supplied as an existing secret (default), or created by the chart. Both methods will require the API key retrieved from the CloudZero platform.

If using a Secret external to this chart for the API key, ensure the Secret is created in the same namespace as the chart and that the Secret data follows the format:

```yaml
data:
  value: <API_KEY>
```

For example, the Secret could be created with:
```bash
kubectl create secret -n example-namespace generic example-secret-name --from-literal=value=<example-api-key-value>
```
The Secret can then be used by the agent by giving `example-secret-name` as the Secret name for the `existingSecretName` argument.

### Metric Exporters
This chart uses metrics from the [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics) and [node-exporter](https://github.com/prometheus/node_exporter) projects as chart dependencies. By default, these subcharts are disabled so that the agent can scrape metrics from existing instances of `kube-state-metrics` and `node-exporter`. They can optionally be enabled with the settings:
```yaml
kube-state-metrics:
  enabled: true
prometheus-node-exporter:
  enabled: true
```
This will deploy the required resources for metric scraping.

### Exporting Pod Labels
Pod labels can be exported as metrics for use in the CloudZero platform by using the `metric-labels-allowlist` CLI argument to the `kube-state-metrics` container. This is disabled by default due to the increase in cardinality that exporting all pod labels introduces.

To export **all** pod labels, set the following in the helm chart:
```yaml
kube-state-metrics:
  extraArgs:
    - --metric-labels-allowlist=pods=[*]
```
A subset of relevant pod labels can be included; as an example, exporting only pod labels that start with `foobar_` could be achieved with the following:
```yaml
kube-state-metrics:
  extraArgs:
    - --metric-labels-allowlist=pods=[foobar_*]
```
See the `kube-state-metrics` [documentation](https://github.com/kubernetes/kube-state-metrics/tree/main/docs#cli-arguments) for more details.

## Values

| Key               | Type   | Default               | Description                                                                                                             |
|-------------------|--------|-----------------------|-------------------------------------------------------------------------------------------------------------------------|
| cloudAccountId    | string | `nil`                 | Account ID of the account the cluster is running in.                                                                    |
| clusterName       | string | `nil`                 | Name of the cluster. Required to be RFC 1123 compliant.                                                                 |
| host              | string | `"api.cloudzero.com"` | CloudZero host to send metrics to.                                                                                      |
| apiKey            | string | `nil`                 | The CloudZero API key to use to export metrics. Only used if `existingSecretName` is not set.                           |
| existingSecretName| string | `nil`                  | The name of the secret that contains the CloudZero API key. Required if not providing the API key via the apiKey value.|

## Requirements

| Repository                                         | Name                     | Version |
|----------------------------------------------------|--------------------------|---------|
| https://prometheus-community.github.io/helm-charts | kube-state-metrics       | 5.15.*  |
| https://prometheus-community.github.io/helm-charts | prometheus-node-exporter | 4.24.*  |
