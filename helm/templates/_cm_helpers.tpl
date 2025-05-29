{{/*
Configuration for the webhook-server Deployment. Configuration is defined in this tpl so that we can roll Deployment pods based on a checksum of these values
*/}}
{{ define "cloudzero-agent.insightsController.configuration" -}}
{{- $values := include "cloudzero-agent.values" . | fromYaml -}}
cloud_account_id: {{ $values.cloudAccountId }}
region: {{ $values.region }}
cluster_name: {{ $values.clusterName }}
destination: {{ include "cloudzero-agent.metricsDestination" . }}
logging:
  level: {{ $values.insightsController.server.logging.level }}
remote_write:
  send_interval: {{ $values.insightsController.server.send_interval }}
  max_bytes_per_send: 500000
  send_timeout: {{ $values.insightsController.server.send_timeout }}
  max_retries: 3
k8s_client:
  timeout: 30s
database:
  retention_time: 24h
  cleanup_interval: 3h
  batch_update_size: 500
api_key_path: {{ include "cloudzero-agent.secretFileFullPath" . }}
{{- $namespace := .Release.Namespace }}
{{- with $values.insightsController }}
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
  {{- $values.insightsController.labels | toYaml | nindent 4 }}
  annotations:
  {{- $values.insightsController.annotations | toYaml | nindent 4 }}
{{- end}}


{{/*
Configuration for the aggregator Deployment. Configuration is defined in this tpl so that we can roll Deployment pods based on a checksum of these values
*/}}
{{ define "cloudzero-agent.aggregator.configuration" -}}
{{- $values := include "cloudzero-agent.values" . | fromYaml -}}
{{- $internalValues := include "cloudzero-agent.internalDefaults" . | fromYaml -}}
cloud_account_id: {{ include "cloudzero-agent.cleanString" $values.cloudAccountId }}
cluster_name: {{ include "cloudzero-agent.cleanString" $values.clusterName }}
region: {{ include "cloudzero-agent.cleanString" $values.region }}

metrics:
  {{- include "cloudzero-agent.generateMetricFilters" (dict "name" "cost" "filters"                 $internalValues.metricFilters.cost.name) | nindent 2 }}
  {{- include "cloudzero-agent.generateMetricFilters" (dict "name" "cost_labels" "filters"          $internalValues.metricFilters.cost.labels) | nindent 2 }}
  {{- include "cloudzero-agent.generateMetricFilters" (dict "name" "observability" "filters"        $internalValues.metricFilters.observability.name) | nindent 2 }}
  {{- include "cloudzero-agent.generateMetricFilters" (dict "name" "observability_labels" "filters" $internalValues.metricFilters.observability.labels) | nindent 2 }}
server:
  mode: http
  port: {{ $values.aggregator.collector.port }}
  profiling: {{ $values.aggregator.profiling }}
logging:
  level: "{{ $values.aggregator.logging.level }}"
database:
  storage_path: {{ $values.aggregator.mountRoot }}/data
  max_records: {{ $values.aggregator.database.maxRecords }}
  cost_max_interval: {{ $values.aggregator.database.costMaxInterval }}
  observability_max_interval: {{ $values.aggregator.database.observabilityMaxInterval }}
  compression_level: {{ $values.aggregator.database.compressionLevel }}
  purge_rules:
    metrics_older_than: {{ $values.aggregator.database.purgeRules.metricsOlderThan }}
    lazy: {{ $values.aggregator.database.purgeRules.lazy }}
    percent: {{ $values.aggregator.database.purgeRules.percent }}
  {{- if $values.aggregator.database.emptyDir.enabled }}
  available_storage: {{ $values.aggregator.database.emptyDir.sizeLimit }}
  {{- end}}
