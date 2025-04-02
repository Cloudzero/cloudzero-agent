# **Helm Chart Job Execution and Upgrade Behavior**

This document outlines the expected behavior of the **CloudZero Agent Jobs** under different upgrade scenarios, including version upgrades and forced upgrades (`--force`).

## **Jobs Overview**

The Helm chart deploys two Jobs:

- **`backfill`**: Ensures that the current state of the cluster is captured and uploaded to the CloudZero platform.
- **`init-cert`**: Generates or renews the internal certificate needed for communication between Kubernetes and the webhook server.

Both the **`backfill`** and **`init-cert`** Jobs expire after a configurable period of time, ensuring that re-initialization can occur on changes to the chart.

---

## **Upgrade Scenarios and Job Behavior**

| Upgrade Scenario                         | `backfill` Job Behavior                           | `init-cert` Job Behavior                             |
| ---------------------------------------- | ------------------------------------------------- | ---------------------------------------------------- |
| **Standard version upgrade**             | Runs on every version/image upgrade               | Runs on every version/image upgrade                  |
| **Forced upgrade (`--force`)**           | Runs again after the Job is automatically deleted | Always runs and ensures a new certificate is created |
| **Upgrade Without Chart Version Change** | Runs again if Job TTL has expired                 | Runs again if Job TTL has expired                    |

---

## **ArgoCD Integration**

If installing this Helm chart using ArgoCD, set the following annotations in the `initBackfillJob` and `initCertJob` fields to ensure that ArgoCD does not constantly consider the Application out of sync:

```yaml
initBackfillJob:
  annotations:
    argocd.argoproj.io/hook: PostSync
    argocd.argoproj.io/hook-delete-policy: HookSucceeded
initCertJob:
  annotations:
    argocd.argoproj.io/hook: PostSync
    argocd.argoproj.io/hook-delete-policy: HookSucceeded
```

See the ArgoCD [Hook Deletion](https://argo-cd.readthedocs.io/en/stable/user-guide/resource_hooks/#hook-deletion-policies) documentation for further details.

---

## **Common Issues & Troubleshooting**

### **Issue: Forced Upgrade (`--force`) Fails Due to Running `backfill` Job**

**Problem:**

- If a **forced upgrade (`--force`)** is triggered while the **previous `backfill` Job is still running**, the upgrade will fail.
- This happens because **Kubernetes Jobs are immutable**—Helm cannot replace an existing running Job.
- The error in this case may include a message similar to:

```sh
Error: UPGRADE FAILED: failed to replace object: Job.batch "cloudzero-agent-backfill"
```

**Solution:**

1. **Manually delete the running Jobs and retry the upgrade**
   ```sh
   kubectl delete job -n <NAMESPACE> -l app.kubernetes.io/component=webhook-server
   helm upgrade --install <RELEASE_NAME> cloudzero/cloudzero-agent -n <NAMESPACE> --version <version> --force
   ```
   - This immediately removes all initialization Jobs, allowing Helm to recreate them.

---

## **Implementation Notes**

1. **Deployment Rollout:** The `init-cert` Job includes a mechanism to force a Deployment restart when it completes.
