{{/*
================================================================================
                    GRAFANA ALLOY RIVER CONFIGURATION HELPERS
================================================================================

This template file generates Grafana Alloy River configuration for the CloudZero
Agent. Alloy replaces Prometheus as the metrics collector when the agent is
deployed in "clustered" mode (components.agent.mode: "clustered").

================================================================================
                          ARCHITECTURE OVERVIEW
================================================================================

Alloy uses a pipeline-based architecture where data flows through connected
components. The CloudZero Agent configuration creates this pipeline:

    +-------------------+     +-------------------+     +-------------------+
    |     DISCOVERY     | --> |      SCRAPE       | --> |     RELABEL       |
    | (Find targets)    |     | (Collect metrics) |     | (Filter/transform)|
    +-------------------+     +-------------------+     +-------------------+
                                                                |
                                                                v
                                                    +-------------------+
                                                    |   REMOTE WRITE    |
                                                    | (Send to agg)     |
                                                    +-------------------+

Each scrape job follows this pattern, with variations based on the source.

================================================================================
                            COMPONENT TYPES
================================================================================

1. discovery.kubernetes  - Discovers Kubernetes resources (nodes, services, etc.)
2. discovery.relabel     - Transforms target labels BEFORE scraping
                          (This is where __meta_* labels are available!)
3. prometheus.scrape     - Scrapes metrics from discovered targets
4. prometheus.relabel    - Transforms metrics/labels AFTER scraping
                          (__meta_* labels are NOT available here!)
5. prometheus.remote_write - Sends metrics to a remote endpoint

IMPORTANT: The distinction between discovery.relabel and prometheus.relabel is
critical. Target metadata labels (__meta_*) are only available in discovery.relabel,
not in prometheus.relabel. If you need to use __meta_* labels, you must do so
in a discovery.relabel component BEFORE the prometheus.scrape component.

================================================================================
                         DATA FLOW PER SCRAPE JOB
================================================================================

KUBE-STATE-METRICS (KSM):
  Static target --> prometheus.scrape --> prometheus.relabel (metrics filter)
                                      --> prometheus.relabel (labels filter)
                                      --> prometheus.remote_write

CADVISOR:
  discovery.kubernetes (nodes) --> discovery.relabel (add node label)
                               --> prometheus.scrape (TLS + auth)
                               --> prometheus.relabel (metrics filter)
                               --> prometheus.relabel (labels filter)
                               --> prometheus.remote_write

WEBHOOK:
  discovery.kubernetes (endpoints) --> prometheus.scrape (HTTPS)
                                   --> prometheus.relabel (metrics filter)
                                   --> prometheus.remote_write

AGGREGATOR:
  Static target --> prometheus.scrape --> prometheus.relabel (metrics filter)
                                      --> prometheus.remote_write

ALLOY SELF:
  discovery.relabel (localhost) --> prometheus.scrape --> prometheus.relabel
                                                      --> prometheus.remote_write

DCGM (GPU):
  discovery.kubernetes (services) --> prometheus.scrape
                                  --> prometheus.relabel (provenance)
                                  --> prometheus.relabel (metrics filter)
                                  --> prometheus.relabel (attribution filter)
                                  --> prometheus.remote_write

================================================================================
                     PROMETHEUS VS ALLOY TERMINOLOGY
================================================================================

Prometheus YAML                    Alloy River
----------------                   -----------
scrape_configs:                    prometheus.scrape "name" { ... }
relabel_configs:                   discovery.relabel "name" { ... }
metric_relabel_configs:            prometheus.relabel "name" { ... }
kubernetes_sd_configs:             discovery.kubernetes "name" { ... }
static_configs:                    targets = [{ __address__ = "..." }]
remote_write:                      prometheus.remote_write "name" { ... }

================================================================================
                            CONFIGURATION REFERENCE
================================================================================

For more information on River syntax and Alloy components, see:
- River Language: https://grafana.com/docs/alloy/latest/reference/config-language/
- Components:     https://grafana.com/docs/alloy/latest/reference/components/
- prometheus.*:   https://grafana.com/docs/alloy/latest/reference/components/prometheus/
- discovery.*:    https://grafana.com/docs/alloy/latest/reference/components/discovery/

*/}}


