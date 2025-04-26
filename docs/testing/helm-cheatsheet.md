# Helm Chart Development Cheatsheet

## Listing Helm Releases

To list all Helm releases in a specific namespace:

```bash
helm list --namespace <namespace>
```

To list all Helm releases across all namespaces:

```bash
helm list --all-namespaces
```

## Getting Deployed Chart Values and Manifest

### 1. Get Deployed Chart Values

To retrieve the values used in a deployed release:

```bash
helm get values <release-name> --namespace <namespace>
```

### 2. Get Deployed Chart Manifest

To view the Kubernetes manifests of a deployed release:

```bash
helm get manifest <release-name> --namespace <namespace>
```

## Rendering the Chart Before Deployment

### 1. Navigate to the Helm Directory

To move into the Helm directory:

```bash
cd helm
```

### 2. Create an Override YAML File

To create an override YAML file with custom values, use the following command:

```bash
cat > override-values.yaml <<EOF
cloudAccountId: "975482786146"
clusterName: aws-cirrus-jb-cluster
region: us-east-2
existingSecretName: existing-dev-api-key
host: dev-api.cloudzero.com

commonMetaLabels:
    engr.os.com/component: cloudzero
    engr.os.com/initiative: odc
    engr.os.com/ring: ring
    engr.os.com/service: cloudzero
    engr.os.com/stamp: stamp

insightsController:
    annotations:
        enabled: false
        patterns:
        - .*
    enabled: true
    labels:
        enabled: true
        patterns:
        - .*
    resources:
        deployments: true
        namespaces: true
        pods: true
    tls:
        caBundle: ""
        crt: ""
        enabled: true
        key: ""
        mountPath: /etc/certs
        secret:
            create: true
            name: ""
        useCertManager: false
EOF
```

> **Note:** Ensure the values in the `override-values.yaml` file match your specific deployment requirements.

### 3. Build Chart Dependencies

To download and build the chart dependencies:

```bash
helm dependency build
```

### 4. Render Manifests and Values Without Deploying

To preview the combined values that will be used during deployment (without applying them):

```bash
helm template <release-name> . --values values.yaml --values override-values.yaml --dry-run --namespace <namespace>
```

> **Note:** Replace `<release-name>` with your Helm release name (e.g., `cloudzero-agent`).

---

## Fake Deployments with `--dry-run`

To simulate a deployment and debug potential issues without actually deploying:

```bash
helm install <release-name> . --values values.yaml --values override-values.yaml --namespace <namespace> --dry-run --debug
```

---

## Upgrade with Debugging

To upgrade an existing release and inspect the changes before applying:

```bash
helm upgrade --install <release-name> . --values values.yaml --values override-values.yaml --namespace <namespace> --dry-run --debug
```

## Wait for Install to Complete with Timeout

To make Helm wait for the install to complete, use the `--wait` flag. You can also set a timeout for the operation using the `--timeout` flag.

```bash
helm install <release-name> . --values values.yaml --values override-values.yaml --namespace <namespace> --wait --timeout 5m
```

### Explanation:

- `--wait`: Ensures Helm waits until all resources are in a ready state before completing the install.
- `--timeout`: Specifies the maximum time to wait for the operation to complete (e.g., `5m` for 5 minutes).

> **Note:** Replace `<release-name>` and `<namespace>` with the appropriate values for your deployment.

---

## Additional Helm Commands

### 1. Show Chart Information

To display detailed information about a chart:

```bash
helm show all <chart>
```

### 2. Search for Charts in a Repository

To search for a chart in a specific repository:

```bash
helm search repo cloudzero-beta/cloudzero-agent --devel
```

### 3. Add and Update a Helm Repository

To add a Helm repository and update the local cache:

```bash
helm repo add cloudzero-beta https://cloudzero.github.io/cloudzero-charts/beta
helm repo update
```

### 4. Rollback a Release

To roll back a release to a previous revision:

```bash
helm rollback <release-name> --namespace <namespace>
```

### 5. Uninstall a Release

To uninstall a release and delete its resources:

```bash
helm uninstall <release-name> --namespace <namespace>
```

### 6. Lint the Chart with Debugging

To validate the chart for potential errors with debugging enabled:

```bash
helm lint . --values values.yaml --values override-values.yaml --namespace <namespace> --debug
```

### 7. Package the Chart

To package the chart into a `.tgz` file for distribution:

```bash
helm package .
```

### 8. Run Tests

To run tests for a deployed release:

```bash
helm test <release-name> --namespace <namespace>
```

---

### 9. View Events Related to the Install

To view events related to a Helm release installation, use the following command to describe the release's resources:

```bash
kubectl describe <resource-type> <resource-name> --namespace <namespace>
```

For example, to view events for a pod created by the release:

```bash
kubectl describe pod <pod-name> --namespace <namespace>
```

> **Note:** Replace `<resource-type>`, `<resource-name>`, and `<namespace>` with the appropriate values for your release.

### 10. View Kubernetes Events

To view real-time Kubernetes events in a specific namespace:

```bash
kubectl get events --namespace <namespace> --watch
```

To filter events by a specific resource, use the `--field-selector` flag. For example, to view events for a specific pod:

```bash
kubectl get events --namespace <namespace> --field-selector involvedObject.name=<pod-name>
```

> **Note:** Replace `<namespace>` and `<pod-name>` with the appropriate values for your use case.

### 11. Clear Previous Events

To delete all events in a specific namespace and clear previous information:

```bash
kubectl delete events --namespace <namespace>
```

> **Note:** Replace `<namespace>` with the appropriate namespace name.

---

### Get Pod Manifest and Delete Pod or Deployment

#### 1. Get the Manifest for a Pod

To retrieve the manifest for a specific pod running in a namespace:

```bash
kubectl get pod <pod-name> --namespace <namespace> -o yaml
```

> **Note:** Replace `<pod-name>` and `<namespace>` with the appropriate values.

#### 2. Delete a Pod

To delete a specific pod in a namespace:

```bash
kubectl delete pod <pod-name> --namespace <namespace>
```

> **Note:** Replace `<pod-name>` and `<namespace>` with the appropriate values.

#### 2. Get the Manifest for a Pod

To retrieve the manifest for a specific pod running in a namespace:

```bash
kubectl get deployment <pod-name> --namespace <namespace> -o yaml
```

> **Note:** Replace `<pod-name>` and `<namespace>` with the appropriate values.

#### 2. Delete a Deployment

To delete a deployment and its associated pods in a namespace:

```bash
kubectl delete deployment <deployment-name> --namespace <namespace>
```

> **Note:** Replace `<deployment-name>` and `<namespace>` with the appropriate values.
