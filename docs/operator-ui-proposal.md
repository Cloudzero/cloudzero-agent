# CloudZero Agent Operator UI Proposal

## 1. Introduction

This document proposes a User Interface for the CloudZero Agent Kubernetes Operator. The target audience is **cluster maintainers** -- the people responsible for ensuring the CloudZero Agent installation is healthy, properly configured, and operating correctly within their Kubernetes cluster.

The operator currently watches a `CloudZeroAgent` CRD (group `agent.cloudzero.com`, version `v1alpha1`) and reconciles TLS certificate state. The CRD status exposes `[]metav1.Condition` (currently `CertificateValid`), with additional conditions planned: Job completion, storage health, memory pressure, and overall availability. This UI must accommodate both the current narrow scope and the rich future state described in the operator proposal.

---

## 2. Ideas Brainstorm: What Cluster Maintainers Want to See

### 2.1 Health at a Glance

- **Overall agent health status**: A single top-level indicator (healthy / degraded / unavailable) derived from aggregated conditions.
- **Condition breakdown**: Each `metav1.Condition` shown with its status, reason, message, and last-transition time.
- **Time since last reconciliation**: How recently the operator last touched this resource.
- **Condition history timeline**: Visual timeline showing when conditions transitioned, useful for correlating with incidents.

### 2.2 TLS Certificate Details

- **Certificate validity window**: Not-before and not-after dates, shown against a visual timeline.
- **Time until expiry**: Countdown with color coding (green > 30 days, yellow 7-30 days, red < 7 days).
- **Renewal threshold marker**: Where on the validity timeline the operator will trigger renewal.
- **TLS mode**: Whether certs are managed, cert-manager, or user-supplied.
- **Algorithm and key size**: Displayed for audit purposes.
- **Secret name and webhook name**: Clickable references for quick `kubectl` commands.
- **Certificate chain details**: Subject, issuer, SANs, serial number.
- **Renewal history**: Log of when certificates were last generated or renewed.

### 2.3 Component Health (Phase 2+)

- **Pod status grid**: All agent-related pods (collector, aggregator, webhook, kube-state-metrics) with their phase, restart count, and resource usage.
- **Memory pressure indicators**: Per-component memory usage vs. limits, with the operator's resource management mode shown (Observe / Recommend / AutoRemediate).
- **Resource recommendations**: If in Recommend mode, show the operator's suggested limit increases.
- **Auto-remediation log**: History of automatic resource patches applied by the operator in AutoRemediate mode.

### 2.4 Job Management (Phase 1 future)

- **Job status table**: All operator-managed Jobs (cert init, backfill, config loader) with their completion status, attempt count, and duration.
- **Job dependency graph**: Visual representation of Job ordering enforced by the operator.
- **Failed Job details**: Logs or error messages from the most recent failed attempt.

### 2.5 Storage Health (Phase 1 future)

- **PVC status**: Bound/pending/lost state for any PVCs used by the agent.
- **Disk pressure**: Current usage percentage and threshold at which the operator raises a condition.
- **SQLite persistence status**: Whether the webhook is using persistent or ephemeral storage.

### 2.6 Configuration View

- **Spec display**: The full `CloudZeroAgentSpec` rendered in a readable format, not raw YAML.
- **Effective configuration**: What values are active (including defaults).
- **Drift detection**: Whether the running state matches the declared spec.
- **Cluster identity**: Cluster name, region, cloud account ID from the broader Helm values context.

### 2.7 Events and Audit

- **Kubernetes Events**: Filtered to the `CloudZeroAgent` resource and its managed resources (Secrets, Jobs, Deployments).
- **Reconciliation log**: Recent reconcile cycles with outcome (success, error, requeue) and duration.
- **Operator version and uptime**: The operator controller-manager's own health.

### 2.8 Actionable Guidance

- **Next steps**: When something is degraded, suggest concrete remediation steps.
- **Link to runbooks**: For each condition, link to relevant documentation.
- **Quick-copy kubectl commands**: For common diagnostic actions (describe resource, get logs, check events).

---

## 3. Technical Architecture Options

### 3.1 Option A: Operator-Served HTTP Status Endpoint

