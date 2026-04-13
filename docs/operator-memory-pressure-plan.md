# Implementation Plan: Memory Pressure Management (Operator Phase 2)

## 1. CRD Type Changes

**File:** `operator/api/v1alpha1/cloudzeroagent_types.go`

### New types

```go
// ResourceManagementMode controls how the operator responds to memory pressure.
// +kubebuilder:validation:Enum=Observe;Recommend;AutoRemediate
type ResourceManagementMode string

const (
    ResourceManagementModeObserve       ResourceManagementMode = "Observe"
    ResourceManagementModeRecommend     ResourceManagementMode = "Recommend"
    ResourceManagementModeAutoRemediate ResourceManagementMode = "AutoRemediate"
)

// ComponentName identifies a CloudZero agent component.
// +kubebuilder:validation:Enum=kubeStateMetrics;collector;aggregator;webhook
type ComponentName string

const (
    ComponentKubeStateMetrics ComponentName = "kubeStateMetrics"
    ComponentCollector        ComponentName = "collector"
    ComponentAggregator       ComponentName = "aggregator"
    ComponentWebhook          ComponentName = "webhook"
)

// MemoryBounds defines the min/max memory limits the operator may set.
type MemoryBounds struct {
    // +kubebuilder:validation:Pattern=`^[0-9]+(Ki|Mi|Gi|Ti)?$`
    Min string `json:"min"`
    // +kubebuilder:validation:Pattern=`^[0-9]+(Ki|Mi|Gi|Ti)?$`
    Max string `json:"max"`
}

type ComponentResourceSpec struct {
    Memory MemoryBounds `json:"memory"`
}

type ResourceManagementSpec struct {
    // +kubebuilder:default=Observe
    Mode ResourceManagementMode `json:"mode"`

    // Memory usage % of limit that constitutes pressure. Default 85.
    // +kubebuilder:default=85
    // +kubebuilder:validation:Minimum=50
    // +kubebuilder:validation:Maximum=99
    PressureThresholdPercent int `json:"pressureThresholdPercent,omitempty"`

    // How much to increase the limit by on each AutoRemediate patch (% of current limit). Default 25.
    // +kubebuilder:default=25
    // +kubebuilder:validation:Minimum=10
    // +kubebuilder:validation:Maximum=100
    ScaleUpStepPercent int `json:"scaleUpStepPercent,omitempty"`

    // Minimum time between successive AutoRemediate patches for the same component.
    // +kubebuilder:default="10m"
    CooldownPeriod string `json:"cooldownPeriod,omitempty"`

    // +optional
    Components map[ComponentName]ComponentResourceSpec `json:"components,omitempty"`
}
```

### Extend `CloudZeroAgentSpec`

```go
type CloudZeroAgentSpec struct {
    TLS                TLSSpec                 `json:"tls,omitempty"`
    // +optional
    ResourceManagement *ResourceManagementSpec `json:"resourceManagement,omitempty"`
}
```

### Extend `CloudZeroAgentStatus`

```go
type ComponentMemoryStatus struct {
    ComponentName     ComponentName `json:"componentName"`
    CurrentUsageBytes int64        `json:"currentUsageBytes"`
    LimitBytes        int64        `json:"limitBytes"`
    UsagePercent      int          `json:"usagePercent"`
    // +optional
    LastPatchedAt  *metav1.Time `json:"lastPatchedAt,omitempty"`
    LastObservedAt metav1.Time  `json:"lastObservedAt"`
}

type CloudZeroAgentStatus struct {
    Conditions      []metav1.Condition      `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
    // +optional
    ComponentMemory []ComponentMemoryStatus `json:"componentMemory,omitempty"`
}
```

### New print columns (on `CloudZeroAgent` struct markers)

```go
// +kubebuilder:printcolumn:name="Memory Pressure",type=string,JSONPath=`.status.conditions[?(@.type=="MemoryPressure")].status`
// +kubebuilder:printcolumn:name="Resource Mgmt",type=string,JSONPath=`.spec.resourceManagement.mode`
```

### New condition type constants

- `ConditionMemoryPressure = "MemoryPressure"` — True when any component exceeds threshold
- `ConditionResourceLimitPatched = "ResourceLimitPatched"` — True when operator has patched a Deployment's limits

---

## 2. Metrics API Client

**New file:** `operator/internal/metricsapi/interface.go`

Follows the exact pattern of `certmanager/interface.go`:

```go
//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=mocks/metrics_client_mock.go -package=mocks -source=interface.go

