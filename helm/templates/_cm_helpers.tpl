{{/*
================================================================================
              CLOUDZERO AGENT CONFIGURATION HELPERS
================================================================================

This template file generates configuration for CloudZero Agent components:

1. COMPONENT CONFIGURATIONS (YAML)
   - Insights Controller (webhook server) configuration
   - Aggregator (collector + shipper) configuration

2. PROMETHEUS SCRAPE CONFIGURATIONS
   - Scrape job definitions for metrics collection
   - Remote write configuration for aggregator integration

================================================================================
                     PROMETHEUS ARCHITECTURE OVERVIEW
================================================================================

Prometheus collects metrics through a pull-based model where it scrapes HTTP
endpoints at configured intervals. The CloudZero Agent configures Prometheus
to collect cost-relevant metrics from various sources.

Data Flow:

    +-----------------+     +-----------------+     +-----------------+
    | Kubernetes      | --> | Prometheus      | --> | CloudZero       |
    | Targets         |     | Scrape + Filter |     | Aggregator      |
    | (KSM, cAdvisor, |     | (relabel_configs|     | (remote_write)  |
    |  webhook, etc.) |     |  metric_relabel)|     |                 |
    +-----------------+     +-----------------+     +-----------------+
                                                            |
                                                            v
                                                    +-----------------+
                                                    | CloudZero       |
                                                    | Platform        |
                                                    +-----------------+

================================================================================
                        PROMETHEUS RELABELING STAGES
================================================================================

Prometheus has TWO distinct relabeling stages that serve different purposes:

1. relabel_configs (BEFORE scrape):
   - Operates on TARGET metadata (__meta_* labels)
   - Used for target selection and label enrichment
   - Can modify __address__, __metrics_path__, etc.
   - Runs BEFORE metrics are scraped

2. metric_relabel_configs (AFTER scrape):
   - Operates on SCRAPED METRICS
   - Used for metric filtering (keep/drop)
   - Used for label filtering (labelkeep/labeldrop)
   - __meta_* labels are NOT available here!
   - Runs AFTER metrics are scraped

Example flow for cAdvisor:

    +-------------------+     +-------------------+     +-------------------+
    | kubernetes_sd     | --> | relabel_configs   | --> | SCRAPE            |
    | discovers nodes   |     | - add node label  |     | /metrics/cadvisor |
    | (__meta_* labels) |     | - set __address__ |     |                   |
    +-------------------+     +-------------------+     +-------------------+
                                                                |
                                                                v
                                                    +-------------------+
                                                    | metric_relabel    |
                                                    | - keep metrics    |
                                                    | - keep labels     |
                                                    +-------------------+
                                                                |
                                                                v
                                                        [remote_write]

================================================================================
                           SCRAPE JOB SUMMARY
================================================================================

Job Name                    | Source                | Purpose
----------------------------|----------------------|---------------------------
static-kube-state-metrics   | KSM service          | K8s object state (cost)
cloudzero-nodes-cadvisor    | kubelet /metrics/... | Container usage (cost)
cloudzero-webhook-job       | Webhook endpoints    | Webhook health (obs)
cloudzero-aggregator-job    | Aggregator endpoints | Aggregator health (obs)
static-prometheus           | localhost:9090       | Prometheus health (obs)
cloudzero-dcgm-exporter     | DCGM services        | GPU metrics (cost)

Legend: cost = cost allocation, obs = observability

================================================================================
                        CONFIGURATION REFERENCE
================================================================================

For more information on Prometheus configuration:
- Scrape configs: https://prometheus.io/docs/prometheus/latest/configuration/configuration/#scrape_config
- Relabeling:     https://prometheus.io/docs/prometheus/latest/configuration/configuration/#relabel_config
- Remote write:   https://prometheus.io/docs/prometheus/latest/configuration/configuration/#remote_write

*/}}


{{/* =========================================================================
                    INSIGHTS CONTROLLER CONFIGURATION
============================================================================ */}}

