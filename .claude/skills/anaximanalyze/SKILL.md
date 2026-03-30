---
name: anaximanalyze
description: >
  Analyzes Anaximander diagnostic output from a CloudZero Agent installation.
  Produces a customer-ready report identifying configuration issues, resource
  problems, and deployment health. Run when someone provides Anaximander output
  (a directory or .tar.gz archive) and asks for analysis, diagnosis, or
  troubleshooting of a CloudZero Agent deployment.
user-invocable: true
---

# Analyzing Anaximander Diagnostic Output

Anaximander (`scripts/anaximander.sh`) collects comprehensive diagnostic data
from a Kubernetes cluster running the CloudZero Agent. Customers run it when
something isn't working, and send the resulting archive to CloudZero for
analysis.

Your job is to read the output, identify what's wrong, and produce a clear,
actionable report that can be sent directly to the customer.

## Important: Output Sensitivity

The report you produce will be shared with the customer. Keep it clean:

- **DO NOT** include CloudZero-internal ticket numbers, Slack references, or
  internal tooling names
- **DO NOT** reference proprietary architecture details beyond what the customer
  can observe in their own cluster
- **DO NOT** speculate about CloudZero platform-side issues without evidence
- **DO** reference Kubernetes concepts, Helm values, and kubectl commands freely
- **DO** include specific values from their output (memory usage, pod counts,
  error messages) -- this is their own data

If you discover something that requires internal investigation (e.g., a
suspected platform-side bug), note it separately outside the customer report
for the CloudZero engineer running the analysis.

## Locating the Output

Anaximander output arrives in one of two forms:

1. **A `.tar.gz` archive** -- extract it first with `tar -xzf <file>`
2. **A directory** already extracted

The output may be flat (all files in one directory) or nested (files under a
subdirectory named after the namespace or a timestamped label). Both are valid.
Look for `metadata.txt` as your anchor -- it's always present and tells you
where the collection happened.

## Analysis Approach

The analysis happens in two phases. **Phase 1 comes first and is the
primary analysis.** Look at the data with fresh eyes, understand the
deployment, and identify everything that looks wrong or unusual. **Phase 2**
then cross-references your findings against the Debugging Guide to catch
any known patterns you might have missed and to inform your recommendations.

Do NOT read the Debugging Guide before Phase 1. The guide is valuable for
validation and completeness, but reading it first biases you toward confirming
known patterns rather than actually seeing what's in front of you.

### Test Your Hypotheses

This is critical: **if a hypothesis can be verified from the data you have,
verify it before reporting it.** Do not speculate about causes when the
evidence to confirm or rule them out is sitting in the diagnostic output.

For example:

- If you suspect a certificate has expired, check `tls-cert-metadata.txt`
  or `webhook-config.txt` for the actual `notBefore`/`notAfter` dates. The
  agent's self-signed certificates default to 100-year validity, so don't
  assume a short expiration without checking.
- If you suspect a component is OOMKilled, check `describe-all.txt` for
  `Last State: Terminated, Reason: OOMKilled` rather than inferring it from
  high memory usage alone.
- If you suspect network connectivity issues, check whether the shipper's
  next cycle succeeded rather than assuming a single error is persistent.

When you write a finding, state what the data shows. If you've confirmed
something, say "the certificate expires on [date]" not "the certificate may
have expired." If the data to verify a hypothesis genuinely isn't available
in the Anaximander output, say so explicitly and provide the exact command
the customer can run to check -- but this should be the exception, not the
norm. Most things can be verified from the files you have.

---

## Phase 1: Data-First Exploration

Read all the files. Understand the deployment. Find the problems.

### Step 1: Orientation (metadata.txt, command-results.txt)

Read `metadata.txt` first to understand the basics:

- **Kubernetes context** -- identifies the cluster
- **Namespace** -- where the agent is installed
- **Collection time** -- how recent the data is

Then read `command-results.txt` to see whether the collection itself had
problems. Any `[FAILED]` entries mean you're working with incomplete data --
note this. Common benign failures:

- `helm list` fails when the customer uses ArgoCD/FluxCD (Helm secrets are in
  a different namespace) -- this is normal
- `kubectl top` fails when metrics-server isn't installed
- cAdvisor metrics fail on some managed Kubernetes platforms

### Step 2: Cluster Environment (nodes.txt)

Read `nodes.txt` to understand the infrastructure:

- **Node count** -- affects expected resource usage
- **Kubernetes version**
- **Container runtime** -- `containerd://` and `cri-o://` are fine. `docker://`
  on Kubernetes 1.24+ causes cAdvisor label problems (empty `container`,
  `image`, `name` labels on metrics).
