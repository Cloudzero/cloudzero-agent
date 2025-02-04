package instr

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	spanOnce sync.Once

	functionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "function_execution_seconds",
			Help:    "Time taken for a function execution",
			Buckets: prometheus.DefBuckets, // Default buckets
		},
		[]string{"function_name", "error"},
	)
)

// An extremely basic span function wrapper to track execution time.
// This is NOT a replacement for otel, just a simple exercise that may prove useful
// in certain cases
func (p *PrometheusMetrics) Span(name string, fn func() error) {
	spanOnce.Do(func() {
		p.registry.MustRegister(functionDuration)
		(*p.metrics) = append((*p.metrics), functionDuration)
	})

	start := time.Now()
	err := fn() // run the actual function
	duration := time.Since(start).Seconds()
	if err == nil {
		functionDuration.WithLabelValues(name, "").Observe(duration)
	} else {
		functionDuration.WithLabelValues(name, err.Error()).Observe(duration)
	}
}
