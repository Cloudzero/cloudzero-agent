{{/*
Internal default values.

You can think of this as similar to `.Values`, but without allowing people
installing the Helm chart to override the values easily.  This is used in places
where we want to reuse values instead of hardcode them, but at the same time the
values shouldn't really be changed.
*/}}
{{- define "cloudzero-agent.internalDefaults" -}}

# -- The following lists of metrics are required for CloudZero to function.
# -- Modifications made to these lists may cause issues with the processing of cluster data
kubeMetrics:
  - kube_node_info
  - kube_node_status_capacity
  - kube_pod_container_resource_limits
  - kube_pod_container_resource_requests
  - kube_pod_labels
  - kube_pod_info
containerMetrics:
  - container_cpu_usage_seconds_total
  - container_memory_working_set_bytes
  - container_network_receive_bytes_total
  - container_network_transmit_bytes_total
insightsMetrics:
  - go_memstats_alloc_bytes
  - go_memstats_heap_alloc_bytes
  - go_memstats_heap_idle_bytes
  - go_memstats_heap_inuse_bytes
  - go_memstats_heap_objects
  - go_memstats_last_gc_time_seconds
  - go_memstats_alloc_bytes
  - go_memstats_stack_inuse_bytes
  - go_goroutines
  - process_cpu_seconds_total
  - process_max_fds
  - process_open_fds
  - process_resident_memory_bytes
  - process_start_time_seconds
  - process_virtual_memory_bytes
  - process_virtual_memory_max_bytes
  - remote_write_timeseries_total
  - remote_write_response_codes_total
  - remote_write_payload_size_bytes
  - remote_write_failures_total
  - remote_write_records_processed_total
  - remote_write_db_failures_total
  - http_requests_total
  - storage_write_failure_total
  - czo_webhook_types_total
  - czo_storage_types_total
  - czo_ingress_types_total
  - czo_gateway_types_total
prometheusMetrics:
  - go_memstats_alloc_bytes
  - go_memstats_heap_alloc_bytes
  - go_memstats_heap_idle_bytes
  - go_memstats_heap_inuse_bytes
  - go_memstats_heap_objects
  - go_memstats_last_gc_time_seconds
  - go_memstats_alloc_bytes
  - go_memstats_stack_inuse_bytes
  - go_goroutines
  - process_cpu_seconds_total
  - process_max_fds
  - process_open_fds
  - process_resident_memory_bytes
  - process_start_time_seconds
  - process_virtual_memory_bytes
  - process_virtual_memory_max_bytes
  - prometheus_agent_corruptions_total
  - prometheus_api_remote_read_queries
  - prometheus_http_requests_total
  - prometheus_notifications_alertmanagers_discovered
  - prometheus_notifications_dropped_total
  - prometheus_remote_storage_bytes_total
  - prometheus_remote_storage_histograms_failed_total
  - prometheus_remote_storage_histograms_total
  - prometheus_remote_storage_metadata_bytes_total
  - prometheus_remote_storage_metadata_failed_total
  - prometheus_remote_storage_metadata_retried_total
  - prometheus_remote_storage_metadata_total
  - prometheus_remote_storage_samples_dropped_total
  - prometheus_remote_storage_samples_failed_total
  - prometheus_remote_storage_samples_in_total
  - prometheus_remote_storage_samples_total
  - prometheus_remote_storage_shard_capacity
  - prometheus_remote_storage_shards
  - prometheus_remote_storage_shards_desired
  - prometheus_remote_storage_shards_max
  - prometheus_remote_storage_shards_min
  - prometheus_sd_azure_cache_hit_total
  - prometheus_sd_azure_failures_total
  - prometheus_sd_discovered_targets
  - prometheus_sd_dns_lookup_failures_total
  - prometheus_sd_failed_configs
  - prometheus_sd_file_read_errors_total
  - prometheus_sd_file_scan_duration_seconds
  - prometheus_sd_file_watcher_errors_total
  - prometheus_sd_http_failures_total
  - prometheus_sd_kubernetes_events_total
  - prometheus_sd_kubernetes_http_request_duration_seconds
  - prometheus_sd_kubernetes_http_request_total
  - prometheus_sd_kubernetes_workqueue_depth
  - prometheus_sd_kubernetes_workqueue_items_total
  - prometheus_sd_kubernetes_workqueue_latency_seconds
  - prometheus_sd_kubernetes_workqueue_longest_running_processor_seconds
  - prometheus_sd_kubernetes_workqueue_unfinished_work_seconds
  - prometheus_sd_kubernetes_workqueue_work_duration_seconds
  - prometheus_sd_received_updates_total
  - prometheus_sd_updates_delayed_total
  - prometheus_sd_updates_total
  - prometheus_target_scrape_pool_reloads_failed_total
  - prometheus_target_scrape_pool_reloads_total
  - prometheus_target_scrape_pool_sync_total
  - prometheus_target_scrape_pools_failed_total
  - prometheus_target_scrape_pools_total
  - prometheus_target_sync_failed_total
  - prometheus_target_sync_length_seconds

