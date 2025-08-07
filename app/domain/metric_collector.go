// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package domain provides the core business logic for the CloudZero Kubernetes agent.
//
// This package contains the primary domain entities and services that implement
// the agent's core functionality:
//
//   - MetricCollector: Receives and processes Prometheus metrics from remote write endpoints
//   - MetricFilter: Separates cost metrics from observability metrics based on configuration
//   - File monitoring and webhook processing for Kubernetes resource tracking
//   - Storage abstractions for persisting metrics data
//
// The domain layer implements clean architecture principles, with business logic
// isolated from infrastructure concerns. External dependencies are injected through
// interfaces defined in the types package.
//
// Key workflows:
//   - Prometheus remote write → MetricCollector → MetricFilter → Storage
//   - Kubernetes webhook → WebhookController → Storage
//   - Periodic metric shipping via MetricShipper
//
// Integration points:
//   - app/handlers: HTTP endpoint handlers that delegate to domain services
//   - app/config: Configuration validation and settings management  
//   - app/storage: Persistence implementations (disk, sqlite)
//   - app/types: Shared interfaces and data structures
package domain

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	prom "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/prometheus/prometheus/prompb"
	writev2 "github.com/prometheus/prometheus/prompb/io/prometheus/write/v2"
	"github.com/prometheus/prometheus/storage/remote"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	config "github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

// Metric processing errors
var (
	// ErrJSONUnmarshal indicates failure to parse Prometheus protobuf data from request body.
	// This error occurs when the remote write payload is malformed or uses an unsupported format.
	ErrJSONUnmarshal = errors.New("failed to parse metric from request body")
	
	// ErrMetricIDMismatch indicates a validation error where the metric ID in the URL path
	// does not match the product ID found in the request body.
	ErrMetricIDMismatch = errors.New("metric ID in path does not match product ID in body")
)

// Content type and compression constants for Prometheus remote write protocol
const (
	// SnappyBlockCompression indicates snappy compression is used on the request body.
	// This is the standard compression format used by Prometheus remote write.
	SnappyBlockCompression = "snappy"
	
	// appProtoContentType is the expected MIME type for Prometheus protobuf payloads.
	// Remote write requests must use this content type header.
	appProtoContentType = "application/x-protobuf"
)

// Prometheus protocol version content types
var (
	// v1ContentType is the content type identifier for Prometheus remote write v1 protocol.
	// This is the standard protocol version used by most Prometheus installations.
	v1ContentType = string(prom.RemoteWriteProtoMsgV1)
	
	// v2ContentType is the content type identifier for Prometheus remote write v2 protocol.
	// This newer version includes enhanced metadata and statistics reporting.
	v2ContentType = string(prom.RemoteWriteProtoMsgV2)
)

// Prometheus metrics for monitoring metric ingestion and processing
var (
	// metricsReceived tracks the total number of metrics received from remote write requests.
	// This includes all metrics before filtering (cost + observability + dropped).
	metricsReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "metrics_received_total",
			Help: "Total number of metrics received",
		},
		[]string{},
	)
	
	// metricsReceivedCost tracks the number of cost-related metrics received after filtering.
	// These are the metrics that contribute to CloudZero cost analysis.
	metricsReceivedCost = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "metrics_received_cost_total",
			Help: "Total number of cost metrics received",
		},
		[]string{},
	)
	
	// metricsReceivedObservability tracks observability metrics received after filtering.
	// These are standard monitoring metrics that don't directly impact cost calculations.
	metricsReceivedObservability = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "metrics_received_observability_total",
			Help: "Total number of observability metrics received",
		},
		[]string{},
	)
)