- **Node roles** -- control plane vs worker distribution
- **Node age distribution** -- many very new nodes suggest active autoscaling.
  Note the churn pattern -- this affects memory usage and admission webhook
  volume.
- **Cloud provider** -- inferred from node names (`ip-` = AWS, `aks-` = Azure,
  `gke-` = GCP)
- **OS and architecture** -- usually Linux/amd64 but worth confirming

### Step 3: Deployment Health (get-all.txt, describe-all.txt)

Read `get-all.txt` for the quick overview, then `describe-all.txt` for the
details.

**In get-all.txt, look at:**

- **Pod status** -- are all pods `Running` with all containers ready? Flag
  anything other than healthy.
- **Restart counts** -- occasional restarts happen; dozens or hundreds of
  restarts signal a persistent problem.
- **Deployment readiness** -- do desired/current/ready counts match?
- **Job completion** -- any failed jobs?
- **Naming patterns** -- two sets of pods for the same component (e.g., both
  `aggregator-*` and `cloudzero-cz-aggregator-*`) indicates a stale deployment.
- **Age patterns** -- are all components the same age? Very new pods alongside
  older ones might mean a recent rollout or restart.

**In describe-all.txt, look at:**

- **Container states** -- `Last State: Terminated, Reason: OOMKilled,
Exit Code: 137` means the container was killed for exceeding its memory limit.
- **Resource requests and limits** -- note what's configured.
- **Image versions** -- are all components at the same agent version?
- **Volume mounts** -- are expected volumes mounted?
- **Events at the bottom of each pod** -- scheduling failures, image pull
  errors, probe failures, mount issues.
- **Annotations** -- sidecar injection, Gatekeeper mutations, restartedAt, etc.

### Step 4: Resource Usage (top-pods.txt)

Compare actual usage against limits from describe-all.txt.

**Server memory is the most common problem area.** The recommended operating
range is up to ~80% of the configured limit. The standard sizing approach is
1.2x the observed steady-state usage, which gives roughly 20% headroom.

Use these thresholds when deciding what to report:

- **Below 80%:** Within the recommended operating range. Do not flag this or
  suggest increasing memory. This is normal and healthy.
- **80-90%:** Worth mentioning. The server is within recommendations but
  getting close to the edge. Recommend the customer keep an eye on it and
  consider increasing the limit if usage trends upward.
- **Above 90%:** Recommend increasing the limit now, before it OOMKills.
  This is too close to the edge -- a GC spike or metric ingestion burst
  could push it over.
- **OOMKilled (exit code 137 in describe-all.txt):** The limit must
  increase. This is the single most common agent problem.

For sizing recommendations, the best approach is: set the limit generously
(well above current usage), let the server run for a day under normal load,
check actual usage with `kubectl top`, then set the limit to 1.2x that
observed value. This accounts for real workload patterns better than any
formula. For a rough initial estimate, use: `512Mi + (node_count / 100) *
768Mi`. High-churn environments (lots of short-lived pods, frequent scaling)
use more than this formula suggests.

When recommending an increase, always suggest setting both requests and
limits under `components.agent.resources`.

Typical ranges for other components:

- **Aggregator:** 100-300Mi per replica. Over 500Mi = heavy metric volume.
- **Webhook:** 50-150Mi. Over 300Mi is unusual.
- **KSM:** Scales with Kubernetes object count.

### Step 5: Events (events.txt)

Read for OOMKilled, FailedScheduling, FailedMount, Unhealthy (probe failures),
BackOff (crash loops), FailedCreate, and any ExternalSecrets events.

If empty (`No resources found`), note it -- events age out after ~1 hour by
default, so this limits what you can conclude about past problems.

### Step 6: Configuration (helmless logs, webhook/shipper startup logs)

Read the helmless job log (`*-helmless-*-logs.txt`) -- this shows the
effective configuration. Note the configured values for region, account ID,
cluster name, label patterns, resource limits, and any custom settings.

Read the first few lines of webhook and shipper logs for startup warnings:

- **Region mismatch** warnings
- **Account ID mismatch** warnings
- **Cloud provider detection** messages

If the helmless job pods have been cleaned up (common -- they're short-lived
jobs), note that you can't verify what configuration was applied.

Read `helm-list.txt` -- is it populated or empty? Empty means GitOps
deployment.

### Step 7: All Logs -- Read Everything

Read all pod log files. Don't limit yourself to known patterns -- read for
anything that looks wrong.

