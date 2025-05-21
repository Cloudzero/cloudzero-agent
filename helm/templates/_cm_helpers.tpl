{{/*
Configuration for the webhook-server Deployment. Configuration is defined in this tpl so that we can roll Deployment pods based on a checksum of these values
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
{{- end }}
filters:
  labels:
  {{- .Values.insightsController.labels | toYaml | nindent 4 }}
  annotations:
  {{- .Values.insightsController.annotations | toYaml | nindent 4 }}
{{- end}}


{{/*
Configuration for the aggregator Deployment. Configuration is defined in this tpl so that we can roll Deployment pods based on a checksum of these values
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
logging:
  level: "{{ .Values.aggregator.logging.level }}"
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
{{- end}}

{{/* Define remote_write configuration for Prometheus */}}
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

{{/* Define static-prometheus scrape job configuration */}}
{{- define "cloudzero-agent.prometheus.scrapePrometheus" -}}
- job_name: static-prometheus
  scrape_interval: {{ .Values.prometheusConfig.scrapeJobs.prometheus.scrapeInterval }}
  static_configs:
    - targets:
        - localhost:9090
  metrics_path: /metrics
  metric_relabel_configs:
    - source_labels: [__name__]
      regex: "^({{ join "|" (include "cloudzero-agent.defaults" . | fromYaml).prometheusMetrics }})$"
      action: keep
{{- end -}}

{{/* Define cloudzero-aggregator-job scrape job configuration */}}
{{- define "cloudzero-agent.prometheus.scrapeAggregator" -}}
- job_name: cloudzero-aggregator-job
  scrape_interval: {{ .Values.prometheusConfig.scrapeJobs.prometheus.scrapeInterval }}
  kubernetes_sd_configs:
    - role: endpoints
      kubeconfig_file: ""
      follow_redirects: true
      enable_http2: true
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

{{/* Define static-kube-state-metrics scrape job configuration */}}
{{- define "cloudzero-agent.prometheus.scrapeKubeStateMetrics" -}}
# Kube State Metrics Scrape Job
# static-kube-state-metrics
#
# Kube State Metrics provides the CloudZero Agent with information
# regarding the configuration and state of various Kubernetes objects
# (nodes, pods, etc.), including where they are located in the cluster.
- job_name: static-kube-state-metrics
  honor_timestamps: true
  track_timestamps_staleness: false
  scrape_interval: {{ .Values.prometheusConfig.scrapeJobs.kubeStateMetrics.scrapeInterval }}
  scrape_timeout: 10s
  scrape_protocols:
  - OpenMetricsText1.0.0
  - OpenMetricsText0.0.1
  - PrometheusText0.0.4
  metrics_path: /metrics
  scheme: http
  enable_compression: true
  follow_redirects: true
  enable_http2: true

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
    - separator: ;
      regex: __meta_kubernetes_service_label_(.+)
      replacement: $1
      action: labelmap

    # Replace __meta_kubernetes_namespace labels with "namespace"
    - source_labels: [__meta_kubernetes_namespace]
      separator: ;
      regex: (.*)
      target_label: namespace
      replacement: $1
      action: replace

    # Replace __meta_kubernetes_service_name labels with "service"
    - source_labels: [__meta_kubernetes_service_name]
      separator: ;
      regex: (.*)
      target_label: service
      replacement: $1
      action: replace

    # Replace "__meta_kubernetes_pod_node_name" labels to "node"
    - source_labels: [__meta_kubernetes_pod_node_name]
      separator: ;
      regex: (.*)
      target_label: node
      replacement: $1
      action: replace
  # We filter out all but a select few metrics and labels.
  metric_relabel_configs:

    # Metric names to keep.
    - source_labels: [__name__]
      regex: {{ printf "^(%s)$" (join "|" (include "cloudzero-agent.defaults" . | fromYaml).kubeMetrics) }}
      action: keep

    # Metric labels to keep.
    - separator: ;
      regex: ^(board_asset_tag|container|created_by_kind|created_by_name|image|instance|name|namespace|node|node_kubernetes_io_instance_type|pod|product_name|provider_id|resource|unit|uid|_.*|label_.*|app.kubernetes.io/*|k8s.*)$
      replacement: $1
      action: labelkeep

  static_configs:
    - targets:
      - {{ include "cloudzero-agent.kubeStateMetrics.kubeStateMetricsSvcTargetName" . }}
{{- end -}}

{{/* Define cloudzero-webhook-job scrape job configuration */}}
{{- define "cloudzero-agent.prometheus.scrapeWebhookJob" -}}
- job_name: cloudzero-webhook-job
  metrics_path: /metrics
  scheme: https
  enable_compression: true
  tls_config:
    insecure_skip_verify: true
  follow_redirects: true
  enable_http2: true
  kubernetes_sd_configs:
    - role: endpoints
      kubeconfig_file: ""
      follow_redirects: true
      enable_http2: true

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

