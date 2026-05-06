# CloudZero Agent Kubernetes Operator

## What This Is

The CloudZero Agent Operator is a Kubernetes controller that continuously reconciles the CloudZero Agent's desired state against its actual state on the cluster. It replaces one-shot Helm Jobs and manual runbook steps with an automated, self-healing control loop. The first capability is TLS certificate lifecycle management for the agent's validating webhook: the operator detects missing or expiring certificates, generates new ones, patches the Secret and ValidatingWebhookConfiguration, and reports health via standard Kubernetes status conditions. An operator (rather than a Job or script) is the right pattern because the agent has multi-component state (certs, Jobs, webhook config, storage) that must be kept consistent across restarts and failures -- exactly the problem the controller-runtime reconciliation model is designed for.

## Why We Built It

Analysis of hundreds of customer records in the burn book identified 10 recurring issue patterns requiring manual intervention:

| # | Pattern | Priority | Customers | Status |
|---|---------|----------|-----------|--------|
| 1 | KSM / Agent Server OOMKill | HIGH | 8+ (Booking, Braze, DoorDash, etc.) | Phase 2 planned |
| 2 | Webhook TLS cert expiry + VWC drift | HIGH | 5+ (Booking, DraftKings, DoorDash) | Phase 1 done; VWC watch needed |
| 3 | Webhook unreachable (network/CNI) | HIGH | 5+ (Kilo Health, Syncron, IAS) | Not planned |
| 4 | Silent agent failures / false "connected" | HIGH | 7+ (Akuna Capital, Infoblox, Booking) | Partially (Phase 1 conditions) |
| 5 | PVC / volume mount failures | MEDIUM | 4+ (IAS, Braze, Wise) | Partially (Phase 1 storage) |
| 6 | Configuration errors | MEDIUM | 6+ (Infoblox, BMC, Acquia) | Not planned |
| 7 | Helm upgrade / GitOps failures | MEDIUM | 7+ (Booking, Wise, DraftKings) | Not planned |
| 8 | Job failures (init-cert, backfill) | MEDIUM | 4+ (DVAG, Booking, Invesco) | Needs diagnostics enhancement |
| 9 | Network egress / firewall blocking uploads | MEDIUM | 5+ (ChargePoint, Twilio, BMC) | Not planned |
| 10 | Label collection / webhook config bugs | LOW | 6+ (Twilio, Upstart, Shutterstock) | Not planned |

The common thread: state that should be continuously reconciled is instead managed by one-shot mechanisms (Jobs, manual `kubectl` commands) that fail silently.

## Architecture

### Directory Layout

```
operator/
  cmd/
    main.go                          # Entry point; wires manager, cert service, reconciler
  api/
    v1alpha1/
      cloudzeroagent_types.go        # CRD Go types (CloudZeroAgent, TLSSpec, TLSMode)
      groupversion_info.go           # GVK registration
      zz_generated.deepcopy.go       # Generated DeepCopy methods
  internal/
    controller/
      cloudzeroagent_controller.go   # Reconciler: TLS state machine, status conditions
      reconciler_unit_test.go        # Unit tests (fake client + mock CertManager)
      cloudzeroagent_controller_test.go  # Integration test scaffolding
      suite_test.go                  # Test suite bootstrap
    certmanager/
      interface.go                   # CertManager interface (4 methods)
      adapter.go                     # Adapter wrapping the root module's CertificateService
      mocks/
        cert_manager_mock.go         # Generated mock (mockgen)
  config/
    crd/bases/                       # Generated CRD YAML
    rbac/                            # Generated RBAC (from kubebuilder markers)
    manager/                         # Deployment manifest for the operator
    samples/                         # Example CloudZeroAgent CRs
    default/                         # Kustomize overlay combining all config
  Dockerfile                         # Multi-stage build (golang:alpine -> distroless)
  Makefile                           # Build, test, deploy targets
  go.mod                             # Separate Go module
```

### Separate Go Module

The operator is a **separate Go module** (`github.com/cloudzero/cloudzero-agent/operator`) with a `replace` directive pointing to the parent repo:

```
replace github.com/cloudzero/cloudzero-agent => ../
```

