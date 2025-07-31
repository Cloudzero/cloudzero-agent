# Webhook Test Manifests

These manifests are designed to test the always-allow webhook behavior:

## Test Resources

1. **test-namespace.yaml** - Creates `webhook-test` namespace
2. **test-deployment.yaml** - Creates a deployment with 2 nginx replicas in the test namespace
3. **test-pod.yaml** - Creates a standalone nginx pod in the test namespace
4. **test-storageclass.yaml** - Creates a new StorageClass with gp3 EBS provisioner
5. **test-gatewayclass.yaml** - Creates a GatewayClass with fake controller (safe, won't conflict)

All resources have consistent labels: `team: cirrus`, `purpose: testing`, `test: always-allow`

## Usage

Apply the manifests to test webhook validation:

```bash
kubectl apply -f tests/manifests/test-namespace.yaml
kubectl apply -f tests/manifests/test-deployment.yaml
kubectl apply -f tests/manifests/test-pod.yaml
kubectl apply -f tests/manifests/test-storageclass.yaml
kubectl apply -f tests/manifests/test-gatewayclass.yaml
```

Clean up after testing:

```bash
kubectl delete -f tests/manifests/
```

## Expected Behavior

With the always-allow webhook, all resources should be admitted successfully even if there are validation issues or webhook errors.