package metricsapi

import "context"

type PodMemoryUsage struct {
    PodName      string
    Namespace    string
    UsageBytes   int64
}

type MetricsClient interface {
    // GetPodMemoryUsage queries metrics.k8s.io/v1beta1 PodMetrics for pods
    // matching the given label selector in the given namespace.
    // Returns the sum of memory usage across all containers in each pod.
    GetPodMemoryUsage(ctx context.Context, namespace string, labelSelector string) ([]PodMemoryUsage, error)
}
```

**New file:** `operator/internal/metricsapi/adapter.go`

Uses `k8s.io/metrics/pkg/client/clientset/versioned` to query PodMetrics:

```go
func (a *Adapter) GetPodMemoryUsage(ctx context.Context, namespace string, labelSelector string) ([]PodMemoryUsage, error) {
    podMetricsList, err := a.metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, metav1.ListOptions{
        LabelSelector: labelSelector,
    })
    // Sum memory across all containers in each pod
    for _, pm := range podMetricsList.Items {
        var totalBytes int64
        for _, c := range pm.Containers {
            if mem, ok := c.Usage["memory"]; ok {
                totalBytes += mem.Value()
            }
        }
        // append PodMemoryUsage{...}
    }
}
```

**New file:** `operator/internal/metricsapi/mocks/metrics_client_mock.go` — generated by `go generate`.

**go.mod change:** Add `k8s.io/metrics v0.35.3` (matches existing `k8s.io/apimachinery v0.35.3`).

---

## 3. Reconciler Changes

**File:** `operator/internal/controller/cloudzeroagent_controller.go`

### Restructure `Reconcile` into sub-paths

```
Reconcile(ctx, req)
  1. Get the CloudZeroAgent CR
  2. reconcileTLS(ctx, agent)               -- existing logic, extracted to method
  3. reconcileResourceManagement(ctx, agent) -- new
  4. Single status update with merged conditions
```

Refactor `setCondition` so conditions accumulate in-memory and flush once at end — avoids update conflicts.

### New fields on `CloudZeroAgentReconciler`

```go
type CloudZeroAgentReconciler struct {
    client.Client
    Scheme        *runtime.Scheme
    CertManager   certmanager.CertManager
    MetricsClient metricsapi.MetricsClient
    EventRecorder record.EventRecorder
}
```

### New RBAC markers

```go
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=metrics.k8s.io,resources=pods,verbs=get;list
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch
```

### Component-to-selector registry

Defined in `resource_management.go`:

```go
type componentInfo struct {
    LabelSelector string
    ContainerName string
}

var componentRegistry = map[agentv1alpha1.ComponentName]componentInfo{
    agentv1alpha1.ComponentKubeStateMetrics: {LabelSelector: "app.kubernetes.io/name=ksm",              ContainerName: "ksm"},
    agentv1alpha1.ComponentCollector:        {LabelSelector: "app.kubernetes.io/component=collector",   ContainerName: "collector"},
    agentv1alpha1.ComponentAggregator:       {LabelSelector: "app.kubernetes.io/component=aggregator",  ContainerName: "aggregator"},
    agentv1alpha1.ComponentWebhook:          {LabelSelector: "app.kubernetes.io/component=webhook",     ContainerName: "webhook"},
}
```

> Note: actual label selectors must be verified against Helm templates before finalizing.

### Requeue interval

```go
resourceManagementRequeueInterval = 60 * time.Second
```

The final returned `Result` uses `min(tlsRequeue, resourceMgmtRequeue)`.

---

## 4. Event Emission

Wire in `cmd/main.go`:

```go
eventRecorder := mgr.GetEventRecorderFor("cloudzero-agent-operator")
```

In `reconcileResourceManagement`, when pressure is detected in Recommend/AutoRemediate mode:

```go
r.EventRecorder.Eventf(agent, corev1.EventTypeWarning, "MemoryPressureDetected",
    "Component %s memory usage at %d%% of limit (%s/%s). Recommend increasing memory limit to at least %s.",
    componentName, usagePercent, formatBytes(currentUsage), formatBytes(limitBytes), formatBytes(recommendedLimit))
