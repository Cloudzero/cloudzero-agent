# Remote/Dynamic Configuration in Kubernetes Agents: Comparative Analysis

## 1. Datadog Agent — "Remote Configuration"

| Aspect | Details |
|--------|---------|
| **Feature name** | Remote Configuration |
| **Config fetch** | Poll-based over HTTPS (port 443). Agents periodically poll the Datadog SaaS backend. All communication is outbound — no inbound access required. |
| **Apply method** | Hot reload for most features (APM sampling rates, security rules). Initial enablement requires an agent restart. |
| **Source of truth** | Remote-first. Datadog UI is the source of truth; local config files only enable/disable Remote Configuration, they don't override individual settings. |
| **Failure handling** | Agent retries with backoff on polling failures. Automatic rollback to previous version if an agent upgrade fails (monitored over ~5 minutes). |
| **Audit** | Datadog Audit Trail tracks who requested config changes, API key events, and Remote Configuration events. |
| **Docs** | https://docs.datadoghq.com/remote_configuration/ |

**Takeaway**: Tightly coupled to their SaaS. Works because Datadog controls both sides. The audit trail integration is a strong pattern to emulate.

---

## 2. OpenTelemetry Collector — "OpAMP"

| Aspect | Details |
|--------|---------|
| **Feature name** | OpAMP (Open Agent Management Protocol) |
| **Config fetch** | Dual transport: WebSocket (server-push) or HTTP (client-poll, 30s default). Both use identical message formats. |
| **Apply method** | A separate **OpAMP Supervisor** binary receives config, merges with local config, writes `effective.yaml`, then **restarts the Collector process**. Not a hot reload — supervised restart. |
| **Source of truth** | Explicit merge priority order (lowest to highest): own-telemetry config → user local config → OpAMP extension config → **remote config wins**. The server detects divergence via config hash comparison. |
| **Failure handling** | If new config causes crash or failed health checks, Supervisor **automatically reverts to last known good config**. WebSocket reconnects automatically; HTTP uses exponential backoff. Agents report `RemoteConfigStatus` as APPLIED, APPLYING, or FAILED. |
| **Audit** | Agents report health, effective config, and package status continuously. Sequence numbers detect lost messages; `ReportFullState` triggers full re-sync. |
| **Docs** | https://opentelemetry.io/docs/specs/opamp/ |

**Takeaway**: The most complete and well-specified protocol. Automatic rollback on health failure, explicit merge priority, and the APPLIED/APPLYING/FAILED status reporting are the strongest patterns here.

---

## 3. Prometheus Operator — "Config Reloader Sidecar"

| Aspect | Details |
|--------|---------|
| **Feature name** | `prometheus-config-reloader` sidecar + CRD reconciliation |
| **Config fetch** | Kubernetes-native: Operator watches CRDs (ServiceMonitor, PodMonitor, PrometheusRule), reconciles them into ConfigMaps. Reloader sidecar watches mounted volumes for filesystem changes. No external API polling. |
| **Apply method** | Hot reload via HTTP POST to `/-/reload`. No pod restart needed. |
| **Source of truth** | Kubernetes CRDs are the sole source of truth. Operator is the sole config generator. |
| **Failure handling** | **Known weakness**: if invalid config is pushed, Prometheus rejects the reload. The reloader has a documented bug where it can get stuck and stop retrying even after config is corrected (issues #4708, #6145). No automatic rollback. |
| **Audit** | `prometheus_config_last_reload_successful` and `prometheus_config_last_reload_success_timestamp_seconds` metrics. |
| **Docs** | https://prometheus.io/docs/prometheus/latest/configuration/configuration/ |

**Takeaway**: Closest architectural match to what CloudZero would build. The stuck-reloader bug is an important cautionary tale — any reloader must have retry/recovery logic and a watchdog to detect stuck states.

---

## 4. Fluent Bit / Fluent Operator

| Aspect | Details |
|--------|---------|
| **Feature name** | Fluent Operator (CRD-based) + Fluent Bit Hot Reload / Fluent Bit Watcher |
| **Config fetch** | Operator watches CRDs (`ClusterInput`, `ClusterFilter`, `ClusterOutput`), constructs config, stores in a Secret mounted to the DaemonSet. |
| **Apply method** | Two mechanisms: (1) Native hot reload via HTTP `PUT /api/v2/reload` or `SIGHUP`. (2) `fluent-bit-watcher` — a wrapper binary that restarts the Fluent Bit process inside the same pod (not a pod restart). |
| **Source of truth** | CRDs are the single source of truth. |
| **Failure handling** | Poorly documented. No rollback mechanism. If new config is invalid, the process may fail to start. |
| **Audit** | Minimal — `hot_reload_count` from HTTP API only. |
| **Docs** | https://docs.fluentbit.io/manual/administration/hot-reload |

**Takeaway**: The two-tier approach (Operator manages CRDs, watcher handles process lifecycle) is pragmatic but the failure handling gap is a real risk. Avoid the custom image requirement the watcher introduces.

---

## Comparison Table

| Dimension | Datadog | OTel/OpAMP | Prometheus Operator | Fluent Bit/Operator |
|-----------|---------|------------|--------------------|--------------------|
| Transport | HTTPS poll | WebSocket (push) or HTTP (poll) | K8s API watch | K8s API watch |
| Apply method | Hot reload | Supervised process restart | Hot reload (HTTP POST) | Hot reload or process restart |
| Source of truth | SaaS UI (remote wins) | Configurable merge order | K8s CRDs only | K8s CRDs only |
| Config rollback | Yes (upgrades) | Yes (automatic on health failure) | No (stuck-reloader bugs) | No (undocumented) |
| Audit | Full audit trail | Status reporting + own telemetry | Prometheus metrics | `hot_reload_count` only |
| Maturity | Production (proprietary) | Beta/GA (open standard) | Production | Production |

---

## Recommendations for CloudZero Remote Config Design

**1. OpAMP-style automatic rollback** — if new remote config causes the agent to fail health checks, automatically revert to the last known good config. This is the most important safety mechanism.

**2. Explicit merge priority order** — define a clear, documented precedence:
```
operator defaults < CRD spec < remote API config (highest)
```
Make this deterministic and document it. Users who want local config to win should be able to set `spec.remoteConfig.localOverride: true`.

**3. CRDs remain the primary interface** — remote config enriches what's in the CRD spec; it doesn't replace it. The operator surfaces remote-config-derived values in `.status.remoteConfig` so they're observable and auditable without touching the spec.

**4. Status reporting (OpAMP pattern)** — report config status on the CRD:
```yaml
status:
  remoteConfig:
    status: Applied       # Applied | Applying | Failed | Disabled
    fetchedAt: "2026-04-17T10:00:00Z"
    configVersion: "abc123"
    message: "Remote config applied successfully"
```

**5. Avoid the Prometheus stuck-state bug** — implement retry with backoff on fetch/apply failures, plus a watchdog that forces a full re-sync if the agent has been in a failed state for too long.

**6. Audit trail** — emit a Kubernetes Event on every remote config fetch and apply, with the config version/hash. Expose `remote_config_last_applied_timestamp` and `remote_config_last_applied_successful` as status fields.