**Description**: Add an HTTP handler to the operator's controller-manager process that serves a status API (JSON) and optionally a lightweight HTML dashboard.

**How it works**:
- The operator already has HTTP listeners: health probes on `:8081` and metrics on the configured metrics address. Controller-runtime's manager supports adding arbitrary `Runnable` components.
- Add a new HTTP server (or mount additional handlers on the existing health probe server) that reads the `CloudZeroAgent` CR status and returns it as structured JSON.
- Optionally serve a single-page HTML dashboard using Go's `embed` package to bundle static assets into the binary.

**Data flow**: Operator process reads CRD status from its in-memory cache (the informer cache provided by controller-runtime) -- no additional Kubernetes API calls needed.

**Trade-offs**:
- (+) Zero additional deployment -- the dashboard lives inside the existing operator process.
- (+) Data is always fresh from the informer cache, sub-second latency.
- (+) No external dependencies (no Node.js, no separate frontend build).
- (+) Can be `kubectl port-forward`ed for immediate access.
- (+) Naturally secured behind Kubernetes RBAC (the Service is cluster-internal).
- (-) UI sophistication is limited by what can be embedded (static HTML/JS/CSS).
- (-) Couples UI lifecycle to operator lifecycle -- a UI bug could theoretically crash the operator.
- (-) Not accessible from outside the cluster without an Ingress.

### 3.2 Option B: kubectl Plugin (`kubectl cz status`)

**Description**: A standalone Go binary distributed as a kubectl plugin that queries the Kubernetes API directly and renders a rich terminal UI.

**How it works**:
- Uses `client-go` to fetch `CloudZeroAgent` resources and related objects (Secrets, Jobs, Pods, Events).
- Renders output using a terminal UI library (e.g., `charmbracelet/bubbletea` for interactive TUI, or `tablewriter` for static table output).
- Distributed as a single binary, installable via `kubectl krew` or direct download.

**Data flow**: Plugin talks directly to the Kubernetes API server using the user's kubeconfig.

**Trade-offs**:
- (+) No in-cluster deployment required beyond the CRD itself.
- (+) Terminal-native -- fits the workflow of cluster maintainers who live in the terminal.
- (+) Can be very rich: live-updating status, drill-down into related resources.
- (+) Zero coupling with the operator process.
- (+) Leverages the user's existing RBAC -- no additional auth to configure.
- (-) Must be separately installed by each user.
- (-) Cannot be bookmarked or shared as a URL.
- (-) Requires building and distributing a separate binary per platform.
- (-) Not suitable for NOC-style "dashboard on a screen" use cases.

### 3.3 Option C: Standalone Web Application

**Description**: A separate web application (React/Vue frontend, Go or Node.js backend) deployed alongside the operator.

**Trade-offs**:
- (+) Maximum UI flexibility and interactivity.
- (-) Significant additional deployment surface (extra container, service, possibly ingress).
- (-) Authentication and authorization must be solved separately.
- (-) Much higher development and maintenance cost.
- (-) Overkill for the current scope (a single CRD with a few conditions).

### 3.4 Option D: Kubernetes Dashboard Extension / Headlamp Plugin

**Description**: Build a plugin for an existing Kubernetes dashboard (Kubernetes Dashboard, Headlamp, or Lens).

**Trade-offs**:
- (+) Integrates into a tool the user may already have.
- (-) Ties the UI to a specific dashboard product.
- (-) Plugin APIs vary in maturity and capability.
- (-) Assumes the user has one of these dashboards deployed.

### 3.5 Recommended Approach: Hybrid of A and B

**Option A (operator-served endpoint) as the primary data layer, with Option B (kubectl plugin) as the primary user interface.** This combines the strengths of both:

1. The operator serves a lightweight `/status` JSON API endpoint -- trivial to add and provides a stable, documented data source.
2. A `kubectl cz` plugin provides the primary interactive experience for cluster maintainers.
3. The operator optionally serves a minimal embedded HTML dashboard for quick browser access via `kubectl port-forward`.

The kubectl plugin can talk directly to the Kubernetes API (bypassing the operator endpoint) for maximum reliability, and can also optionally hit the operator's `/status` endpoint for pre-aggregated data.

---

## 4. Detailed Implementation Plan

### 4.1 Phase 1: Operator Status API Endpoint