{{/* =========================================================================
                         MAIN CONFIGURATION ENTRY POINT
============================================================================ */}}

{{/*
cloudzero-agent.alloy.riverConfig - Generate complete Alloy River configuration

This is the main entry point that orchestrates all scrape job configurations
and the remote write sink. Each scrape job is conditionally included based
on its enabled state in Values.

The configuration is organized in this order:
1. Cost-critical scrape jobs (KSM, cAdvisor)
2. Operational monitoring jobs (webhook, aggregator, self)
3. Optional jobs (GPU metrics)
4. Remote write sink (shared by all jobs)

Usage: {{ include "cloudzero-agent.alloy.riverConfig" . }}
*/}}
{{- define "cloudzero-agent.alloy.riverConfig" -}}
// ============================================================================
// CloudZero Agent - Alloy Configuration
// Generated by Helm chart version {{ .Chart.Version }}
// ============================================================================
//
// This configuration collects Kubernetes metrics for CloudZero cost allocation.
// Data flows through discovery -> scrape -> relabel -> remote_write pipelines.
//
// Configuration sections:
// 1. Kube-State-Metrics  - Kubernetes object state (pods, nodes, resources)
// 2. cAdvisor            - Container resource usage (CPU, memory, network)
// 3. Webhook             - CloudZero webhook server health (observability)
// 4. Aggregator          - CloudZero aggregator health (observability)
// 5. Alloy Self          - Alloy's own metrics (observability)
// 6. DCGM GPU            - NVIDIA GPU metrics (optional)
// 7. Remote Write        - Sends all metrics to CloudZero aggregator
// ============================================================================

{{- if .Values.prometheusConfig.scrapeJobs.kubeStateMetrics.enabled }}
{{ include "cloudzero-agent.alloy.scrapeKubeStateMetrics" . }}
{{- end }}

{{- if include "cloudzero-agent.Values.integrations.cAdvisor.enabled" . }}
{{- include "cloudzero-agent.alloy.scrapeCAdvisor" . }}
{{- end }}

{{- if .Values.insightsController.enabled }}
{{ include "cloudzero-agent.alloy.scrapeWebhook" . }}
{{- end }}

{{- if .Values.prometheusConfig.scrapeJobs.aggregator.enabled }}
{{ include "cloudzero-agent.alloy.scrapeAggregator" . }}
{{- end }}

{{- if .Values.prometheusConfig.scrapeJobs.prometheus.enabled }}
{{ include "cloudzero-agent.alloy.scrapeAlloy" . }}
{{- end }}

{{- if .Values.prometheusConfig.scrapeJobs.gpu.enabled }}
{{ include "cloudzero-agent.alloy.scrapeGPU" . }}
{{- end }}

{{ include "cloudzero-agent.alloy.remoteWrite" . }}
{{- end -}}


{{/* =========================================================================
                           KUBE-STATE-METRICS SCRAPE JOB
============================================================================ */}}

{{/*
cloudzero-agent.alloy.scrapeKubeStateMetrics - Collect Kubernetes object state

Kube-State-Metrics (KSM) provides information about the configuration and state
of Kubernetes objects. This is essential for CloudZero cost allocation because
it provides:

  - kube_node_info           - Node instance types for cost correlation
  - kube_node_status_capacity - Node resource capacity
  - kube_pod_info            - Pod placement and ownership
  - kube_pod_labels          - Labels for cost attribution
  - kube_pod_container_resource_* - Resource requests and limits

Pipeline flow:
  +---------------------+     +---------------------+     +---------------------+
  | prometheus.scrape   | --> | prometheus.relabel  | --> | prometheus.relabel  |
  | "kube_state_metrics"|     | (keep cost metrics) |     | (keep cost labels)  |
  +---------------------+     +---------------------+     +---------------------+
                                                                   |
                                                                   v
                                                          [remote_write]

This uses a static target because KSM is typically deployed as a single service
with a well-known address within the cluster.

Usage: {{ include "cloudzero-agent.alloy.scrapeKubeStateMetrics" . }}
*/}}
{{- define "cloudzero-agent.alloy.scrapeKubeStateMetrics" -}}
// ============================================================================
// KUBE-STATE-METRICS SCRAPE JOB
// ============================================================================
//
// Purpose: Collect Kubernetes object state for cost allocation
//
// Data flow:
//   static target -> scrape -> filter metrics -> filter labels -> remote_write
//
// Metrics collected:
//   - kube_node_info, kube_node_status_capacity (node cost attribution)
//   - kube_pod_info, kube_pod_labels (pod cost attribution)
//   - kube_pod_container_resource_* (resource requests/limits)
// ============================================================================