// MetricCollector is responsible for receiving, processing, and storing Prometheus metrics
// from remote write endpoints. It serves as the primary entry point for metric data
// in the CloudZero agent architecture.
//
// The collector handles both Prometheus remote write protocol versions (v1 and v2),
// decompresses snappy-compressed payloads, applies metric filtering to separate cost
// metrics from observability metrics, and stores them in appropriate storage backends.
//
// Key responsibilities:
//   - Protocol parsing: Supports both Prometheus remote write v1 and v2 formats
//   - Decompression: Handles snappy-compressed request bodies
//   - Metric filtering: Separates metrics based on configured rules
//   - Storage routing: Sends cost metrics and observability metrics to different stores
//   - Performance optimization: Implements immediate flush for first batch of cost metrics
//   - Monitoring: Exposes Prometheus metrics for observability
//
// Usage:
//   collector, err := NewMetricCollector(settings, clock, costStore, observabilityStore)
//   if err != nil { /* handle error */ }
//   defer collector.Close()
//   
//   stats, err := collector.PutMetrics(ctx, contentType, encoding, body)
type MetricCollector struct {
	// settings contains the agent configuration including cluster name and cloud account ID
	settings *config.Settings
	
	// costStore persists cost-related metrics that feed into CloudZero cost analysis
	costStore types.WritableStore
	
	// observabilityStore persists standard monitoring metrics for operational visibility
	observabilityStore types.WritableStore
	
	// filter applies configured rules to separate cost metrics from observability metrics
	filter *MetricFilter
	
	// clock provides testable time operations for metric timestamps
	clock types.TimeProvider
	
	// cancelFunc stops the background cache rotation goroutine
	cancelFunc context.CancelFunc
	
	// initialFlush tracks whether the first batch of cost metrics has been immediately flushed.
	// This optimization ensures cost data reaches CloudZero servers quickly after agent startup.
	initialFlush bool
}

// Settings returns the configuration settings used by this metric collector.
// This method provides access to cluster name, cloud account ID, and other
// configuration values that are embedded in collected metrics.
func (d *MetricCollector) Settings() *config.Settings {
	return d.settings
}

// NewMetricCollector creates a new MetricCollector instance and starts the background
// cache rotation goroutine. The collector is immediately ready to receive metrics
// via PutMetrics calls.
//
// Parameters:
//   - s: Agent configuration settings including cluster info and metric filtering rules
//   - clock: Time provider for testable timestamps (typically types.RealTimeProvider)
//   - costStore: Storage backend for cost-related metrics (typically disk storage)
//   - observabilityStore: Storage backend for monitoring metrics (typically disk storage)
//
// Returns:
//   - *MetricCollector: Ready-to-use collector instance
//   - error: Configuration or initialization error
//
// The collector must be closed with Close() to properly stop background goroutines.
//
// Example:
//   collector, err := NewMetricCollector(settings, types.RealTimeProvider{}, costStore, observabilityStore)
//   if err != nil {
//       return fmt.Errorf("failed to create collector: %w", err)
//   }
//   defer collector.Close()
func NewMetricCollector(s *config.Settings, clock types.TimeProvider, costStore types.WritableStore, observabilityStore types.WritableStore) (*MetricCollector, error) {
	filter, err := NewMetricFilter(&s.Metrics)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	collector := &MetricCollector{
		settings:           s,
		costStore:          costStore,
		observabilityStore: observabilityStore,
		filter:             filter,
		clock:              clock,
		cancelFunc:         cancel,
	}
	go collector.rotateCachePeriodically(ctx)
	return collector, nil
}