**Goal**: Add a JSON API endpoint to the operator that returns the current agent health status.

#### New package: `operator/internal/statusapi`

Create a new package that implements `manager.Runnable` from controller-runtime. This server:

- Binds to a configurable address (flag `--status-bind-address`, default `:9090`).
- Serves `GET /status` returning JSON with the aggregated `CloudZeroAgent` status.
- Serves `GET /status/{name}` for a specific `CloudZeroAgent` resource.
- Uses the manager's informer cache (`client.Reader`) to read CR status -- no direct API calls.

**JSON response shape** (initial):

```json
{
  "apiVersion": "agent.cloudzero.com/v1alpha1",
  "kind": "CloudZeroAgent",
  "metadata": {
    "name": "cloudzero-agent",
    "namespace": "cloudzero",
    "creationTimestamp": "2026-04-01T12:00:00Z"
  },
  "spec": {
    "tls": {
      "mode": "managed",
      "secretName": "cloudzero-agent-tls",
      "validityDuration": "8760h",
      "renewalThreshold": "720h",
      "algorithm": "ECDSA",
      "keySize": 256
    }
  },
  "status": {
    "conditions": [
      {
        "type": "CertificateValid",
        "status": "True",
        "reason": "CertificateCurrent",
        "message": "TLS certificate exists and is not near expiry",
        "lastTransitionTime": "2026-04-01T12:05:00Z",
        "observedGeneration": 1
      }
    ]
  },
  "computed": {
    "overallHealth": "Healthy",
    "certificateExpiry": "2027-04-01T12:05:00Z",
    "certificateRenewalAt": "2027-03-02T12:05:00Z",
    "conditionsSummary": {
      "total": 1,
      "true": 1,
      "false": 0,
      "unknown": 0
    }
  }
}
```

The `computed` field contains derived values that the UI can display without recalculating them.

**Files**:
- **New**: `operator/internal/statusapi/server.go` -- HTTP server implementing `manager.Runnable`.
- **New**: `operator/internal/statusapi/handler.go` -- Request handler that reads from the cache.
- **New**: `operator/internal/statusapi/types.go` -- Response types including `computed` fields.
- **New**: `operator/internal/statusapi/server_test.go` -- Unit tests using a fake client.
- **Modify**: `operator/cmd/main.go` -- Add `--status-bind-address` flag, register server with manager.

### 4.2 Phase 2: Embedded HTML Dashboard

**Goal**: Serve a minimal browser-accessible dashboard from the operator process using Go's `embed` package. No build tools -- plain HTML, CSS, and vanilla JavaScript.

**Dashboard layout**:

```
+------------------------------------------------------------------+
|  CloudZero Agent Status                    Last updated: 10s ago  |
+------------------------------------------------------------------+
|  Overall Health: [HEALTHY]                                        |
|                                                                    |
|  Conditions                                                        |
|  +------------------+--------+-----------------+----------------+ |
|  | Type             | Status | Reason          | Message        | |
|  | CertificateValid | True   | CertCurrent     | TLS cert is... | |
|  | StorageHealthy   | True   | PVCBound        | PVC bound...   | |
|  | BackfillComplete | False  | JobFailed       | backfill-job...| |
|  +------------------+--------+-----------------+----------------+ |
|                                                                    |
|  TLS Certificate                                                   |
|    Mode: managed          Algorithm: ECDSA P-256                  |
|    Expires: 2027-04-01  (356 days remaining)                      |
|    Renewal at: 2027-03-02                                         |
|    [=========================================------]  92%          |
|                                                                    |
|  Recent Events                                                     |
|    2026-04-10 09:15 - CertificateInstalled                        |
|    2026-04-09 09:15 - Reconciled: No action needed                |
+------------------------------------------------------------------+
```

**Files**:
- **New**: `operator/internal/statusapi/static/index.html`
- **New**: `operator/internal/statusapi/static/style.css`
- **New**: `operator/internal/statusapi/static/app.js`
- **New**: `operator/internal/statusapi/embed.go` -- `//go:embed` directive.
- **Modify**: `operator/internal/statusapi/server.go` -- Mount static file handler at `/`.

### 4.3 Phase 3: kubectl Plugin

