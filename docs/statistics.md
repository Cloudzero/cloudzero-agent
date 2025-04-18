# CloudZero Insights Controller Prometheus Statistics

## /metrics Endpoint

The `/metrics` endpoint is exposed to provide Prometheus-style metrics. This endpoint is designed to be scraped by Prometheus to collect various statistics about the CloudZero Insights Controller.

### Features

- **Standard Metrics**: Includes default metrics provided by the Prometheus client libraries.
- **Custom Metrics**: Additional application-specific metrics to monitor the performance and health of the CloudZero Insights Controller.

### Usage

To access the metrics, simply make an HTTP GET request to the `/metrics` endpoint:

```
GET /metrics
```

### Example

```sh
curl https://<controller-pod-ip>:8443/metrics
```

Replace `<controller-pod-ip>` with the appropriate values for your deployment. You may need to use port forwarding to access the pod directly from your local machine.

### Integration with Prometheus

The Insights Controller deployment includes:

- [Kubernetes Service](https://github.com/Cloudzero/cloudzero-charts/blob/release/1.0.0/charts/cloudzero-insights-controller/templates/service.yaml)
- [ReplicaSet](https://github.com/Cloudzero/cloudzero-charts/blob/release/1.0.0/charts/cloudzero-insights-controller/templates/deploy.yaml#L7) - the Insights Controller application. [ReplicaSet](https://github.com/Cloudzero/cloudzero-charts/blob/release/1.0.0/charts/cloudzero-insights-controller/templates/deploy.yaml#L9) defaults to 3.

To integrate with Prometheus, add the following job configuration to your Prometheus configuration file. This configuration leverages Kubernetes service discovery to automatically find and scrape all instances of the CloudZero Insights Controller.

```yaml
scrape_configs:
  - job_name: "cloudzero-webhook"
    kubernetes_sd_configs:
      - role: endpoints
    relabel_configs:
      - source_labels: [__meta_kubernetes_service_label_app]
        action: keep
        regex: cloudzero-webhook
```

#### Namespace and Labels

Note that the above assumes the target is uniquely identified by the label `app=cloudzero-insights-controller`, modification may be necessary to perform service discovery accross different namespaces in the `relabel_configs` section for a custom scrape configuration.

### Metrics Overview

The `/metrics` endpoint provides a variety of metrics, including but not limited to:

- **HTTP request durations**
- **Error rates**
- **Resource usage**

These metrics help in monitoring the health and performance of the CloudZero Insights Controller, enabling proactive issue detection and resolution.

## Metrics, Gauges, and Histograms

The default Prometheus registry in Go, when using the Prometheus client library, automatically exposes a set of standard metrics related to the Go runtime and process. These metrics provide insights into the performance and behavior of the Go application and the underlying system. Below are the key metrics that are auto-exposed:

### Go Runtime Metrics

| Metric Name                        | Description                                                              |
| ---------------------------------- | ------------------------------------------------------------------------ |
| `go_gc_duration_seconds`           | A summary of the GC (garbage collection) duration.                       |
| `go_goroutines`                    | The number of goroutines that currently exist.                           |
| `go_info`                          | Information about the Go environment.                                    |
| `go_memstats_alloc_bytes`          | The number of bytes allocated and still in use.                          |
| `go_memstats_alloc_bytes_total`    | The total number of bytes allocated, even if freed.                      |
| `go_memstats_buck_hash_sys_bytes`  | The number of bytes used by the profiling bucket hash table.             |
| `go_memstats_frees_total`          | The total number of frees.                                               |
| `go_memstats_gc_cpu_fraction`      | The fraction of this program's available CPU time used by the GC.        |
| `go_memstats_gc_sys_bytes`         | The number of bytes used for garbage collection system metadata.         |
| `go_memstats_heap_alloc_bytes`     | The number of heap bytes allocated and still in use.                     |
| `go_memstats_heap_idle_bytes`      | The number of heap bytes waiting to be used.                             |
| `go_memstats_heap_inuse_bytes`     | The number of heap bytes that are in use.                                |
| `go_memstats_heap_objects`         | The number of allocated objects.                                         |
| `go_memstats_heap_released_bytes`  | The number of heap bytes released to the OS.                             |
| `go_memstats_heap_sys_bytes`       | The number of heap bytes obtained from the system.                       |
| `go_memstats_last_gc_time_seconds` | The time the last garbage collection finished.                           |
| `go_memstats_lookups_total`        | The total number of pointer lookups.                                     |
| `go_memstats_mallocs_total`        | The total number of mallocs.                                             |
| `go_memstats_mcache_inuse_bytes`   | The number of bytes in use by mcache structures.                         |
| `go_memstats_mcache_sys_bytes`     | The number of bytes used for mcache structures obtained from the system. |
| `go_memstats_mspan_inuse_bytes`    | The number of bytes in use by mspan structures.                          |
| `go_memstats_mspan_sys_bytes`      | The number of bytes used for mspan structures obtained from the system.  |
| `go_memstats_next_gc_bytes`        | The target heap size of the next GC.                                     |
| `go_memstats_other_sys_bytes`      | The number of bytes used for other system allocations.                   |
| `go_memstats_stack_inuse_bytes`    | The number of bytes in use by the stack allocator.                       |
| `go_memstats_stack_sys_bytes`      | The number of bytes obtained from the system for stack memory.           |
| `go_memstats_sys_bytes`            | The total number of bytes obtained from the system.                      |
| `go_threads`                       | The number of OS threads created.                                        |

### Process Metrics

| Metric Name                        | Description                                            |
| ---------------------------------- | ------------------------------------------------------ |
| `process_cpu_seconds_total`        | Total user and system CPU time spent in seconds.       |
| `process_max_fds`                  | Maximum number of open file descriptors.               |
| `process_open_fds`                 | Number of open file descriptors.                       |
| `process_resident_memory_bytes`    | Resident memory size in bytes.                         |
| `process_start_time_seconds`       | Start time of the process since unix epoch in seconds. |
| `process_virtual_memory_bytes`     | Virtual memory size in bytes.                          |
| `process_virtual_memory_max_bytes` | Maximum amount of virtual memory available in bytes.   |

These metrics are crucial for monitoring the health and performance of Go applications and can be easily scraped by Prometheus for further analysis and alerting.

### Remote Write Metrics

The Remote Write Metrics track the status of information related to sending metrics to the CloudZero platform remote write endpoint.

| Metric Name                             | Description                                                                           |
| --------------------------------------- | ------------------------------------------------------------------------------------- |
| `remote_write_timeseries_total`         | Total number of timeseries attempted to be sent to remote write endpoint.             |
| `remote_write_request_duration_seconds` | Histogram of request durations to remote write endpoint.                              |
| `remote_write_response_codes_total`     | Count of response codes from remote write endpoint.                                   |
| `remote_write_payload_size_bytes`       | Size of payloads sent to remote write endpoint in bytes.                              |
| `remote_write_failures_total`           | Total number of failed attempts to write metrics to the remote endpoint.              |
| `remote_write_backlog_records`          | Number of records that are currently waiting to be sent to the remote write endpoint. |
| `remote_write_records_processed_total`  | Total number of records successfully processed (sent and marked as sent_at).          |
| `remote_write_db_failures_total`        | Total number of failures when updating sent_at for records in the database.           |

### Storage Metrics

| Metric Name                   | Description                                                                                             |
| ----------------------------- | ------------------------------------------------------------------------------------------------------- |
| `storage_write_failure_total` | Total number of storage write failures. Labeled by resource_type, namespace, resource_name, and action. |

This metric helps in monitoring and alerting on storage write failures, providing insights into the reliability and performance of the storage subsystem.

### HTTP Middleware Metrics

As a service (admission controller), we use the following metrics to track incoming API invocations using our HTTP middleware:

| Metric Name                     | Description                                                                      |
| ------------------------------- | -------------------------------------------------------------------------------- |
| `http_requests_total`           | Count of all HTTP requests processed, labeled by route, method, and status code. |
| `http_request_duration_seconds` | Histogram of request durations, labeled by route, method, and status code.       |

These metrics are defined and registered in our middleware package and are crucial for monitoring the performance and behavior of our HTTP endpoints.