```

When a patch is applied:

```go
r.EventRecorder.Eventf(agent, corev1.EventTypeNormal, "ResourceLimitPatched",
    "Patched Deployment %s/%s container %s memory limit from %s to %s",
    namespace, deploymentName, containerName, oldLimit, newLimit)
```

---

## 5. AutoRemediate Logic

**File:** `operator/internal/controller/resource_management.go`

### Computing the new limit

```go
func computeNewLimit(currentLimit resource.Quantity, stepPercent int, bounds MemoryBounds) (resource.Quantity, error) {
    currentBytes := currentLimit.Value()
    newBytes := currentBytes + (currentBytes * int64(stepPercent) / 100)

    // Parse and clamp to [min, max]
    minQ, maxQ := parseBounds(bounds)
    if newBytes < minQ.Value() { newBytes = minQ.Value() }
    if newBytes > maxQ.Value() { newBytes = maxQ.Value() }

    if currentLimit.Value() >= maxQ.Value() {
        return currentLimit, ErrAlreadyAtMax
    }
    return *resource.NewQuantity(newBytes, resource.BinarySI), nil
}
```

### Patching the Deployment

```go
func (r *CloudZeroAgentReconciler) patchDeploymentMemoryLimit(
    ctx context.Context,
    deployment *appsv1.Deployment,
    containerName string,
    newLimit resource.Quantity,
) error {
    patch := client.MergeFrom(deployment.DeepCopy())

    for i := range deployment.Spec.Template.Spec.Containers {
        c := &deployment.Spec.Template.Spec.Containers[i]
        if c.Name == containerName {
            c.Resources.Limits[corev1.ResourceMemory] = newLimit
            // Clamp request down if it would exceed new limit
            if req := c.Resources.Requests[corev1.ResourceMemory]; req.Cmp(newLimit) > 0 {
                c.Resources.Requests[corev1.ResourceMemory] = newLimit
            }
        }
    }
    // Trigger rolling restart
    deployment.Spec.Template.Annotations["cloudzero.com/restartedAt"] = time.Now().Format(time.RFC3339)

    return r.Patch(ctx, deployment, patch)
}
```

### Idempotency guards

- Skip if `currentLimit >= newLimit` (already at desired value)
- Skip if within cooldown: `time.Since(lastPatchedAt) < cooldownPeriod`
- If at `bounds.Max` and pressure persists: set `MemoryPressure=True/AtMaxBound`, emit Warning, no patch

---

## 6. Context Awareness — Backfill Job Suppression

When a backfill Job is running, KSM memory spikes are expected (KSM tracks additional resource metadata objects). Only applies to `ComponentKubeStateMetrics`.

```go
func (r *CloudZeroAgentReconciler) isBackfillRunning(ctx context.Context, namespace string) (bool, error) {
    var jobList batchv1.JobList
    if err := r.List(ctx, &jobList,
        client.InNamespace(namespace),
        client.MatchingLabels{"job-type": "backfill"},
    ); err != nil {
        return false, err
    }
    for _, job := range jobList.Items {
        if job.Status.CompletionTime == nil && job.Status.Active > 0 {
            return true, nil
        }
    }
    return false, nil
}
```

**Behavior when backfill is running:**

| Mode | Behavior |
|------|----------|
| Observe | Do NOT set `MemoryPressure=True`; include "backfill running, spike expected" in message |
| Recommend | Emit Normal (not Warning) event noting spike is expected |
| AutoRemediate | Skip patching entirely |

---

## 7. Tests

**File:** `operator/internal/controller/resource_management_test.go`

| # | Test | What it verifies |
|---|------|-----------------|
| 1 | `ResourceManagement_Nil_NoAction` | No metrics calls, no conditions when spec is nil |
| 2 | `ObserveMode_NormalUsage` | `MemoryPressure=False`, `ComponentMemory` status populated |
| 3 | `ObserveMode_HighUsage` | `MemoryPressure=True` with component detail in message |
| 4 | `RecommendMode_HighUsage_EmitsEvent` | Warning event emitted with recommendation text |
| 5 | `AutoRemediate_HighUsage_PatchesDeployment` | Deployment limit increased, `restartedAt` annotation set |
| 6 | `AutoRemediate_ClampToMax` | Patch clamps to max bound, not beyond |
| 7 | `AutoRemediate_AlreadyAtMax_NoOp` | No patch, Warning event about manual intervention |
| 8 | `AutoRemediate_CooldownRespected` | No patch when within cooldown period |
| 9 | `BackfillRunning_KSM_Suppressed` | KSM pressure not raised while backfill Job is active |
| 10 | `BackfillRunning_OtherComponent_NotSuppressed` | Collector pressure IS raised during backfill |
| 11 | `MetricsUnavailable_ConditionUnknown` | `MemoryPressure=Unknown/MetricsUnavailable` |
| 12 | `InvalidBounds_ConditionUnknown` | `MemoryPressure=Unknown/InvalidConfig` when min > max |
| 13 | `TestComputeNewLimit` (table-driven) | Bounds clamping, step math, ErrAlreadyAtMax |

---

## 8. Files Summary

### New files

| File | Purpose |
|------|---------|
| `operator/internal/metricsapi/interface.go` | `MetricsClient` interface + mockgen directive |
| `operator/internal/metricsapi/adapter.go` | Real implementation via `k8s.io/metrics` |
| `operator/internal/metricsapi/mocks/metrics_client_mock.go` | Generated mock |
| `operator/internal/controller/resource_management.go` | `reconcileResourceManagement`, `computeNewLimit`, `patchDeploymentMemoryLimit`, `isBackfillRunning`, component registry |
| `operator/internal/controller/resource_management_test.go` | All 13 unit tests |

### Modified files

| File | Changes |
|------|---------|
| `operator/api/v1alpha1/cloudzeroagent_types.go` | New types; extend Spec and Status; add print columns |
| `operator/internal/controller/cloudzeroagent_controller.go` | Add `MetricsClient`/`EventRecorder` fields; extract `reconcileTLS`; add RBAC markers; single status flush |
| `operator/cmd/main.go` | Create `metricsapi.Adapter`; create `EventRecorder`; inject into reconciler |
| `operator/go.mod` | Add `k8s.io/metrics v0.35.3` |
| `operator/internal/controller/reconciler_unit_test.go` | Update `newTestReconciler` for new fields (nil-safe) |

**After type changes:** run `make generate && make manifests` to regenerate CRD and deepcopy. Update `helm/templates/operator-crd.yaml` from the new generated output.

---

## 9. End-to-End Verification (KIND)

```bash
# 1. Start cluster with metrics-server
make kind-up
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
kubectl -n kube-system patch deployment metrics-server --type=json \
  -p '[{"op":"add","path":"/spec/template/spec/containers/0/args/-","value":"--kubelet-insecure-tls"}]'