// STEP 1: Scrape KSM endpoint
// KSM is deployed as a ClusterIP service, so we use a static target
prometheus.scrape "kube_state_metrics" {
  targets = [{
    __address__ = "{{ include "cloudzero-agent.kubeStateMetrics.kubeStateMetricsSvcTargetName" . }}",
  }]
  forward_to      = [prometheus.relabel.kube_state_metrics_filter.receiver]
  scrape_interval = "{{ .Values.prometheusConfig.scrapeJobs.kubeStateMetrics.scrapeInterval }}"

  // Enable clustering for distributed scraping across Alloy replicas
  clustering {
    enabled = true
  }
}

// STEP 2: Filter metrics - keep only CloudZero cost metrics
// This reduces data volume by dropping metrics we don't need
prometheus.relabel "kube_state_metrics_filter" {
  forward_to = [prometheus.relabel.kube_state_metrics_labels.receiver]

  rule {
    source_labels = ["__name__"]
    regex         = "^({{ join "|" (include "cloudzero-agent.defaults" . | fromYaml).kubeMetrics }})$"
    action        = "keep"
  }
}

// STEP 3: Filter labels - keep only labels needed for cost attribution
// This further reduces data volume and ensures consistent label sets
prometheus.relabel "kube_state_metrics_labels" {
  forward_to = [prometheus.remote_write.cloudzero.receiver]

  rule {
    regex  = "^({{ include "cloudzero-agent.requiredMetricLabels" . }})$"
    action = "labelkeep"
  }
}
{{- end -}}


{{/* =========================================================================
                              CADVISOR SCRAPE JOB
============================================================================ */}}

{{/*
cloudzero-agent.alloy.scrapeCAdvisor - Collect container resource usage metrics

cAdvisor (Container Advisor) provides actual resource usage metrics for containers.
This is the core of CloudZero cost allocation because it shows what resources
containers are actually consuming (not just what they requested).

Metrics collected:
  - container_cpu_usage_seconds_total      - CPU time consumed
  - container_memory_working_set_bytes     - Memory in active use
  - container_network_receive_bytes_total  - Network ingress
  - container_network_transmit_bytes_total - Network egress

IMPORTANT: The "node" label is critical for cost allocation. It must be added
during the discovery phase (discovery.relabel) because that's where the
__meta_kubernetes_node_name metadata is available. If you try to add it in
prometheus.relabel, the __meta_* labels will not be present.

Pipeline flow:
  +---------------------+     +---------------------+     +---------------------+
  | discovery.kubernetes| --> | discovery.relabel   | --> | prometheus.scrape   |
  | (discover nodes)    |     | (add node label,    |     | (TLS + auth)        |
  |                     |     |  map node metadata) |     |                     |
  +---------------------+     +---------------------+     +---------------------+
                                                                   |
                                                                   v
                                                    +---------------------+
                                                    | prometheus.relabel  |
                                                    | (filter metrics,    |
                                                    |  filter labels)     |
                                                    +---------------------+
                                                                   |
                                                                   v
                                                          [remote_write]

TLS Configuration:
  - Uses ServiceAccount token for authentication (bearer token)
  - CA certificate validates kubelet identity
  - insecure_skip_verify=true handles self-signed kubelet certs (matches Prometheus)

Usage: {{ include "cloudzero-agent.alloy.scrapeCAdvisor" . }}
*/}}
{{- define "cloudzero-agent.alloy.scrapeCAdvisor" -}}
{{- $directNodeAccess := .Values.integrations.cAdvisor.directNodeAccess.enabled | default false -}}
{{- $kubeletPort := .Values.integrations.cAdvisor.port | default 10250 -}}
{{- $insecureSkipVerify := .Values.integrations.cAdvisor.tls.insecureSkipVerify -}}
// ============================================================================
// CADVISOR SCRAPE JOB
// ============================================================================
//
// Purpose: Collect container resource usage for cost allocation
//
// Data flow:
//   discover nodes -> add node label -> scrape -> filter -> remote_write
//
// CRITICAL: The node label mapping happens in discovery.relabel because
// __meta_kubernetes_node_name is only available BEFORE scraping, not after!
//
// Metrics collected:
//   - container_cpu_usage_seconds_total
//   - container_memory_working_set_bytes
//   - container_network_receive_bytes_total
//   - container_network_transmit_bytes_total
//   - container_resources_gpu_* (if GPU metrics enabled)
// ============================================================================

