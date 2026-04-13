# Operator Roadmap: Issue Patterns from the Burn Book

## Executive Summary

After reviewing hundreds of customer records in the burn book, 10 distinct recurring issue patterns were identified that have required manual intervention, caused data gaps, or left agents silently broken. Ordered by priority based on frequency and customer impact.

---

## Pattern 1: KSM and Agent Server OOMKill / Memory Pressure

**Priority: HIGH**

**Customers affected:** Booking.com, Braze, DoorDash, American Airlines, Universal Parks & Resorts, Alation, HedgeServ, Akuna Capital (8+)

**Examples:**
- Booking.com: KSM OOMKilled 4,840 times in 16 hours; default 256Mi completely insufficient for large clusters. Required 3Gi.
- Braze: US01 cluster (~953 nodes) hit 14.9Gi with 16Gi limit; required 24Gi after iterative guessing.
- DoorDash: Agent server OOMing at 25Gi on 2,500-node clusters; bumped to 48Gi.
- Akuna Capital: Pod failing for 8 days; UI showed "connected" despite CrashLoopBackOff.

**Operator feature:**
- Watch pod metrics for `cloudzero-agent-server`, `cloudzero-state-metrics`, aggregator pods via Metrics API
- `MemoryPressure` condition with component-level detail (which pod, current %, limit, trend)
- Observe: event at 80% with sizing recommendation (formula: 512Mi + 0.75Gi per 100 nodes)
- AutoRemediate: patch Deployment limits within configured bounds, trigger rolling restart
- Suppress during backfill Job (expected KSM spike) — already in Phase 2 plan

---

## Pattern 2: Webhook TLS Cert Expiry and VWC Drift

**Priority: HIGH**

**Customers affected:** Booking.com, DraftKings, DoorDash, DVAG (5+)

**Examples:**
- Booking.com: Self-signed cert expired after 126 days. `failurePolicy: Ignore` caused API server to silently discard all admission requests. Webhook pods looked healthy but received zero traffic. Multi-week investigation.
- Booking.com (separate incident): ArgoCD pruned the VWC because it wasn't tracked in ArgoCD manifest — the init-cert Job created it as a live resource and ArgoCD deleted it on next sync.

**Operator feature (enhancement to existing Phase 1):**
- Watch the VWC itself — recreate it if it disappears (ArgoCD prune case)
- `WebhookReachable` condition: periodically verify the caBundle in VWC matches the current TLS secret; patch if drifted
- Active health check: send synthetic admission review to webhook endpoint; set `WebhookHealthy=False` if it fails (catches `failurePolicy: Ignore` masking)
- Emit Warning event 30 days before expiry as a safety net regardless of auto-rotation

---

## Pattern 3: Webhook Unreachable (Network/CNI Issues)

**Priority: HIGH**

**Customers affected:** Kilo Health, Syncron, IAS, DraftKings, PicPay (5+)

**Examples:**
- Kilo Health: Network policies blocked API server → cloudzero namespace ingress. 15s timeout per request effectively killed the API server.
- Syncron: Calico using non-RFC1918 IP pool (194.168.0.0/16). Kubernetes API server silently rejects webhook calls to non-private IPs. Required `hostNetwork: true`.
- IAS: Only backfill job collecting labels; webhook never invoked due to Istio/network policies.

**Operator feature:**
- Pre-flight checks on first reconcile: detect non-RFC1918 pod IPs, NetworkPolicies blocking API server ingress, DNS resolution of webhook service
- `WebhookReceivingTraffic` condition: if zero admission requests received within 10 minutes of VWC creation, flag with likely cause
- Event with specific remediation guidance (e.g., "Webhook pods have non-RFC1918 IPs from Calico CNI; consider setting hostNetwork: true")

---

## Pattern 4: Silent Agent Failures / False "Connected" Status

**Priority: HIGH**

**Customers affected:** Akuna Capital, Infoblox, Booking.com, Cox Automotive, DVAG, Recruitics, Platform Science (7+)

