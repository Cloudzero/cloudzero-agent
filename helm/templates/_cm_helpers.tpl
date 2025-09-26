{{/*
CloudZero Agent Insights Controller Configuration Template

Generates the complete configuration for the CloudZero Agent Insights Controller (webhook server)
deployment. This template centralizes all configuration parameters needed for Kubernetes admission
webhook operations, cost allocation metadata processing, and CloudZero platform integration.

Configuration includes:
- CloudZero platform integration settings (cloud account, region, cluster identification)
- Kubernetes admission webhook server configuration (TLS, timeouts, networking)
- Metric collection and filtering for cost allocation analysis
- Database settings for resource metadata persistence
- Logging and monitoring configuration for operational observability

The configuration is defined as a template to enable:
- Configuration drift detection through checksum-based pod rolling updates
- Centralized configuration management across multiple chart templates
- Consistent parameter validation and default value application
- Environment-specific customization through values.yaml overrides

Usage: This template is consumed by ConfigMap and Deployment templates to ensure
consistent configuration across all Insights Controller components.
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


{{/*
CloudZero Agent Aggregator Configuration Template

Generates the complete configuration for the CloudZero Agent Aggregator deployment, which handles
Prometheus remote_write metric ingestion, data processing, filtering, and transmission to the
CloudZero platform for cost allocation analysis.

Configuration includes:
- CloudZero platform integration (API keys, endpoints, retry policies)
- Metric filtering rules for cost vs observability data classification
- Database configuration for metric storage, retention, and compression
- HTTP server settings for Prometheus remote_write endpoint
- Data processing parameters (batch sizes, intervals, compression levels)
- Monitoring and logging configuration for operational visibility

The aggregator processes high-volume metric streams from Prometheus instances across
Kubernetes clusters, applying intelligent filtering to identify cost-relevant metrics
while maintaining operational observability data for monitoring and troubleshooting.

Template benefits:
- Configuration checksum-based deployment updates for zero-downtime changes
- Centralized parameter management with validation and defaults
- Environment-specific customization through values.yaml inheritance
- Consistent configuration across aggregator components (collector, shipper)
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

{{/*
Prometheus Remote Write Configuration Template

Generates Prometheus remote_write configuration for sending metrics to CloudZero Agent.
This template creates the YAML configuration block that Prometheus instances use to
forward metrics to the CloudZero Agent aggregator for cost allocation processing.

Configuration features:
- Target URL: CloudZero Agent aggregator remote_write endpoint
- Authentication: CloudZero API key-based authorization
- Metric filtering: Write-time filtering using relabel configs
- Metadata handling: Disabled to reduce overhead and focus on metric data

The remote_write configuration enables:
- High-throughput metric transmission from Prometheus to CloudZero Agent
- Intelligent metric selection to reduce bandwidth and processing overhead
- Secure authentication using CloudZero platform credentials
- Reliable delivery with Prometheus built-in retry mechanisms

This template is used by Prometheus configuration templates to establish
the integration between existing Prometheus deployments and CloudZero cost allocation.
*/}}
{{- define "cloudzero-agent.aggregator.remoteWrite" -}}
remote_write:
  - url: {{ include "cloudzero-agent.metricsDestination" . }}
    authorization:
      credentials_file: {{ include "cloudzero-agent.secretFileFullPath" . }}
    write_relabel_configs:
      - source_labels: [__name__]
        regex: "^({{ include "cloudzero-agent.combineMetrics" . }})$"
        action: keep
    metadata_config:
      send: false
{{- end -}}

{{/*
Prometheus Self-Monitoring Scrape Job Configuration Template

Generates Prometheus scrape job configuration for collecting operational metrics from
the embedded Prometheus instance itself. This enables monitoring of Prometheus health,
performance, and integration status within the CloudZero cost allocation pipeline.

Metrics collected include:
- Prometheus server performance: Query latency, storage usage, rule evaluation
- Remote storage integration: CloudZero Agent connectivity and throughput metrics
- Service discovery: Target discovery and scraping success rates
- Resource utilization: Memory usage, goroutine counts, and system metrics

Scrape job features:
- Local target: Scrapes localhost:9090 for embedded Prometheus metrics
- Metric filtering: Includes only CloudZero-relevant Prometheus operational metrics
- Configurable intervals: Balances monitoring granularity with resource usage
- Label preservation: Maintains essential labels for operational correlation

This configuration ensures CloudZero platform visibility into Prometheus integration
health and enables proactive monitoring of the cost allocation data pipeline.
*/}}
{{- define "cloudzero-agent.prometheus.scrapePrometheus" -}}
- job_name: static-prometheus
  scrape_interval: {{ .Values.prometheusConfig.scrapeJobs.prometheus.scrapeInterval }}
  static_configs:
    - targets:
        - localhost:9090
  metric_relabel_configs:
    - source_labels: [__name__]
      regex: "^({{ join "|" (include "cloudzero-agent.defaults" . | fromYaml).prometheusMetrics }})$"
      action: keep
{{- end -}}

