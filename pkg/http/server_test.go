package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
)

// Basic test with mimimal configuration
func TestNewServer(t *testing.T) {
	cfg := &config.Settings{
		Server: config.Server{
			Port:         "8080",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		},
	}

	routes := []RouteSegment{
		{
			Route: "/test",
			Hook:  http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
		},
	}

	server := NewServer(cfg, routes)

	assert.Equal(t, ":8080", server.Addr)
	assert.Equal(t, 5*time.Second, server.ReadTimeout)
	assert.Equal(t, 10*time.Second, server.WriteTimeout)
	assert.Equal(t, 15*time.Second, server.IdleTimeout)

	handler := server.Handler

	tests := []struct {
		route        string
		expectedCode int
	}{
		{route: "/test", expectedCode: http.StatusOK},
		{route: "/healthz", expectedCode: http.StatusOK},
		{route: "/metrics", expectedCode: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.route, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, tt.route, nil)
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)
			assert.Equal(t, tt.expectedCode, rr.Code)
		})
	}
}

func TestMetricsInterface(t *testing.T) {
	var basicMetrics = []string{
		// # HELP go_gc_duration_seconds A summary of the wall-time pause (stop-the-world) duration in garbage collection cycles.
		// # TYPE go_gc_duration_seconds summary
		"go_gc_duration_seconds",
		"go_gc_duration_seconds",
		"go_gc_duration_seconds",
		"go_gc_duration_seconds",
		"go_gc_duration_seconds",
		"go_gc_duration_seconds_sum",
		"go_gc_duration_seconds_count",
		// # HELP go_gc_gogc_percent Heap size target percentage configured by the user, otherwise 100. This value is set by the GOGC environment variable, and the runtime/debug.SetGCPercent function. Sourced from /gc/gogc:percent
		// # TYPE go_gc_gogc_percent gauge
		"go_gc_gogc_percent",
		// # HELP go_gc_gomemlimit_bytes Go runtime memory limit configured by the user, otherwise math.MaxInt64. This value is set by the GOMEMLIMIT environment variable, and the runtime/debug.SetMemoryLimit function. Sourced from /gc/gomemlimit:bytes
		// # TYPE go_gc_gomemlimit_bytes gauge
		"go_gc_gomemlimit_bytes",
		// # HELP go_goroutines Number of goroutines that currently exist.
		// # TYPE go_goroutines gauge
		"go_goroutines",
		// # HELP go_info Information about the Go environment.
		// # TYPE go_info gauge
		"go_inf",
		// # HELP go_memstats_alloc_bytes Number of bytes allocated in heap and currently in use. Equals to /memory/classes/heap/objects:bytes.
		// # TYPE go_memstats_alloc_bytes gauge
		"go_memstats_alloc_bytes",
		// # HELP go_memstats_alloc_bytes_total Total number of bytes allocated in heap until now, even if released already. Equals to /gc/heap/allocs:bytes.
		// # TYPE go_memstats_alloc_bytes_total counter
		"go_memstats_alloc_bytes_total",
		// # HELP go_memstats_buck_hash_sys_bytes Number of bytes used by the profiling bucket hash table. Equals to /memory/classes/profiling/buckets:bytes.
		// # TYPE go_memstats_buck_hash_sys_bytes gauge
		"go_memstats_buck_hash_sys_bytes",
		// # HELP go_memstats_frees_total Total number of heap objects frees. Equals to /gc/heap/frees:objects + /gc/heap/tiny/allocs:objects.
		// # TYPE go_memstats_frees_total counter
		"go_memstats_frees_total",
		// # HELP go_memstats_gc_sys_bytes Number of bytes used for garbage collection system metadata. Equals to /memory/classes/metadata/other:bytes.
		// # TYPE go_memstats_gc_sys_bytes gauge
		"go_memstats_gc_sys_bytes",
		// # HELP go_memstats_heap_alloc_bytes Number of heap bytes allocated and currently in use, same as go_memstats_alloc_bytes. Equals to /memory/classes/heap/objects:bytes.
		// # TYPE go_memstats_heap_alloc_bytes gauge
		"go_memstats_heap_alloc_bytes",
		// # HELP go_memstats_heap_idle_bytes Number of heap bytes waiting to be used. Equals to /memory/classes/heap/released:bytes + /memory/classes/heap/free:bytes.
		// # TYPE go_memstats_heap_idle_bytes gauge
		"go_memstats_heap_idle_bytes",
		// # HELP go_memstats_heap_inuse_bytes Number of heap bytes that are in use. Equals to /memory/classes/heap/objects:bytes + /memory/classes/heap/unused:bytes
		// # TYPE go_memstats_heap_inuse_bytes gauge
		"go_memstats_heap_inuse_bytes",
		// # HELP go_memstats_heap_objects Number of currently allocated objects. Equals to /gc/heap/objects:objects.
		// # TYPE go_memstats_heap_objects gauge
		"go_memstats_heap_objects",
		// # HELP go_memstats_heap_released_bytes Number of heap bytes released to OS. Equals to /memory/classes/heap/released:bytes.
		// # TYPE go_memstats_heap_released_bytes gauge
		"go_memstats_heap_released_bytes",
		// # HELP go_memstats_heap_sys_bytes Number of heap bytes obtained from system. Equals to /memory/classes/heap/objects:bytes + /memory/classes/heap/unused:bytes + /memory/classes/heap/released:bytes + /memory/classes/heap/free:bytes.
		// # TYPE go_memstats_heap_sys_bytes gauge
		"go_memstats_heap_sys_bytes",
		// # HELP go_memstats_last_gc_time_seconds Number of seconds since 1970 of last garbage collection.
		// # TYPE go_memstats_last_gc_time_seconds gauge
		"go_memstats_last_gc_time_seconds",
		// # HELP go_memstats_mallocs_total Total number of heap objects allocated, both live and gc-ed. Semantically a counter version for go_memstats_heap_objects gauge. Equals to /gc/heap/allocs:objects + /gc/heap/tiny/allocs:objects.
		// # TYPE go_memstats_mallocs_total counter
		"go_memstats_mallocs_total",
		// # HELP go_memstats_mcache_inuse_bytes Number of bytes in use by mcache structures. Equals to /memory/classes/metadata/mcache/inuse:bytes.
		// # TYPE go_memstats_mcache_inuse_bytes gauge
		"go_memstats_mcache_inuse_bytes",
		// # HELP go_memstats_mcache_sys_bytes Number of bytes used for mcache structures obtained from system. Equals to /memory/classes/metadata/mcache/inuse:bytes + /memory/classes/metadata/mcache/free:bytes.
		// # TYPE go_memstats_mcache_sys_bytes gauge
		"go_memstats_mcache_sys_bytes",
		// # HELP go_memstats_mspan_inuse_bytes Number of bytes in use by mspan structures. Equals to /memory/classes/metadata/mspan/inuse:bytes.
		// # TYPE go_memstats_mspan_inuse_bytes gauge
		"go_memstats_mspan_inuse_bytes",
		// # HELP go_memstats_mspan_sys_bytes Number of bytes used for mspan structures obtained from system. Equals to /memory/classes/metadata/mspan/inuse:bytes + /memory/classes/metadata/mspan/free:bytes.
		// # TYPE go_memstats_mspan_sys_bytes gauge
		"go_memstats_mspan_sys_bytes",
		// # HELP go_memstats_next_gc_bytes Number of heap bytes when next garbage collection will take place. Equals to /gc/heap/goal:bytes.
		// # TYPE go_memstats_next_gc_bytes gauge
		"go_memstats_next_gc_bytes",
		// # HELP go_memstats_other_sys_bytes Number of bytes used for other system allocations. Equals to /memory/classes/other:bytes.
		// # TYPE go_memstats_other_sys_bytes gauge
		"go_memstats_other_sys_bytes",
		// # HELP go_memstats_stack_inuse_bytes Number of bytes obtained from system for stack allocator in non-CGO environments. Equals to /memory/classes/heap/stacks:bytes.
		// # TYPE go_memstats_stack_inuse_bytes gauge
		"go_memstats_stack_inuse_bytes",
		// # HELP go_memstats_stack_sys_bytes Number of bytes obtained from system for stack allocator. Equals to /memory/classes/heap/stacks:bytes + /memory/classes/os-stacks:bytes.
		// # TYPE go_memstats_stack_sys_bytes gauge
		"go_memstats_stack_sys_bytes",
		// # HELP go_memstats_sys_bytes Number of bytes obtained from system. Equals to /memory/classes/total:byte.
		// # TYPE go_memstats_sys_bytes gauge
		"go_memstats_sys_bytes",
		// # HELP go_sched_gomaxprocs_threads The current runtime.GOMAXPROCS setting, or the number of operating system threads that can execute user-level Go code simultaneously. Sourced from /sched/gomaxprocs:threads
		// # TYPE go_sched_gomaxprocs_threads gauge
		"go_sched_gomaxprocs_threads",
		// # HELP go_threads Number of OS threads created.
		// # TYPE go_threads gauge
		"go_threads",
		// # HELP promhttp_metric_handler_requests_in_flight Current number of scrapes being served.
		// # TYPE promhttp_metric_handler_requests_in_flight gauge
		"promhttp_metric_handler_requests_in_flight",
		// # HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
		// # TYPE promhttp_metric_handler_requests_total counter
		"promhttp_metric_handler_requests_total",
		"promhttp_metric_handler_requests_total",
		"promhttp_metric_handler_requests_total",
	}

	cfg := &config.Settings{
		Server: config.Server{
			Port:         "8080",
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  15 * time.Second,
		},
	}

	routes := []RouteSegment{
		{
			Route: "/test",
			Hook:  http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }),
		},
	}

	server := NewServer(cfg, routes)

	handler := server.Handler

	req, err := http.NewRequest(http.MethodGet, "/metrics", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req) // Use the handler interface directly

	assert.Equal(t, http.StatusOK, rr.Code)
	body := rr.Body.String()

	for _, metric := range basicMetrics {
		assert.Contains(t, body, metric)
	}
}
