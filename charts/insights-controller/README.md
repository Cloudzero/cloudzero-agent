# Cloudzero Insights Controller Helm Chart

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE-OF-CONDUCT.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
![GitHub release](https://img.shields.io/github/release/Cloudzero/cloudzero-charts.svg)

A Helm chart for a validating admission webhook to send cluster metrics to the CloudZero platform.

## Overview

This Validating Admission Webhook monitors and intercepts `CREATE` and `UPDATE` operations on the following Kubernetes resources:

- `Deployment`
- `Pod`
- `StatefulSet`
- `Node`
- `Namespace`

The webhook captures the labels from these resources and uploads them to the CloudZero API endpoint. For both `CREATE` and `UPDATE` operations, the full set of labels is sent to the API, ensuring that the most up-to-date labels are always uploaded. For `Deployment` and `Statefulset` resources, annotations are also uploaded.


## Prerequisites

- Kubernetes 1.23+
- Helm 3+
- A CloudZero API key

## Installation

This helm chart is best used alongside the [cloudzero-agent](https://github.com/Cloudzero/cloudzero-charts/tree/develop/charts/cloudzero-agent) chart. In this case, the same API key can be used for both installations.

### Get Helm Repository Info

```console
helm repo add cloudzero https://cloudzero.github.io/cloudzero-charts
helm repo update
```

_See [`helm repo`](https://helm.sh/docs/helm/helm_repo/) for command documentation._

The chart can be installed directly with Helm or any other common Kubernetes deployment tools. See the next section for different deployment configurations.

### Deployment Configurations and Certificate Management

This chart contains a `ValidatingWebhookConfiguration` resource, which uses a certificate in order validate requests to the webhook server. See related Kubernetes documentation [here](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#configure-admission-webhooks-on-the-fly).


**There are two ways to install the chart as it relates to certificate management:**

1. (Default) Manage certificates using [cert-manager](https://github.com/cert-manager/cert-manager/tree/master).
By default, the chart installs [cert-manager](https://github.com/cert-manager/cert-manager/tree/master) as a subchart. `cert-manager` handles the creation of the certificate and injects the CA bundle into the `ValidatingWebhookConfiguration` resource. For details on how cert-manager does this, see [here](https://cert-manager.io/docs/concepts/ca-injector/).

To install the chart with this configuration, install the chart with the following helm command. The default configuration uses cert-manager to create the certificate:

```console
helm install <RELEASE_NAME> cloudzero/insights-controller \
    --set existingSecretName=<NAME_OF_SECRET> \
    --set clusterName=<CLUSTER_NAME> \
    --set-string cloudAccountId=<CLOUD_ACCOUNT_ID> \
    --set region=<REGION>
```

If `cert-manager` CRDs are not already installed, the installation may fail with the error message that contains:
```console
no matches for kind "Certificate" in version "cert-manager.io/v1"
```

If this happens, run the following:

```bash
helm install <RELEASE_NAME> cloudzero/insights-controller \
    --set webhook.issuer.enabled=false \
    --set webhook.certificate.enabled=false \
    --set cert-manager.installCRDs=true
```
Or, alternatively, [install the cert-manager CRDs yourself](https://cert-manager.io/docs/installation/helm/).
Then rerun the original command:
```console
helm install <RELEASE_NAME> cloudzero/insights-controller \
    --set existingSecretName=<NAME_OF_SECRET> \
    --set clusterName=<CLUSTER_NAME> \
    --set-string cloudAccountId=<CLOUD_ACCOUNT_ID> \
    --set region=<REGION>
```

2. The second option is to bring your own certificate. In this case, the tls information must be mounted to the server Deployment at the `/etc/certs/` path in a file formatted as:
```
ca.crt: <CA_CRT>
tls.crt: <TLS_CERT>
tls.key: <TLS_KEY>
```
An example command would be:
```bash
helm install <RELEASE_NAME> cloudzero/insights-controller \
    --set existingSecretName=<NAME_OF_SECRET> \
    --set clusterName=<CLUSTER_NAME> \
    --set-string cloudAccountId=<CLOUD_ACCOUNT_ID> \
    --set region=<REGION> \
    -f config.yaml
```
where `config.yaml` is:
```
server:
  tls:
    useManagedSecret: false
  volumeMounts:
    - name: your-tls-volume
      mountPath: /etc/certs
      readOnly: true
  volumes:
    - name: tls-certs
      secret:
        secretName: your-tls-secret-name
webhook:
  issuer:
    enabled: false
  certificate:
    enabled: false
  caBundle: '<YOUR_CA_BUNDLE>'

cert-manager:
  enabled: false
```

## Troubleshooting

### `<RELEASE-NAME>-server` pod stuck in `Pending` state
  The server pod, which handles incoming webhook requests, may be stuck in this state if the TLS secret is not available. Confirm this is the case by describing the server pod:
  ```console
  kubectl describe pod -l app.kubernetes.io/name=insights-controller
  ```
  If the event log shows that the pod cannot be created due to a missing volume, check that the TLS secret has been created successfully:
  ```console
  kubectl get secret -l app.kubernetes.io/name=insights-controller
  ```
  If no secrets are returned by that command, then cert-manager did not provision a certificate. Consult the `cert-manager` pod logs and/or the cert-manager CRDs for more infomration:
  ```console
  kubectl get certificaterequests
  kubectl get certificates
  kubectl get certificatesigningrequests
  kubectl get issuers
  ```
