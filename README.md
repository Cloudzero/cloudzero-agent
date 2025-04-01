# CloudZero Agent

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE-OF-CONDUCT.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
![GitHub release](https://img.shields.io/github/release/Cloudzero/cloudzero-agent-validator.svg)

<img src="./docs/assets/deployment.png" alt="deployment" width="700">

This repository contains several applications to support Kubernetes integration
with the CloudZero platform, including:

- _CloudZero Insights Controller_ - provides telemetry to the CloudZero platform
  to enabling complex cost allocation and analysis.
- _CloudZero Collector_ - the collector application exposes a prometheus remote
  write API which can receive POST requests from prometheus in either v1 or v2
  encoded format. It decodes the messages, then writes them to the `data`
  directory as parquet files with snappy compression.
- _CloudZero Shipper_ - the shipper application watches the data directory
  looking for completed parquet files on a regular interval (eg. 10 min), then
  will call the `CloudZero upload API` to allocate S3 Presigned PUT URLS. These
  URLs are used to upload the file. The application has the ability to compress
  the files before sending them to S3.
- _CloudZero Agent Validator_ - the validator application performs various
  validation checks.

## ⚡ Getting Started With CloudZero Insights Controller

The easiest way to get started with the _CloudZero Insights Controller_ is by
using the `cloudzero-agent` Helm chart from the [cloudzero-charts
repository](https://github.com/Cloudzero/cloudzero-charts).

### Installation

See the [Installation Guide](./INSTALL.md) for details.

### Configuration

See the [Configuration Guide](./CONFIGURATION.md) for details.

###### Cleanup

```sh
make undeploy-admission-controller
make undeploy-test-app
```

### Debugging

The applications are based on a scratch container, so no shell is available. The
container images are less than 8MB.

To monitor the data directory, you must deploy a `debug` container as follows:

1. **Deploy a debug container**

   ```bash
   kubectl apply  -f cluster/deployments/debug/deployment.yaml
   ```

2. **Attach to the shell of the debug container**

   ```bash
   kubectl exec -it temp-shell -- /bin/sh
   ```

   To inspect the data directory, `cd /cloudzero/data`

---

### **Clean Up**

```bash
eksctl delete cluster -f cluster/cluster.yaml --disable-nodegroup-eviction
```

## Collector & Shipper Architecture

![](./docs/assets/overview.png)

This project provides a collector application, written in golang, which provides two applications:

- `Collector` - the collector application exposes a prometheus remote write API
  which can receive POST requests from prometheus in either v1 or v2 encoded
  format. It decodes the messages, then writes them to the `data` directory as
  Brotri-compressed JSON.
- `Shipper` - the shipper application watches the data directory looking for
  completed parquet files on a regular interval (eg. 10 min), then will call the
  `CloudZero upload API` to allocate S3 Presigned PUT URLS. These URLs are used
  to upload the file. The application has the ability to compress the files
  before sending them to S3.

## Message Format

The output of the _CloudZero Insights Controller_ application is a JSON object
that represents `cloudzero` metrics, which is POSTed to the CloudZero remote
write API. The format of these objects is based on the Prometheus `Timeseries`
protobuf message, defined
[here](https://github.com/prometheus/prometheus/blob/main/prompb/types.proto#L122-L130).
Protobuf definitions for the `cloudzero` metrics are in the `proto/` directory.

There are four kinds of objects that can be sent:

1. **Pod metrics**

### Metric Names

- `cloudzero_pod_labels`
- `cloudzero_pod_annotations`

### Required Fields

- `__name__`; will be one of the valid pod metric names
- `namespace`; the namespace that the pod is launched in
- `resource_type`; will always be `pod` for pod metrics

<details open>
<summary>Example</summary>

```json
{
  "labels": [
    {
      "name": "__name__",
      "value": "cloudzero_pod_labels"
    },
    {
      "name": "namespace",
      "value": "default"
    },
    {
      "name": "pod",
      "value": "hello-28889630-955wd"
    },
    {
      "name": "resource_type",
      "value": "pod"
    },
    {
      "name": "label_batch.kubernetes.io/controller-uid",
      "value": "cc52c38d-b461-40ab-a65d-2d5a68ac08e5"
    },
    {
      "name": "label_batch.kubernetes.io/job-name",
      "value": "hello-28889630"
    },
    {
      "name": "label_controller-uid",
      "value": "cc52c38d-b461-40ab-a65d-2d5a68ac08e5"
    },
    {
      "name": "label_job-name",
      "value": "hello-28889630"
    }
  ],
  "samples": [
    {
      "value": 1.0,
      "timestamp": "1733378003953"
    }
  ]
}
```

</details>

2. **Workload Metrics**

### Metric Names

- `cloudzero_deployment_labels`
- `cloudzero_deployment_annotations`
- `cloudzero_statefulset_labels`
- `cloudzero_statefulset_annotations`
- `cloudzero_daemonset_labels`
- `cloudzero_daemonset_annotations`
- `cloudzero_job_labels`
- `cloudzero_job_annotations`
- `cloudzero_cronjob_labels`
- `cloudzero_cronjob_annotations`

### Required Fields

- `__name__`; will be one of the valid workload metric names
- `namespace`; the namespace that the workload is launched in
- `workload`; the name of the workload
- `resource_type`; will be one of `deployment`, `statefulset`, `daemonset`,
  `job`, or `cronjob`

<details open>
<summary>Example</summary>

```json
{
  "labels": [
    {
      "name": "__name__",
      "value": "cloudzero_deployment_labels"
    },
    {
      "name": "namespace",
      "value": "default"
    },
    {
      "name": "workload",
      "value": "hello"
    },
    {
      "name": "resource_type",
      "value": "deployment"
    },
    {
      "name": "label_component",
      "value": "greeting"
    },
    {
      "name": "label_foo",
      "value": "bar"
    }
  ],
  "samples": [
    {
      "value": 1.0,
      "timestamp": "1733378003953"
    }
  ]
}
```

</details>

3.  **Namespace Metrics**

### Metric Names

- `cloudzero_namespace_labels`
- `cloudzero_namespace_annotations`

### Required Fields

- `__name__`; will be one of the valid namespace metric names
- `namespace`; the name of the namespace
- `resource_type`; will always be `namespace` for namespace metrics

<details open>
<summary>Example</summary>

```json
{
  "labels": [
    {
      "name": "__name__",
      "value": "cloudzero_namespace_labels"
    },
    {
      "name": "namespace",
      "value": "default"
    },
    {
      "name": "resource_type",
      "value": "namespace"
    },
    {
      "name": "label_engr.os.com/component",
      "value": "foo"
    },
    {
      "name": "label_kubernetes.io/metadata.name",
      "value": "default"
    }
  ],
  "samples": [
    {
      "value": 1.0,
      "timestamp": "1733880410225"
    }
  ]
}
```

</details>

4.  **Node Metrics**

### Metric Names

- `cloudzero_node_labels`
- `cloudzero_node_annotations`

### Required Fields

- `__name__`; will be one of the valid node metric names
- `node`; the name of the node
- `resource_type`; will always be `node` for node metrics

<details open>
<summary>Example</summary>

```json
{
  "labels": [
    {
      "name": "__name__",
      "value": "cloudzero_node_labels"
    },
    {
      "name": "resource_type",
      "value": "node"
    },
    {
      "name": "label_alpha.eksctl.io/nodegroup-name",
      "value": "spot-nodes"
    },
    {
      "name": "label_beta.kubernetes.io/arch",
      "value": "amd64"
    }
  ],
  "samples": [
    {
      "value": 1.0,
      "timestamp": "1733880410225"
    }
  ]
}
```

</details>

## 🤝 How to Contribute

We appreciate feedback and contribution to this repo! Before you get started,
please see the following:

- [This repo's contribution guide](CONTRIBUTING.md)

## 🤔 Support + Feedback

Contact support@cloudzero.com for usage, questions, specific cases. See the
[CloudZero Docs](https://docs.cloudzero.com/) for general information on
CloudZero.

## 🛡️ Vulnerability Reporting

Please do not report security vulnerabilities on the public GitHub issue
tracker. Email [security@cloudzero.com](mailto:security@cloudzero.com) instead.

## ☁️ What is CloudZero?

CloudZero is the only cloud cost intelligence platform that puts engineering in
control by connecting technical decisions to business results.:

- [Cost Allocation And Tagging](https://www.cloudzero.com/tour/allocation)
  Organize and allocate cloud spend in new ways, increase tagging coverage, or
  work on showback.
- [Kubernetes Cost Visibility](https://www.cloudzero.com/tour/kubernetes)
  Understand your Kubernetes spend alongside total spend across containerized
  and non-containerized environments.
- [FinOps And Financial Reporting](https://www.cloudzero.com/tour/finops)
  Operationalize reporting on metrics such as cost per customer, COGS, gross
  margin. Forecast spend, reconcile invoices and easily investigate variance.
- [Engineering Accountability](https://www.cloudzero.com/tour/engineering)
  Foster a cost-conscious culture, where engineers understand spend, proactively
  consider cost, and get immediate feedback with fewer interruptions and faster
  and more efficient innovation.
- [Optimization And Reducing Waste](https://www.cloudzero.com/tour/optimization)
  Focus on immediately reducing spend by understanding where we have waste,
  inefficiencies, and discounting opportunities.

Learn more about [CloudZero](https://www.cloudzero.com/) on our website
[www.cloudzero.com](https://www.cloudzero.com/)

## 📜 License

This project is licensed under the Apache 2.0 [LICENSE](LICENSE).
