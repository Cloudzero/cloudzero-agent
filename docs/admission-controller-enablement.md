# Ensuring Admission Control Support is Enabled

To ensure that the `admissionregistration.k8s.io/v1` API is enabled in your Kubernetes cluster, you can use kubectl to check the available API resources and versions. Here's how you can verify:

1. List All API Resources: Use kubectl api-resources to list all the API resources available in your cluster. This command shows which API groups and versions are enabled.

    ```bash
    kubectl api-resources
    ```

    Look for resources like `ValidatingWebhookConfiguration` under the `admissionregistration.k8s.io` API group.

2. Check Specific API Version: To specifically check for the `admissionregistration.k8s.io/v1` version, you can use kubectl api-versions:

    ```bash
    kubectl api-versions | grep admissionregistration.k8s.io/v1
    ```

    This command filters the list of available API versions to see if `admissionregistration.k8s.io/v1` is listed.

3. Describe API Version: If you want more detailed information about what's included in the admissionregistration.k8s.io/v1 API, you can use kubectl explain:

    ```bash
    kubectl explain --api-version=admissionregistration.k8s.io/v1
    ```

    This will give you a description of the API version and the resources available under it.

## Ensuring the API is Enabled

If the `admissionregistration.k8s.io/v1` API is not listed, it may be due to the Kubernetes version you are using or specific cluster configurations. Here’s what you can consider:

* Upgrade Kubernetes: If your cluster is running an older version of Kubernetes, consider upgrading to a version that supports admissionregistration.k8s.io/v1, which is generally available starting from Kubernetes 1.16 and above.

* Cluster Configuration: In some Kubernetes setups, especially managed Kubernetes services, certain APIs may be disabled by default. Check with your cloud provider's documentation or the cluster administrator to understand how to enable specific APIs.

* Feature Gates: In rare cases, certain APIs might be controlled by feature gates. You would need to ensure that no feature gate is disabling this API.

## Trouble Shooting

### Running `kubectl api-resources`, I get the following output. Is admission controllers enabled?

**Example Output**

```bash
validatingadmissionpolicies                      admissionregistration.k8s.io/v1   false        ValidatingAdmissionPolicy
validatingadmissionpolicybindings                admissionregistration.k8s.io/v1   false        ValidatingAdmissionPolicyBinding
validatingwebhookconfigurations                  admissionregistration.k8s.io/v1   false        ValidatingWebhookConfiguration
```

In the output, the admissionregistration.k8s.io/v1 API **_is available_** but the resources like MutatingWebhookConfiguration and ValidatingWebhookConfiguration are marked as "false" under the NAMESPACED column. This "false" value does not indicate that these resources are disabled; rather, it indicates that these resources are **_not namespaced—they are cluster-wide resources_**.

#### Understanding the Output

* *NAMESPACED:* false — This means the resource is not tied to a specific namespace but applies to the entire cluster.

* *API Version*: admissionregistration.k8s.io/v1 — This shows the API version is indeed available and active in your cluster.

##### Actions for Using Webhook Configurations

Since the API and its associated resources are available (not disabled), you can proceed to use them directly. 

Here’s how you can work with these resources:

Creating Mutating Webhook Configuration: To create a mutating webhook, you will need a definition file that describes the webhook. Here's an example of how it might look:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: MutatingWebhookConfiguration
metadata:
  name: example-mutating-webhook
webhooks:
  - name: "example.webhook.com"
    clientConfig:
      service:
        name: webhook-service
        namespace: default
        path: "/mutate"
      caBundle: <base64 encoded CA certificate>
    rules:
      - operations: ["CREATE", "UPDATE"]
        apiGroups: [""]
        apiVersions: ["v1"]
        resources: ["pods"]
    admissionReviewVersions: ["v1"]
    sideEffects: None
```

Creating Validating Webhook Configuration: Similarly, for creating a validating webhook, you'd define it like this:

```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
name: example-validating-webhook
webhooks:
- name: "validate.webhook.com"
    clientConfig:
    service:
        name: webhook-service
        namespace: default
        path: "/validate"
    caBundle: <base64 encoded CA certificate>
    rules:
    - operations: ["CREATE", "UPDATE"]
        apiGroups: [""]
        apiVersions: ["v1"]
        resources: ["pods"]
    admissionReviewVersions: ["v1"]
    sideEffects: None
    timeoutSeconds: 5
```

#### Deployment and Testing

* Apply the Configuration: Use kubectl apply -f <filename.yaml> to create these webhook configurations in your cluster.
* Test the Webhooks: Make sure your webhooks are correctly intercepting the requests as specified by their rules.

#### Troubleshooting
Logs: Check the logs of your webhook server if you encounter issues during operation. This can give you insights into whether requests are reaching the server and how they are being processed.
Debugging Connectivity: Ensure that the Kubernetes API server can communicate with your webhook service. Network policies, service configurations, or incorrect caBundle values can prevent proper operation.
By following these guidelines, you can effectively utilize the webhooks with the admissionregistration.k8s.io/v1 API in your Kubernetes cluster.