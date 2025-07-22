# Validating the Deployment

This guide provides instructions on how to validate the deployment of the Helm chart and identify common problems. It outlines how to use the environment validation container to gather insight into issues post deployment.

## Step by Step

![Validator output](./assets/validator.png)

### 1. Get the pod names for the deployment

To retrieve the pod names for the deployment, use the following command:

```sh
kubectl -n cloudzero-agent get pods
```

> Note: Replace `cloudzero-agent` with the correct namespace for your deployment.

### 2. Identify the correct pod name

To inspect the logs of the validator containers, you need to identify the pod name for the `cloudzero-agent-server` pod.

### 3. Read the logs for the validator containers

The validator runs in multiple phases. To see the validation results:

**For pre-start validation (most common):**
```sh
kubectl -n cloudzero-agent logs <pod_name> -c env-validator-run
```

**For lifecycle validation logs:**
```sh
kubectl -n cloudzero-agent exec -ti -c cloudzero-agent-server <pod_name> -- cat cloudzero-agent-validator.log
```

**To check for validation errors quickly:**
```sh
kubectl -n cloudzero-agent exec -ti -c cloudzero-agent-server <pod_name> -- cat cloudzero-agent-validator.log | jq -r 'select(.checks) | .checks[] | select(.error) | "\(.name): \(.error)"'
```

**To capture full diagnostics for support:**
```sh
kubectl -n cloudzero-agent exec -ti -c cloudzero-agent-server <pod_name> -- cat cloudzero-agent-validator.log > cloudzero-diagnostics.log
```

> Note: The `env-validator-run` container performs the pre-start diagnostics during pod initialization. The lifecycle validation logs contain structured JSON output with detailed diagnostic information.

Diagnostics are run at 3 lifecycle phases of the `cloudzero-agent` pod deployment:

1. `Pod initialization` - basic configuration elements are validated, such as the API key and egress reachability.
2. `Post pod start` - the prometheus container runs the `post-start` checks, then posts a `cluster up` status to the Cloudzero API. Checks include validating the API key, capturing the Kubernetes version, inspecting the scrape configuration, and checking the kube-state-metrics service. The results are logged to the `/prometheus/cloudzero-validator.log` file in the container.
3. `Pre pod stop` - the prometheus container runs the `pre-stop` checks (usually none), then posts a `cluster down` status to the Cloudzero API.

Based on the above statements, it is also possible to diagnose from the perspective of the prometheus container. To inspect the logs, use the following command, replacing the pod name with that of your current deployment:

```sh
kubectl -n $NS exec -ti -c cloudzero-agent-server cloudzero-agent-server-766b4865dc-nrwc5 -- sh -c 'cat cloudzero-agent-validator.log'
```

> Remember to use the correct namespace and pod identity.

### 4. Get the correct validator container name

If you're unsure of the exact container name, first list all containers:

```sh
kubectl -n cloudzero-agent get pod <pod_name> -o jsonpath='{.spec.containers[*].name}'
kubectl -n cloudzero-agent get pod <pod_name> -o jsonpath='{.spec.initContainers[*].name}'
```

Look for containers ending in `-run` for the validation results.

### 5. Interpret the Results

In the screenshot above, notice the `checks` section. This section allows you to view the results of the configured checks. For any checks that are not passing, an error message will be captured to help diagnose the problem.

---

## Troubleshooting

The CloudZero Agent has the following requirements:

1. It must be able to communicate with the `Kubernetes metrics server`.
2. It must be provided with a valid Cloudzero API Token.
3. It must be able to communicate with the Cloudzero API to send metrics.
4. It must be configured to collect the correct metrics and labels to the Cloudzero API.

Based on these 5 requirements, the checks have been designed to help identify problems quickly during a new deployment. Using the tool, and log output, it should be possible to confirm this information. 

**When contacting support**, please provide:
1. The diagnostic JSON file: `cloudzero-diagnostics.json` (created using the command in step 3)
2. Pod descriptions: `kubectl -n <namespace> describe all`
3. Any error output from the error checking command

Contact support@cloudzero.com with this information for assistance.