# metricFilters is used to determine which metrics are sent to CloudZero, as
# well as whether they are considered to be cost metrics or observability
# metrics.
#
# There are two sets of filters for each type (cost/observability): name and
# labels. The name filters are applied to the name to determine whether the
# metric should be included in the relevant output. If it is to be included, the
# relevant labels filters are applied to each label to determine whether the
# label should be included.
#
# In the event that there are no filters, the subject is always assumed to
# match.
#
# Note that for each match type (exact, prefix, suffix, contains, regex) there
# is an "additional..." property. This is to allow you to supply supplemental
# filters without clobbering the defaults. In general, the "additional..."
# properties should be used in your overrides file, and the unprefixed versions
# should be left alone.
metricFilters:
  cost:
    name:
      exact:
        - container_cpu_usage_seconds_total
        - container_memory_working_set_bytes
        - container_network_receive_bytes_total
        - container_network_transmit_bytes_total
        - kube_node_info
        - kube_node_status_capacity
        - kube_pod_container_resource_limits
        - kube_pod_container_resource_requests
        - kube_pod_labels
        - kube_pod_info
      prefix:
        - "cloudzero_"
      suffix: []
      contains: []
      regex: []
    labels:
      exact:
        - board_asset_tag
        - container
        - created_by_kind
        - created_by_name
        - image
        - instance
        - name
        - namespace
        - node
        - node_kubernetes_io_instance_type
        - pod
        - product_name
        - provider_id
        - resource
        - unit
        - uid
      prefix:
        - "_"
        - "label_"
        - "app.kubernetes.io/"
        - "k8s."
      suffix: []
      contains: []
      regex: []

  observability:
    name:
      exact:
        - go_gc_duration_seconds
        - go_gc_duration_seconds_count
        - go_gc_duration_seconds_sum
        - go_gc_gogc_percent
        - go_gc_gomemlimit_bytes
        - go_goroutines
        - go_memstats_alloc_bytes
        - go_memstats_heap_alloc_bytes
        - go_memstats_heap_idle_bytes
        - go_memstats_heap_inuse_bytes
        - go_memstats_heap_objects
        - go_memstats_last_gc_time_seconds
        - go_memstats_stack_inuse_bytes
        - go_threads
        - http_request_duration_seconds_bucket
        - http_request_duration_seconds_count
        - http_request_duration_seconds_sum
        - http_requests_total
        - process_cpu_seconds_total
        - process_max_fds
        - process_open_fds
        - process_resident_memory_bytes
        - process_start_time_seconds
        - process_virtual_memory_bytes
        - process_virtual_memory_max_bytes
        - prometheus_agent_corruptions_total
        - prometheus_api_remote_read_queries
        - prometheus_http_requests_total
        - prometheus_notifications_alertmanagers_discovered
        - prometheus_notifications_dropped_total
        - prometheus_remote_storage_bytes_total
        - prometheus_remote_storage_exemplars_in_total
        - prometheus_remote_storage_histograms_failed_total
        - prometheus_remote_storage_histograms_in_total
        - prometheus_remote_storage_histograms_total
        - prometheus_remote_storage_metadata_bytes_total
        - prometheus_remote_storage_metadata_failed_total
        - prometheus_remote_storage_metadata_retried_total
        - prometheus_remote_storage_metadata_total
        - prometheus_remote_storage_samples_dropped_total
        - prometheus_remote_storage_samples_failed_total
        - prometheus_remote_storage_samples_in_total
        - prometheus_remote_storage_samples_total
        - prometheus_remote_storage_shard_capacity
        - prometheus_remote_storage_shards
        - prometheus_remote_storage_shards_desired
        - prometheus_remote_storage_shards_max
        - prometheus_remote_storage_shards_min
        - prometheus_remote_storage_string_interner_zero_reference_releases_total
        - prometheus_sd_azure_cache_hit_total
        - prometheus_sd_azure_failures_total
        - prometheus_sd_discovered_targets
        - prometheus_sd_dns_lookup_failures_total
        - prometheus_sd_failed_configs
        - prometheus_sd_file_read_errors_total
        - prometheus_sd_file_scan_duration_seconds
        - prometheus_sd_file_watcher_errors_total
        - prometheus_sd_http_failures_total
        - prometheus_sd_kubernetes_events_total
        - prometheus_sd_kubernetes_http_request_duration_seconds
        - prometheus_sd_kubernetes_http_request_total
        - prometheus_sd_kubernetes_workqueue_depth
        - prometheus_sd_kubernetes_workqueue_items_total
        - prometheus_sd_kubernetes_workqueue_latency_seconds
        - prometheus_sd_kubernetes_workqueue_longest_running_processor_seconds
        - prometheus_sd_kubernetes_workqueue_unfinished_work_seconds
        - prometheus_sd_kubernetes_workqueue_work_duration_seconds
        - prometheus_sd_received_updates_total
        - prometheus_sd_updates_delayed_total
        - prometheus_sd_updates_total
        - prometheus_target_scrape_pool_reloads_failed_total
        - prometheus_target_scrape_pool_reloads_total
        - prometheus_target_scrape_pool_sync_total
        - prometheus_target_scrape_pools_failed_total
        - prometheus_target_scrape_pools_total
        - prometheus_target_sync_failed_total
        - prometheus_target_sync_length_seconds
        - promhttp_metric_handler_requests_in_flight
        - promhttp_metric_handler_requests_total
        - remote_write_db_failures_total
        - remote_write_failures_total
        - remote_write_payload_size_bytes
        - remote_write_records_processed_total
        - remote_write_response_codes_total
        - remote_write_timeseries_total
        - storage_write_failure_total
        # webhook
        - czo_webhook_types_total
        - czo_storage_types_total
        - czo_ingress_types_total
        - czo_gateway_types_total
        # shipper
        - function_execution_seconds
        - shipper_shutdown_total
        - shipper_new_files_error_total
        - shipper_new_files_processing_current
        - shipper_handle_request_file_count
        - shipper_handle_request_success_total
        - shipper_presigned_url_error_total
        - shipper_replay_request_total
        - shipper_replay_request_current
        - shipper_replay_request_file_count
        - shipper_replay_request_error_total
        - shipper_replay_request_abandon_files_total
        - shipper_replay_request_abandon_files_error_total
        - shipper_disk_total_size_bytes
        - shipper_current_disk_usage_bytes
        - shipper_current_disk_usage_percentage
        - shipper_current_disk_unsent_file
        - shipper_current_disk_sent_file
        - shipper_disk_replay_request_current
        - shipper_disk_cleanup_failure_total
        - shipper_disk_cleanup_success_total
        - shipper_disk_cleanup_percentage
      prefix:
        - czo_
      suffix: []
      contains: []
      regex: []
    labels:
      exact: []
      prefix: []
      suffix: []
      contains: []
      regex: []
{{- end -}}