**What healthy looks like:**

- Server: config reloads, WAL replay, periodic scraping. Deprecation warnings
  from the Kubernetes API are informational.
- Aggregator collector: periodic flushes with row counts.
- Aggregator shipper: `"Successfully ran the shipper cycle"` every 10 minutes.
- Webhook: periodic push operations with record counts > 0 (if admission
  webhook is configured), startup messages, TLS certificate initialization.
- KSM: startup messages, periodic metric exports. Watch stream decode errors
  are usually benign (client reconnects).
- Validator: check results with pass/fail. All passing is good.
- Backfill: resource discovery, writes to storage, flush with record count > 0.

**What to flag:**

- Any `error` or `fatal` level log entries
- `"context canceled"` errors (timeouts)
- `"connection reset by peer"` or `"use of closed network connection"`
- TLS handshake errors
- S3 upload failures
- Storage write failures
- Unexpected EOF
- Patterns that repeat -- how often? Across how many pods?
- Zero record counts where you'd expect non-zero (e.g., webhook pusher
  always showing count:0 means the webhook isn't receiving admission requests).
  This is a critical pattern: if all webhook replicas show `"Found records to
send", count: 0` on every flush cycle for the entire log window, the
  `ValidatingWebhookConfiguration` is likely missing or has an empty/invalid
  `caBundle`. The API server silently skips the webhook because `failurePolicy`
  is `Ignore`. See the Debugging Guide section "Webhook Not Receiving Admission
  Requests" for the full diagnostic procedure. The fix is enabling cert-manager
  (`insightsController.tls.useCertManager: true`) or manually restoring the
  `caBundle` from the webhook TLS secret. Recommend upgrading to the latest
  chart version, which hardened the webhook's TLS configuration against service
  mesh interference.

**Context matters:** A single transient error that self-resolved is different
from a pattern of repeated failures. Note the frequency and whether errors
correlate across components or with specific timestamps.

### Step 8: Infrastructure (secrets, network, service mesh, cAdvisor)

Read `get-secrets.txt`, `secret-sizes.txt`, `network-policies.yaml`,
`service-mesh-detection.txt`, and `cadvisor-metrics.txt`.

- Do expected secrets exist (API key, webhook TLS, image pull)?
- Any network policies that could restrict traffic?
- Any service mesh detected? If so, are sidecars injected into agent pods?
- In cAdvisor metrics, are container labels (`container`, `image`, `name`)
  populated on per-container metrics?

### Step 8b: Webhook Configuration (webhook-config.txt)

If present, read `webhook-config.txt`. This file contains the
`ValidatingWebhookConfiguration` that registers the CloudZero webhook
with the Kubernetes API server, along with caBundle validation results.

- Does a VWC exist referencing the agent's namespace?
- Is the `caBundle` present and valid (typically 1000-2000 characters)?
- If the caBundle is empty, the API server silently skips the webhook
  (`failurePolicy: Ignore`) and the webhook will show `count: 0` on
  every flush. See "Webhook Not Receiving Admission Requests" in the
  Debugging Guide.
- Check certificate dates -- has it expired? The `webhook-config.txt` file
  includes decoded certificate details (notBefore, notAfter, fingerprint).
  Also check `tls-cert-metadata.txt` for the TLS secret's certificate
  details. The agent's self-signed certificates default to 100-year
  validity, so do not assume a short expiration -- read the actual dates
  from the output. If neither file is present (older Anaximander version),
  note that the certificate dates could not be verified.

Note: Older versions of anaximander may not include these files. If they're
absent, check for the `count: 0` pattern in webhook logs as an indirect
indicator of VWC/caBundle problems.

### Step 9: Scrape Configuration (scrape-config-info.txt, config files)

- Prometheus or Alloy mode?
- Expected scrape targets present?
- Remote write pointing to aggregator?
- Scrape interval reasonable (15s-60s)?
- Any custom additions?

### Step 10: Synthesize

Before moving to Phase 2, write down your findings so far. What problems
did you find? What looks unusual? What questions do you have? This captures
your unbiased read of the data.

---

## Phase 2: Debugging Guide Cross-Reference

Now fetch the Debugging Guide to validate and enrich your Phase 1 findings.

Fetch the latest version from the wiki:

```text
https://raw.githubusercontent.com/wiki/Cloudzero/cloudzero-agent/Debugging-Guide.md
```

Use WebFetch to retrieve it. If the fetch fails, fall back to the local copy
at `docs/wiki/Debugging-Guide.md` in the agent repository.

