// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package instr

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	api "go.opentelemetry.io/otel/sdk/metric"
)

/*
Sources:
- https://signoz.io/opentelemetry/go/
- https://betterstack.com/community/guides/observability/opentelemetry-metrics-golang/
- https://opentelemetry.io/docs/specs/otel/metrics/sdk_exporters/prometheus/
- https://pkg.go.dev/go.opentelemetry.io/otel/exporters/prometheus
- https://github.com/open-telemetry/opentelemetry-go-contrib/tree/main/examples/prometheus

*/

const OTEL_SCOPE_NAME = "github.com/Cloudzero/cloudzero-insights-controller"
const OTEL_SCOPE_VERSION = "0.0.1-dev1"

var (
	initOnce sync.Once
	provider *api.MeterProvider

	// testing metric
	testCounter metric.Int64Counter

	// START - define metrics -------------

	// the total number of requests we have posting to the remote write endpoint
	RemoteWriteRequestCount metric.Int64Counter

	// the total duration in seconds a remote write request takes
	RemoteWriteRequestDurationSeconds metric.Float64Histogram

	// status code counts broken down with whatever is labeled on metric post time
	RemoteWriteStatusCodes metric.Int64Counter

	// total size of a payload written to the remote write endpoint.
	RemoteWritePayloadSizeBytes metric.Int64Histogram

	// total number of failures on the remote write endpoint
	RemoteWriteFailures metric.Int64Counter
)

func init() {
	// extra insurance that the particular init function is only ran ONCE
	initOnce.Do(func() {
		if err := _init(); err != nil {
			log.Fatalf("Failed to setup otel metrics: %s", err.Error())
		}
	})
}

// create the metric resources. When using the `prometheus` package, a meter provider lets
// us use otel metrics while exposing to a prometheus-compatable write endpoint
func _init() error {
	var err error
	exporter, err := prometheus.New()
	if err != nil {
		log.Fatal(err)
	}
	provider = api.NewMeterProvider(api.WithReader(exporter))
	otel.SetMeterProvider(provider)

	// read environment to get the scope name and version
	name := os.Getenv("OTEL_SCOPE_NAME")
	if name == "" {
		name = OTEL_SCOPE_NAME
	}
	version := os.Getenv("OTEL_SCOPE_VERSION")
	if version == "" {
		version = OTEL_SCOPE_VERSION
	}

	// initialize the metrics we want to track
	meter := provider.Meter(name, metric.WithInstrumentationVersion(version))

	testCounter, err = meter.Int64Counter("test_counter")
	if err != nil {
		return err
	}

	// START - initialize all metrics -----------------

	RemoteWriteRequestCount, err = meter.Int64Counter(
		"remote_write_request_total",
		metric.WithDescription("Total number of write attempts against the write endpoint"),
	)
	if err != nil {
		return err
	}

	RemoteWriteRequestDurationSeconds, err = meter.Float64Histogram(
		"remote_write_request_duration",
		metric.WithDescription("Duration of requests to the remote write endpoint"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(prom.DefBuckets...),
	)
	if err != nil {
		return err
	}

	RemoteWriteStatusCodes, err = meter.Int64Counter(
		"remote_write_status_codes_total",
		metric.WithDescription("Count of response status codes from remote write endpoint"),
	)
	if err != nil {
		return err
	}

	RemoteWritePayloadSizeBytes, err = meter.Int64Histogram(
		"remote_write_payload_size_bytes",
		metric.WithDescription("Payload size posted to the remote write endpoint"),
		metric.WithUnit("byte"),
		metric.WithExplicitBucketBoundaries(prom.ExponentialBuckets(256, 2, 10)...),
	)
	if err != nil {
		return err
	}

	RemoteWriteFailures, err = meter.Int64Counter(
		"remote_write_failure_total",
		metric.WithDescription("Total number of failures for the remote write endpoint"),
	)
	if err != nil {
		return err
	}

	return nil
}

// Must use this handler, as the metrics endpoint needs the `init` function to be called first
func Handler() http.Handler {
	return promhttp.Handler()
}

// Usually do not need to call this, but may prove useful in specific scenarios
func Flush(ctx context.Context) error {
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := provider.ForceFlush(c); err != nil {
		return fmt.Errorf("error flushing metrics: %w", err)
	}
	return nil
}