{{- define "cloudzero-agent._getValues" -}}
# -- CloudZero host to send metrics to.
host: api.cloudzero.com
# -- Account ID of the account the cluster is running in. This must be a string - even if it is a number in your system.
cloudAccountId: null
# -- Name of the clusters.
clusterName: null
# -- Region the cluster is running in.
region: null

# -- CloudZero API key. Required if existingSecretName is null.
apiKey: null
# -- If set, the agent will use the API key in this Secret to authenticate with CloudZero.
existingSecretName: null

# Agent largely contains top-level settings which are often shared by multiple
# components within this chart, or used as defaults in case values are not
# explicitly set per-component.
defaults:
  # Federated mode deploys the agent as a DaemonSet, with each node in the
  # cluster running its own metrics collector.
  federation:
    # Whether to enable federated mode.
    enabled: false

  # The default image settings which will be fallen back on for all components.
  #
  # Note that all image values (including repository, tag, etc.) are valid here,
  # though for the most part they are overridden by the component-specific
  # settings anyways.
  image:
    pullPolicy: IfNotPresent
    pullSecrets:
  # If set, these DNS settings will be attached to resources which support it.
  dns:
    # DNS policy to use on all pods.
    #
    # Valid values include:
    #
    # - "Default"
    # - "ClusterFirst"
    # - "ClusterFirstWithHostNet"
    # - "None"
    #
    # Somewhat counterintuitively, "Default" is not actually the default,
    # "ClusterFirst" is.
    #
    # For details, see the Kubernetes documentation:
    # https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-s-dns-policy
    #
    # Note that if you set this, you'll likely also want to set kubeStateMetrics.dnsPolicy
    # to the same value.
    policy:
    # DNS configuration to use on all pods.
    #
    # There are currently three properties which can be specified: nameservers,
    # searches, and options.
    #
    # For details, see the Kubernetes documentation:
    # https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#pod-dns-config
    #
    # Note that if you set this, you'll likely also want to set kubeStateMetrics.dnsConfig
    # to the same value.
    config: {}
  # Labels to be added to all resources.
  #
  # Labels are organized as key/value pairs. For example, if you wanted to set a
  # my.org/team label to the value "superstars":
  #
  #   labels:
  #     my.org/team: superstars
  #
  # Note that this chart will unconditionally add the following labels:
  #
  #  - app.kubernetes.io/version
  #  - helm.sh/chart
  #  - app.kubernetes.io/managed-by
  #  - app.kubernetes.io/part-of
  #
  # Additionally, certain components will add additional labels. Any labels
  # specified here will be *in addition* to the labels added automatically, not
  # instead of them.
  #
  # For more information, see the Kubernetes documentation:
  # https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/
  #
  # You may also be interested in this list of well-known labels:
  # https://kubernetes.io/docs/reference/labels-annotations-taints/
  #
  # Note that if you set this, you'll likely also want to set
  # kubeStateMetrics.customLabels.
  labels: {}
  # Annotations to be added to all resources.
  #
  # Similar to labels, annotations are organized as key/value pairs, and
  # annotations specified here will be merged into any annotations added
  # automatically by the chart.
  #
  # For more information, see the Kubernetes documentation:
  # https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/
  #
  # Note that if you set this, you'll likely also want to set
  # kubeStateMetrics.annotations.
  annotations: {}
  # Affinity settings to be added to all resources.
  #
  # Affinity settings are used to control the scheduling of pods. For more
  # information, see the Kubernetes documentation:
  # https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity
  #
  # Note that if you set this, you'll likely also want to set
  # kubeStateMetrics.affinity.
  affinity: {}
  # Tolerations to be added to all resources.
  #
  # Tolerations are used to control the scheduling of pods. For more
  # information, see the Kubernetes documentation:
  # https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/
  #
  # Note that if you set this, you'll likely also want to set
  # kubeStateMetrics.tolerations.
  tolerations: []
  # Node Selector to be added to all resources.
  #
  # Node Selector is used to control the scheduling of pods. For more
  # information, see the Kubernetes documentation:
  # https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#nodeselector
  #
  # Note that if you set this, you'll likely also want to set
  # kubeStateMetrics.nodeSelector.
  nodeSelector: {}
  # Pod Disruption Budget to be added to all resources.
  #
  # For information about PDBs, see the Kubernetes documentation:
  # https://kubernetes.io/docs/concepts/workloads/pods/disruptions/
  #
  # Note that if you set this, you'll likely also want to set
  # kubeStateMetrics.podDisruptionBudget.
  podDisruptionBudget:
    minAvailable: 1
    maxUnavailable:
  # If set, this priority class name will be used for all deployments and jobs.
  #
  # Note that, if used, you will need to create the PriorityClass resource
  # yourself; this chart is only capable of referencing an existing priority
  # class, not creating one from whole cloth.
  #
  # For more information, see the Kubernetes documentation:
  # https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/
  #
  # Note that if you set this, you'll likely also want to set
  # kubeStateMetrics.priorityClassName.
  priorityClassName:

# Component-specific configuration settings.
components:
  # The agent here refers to the CloudZero Agent, which is the component that
  # collects metrics from the cluster and sends them to the aggregator.
  agent:
    # This is the image which contains most of the code that makes this chart
    # work. Since 1.1, CloudZero uses a single container image, with multiple
    # executables, to provide CloudZero functionality.
    image:
      repository: ghcr.io/cloudzero/cloudzero-agent/cloudzero-agent
      tag: 1.1.2  # <- Software release corresponding to this chart version.
    podDisruptionBudget:
      minAvailable:
      maxUnavailable:

  # kubectl contains details about where to find the kubectl image.  This chart
  # uses the kubectl image as part of the job to initialize certificates.
  kubectl:
    image:
      repository: docker.io/bitnami/kubectl
      tag: "1.32.0"

  # prometheus contains details about where to find the Prometheus image.
  # Prometheus is critical to the functionality of this chart, and is used to
  # scrape metrics.
  prometheus:
    image:
      repository: quay.io/prometheus/prometheus
      tag:  # This will fall back on .Chart.AppVersion if not set.

  # prometheusReloader contains details about where to find the Prometheus
  # reloader image.
  #
  # prometheus-config-reloader will watch the Prometheus configuration for
  # changes and restart the Prometheus pod as necessary.
  prometheusReloader:
    image:
      repository: quay.io/prometheus-operator/prometheus-config-reloader
      tag: "v0.82.0"

  # Settings for the aggregator component. This is the piece which accepts
  # metrics from the agent, webhook, etc., and sends them to the CloudZero API
  # after some processing.
  aggregator:
    replicas: 3
    podDisruptionBudget:
      minAvailable:
      maxUnavailable:
    tolerations: []

  # Settings for the webhook server.
  webhookServer:
    replicas: 3
    podDisruptionBudget:
      minAvailable:
      maxUnavailable:

#######################################################################################
#######################################################################################
####                                                                               ####
####  Values below this point are not considered API stable. Use at your own risk. ####
####  If you do require them for some reason, please let us know so we can work on ####
####  covering your use case in the stable section.                                ####
####                                                                               ####
#######################################################################################
#######################################################################################

prometheusConfig:
  configMapNameOverride: ""
  configMapAnnotations: {}
  configOverride: ""
  globalScrapeInterval: 60s
  scrapeJobs:
    # -- Enables the kube-state-metrics scrape job.
    kubeStateMetrics:
      enabled: true
      # Scrape interval for kubeStateMetrics job
      scrapeInterval: 60s
    # -- Enables the cadvisor scrape job.
    cadvisor:
      enabled: true
      # Scrape interval for nodesCadvisor job
      scrapeInterval: 60s
    # -- Enables the prometheus scrape job.
    prometheus:
      enabled: true
      # Scrape interval for prometheus job
      scrapeInterval: 120s
    aggregator:
      enabled: true
      # Scrape interval for aggregator job
      scrapeInterval: 120s
    # -- Any items added to this list will be added to the Prometheus scrape configuration.
    additionalScrapeJobs: []

# General server settings that apply to both the prometheus agent server and the webhook server
serverConfig:
  # -- The agent will use this file path on the container filesystem to get the CZ API key.
  containerSecretFilePath: /etc/config/secrets/
  # -- The agent will look for a file with this name to get the CZ API key.
  containerSecretFileName: value

# -- The following settings are for the init-backfill-job, which is used to backfill data from the cluster to CloudZero.
initBackfillJob:
  annotations: {}
  tolerations: []
  # -- By default, all image settings use those set in insightsController.server. Optionally use the below to override. This should not be common.
  # imagePullSecrets: []
  image:
    repository:
    tag:
    digest:
    pullPolicy:
  enabled: true

# -- This is a deprecated field that is replaced by initBackfillJob. However, the fields are identical, and initScrapeJob can still be used to configure the backFill/scrape Job.
# initScrapeJob:
# -- By default, all image settings use those set in insightsController.server. Optionally use the below to override. This should not be common.
# imagePullSecrets: []
# image:
#   repository:
#   tag:
#   pullPolicy:

initCertJob:
  enabled: true
  # -- Defaults to the same setting as the insightsController.server if set, otherwise left empty.
  # imagePullSecrets: []
  annotations: {}
  tolerations: []
  image:
    repository:
    pullPolicy:
    digest:
    tag:
  rbac:
    create: true
    serviceAccountName: ""
    clusterRoleName: ""
    clusterRoleBindingName: ""

  # -- Overriding static scrape target address for an existing KSM.
  # -- Set to service <service-name>.<namespace>.svc.cluster.local:port if built-in is disabled (enable=false above)
  # targetOverride: kube-state-metrics.monitors.svc.cluster.local:8080
  # -- If targetOverride is set and kubeStateMetrics.enabled is true, it is likely that fullnameOverride below must be set as well.
  # -- This should not be a common configuration
  # fullnameOverride: "kube-state-metrics"

# -- Annotations to be added to the Secret, if the chart is configured to create one
secretAnnotations: {}
imagePullSecrets: []

# environment validator image allows for CI to use a different image in testing
validator:
  serviceEndpoints:
    kubeStateMetrics:
  # -- Flag to skip validator failure if unable to connect to the CloudZero API.
  name: env-validator
  image:
    repository:
    tag:
    digest:
    pullPolicy:
    pullSecrets:

server:
  name: server
  # Container image configuration for the server.
  #
  # Deprecated. Please use components.agent.image instead.
  image:
    repository:
    tag:
    digest:
    pullPolicy:
  # Node selector configuration for the server pods.
  #
  # See the Kubernetes documentation for details:
  # https://kubernetes.io/docs/tasks/configure-pod-container/assign-pods-nodes/
  nodeSelector: {}
  # Resource requirements and limits for the server.
  #
  # For details, see the Kubernetes documentation on resource management:
  # https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
  resources:
    requests:
      memory: 512Mi
      cpu: 250m
    limits:
      memory: 1024Mi
  # Annotations to add to the server Deployment.
  deploymentAnnotations: {}
  # Annotations to add to the server pods.
  podAnnotations: {}
  # Whether the server is running in agent mode.
  agentMode: true
  # Command-line arguments to pass to the server.
  args:
    - --config.file=/etc/config/prometheus/configmaps/prometheus.yml
    - --web.enable-lifecycle
    - --web.console.libraries=/etc/prometheus/console_libraries
    - --web.console.templates=/etc/prometheus/consoles
  # Configuration for persistent storage.
  persistentVolume:
    existingClaim: ""
    enabled: false
    mountPath: /data
    subPath: ""
    storageClass: ""
    size: 8Gi
    accessModes:
      - ReadWriteOnce
    # Annotations to add to the PersistentVolumeClaim.
    annotations: {}
  # --Limit the size to 8Gi to lower impact on the cluster, and to provide a reasonable backup for the WAL
  emptyDir:
    sizeLimit: 8Gi
  # Affinity rules
  #
  # See the Kubernetes documentation for details:
  # https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity
  affinity: {}
  # Tolerations configuration for the aggregator pods.
  #
  # See the Kubernetes documentation for details:
  # https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
  tolerations: []
  # Topology spread constraints for pod scheduling
  topologySpreadConstraints: []
  # Termination grace period in seconds
  terminationGracePeriodSeconds: 300
  # Readiness probe configuration
  readinessProbe:
    initialDelaySeconds: 30
    periodSeconds: 5
    timeoutSeconds: 4
    failureThreshold: 3
    successThreshold: 1
  # Liveness probe configuration
  livenessProbe:
    initialDelaySeconds: 30
    periodSeconds: 15
    timeoutSeconds: 10
    failureThreshold: 3
    successThreshold: 1

# Configuration for the webhook server (née Insights Controller), which collects
# and processes Kubernetes resource metadata for cost attribution and analysis.
insightsController:
  # Whether to enable the insights controller.
  #
  # It is strongly recommended that this feature be enabled as it provides
  # important functionality.
  enabled: true
  # Configuration for collecting labels from Kubernetes resources.
  labels:
    # Whether to enable collection of specific labels for cost attribution
    # dimensions.
    #
    # It is strongly recommended that this feature be enabled as it provides
    # important functionality.
    enabled: true
    # List of Go-style regular expressions used to filter desired labels.
    #
    # Caution: The CloudZero system has a limit of 300 labels and annotations,
    # so it is advisable to provide a specific list of required labels rather
    # than a wildcard.
    patterns:
      - "app.kubernetes.io/component"
    # Specify which resources to collect labels from.
    resources:
      # Whether to collect labels from CronJobs.
      cronjobs: false
      # Whether to collect labels from DaemonSets.
      daemonsets: false
      # Whether to collect labels from Deployments.
      deployments: false
      # Whether to collect labels from Jobs.
      jobs: false
      # Whether to collect labels from Namespaces.
      namespaces: true
      # Whether to collect labels from Nodes.
      nodes: false
      # Whether to collect labels from Pods.
      pods: true
      # Whether to collect labels from StatefulSets.
      statefulsets: false
  # Configuration for collecting annotations from Kubernetes resources.
  annotations:
    # Whether to enable collection of annotations for cost attribution dimensions.
    enabled: false
    # List of Go-style regular expressions used to filter desired annotations.
    # Caution: The CloudZero system has a limit of 300 labels and annotations,
    # so it is advisable to provide a specific list of required annotations rather than a wildcard.
    patterns:
      - ".*"  # Use regex patterns to specify which annotations to collect. ".*" collects all annotations.
    # Specify which resources to collect annotations from.
    resources:
      # Whether to collect annotations from CronJobs.
      cronjobs: false
      # Whether to collect annotations from DaemonSets.
      daemonsets: false
      # Whether to collect annotations from Deployments.
      deployments: false
      # Whether to collect annotations from Jobs.
      jobs: false
      # Whether to collect annotations from Namespaces.
      namespaces: true
      # Whether to collect annotations from Nodes.
      nodes: false
      # Whether to collect annotations from Pods.
      pods: true
      # Whether to collect annotations from StatefulSets.
      statefulsets: false
  # Configuration for TLS certificates used by the insights controller.
  tls:
    # Whether to enable TLS certificate management.
    enabled: true
    # TLS certificate in PEM format. If empty, a certificate will be generated.
    crt: ""
    # TLS private key in PEM format. If empty, a key will be generated.
    key: ""
    # Configuration for the TLS certificate Secret.
    secret:
      # Whether to create a Secret to store the TLS certificate and key.
      create: true
      # Name of the Secret to create. If empty, a name will be generated.
      name: ""
    # Path where the TLS certificate and key will be mounted in the container.
    mountPath: /etc/certs
    # CA bundle for validating admission webhook requests. If empty, the default
    # self-signed certificate will be used.
    caBundle: ""
    # Whether to use cert-manager for certificate management. If disabled, a
    # self-signed certificate will be used.
    useCertManager: false
  # Configuration for the webhook server component.
  server:
    # Name of the webhook server component.
    name: webhook-server
    # Number of replicas to run for the webhook server.
    replicaCount:
    # Container image configuration for the webhook server.
    #
    # Deprecated. Please use components.webhookServer.image instead.
    image:
      repository:
      tag:
      pullPolicy:
    # Port that the webhook server listens on.
    port: 8443
    # Timeout for reading HTTP requests.
    #
    # This is formatted as a Go duration string; see
    # https://pkg.go.dev/time#ParseDuration for details.
    read_timeout: 10s
    # Timeout for writing HTTP responses.
    #
    # This is formatted as a Go duration string; see
    # https://pkg.go.dev/time#ParseDuration for details.
    write_timeout: 10s
    # Timeout for sending data to clients.
    #
    # This is formatted as a Go duration string; see
    # https://pkg.go.dev/time#ParseDuration for details.
    send_timeout: 1m
    # Interval between sending data to clients.
    #
    # This is formatted as a Go duration string; see
    # https://pkg.go.dev/time#ParseDuration for details.
    send_interval: 1m
    # Timeout for idle connections.
    #
    # This is formatted as a Go duration string; see
    # https://pkg.go.dev/time#ParseDuration for details.
    idle_timeout: 120s
    # Configuration for logging in the webhook server.
    logging:
      # Logging level for the webhook server.
      level: info
    # Configuration for the health check endpoint.
    healthCheck:
      # Whether to enable the health check endpoint.
      enabled: true
      # Path for the health check endpoint.
      path: /healthz
      # Port for the health check endpoint.
      port: 8443
      # Initial delay before starting health checks.
      initialDelaySeconds: 15
      # Interval between health checks.
      periodSeconds: 20
      # Timeout for health check requests.
      timeoutSeconds: 3
      # Number of successful checks required to mark as healthy.
      successThreshold: 1
      # Number of failed checks before marking as unhealthy.
      failureThreshold: 5
    # Node selector configuration for the webhook server pods.
    #
    # See the Kubernetes documentation for details:
    # https://kubernetes.io/docs/tasks/configure-pod-container/assign-pods-nodes/
    nodeSelector: {}
    # Tolerations configuration for the webhook server pods.
    #
    # See the Kubernetes documentation for details:
    # https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
    tolerations: []
    # Affinity rules for the webhook server pods.
    #
    # See the Kubernetes documentation for details:
    # https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity
    affinity: {}
    # Annotations to add to the webhook server Deployment.
    deploymentAnnotations: {}
    # Annotations to add to the webhook server pods.
    podAnnotations: {}
  # Additional volume mounts to add to the insights controller pods.
  volumeMounts: []
  # Additional volumes to add to the insights controller pods.
  volumes: []
  # Resource requirements and limits for the insights controller.
  resources: {}
  # Annotations to add to the insights controller pods.
  podAnnotations: {}
  # Labels to add to the insights controller pods.
  podLabels: {}
  # Configuration for the insights controller Service.
  service:
    # Port that the insights controller Service listens on.
    port: 443
  # Configuration for the validating admission webhook.
  webhooks:
    # Annotations to add to the validating admission webhook.
    annotations: {}
    # Namespace selector for the validating admission webhook.
    namespaceSelector: {}
    # Path for the validating admission webhook.
    path: /validate
    caInjection:
  configurationMountPath:
  ConfigMapNameOverride:

# Configuration for the service account in the chart.
serviceAccount:
  # Whether to create the service account.
  create: true
  # Name of the service account to create.
  #
  # If not set, one will be generated automatically.
  name: ""
  # Annotations to add to the service account.
  #
  # For more information, see the Kubernetes documentation:
  # https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/
  annotations: {}

# Configuration for the RBAC resources in the chart (e.g., ClusterRole,
# ClusterRoleBinding, Role, RoleBinding, etc.).
rbac:
  # Whether to create the RBAC resources.
  create: true

# These labels are added to all resources in the chart.
#
# Deprecated. Please use defaults.labels instead.
commonMetaLabels: {}

# Configuration for the ConfigMap reloader component, which watches for changes in
# ConfigMaps and triggers a reload of the affected components.
configmapReload:
  # Configuration specific to the Prometheus ConfigMap reloader.
  prometheus:
    # Whether to enable the Prometheus ConfigMap reloader.
    enabled: true
    # Container image configuration for the Prometheus ConfigMap reloader.
    image:
      repository:
      tag:
      digest:
      pullPolicy:
    # Resource requirements and limits for the Prometheus ConfigMap reloader.
    #
    # For details, see the Kubernetes documentation on resource management:
    # https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
    resources: {}

# The aggregator provides an intermediary between the CloudZero Agent and the
# CloudZero API. It is composed of two applications, the collector and the
# shipper.
#
# The collector application provides an endpoint for the CloudZero Agent to
# write metrics to. It filters out any unwanted metrics as it receives them,
# aggregates the wanted metrics, and stores them in a compressed format on disk
# until they are ready to be uploaded to the CloudZero servers. Once the
# collector has aggregated sufficient metrics (or a given amount of time has
# elapsed) the data is sent to the shipper.
#
# The shipper will process the completed metrics files and push them to the
# remote server. It will also handle any requests from the server to re-send any
# missing or incomplete data, ensuring that there is no data loss in the event
# of any loss of communication with the CloudZero API, even when a
# misconfiguration (such as an incorrect API key) prevents it.
aggregator:
  # Configuration for logging behavior of the aggregator components.
  logging:
    # Logging level that will be posted to stdout.
    # Valid values are: 'debug', 'info', 'warn', 'error'
    level: info
  # Top-level directory containing CloudZero data. There will be subdirectories
  # for configuration (the mounted ConfigMap) and the API key (typically a
  # mounted Secret), and data to be uploaded to CloudZero, specifically metrics.
  # This value is really only visible internally in the container, so you
  # shouldn't generally need to change it.
  #
  # Set `aggregator.database.purgeRules` to control the cleanup behavior of this
  # directory.
  mountRoot: /cloudzero
  # Whether to enable the profiling endpoint (/debug/pprof/). This should
  # generally be disabled in production.
  profiling: false
  # Container image configuration for the aggregator components.
  #
  # Deprecated. Please use components.aggregator.image instead.
  image:
    repository:
    tag:
    digest:
    pullPolicy:
  cloudzero:
    # Interval between attempts to ship metrics to the remote endpoint.
    #
    # This is formatted as a Go duration string; see
    # https://pkg.go.dev/time#ParseDuration for details.
    sendInterval: 1m
    # Max time the aggregator will spend attempting to ship metrics to the remote endpoint.
    #
    # This is formatted as a Go duration string; see
    # https://pkg.go.dev/time#ParseDuration for details.
    sendTimeout: 30s
    # Interval at which the aggregator will rotate its internal data structures.
    #
    # This is formatted as a Go duration string; see
    # https://pkg.go.dev/time#ParseDuration for details.
    rotateInterval: 30m
  # Configuration for the aggregator's database and storage behavior.
  database:
    # Max number of records per file. Use this to adjust file sizes uploaded to
    # the server. The default value should generally be left unchanged.
    maxRecords: 1500000
    # Max interval to flush a cost metrics file. This is mostly useful for
    # smaller clusters with little activity.
    costMaxInterval: 10m
    # Max interval to flush an observability metrics file. This is mostly useful
    # for smaller clusters with little activity.
    observabilityMaxInterval: 30m
    # Compression level to use when compressing metrics files on-disk.
    #
    # Valid value range from 0-11, with higher values yielding improved
    # compression ratios at the expense of speed and memory usage.
    #
    # Read more about brotli compression here:
    # https://github.com/google/brotli/blob/master/c/tools/brotli.md#options
    compressionLevel: 8
    # The rules that the application will follow in respect to cleaning up old
    # files that have been uploaded to the CloudZero platform.
    #
    # Generally, the defaults will be okay for the majority of use cases. But,
    # the options are here for more advanced users to optimize disk usage. For
    # example, the default case is to keep uploaded files around for 90 days, as
    # this falls in line with most customer's data tolerance policies. But, if
    # deployed on a more active and/or larger cluster, this value can be lowered
    # to keep disk usage lower with the tradeoff of less data-retention.
    # Regardless of what you define here if there is disk pressure detected,
    # files will be deleted (oldest first) to free space.
    purgeRules:
      # How long to keep uploaded files. This option can be useful to optimize
      # the storage required by the collector/shipper architecture on your
      # nodes.
      #
      # `2160h` is 90 days, and is a reasonable default. This can reasonably be
      # any value, as the application will force remove files if space is
      # constrained.
      #
      # `0s` is also a valid option and can signify that you do not want to keep
      # uploaded files at all. Though do note that this could possibly result in
      # data loss if there are transient upload failures during the lifecycle of
      # the application.
      metricsOlderThan: 2160h
      # If set to true (default), then files older than `metricsOlderThan` will
      # not be deleted unless there is detected storage pressure. For example,
      # if there are files older than `metricsOlderThan` but only 30% of storage
      # space is used, the files will not be deleted.
      lazy: true
      # This controls the percentage of files the application will remove when
      # there is critical storage pressure. This is defined by >95% of storage
      # usage.
      percent: 20
    # Configuration for the emptyDir volume used by the aggregator.
    emptyDir:
      # Whether to enable the emptyDir volume for the aggregator.
      enabled: true
      # Size limit for the emptyDir volume. If not set, no limit is applied.
      sizeLimit: ""
  # Configuration for the collector component of the aggregator.
  collector:
    # Port that the collector listens on for incoming metrics.
    port: 8080
    # Resource requirements and limits for the collector component.
    #
    # For details, see the Kubernetes documentation on resource management:
    # https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
    resources:
      requests:
        memory: "64Mi"
        cpu: "100m"
      limits:
        memory: "1024Mi"
        cpu: "2000m"
  # Configuration for the shipper component of the aggregator.
  shipper:
    # Port that the shipper listens on for internal communication.
    port: 8081
    # Resource requirements and limits for the shipper component.
    #
    # For details, see the Kubernetes documentation on resource management:
    # https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
    resources:
      requests:
        memory: "64Mi"
        cpu: "100m"
      limits:
        memory: "1024Mi"
        cpu: "2000m"
  # Node selector configuration for the aggregator pods.
  #
  # See the Kubernetes documentation for details:
  # https://kubernetes.io/docs/tasks/configure-pod-container/assign-pods-nodes/
  nodeSelector: {}
  # Tolerations configuration for the aggregator pods.
  #
  # See the Kubernetes documentation for details:
  # https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
  tolerations: []
  # Affinity rules for the aggregator pods.
  #
  # See the Kubernetes documentation for details:
  # https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity
  affinity: {}
{{- end -}}

