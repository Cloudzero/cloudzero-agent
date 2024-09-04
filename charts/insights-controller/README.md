# Cloudzero Insights Controller Helm Chart

[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-2.1-4baaaa.svg)](CODE-OF-CONDUCT.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
![GitHub release](https://img.shields.io/github/release/Cloudzero/cloudzero-charts.svg)

A Helm chart for a validating admission webhook to send cluster metrics to the CloudZero platform.

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
helm install <RELEASE_NAME> cloudzero/insights-controller
```

See the next section for different deployment configurations.

#### Certificate Management

This chart contains a `ValidatingWebhookConfiguration` resource, which can use a certificate in order validate requests to the webhook server. See documentation [here](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#configure-admission-webhooks-on-the-fly).


There are several ways to configure the chart certificate setup:

1. Manage certificates using [cert-manager](https://github.com/cert-manager/cert-manager/tree/master).
By default, the chart installs [cert-manager](https://github.com/cert-manager/cert-manager/tree/master) as a subchart. `cert-manager` handles the creation of the certificate and injects the CA bundle into the `ValidatingWebhookConfiguration` resource. For details on how cert-manager does this, see [here](https://cert-manager.io/docs/concepts/ca-injector/)

To install the chart with this configuration, run the following:

```bash
helm install <RELEASE_NAME> cloudzero/insights-controller
```
If `cert-manager` CRDs are not already installed, the installation may fail. If this happens, run the following:

```bash
helm install <RELEASE_NAME> cloudzero/insights-controller \
    --set webhook.issuer.enabeld=false \
    --set webhook.certificate.enabeld=false \
```
And then rerun the original command:
```bash
helm install <RELEASE_NAME> cloudzero/insights-controller
```

2. The second option is to not use a certificate. While it is a good practice to secure connections, it is not strictly required by the `ValidatingWebhookConfiguration` spec. To run without a certifciate, run the following:
```bash
helm install <RELEASE_NAME> cloudzero/insights-controller \
    --set webhook.issuer.enabeld=false \
    --set webhook.certificate.enabeld=false \
    --set cert-manager.enabled=false \
    --set server.service.port=80 \
    --set server.tlsMount.useManagedSecret=false \
```

3. The third option is to bring your own certificate. In this case, the tls information must be mounted to the server Deployment at the `/etc/certs/` path in the format:
```
  ca.crt: <CA_CRT>
  tls.crt: <TKS_CERT>
  tls.key: <TLS_KEY>
```
An example command would be:
```bash
helm install <RELEASE_NAME> cloudzero/insights-controller -f config.yaml
```
where `config.yaml` is:
```
server:
  tlsMount:
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