{{/*
cloudzero-agent.insightsController.configuration - Webhook server configuration

Generates YAML configuration for the CloudZero Insights Controller (webhook server).
This component handles Kubernetes admission webhooks for cost allocation metadata.

Configuration includes:
- CloudZero platform integration (account, region, cluster)
- TLS certificate paths for webhook HTTPS
- Server settings (port, timeouts, reconnection)
- Label/annotation filters for resource tracking

Usage: {{ include "cloudzero-agent.insightsController.configuration" . }}
*/}}
{{ define "cloudzero-agent.insightsController.configuration" -}}
cloud_account_id: {{ .Values.cloudAccountId }}
region: {{ .Values.region }}
cluster_name: {{ .Values.clusterName }}
destination: {{ include "cloudzero-agent.metricsDestination" . }}
logging:
  level: {{ .Values.insightsController.server.logging.level }}
remote_write:
  send_interval: {{ .Values.insightsController.server.send_interval }}
  max_bytes_per_send: 500000
  send_timeout: {{ .Values.insightsController.server.send_timeout }}
  max_retries: 3
k8s_client:
  timeout: 30s
database:
  retention_time: 24h
  cleanup_interval: 3h
  batch_update_size: 500
api_key_path: {{ include "cloudzero-agent.secretFileFullPath" . }}
{{- $namespace := .Release.Namespace }}
{{- with .Values.insightsController }}
certificate:
  key: {{ .tls.mountPath }}/tls.key
  cert: {{ .tls.mountPath }}/tls.crt
server:
  namespace: {{ $namespace }}
  domain: {{ include "cloudzero-agent.serviceName" $ }}
  port: {{ .server.port }}
  read_timeout: {{ .server.read_timeout }}
  write_timeout: {{ .server.write_timeout }}
  idle_timeout: {{ .server.idle_timeout }}
  reconnect_frequency: {{ .server.reconnectFrequency | default 16 }}
{{- end }}
filters:
  labels:
  {{- .Values.insightsController.labels | toYaml | nindent 4 }}
  annotations:
  {{- .Values.insightsController.annotations | toYaml | nindent 4 }}
{{- end}}


{{/* =========================================================================
                       AGGREGATOR CONFIGURATION
============================================================================ */}}

{{/*
cloudzero-agent.aggregator.configuration - Aggregator component configuration

Generates YAML configuration for the CloudZero Aggregator deployment.
The aggregator receives metrics via Prometheus remote_write, processes them,
and ships them to the CloudZero platform.

Configuration includes:
- CloudZero platform integration (API key, endpoints)
- Metric filtering rules (cost vs observability)
- Database settings (storage, retention, compression)
- HTTP server settings for remote_write endpoint

Usage: {{ include "cloudzero-agent.aggregator.configuration" . }}
*/}}
{{ define "cloudzero-agent.aggregator.configuration" -}}
cloud_account_id: {{ include "cloudzero-agent.cleanString" .Values.cloudAccountId }}
cluster_name: {{ include "cloudzero-agent.cleanString" .Values.clusterName }}
region: {{ include "cloudzero-agent.cleanString" .Values.region }}