// STEP 1: Discover Kubernetes nodes
// Each node runs a kubelet with cAdvisor integrated at /metrics/cadvisor
discovery.kubernetes "cadvisor" {
  role = "node"
}

{{- if $directNodeAccess }}
// STEP 2: Pre-scrape target relabeling (DIRECT NODE ACCESS MODE)
// Connect directly to kubelets on port {{ $kubeletPort }}, bypassing API server proxy
// This only requires nodes/metrics RBAC (not nodes/proxy)
//
// IMPORTANT: __meta_* labels are ONLY available here, not in prometheus.relabel!
discovery.relabel "cadvisor" {
  targets = discovery.kubernetes.cadvisor.targets

  // Rewrite target address to node's internal IP on kubelet port
  rule {
    source_labels = ["__meta_kubernetes_node_address_InternalIP"]
    target_label  = "__address__"
    replacement   = "$1:{{ $kubeletPort }}"
  }

  // Map node name to "node" label for cost allocation
  // This label identifies which node each metric came from
  rule {
    source_labels = ["__meta_kubernetes_node_name"]
    target_label  = "node"
  }

  // Copy node labels (e.g., node_kubernetes_io_instance_type for cost correlation)
  // This extracts labels from __meta_kubernetes_node_label_<name> format
  rule {
    regex  = "__meta_kubernetes_node_label_(.+)"
    action = "labelmap"
  }
}

// STEP 3: Scrape cAdvisor metrics directly from kubelets
// Kubelets expose cAdvisor at https://<node>:{{ $kubeletPort }}/metrics/cadvisor
prometheus.scrape "cadvisor" {
  targets         = discovery.relabel.cadvisor.output
  forward_to      = [prometheus.relabel.cadvisor_filter.receiver]
  scrape_interval = "{{ .Values.prometheusConfig.scrapeJobs.cadvisor.scrapeInterval }}"
  metrics_path    = "/metrics/cadvisor"
  scheme          = "https"

  // Authentication: Use ServiceAccount token mounted in the pod
  bearer_token_file = "/var/run/secrets/kubernetes.io/serviceaccount/token"

  // TLS Configuration:
  // - ca_file: Validates the kubelet's server certificate chain
  // - insecure_skip_verify: Configurable for environments with custom kubelet certs
  tls_config {
    ca_file              = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
    insecure_skip_verify = {{ $insecureSkipVerify }}
  }

  // Enable clustering for distributed scraping across Alloy replicas
  // Each Alloy instance will scrape a subset of nodes
  clustering {
    enabled = true
  }
}
{{- else }}
// STEP 2: Pre-scrape target relabeling (API SERVER PROXY MODE - default)
// Route requests through Kubernetes API server at /api/v1/nodes/<node>/proxy/
// This requires nodes/proxy RBAC permission
//
// IMPORTANT: __meta_* labels are ONLY available here, not in prometheus.relabel!
discovery.relabel "cadvisor" {
  targets = discovery.kubernetes.cadvisor.targets

  // Route all requests through the Kubernetes API server
  rule {
    target_label = "__address__"
    replacement  = "kubernetes.default.svc.cluster.local:443"
  }

  // Set metrics path to API proxy endpoint for each node
  rule {
    source_labels = ["__meta_kubernetes_node_name"]
    target_label  = "__metrics_path__"
    replacement   = "/api/v1/nodes/$1/proxy/metrics/cadvisor"
  }

  // Map node name to "node" label for cost allocation
  // This label identifies which node each metric came from
  rule {
    source_labels = ["__meta_kubernetes_node_name"]
    target_label  = "node"
  }

  // Copy node labels (e.g., node_kubernetes_io_instance_type for cost correlation)
  // This extracts labels from __meta_kubernetes_node_label_<name> format
  rule {
    regex  = "__meta_kubernetes_node_label_(.+)"
    action = "labelmap"
  }
}

