// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package instr

import (
	"context"
	"sync"
	"time"

	"github.com/cloudzero/cloudzero-agent-validator/app/logging"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
	ctx      context.Context
	id       string
	parentID string
	name     string
	start    time.Time
	err      error
	ended    bool
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

func (s *Span) StartChildSpan(name string) *Span {
	return &Span{
		ctx:      s.ctx,
		id:       uuid.NewString(),
		parentID: s.id,
		name:     name,
		start:    time.Now(),
	}
}

// SpanLogger fetches a logger for the supplied span id. This will search
// up the context for a potential parent id, and properly set all attributes
// on the found logger.
//
// The base logger used is the default logger from the passed context.
func SpanLogger(ctx context.Context, id string, attrs ...logging.Attr) zerolog.Logger {
	// build the logger
	builder := log.Ctx(ctx).With().Str("spanId", id)

	// search the current context for a span
	parentID := getParentID(ctx)
	if parentID != "" {
		builder = builder.Str("parentSpanId", parentID)
	}

	// add the attributes
	for _, attr := range attrs {
		builder = attr(builder)
	}

	// return as a logger
	return builder.Logger()
}

// Logger returns a new logger for the span.
func (s *Span) Logger(attrs ...logging.Attr) zerolog.Logger {
	return SpanLogger(s.ctx, s.id, attrs...)
}

// StartSpan starts a span using the default prometheus registry
func StartSpan(ctx context.Context, name string) *Span {
	spanOnce.Do(func() {
		prometheus.MustRegister(functionDuration)
	})

	return &Span{
		ctx:      ctx,
		id:       uuid.NewString(),
		parentID: getParentID(ctx), // search context for a parent span
		name:     name,
		start:    time.Now(),
	}
}

// StartSpan starts a span with a given function name using the existing
// registry found in the PrometheusMetrics
func (p *PrometheusMetrics) StartSpan(ctx context.Context, name string) *Span {
	spanOnce.Do(func() {
		p.registry.MustRegister(functionDuration)
		(*p.metrics) = append((*p.metrics), functionDuration)
	})

	return &Span{
		ctx:      ctx,
		id:       uuid.NewString(),
		parentID: getParentID(ctx), // search context for a parent span
		name:     name,
		start:    time.Now(),
	}
}

// Span provides an extremely basic span function wrapper to track execution
// time. This is NOT a replacement for otel, just a simple exercise that may
// prove useful in certain cases. The function provides the spanId as the
// argument `id` in the function.
//
// In addition, this function transiently passes the error to the caller
func (p *PrometheusMetrics) Span(name string, fn func(id string) error) error {
	spanOnce.Do(func() {
		p.registry.MustRegister(functionDuration)
		(*p.metrics) = append((*p.metrics), functionDuration)
	})

	span := p.StartSpan(context.Background(), name)
	defer span.End()
	err := fn(span.id) // run the actual function
	return span.Error(err)
}

// SpanCtx functions the same as Span but accepts a context.
//
// The context will contain a spanID key to track the current span, and if SpanCtx is called within the runtime
// of another span, the parent span id will be embeded into the span.
func (p *PrometheusMetrics) SpanCtx(ctx context.Context, name string, fn func(ctx context.Context, id string) error) error {
	spanOnce.Do(func() {
		p.registry.MustRegister(functionDuration)
		(*p.metrics) = append((*p.metrics), functionDuration)
	})

	// search the current context for a span
	parentID := getParentID(ctx)

	// create the span
	span := p.StartSpan(ctx, name)
	span.parentID = parentID
	defer span.End()

	// create a new context with the span id as the context key
	ctxWithSpan := context.WithValue(ctx, spanIDKey, span.id)

	// call with the embeded context
	err := fn(ctxWithSpan, span.id)
	return span.Error(err)
}