**Examples:**
- Akuna Capital: Pod failing for 8 days; UI showed "connected" despite CrashLoopBackOff — sidecar running but main container dead.
- Infoblox: 67 clusters appeared disconnected; getting label data but nothing from Prometheus.
- Booking.com: 39 of 54 clusters missing webhook server metrics; green-state clusters not providing data.

**Operator feature:**
- `AgentHealthy` condition with per-component breakdown: `ServerRunning`, `ShipperRunning`, `KSMRunning`, `WebhookServerRunning`
- Check ALL containers in a pod are Ready (not just pod phase) — catches sidecar-running-but-main-dead
- `DataFlowStale` condition: if shipper hasn't successfully uploaded in configurable window (default 30m based on upload cycle), flag it even if pods look healthy
- Events for state transitions (Healthy → Degraded, Degraded → Available) to provide audit trail

---

## Pattern 5: PVC / Volume Mount Failures

**Priority: MEDIUM**

**Customers affected:** IAS, Braze, Wise, Priceline (4+)

**Examples:**
- IAS: Multi-Attach error for PVC — multiple pods trying to attach same EBS volume (ReadWriteOnce). Recurring across versions 1.1.2 and 1.2.2.
- Wise: Helm upgrade fails when PVC enabled; must delete prior PV claim before upgrading.
- Priceline: Aggregator pods requiring periodic deployment rolls to reset; emptyDir confusion.

**Operator feature:**
- `StorageHealthy` condition; watch for `FailedAttachVolume`, `FailedMount`, `Multi-Attach` events on agent pods
- Pre-upgrade validation: if `server.persistentVolume.enabled: true` with ReadWriteOnce, warn that rolling updates will cause Multi-Attach conflicts
- AutoRemediate: delete old pod preventing volume attachment on Multi-Attach errors (after verifying ownership)
- Track emptyDir usage; warn before kubelet eviction

---

## Pattern 6: Configuration Errors

**Priority: MEDIUM**

**Customers affected:** Booking.com, BMC, Infoblox, Acquia, DoorDash (6+)

**Examples:**
- Infoblox: Multiple clusters with incorrect Cloud Account IDs (e.g., "gcp-eng-eu-ddiaas-prod", "test-account-id") that didn't match actual GCP project IDs.
- Acquia: Extra quotes in `cloud_account_id` caused S3 upload URL mismatches.
- BMC: IMDS metadata service blocked causing auto-detection timeouts; no clear error surfaced.
- DoorDash: API key missing container-metrics permissions; failed silently.

**Operator feature:**
- `ConfigurationValid` condition with reason codes: `InvalidAPIKey`, `InvalidCloudAccountID`, `RegionMismatch`, `IMDSUnavailable`
- Validate on reconcile: API key secret exists and non-empty; cloud account ID format (numeric for AWS, project ID for GCP); region is valid or empty
- Lightweight API call to verify API key has required permissions
- IMDS reachability check; if unreachable and auto-detection is configured, surface immediately with remediation guidance

---

## Pattern 7: Helm Upgrade / GitOps Deployment Failures

**Priority: MEDIUM**

**Customers affected:** Booking.com, Wise, Priceline, PicPay, DraftKings, Shutterstock (7+)

**Examples:**
- Booking.com: ArgoCD pruned VWC (see Pattern 2); also env-validator CrashLoopBackOff from stale ConfigMap used across versions.
- DraftKings: PDB blocking upgrade to 1.2.9 — cluster admins enforce policy preventing PDBs for 1-replica Deployments.
- Shutterstock: PDB preventing Karpenter node consolidation.

**Operator feature:**
- Pre-upgrade checks: PVC compatibility, PDB cluster policy compatibility, ConfigMap schema compatibility
- `UpgradeReady` condition gating upgrades on pre-flight pass
- GitOps awareness: detect ArgoCD/Flux annotations; warn about resources created by Jobs but not tracked in GitOps manifests (specifically the VWC)
- If PDB creation fails due to cluster policy, auto-configure without PDBs and set an explanatory condition

