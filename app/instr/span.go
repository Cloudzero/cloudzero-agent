// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package instr

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	spanOnce sync.Once

	functionDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "function_execution_seconds",
			Help:    "Time taken for a function execution",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
		},
		[]string{"function_name", "error"},
	)
)

type Span struct {
	id    string
	name  string
	start time.Time
	err   error
	ended bool
}

// Start a span with a given function name
func (p *PrometheusMetrics) StartSpan(name string) *Span {
	spanOnce.Do(func() {
		p.registry.MustRegister(functionDuration)
		(*p.metrics) = append((*p.metrics), functionDuration)
	})

	return &Span{
		id:    uuid.NewString(),
		name:  name,
		start: time.Now(),
	}
}

// Error observes an error and optionally transiently passes it to the caller
func (s *Span) Error(err error) error {
	s.err = err
	return err
}

// End ends the span and observes the underlying prometheus metric
func (s *Span) End() {
	if !s.ended {
		duration := time.Since(s.start).Seconds()
		if s.err == nil {
			functionDuration.WithLabelValues(s.name, "").Observe(duration)
		} else {
			functionDuration.WithLabelValues(s.name, s.err.Error()).Observe(duration)
		}
	}
}

// An extremely basic span function wrapper to track execution time.
// This is NOT a replacement for otel, just a simple exercise that may prove useful
// in certain cases.
//
// In addition, this function transiently passes the error to the caller
func (p *PrometheusMetrics) Span(name string, fn func(id string) error) error {
	spanOnce.Do(func() {
		p.registry.MustRegister(functionDuration)
		(*p.metrics) = append((*p.metrics), functionDuration)
	})

	span := p.StartSpan(name)
	defer span.End()
	err := fn(span.id) // run the actual function
	return span.Error(err)
}