metrics:
  {{- include "cloudzero-agent.generateMetricFilters" (dict "name" "cost" "filters"                 (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.cost.name) | nindent 2 }}
  {{- include "cloudzero-agent.generateMetricFilters" (dict "name" "cost_labels" "filters"          (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.cost.labels) | nindent 2 }}
  {{- include "cloudzero-agent.generateMetricFilters" (dict "name" "observability" "filters"        (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.observability.name) | nindent 2 }}
  {{- include "cloudzero-agent.generateMetricFilters" (dict "name" "observability_labels" "filters" (include "cloudzero-agent.defaults" . | fromYaml).metricFilters.observability.labels) | nindent 2 }}
server:
  mode: http
  port: {{ .Values.aggregator.collector.port }}
  profiling: {{ .Values.aggregator.profiling }}
  reconnect_frequency: {{ .Values.aggregator.reconnectFrequency }}
logging:
  level: "{{ .Values.aggregator.logging.level }}"
  capture: {{ .Values.aggregator.logging.capture }}
database:
  storage_path: {{ .Values.aggregator.mountRoot }}/data
  max_records: {{ .Values.aggregator.database.maxRecords }}
  cost_max_interval: {{ .Values.aggregator.database.costMaxInterval }}
  observability_max_interval: {{ .Values.aggregator.database.observabilityMaxInterval }}
  compression_level: {{ .Values.aggregator.database.compressionLevel }}
  purge_rules:
    metrics_older_than: {{ .Values.aggregator.database.purgeRules.metricsOlderThan }}
    lazy: {{ .Values.aggregator.database.purgeRules.lazy }}
    percent: {{ .Values.aggregator.database.purgeRules.percent }}
  {{- if .Values.aggregator.database.emptyDir.enabled }}
  available_storage: {{ .Values.aggregator.database.emptyDir.sizeLimit }}
  {{- end}}
cloudzero:
  api_key_path: {{ include "cloudzero-agent.secretFileFullPath" . }}
  send_interval: {{ .Values.aggregator.cloudzero.sendInterval }}
  send_timeout: {{ .Values.aggregator.cloudzero.sendTimeout }}
  rotate_interval: {{ .Values.aggregator.cloudzero.rotateInterval }}
  host: {{ .Values.host }}
  http_max_retries: {{ .Values.aggregator.cloudzero.httpMaxRetries }}
  http_max_wait: {{ .Values.aggregator.cloudzero.httpMaxWait }}
{{- end}}


{{/* =========================================================================
                      PROMETHEUS REMOTE WRITE CONFIGURATION
============================================================================ */}}

{{/*
cloudzero-agent.aggregator.remoteWrite - Remote write to CloudZero aggregator

Generates the remote_write configuration block for Prometheus. This sends
all scraped and filtered metrics to the CloudZero Aggregator.

Configuration:
- URL: CloudZero Aggregator's /collector endpoint
- Auth: CloudZero API key from mounted secret
- Filtering: write_relabel_configs for final metric selection
- Metadata: Disabled (not needed for cost metrics)

Usage: {{ include "cloudzero-agent.aggregator.remoteWrite" . }}
*/}}
{{- define "cloudzero-agent.aggregator.remoteWrite" -}}
# ============================================================================
# REMOTE WRITE TO CLOUDZERO AGGREGATOR
# ============================================================================
#
# Sends all collected metrics to the CloudZero Aggregator for processing
# and forwarding to the CloudZero platform.
#
# The write_relabel_configs provides a final filter to ensure only
# cost-relevant metrics are transmitted.
# ============================================================================
remote_write:
  - url: {{ include "cloudzero-agent.metricsDestination" . }}
    authorization:
      credentials_file: {{ include "cloudzero-agent.secretFileFullPath" . }}
    write_relabel_configs:
      # Final filter: only send metrics matching the cost/observability patterns
      - source_labels: [__name__]
        regex: "^({{ include "cloudzero-agent.combineMetrics" . }})$"
        action: keep
    metadata_config:
      # Disable metadata - not needed for CloudZero cost metrics
      send: false
{{- end -}}


{{/* =========================================================================
                    KUBE-STATE-METRICS SCRAPE JOB
============================================================================ */}}

{{/*
cloudzero-agent.prometheus.scrapeKubeStateMetrics - Kubernetes object state metrics

Collects metrics about Kubernetes object configuration and state from
Kube-State-Metrics (KSM). This is essential for cost allocation because
it provides resource requests, limits, and placement information.

Data flow:
  static_configs   -->   relabel_configs     -->   SCRAPE
  (KSM service)         (extract labels)          (get metrics)
                                                       |
                                                       v
                                             metric_relabel_configs
                                             (filter metrics/labels)
                                                       |
                                                       v
                                                 [remote_write]

Metrics collected:
  - kube_node_info           - Node metadata (instance type, provider ID)
  - kube_node_status_capacity - Node resource capacity
  - kube_pod_info            - Pod placement and ownership
  - kube_pod_labels          - Pod labels for cost attribution
  - kube_pod_container_resource_requests - Resource requests
  - kube_pod_container_resource_limits   - Resource limits

relabel_configs (BEFORE scrape):
  - labelmap: Copy __meta_kubernetes_service_label_* to metric labels
  - namespace: Extract from __meta_kubernetes_namespace
  - service: Extract from __meta_kubernetes_service_name
  - node: Extract from __meta_kubernetes_pod_node_name

metric_relabel_configs (AFTER scrape):
  - keep: Only kube_* metrics needed for cost allocation
  - labelkeep: Only labels needed for cost attribution

Usage: {{ include "cloudzero-agent.prometheus.scrapeKubeStateMetrics" . }}
*/}}
{{- define "cloudzero-agent.prometheus.scrapeKubeStateMetrics" -}}
# ============================================================================
# KUBE-STATE-METRICS SCRAPE JOB
# ============================================================================
#
# Purpose: Collect Kubernetes object state for cost allocation
#
# KSM provides information about the configuration and state of Kubernetes
# objects (nodes, pods, etc.), including resource requests, limits, and
# placement. This data is essential for accurate cost attribution.
#
# Data flow:
#   static target -> relabel (extract metadata) -> scrape ->
#   metric_relabel (filter) -> remote_write
# ============================================================================
- job_name: static-kube-state-metrics
  scrape_interval: {{ .Values.prometheusConfig.scrapeJobs.kubeStateMetrics.scrapeInterval }}

  # -------------------------------------------------------------------------
  # STEP 1: relabel_configs - Extract Kubernetes metadata BEFORE scraping
  # -------------------------------------------------------------------------
  # These run BEFORE the scrape, operating on target metadata (__meta_* labels).
  # This is where we extract service discovery information into metric labels.
  #
  # Given a Kubernetes Service like:
  #
  #   apiVersion: v1
  #   kind: Service
  #   metadata:
  #     name: my-service
  #     namespace: my-namespace
  #     labels:
  #       app: my-app
  #
  # Service discovery provides labels like:
  #   __meta_kubernetes_service_name: my-service
  #   __meta_kubernetes_namespace: my-namespace
  #   __meta_kubernetes_service_label_app: my-app
  #
  # We transform these into metric labels for cost attribution.
  relabel_configs:
    # Copy all service labels to metric labels
    # __meta_kubernetes_service_label_app -> app
    - regex: __meta_kubernetes_service_label_(.+)
      action: labelmap

    # Extract namespace for cost attribution
    - source_labels: [__meta_kubernetes_namespace]
      target_label: namespace

    # Extract service name
    - source_labels: [__meta_kubernetes_service_name]
      target_label: service

    # Extract node name for node-level cost correlation
    - source_labels: [__meta_kubernetes_pod_node_name]
      target_label: node

  # -------------------------------------------------------------------------
  # STEP 2: metric_relabel_configs - Filter metrics AFTER scraping
  # -------------------------------------------------------------------------
  # These run AFTER the scrape, operating on the actual metrics.
  # Note: __meta_* labels are NOT available here!
  metric_relabel_configs:
    # Keep only the Kubernetes state metrics needed for cost allocation
    - source_labels: [__name__]
      regex: {{ printf "^(%s)$" (join "|" (include "cloudzero-agent.defaults" . | fromYaml).kubeMetrics) }}
      action: keep

    # Keep only labels needed for cost attribution
    # This reduces cardinality and storage requirements
    - regex: {{ printf "^(%s)$" (include "cloudzero-agent.requiredMetricLabels" .) }}
      action: labelkeep

  # -------------------------------------------------------------------------
  # STEP 3: Target configuration
  # -------------------------------------------------------------------------
  # KSM is typically a single service, so we use a static target
  static_configs:
    - targets:
      - {{ include "cloudzero-agent.kubeStateMetrics.kubeStateMetricsSvcTargetName" . }}
{{- end -}}


{{/* =========================================================================
                         CADVISOR SCRAPE JOB
============================================================================ */}}

{{/*
cloudzero-agent.prometheus.scrapeCAdvisor - Container resource usage metrics

Collects actual container resource usage from cAdvisor, which is integrated
into the kubelet on each node. This is the core of cost allocation - it shows
what resources containers are actually consuming.

TWO SCRAPING MODES:

1. Cluster-wide mode (scrapeLocalNodeOnly=false):
   - Uses Kubernetes API proxy: kubernetes.default.svc:443
   - Path: /api/v1/nodes/<node>/proxy/metrics/cadvisor
   - Used in Deployment mode

2. Local node mode (scrapeLocalNodeOnly=true):
   - Scrapes kubelet directly: <node-ip>:10250
   - Path: /metrics/cadvisor
   - Used in DaemonSet (federated) mode
   - Requires ${NODE_NAME} substitution via init container

Data flow:
  kubernetes_sd    -->   relabel_configs     -->   SCRAPE
  (discover nodes)      (set path/address,        (kubelet API)
                         add node label)               |
                                                       v
                                             metric_relabel_configs
                                             (filter metrics/labels)
                                                       |
                                                       v
                                                 [remote_write]

Metrics collected:
  - container_cpu_usage_seconds_total      - CPU time consumed
  - container_memory_working_set_bytes     - Active memory usage
  - container_network_receive_bytes_total  - Network ingress
  - container_network_transmit_bytes_total - Network egress

TLS Configuration:
  - Uses ServiceAccount bearer token for authentication
  - Uses cluster CA to verify kubelet certificate
  - insecure_skip_verify=true handles self-signed kubelet certs

CRITICAL: The "node" label mapping happens in relabel_configs because
__meta_kubernetes_node_name is only available BEFORE scraping!

Usage: {{ include "cloudzero-agent.prometheus.scrapeCAdvisor" (dict "root" . "scrapeLocalNodeOnly" false) }}
*/}}
{{- define "cloudzero-agent.prometheus.scrapeCAdvisor" -}}
{{- $scrapeLocal := .scrapeLocalNodeOnly | default false -}}
{{- $directNodeAccess := .root.Values.integrations.cAdvisor.directNodeAccess.enabled | default false -}}
{{- $kubeletPort := .root.Values.integrations.cAdvisor.port | default 10250 -}}
{{- $insecureSkipVerify := .root.Values.integrations.cAdvisor.tls.insecureSkipVerify -}}
# ============================================================================
# CADVISOR SCRAPE JOB
# ============================================================================
#
# Purpose: Collect container resource usage for cost allocation
#
# cAdvisor (Container Advisor) provides actual resource consumption metrics
# for containers. This is the foundation of container cost allocation.
#
{{- if $scrapeLocal }}
# Mode: FEDERATED (DaemonSet)
# - Each Prometheus instance scrapes only its local node's kubelet
# - Uses direct kubelet access at <node-ip>:{{ $kubeletPort }}/metrics/cadvisor
# - ${NODE_NAME} is replaced by init container with actual node name
{{- else if $directNodeAccess }}
# Mode: DIRECT NODE ACCESS
# - Single Prometheus scrapes all nodes directly via kubelet
# - Uses <node-ip>:{{ $kubeletPort }}/metrics/cadvisor
# - Requires network connectivity from collector pod to all nodes
# - Only requires nodes/metrics RBAC (not nodes/proxy)
{{- else }}
# Mode: API SERVER PROXY (default)
# - Single Prometheus scrapes all nodes via Kubernetes API proxy
# - Uses kubernetes.default.svc:443/api/v1/nodes/<node>/proxy/metrics/cadvisor
# - Requires nodes/proxy RBAC permission
{{- end }}
#
# CRITICAL: The node label must be added in relabel_configs (not
# metric_relabel_configs) because __meta_kubernetes_node_name is only
# available BEFORE scraping!
# ============================================================================
- job_name: cloudzero-nodes-cadvisor
  scrape_interval: {{ .root.Values.prometheusConfig.scrapeJobs.cadvisor.scrapeInterval }}
  scheme: https

  # -------------------------------------------------------------------------
  # Authentication and TLS
  # -------------------------------------------------------------------------
  # Kubelets require authentication. We use the ServiceAccount token mounted
  # in the pod for bearer token authentication.
  authorization:
    type: Bearer
    credentials_file: /var/run/secrets/kubernetes.io/serviceaccount/token

  # TLS configuration for secure kubelet communication:
  # - ca_file: Cluster CA validates kubelet identity
  # - insecure_skip_verify: true handles self-signed kubelet certificates
  #   (common in many Kubernetes distributions)
  tls_config:
    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    insecure_skip_verify: {{ $insecureSkipVerify }}

  {{- if $scrapeLocal }}
  # -------------------------------------------------------------------------
  # LOCAL NODE MODE: Direct kubelet access
  # -------------------------------------------------------------------------
  metrics_path: /metrics/cadvisor

  relabel_configs:
    # Extract node name to "node_name" label
    - source_labels: [__meta_kubernetes_node_name]
      target_label: node_name

    # Filter to only the local node
    # NOTE: ${NODE_NAME} is a placeholder replaced by an init container
    # with the actual node name using sed. This is necessary because
    # Prometheus doesn't support environment variable substitution.
    - source_labels: [__meta_kubernetes_node_name]
      regex: ${NODE_NAME}
      action: keep

    # Set target address to node's internal IP on kubelet port
    - source_labels: [__meta_kubernetes_node_address_InternalIP]
      target_label: __address__
      replacement: ${1}:{{ $kubeletPort }}
  {{- else if $directNodeAccess }}
  # -------------------------------------------------------------------------
  # DIRECT NODE ACCESS MODE: Scrape all nodes directly via kubelet
  # -------------------------------------------------------------------------
  # This bypasses the API server proxy and only requires nodes/metrics RBAC.
  # Requires network connectivity from collector pod to all nodes on port {{ $kubeletPort }}.
  metrics_path: /metrics/cadvisor

  relabel_configs:
    # Connect directly to node's internal IP on kubelet port
    - source_labels: [__meta_kubernetes_node_address_InternalIP]
      target_label: __address__
      replacement: ${1}:{{ $kubeletPort }}
  {{- else }}
  # -------------------------------------------------------------------------
  # API SERVER PROXY MODE (default): Route through Kubernetes API
  # -------------------------------------------------------------------------
  # Requires nodes/proxy RBAC permission.
  relabel_configs:
    # Route through Kubernetes API server
    - target_label: __address__
      replacement: kubernetes.default.svc.cluster.local:443

    # Use API proxy path to reach each node's cAdvisor
    # /api/v1/nodes/<node>/proxy/metrics/cadvisor
    - source_labels: [__meta_kubernetes_node_name]
      target_label: __metrics_path__
      replacement: /api/v1/nodes/$1/proxy/metrics/cadvisor
  {{- end }}

    # -------------------------------------------------------------------------
    # Common relabel_configs (both modes)
    # -------------------------------------------------------------------------
    # Copy node labels to metric labels for cost correlation
    # e.g., node_kubernetes_io_instance_type for EC2 instance type
    - regex: __meta_kubernetes_node_label_(.+)
      action: labelmap

    # Add "node" label for cost attribution
    # CRITICAL: This must be in relabel_configs, not metric_relabel_configs,
    # because __meta_kubernetes_node_name is only available BEFORE scraping!
    - source_labels: [__meta_kubernetes_node_name]
      target_label: node

  # -------------------------------------------------------------------------
  # metric_relabel_configs - Filter AFTER scraping
  # -------------------------------------------------------------------------
  metric_relabel_configs:
    # Keep only labels needed for cost attribution
    - action: labelkeep
      regex: {{ printf "^(%s)$" (include "cloudzero-agent.requiredMetricLabels" .root) }}

    # Keep only container metrics needed for cost allocation
    - source_labels: [__name__]
      regex: {{ printf "^(%s)$" (join "|" (include "cloudzero-agent.defaults" .root | fromYaml).containerMetrics) }}
      action: keep

  # -------------------------------------------------------------------------
  # Service Discovery
  # -------------------------------------------------------------------------
  kubernetes_sd_configs:
    - role: node
      kubeconfig_file: ""
{{- end -}}


{{/* =========================================================================
                         WEBHOOK SCRAPE JOB
============================================================================ */}}

{{/*
cloudzero-agent.prometheus.scrapeWebhookJob - Webhook server health metrics

Monitors the CloudZero Webhook server (Insights Controller) for operational
visibility. These are observability metrics, not cost metrics.

Data flow:
  kubernetes_sd    -->   relabel_configs     -->   SCRAPE
  (endpoints)           (filter to webhook)       (HTTPS)
                                                       |
                                                       v
                                             metric_relabel_configs
                                             (filter metrics)
                                                       |
                                                       v
                                                 [remote_write]

Metrics collected:
  - Process metrics (CPU, memory, goroutines)
  - HTTP request metrics
  - Webhook-specific business metrics (czo_webhook_types_total, etc.)

TLS: Uses HTTPS with insecure_skip_verify because the webhook uses
a self-signed certificate generated at deployment time.

Usage: {{ include "cloudzero-agent.prometheus.scrapeWebhookJob" . }}
*/}}
{{- define "cloudzero-agent.prometheus.scrapeWebhookJob" -}}
# ============================================================================
# WEBHOOK SCRAPE JOB
# ============================================================================
#
# Purpose: Monitor CloudZero webhook server health (observability metrics)
#
# The webhook server handles Kubernetes admission webhooks. Monitoring it
# ensures visibility into admission processing latency and throughput.
#
# Uses HTTPS with insecure_skip_verify because the webhook uses a self-signed
# certificate generated at deployment time.
# ============================================================================
- job_name: cloudzero-webhook-job
  scheme: https
  tls_config:
    insecure_skip_verify: true

  # Discover webhook service endpoints
  kubernetes_sd_configs:
    - role: endpoints
      kubeconfig_file: ""

  # Filter to only the webhook service
  relabel_configs:
    - source_labels: [__meta_kubernetes_endpoints_name]
      action: keep
      regex: {{ include "cloudzero-agent.insightsController.server.webhookFullname" . }}

  # Keep only CloudZero observability metrics
  metric_relabel_configs:
    - source_labels: [__name__]
      regex: "^({{ join "|" (include "cloudzero-agent.defaults" . | fromYaml).insightsMetrics }})$"
      action: keep
{{- end -}}


{{/* =========================================================================
                        AGGREGATOR SCRAPE JOB
============================================================================ */}}

{{/*
cloudzero-agent.prometheus.scrapeAggregator - Aggregator health metrics

Monitors the CloudZero Aggregator for operational visibility. These are
observability metrics that help track the health of the data pipeline.

Data flow:
  kubernetes_sd    -->   relabel_configs     -->   SCRAPE
  (endpoints)           (filter to agg ports)     (HTTP)
                                                       |
                                                       v
                                             metric_relabel_configs
                                             (filter metrics)
                                                       |
                                                       v
                                                 [remote_write]

The aggregator exposes metrics on both collector and shipper ports.
Both are scraped for comprehensive pipeline visibility.

Usage: {{ include "cloudzero-agent.prometheus.scrapeAggregator" . }}
*/}}
{{- define "cloudzero-agent.prometheus.scrapeAggregator" -}}
# ============================================================================
# AGGREGATOR SCRAPE JOB
# ============================================================================
#
# Purpose: Monitor CloudZero aggregator health (observability metrics)
#
# The aggregator receives metrics via remote_write and ships them to
# CloudZero. Monitoring it provides visibility into the data pipeline.
#
# Scrapes both collector and shipper ports for comprehensive monitoring.
# ============================================================================
- job_name: cloudzero-aggregator-job
  scrape_interval: {{ .Values.prometheusConfig.scrapeJobs.prometheus.scrapeInterval }}

  # Discover aggregator endpoints in this namespace
  kubernetes_sd_configs:
    - role: endpoints
      kubeconfig_file: ""
      namespaces:
        names:
          - {{ .Release.Namespace }}

  relabel_configs:
    # Filter to only the aggregator service
    - source_labels: [__meta_kubernetes_service_name]
      action: keep
      regex: {{ include "cloudzero-agent.aggregator.name" . }}

    # Filter to collector and shipper ports
    - source_labels: [__meta_kubernetes_pod_container_port_name]
      action: keep
      regex: port-(shipper|collector)

  # Keep cost + observability metrics
  metric_relabel_configs:
    - source_labels: [__name__]
      regex: "{{ include "cloudzero-agent.generateMetricNameFilterRegex" .Values }}"
      action: keep
{{- end -}}


{{/* =========================================================================
                      PROMETHEUS SELF-SCRAPE JOB
============================================================================ */}}

{{/*
cloudzero-agent.prometheus.scrapePrometheus - Prometheus self-monitoring

Scrapes Prometheus's own metrics for operational visibility into the
metrics collection pipeline health.

Metrics collected:
  - Process metrics (memory, CPU, goroutines)
  - Remote storage metrics (samples sent, failures)
  - Service discovery metrics (targets discovered)
  - Scrape metrics (duration, success rate)

Usage: {{ include "cloudzero-agent.prometheus.scrapePrometheus" . }}
*/}}
{{- define "cloudzero-agent.prometheus.scrapePrometheus" -}}
# ============================================================================
# PROMETHEUS SELF-SCRAPE JOB
# ============================================================================
#
# Purpose: Monitor Prometheus's own health and performance (observability)
#
# Scrapes localhost:9090 to collect Prometheus operational metrics.
# This provides visibility into scrape success, remote write health,
# and resource usage.
# ============================================================================
- job_name: static-prometheus
  scrape_interval: {{ .Values.prometheusConfig.scrapeJobs.prometheus.scrapeInterval }}

  static_configs:
    - targets:
        - localhost:9090

  # Keep only Prometheus operational metrics needed for monitoring
  metric_relabel_configs:
    - source_labels: [__name__]
      regex: "^({{ join "|" (include "cloudzero-agent.defaults" . | fromYaml).prometheusMetrics }})$"
      action: keep
{{- end -}}


{{/* =========================================================================
                          DCGM GPU SCRAPE JOB
============================================================================ */}}

{{/*
cloudzero-agent.prometheus.scrapeGPU - NVIDIA GPU metrics for cost allocation

Collects GPU utilization and memory metrics from NVIDIA DCGM Exporter
for GPU cost allocation. Only metrics with container attribution are kept.

Raw DCGM metrics collected:
  - DCGM_FI_DEV_GPU_UTIL - GPU compute utilization (0-100%)
  - DCGM_FI_DEV_FB_USED  - GPU frame buffer memory used (bytes)
  - DCGM_FI_DEV_FB_FREE  - GPU frame buffer memory free (bytes)

These raw metrics are transformed by the CloudZero collector into:
  - container_resources_gpu_usage_percent
  - container_resources_gpu_memory_usage_percent

Data flow:
  kubernetes_sd    -->   relabel_configs     -->   SCRAPE
  (services)            (add provenance,          (DCGM)
                         k8s metadata)                |
                                                       v
                                             metric_relabel_configs
                                             (filter metrics,
                                              require attribution)
                                                       |
                                                       v
                                                 [remote_write]

Note: This is specific to NVIDIA DCGM Exporter. Future GPU vendors
(AMD, Intel) will have separate scrape jobs.

Usage: {{ include "cloudzero-agent.prometheus.scrapeGPU" . }}
*/}}
{{- define "cloudzero-agent.prometheus.scrapeGPU" -}}
# ============================================================================
# NVIDIA DCGM GPU METRICS SCRAPE JOB
# ============================================================================
#
# Purpose: Collect GPU metrics for cost allocation
#
# DCGM (Data Center GPU Manager) Exporter provides per-GPU and per-container
# utilization metrics. These are essential for GPU cost allocation.
#
# Raw DCGM metrics are transformed by the CloudZero collector into
# container_resources_gpu_* metrics for consistent cost allocation.
#
# Note: This job is specific to NVIDIA DCGM Exporter. Future GPU vendors
# (AMD, Intel) will have separate scrape jobs.
# ============================================================================
- job_name: cloudzero-dcgm-exporter
  scrape_interval: {{ .Values.prometheusConfig.scrapeJobs.gpu.scrapeInterval }}

  # Discover DCGM Exporter services via label selector
  kubernetes_sd_configs:
    - role: service
      kubeconfig_file: ""
      selectors:
        - role: service
          label: "app.kubernetes.io/name=dcgm-exporter"

  # -------------------------------------------------------------------------
  # relabel_configs - Add metadata BEFORE scraping
  # -------------------------------------------------------------------------
  relabel_configs:
    # Add provenance label to identify metric source
    # This helps downstream processing identify these as DCGM metrics
    - source_labels: [__meta_kubernetes_service_label_app_kubernetes_io_name]
      regex: dcgm-exporter
      replacement: dcgm
      target_label: provenance

    # Add Kubernetes metadata for context
    - source_labels: [__meta_kubernetes_namespace]
      target_label: kubernetes_namespace

    - source_labels: [__meta_kubernetes_service_name]
      target_label: kubernetes_service

  # -------------------------------------------------------------------------
  # metric_relabel_configs - Filter AFTER scraping
  # -------------------------------------------------------------------------
  metric_relabel_configs:
    # Keep only the 3 DCGM metrics needed for cost allocation
    - source_labels: [__name__]
      regex: DCGM_FI_DEV_GPU_UTIL|DCGM_FI_DEV_FB_USED|DCGM_FI_DEV_FB_FREE
      action: keep

    # Drop metrics without container attribution
    # (These are node-level GPU metrics that can't be attributed to workloads)
    - source_labels: [container]
      regex: ^$
      action: drop

    - source_labels: [pod]
      regex: ^$
      action: drop

    - source_labels: [namespace]
      regex: ^$
      action: drop
{{- end -}}
