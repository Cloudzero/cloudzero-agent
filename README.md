# CloudZero Insights Controller

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE-OF-CONDUCT.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
![GitHub release](https://img.shields.io/github/release/Cloudzero/cloudzero-insights-controller.svg)

<img src="./docs/assets/deployment.png" alt="deployment" width="700">

This repository contains several applications to support Kubernetes integration with the CloudZero platform, including:

* *CloudZero Insights Controller* - provides telemetry to the CloudZero platform to enabling complex cost allocation and analysis.
* *CloudZero Collector* - the collector application exposes a prometheus remote write API which can receive POST requests from prometheus in either v1 or v2 encoded format. It decodes the messages, then writes them to the `data` directory as parquet files with snappy compression.
* *CloudZero Shipper* - the shipper application watches the data directory looking for completed parquet files on a regular interval (eg. 10 min), then will call the `CloudZero upload API` to allocate S3 Presigned PUT URLS. These URLs are used to upload the file. The application has the ability to compress the files before sending them to S3.

## ⚡ Getting Started With CloudZero Insights Controller

The easiest way to get started with the *CloudZero Insights Controller* is by using the `cloudzero-agent` Helm chart from the [cloudzero-charts repository](https://github.com/Cloudzero/cloudzero-charts).

### Installation

See the [Installation Guide](./INSTALL.md) for details.

### Configuration

See the [Configuration Guide](./CONFIGURATION.md) for details.

### Developer Quick Start

1. Build the image

   ```sh
   TAG=poc-simple make package
   ```

2. Deploy the admission controller

   ```sh
   make deploy-admission-controller
   ```

3. Monitor the logs in one console

   ```sh
   ./scripts/monitor-admission-contoller.sh
   ```

4. In another console, deploy a test app.

   ```sh
   make deploy-test-app
   ```

   > NOW - check out the logs in 3

###### Cleanup

```sh
make undeploy-admission-controller
make undeploy-test-app
```

## Getting Started With CloudZero Collector & Shipper

## DEPLOY QUICK START

### **Step 1: Create an Amazon EKS Cluster**

We'll use `eksctl` to create a new EKS cluster.

Run the following command to create the cluster:

```bash
eksctl create cluster -f cluster/cluster.yaml
```

> **Note**: This process may take several minutes to complete. Once finished, `kubectl` will be configured to interact with your new cluster.

---

### **Step 2: Install External Secrets Operator**

Next use Helm to install the External Secrets Operator into your cluster.

1. **Add the External Secrets Helm Repository**

   ```bash
   helm repo add external-secrets https://charts.external-secrets.io
   helm repo update
   ```

2. **Install the Operator Using Helm**

   ```bash
   helm install external-secrets external-secrets/external-secrets \
     --namespace external-secrets --create-namespace
   ```

---

### **Step 3: Create a Secret in AWS Secrets Manager**

We'll create a secret in AWS Secrets Manager that we want to sync to Kubernetes.

1. **Create the Secret**

   ```bash
   aws secretsmanager create-secret \
     --region us-east-2 \
     --name 'dev/cloudzero-secret-api-key' \
     --secret-string "{\"apiToken\":\"$CZ_DEV_API_TOKEN\"}"
   ```

   > **Note**:
   > Replace `$CZ_API_TOKEN` with your actual API token or ensure that the `CZ_API_TOKEN` environment variable is set.

---

### **Step 4: Configure IAM Roles for Service Accounts (IRSA)**

To allow the operator to securely access AWS Secrets Manager, we'll use IAM Roles for Service Accounts.

1. **Enable and Associate the OIDC Provider**

   If you haven't enabled the OIDC provider for your cluster, run:

   ```bash
   eksctl utils associate-iam-oidc-provider --cluster aws-cirrus-jb-cluster --approve
   ```

2. **Create the IAM Policy**

   Run:

   ```bash
   aws iam create-policy \
     --policy-name ExternalSecretsPolicy \
     --policy-document file://cluster/deployments/cloudzero-secrets/external-secrets-policy.json
   ```

3. **Create the IAM Service Account Using `eksctl`**

   ```bash
   eksctl create iamserviceaccount \
     --name external-secrets-irsa \
     --namespace default \
     --cluster aws-cirrus-jb-cluster \
     --role-name external-secrets-irsa-role \
     --attach-policy-arn arn:aws:iam::975482786146:policy/ExternalSecretsPolicy \
     --approve \
     --override-existing-serviceaccounts
   ```

4. **Verify the Service Account Creation**

   ```bash
   kubectl get serviceaccount external-secrets-irsa -n default -o yaml
   ```

---

### **Step 5: Deployment the Cloudzero Collector Application Set**

1. **Build the Container Images**

    ```bash
    make package
    ```

2. **Make the `cloudzero` namespace**

    ```bash
    kubectl apply  -f cluster/deployments/namespace.yml
    ```

3. **Deploy the External Secret Store**

    ```bash
    kubectl apply  -f cluster/deployments/cloudzero-secrets/secretstore.yaml
    ```

4. **Add the External Secret**

    ```bash
    kubectl apply  -f cluster/deployments/cloudzero-secrets/externalsecret.yaml
    ```

5. **Deploy Collector Application Set**

    ```bash
    kubectl apply -f cluster/deployments/cloudzero-collector/config.yaml \
                  -f cluster/deployments/cloudzero-collector/deployment.yaml
    ```


### **Step 6: Deploy Federated Cloudzero Agent**

1. **Deploy the  Federated Cloudzero Agent**

    ```bash
    kubectl apply  -f app/manifests/prometheus-federated/deployment.yml
    ```

---

### Debugging

The applications are based on a scratch container, so no shell is available. The container images are less than 8MB.

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

* `Collector` - the collector application exposes a prometheus remote write API which can receive POST requests from prometheus in either v1 or v2 encoded format. It decodes the messages, then writes them to the `data` directory as parquet files with snappy compression.
* `Shipper` - the shipper application watches the data directory looking for completed parquet files on a regular interval (eg. 10 min), then will call the `CloudZero upload API` to allocate S3 Presigned PUT URLS. These URLs are used to upload the file. The application has the ability to compress the files before sending them to S3.

## Message Format

The output of the *CloudZero Insights Controller* application is a JSON object that represents `cloudzero` metrics, which is POSTed to the CloudZero remote write API. The format of these objects is based on the Prometheus `Timeseries` protobuf message, defined [here](https://github.com/prometheus/prometheus/blob/main/prompb/types.proto#L122-L130). Protobuf definitions for the `cloudzero` metrics are in the `proto/` directory.

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
- `resource_type`; will be one of `deployment`, `statefulset`, `daemonset`, `job`, or `cronjob`

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

We appreciate feedback and contribution to this repo! Before you get started, please see the following:

- [This repo's contribution guide](CONTRIBUTING.md)

## 🤔 Support + Feedback

Contact support@cloudzero.com for usage, questions, specific cases. See the [CloudZero Docs](https://docs.cloudzero.com/) for general information on CloudZero.

## 🛡️ Vulnerability Reporting

Please do not report security vulnerabilities on the public GitHub issue tracker. Email [security@cloudzero.com](mailto:security@cloudzero.com) instead.

## ☁️ What is CloudZero?

CloudZero is the only cloud cost intelligence platform that puts engineering in control by connecting technical decisions to business results.:

- [Cost Allocation And Tagging](https://www.cloudzero.com/tour/allocation) Organize and allocate cloud spend in new ways, increase tagging coverage, or work on showback.
- [Kubernetes Cost Visibility](https://www.cloudzero.com/tour/kubernetes) Understand your Kubernetes spend alongside total spend across containerized and non-containerized environments.
- [FinOps And Financial Reporting](https://www.cloudzero.com/tour/finops) Operationalize reporting on metrics such as cost per customer, COGS, gross margin. Forecast spend, reconcile invoices and easily investigate variance.
- [Engineering Accountability](https://www.cloudzero.com/tour/engineering) Foster a cost-conscious culture, where engineers understand spend, proactively consider cost, and get immediate feedback with fewer interruptions and faster and more efficient innovation.
- [Optimization And Reducing Waste](https://www.cloudzero.com/tour/optimization) Focus on immediately reducing spend by understanding where we have waste, inefficiencies, and discounting opportunities.

Learn more about [CloudZero](https://www.cloudzero.com/) on our website [www.cloudzero.com](https://www.cloudzero.com/)

## 📜 License

This project is licenced under the Apache 2.0 [LICENSE](LICENSE).