// STEP 3: Scrape cAdvisor metrics via API server proxy
// API server forwards requests to kubelets at /api/v1/nodes/<node>/proxy/metrics/cadvisor
prometheus.scrape "cadvisor" {
  targets         = discovery.relabel.cadvisor.output
  forward_to      = [prometheus.relabel.cadvisor_filter.receiver]
  scrape_interval = "{{ .Values.prometheusConfig.scrapeJobs.cadvisor.scrapeInterval }}"
  scheme          = "https"

  // Authentication: Use ServiceAccount token mounted in the pod
  bearer_token_file = "/var/run/secrets/kubernetes.io/serviceaccount/token"

  // TLS Configuration:
  // - ca_file: Validates the API server's certificate chain
  // - insecure_skip_verify: Configurable for environments with custom certs
  tls_config {
    ca_file              = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
    insecure_skip_verify = {{ $insecureSkipVerify }}
  }

  // Enable clustering for distributed scraping across Alloy replicas
  // Each Alloy instance will scrape a subset of nodes
  clustering {
    enabled = true
  }
}
{{- end }}

// STEP 4: Filter metrics - keep only CloudZero cost metrics
prometheus.relabel "cadvisor_filter" {
  forward_to = [prometheus.relabel.cadvisor_labels.receiver]

  rule {
    source_labels = ["__name__"]
    regex         = "^({{ join "|" (include "cloudzero-agent.defaults" . | fromYaml).containerMetrics }})$"
    action        = "keep"
  }
}

// STEP 5: Filter labels - keep only labels needed for cost attribution
prometheus.relabel "cadvisor_labels" {
  forward_to = [prometheus.remote_write.cloudzero.receiver]

  rule {
    regex  = "^({{ include "cloudzero-agent.requiredMetricLabels" . }})$"
    action = "labelkeep"
  }
}
{{- end -}}


{{/* =========================================================================
                             WEBHOOK SCRAPE JOB
============================================================================ */}}

{{/*
cloudzero-agent.alloy.scrapeWebhook - Monitor CloudZero webhook server health

The CloudZero Webhook server handles Kubernetes admission webhooks and needs
monitoring for operational visibility. These are observability metrics, not
cost metrics.

Pipeline flow:
  +---------------------+     +---------------------+     +---------------------+
  | discovery.kubernetes| --> | prometheus.scrape   | --> | prometheus.relabel  |
  | (endpoints)         |     | (HTTPS, skip verify)|     | (filter metrics)    |
  +---------------------+     +---------------------+     +---------------------+
                                                                   |
                                                                   v
                                                          [remote_write]

The webhook uses HTTPS with a self-signed certificate, so we skip TLS verification.

Usage: {{ include "cloudzero-agent.alloy.scrapeWebhook" . }}
*/}}
{{- define "cloudzero-agent.alloy.scrapeWebhook" -}}
// ============================================================================
// WEBHOOK SCRAPE JOB
// ============================================================================
//
// Purpose: Monitor CloudZero webhook server health (observability metrics)
//
// Data flow:
//   discover endpoints -> scrape (HTTPS) -> filter metrics -> remote_write
//
// Note: Uses HTTPS with insecure_skip_verify because the webhook server
// uses a self-signed certificate generated at deployment time.
// ============================================================================

// STEP 1: Discover webhook service endpoints
discovery.kubernetes "webhook" {
  role = "endpoints"

  selectors {
    role  = "endpoints"
    field = "metadata.name={{ include "cloudzero-agent.insightsController.server.webhookFullname" . }}"
  }
}