This keeps `controller-runtime` and its transitive dependencies (50+ packages) out of the main agent module's dependency tree. The agent's collector, shipper, and webhook binaries never pull in controller-runtime. The operator imports only what it needs from the root module (the `certificate` domain service and its types).

### Dependency Flow

```
operator/cmd/main.go
  -> controller-runtime (manager, client, scheme)
  -> operator/internal/controller (reconciler)
       -> operator/api/v1alpha1 (CRD types)
       -> operator/internal/certmanager (CertManager interface)
            -> certmanager.Adapter
                 -> github.com/cloudzero/cloudzero-agent/app/domain/certificate (root module)
```

## CRD: CloudZeroAgent

**Group:** `agent.cloudzero.com` | **Version:** `v1alpha1` | **Kind:** `CloudZeroAgent` | **Short name:** `cza`

### Spec

```yaml
apiVersion: agent.cloudzero.com/v1alpha1
kind: CloudZeroAgent
metadata:
  name: cloudzero-agent
  namespace: cloudzero
spec:
  tls:
    mode: managed                    # How certs are managed
    secretName: cloudzero-agent-tls  # K8s Secret holding TLS data
    webhookName: cloudzero-agent-webhook  # VWC to patch with CA bundle
    serviceName: cloudzero-agent-webhook  # DNS name for cert SANs
    algorithm: ECDSA                 # RSA | ECDSA | Ed25519
    keySize: 256                     # Bits (RSA: min 2048; ECDSA: 256/384/521; ignored for Ed25519)
    validityDuration: "8760h"        # Cert lifetime (1 year)
    renewalThreshold: "720h"         # Renew when < 30 days remain
```

### TLS Mode Values

| Mode | Behavior |
|------|----------|
| `managed` | Operator generates self-signed CA + server cert, patches Secret and VWC, rotates before expiry |
| `cert-manager` | Operator skips cert management entirely; cert-manager handles lifecycle |
| `user-supplied` | Operator skips cert management entirely; user manages certs externally |

### Status

```yaml
status:
  conditions:
    - type: CertificateValid
      status: "True"          # True | False | Unknown
      reason: CertificateCurrent
      message: "TLS certificate exists and is not near expiry"
      lastTransitionTime: "2026-04-01T12:05:00Z"
      observedGeneration: 1
```

**Print columns:** TLS Mode, Certificate Valid, Age.

## Reconciler Design

### TLS Reconciliation State Machine

```
Reconcile(CloudZeroAgent)
  |
  |-- mode == cert-manager or user-supplied?
  |     YES --> set CertificateValid=Unknown/ExternallyManaged --> requeue 24h
  |
  |-- mode == managed
  |     |
  |     |-- parse renewalThreshold and validityDuration
  |     |     FAIL --> set CertificateValid=False/InvalidConfig --> requeue 24h
  |     |
  |     |-- ValidateExisting(secret)?
  |     |     |
  |     |     YES --> ValidateExpiry(secret, threshold)?
  |     |     |         |
  |     |     |         YES --> set CertificateValid=True/CertificateCurrent --> requeue 24h
  |     |     |         |
  |     |     |         NO  --> (fall through to generate)
  |     |     |
  |     |     NO  --> (fall through to generate)
  |     |
  |     |-- Generate(serviceName, namespace, keySize, validity, algorithm)
  |     |     FAIL --> set CertificateValid=False/GenerationFailed --> requeue 24h
  |     |
  |     |-- UpdateResources(secret, webhook, certData)
  |     |     FAIL --> set CertificateValid=False/InstallFailed --> requeue 24h
  |     |
  |     |-- set CertificateValid=True/CertificateInstalled --> requeue 24h
  |
  |-- unknown mode --> set CertificateValid=False/InvalidMode --> requeue 24h
```

Every path ends with a 24-hour requeue (`defaultRequeueInterval`). The reconciler also triggers on any change to the `CloudZeroAgent` resource (via the standard controller-runtime watch).

### CertManager Interface