cloudzero:
  api_key_path: {{ include "cloudzero-agent.secretFileFullPath" . }}
  send_interval: {{ $values.aggregator.cloudzero.sendInterval }}
  send_timeout: {{ $values.aggregator.cloudzero.sendTimeout }}
  rotate_interval: {{ $values.aggregator.cloudzero.rotateInterval }}
  host: {{ $values.host }}
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
{{- $values := include "cloudzero-agent.values" . | fromYaml -}}
- job_name: static-prometheus
  scrape_interval: {{ $values.prometheusConfig.scrapeJobs.prometheus.scrapeInterval }}
  static_configs:
    - targets:
        - localhost:9090
  metric_relabel_configs:
    - source_labels: [__name__]
      regex: "^({{ join "|" (include "cloudzero-agent.internalDefaults" . | fromYaml).prometheusMetrics }})$"
      action: keep
{{- end -}}

{{/* Define cloudzero-aggregator-job scrape job configuration */}}
{{- define "cloudzero-agent.prometheus.scrapeAggregator" -}}
{{- $values := include "cloudzero-agent.values" . | fromYaml -}}
- job_name: cloudzero-aggregator-job
  scrape_interval: {{ $values.prometheusConfig.scrapeJobs.prometheus.scrapeInterval }}
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
      regex: "{{ include "cloudzero-agent.generateMetricNameFilterRegex" $values }}"
      action: keep
{{- end -}}

{{/* Define static-kube-state-metrics scrape job configuration */}}
{{- define "cloudzero-agent.prometheus.scrapeKubeStateMetrics" -}}
{{- $values := include "cloudzero-agent.values" . | fromYaml -}}
# Kube State Metrics Scrape Job
# static-kube-state-metrics
#
# Kube State Metrics provides the CloudZero Agent with information
# regarding the configuration and state of various Kubernetes objects
# (nodes, pods, etc.), including where they are located in the cluster.
- job_name: static-kube-state-metrics
  scrape_interval: {{ $values.prometheusConfig.scrapeJobs.kubeStateMetrics.scrapeInterval }}

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
      regex: {{ printf "^(%s)$" (join "|" (include "cloudzero-agent.internalDefaults" . | fromYaml).kubeMetrics) }}
      action: keep

    # Metric labels to keep.
    - regex: ^(board_asset_tag|container|created_by_kind|created_by_name|image|instance|name|namespace|node|node_kubernetes_io_instance_type|pod|product_name|provider_id|resource|unit|uid|_.*|label_.*|app.kubernetes.io/*|k8s.*)$
      action: labelkeep

  static_configs:
    - targets:
      - {{ include "cloudzero-agent.kubeStateMetrics.kubeStateMetricsSvcTargetName" . }}
{{- end -}}

{{/* Define cloudzero-webhook-job scrape job configuration */}}
{{- define "cloudzero-agent.prometheus.scrapeWebhookJob" -}}
{{- $values := include "cloudzero-agent.values" . | fromYaml -}}
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
      regex: "^({{ join "|" (include "cloudzero-agent.internalDefaults" . | fromYaml).insightsMetrics }})$"
      action: keep
{{- end -}}

{{/* Define cloudzero-nodes-cAdvisor scrape job configuration */}}
{{- define "cloudzero-agent.prometheus.scrapeCAdvisor" -}}
{{- $values := include "cloudzero-agent.values" .root | fromYaml -}}
{{- $scrapeLocal := .scrapeLocalNodeOnly | default false -}}
# cAdvisor Scrape Job cloudzero-nodes-cadvisor
#
# This job scrapes metrics about container resource usage (CPU, memory,
# network, etc.).
- job_name: cloudzero-nodes-cadvisor

  scrape_interval: {{ $values.prometheusConfig.scrapeJobs.cadvisor.scrapeInterval }}
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
      regex: {{ printf "^(%s)$" (join "|" (include "cloudzero-agent.internalDefaults" .root | fromYaml).containerMetrics) }}
      action: keep

  kubernetes_sd_configs:
    - role: node
      kubeconfig_file: ""
{{- end -}}