kubectl -n kube-system rollout status deployment/metrics-server --timeout=120s

# 2. Install agent
make helm-install helm-wait

# 3. Observe mode — check ComponentMemory populated
kubectl get cloudzeroagent -o jsonpath='{.items[0].status.componentMemory}' | python3 -m json.tool

# 4. Recommend mode — create artificial load, check for Warning events
kubectl patch cloudzeroagent cloudzero-agent --type=merge \
  -p '{"spec":{"resourceManagement":{"mode":"Recommend"}}}'
for i in $(seq 1 5000); do kubectl create cm dummy-$i --from-literal=k=v; done
kubectl describe cloudzeroagent cloudzero-agent | grep -A10 Events

# 5. AutoRemediate — verify Deployment limit patched
kubectl patch cloudzeroagent cloudzero-agent --type=merge \
  -p '{"spec":{"resourceManagement":{"mode":"AutoRemediate","components":{"kubeStateMetrics":{"memory":{"min":"128Mi","max":"1Gi"}}}}}}'
# Wait 60s for reconcile
kubectl get deployment -l app.kubernetes.io/name=ksm \
  -o jsonpath='{.items[0].spec.template.spec.containers[0].resources.limits.memory}'

# 6. Backfill suppression
kubectl create job backfill-test --image=busybox -- sleep 300
kubectl label job backfill-test job-type=backfill
# Verify no MemoryPressure=True while job is running

# 7. Cleanup
for i in $(seq 1 5000); do kubectl delete cm dummy-$i --ignore-not-found; done
kubectl delete job backfill-test
make kind-down
```