```go
type CertManager interface {
    ValidateExisting(ctx, namespace, secretName) (bool, error)
    ValidateExpiry(ctx, namespace, secretName, threshold) (bool, error)
    Generate(ctx, serviceName, namespace, keySize, validity, algorithm) (*CertificateData, error)
    UpdateResources(ctx, namespace, secretName, webhookName, certData) error
}
```

The interface exists for **testability and decoupling**. The reconciler depends only on these four methods. In production, `certmanager.Adapter` wraps the root module's `CertificateService`. In tests, `certmocks.MockCertManager` (generated by mockgen) replaces it entirely -- no Kubernetes API server needed.

## Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| Separate Go module for operator | Keeps controller-runtime (50+ transitive deps) out of the main agent module. Agent binaries stay lean. |
| `replace` directive in go.mod | Points `github.com/cloudzero/cloudzero-agent => ../` so the operator can import the root module's domain layer without publishing a separate package. |
| Fake client for tests (not envtest) | Tests run in < 1 second with no external binaries. The fake client + mock CertManager covers all reconciler branches without a real API server. |
| mockgen via `go run` (not binary install) | `go run go.uber.org/mock/mockgen@v0.6.0` in the `//go:generate` directive avoids requiring mockgen in PATH. Pinned version ensures reproducibility. |
| Single status flush per reconcile | Every code path calls `setCondition` exactly once, which does a single `Status().Update()`. Avoids update conflicts from multiple writes. |
| 24-hour requeue interval | Certificates have long lifetimes (default 1 year) and 30-day renewal windows. Daily checks are more than sufficient; shorter intervals would be wasted API calls. |
| CertManager interface (4 methods) | Minimal surface area. Each method maps to one step of the reconciliation flow. Easy to mock, easy to reason about. |
| Adapter pattern for domain reuse | `certmanager.Adapter` wraps the existing `CertificateService` from `app/domain/certificate`. No logic is duplicated -- the operator delegates all crypto and K8s operations to the root module. |
| Dockerfile builds from repo root | The `replace` directive means the build context must include both the operator and root module source. The Dockerfile copies both, then builds from `operator/`. |
| Distroless runtime image | Minimal attack surface. No shell, no package manager. Only the compiled binary runs as non-root (UID 65532). |

## Testing

### Running Tests

From the `operator/` directory:

```bash
# Unit tests only (fast, no external deps)
cd operator && go test ./internal/controller/ -v

# Full test suite (includes envtest setup for integration scaffolding)
cd operator && make test

# Lint
cd operator && make lint
```

### What's Covered

The unit tests in `reconciler_unit_test.go` cover every branch of the reconciler:

| Test | Scenario |
|------|----------|
| `NoCertExists_GeneratesAndInstalls` | No existing cert -> generate -> install -> CertificateValid=True |
| `CertValidAndCurrent_NoGenerate` | Cert exists and not near expiry -> no-op -> CertificateValid=True |
| `CertNearExpiry_Regenerates` | Cert exists but within renewal threshold -> regenerate -> install |
| `GenerateFails_ConditionFalse` | Generate returns error -> CertificateValid=False/GenerationFailed |
| `InstallFails_ConditionFalse` | UpdateResources returns error -> CertificateValid=False/InstallFailed |
| `CertManagerMode_SetsUnknown` | mode=cert-manager -> CertificateValid=Unknown/ExternallyManaged |
| `UserSuppliedMode_SetsUnknown` | mode=user-supplied -> CertificateValid=Unknown/ExternallyManaged |
| `InvalidRenewalThreshold` | Bad duration string -> CertificateValid=False/InvalidConfig |
| `ResourceNotFound_NoError` | Deleted CR -> no error, no requeue |

### Why Fake Client Over envtest

- **Speed**: Tests complete in under 1 second. envtest requires downloading and running etcd + kube-apiserver binaries.
- **Determinism**: No race conditions from real API server timing. The fake client is synchronous.
- **Simplicity**: The reconciler's interesting logic is the state machine, not API server interactions. Mocking the CertManager interface isolates what matters.
- **CI-friendly**: No external binaries to install or cache.

## Running Locally

Prerequisites: a running Kubernetes cluster (KIND works) with `kubectl` configured.

