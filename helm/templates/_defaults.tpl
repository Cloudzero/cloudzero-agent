{{/*
Internal default values.

You can think of this as similar to `.Values`, but without allowing people
installing the Helm chart to override the values easily.  This is used in places
where we want to reuse values instead of hardcode them, but at the same time the
values shouldn't really be changed.
*/}}
{{- define "cloudzero-agent.defaults" -}}

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
