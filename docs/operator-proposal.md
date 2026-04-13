# CloudZero Agent Operator Proposal

## What

Introduce a Kubernetes operator for the CloudZero Agent that continuously reconciles the agent's desired state against its actual state on the cluster. This replaces the current model — where state is managed by one-shot Helm Jobs, in-memory lifecycle tracking, and manual runbook steps — with an automated, self-healing control loop.

The operator would be defined by a `CloudZeroAgent` CRD. A controller watches that resource and ensures:

- TLS certificates are valid, not expired, and correctly referenced by the webhook configuration
- Installation Jobs have completed successfully, with failed jobs retried and dependency ordering enforced
- Lifecycle state (init, healthy, degraded) is persisted in `.status.conditions` and survives pod restarts
- Storage health (disk pressure, PVC availability, SQLite persistence) is monitored and surfaced on the cluster

## Why

The current agent has several gaps in state management that require manual intervention or leave the system silently broken:

| Problem | Impact |
|---------|--------|
| Built-in cert path generates 100-year certs with no rotation | Certs are never renewed; rotation requires manual Secret deletion and Job re-run |
| Cert Job has `restartPolicy: Never` | A transient failure (e.g. RBAC not yet propagated) leaves the webhook non-functional with no retry |
| Jobs have no dependency ordering | The webhook deployment can start before the cert Job completes |
| Lifecycle state is in-memory only | Pod restart loses all diagnostic results; the CloudZero platform has no visibility until the next `helm upgrade` |
| SQLite resource metadata is in-memory by default | All webhook resource metadata is lost on pod restart; the backfill CronJob compensates but introduces a gap |
| Disk pressure is not surfaced to the cluster | Operators have no cluster-native signal that storage is exhausted |

An operator encodes the knowledge currently in runbooks and docs directly into the control loop, so these conditions are detected and resolved automatically rather than reactively.

A related class of problem is **component memory pressure**. The `cloudzero-kube-state-metrics` deployment and agent components periodically hit memory limits, requiring a human operator to notice the OOMKill and manually increase resource limits. This is operational toil that an operator can eliminate.

## Why the Operator Pattern

A Kubernetes operator is the idiomatic solution for managing stateful application lifecycle on a cluster. It extends the Kubernetes API with a domain-specific resource type (`CloudZeroAgent`) and runs a controller that continuously reconciles the cluster toward the desired state declared in that resource.

This fits the CloudZero Agent well because:

- **The agent has multi-component state** (certs, Jobs, webhook config, storage) that must be kept consistent with each other — exactly the problem operators are designed for
- **Reconciliation is idempotent by design** — the controller can safely re-run after crashes, restarts, or partial failures
- **Status conditions** (`Available`, `Degraded`, `Progressing`) give cluster operators a standard, machine-readable view of agent health without needing to inspect logs
- **The existing hexagonal architecture** maps cleanly onto an operator — domain services and interfaces slot directly into the reconciler as business logic, with the controller becoming a new primary adapter

## Why Kubebuilder

Kubebuilder is the standard scaffolding framework for building operators in Go. It generates the CRD boilerplate, RBAC annotations, controller wiring, and webhook scaffolding, letting development focus on reconciler logic rather than plumbing.

Specifically:

- Built on `controller-runtime`, the same library used by cert-manager and the Prometheus Operator
- Generates CRD schemas from Go structs with validation markers — no hand-written JSON Schema
- Integrates with the existing Go module and Makefile-based build system
- First-class support for admission webhooks, which the agent already uses
- Large ecosystem and community — kubebuilder operators are well-understood by the Kubernetes community

## Proposed CRD Shape (Sketch)

```yaml
apiVersion: cloudzero.com/v1alpha1
kind: CloudZeroAgent
metadata:
  name: my-cluster
spec:
  apiKeySecret: cloudzero-api-key
  clusterName: production-eks
  region: us-east-1
  tls:
    mode: managed         # managed | cert-manager | user-supplied
  features:
    webhook: true
    backfill: true
    gpuMetrics: false
  resourceManagement:
    mode: Observe       # Observe | Recommend | AutoRemediate
    components:
      kubeStateMetrics:
        memory:
          min: 256Mi
          max: 2Gi
      collector:
        memory:
          min: 128Mi
          max: 1Gi
status:
  conditions:
    - type: Available
      status: "True"
    - type: CertificateValid
      status: "True"
    - type: BackfillComplete
      status: "False"
      reason: JobFailed
      message: "backfill-job failed after 3 attempts"
    - type: MemoryPressure
      status: "True"
      reason: KSMHighUsage
      message: "cloudzero-kube-state-metrics memory usage at 92% of limit"
```

## Proposed Scope

### Phase 1 — Core State Management

1. Scaffold operator with kubebuilder, define `CloudZeroAgent` CRD
2. Reconcile TLS certificate state — detect expiry, trigger renewal, patch webhook config
3. Reconcile Job completion — enforce ordering, retry failures, surface results in `.status`
4. Persist lifecycle state in `.status.conditions` using standard Kubernetes condition types
5. Expose storage health (disk pressure, PVC status) as status conditions

### Phase 1.5 — In-Cluster Deployment

Before Phase 2, the operator needs to run inside the cluster rather than locally. This involves:

1. **Build and push the operator image** — `operator/Dockerfile` already exists; integrate its build into the Helm chart release pipeline (alongside the existing agent images in `docker/`)
2. **Add operator manifests to the Helm chart** — deploy the operator as a `Deployment` in the same namespace as the agent, with its `ServiceAccount`, `ClusterRole`, and `ClusterRoleBinding` generated from the kubebuilder RBAC markers
3. **Add `CloudZeroAgent` CRD to the chart** — include `operator/config/crd/bases/` in the Helm chart so the CRD is installed as part of `helm install`
4. **Create the `CloudZeroAgent` CR as a Helm template** — auto-generate a CR populated from `values.yaml` (TLS settings, secret name, webhook name) so users get operator management out of the box
5. **Leader election** — enable controller-runtime leader election so the operator `Deployment` can safely run with multiple replicas

This is a prerequisite for production use. Until it is complete, the operator must be run locally with `KUBECONFIG` pointed at the target cluster.

### Phase 2 — Resource Management

6. Observe memory usage of agent components via the Kubernetes Metrics API
7. Surface memory pressure as `.status.conditions` with component-level detail (`Observe` mode)
8. Emit Kubernetes `Events` recommending limit increases (`Recommend` mode)
9. Automatically patch `Deployment` resource limits within configured bounds and trigger rolling restarts (`AutoRemediate` mode)

The `mode` field gives human operators control over automation aggressiveness. Phase 2 ships in `Observe` mode by default, with `AutoRemediate` opt-in. This approach is similar to a Vertical Pod Autoscaler but domain-aware — the operator understands CloudZero-specific context (e.g. a backfill Job running is expected to spike KSM memory) rather than applying a generic algorithm.