{{/*
CloudZero Aggregator Monitoring Scrape Job Configuration Template

Generates Prometheus scrape job configuration for monitoring CloudZero Agent aggregator
components. This enables comprehensive operational monitoring of the cost allocation
data processing pipeline, including metric ingestion, filtering, storage, and transmission.

Monitoring targets:
- Aggregator collector: Prometheus remote_write ingestion performance
- Aggregator shipper: CloudZero platform transmission metrics and status
- Storage subsystem: Database performance, retention, and health metrics
- Processing pipeline: Throughput, error rates, and data quality metrics

Service discovery features:
- Kubernetes endpoint discovery: Automatically discovers aggregator instances
- Namespace scoping: Restricts discovery to CloudZero Agent deployment namespace
- Port filtering: Targets specific aggregator monitoring ports (collector, shipper)
- Label enrichment: Adds Kubernetes metadata for operational correlation

This configuration provides essential operational visibility for:
- Performance monitoring and capacity planning
- Error detection and troubleshooting
- Data quality assurance and cost allocation accuracy
- Integration health monitoring with CloudZero platform
*/}}
{{- define "cloudzero-agent.prometheus.scrapeAggregator" -}}
- job_name: cloudzero-aggregator-job
  scrape_interval: {{ .Values.prometheusConfig.scrapeJobs.prometheus.scrapeInterval }}
  kubernetes_sd_configs:
    - role: endpoints
      kubeconfig_file: ""
      namespaces:
        names:
          - {{ .Release.Namespace }}
  relabel_configs:
    - source_labels: [__meta_kubernetes_service_name]
      action: keep
      regex: {{ include "cloudzero-agent.aggregator.name" . }}
    - source_labels: [__meta_kubernetes_pod_container_port_name]
      action: keep
      regex: port-(shipper|collector)
  metric_relabel_configs:
    - source_labels: [__name__]
      regex: "{{ include "cloudzero-agent.generateMetricNameFilterRegex" .Values }}"
      action: keep
{{- end -}}

