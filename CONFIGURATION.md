# Configuration File Documentation

This document provides an overview of the configuration file used by the validator.

A template configuration can be created using the `cloudzero-agent-validator config generate` command.

## Versions

The `versions` section specifies the versions of the chart and agent being used.

| Key             | Description       | Required | Default Values |
|-----------------|-------------------|----------|----------------|
| chart_version   | Ideally this should be defined to match the `cloudzero-agent` chart release version which is installed | Optional |                |
| agent_version   | Ideally this shoud be defined to match the prometheus agent which is deployed in the chart. | Optional |                |

## Logging

The `logging` section configures the logging settings.

| Key       | Description       | Required | Default Values |
|-----------|-------------------|----------|----------------|
| level     | The log level | Optional | `info`      |
| location  | The location of the log file | Optional | `/prometheus/cloudzero-agent-validator.log` |

## Deployment

The `deployment` section contains deployment-related settings.

| Key             | Description       | Required | Default Values |
|-----------------|-------------------|----------|----------------|
| account_id      | The account ID | Mandatory |                |
| cluster_name    | The name of the cluster | Mandatory |                |
| region          | The region of the deployment | Mandatory |                |

## CloudZero

The `cloudzero` section configures the CloudZero integration.

| Key               | Description       | Required | Default Values |
|-------------------|-------------------|----------|----------------|
| host              | The CloudZero API host | Mandatory | `https://api.cloudzero.com` |
| credentials_file  | The location of the API key file | Mandatory | `/etc/config/prometheus/secrets/value` |
| disable_telemetry | disables telemetry push to cloudzero API. Warning disabling this will result in the inability to see status of clusters in the dashboard. | Optional | `false` |

## Prometheus

The `prometheus` section configures Prometheus settings.

| Key                                      | Description       | Required | Default Values |
|------------------------------------------|-------------------|----------|----------------|
| kube_state_metrics_service_endpoint      | The endpoint for kube-state-metrics service | Mandatory |          |
| prometheus_node_exporter_service_endpoint| The endpoint for node-exporter service | Mandatory |                |
| configurations                           | List of one or more configuration files locations for prometheus to validate | Mandatory |                |

## Diagnostics

The `diagnostics` section defines the stages and checks for diagnostics.

### Stages

The `stages` list contains the different stages of diagnostics.

| Key       | Description       | Required | Default Values |
|-----------|-------------------|----------|----------------|
| name      | The name of the stage | Mandatory |                |
| enforce   | Whether to enforce the checks in the stage | Mandatory |                |
| checks    | The list of checks to perform in the stage | Mandatory |                |

### Checkers

The following table describes the available checkers:

| Checker                          | Description       |
|----------------------------------|-------------------|
| `api_key_valid`                  | Checks the API Key is valid |
| `k8s_version`                    | Checks the Kubernetes compatability |
| `egress_reachable`               | Checks pod can communicate with the Cloudzero API |
| `kube_state_metrics_reachable`   | Checks the kubernetes state metrics service is reachable |
| `node_exporter_reachable`        | Checks the prometheus node exporter service is reachable |
| `scrape_cfg`                     |  Checks the prometheus configurations exist and contain the necessary scrape configuration |

## Example

To see an example, run the application with the following parameters:
```sh
$ cloudzero-agent-validator config generate -account 1234 -cluster foo -region us-east-1
```
