# Testing the CloudZero-Agent Validating Webhook

This guide shows how to send a fake Pod creation **AdmissionReview** to your validating webhook, either **locally** (via port-forward) or **in-cluster** (using a debug pod). It also references Istio-specific instructions in [helm/docs/istio.md](../../helm/docs/istio.md).

---

## Prerequisites

- `kubectl` configured for your EKS/GKE/AKS cluster  
- `curl` or the `curlimages/curl` image (for in-cluster)  
- The webhook deployed in the `cloudzero-agent` namespace  

---

## 1. Create the Test AdmissionReview File

Save the following JSON as `pod-admission.json` in your local repo under `tests/data/`:

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

> **Note:** No external URL is needed—this file is maintained in your repo under the `develop` branch.

---

## 2. Test Locally (Port-Forward)

1. **Start port-forward**  
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
> - The `-k` flag skips TLS verification for self-signed certs.  
> - Adjust host/port if your webhook is exposed differently.

---

## 3. Test In-Cluster (Debug Pod)

```bash
kubectl run -i --tty --rm debug \
  --image=curlimages/curl --restart=Never \
  --namespace cloudzero-agent -- \
  bash -c '
    curl -k -v \
      -H "Content-Type: application/json" \
      --data-binary @<(cat tests/data/pod-admission.json) \
      https://cloudzero-agent-webhook-server-svc.cloudzero-agent.svc.cluster.local/validate
  '
```  

> Swap out `cloudzero-agent` or service name if your setup differs.

---

## 4. Inspect the TLS Certificate

Sometimes you’ll need to confirm which certificate your webhook is presenting:

### a) Via `curl -v`

In the verbose output you’ll see:
```
* Server certificate:
*  subject: CN=cloudzero-agent-webhook-server-svc
*  start date: ...
*  expire date: ...
*  issuer: CN=cloudzero-agent-webhook-server-svc
```

### b) Via `openssl s_client`

```bash
openssl s_client -connect localhost:8443 -showcerts </dev/null | openssl x509 -noout -text
```

Or in-cluster:

```bash
kubectl run --rm -i tmp --image=alpine --restart=Never -- \
  sh -c "apk add --no-cache openssl && \
         openssl s_client -connect cloudzero-agent-webhook-server-svc.cloudzero-agent.svc.cluster.local:443 -showcerts </dev/null | openssl x509 -noout -text"
```

This prints full cert details (Subject, Issuer, SANs, validity, fingerprints), invaluable for TLS debugging.

---

## 5. Deployment Considerations

When deploying your webhook in production, consider different TLS setups:

1. **Self-signed certificates**  
   - Use `-k` in tests or import the CA into your trust store.  
      > Check the certificate details using `curl -v` or `openssl s_client`. Look for the `issuer` field in the certificate output. If the `issuer` matches the `subject`, it indicates a self-signed certificate.  
   - Issue a cert for `cloudzero-agent-webhook-server-svc.cloudzero-agent.svc`.  

2. **cert-manager**  
   - Issue a cert for `cloudzero-agent-webhook-server-svc.cloudzero-agent.svc`.  
3. **Istio**  
   - See [helm/docs/istio.md](helm/docs/istio.md) for full Istio-specific configuration.  
   - At minimum, disable sidecar injection or mTLS on the webhook-server pods to prevent conflicts.

