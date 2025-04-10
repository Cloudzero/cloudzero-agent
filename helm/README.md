# CloudZero Agent Helm Chart

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE-OF-CONDUCT.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
![GitHub release](https://img.shields.io/github/release/Cloudzero/cloudzero-charts.svg)

A Helm chart for deploying Prometheus in agent mode to send cluster metrics to the CloudZero platform.

For the latest release, see [Releases](https://github.com/Cloudzero/cloudzero-charts/releases). You can also [enable release notifications](#enabling-release-notifications).

## Prerequisites

- Kubernetes 1.23+
- Helm 3+
- A CloudZero API key
- Each Kubernetes cluster must have a route to the internet and a rule that allows egress from the agent to the CloudZero collector endpoint at https://api.cloudzero.com on port 443

### Recommended Knowledge

For an optimal installation experience, we recommend the following:

- Have a basic understanding of Kubernetes and Helm charts.
- Be prepared with a list of the labels and annotations your organization plans to collect if you don't want the default: all pod and namespace labels with the key `app.kubernetes.io/component`. See [Labels and Annotations](#labels-and-annotations).

## Installation

### Quick Start

> ⚠️ By default, this chart exports only the app.kubernetes.io/component label from pods and namespaces. No annotations are exported. While this provides a safe default for demo purposes, it may be insufficient for your organization.
>
> **Recommendations:**
>
> - Configure additional labels to align with your organization's FinOps tagging practices.
> - Review the [Labels and Annotations](#labels-and-annotations) section for guidance on exposing an expanded set of labels or annotations to meet your organization’s specific requirements.

#### 1. Add CloudZero Helm Repository

Refer to the [`helm repo`](https://helm.sh/docs/helm/helm_repo/) documentation for command details. To use a beta version, refer to the [beta installation document](./BETA-INSTALLATION.md) for the appropriate channel.

```console
helm repo add cloudzero https://cloudzero.github.io/cloudzero-charts
helm repo update
```

#### 2. Install Helm Chart

```console
helm install <RELEASE_NAME> cloudzero/cloudzero-agent \
    --set apiKey=<CLOUDZERO_API_KEY> \
    --set clusterName=<CLUSTER_NAME> \
    --set-string cloudAccountId=<CLOUD_ACCOUNT_ID> \
    --set region=<REGION>
```

---

### Advanced Install

The "Quick Start" option will cover most test and demo use cases, but may not be appropriate for a production deployment. This section provides configuration options to ensure production quality deployment.

Below is an example of a configuration file that one might use to configure some more advanced features of the chart.

```yaml
# -- Account ID of the account the cluster is running in. This must be a string - even if it is a number in your system.
cloudAccountId: YOUR_CLOUD_ACCOUNT_ID
# -- Name of the clusters.
clusterName: YOUR_CLUSTER_NAME
# -- Region the cluster is running in.
region: YOUR_CLOUD_REGION
# -- CloudZero API key. Required if existingSecretName is null.
apiKey: YOUR_CLOUDZERO_API_KEY
# -- If set, the agent will use the API key in this Secret to authenticate with CloudZero. This may be preferable for users who would like to manage the CloudZero API key in a Secret external to this helm chart. See *Secret Management* below for details.
existingSecretName: YOUR_EXISTING_API_KEY_K8S_SECRET

# -- Configuration for managing the gathering of labels and annotations. See the below *Labels and Annotations* section for more details.
# -- Note that this configuration, and support for annotations and labels on resources other than pods, is only supported in versions post-1.0.0.
insightsController:
  # -- By default, a ValidatingAdmissionWebhook will be deployed that records all created labels and annotations
  enabled: true
  labels:
    # -- Determines whether the agent will gather labels from Kubernetes resources.
    enabled: true
    # -- This value MUST be set to a list of regular expressions which will be used to gather labels from pods, deployments, statefulsets, daemonsets, cronjobs, jobs, nodes, and namespaces
    patterns:
      - "^foo" # -- Match all labels whose key starts with "foo"
      - "bar$" # -- Match all labels whose key ends with "bar"
    # -- Labels can be gathered from pods and namespaces by default. See the values.yaml for more options.
    resources:
      pods: true
      namespaces: true
      deployments: true
      statefulsets: true
      nodes: true
      jobs: true
      cronjobs: true
      daemonsets: true
  annotations:
    # -- By default, the gathering of annotations is not enabled. To enable, set this field to true
    enabled: false
    patterns:
      - ".*" # -- match all annotations. This is not recommended.
  tls:
    # -- If disabled, the insights controller will not mount a TLS certificate from a Secret, and the user is responsible for configuring a method of providing TLS information to the webhook-server container.
    enabled: true
    # -- If left as an empty string, the certificate will be generated by the chart. Otherwise, the provided value will be used.
    crt: ""
    # -- If left as an empty string, the certificate private key will be generated by the chart. Otherwise, the provided value will be used.
    key: ""
    secret:
      # -- If set to true, a Secret will be created to store the TLS certificate and key.
      create: true
      # -- If set, the Secret will be created with this name. Otherwise, a default name will be generated.
      name: ""
    # -- The following TLS certificate information is for a self signed certificate. It is used as a default value for the validating admission webhook and the webhook server.
    # -- This path determines the location within the container where the TLS certificate and key will be mounted.
    mountPath: /etc/certs
    # -- This is the caBundle used by the Validating Admission Webhook when sending requests to the webhook server. If left empty, the default self-signed certificate will be used.
    # Set this value to an empty string if using cert-manager to manage the certificate instead. Otherwise, set this to the base64 encoded caBundle of the desired certificate.
    caBundle: ""
    # -- If enabled, the certificate will be managed by cert-manager, which must already be present in the cluster.
    # If disabled, a default self-signed certificate will be used.
    useCertManager: false
```

### Mandatory Values

There are several mandatory values that must be specified for the chart to install properly. Below are the required settings along with strategies for providing custom values during installation:

| Key                | Type   | Default               | Description                                                                                                                                    |
| ------------------ | ------ | --------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| cloudAccountId     | string | `nil`                 | Account ID in AWS or Subscription ID in Azure or Project Number in GCP where the cluster is running. Must be a string due to Helm limitations. |
| clusterName        | string | `nil`                 | Name of the cluster. Must be RFC 1123 compliant.                                                                                               |
| host               | string | `"api.cloudzero.com"` | CloudZero host to send metrics to.                                                                                                             |
| apiKey             | string | `nil`                 | The CloudZero API key to use for exporting metrics. Only used if `existingSecretName` is not set.                                              |
| existingSecretName | string | `nil`                 | Name of the secret that contains the CloudZero API key. Required if not providing the API key via `apiKey`.                                    |
| region             | string | `nil`                 | Region where the cluster is running (e.g., `us-east-1`, `eastus`). For more information, see AWS or Azure documentation.                       |

> It is recommended to use a `values-override.yaml` file for customizations. For details, refer to the [official Helm documentation](https://helm.sh/docs/helm/helm_install/#synopsis).

### Labels and Annotations

> ⚠️ CloudZero supports a maximum of **300 labels** for Kubernetes resources. Ensure you configure regex patterns to gather only the necessary labels/annotations. Additional labels after the first 300 are discarded.

> ⚠️ Note that this configuration, and support for annotations and labels on resources other than pods, is only supported in versions post-1.0.0.

**By default**, this chart exports pod and namespace labels with keys matching `app.kubernetes.io/component`, and no annotations. You can configure what labels and/or annotations are exported by following the steps in this section.

This chart allows the exporting of labels and annotations from the following resources:

- `Pod`
- `Deployment`
- `StatefulSet`
- `Daemonset`
- `Job`
- `CronJob`
- `Node`
- `Namespace`

The export of labels and annotations from a cluster can be turned on or off within the `insightsController` field. For example, the following enables exporting both labels and annotations from pods and namespaces:

```yaml
insightsController:
  enabled: true
  labels:
    enabled: true
  annotations:
    enabled: true
```

It is recommended to supply a list of regexes to filter only the labels/annotations required:

```yaml
insightsController:
  enabled: true
  labels:
    enabled: true
    patterns:
      - "^foo" # -- Match all labels whose key starts with "foo"
      - "bar$" # -- Match all labels whose key ends with "bar"
```

Labels/annotations can also be gathered from more than just pods and namespaces. An example of gathering labels from all available resources would be:

```yaml
insightsController:
  enabled: true
  labels:
    enabled: true
    resources:
      pods: true
      namespaces: true
      deployments: true
      statefulsets: true
      nodes: true
      jobs: true
      cronjobs: true
      daemonsets: true
```

Additional Notes:

- Labels and annotations exports are managed in the `insightsController` section of the `values.yaml` file.
- By default, only labels from pods and namespaces are exported. To enable more resources, see the `insightsController.labels.resources` and `insightsController.annotations.resources` section of the `values.yaml` file.
- To disambiguate labels/annotations between resources, a prefix representing the resource type is prepended to the label key in the [CloudZero Explorer](https://app.cloudzero.com/explorer). For example, a `foo=bar` node label would be presented as `node:foo: bar`. The exception is pod labels which do not have resource prefixes for backward compatibility with previous versions.
- Annotations are not exported by default; see the `insightsController.annotations.enabled` setting to enable. To disambiguate annotations from labels, an `annotation` prefix is prepended to the annotation key; i.e., an `foo: bar` annotation on a namespace would be represented in the Explorer as `node:annotation:foo: bar`
- For both labels and annotations, the `patterns` array applies across all resource types; i.e., setting `['^foo']` for `insightsController.labels.patterns` will match label keys that start with `foo` for all resource types set to `true` in `insightsController.labels.resources`.

### Secret Management

The chart requires a CloudZero API key to send metric data. Admins can retrieve API keys [here](https://app.cloudzero.com/organization/api-keys).

The API key is typically stored in a Secret in the cluster. The `cloudzero-agent` chart will create a Secret if the API key is provided via the `apiKey` argument. Alternatively, the API key can be stored in a Secret external to the chart; this Secret name would then be set as the `existingSecretName` argument. If creating a Secret external to the chart, ensure the Secret is in the same namespace as the chart and follows this format:

**Example User-Created Secret Content**

```yaml
data:
  value: <API_KEY>
```

Example of creating a secret:

```console
kubectl create secret -n example-namespace generic example-secret-name --from-literal=value=<example-api-key-value>
```

The secret can then be used with `existingSecretName`.

### Update Helm Chart

If you are updating an existing installation, pull the latest chart information:

```console
helm repo update
```

Next, upgrade the installation to the latest chart version:

```console
helm upgrade --install <RELEASE_NAME> cloudzero/cloudzero-agent -f configuration.example.yaml
```

#### Getting All Image References

A common customization may be to mirror the container images used in this chart into a private registry. To fetch all image references used in this chart, use the following commands:

```console
CHART_VERSION=1.0.0        # Set this to the chart version for which container image references should be fetched
CHART_REPO=cloudzero       # Set this to the name of cloudzero helm repository.
helm template $CHART_REPO/cloudzero-agent --version $CHART_VERSION --set clusterName=foobar --set cloudAccountId=foobar --set region=foobar | grep -i image: | tr -d '"' | sort | uniq | awk '{print $NF}'
```

This will return the latest image references for that particular chart version.

#### Passing Values to Subcharts

Values can be passed to subcharts like [kube-state-metrics](https://github.com/prometheus-community/helm-charts/blob/main/charts/kube-state-metrics/values.yaml) by adding entries in `values-override.yaml` as per their specifications.

A common addition may be to pull the container images from custom image registries/repositories:

`values-override.yaml`

```yaml
kubeStateMetrics:
  enabled: true
  image:
    registry: my-custom-registry.io
    repository: my-custom-kube-state-metrics/kube-state-metrics
```

## Contributing

We welcome contributions to the CloudZero Agent Helm chart. Contributions for this chart are managed through the [CloudZero Agent repository](https://github.com/cloudzero/cloudzero-agent), which is then automatically synced to the [CloudZero Charts repository](https://github.com/cloudzero/cloudzero-charts). We cannot accept contributions for anything in this directory through the CloudZero Charts repository as they would be overwritten automatically the next time a change is made in the CloudZero Agent Validator repository.

## Dependencies

| Repository                                         | Name               | Version |
| -------------------------------------------------- | ------------------ | ------- |
| https://prometheus-community.github.io/helm-charts | kube-state-metrics | 5.15.\* |

Note that while `kube-state-metrics` is listed as a dependency, it is referred to as `cloudzero-state-metrics` within the helm chart. This is to enforce the idea that this KSM deployment is used exclusively by the `cloudzero-agent`.

## Enabling Release Notifications

To receive a notification when a new version of the chart is [released](https://github.com/Cloudzero/cloudzero-charts/releases), you can [watch the repository](https://docs.github.com/en/account-and-profile/managing-subscriptions-and-notifications-on-github/setting-up-notifications/configuring-notifications#configuring-your-watch-settings-for-an-individual-repository):

1. Navigate to the [repository main page](https://github.com/Cloudzero/cloudzero-charts).
2. Select **Watch > Custom**.
3. Check the **Releases** box.
4. Select **Apply**.

## Useful References

- [Memory Sizing Guide](./docs/sizing-guide.md)
- [Deployment Validation Guide](./docs/deploy-validation.md)
- Using istio? [Read on here](./docs/istio.md)
- [Chart release notes](./docs/releases/)

---