---

## Pattern 8: Job Failures (init-cert, env-validator, backfill, helmless)

**Priority: MEDIUM**

**Customers affected:** Booking.com, DVAG, Invesco, Shutterstock (4+)

**Examples:**
- DVAG: init-cert Job pulling from public bitnami registry, blocked by security policy — lacked `imagePullSecrets`.
- Booking.com: env-validator CrashLoopBackOff because ConfigMap from March 18 not updated alongside image bump to 1.2.10.

**Operator feature (enhancement to planned Phase 1):**
- Watch all agent Jobs; track completion status, failure count, error messages in conditions
- Inspect pod logs/events on failure; surface specific error (e.g., "init-cert failed: image pull failed for bitnami/kubectl, imagePullSecrets not configured")
- If CR specifies `imagePullSecrets`, propagate to ALL Jobs and subcharts

---

## Pattern 9: Network Egress / Firewall Blocking Uploads

**Priority: MEDIUM**

**Customers affected:** ChargePoint, SandboxAQ, Twilio, BMC, Rapid7 (5+)

**Examples:**
- ChargePoint: TLS handshake timeout to CloudZero API due to firewall restrictions.
- SandboxAQ: Shipper failing to upload metrics to S3 with "context deadline exceeded". Cluster disconnected.
- Twilio: Shipper failures uploading parquet files on 800+ node clusters; disk pressure accumulating when shipper couldn't send.

**Operator feature:**
- `EgressConnectivity` condition: periodically verify CloudZero API endpoint reachable, S3 presigned URL upload works
- Reason codes: `APIUnreachable`, `S3UploadFailed`, `TLSHandshakeTimeout`, `IMDSUnavailable`
- If shipper uploads fail AND ephemeral storage is growing, emit critical Event warning of imminent node eviction

---

## Pattern 10: Label Collection / Webhook Config Bugs

**Priority: LOW** (mostly resolved by code fixes, but operator can prevent recurrence)

**Customers affected:** Twilio, Upstart, Shutterstock, Syncron, Booking.com, IAS (6+)

**Examples:**
- Twilio/Upstart: v1.2.4 bug — VWC used singular resource names ("pod") instead of plural ("pods"), silently missing most resources.
- Booking.com: 300+ label limit exceeded, requiring regex filtering.

**Operator feature:**
- `LabelFlowHealthy` condition: verify webhook server has received at least N admission requests in recent window; flag if zero
- Audit VWC resource types on reconcile (plural names, expected rule set); emit Warning if `failurePolicy: Ignore` is set
- Monitor backfill CronJob schedule; if hasn't run in expected interval (default 3h), set condition

---

## Summary Table

| # | Pattern | Priority | Customers | Already Planned? |
|---|---------|----------|-----------|-----------------|
| 1 | OOMKill / Memory Pressure | HIGH | 8+ | Phase 2 |
| 2 | Webhook TLS Cert Expiry + VWC Drift | HIGH | 5+ | Phase 1 done — needs VWC watch enhancement |
| 3 | Webhook Unreachable (Network/CNI) | HIGH | 5+ | Not planned |
| 4 | Silent Agent Failures / False "Connected" | HIGH | 7+ | Partially (Phase 1 conditions) |
| 5 | PVC / Volume Mount Failures | MEDIUM | 4+ | Partially (Phase 1 storage health) |
| 6 | Configuration Errors | MEDIUM | 6+ | Not planned |
| 7 | Helm Upgrade / GitOps Failures | MEDIUM | 7+ | Not planned |
| 8 | Job Failures | MEDIUM | 4+ | Phase 1 — needs diagnostics enhancement |
| 9 | Network Egress / Upload Failures | MEDIUM | 5+ | Not planned |
| 10 | Label Collection / Webhook Config Bugs | LOW | 6+ | Not planned |