{{/*
Kube State Metrics Scrape Job Configuration Template

Generates comprehensive Prometheus scrape job configuration for collecting Kubernetes
cluster state metrics essential for CloudZero cost allocation analysis. Kube State Metrics
provides the foundational data about Kubernetes resources, their configuration, and
relationships required for accurate cost attribution.

Essential metrics collected:
- Node information: Instance types, capacity, availability zones for cost correlation
- Pod specifications: Resource requests, limits, and actual placement for usage analysis
- Workload metadata: Labels, annotations, and ownership for cost allocation grouping
- Resource states: Running, pending, failed states for operational cost tracking

Label processing and enrichment:
- Kubernetes metadata extraction: Service names, namespaces, node assignments
- Label mapping: Preserves application and infrastructure labels for cost grouping
- Namespace attribution: Enables namespace-level cost allocation and chargeback
- Resource hierarchy: Maintains pod-to-node relationships for infrastructure cost mapping

Metric filtering:
- Selective metric inclusion: Only cost-relevant Kubernetes state metrics
- Label filtering: Preserves essential labels while removing noise
- Performance optimization: Reduces data volume while maintaining cost accuracy

This configuration is critical for CloudZero's Kubernetes cost allocation accuracy,
providing the cluster state foundation for all downstream cost analysis and optimization.
*/}}
{{- define "cloudzero-agent.prometheus.scrapeKubeStateMetrics" -}}
# Kube State Metrics Scrape Job
# static-kube-state-metrics
#
# Kube State Metrics provides the CloudZero Agent with information
# regarding the configuration and state of various Kubernetes objects
# (nodes, pods, etc.), including where they are located in the cluster.
- job_name: static-kube-state-metrics
  scrape_interval: {{ .Values.prometheusConfig.scrapeJobs.kubeStateMetrics.scrapeInterval }}

  # Given a Kubernetes resource with a structure like:
  #
  #   apiVersion: v1
  #   kind: Service
  #   metadata:
  #     name: my-service
  #     namespace: my-namespace
  #     labels:
  #       app: my-app
  #       environment: production
  #
  # Kube State Metrics should provide labels such as:
  #
  #   __meta_kubernetes_service_name:               my-name
  #   __meta_kubernetes_namespace:                  my-namespace
  #   __meta_kubernetes_service_label_app:          my-app
  #   __meta_kubernetes_service_label_environment:  production
  #
  # We read these into the CloudZero Agent as:
  #
  #   service: my-name
  #   namespace: my-namespace
  #   app: my-app
  #   environment: production
  relabel_configs:

    # Relabel __meta_kubernetes_service_label_(.+) labels to $1.
    - regex: __meta_kubernetes_service_label_(.+)
      action: labelmap

    # Replace __meta_kubernetes_namespace labels with "namespace"
    - source_labels: [__meta_kubernetes_namespace]
      target_label: namespace

    # Replace __meta_kubernetes_service_name labels with "service"
    - source_labels: [__meta_kubernetes_service_name]
      target_label: service

    # Replace "__meta_kubernetes_pod_node_name" labels to "node"
    - source_labels: [__meta_kubernetes_pod_node_name]
      target_label: node
  # We filter out all but a select few metrics and labels.
  metric_relabel_configs:

    # Metric names to keep.
    - source_labels: [__name__]
      regex: {{ printf "^(%s)$" (join "|" (include "cloudzero-agent.defaults" . | fromYaml).kubeMetrics) }}
      action: keep

    # Metric labels to keep.
    - regex: ^(board_asset_tag|container|created_by_kind|created_by_name|image|instance|name|namespace|node|node_kubernetes_io_instance_type|pod|product_name|provider_id|resource|unit|uid|_.*|label_.*|app.kubernetes.io/*|k8s.*)$
      action: labelkeep

  static_configs:
    - targets:
      - {{ include "cloudzero-agent.kubeStateMetrics.kubeStateMetricsSvcTargetName" . }}
{{- end -}}

{{/*
CloudZero Webhook Monitoring Scrape Job Configuration Template

Generates Prometheus scrape job configuration for monitoring CloudZero Agent webhook
operations and Kubernetes admission control performance. This enables comprehensive
visibility into webhook processing latency, admission request volumes, and integration
health with Kubernetes API servers.

Webhook monitoring metrics:
- Admission request processing: Latency, throughput, and success rates
- Resource type distribution: Admission patterns by Kubernetes resource types
- Error tracking: Failed admissions, parsing errors, and integration failures
- Performance metrics: Memory usage, goroutine counts, and HTTP connection health

Scraping configuration:
- HTTPS endpoint: Secure connection to webhook service using TLS
- Service discovery: Kubernetes endpoint discovery for webhook service instances
- Certificate handling: Bypasses certificate validation for internal webhook certificates
- Metric filtering: Focuses on CloudZero-specific operational and business metrics

This monitoring is essential for:
- Ensuring webhook performance doesn't impact cluster operations
- Tracking cost allocation metadata collection completeness
- Detecting integration issues with Kubernetes API servers
- Monitoring webhook contribution to overall cost allocation accuracy
*/}}
{{- define "cloudzero-agent.prometheus.scrapeWebhookJob" -}}
- job_name: cloudzero-webhook-job
  scheme: https
  tls_config:
    insecure_skip_verify: true

  kubernetes_sd_configs:
    - role: endpoints
      kubeconfig_file: ""

  relabel_configs:
    # Keep __meta_kubernetes_endpoints_name labels.
    - source_labels: [__meta_kubernetes_endpoints_name]
      action: keep
      regex: {{ include "cloudzero-agent.insightsController.server.webhookFullname" . }}-svc

  metric_relabel_configs:
    # Metrics to keep.
    - source_labels: [__name__]
      regex: "^({{ join "|" (include "cloudzero-agent.defaults" . | fromYaml).insightsMetrics }})$"
      action: keep
{{- end -}}

