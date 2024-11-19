# Application Architecture

![](./docs/assets/overview.png)

This project provides a collector application, written in golang, which provides two applications:

* `Collector` - the collector application exposes a prometheus remote write API which can receive POST requests from prometheus in either v1 or v2 encoded format. It decodes the messages, then writes them to the `data` directory as parquet files with snappy compression.
* `Shipper` - the shipper application watches the data directory looking for completed parquet files on a regular interval (eg. 10 min), then will call the `CloudZero upload API` to allocate S3 Presigned PUT URLS. These URLs are used to upload the file. The application has the ability to compress the files before sending them to S3.


---

## DEPLOY QUICK START

### **Step 1: Create an Amazon EKS Cluster**

We'll use `eksctl` to create a new EKS cluster.

Run the following command to create the cluster:

```bash
eksctl create cluster -f cluster/cluster.yaml
```

> **Note**: This process may take several minutes to complete. Once finished, `kubectl` will be configured to interact with your new cluster.

---

## **Step 2: Install External Secrets Operator**

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

## **Step 3: Create a Secret in AWS Secrets Manager**

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

## **Step 4: Configure IAM Roles for Service Accounts (IRSA)**

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

## **Step 5: Deployment the Cloudzero Collector Application Set**

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


## **Step 6: Deploy Federated Cloudzero Agent**

1. **Deploy the  Federated Cloudzero Agent**

    ```bash
    kubectl apply  -f app/manifests/prometheus-federated/deployment.yml
    ```

---

## Debugging

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

## **Clean Up**

```bash
eksctl delete cluster -f cluster/cluster.yaml --disable-nodegroup-eviction
```