**Goal**: Provide a `kubectl cz` plugin for terminal-based status inspection.

**Subcommands**:

| Command | Description |
|---------|-------------|
| `kubectl cz status` | Overview of all CloudZeroAgent resources with aggregated health |
| `kubectl cz status <name>` | Detailed status of a specific resource |
| `kubectl cz conditions` | Table of all conditions with transition times |
| `kubectl cz cert` | Certificate details including expiry countdown |
| `kubectl cz events` | Recent Kubernetes Events for the agent |
| `kubectl cz diagnose` | Run diagnostic checks and suggest remediation |

**Repository structure**:

```
operator/
  cmd/
    main.go                  # existing operator binary
    kubectl-cz/
      main.go                # plugin entry point
      cmd/
        root.go
        status.go
        conditions.go
        cert.go
        events.go
        diagnose.go
```

### 4.4 Phase 4: Enriched Data Collection

**Goal**: Enrich the status API with data beyond the CRD status.

- Read the TLS Secret to extract actual certificate details (expiry, SANs, issuer).
- List agent-related Pods and their status (requires additional RBAC: `pods` get/list).
- List Kubernetes Events filtered to the CloudZeroAgent resource.
- Read PVC status for storage health.
- Query the Metrics API for resource usage (if available).

**Files to modify**:
- `operator/internal/controller/cloudzeroagent_controller.go` -- Add RBAC markers for pods, events, PVCs.
- `operator/internal/statusapi/handler.go` -- Fetch and include enriched data.
- `operator/internal/statusapi/types.go` -- Extended response types.

---

## 5. Technology Choices

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Status API framework | `net/http` (stdlib) | Two endpoints; no framework needed. |
| JSON serialization | `encoding/json` (stdlib) | Standard, sufficient. |
| Embedded dashboard | Go `embed` + vanilla JS | Zero frontend build tooling. |
| kubectl plugin framework | `spf13/cobra` | Already a transitive dependency; standard for kubectl plugins. |
| Terminal rendering | `tablewriter` or `charmbracelet/lipgloss` | Tables for structured data, lipgloss for color-coded status. |
| Client library | `client-go` + controller-runtime `client.Reader` | Reuse existing client infrastructure. |

---

## 6. Phased Delivery

| Phase | Scope | Dependencies |
|-------|-------|--------------|
| **1** | JSON status API endpoint in operator | None -- can start immediately |
| **2** | Embedded HTML dashboard | Phase 1 |
| **3** | kubectl plugin (`kubectl cz`) | CRD types (existing) -- can run in parallel with Phase 1 |
| **4** | Enriched data (cert details, pods, events) | Phase 1 |
| **5** | Enhanced dashboard (charts, live updates) | Phase 2 + Phase 4 |

Phases 1 and 3 can proceed in parallel since the kubectl plugin reads directly from the Kubernetes API.

---

## 7. Security Considerations

- **The status API is cluster-internal by default.** Binds to a port on the operator pod; only accessible via `kubectl port-forward` or a ClusterIP Service.
- **Read-only.** GET requests only; no mutations accepted.
- **No secrets exposed.** Returns condition status, configuration metadata, and computed fields. TLS private keys, API keys, and Secret contents are never returned. Certificate details (subject, expiry, SANs) are public certificate fields.
- **kubectl plugin uses the caller's RBAC.** Requires only get/list/watch on `CloudZeroAgent` resources (the existing `cloudzeroagent-viewer-role`).
- **Phase 4 enrichment** adds read-only access to pods, events, and PVCs -- standard cluster-reader permissions.

---

## 8. Open Questions

1. **Should the status API require authentication?** Initial implementation relies on network-level isolation (ClusterIP). External exposure would require token-based auth or the Kubernetes API server auth proxy.
2. **Should the kubectl plugin be distributed via Krew?** Adds discoverability but requires maintaining a separate Krew manifest repo. Alternative: binary download from GitHub Releases.
3. **Should the embedded dashboard support multiple CloudZeroAgent resources?** Most clusters will have exactly one. Multi-resource support can be added later.
4. **Server-Sent Events for live dashboard updates?** Polling (10s interval) is simpler to start. SSE can be added in Phase 5 if real-time updates matter.