// STEP 2: Scrape webhook metrics over HTTPS
prometheus.scrape "webhook" {
  targets         = discovery.kubernetes.webhook.targets
  forward_to      = [prometheus.relabel.webhook_filter.receiver]
  scrape_interval = "{{ .Values.prometheusConfig.scrapeJobs.prometheus.scrapeInterval }}"
  scheme          = "https"

  // Skip TLS verification - webhook uses self-signed cert
  tls_config {
    insecure_skip_verify = true
  }

  clustering {
    enabled = true
  }
}

// STEP 3: Filter metrics - keep only observability metrics
prometheus.relabel "webhook_filter" {
  forward_to = [prometheus.remote_write.cloudzero.receiver]

  rule {
    source_labels = ["__name__"]
    regex         = "^({{ join "|" (include "cloudzero-agent.defaults" . | fromYaml).insightsMetrics }})$"
    action        = "keep"
  }
}
{{- end -}}


{{/* =========================================================================
                            AGGREGATOR SCRAPE JOB
============================================================================ */}}

{{/*
cloudzero-agent.alloy.scrapeAggregator - Monitor CloudZero aggregator health

The CloudZero Aggregator receives metrics from Prometheus/Alloy and forwards
them to the CloudZero platform. Monitoring it ensures visibility into the
data pipeline health.

Pipeline flow:
  +---------------------+     +---------------------+
  | prometheus.scrape   | --> | prometheus.relabel  | --> [remote_write]
  | (static target)     |     | (filter metrics)    |
  +---------------------+     +---------------------+

Usage: {{ include "cloudzero-agent.alloy.scrapeAggregator" . }}
*/}}
{{- define "cloudzero-agent.alloy.scrapeAggregator" -}}
// ============================================================================
// AGGREGATOR SCRAPE JOB
// ============================================================================
//
// Purpose: Monitor CloudZero aggregator health (observability metrics)
//
// Data flow:
//   static target -> scrape -> filter metrics -> remote_write
//
// The aggregator exposes metrics on its shipper port.
// ============================================================================

// STEP 1: Scrape aggregator shipper port
prometheus.scrape "aggregator" {
  targets = [{
    __address__ = "{{ include "cloudzero-agent.aggregator.name" . }}:{{ .Values.aggregator.shipper.port }}",
  }]
  forward_to      = [prometheus.relabel.aggregator_filter.receiver]
  scrape_interval = "{{ .Values.prometheusConfig.scrapeJobs.aggregator.scrapeInterval }}"

  clustering {
    enabled = true
  }
}

// STEP 2: Filter metrics - keep cost + observability metrics
prometheus.relabel "aggregator_filter" {
  forward_to = [prometheus.remote_write.cloudzero.receiver]

  rule {
    source_labels = ["__name__"]
    regex         = "{{ include "cloudzero-agent.generateMetricNameFilterRegex" .Values }}"
    action        = "keep"
  }
}
{{- end -}}


{{/* =========================================================================
                             ALLOY SELF-SCRAPE JOB
============================================================================ */}}

{{/*
cloudzero-agent.alloy.scrapeAlloy - Monitor Alloy's own performance

Alloy exposes Prometheus-compatible metrics about its own operation. These
help monitor Alloy's health, memory usage, and scrape success rates.

Pipeline flow:
  +---------------------+     +---------------------+     +---------------------+
  | discovery.relabel   | --> | prometheus.scrape   | --> | prometheus.relabel  |
  | (add job/instance)  |     | (localhost:9090)    |     | (filter metrics)    |
  +---------------------+     +---------------------+     +---------------------+
                                                                   |
                                                                   v
                                                          [remote_write]

The discovery.relabel adds job and instance labels to identify this Alloy instance.

Usage: {{ include "cloudzero-agent.alloy.scrapeAlloy" . }}
*/}}
{{- define "cloudzero-agent.alloy.scrapeAlloy" -}}
// ============================================================================
// ALLOY SELF-SCRAPE JOB
// ============================================================================
//
// Purpose: Monitor Alloy's own health and performance (observability)
//
// Data flow:
//   add job/instance labels -> scrape localhost -> filter metrics -> remote_write
//
// Uses discovery.relabel to set the job name and instance (hostname) labels.
// ============================================================================