```bash
# 1. Start a KIND cluster (from repo root)
make kind-up

# 2. Install the CRD
cd operator && make install

# 3. Create a sample CloudZeroAgent resource
kubectl apply -f operator/config/samples/local-agent-cr.yaml

# 4. Run the operator (connects to cluster via KUBECONFIG)
cd operator && make run

# 5. Check status
kubectl get cloudzeroagent -o wide
kubectl describe cloudzeroagent cloudzero-agent

# 6. Cleanup
cd operator && make uninstall
```

For Docker-based deployment:

```bash
# Build from repo root (required because of replace directive)
docker build -f operator/Dockerfile -t cloudzero-agent-operator:latest .

# Deploy to cluster
cd operator && make deploy IMG=cloudzero-agent-operator:latest
```

## Roadmap

### Phase 1.5: In-Cluster Deployment (done)

- Operator Dockerfile and Kustomize manifests exist
- Deployment, ServiceAccount, ClusterRole, ClusterRoleBinding generated from RBAC markers
- CRD installed via `make install` or Kustomize
- Leader election support wired (flag `--leader-elect`)

### Phase 2: Memory Pressure Management (planned)

- Watch pod metrics via `metrics.k8s.io` API for agent components (KSM, collector, aggregator, webhook)
- Three modes: `Observe` (set conditions), `Recommend` (emit events with sizing guidance), `AutoRemediate` (patch Deployment limits within bounds, trigger rolling restart)
- Backfill-aware: suppress KSM pressure alerts during active backfill Jobs
- New CRD fields: `spec.resourceManagement` with mode, pressure threshold, scale-up step, cooldown, per-component memory bounds
- New status: `MemoryPressure` condition, `ComponentMemory` status array
- Detailed implementation plan exists in `docs/operator-memory-pressure-plan.md`

### UI (planned)

- Phase 1: JSON status API endpoint served by the operator process (reads from informer cache)
- Phase 2: Embedded HTML dashboard via Go `embed` (no frontend build tooling)
- Phase 3: `kubectl cz` plugin for terminal-based status inspection
- Phase 4: Enriched data (cert details, pod status, events, PVC health)
- Detailed plan in `docs/operator-ui-proposal.md`

### Phase 3: Remote Configuration (planned, not yet started)

Allow the operator to poll a CloudZero API endpoint for config updates (label scrape rules, resource exclusions, feature flags) and apply them without a Helm upgrade.

Key design decisions:
- **Opt-in** via `spec.remoteConfig.enabled: true`
- **CRD spec wins by default**; remote config is lowest-priority unless `spec.remoteConfig.localOverride: false` (the default)
- **Automatic rollback** if new config fails agent health checks (OpAMP pattern)
- **GitOps-safe** — applied to operator state/status, not Deployment specs directly
- **Graceful degradation** — holds last known good config if API unreachable; sets `RemoteConfigStale` condition
- Status reported in `.status.remoteConfig` with `Applied | Applying | Failed | Stale | Disabled`

See `docs/operator-proposal.md` (Phase 3) and `docs/remote-config-research.md` for full design and prior art.

### Burn-Book-Derived Items (not yet planned)

| Item | Source Pattern |
|------|---------------|
| VWC watch -- recreate if pruned (ArgoCD), verify caBundle matches Secret | Pattern 2 |
| Active webhook health check -- synthetic admission review | Pattern 2 |
| Network pre-flight checks (non-RFC1918 pod IPs, NetworkPolicy, DNS) | Pattern 3 |
| Per-container readiness checks (catch sidecar-running-but-main-dead) | Pattern 4 |
| DataFlowStale condition (shipper upload monitoring) | Pattern 4 |
| StorageHealthy condition (Multi-Attach, PVC status, emptyDir usage) | Pattern 5 |
| ConfigurationValid condition (API key, cloud account ID, IMDS) | Pattern 6 |
| Pre-upgrade validation (PVC compat, PDB policy, ConfigMap schema) | Pattern 7 |
| EgressConnectivity condition (API reachability, S3 upload, TLS handshake) | Pattern 9 |
| LabelFlowHealthy condition (VWC audit, backfill CronJob monitoring) | Pattern 10 |