The guide is large. Focus on:

- **Appendix E: Common Error Patterns** -- cross-reference any error messages
  you found in Phase 1
- **Configuration Diagnostics** -- if you found config mismatches
- **High Memory Usage** (especially section D about KSM connection resets) --
  if memory was elevated
- **Webhook Not Receiving Admission Requests** -- if webhook flush counts are
  always zero
- **Webhook Storage Write Failures** -- if you found storage errors
- **Cannot Reach S3 Buckets** (especially cause E about transient connections)
  -- if you found shipper errors
- **Deployment Integrity Diagnostics** -- if you found stale deployments
- **Appendix C: Anaximander** -- recommended analysis order and what to look
  for in each file

**What to do with the guide:**

1. Check if any of your Phase 1 findings match known patterns. If so, use the
   guide's resolution steps to inform your recommendations.
2. Check if the guide's checklist surfaces any issues you missed. Walk through
   the known-problem patterns and verify each one against the data.
3. For novel issues (things you found that aren't in the guide), your Phase 1
   analysis stands on its own. Note these for the engineer as potential
   additions to the guide.

---

## Phase 3: Version and Release Notes

After Phase 2, check whether the customer is running the latest chart version.
If they are, skip this phase entirely -- there is nothing to add.

### Step 1: Look up the latest release

Fetch the latest release from the GitHub API:

```text
https://api.github.com/repos/Cloudzero/cloudzero-agent/releases/latest
```

Parse `tag_name` (strip the `v` prefix to get the version number) and
`published_at` for the release date. No authentication is required -- the
repository is public. This works with WebFetch; no `gh` CLI needed.

### Step 2: Compare against the customer's version

The customer's chart version is already known from Phase 1 (image tags in
`describe-all.txt`, or `helm-list.txt`). If they match the latest, stop
here -- there is nothing to report.

### Step 3: Fetch release notes for intervening versions

Each version's release notes are published at:

```text
https://raw.githubusercontent.com/Cloudzero/cloudzero-agent/main/helm/docs/releases/<version>.md
```

For example, a customer on 1.2.8 with latest at 1.2.10 needs `1.2.9.md` and
`1.2.10.md`. To get the list of versions between theirs and latest, fetch
the full releases list:

```text
https://api.github.com/repos/Cloudzero/cloudzero-agent/releases
```

Filter to non-prerelease entries, sort by semantic version, and take the
range between the customer's version (exclusive) and latest (inclusive).
Fetch the release notes for each.

### Step 4: Scan for relevance to findings

Compare the release notes against the Phase 1 and Phase 2 findings.
Categorize each match:

- **Direct fix**: A release note explicitly describes fixing the exact
  observed issue. Strong upgrade recommendation.
- **Related improvement**: Changes in the same area that could help (e.g.,
  Istio handling improvements when the customer has Istio). Moderate
  recommendation.
- **General currency**: No specific match. Low-priority standing
  recommendation.

### Step 5: Add to report

Add a finding to the report, **always sorted last** (lowest priority among
findings). The default text when no specific fixes are relevant:

> Version X.Y.Z was released on \<date\>. We recommend upgrading to the
> latest version to benefit from ongoing improvements, bug fixes, and
> security updates.

When specific release notes are relevant to observed issues:

- The version finding should list the specific entries that are relevant
  and explain why they matter for this deployment
