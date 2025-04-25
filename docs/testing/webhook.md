# Testing the CloudZero-Agent Validating Webhook

This guide demonstrates how to test the CloudZero-Agent validating webhook by sending a fake Pod creation **AdmissionReview** request. You can perform the test either **locally** (via port-forward) or **in-cluster** (using a debug pod). Additionally, Istio-specific instructions are available in [helm/docs/istio.md](../../helm/docs/istio.md).

---

## Prerequisites

Ensure the following are available:

- `kubectl` configured for your EKS/GKE/AKS cluster
- `curl` or the `curlimages/curl` image (for in-cluster testing)
- The webhook deployed in the `cloudzero-agent` namespace

---

## 1. Create the Test AdmissionReview File

Save the following JSON as `pod-admission.json` in your local repository under `tests/data/`:

```bash
cat > tests/data/pod-admission.json <<EOF
{
  "apiVersion": "admission.k8s.io/v1",
  "kind": "AdmissionReview",
  "request": {
    "uid": "abcdef12-3456-7890-abcd-ef1234567890",
    "kind": { "group": "", "version": "v1", "kind": "Pod" },
    "resource": { "group": "", "version": "v1", "resource": "pods" },
    "namespace": "default",
    "operation": "CREATE",
    "object": {
      "apiVersion": "v1",
      "kind": "Pod",
      "metadata": { "name": "fake-pod", "namespace": "default", "labels": { "app": "fake" } },
      "spec": { "containers": [ { "name": "busybox", "image": "busybox", "command": ["sleep", "3600"] } ] }
    },
    "oldObject": null
  }
}
EOF
```

> **Note:** This file is maintained in your repository under the `develop` branch.

---

## 2. Test Locally (Port-Forward)

1. **Start port-forwarding**

   ```bash
   kubectl -n cloudzero-agent port-forward svc/cloudzero-agent-webhook-server-svc 8443:443
   ```

2. **Send the request**
   ```bash
   curl -k -v \
     -H "Content-Type: application/json" \
     --data-binary @tests/data/pod-admission.json \
     https://localhost:8443/validate
   ```

> **Notes:**
>
> - The `-k` flag skips TLS verification for self-signed certificates.
> - Adjust the host/port if your webhook is exposed differently.

---

## 4. Inspect the TLS Certificate

To debug TLS issues, inspect the webhook's certificate using one of the following methods:

### a) Using `curl -v`

Run the following command to view certificate details during a request:

```bash
curl -k -v \
  -H "Content-Type: application/json" \
  --data-binary @tests/data/pod-admission.json \
  https://localhost:8443/validate
```

The verbose output includes details such as the certificate's subject, issuer, and validity period.

### b) Using `openssl s_client`

Inspect the certificate directly with `openssl`:

```bash
openssl s_client -connect localhost:8443 -showcerts </dev/null | openssl x509 -noout -text
```

This outputs the certificate's full details, including SANs, validity, and fingerprints.

---

## 5. Deployment Considerations

When deploying the webhook in production, consider the following TLS configurations:

1. **Self-signed certificates**

   - Use `-k` in tests or import the CA into your trust store.
   - Ensure the certificate is issued for `cloudzero-agent-webhook-server-svc.cloudzero-agent.svc`.

2. **cert-manager**

   - Use cert-manager to issue a certificate for `cloudzero-agent-webhook-server-svc.cloudzero-agent.svc`.

3. **Istio**
   - Refer to [helm/docs/istio.md](helm/docs/istio.md) for Istio-specific configurations.
   - Disable sidecar injection or mTLS on webhook-server pods to avoid conflicts.

---

## 5. DNS Troubleshooting

### Cluster DNS IP

Check the cluster's DNS IP by inspecting the CoreDNS ConfigMap:

```bash
kubectl -n kube-system get configmap coredns -o yaml
```

### Pod DNS Configuration

Ensure the pod's `/etc/resolv.conf` file points to the correct DNS server. Inspect it with:

```bash
kubectl exec -it <pod-name> -- cat /etc/resolv.conf
```