// STEP 1: Create target with identifying labels
discovery.relabel "alloy" {
  targets = [{
    __address__ = "localhost:9090",
  }]

  // Set job label to identify this as Alloy self-metrics
  rule {
    target_label = "job"
    replacement  = "alloy"
  }

  // Set instance label to the pod hostname for identification
  rule {
    target_label = "instance"
    replacement  = env("HOSTNAME")
  }
}

// STEP 2: Scrape Alloy's metrics endpoint
prometheus.scrape "alloy" {
  targets         = discovery.relabel.alloy.output
  forward_to      = [prometheus.relabel.alloy_filter.receiver]
  scrape_interval = "{{ .Values.prometheusConfig.scrapeJobs.prometheus.scrapeInterval }}"

  clustering {
    enabled = true
  }
}

// STEP 3: Filter metrics - keep only relevant operational metrics
prometheus.relabel "alloy_filter" {
  forward_to = [prometheus.remote_write.cloudzero.receiver]

  rule {
    source_labels = ["__name__"]
    regex         = "^({{ join "|" (include "cloudzero-agent.defaults" . | fromYaml).prometheusMetrics }})$"
    action        = "keep"
  }
}
{{- end -}}


{{/* =========================================================================
                              DCGM GPU SCRAPE JOB
============================================================================ */}}

{{/*
cloudzero-agent.alloy.scrapeGPU - Collect NVIDIA GPU metrics for cost allocation

The NVIDIA DCGM (Data Center GPU Manager) Exporter provides GPU utilization and
memory metrics. These are essential for GPU cost allocation in Kubernetes.

Raw metrics collected:
  - DCGM_FI_DEV_GPU_UTIL    - GPU compute utilization (0-100%)
  - DCGM_FI_DEV_FB_USED     - GPU memory used (bytes)
  - DCGM_FI_DEV_FB_FREE     - GPU memory free (bytes)

These are transformed by the CloudZero collector into:
  - container_resources_gpu_usage_percent
  - container_resources_gpu_memory_usage_percent

Pipeline flow:
  +---------------------+     +---------------------+     +---------------------+
  | discovery.kubernetes| --> | prometheus.scrape   | --> | prometheus.relabel  |
  | (dcgm services)     |     |                     |     | (add provenance,    |
  |                     |     |                     |     |  k8s metadata)      |
  +---------------------+     +---------------------+     +---------------------+
                                                                   |
                                                                   v
                                                    +---------------------+
                                                    | prometheus.relabel  |
                                                    | (filter metrics)    |
                                                    +---------------------+
                                                                   |
                                                                   v
                                                    +---------------------+
                                                    | prometheus.relabel  |
                                                    | (require container  |
                                                    |  attribution)       |
                                                    +---------------------+
                                                                   |
                                                                   v
                                                          [remote_write]

Usage: {{ include "cloudzero-agent.alloy.scrapeGPU" . }}
*/}}
{{- define "cloudzero-agent.alloy.scrapeGPU" -}}
// ============================================================================
// NVIDIA DCGM GPU METRICS SCRAPE JOB
// ============================================================================
//
// Purpose: Collect GPU metrics for cost allocation
//
// Data flow:
//   discover dcgm -> scrape -> add provenance -> filter metrics ->
//   require attribution -> remote_write
//
// Raw DCGM metrics are transformed by the collector into container_resources_*
// metrics for consistent cost allocation with CPU and memory.
// ============================================================================

// STEP 1: Discover DCGM Exporter services
discovery.kubernetes "dcgm" {
  role = "service"

  selectors {
    role  = "service"
    label = "app.kubernetes.io/name=dcgm-exporter"
  }
}

// STEP 2: Scrape DCGM metrics
prometheus.scrape "dcgm" {
  targets         = discovery.kubernetes.dcgm.targets
  forward_to      = [prometheus.relabel.dcgm_provenance.receiver]
  scrape_interval = "{{ .Values.prometheusConfig.scrapeJobs.gpu.scrapeInterval }}"

  clustering {
    enabled = true
  }
}

