// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package instr

import (
	"context"
	"sync"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/logging"
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

// RunSpan will wrap a function with a span to automatically handle
// error tracking and ending
func RunSpan(ctx context.Context, name string, fn func(ctx context.Context, span *Span) error) error {
	spanOnce.Do(func() {
		prometheus.MustRegister(functionDuration)
	})

	// run the span in as a handler
	span := StartSpan(ctx, name)
	defer span.End()

	// create a new context with the span id as the context key
	ctxWithSpan := context.WithValue(ctx, spanIDKey, span.id)

	// call the function wrapped with span error handler
	return span.Error(fn(ctxWithSpan, span))
}

// Error observes an error and optionally transiently passes it to the caller
func (s *Span) Error(err error) error {
	s.err = err
	return err
}

// GetDuration returns the current duration of the span in seconds
func (s *Span) GetDuration() time.Duration {
	return time.Since(s.start)
}

// TraceLog will print a trace log record using the default zerolog logger reporting
// status about the span
func (s *Span) TraceLog() {
	log.Ctx(s.ctx).Trace().
		Str("spanId", s.id).
		Str("spanName", s.name).
		Str("parentId", s.parentID).
		Time("start", s.start).
		Dur("duration", s.GetDuration()).
		Bool("ended", s.ended).
		Err(s.err).
		Msg("Span status")
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

		// debug print the span status
		s.TraceLog()
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

// StartSpan starts a span with a given function name using the existing
// registry found in the PrometheusMetrics
func (p *PrometheusMetrics) StartSpan(ctx context.Context, name string) *Span {
	spanOnce.Do(func() {
		p.registry.MustRegister(functionDuration)
		(*p.metrics) = append((*p.metrics), functionDuration)
	})

	return StartSpan(ctx, name)
}

// SpanCtx is an extremely basic span function wrapper to track execution time.
// This is NOT a replacement for otel, just a simple exercise that may prove useful
// in certain cases. The function provides the spanId as the argument `id` in the
// function.
//
// # In addition, this function transiently passes the error to the caller
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
	err := span.Error(fn(ctxWithSpan, span.id))

	return err
}
