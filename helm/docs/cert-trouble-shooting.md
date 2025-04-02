# Troubleshooting Guide: Certificate Issues for CloudZero Agent Helm Chart

This guide provides comprehensive steps to troubleshoot certificate-related issues when installing and running the CloudZero Agent Helm chart, which functions as a Kubernetes Admission Controller. Proper certificate management is crucial for securing communication between Kubernetes components and the CloudZero platform. This guide covers common certificate problems, diagnostic steps, and resolution strategies, whether you are using the built in certificate management, `cert-manager`, the `cloudzero-certificate` chart, or your own certificate management solution.

## Table of Contents

1. [Understanding Certificate Management](#understanding-certificate-management)
2. [Common Certificate Issues](#common-certificate-issues)
3. [Diagnostic Steps](#diagnostic-steps)
   - [1. Verify Certificate Configuration](#1-verify-certificate-configuration)
   - [2. Check Certificate Status](#2-check-certificate-status)
   - [3. Inspect Admission Webhook Configuration](#3-inspect-admission-webhook-configuration)
   - [4. Review Pod Logs](#4-review-pod-logs)
   - [5. Validate Secret Contents](#5-validate-secret-contents)
4. [Resolution Strategies](#resolution-strategies)
   - [1. Reinstall or Update `cert-manager`](#1-reinstall-or-update-cert-manager)
   - [2. Regenerate Certificates Using `cloudzero-certificate` Chart](#2-regenerate-certificates-using-cloudzero-certificate-chart)
   - [3. Use a Custom Certificate](#3-use-a-custom-certificate)
   - [4. Correct Certificate References in Configuration](#4-correct-certificate-references-in-configuration)
5. [Best Practices](#best-practices)
6. [Additional Resources](#additional-resources)

---

## Understanding Certificate Management

The CloudZero Agent Helm chart deploys a `ValidatingWebhookConfiguration` resource that requires TLS certificates to secure communication with the Kubernetes API server. By default, the chart automatically generates a self-signed certificate and configures the components to use it without any extra configuration. However, the chart also supports alternative methods:

1. **Using `cert-manager`**: Automates certificate issuance and renewal; see [here](https://cert-manager.io/) for details.
2. **Using `cloudzero-certificate` Chart**: Manually manages certificates via a separate Helm chart.
3. **Bringing Your Own Certificate**: Utilizes externally managed certificates provided by an existing certificate manager.

Proper configuration and management of these certificates are essential for the admission controller to function correctly.

---

## Common Certificate Issues

1. **Certificate Not Issued**: The TLS certificate required for the webhook is not created.
2. **Certificate Expired**: The existing certificate has expired, causing webhook validation failures.
3. **Invalid Certificate Configuration**: Incorrect references to certificate secrets or misconfigured issuer settings.
4. **Webhook Fails to Authenticate**: Mismatch between the CA bundle in the webhook configuration and the actual certificate.
5. **`cert-manager` CRDs Not Installed**: Missing Custom Resource Definitions required by `cert-manager`.

---

## Diagnostic Steps For Non-Default Options

### 1. Verify Certificate Configuration

Ensure that the certificate management configuration in your `values.yaml` or `configuration.example.yaml` is correctly set based on your chosen method (`cert-manager`, `cloudzero-certificate`, or custom certificates).

- **Using `cert-manager`**:
  ```yaml
  insightsController:
    tls:
      useCertManager: true
  ```
- **Using `cloudzero-certificate` Chart**:

  ```yaml
  insightsController:
    tls: # fetch these values from the output of the `cloudzero-certificate` chart
      crt: <base64-encoded-cert-value>
      key: <base64-encoded-key-value>
      caBundle: <base64-encoded-caBundle-value>
  ```

- **Using Custom Certificates**:
  ```yaml
  insightsController:
    tls:
      secret:
        create: false
        name: <name-of-secret-containing-tls-information>
  ```
  OR
  ```yaml
  insightsController:
    tls: # set these values to your own certificate
      crt: <base64-encoded-cert-value>
      key: <base64-encoded-key-value>
      caBundle: <base64-encoded-caBundle-value>
  ```

### 2. Check Certificate Status

#### If Using `cert-manager`:

- **Verify `cert-manager` is Running**:

  ```bash
  kubectl get pods -n <YOUR_NAMESPACE>
  ```

  Ensure all `cert-manager` pods are in the `Running` state. Replace the `-n <YOUR_NAMESPACE>` with the cert-manager namespace. eg. `-n cert-manager`.

  eg.

  ```bash
  $ kubectl get pods -n cert-manager
  NAME                                       READY   STATUS    RESTARTS   AGE
  cert-manager-57d855897b-v5f5s              1/1     Running   0          11d
  cert-manager-cainjector-5c7f79b84b-nwksr   1/1     Running   0          11d
  cert-manager-webhook-657b9f664c-ph9zz      1/1     Running   0          11d
  ```

- **Check the Certificate Manager Logs**

  ```bash
  kubectl logs -n <YOUR_NAMESPACE> <CERTIFICATE_MANAGER_POD>
  ```

  eg.

  ```
  $ kubectl logs -n cert-manager cert-manager-57d855897b-v5f5s
  ```

- **Check Certificate Resources**:

  ```bash
  kubectl get certificates -n <YOUR_NAMESPACE>
  ```

  Ensure that the certificate for the webhook is in the `Ready` condition.

  eg.

  ```bash
  $ kubectl get certificates -n default
  NAME                                         READY   SECRET                               AGE
  cloudzero-agent-webhook-server-certificate   True    cloudzero-agent-webhook-server-tls   6d17h
  ```

- **Describe Certificate**:

  ```bash
  kubectl describe certificate <CERTIFICATE_NAME> -n <YOUR_NAMESPACE>
  ```

  Look for any error messages or issues in the events section.

  eg.

  ```yaml
  $ kubectl describe certificate cloudzero-agent-webhook-server-certificate
  Name:         cloudzero-agent-webhook-server-certificate
  Namespace:    default
  Labels:       app.kubernetes.io/managed-by=Helm
  Annotations:  meta.helm.sh/release-name: cloudzero-agent
                meta.helm.sh/release-namespace: default
  API Version:  cert-manager.io/v1
  Kind:         Certificate
  Metadata:
    Creation Timestamp:  2024-12-16T19:58:07Z
    Generation:          1
    Resource Version:    1194195
    UID:                 e9e5dc99-9896-41a5-8223-3aac8098617b
  Spec:
    Dns Names:
      cloudzero-agent-webhook-server-svc.default.svc
    Duration:  2160h
    Issuer Ref:
      Kind:  Issuer
      Name:  cloudzero-agent-webhook-server-issuer
    Private Key:
      Algorithm:   RSA
      Encoding:    PKCS1
      Size:        2048
    Renew Before:  360h
    Secret Name:   cloudzero-agent-webhook-server-tls
    Secret Template:
      Labels:
        app.kubernetes.io/component:   webhook-server
        app.kubernetes.io/instance:    cloudzero-agent
        app.kubernetes.io/managed-by:  Helm
        app.kubernetes.io/name:        cloudzero-agent
        app.kubernetes.io/part-of:     cloudzero-agent
        app.kubernetes.io/version:     v2.50.1
        helm.sh/chart:                 cloudzero-agent-1.0.0-beta-5
  Status:
    Conditions:
      Last Transition Time:  2024-12-16T19:58:07Z
      Message:               Certificate is up to date and has not expired
      Observed Generation:   1
      Reason:                Ready
      Status:                True
      Type:                  Ready
    Not After:               2025-03-12T22:14:22Z
    Not Before:              2024-12-12T22:14:22Z
    Renewal Time:            2025-02-25T22:14:22Z
  Events:                    <none>
  ```

#### If Using `cloudzero-certificate` Chart or Custom Certificates:

- **Check Secret Existence**:

  ```bash
  kubectl get secret -n <YOUR_NAMESPACE> <YOUR_TLS_SECRET_NAME>
  ```

  Ensure the secret exists and contains `tls.crt`, `tls.key`, and `ca.crt`.

- **Inspect Secret Data**:
  ```bash
  kubectl describe secret <YOUR_TLS_SECRET_NAME> -n <YOUR_NAMESPACE>
  ```
  Verify that the certificate data is correctly populated.

### 3. Inspect Admission Webhook Configuration

- **Retrieve Webhook Configuration**:

  ```bash
  kubectl get validatingwebhookconfigurations
  kubectl describe validatingwebhookconfiguration <WEBHOOK_CONFIGURATION_NAME>
  ```

  eg.

  ```bash
  kubectl describe validatingwebhookconfiguration  cloudzero-agent-webhook-server-webhook-namespaces
  kubectl describe validatingwebhookconfiguration  cloudzero-agent-webhook-server-webhook-pods
  ```

- **Verify CA Bundle**:
  Ensure that the `caBundle` field matches the CA certificate used to sign the webhook server certificate.

### 4. Review Pod Logs

- **Identify Relevant Pods**:
  ```bash
  kubectl get pods -n <YOUR_NAMESPACE>
  ```
- **Check Logs for Errors**:
  ```bash
  kubectl logs <POD_NAME> -n <YOUR_NAMESPACE>
  ```
  Look for certificate-related error messages such as `tls: failed to verify certificate` or `x509: certificate signed by unknown authority`.

### 5. Validate Secret Contents

Ensure that the Kubernetes Secret containing the certificates has the correct keys and valid base64-encoded data.

- **Decode Certificate Data**:
  ```bash
  kubectl get secret <YOUR_TLS_SECRET_NAME> -n <YOUR_NAMESPACE> -o jsonpath='{.data.tls\.crt}' | base64 --decode | openssl x509 -text -noout
  ```
- **Check for Validity and Expiry**:
  Verify that the certificate is not expired and the details (such as Common Name) are correct.

---

## Resolution Strategies

### 1. Reinstall or Update `cert-manager`

If using `cert-manager`, ensure it is correctly installed and up to date.

- **Install `cert-manager` CRDs**:

  ```bash
  kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.2/cert-manager.yaml
  ```

- **Update `cert-manager`**:

  ```bash
  helm repo update
  helm upgrade cert-manager jetstack/cert-manager --namespace cert-manager --version v1.16.2
  ```

- **Restart `cert-manager` Pods**:
  ```bash
  kubectl rollout restart deployment cert-manager -n cert-manager
  ```

### 2. Regenerate Certificates Using `cloudzero-certificate` Chart

If using the `cloudzero-certificate` chart, regenerate the certificates to ensure validity.

1. **Uninstall Existing Certificates**:

   ```bash
   helm uninstall <YOUR_RELEASE_NAME> -n <YOUR_NAMESPACE>
   ```

2. **Reinstall the `cloudzero-certificate` Chart**:

   ```bash
   helm upgrade --install <YOUR_RELEASE_NAME> cloudzero/cloudzero-certificate \
     --namespace <YOUR_NAMESPACE> \
     --set cloudzeroAgentReleaseName=<CLOUDZERO_AGENT_RELEASE_NAME>
   ```

3. **Retrieve and Update CA Bundle**:

   ```bash
   CA_BUNDLE=$(kubectl get secret -n <YOUR_NAMESPACE> <YOUR_RELEASE_NAME>-cloudzero-certificate -o jsonpath='{.data.ca\.crt}')
   ```

4. **Update Configuration**:
   Ensure `configuration.example.yaml` reflects the new secret and CA bundle.

### 3. Use a Custom Certificate

If you prefer to use your own certificate:

1. **Generate Certificates**:
   Ensure the Common Name is `<RELEASE_NAME>.<RELEASE_NAMESPACE>.cluster.local`.

2. **Create Kubernetes Secret**:

   ```bash
   kubectl create secret tls <YOUR_TLS_SECRET_NAME> \
     --cert=path/to/tls.crt \
     --key=path/to/tls.key \
     --namespace <YOUR_NAMESPACE>
   ```

   Additionally, create a secret for the CA bundle:

   ```bash
   kubectl create secret generic <YOUR_CA_SECRET_NAME> \
     --from-file=ca.crt=path/to/ca.crt \
     --namespace <YOUR_NAMESPACE>
   ```

3. **Update Configuration**:
   Modify `configuration.example.yaml` to reference your custom secrets:

   ```yaml
   insightsController:
     server:
       tls:
         nameOverride: <YOUR_TLS_SECRET_NAME>
     webhooks:
       caBundle: <BASE64_ENCODED_CA_BUNDLE>
     certificate:
       enabled: false
     issuer:
       enabled: false
   cert-manager:
     enabled: false
   ```

4. **Reinstall Helm Chart**:
   ```bash
   helm upgrade --install <RELEASE_NAME> cloudzero/cloudzero-agent -f configuration.example.yaml
   ```

### 4. Correct Certificate References in Configuration

Ensure that all certificate-related fields in your configuration files correctly reference the existing secrets and CA bundles.

- **Verify `nameOverride`**:
  The `nameOverride` under `insightsController.server.tls` should match the name of your TLS secret.

- **Ensure `caBundle` is Correct**:
  The `caBundle` should contain the base64-encoded CA certificate that signed the webhook server certificate.

- **Disable Conflicting Certificate Managers**:
  If you switch from one certificate management method to another, ensure that conflicting settings (like `cert-manager.enabled`) are appropriately disabled.

---

## Best Practices

- **Automate Certificate Renewal**: Use `cert-manager` or similar tools to handle certificate issuance and renewal automatically.
- **Monitor Certificate Expiry**: Implement monitoring to alert you before certificates expire.
- **Secure Secret Management**: Ensure that Kubernetes Secrets containing certificates are securely managed and access-controlled.
- **Consistent Naming Conventions**: Maintain consistent naming for secrets and release names to avoid configuration mismatches.
- **Regularly Update Dependencies**: Keep `cert-manager` and other dependencies up to date to benefit from security patches and improvements.

---

## Additional Resources

- [Kubernetes Admission Controllers Documentation](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)
- [`cert-manager` Documentation](https://cert-manager.io/docs/)
- [CloudZero Agent Helm Chart README](./README.md)
- [ValidatingWebhookConfiguration Reference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#validatingwebhookconfiguration-v1-admissionregistration-k8s-io)
- [Helm Troubleshooting Guide](https://helm.sh/docs/faq/troubleshooting/)

---

If you continue to experience certificate-related issues after following this guide, consider reaching out to [support@cloudzero.com](mailto:support@cloudzero.com) or filing an issue on the [CloudZero Helm Charts GitHub repository](https://github.com/Cloudzero/cloudzero-charts/issues).
