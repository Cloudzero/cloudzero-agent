package instr

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	registerOnce sync.Once
)

type PrometheusMetrics struct {
	registry    *prometheus.Registry
	metrics     *[]prometheus.Collector
	noGometrics bool
}

type PrometheusMetricsOpt func(*PrometheusMetrics)

func WithPromMetrics(m ...prometheus.Collector) PrometheusMetricsOpt {
	return func(p *PrometheusMetrics) {
		p.metrics = &m
	}
}

func WithCustomRegistry(registry *prometheus.Registry) PrometheusMetricsOpt {
	return func(p *PrometheusMetrics) {
		p.registry = registry
	}
}

func WithDefaultRegistry() PrometheusMetricsOpt {
	return func(p *PrometheusMetrics) {
		p.registry = prometheus.DefaultRegisterer.(*prometheus.Registry)
	}
}

func WithNoGoMetrics() PrometheusMetricsOpt {
	return func(p *PrometheusMetrics) {
		p.noGometrics = true
	}
}

// Create a new prometheus metrics object. This will setup sane default prometheus
// metrics, with additional configuration with `...PrometheusMetricsOpt`.
func NewPrometheusMetrics(opts ...PrometheusMetricsOpt) (*PrometheusMetrics, error) {
	p := &PrometheusMetrics{}

	// apply the options
	for _, item := range opts {
		item(p)
	}

	// register the metrics
	var registerErr error
	registerOnce.Do(func() {
		// apply a default internal registry if none set
		if p.registry == nil {
			p.registry = prometheus.NewRegistry()

			// if using the default register, include the default go metrics as well if applicable
			if !p.noGometrics {
				p.registry.Register(collectors.NewGoCollector())
				p.registry.Register(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
			}
		}

		// register all the user-defined metrics
		registerErr = p.register()
	})
	if registerErr != nil {
		return nil, registerErr
	}

	return p, nil
}

// Get the http handler for this specific instance of the prometheus
// metrics registry. If the `WithDefaultRegistry` option was used,
// then calling `promhttp.Handler()` will return the same handler
func (p *PrometheusMetrics) Handler() http.Handler {
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{})
}

// Internal only. This registers the metrics WITHOUT BEING SAFE
// this can cause panics if not used correctly.
func (p *PrometheusMetrics) register() error {
	for _, item := range *p.metrics {
		if err := p.registry.Register(item); err != nil {
			return fmt.Errorf("failed to register a metric: %w", err)
		}
	}
	return nil
}

// Removes stored metrics from the registry. Calling this function on a metric not in the
// registry will not cause a panic
func (p *PrometheusMetrics) clearRegistry() {
	for _, item := range *p.metrics {
		p.registry.Unregister(item)
	}
}