// PutMetrics appends metrics and returns write response stats.
func (d *MetricCollector) PutMetrics(ctx context.Context, contentType, encodingType string, body []byte) (*remote.WriteResponseStats, error) {
	var (
		metrics      []types.Metric
		stats        *remote.WriteResponseStats
		decompressed = body
		err          error
	)

	if contentType == "" {
		contentType = appProtoContentType
	}
	contentType, err = parseProtoMsg(contentType)
	if err != nil {
		return nil, err
	}

	if encodingType == SnappyBlockCompression {
		decompressed, err = snappy.Decode(nil, decompressed)
		if err != nil {
			return nil, err
		}
	}

	switch contentType {
	case v1ContentType:
		metrics, err = d.DecodeV1(decompressed)
		if err != nil {
			return nil, ErrJSONUnmarshal
		}
	case v2ContentType:
		metrics, stats, err = d.DecodeV2(decompressed)
		if err != nil {
			return &remote.WriteResponseStats{}, ErrJSONUnmarshal
		}
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}

	costMetrics, observabilityMetrics, droppedMetrics := d.filter.Filter(metrics)

	metricsReceived.WithLabelValues().Add(float64(len(metrics)))
	metricsReceivedCost.WithLabelValues().Add(float64(len(costMetrics)))
	metricsReceivedObservability.WithLabelValues().Add(float64(len(observabilityMetrics)))

	if log.Ctx(ctx).GetLevel() <= zerolog.DebugLevel {
		metricsCount := metricCounter{}
		for _, metric := range costMetrics {
			metricsCount.Add("cost", metric.MetricName)
		}
		for _, metric := range observabilityMetrics {
			metricsCount.Add("observability", metric.MetricName)
		}
		for _, metric := range droppedMetrics {
			metricsCount.Add("dropped", metric.MetricName)
		}

		log.Ctx(ctx).Debug().
			Interface("metricCounts", metricsCount).
			Int("metricsReceived", len(metrics)).
			Int("costMetrics", len(costMetrics)).
			Int("observabilityMetrics", len(observabilityMetrics)).
			Int("droppedMetrics", len(droppedMetrics)).
			Msg("metrics received")
	}

	if costMetrics != nil && d.costStore != nil {
		if err := d.costStore.Put(ctx, costMetrics...); err != nil {
			return stats, err
		}

		// In order to reduce the amount of time until the server starts seeing
		// data, we perform a first flush 🍵 of the cost metrics immediately
		// upon receipt.
		if !d.initialFlush && len(costMetrics) > 0 {
			d.initialFlush = true

			log.Ctx(ctx).Info().Int("count", len(costMetrics)).Msg("first flush of cost metrics")
			if err := d.costStore.Flush(); err != nil {
				return stats, err
			}
		}
	}
	if observabilityMetrics != nil && d.observabilityStore != nil {
		if err := d.observabilityStore.Put(ctx, observabilityMetrics...); err != nil {
			return stats, err
		}
	}
	return stats, nil
}

type metricCounter map[string]map[string]int

func (m metricCounter) Add(metricName string, metricValue string) {
	if _, ok := m[metricName]; !ok {
		m[metricName] = map[string]int{}
	}
	m[metricName][metricValue]++
}

// Flush triggers the flushing of accumulated metrics.
func (d *MetricCollector) Flush(ctx context.Context) error {
	if err := d.costStore.Flush(); err != nil {
		return err
	}
	return d.observabilityStore.Flush()
}

// Close stops the flushing goroutine gracefully.
func (d *MetricCollector) Close() {
	d.cancelFunc()
}

// rotateCachePeriodically runs a background goroutine that flushes metrics at regular intervals.
func (d *MetricCollector) rotateCachePeriodically(ctx context.Context) {
	for range ctx.Done() {
		// Perform a final flush before exiting
		// flushCtx, cancel := context.WithTimeout(context.Background(), d.flushInterval)
		// if err := d.Flush(flushCtx); err != nil {
		// 	log.Ctx(ctx).Err(err).Msg("Error during final flush")
		// }
		// cancel()
		return
	}
}

// parseProtoMsg parses the content type and extracts the proto message version.
func parseProtoMsg(contentType string) (string, error) {
	contentType = strings.TrimSpace(contentType)

	parts := strings.Split(contentType, ";")
	if parts[0] != appProtoContentType {
		return "", fmt.Errorf("expected %v as the first (media) part, got %v content-type", appProtoContentType, contentType)
	}
	// Parse potential https://www.rfc-editor.org/rfc/rfc9110#parameter
	for _, p := range parts[1:] {
		pair := strings.Split(p, "=")
		if len(pair) != 2 {
			return "", fmt.Errorf("as per https://www.rfc-editor.org/rfc/rfc9110#parameter expected parameters to be key-values, got %v in %v content-type", p, contentType)
		}
		if pair[0] == "proto" {
			ret := prom.RemoteWriteProtoMsg(pair[1])
			if err := ret.Validate(); err != nil {
				return "", fmt.Errorf("got %v content type; %w", contentType, err)
			}
			return string(ret), nil
		}
	}
	// No "proto=" parameter, assuming v1.
	return string(prom.RemoteWriteProtoMsgV1), nil
}

