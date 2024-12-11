package http

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Define your Prometheus metrics
var (
	RequestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Count of all HTTP requests processed, labeled by route, method and status code.",
		},
		[]string{"route", "method", "status_code"},
	)

	RequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of request durations, labeled by route, method and status code.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"route", "method", "status_code"},
	)

	httpMiddlewareStatsOnce sync.Once
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func MetricsMiddlewareWrapper(next http.Handler) http.Handler {
	httpMiddlewareStatsOnce.Do(func() {
		prometheus.MustRegister(RequestCount, RequestDuration)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		duration := time.Since(startTime).Seconds()
		statusCode := strconv.Itoa(recorder.status)
		route := r.URL.Path
		method := r.Method

		// Increment the request count
		RequestCount.WithLabelValues(route, method, statusCode).Inc()

		// Observe the request duration
		RequestDuration.WithLabelValues(route, method, statusCode).Observe(duration)
	})
}