{{/*
Default values for the CloudZero Agent.

This function takes .Values as an argument and merges it with the default values
defined here. This allows for component-specific defaults while still allowing
users to override them through values.yaml.
*/}}
{{- define "cloudzero-agent.values" -}}
{{- $defaults := include "cloudzero-agent._getValues" . | fromYaml -}}
{{- mergeOverwrite $defaults .Values | toYaml -}}
{{- end -}}

{{/*
Overridable default values for the CloudZero Agent that can be overridden by users.

This function provides a set of default values that users can override through their values.yaml file.
The values defined here are merged with user-provided values, allowing for customization while maintaining
sensible defaults.
*/}}
{{- define "cloudzero-agent.overridableDefaults" -}}
# The aggregator provides an intermediary between the CloudZero Agent and the
# CloudZero API. It is composed of two applications, the collector and the
# shipper.
#
# The collector application provides an endpoint for the CloudZero Agent to
# write metrics to. It filters out any unwanted metrics as it receives them,
# aggregates the wanted metrics, and stores them in a compressed format on disk
# until they are ready to be uploaded to the CloudZero servers. Once the
# collector has aggregated sufficient metrics (or a given amount of time has
# elapsed) the data is sent to the shipper.
#
# The shipper will process the completed metrics files and push them to the
# remote server. It will also handle any requests from the server to re-send any
# missing or incomplete data, ensuring that there is no data loss in the event
# of any loss of communication with the CloudZero API, even when a
# misconfiguration (such as an incorrect API key) prevents it.
aggregator:
  # Configuration for logging behavior of the aggregator components.
  logging:
    # Logging level that will be posted to stdout.
    # Valid values are: 'debug', 'info', 'warn', 'error'
    level: info
  # Top-level directory containing CloudZero data. There will be subdirectories
  # for configuration (the mounted ConfigMap) and the API key (typically a
  # mounted Secret), and data to be uploaded to CloudZero, specifically metrics.
  # This value is really only visible internally in the container, so you
  # shouldn't generally need to change it.
  #
  # Set `aggregator.database.purgeRules` to control the cleanup behavior of this
  # directory.
  mountRoot: /cloudzero
  # Whether to enable the profiling endpoint (/debug/pprof/). This should
  # generally be disabled in production.
  profiling: false
  # Container image configuration for the aggregator components.
  #
  # Deprecated. Please use components.aggregator.image instead.
  image:
    repository:
    tag:
    digest:
    pullPolicy:
  cloudzero:
    # Interval between attempts to ship metrics to the remote endpoint.
    #
    # This is formatted as a Go duration string; see
    # https://pkg.go.dev/time#ParseDuration for details.
    sendInterval: 1m
    # Max time the aggregator will spend attempting to ship metrics to the remote endpoint.
    #
    # This is formatted as a Go duration string; see
    # https://pkg.go.dev/time#ParseDuration for details.
    sendTimeout: 30s
    # Interval at which the aggregator will rotate its internal data structures.
    #
    # This is formatted as a Go duration string; see
    # https://pkg.go.dev/time#ParseDuration for details.
    rotateInterval: 30m
  # Configuration for the aggregator's database and storage behavior.
  database:
    # Max number of records per file. Use this to adjust file sizes uploaded to
    # the server. The default value should generally be left unchanged.
    maxRecords: 1500000
    # Max interval to flush a cost metrics file. This is mostly useful for
    # smaller clusters with little activity.
    costMaxInterval: 10m
    # Max interval to flush an observability metrics file. This is mostly useful
    # for smaller clusters with little activity.
    observabilityMaxInterval: 30m
    # Compression level to use when compressing metrics files on-disk.
    #
    # Valid value range from 0-11, with higher values yielding improved
    # compression ratios at the expense of speed and memory usage.
    #
    # Read more about brotli compression here:
    # https://github.com/google/brotli/blob/master/c/tools/brotli.md#options
    compressionLevel: 8
    # The rules that the application will follow in respect to cleaning up old
    # files that have been uploaded to the CloudZero platform.
    #
    # Generally, the defaults will be okay for the majority of use cases. But,
    # the options are here for more advanced users to optimize disk usage. For
    # example, the default case is to keep uploaded files around for 90 days, as
    # this falls in line with most customer's data tolerance policies. But, if
    # deployed on a more active and/or larger cluster, this value can be lowered
    # to keep disk usage lower with the tradeoff of less data-retention.
    # Regardless of what you define here if there is disk pressure detected,
    # files will be deleted (oldest first) to free space.
    purgeRules:
      # How long to keep uploaded files. This option can be useful to optimize
      # the storage required by the collector/shipper architecture on your
      # nodes.
      #
      # `2160h` is 90 days, and is a reasonable default. This can reasonably be
      # any value, as the application will force remove files if space is
      # constrained.
      #
      # `0s` is also a valid option and can signify that you do not want to keep
      # uploaded files at all. Though do note that this could possibly result in
      # data loss if there are transient upload failures during the lifecycle of
      # the application.
      metricsOlderThan: 2160h
      # If set to true (default), then files older than `metricsOlderThan` will
      # not be deleted unless there is detected storage pressure. For example,
      # if there are files older than `metricsOlderThan` but only 30% of storage
      # space is used, the files will not be deleted.
      lazy: true
      # This controls the percentage of files the application will remove when
      # there is critical storage pressure. This is defined by >95% of storage
      # usage.
      percent: 20
    # Configuration for the emptyDir volume used by the aggregator.
    emptyDir:
      # Whether to enable the emptyDir volume for the aggregator.
      enabled: true
      # Size limit for the emptyDir volume. If not set, no limit is applied.
      sizeLimit: ""
  # Configuration for the collector component of the aggregator.
  collector:
    # Port that the collector listens on for incoming metrics.
    port: 8080
    # Resource requirements and limits for the collector component.
    #
    # For details, see the Kubernetes documentation on resource management:
    # https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
    resources:
      requests:
        memory: "64Mi"
        cpu: "100m"
      limits:
        memory: "1024Mi"
        cpu: "2000m"
  # Configuration for the shipper component of the aggregator.
  shipper:
    # Port that the shipper listens on for internal communication.
    port: 8081
    # Resource requirements and limits for the shipper component.
    #
    # For details, see the Kubernetes documentation on resource management:
    # https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
    resources:
      requests:
        memory: "64Mi"
        cpu: "100m"
      limits:
        memory: "1024Mi"
        cpu: "2000m"
  # Node selector configuration for the aggregator pods.
  #
  # See the Kubernetes documentation for details:
  # https://kubernetes.io/docs/tasks/configure-pod-container/assign-pods-nodes/
  nodeSelector: {}
  # Tolerations configuration for the aggregator pods.
  #
  # See the Kubernetes documentation for details:
  # https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
  tolerations: []
  # Affinity rules for the aggregator pods.
  #
  # See the Kubernetes documentation for details:
  # https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity
  affinity: {}
{{- end -}}