func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

// DecodeV1 decompresses and decodes a Protobuf v1 WriteRequest, then converts it to a slice of Metric structs.
func (d *MetricCollector) DecodeV1(data []byte) ([]types.Metric, error) {
	// Parse Protobuf v1 WriteRequest
	var writeReq prompb.WriteRequest
	if err := proto.Unmarshal(data, &writeReq); err != nil {
		return nil, err
	}

	// Convert to []types.Metric
	var metrics []types.Metric
	for _, ts := range writeReq.Timeseries {
		labelsMap := make(map[string]string)

		for _, label := range ts.Labels {
			labelsMap[label.Name] = label.Value
		}

		for _, sample := range ts.Samples {
			metric := types.Metric{
				ID:             uuid.New(),
				ClusterName:    d.settings.ClusterName,
				CloudAccountID: d.settings.CloudAccountID,
				CreatedAt:      d.clock.GetCurrentTime(),
				TimeStamp:      timestamp.Time(sample.Timestamp),
				Value:          formatFloat(sample.Value),
			}
			metric.ImportLabels(labelsMap)

			if len(metric.MetricName) == 0 { // don't save garbage metrics
				continue
			}

			metrics = append(metrics, metric)
		}
	}
	return metrics, nil
}

// DecodeV2 decompresses and decodes a Protobuf v2 WriteRequest, then converts it to a slice of Metric structs and collects stats.
func (d *MetricCollector) DecodeV2(data []byte) ([]types.Metric, *remote.WriteResponseStats, error) {
	// Parse Protobuf v2 WriteRequest
	var writeReq writev2.Request
	if err := proto.Unmarshal(data, &writeReq); err != nil {
		return nil, &remote.WriteResponseStats{}, err
	}

	// Initialize statistics
	stats := remote.WriteResponseStats{}

	// Convert to []types.Metric and update stats
	var metrics []types.Metric
	for _, ts := range writeReq.Timeseries {
		labelsMap := make(map[string]string)

		// Decode labels from LabelsRefs using the symbols array
		for i := 0; i < len(ts.LabelsRefs); i += 2 {
			nameIdx := ts.LabelsRefs[i]
			valueIdx := ts.LabelsRefs[i+1]
			if int(nameIdx) >= len(writeReq.Symbols) || int(valueIdx) >= len(writeReq.Symbols) {
				return nil, &remote.WriteResponseStats{}, errors.New("invalid label reference indices")
			}
			labelName := writeReq.Symbols[nameIdx]
			labelValue := writeReq.Symbols[valueIdx]
			labelsMap[labelName] = labelValue
		}

		// Process samples
		for _, sample := range ts.Samples {
			metric := types.Metric{
				ID:             uuid.New(),
				ClusterName:    d.settings.ClusterName,
				CloudAccountID: d.settings.CloudAccountID,
				CreatedAt:      d.clock.GetCurrentTime(),
				TimeStamp:      timestamp.Time(sample.Timestamp),
				Value:          formatFloat(sample.Value),
			}
			metric.ImportLabels(labelsMap)
			metrics = append(metrics, metric)
			stats.Samples++
		}

		// Process histograms
		stats.Histograms += len(ts.Histograms)
		// Process exemplars
		stats.Exemplars += len(ts.Exemplars)
	}

	// Set Confirmed to true, indicating that statistics are reliable
	stats.Confirmed = true

	return metrics, &stats, nil
}