{{/*
Container Advisor (cAdvisor) Scrape Job Configuration Template

Generates Prometheus scrape job configuration for collecting container resource usage
metrics from Kubernetes nodes via cAdvisor. These metrics provide the actual resource
consumption data essential for CloudZero cost allocation accuracy and optimization insights.

Container metrics collected:
- CPU usage: Actual CPU consumption by containers for cost allocation
- Memory utilization: Working set memory usage for memory-based cost attribution
- Network I/O: Container network traffic for network cost correlation
- Storage usage: Container filesystem usage for storage cost analysis

Scraping modes supported:
- Local node scraping: DaemonSet mode scraping only the local node's cAdvisor
- Cluster-wide scraping: Central scraping of all nodes via Kubernetes API proxy
- Authentication: ServiceAccount token-based authentication for secure access
- TLS configuration: Proper certificate handling for secure cAdvisor endpoints

Configuration features:
- Node filtering: Supports both single-node and cluster-wide collection patterns
- Label processing: Enriches metrics with Kubernetes node and container metadata
- Metric filtering: Selects only cost-relevant container resource metrics
- Security: Uses Kubernetes RBAC for secure cAdvisor endpoint access

This configuration is fundamental to CloudZero's container cost allocation,
providing the actual resource usage data needed for accurate cost attribution
and optimization recommendations across Kubernetes workloads.
*/}}
{{- define "cloudzero-agent.prometheus.scrapeCAdvisor" -}}
{{- $scrapeLocal := .scrapeLocalNodeOnly | default false -}}
# cAdvisor Scrape Job cloudzero-nodes-cadvisor
#
# This job scrapes metrics about container resource usage (CPU, memory,
# network, etc.).
- job_name: cloudzero-nodes-cadvisor

  scrape_interval: {{ .root.Values.prometheusConfig.scrapeJobs.cadvisor.scrapeInterval }}
  scheme: https

  # cAdvisor endpoints are protected. In order to access them we need the
  # credentials for the ServiceAccount.
  authorization:
    type: Bearer
    credentials_file: /var/run/secrets/kubernetes.io/serviceaccount/token
  tls_config:
    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    insecure_skip_verify: true

  {{- if $scrapeLocal }}
  # Scrape metrics directly from cAdvisor endpoint.
  metrics_path: /metrics/cadvisor

  # Scrape metrics from cAdvisor
  relabel_configs:

    # Replace "__meta_kubernetes_node_name" labels with "node_name"
    - source_labels: [__meta_kubernetes_node_name]
      target_label: node_name

    # Only scrape metrics for the node we are running on.
    #
    #
    # Note that Prometheus does not handle the regex being a variable. In order
    # to get this to work, we run a sed command in an initContainer to replace
    # '${NODE_NAME}' with the name of the node we are running on. See the agent
    # DaemonSet configuration for details.
    - source_labels: [__meta_kubernetes_node_name]
      regex: ${NODE_NAME}
      action: keep

    # Add port number to __address__ in "__meta_kubernetes_node_address_InternalIP"
    - source_labels: [__meta_kubernetes_node_address_InternalIP]
      target_label: __address__
      replacement: ${1}:10250
  {{- else }}

  # Scrape metrics from cAdvisor.
  relabel_configs:

    # Replace the value of __address__ labels with "kubernetes.default.svc:443"
    - target_label: __address__
      replacement: kubernetes.default.svc:443

    # Replace the value of __metrics_path__ in __meta_kubernetes_node_name with
    # "/api/v1/nodes/$1/proxy/metrics/cadvisor"
    - source_labels: [__meta_kubernetes_node_name]
      target_label: __metrics_path__
      replacement: /api/v1/nodes/$1/proxy/metrics/cadvisor
  {{- end }}

    # Remove "__meta_kubernetes_node_label_" prefix from labels.
    - regex: __meta_kubernetes_node_label_(.+)
      action: labelmap

    # Replace __meta_kubernetes_node_name labels with "node"
    - source_labels: [__meta_kubernetes_node_name]
      target_label: node

  # We only want to keep a select few labels.
  metric_relabel_configs:

    # Labels to keep.
    - action: labelkeep
      regex: {{ printf "^(%s)$" (include "cloudzero-agent.requiredMetricLabels" .root) }}

    # Metrics to keep.
    - source_labels: [__name__]
      regex: {{ printf "^(%s)$" (join "|" (include "cloudzero-agent.defaults" .root | fromYaml).containerMetrics) }}
      action: keep

  kubernetes_sd_configs:
    - role: node
      kubeconfig_file: ""
{{- end -}}
