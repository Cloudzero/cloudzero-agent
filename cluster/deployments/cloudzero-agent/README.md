# Federated Prometheus Installation

To deploy a KSM-focused pod and a cAdvisor-focused DaemonSet, execute the following command:

```sh
kubectl apply -f federated.yaml
```

**Important:** This configuration contains hardcoded write endpoints and includes the `organization_id` in the header for testing purposes.

In a production environment, you should dynamically construct the URL, and the authorizer lambda should add the `organization_id` to the headers.