{{/* Define cloudzero-nodes-cAdvisor scrape job configuration */}}
{{- define "cloudzero-agent.prometheus.scrapeCAdvisor" -}}
{{- $scrapeLocal := .scrapeLocalNodeOnly | default false -}}
# cAdvisor Scrape Job cloudzero-nodes-cadvisor
#
# This job scrapes metrics about container resource usage (CPU, memory,
# network, etc.).
- job_name: cloudzero-nodes-cadvisor # container_* metrics
  honor_timestamps: true
  track_timestamps_staleness: false
  scrape_interval: {{ .root.Values.prometheusConfig.scrapeJobs.cadvisor.scrapeInterval }}
  scrape_timeout: 10s
  scrape_protocols:
  - OpenMetricsText1.0.0
  - OpenMetricsText0.0.1
  - PrometheusText0.0.4
  scheme: https
  enable_compression: true

  # cAdvisor endpoints are protected. In order to access them we need the
  # credentials for the ServiceAccount.
  authorization:
    type: Bearer
    credentials_file: /var/run/secrets/kubernetes.io/serviceaccount/token
  tls_config:
    ca_file: /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
    insecure_skip_verify: true

  follow_redirects: true
  enable_http2: true
  {{- if $scrapeLocal }}
  # Scrape metrics directly from cAdvisor endpoint.
  metrics_path: /metrics/cadvisor

  # Scrape metrics from cAdvisor
  relabel_configs:

    # Replace "__meta_kubernetes_node_name" labels with "node_name"
    - source_labels: [__meta_kubernetes_node_name]
      target_label: node_name
      action: replace

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
      action: replace
      target_label: __address__
      replacement: ${1}:10250
  {{- else }}
  metrics_path: /metrics

  # Scrape metrics from cAdvisor.
  relabel_configs:

    # Replace the value of __address__ labels with "kubernetes.default.svc:443"
    - separator: ;
      regex: (.*)
      target_label: __address__
      replacement: kubernetes.default.svc:443
      action: replace

    # Replace the value of __metrics_path__ in __meta_kubernetes_node_name with
    # "/api/v1/nodes/$1/proxy/metrics/cadvisor"
    - source_labels: [__meta_kubernetes_node_name]
      separator: ;
      regex: (.+)
      target_label: __metrics_path__
      replacement: /api/v1/nodes/$1/proxy/metrics/cadvisor
      action: replace
  {{- end }}

    # Remove "__meta_kubernetes_node_label_" prefix from labels.
    - separator: ;
      regex: __meta_kubernetes_node_label_(.+)
      replacement: $1
      action: labelmap

    # Replace __meta_kubernetes_node_name labels with "node"
    - source_labels: [__meta_kubernetes_node_name]
      target_label: node
      action: replace

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
      follow_redirects: true
      enable_http2: true
{{- end -}}