- The primary finding that the fix addresses (e.g., "Webhook Not Receiving
  Admission Requests") should cross-reference the version finding: "Upgrading
  to the latest chart version is strongly recommended -- see the Chart Version
  finding below for details."
- If the match is strong (direct fix for an active problem), the version
  finding's priority may be elevated accordingly

---

## Report Format

Output the report as markdown. Keep it conversational but precise -- the goal
is to be helpful, not to sound like a wiki article.

```markdown
# CloudZero Agent Diagnostic Report

|                   |                                                      |
| ----------------- | ---------------------------------------------------- |
| **Cluster**       | <context from metadata.txt>                          |
| **Namespace**     | <namespace>                                          |
| **Chart Version** | <version if available>                               |
| **Agent Version** | <from image tags>                                    |
| **Collected**     | <timestamp from metadata.txt>                        |
| **Cluster Size**  | <node count> nodes (<cloud provider>, <k8s version>) |

## Summary

<2-3 sentences describing the overall health of the deployment and the primary
issue(s) found. Be direct -- "The server is running out of memory" not "We
observed elevated memory utilization patterns.">

## Findings

### <Finding 1: Root Cause Title>

<Symptoms, explanation, recommendation with actionable fix.>

### <Finding 2: Root Cause Title>

<Same pattern.>

## Environment Notes

<Anything notable about the environment that isn't a problem but is worth
documenting -- container runtime, service mesh presence, GitOps deployment
method, network policies, image pull secrets, collection completeness, etc.>
```

### Findings: Structure and Grouping

Findings are organized by **root cause**, not by symptom. If multiple symptoms
point to the same underlying problem, group them into a single finding.

Each finding should include:

- **What we observed** -- specific symptoms with numbers from the output
- **What it means** -- the root cause explanation
- **Recommendation** -- a concrete action. This can be a configuration change
  (with a Helm values example), a further investigation task, or an explicit
  "no action needed" if the issue is transient and self-resolved. Include
  specific Helm values snippets when the fix is a configuration change.

**Grouping rules:**

- If symptom B's recommendation is "fix A and this goes away," merge B into A
- If multiple components log the same warning (e.g., region mismatch in
  webhook, shipper, and collector), that's one finding about the root config
  issue, not three findings
- KSM connection resets caused by server memory pressure belong under the
  server memory finding, not as a separate item
- Transient errors that self-resolved (single S3 upload failure, isolated TLS
  timeout) can be mentioned as supporting context in a related finding or
  noted in the environment notes -- they don't need their own finding unless
  they point to a distinct, unresolved problem

**Order findings by priority** -- most impactful issue first.

### Helm Values: Use the Correct Paths

When recommending configuration changes, **always use the `components.*`
paths**. The chart has deprecated top-level paths (`server.*`,
`insightsController.*`, `aggregator.*`, `validator.*`) that still work but
are not API-stable and should not be recommended to customers.

**Resource limits and requests:**

```yaml
# Agent/server (Prometheus)
components:
  agent:
    resources:
      requests:
        memory: "512Mi" # default
        cpu: "250m"
      limits:
        memory: "1024Mi" # default
        cpu: "1000m"

  # Aggregator (collector and shipper are separate containers)
  aggregator:
    collector:
      resources:
        requests:
          memory: "64Mi"
        limits:
          memory: "1024Mi"
    shipper:
      resources:
        requests:
          memory: "64Mi"
        limits:
          memory: "1024Mi"

  # Webhook
  webhookServer:
    resources:
      requests:
        memory: "128Mi"
      limits:
        memory: "512Mi"
```

**KSM** is a subchart, so it's under `kubeStateMetrics`, not `components`:

```yaml
kubeStateMetrics:
  resources:
    requests:
      memory: "256Mi"
    limits:
      memory: "512Mi"
```

**Replicas:**

```yaml
defaults:
  replicas: 3 # shared default for all components
components:
  aggregator:
    replicas: null # falls back to defaults.replicas
  webhookServer:
    replicas: null # falls back to defaults.replicas
```

**Cloud identity** (top-level, not under components):

```yaml
cloudAccountId: ""
clusterName: ""
region: null
```

**Deprecated paths -- do NOT recommend these:**

| Deprecated                         | Use instead                                   |
| ---------------------------------- | --------------------------------------------- |
| `server.resources.*`               | `components.agent.resources.*`                |
| `insightsController.resources.*`   | `components.webhookServer.resources.*`        |
| `aggregator.collector.resources.*` | `components.aggregator.collector.resources.*` |
| `aggregator.shipper.resources.*`   | `components.aggregator.shipper.resources.*`   |
| `validator.resources.*`            | `components.validator.resources.*`            |
| `imagePullSecrets`                 | `defaults.image.pullSecrets`                  |
| `commonMetaLabels`                 | `defaults.labels`                             |

The one exception where you may need to reference paths outside of
`defaults`, `components`, and `integrations` is `insightsController.tls.*`
(e.g., `insightsController.tls.useCertManager: true`). These don't have
`components` equivalents yet.

### Scout (Auto-Detection) Guidance

The agent's Scout can automatically detect region and account ID from the
cluster environment. When the Scout's detected values differ from configured
values, every webhook and shipper pod logs warnings on startup. If the Scout
is detecting correct values, recommend either fixing the hard-coded values or
removing them entirely and letting the Scout auto-detect. The Scout has no
silent failure mode -- if it can't detect, the agent enters CrashLoopBackOff.

### Tone and Style

Keep the tone direct and helpful. Use specific numbers from the output. Tell
them what to do, not just what's wrong.

If everything looks healthy, say so -- a clean bill of health is useful
information too. If there are multiple issues, rank them by impact and
address the most critical first.