// STEP 3: Add provenance label and Kubernetes metadata
// This identifies these metrics as coming from DCGM for downstream processing
prometheus.relabel "dcgm_provenance" {
  forward_to = [prometheus.relabel.dcgm_filter.receiver]

  // Add provenance label to identify metric source
  rule {
    source_labels = ["__meta_kubernetes_service_label_app_kubernetes_io_name"]
    regex         = "dcgm-exporter"
    replacement   = "dcgm"
    target_label  = "provenance"
  }

  // Add Kubernetes metadata for context
  rule {
    source_labels = ["__meta_kubernetes_namespace"]
    target_label  = "kubernetes_namespace"
  }

  rule {
    source_labels = ["__meta_kubernetes_service_name"]
    target_label  = "kubernetes_service"
  }
}

// STEP 4: Filter metrics - keep only the 3 DCGM metrics needed
prometheus.relabel "dcgm_filter" {
  forward_to = [prometheus.relabel.dcgm_attribution.receiver]

  rule {
    source_labels = ["__name__"]
    regex         = "DCGM_FI_DEV_GPU_UTIL|DCGM_FI_DEV_FB_USED|DCGM_FI_DEV_FB_FREE"
    action        = "keep"
  }
}

// STEP 5: Filter out metrics without container attribution
// GPU metrics without container/pod/namespace are node-level and can't be
// attributed to specific workloads for cost allocation
prometheus.relabel "dcgm_attribution" {
  forward_to = [prometheus.remote_write.cloudzero.receiver]

  // Drop if container label is empty
  rule {
    source_labels = ["container"]
    regex         = "^$"
    action        = "drop"
  }

  // Drop if pod label is empty
  rule {
    source_labels = ["pod"]
    regex         = "^$"
    action        = "drop"
  }

  // Drop if namespace label is empty
  rule {
    source_labels = ["namespace"]
    regex         = "^$"
    action        = "drop"
  }
}
{{- end -}}


{{/* =========================================================================
                              REMOTE WRITE SINK
============================================================================ */}}

{{/*
cloudzero-agent.alloy.remoteWrite - Send metrics to CloudZero aggregator

This is the final destination for all scraped metrics. It sends data to the
CloudZero Aggregator running in the same namespace, which then forwards the
data to the CloudZero platform.

The queue_config settings balance throughput with resource usage:
  - capacity: 10000          - Buffer up to 10k samples before blocking
  - max_shards: 10           - Up to 10 parallel senders
  - max_samples_per_send: 5000 - Batch size for efficiency
  - batch_send_deadline: 5s  - Max wait time before sending partial batch

Usage: {{ include "cloudzero-agent.alloy.remoteWrite" . }}
*/}}
{{- define "cloudzero-agent.alloy.remoteWrite" -}}
// ============================================================================
// REMOTE WRITE TO CLOUDZERO AGGREGATOR
// ============================================================================
//
// Purpose: Send all collected metrics to the CloudZero aggregator
//
// All scrape jobs forward their filtered metrics here. The aggregator
// then ships the data to the CloudZero platform for cost analysis.
//
// Queue configuration is tuned for reliability and efficiency:
//   - Large capacity prevents data loss during aggregator hiccups
//   - Multiple shards enable parallel sending
//   - Reasonable batch sizes balance latency and throughput
// ============================================================================

prometheus.remote_write "cloudzero" {
  endpoint {
    url = "http://{{ include "cloudzero-agent.aggregator.name" . }}.{{ .Release.Namespace }}.svc.cluster.local/collector"

    // Disable metadata - not needed for CloudZero cost metrics
    send_exemplars         = false
    send_native_histograms = false

    // Queue configuration for reliable delivery
    queue_config {
      // Maximum number of samples to buffer before blocking new scrapes
      capacity = 10000

      // Parallel sender configuration
      // More shards = higher throughput but more connections
      max_shards = 10
      min_shards = 1

      // Batch configuration
      // Larger batches are more efficient but increase latency
      max_samples_per_send = 5000
      batch_send_deadline  = "5s"

      // Backoff configuration for retries
      min_backoff = "30ms"
      max_backoff = "5s"
    }
  }
}
{{- end -}}
