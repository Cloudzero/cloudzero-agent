{{/*
CloudZero Agent Default Configuration Template

This template defines immutable default values for CloudZero Agent operations that are essential
for proper cost allocation and monitoring functionality. Unlike user-configurable values in
`.Values`, these defaults represent core operational requirements that should not be modified
without understanding the impact on CloudZero platform integration.

The template provides:
- Essential metric lists required for CloudZero cost allocation analysis
- Metric filtering configuration for cost vs observability data classification
- Default operational parameters that ensure reliable CloudZero Agent functionality

Modifying these defaults may result in:
- Incomplete cost allocation data in the CloudZero platform
- Missing operational metrics for monitoring and alerting
- Degraded performance or functionality of CloudZero Agent components

Use Cases:
- Template reusability: Share common configuration across multiple chart templates
- Consistency: Ensure identical metric lists and filters across different deployments
- Maintainability: Centralize critical configuration that should remain stable
- Documentation: Provide clear reference for required CloudZero metrics and filters
*/}}
{{/*
Defines the complete set of default values for CloudZero Agent operations.
This template returns a YAML structure containing all essential configuration
for metric collection, filtering, and CloudZero platform integration.

Structure:
- kubeMetrics: Kubernetes state metrics required for resource cost allocation
- containerMetrics: Container runtime metrics for resource usage tracking
- insightsMetrics: CloudZero Agent operational metrics for monitoring and debugging
- prometheusMetrics: Prometheus server metrics for integration health monitoring
- metricFilters: Classification rules for cost vs observability metric separation

Usage in templates:
{{- $defaults := include "cloudzero-agent.defaults" . | fromYaml -}}
{{- range $defaults.kubeMetrics }}
  - {{ . }}
{{- end }}
*/}}
{{- define "cloudzero-agent.defaults" -}}

# Essential Metric Collections for CloudZero Cost Allocation
#
# These metric lists are critical for CloudZero platform functionality and should not be modified
# without consulting CloudZero support. Each category serves a specific purpose in the cost
# allocation and monitoring pipeline.
#
# WARNING: Removing or modifying these metrics may result in incomplete cost data,
# missing operational insights, or degraded CloudZero platform functionality.

# Kubernetes State Metrics - Essential for Resource Cost Allocation
# These metrics provide the foundational data for attributing costs to specific Kubernetes
# resources, namespaces, and workloads. Required for accurate cost allocation analysis.
kubeMetrics:
  - kube_node_info
  - kube_node_status_capacity
  - kube_pod_container_resource_limits
  - kube_pod_container_resource_requests
  - kube_pod_labels
  - kube_pod_info
# Container Runtime Metrics - Essential for Resource Usage Tracking
# These metrics capture actual resource consumption by containers, enabling CloudZero
# to correlate resource requests/limits with actual usage for cost optimization insights.
# GPU metrics are collected in native DCGM format and transformed by the collector
# to percentage-based metrics for consistent reporting.
containerMetrics:
  - container_cpu_usage_seconds_total
  - container_memory_working_set_bytes
  - container_network_receive_bytes_total
  - container_network_transmit_bytes_total
  - container_resources_gpu_usage_percent
  - container_resources_gpu_memory_usage_percent
# CloudZero Agent Operational Metrics - Essential for Agent Health Monitoring
# These metrics track CloudZero Agent performance, resource usage, and operational health,
# enabling monitoring, alerting, and troubleshooting of the cost allocation pipeline.
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
  - prometheus_remote_write_wal_storage_active_series
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

# Metric Filtering Configuration - Controls Data Classification and Transmission
#
# This configuration determines which metrics are sent to CloudZero and how they are classified
# for cost allocation vs observability analysis. The filtering system provides fine-grained
# control over metric selection and label inclusion based on multiple matching strategies.
#
# Filter Structure:
# - cost: Metrics used for cost allocation and resource attribution analysis
# - observability: Metrics used for operational monitoring and system health tracking
#
# Each filter type (cost/observability) contains:
# - name: Filters applied to metric names to determine inclusion
# - labels: Filters applied to metric labels to determine which labels to include
#
# Matching Strategies:
# - exact: Exact string matches for precise metric selection
# - prefix: Prefix-based matching for metric family grouping
# - suffix: Suffix-based matching for metric type identification
# - contains: Substring matching for flexible metric selection
# - regex: Regular expression matching for complex pattern-based selection
#
# Filter Behavior:
# - Empty filters: All subjects match (inclusive by default)
# - Multiple filters: Any matching filter includes the subject (OR logic)
# - Label filtering: Applied only to metrics that pass name filtering
#
# Extension Mechanism:
# Each filter type supports "additional..." properties for extending default filters
# without overriding core CloudZero requirements. Use additional properties in
# values override files to customize filtering for specific environments.
#
# IMPORTANT: Modifying core filters may impact CloudZero cost allocation accuracy.
# Use additional filters for environment-specific customization.
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
        - container_resources_gpu_usage_percent
        - container_resources_gpu_memory_usage_percent
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
        - resource_type
        - unit
        - uid
        - workload
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
        - prometheus_remote_write_wal_storage_active_series
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
