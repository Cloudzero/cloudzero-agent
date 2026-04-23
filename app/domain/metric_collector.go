// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package domain contains the Application Core business logic for the CloudZero Agent.
//
// This package implements the domain layer of hexagonal architecture, containing
// metric collection, filtering, and processing services. It orchestrates the flow
// from Prometheus remote_write ingestion through classification and storage preparation
// for the CloudZero cost allocation platform.
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
	remoteapi "github.com/prometheus/client_golang/exp/api/remote"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/prometheus/prometheus/prompb"
	writev2 "github.com/prometheus/prometheus/prompb/io/prometheus/write/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	config "github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/app/domain/transform"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

// Domain-level errors for metric collection operations.
var (
	// ErrJSONUnmarshal indicates failure to parse metric data from request body.
	// This error occurs when Prometheus remote_write data is malformed or incompatible.
	ErrJSONUnmarshal = errors.New("failed to parse metric from request body")

	// ErrMetricIDMismatch indicates inconsistency between URL path and request body identifiers.
	// This error prevents data corruption in metric processing operations.
	ErrMetricIDMismatch = errors.New("metric ID in path does not match product ID in body")
)

// Protocol and encoding constants for Prometheus remote_write integration.
const (
	// SnappyBlockCompression identifies snappy compression used in Prometheus remote_write protocol.
	SnappyBlockCompression = "snappy"

	// appProtoContentType is the default content type for protobuf-encoded metric data.
	appProtoContentType = "application/x-protobuf"
)

// Prometheus remote_write protocol version content types.
// These correspond to the official Prometheus remote_write specification.
var (
	// v1ContentType identifies Prometheus remote_write v1 protocol format.
	v1ContentType = string(remoteapi.WriteV1MessageType)

	// v2ContentType identifies Prometheus remote_write v2 protocol format.
	v2ContentType = string(remoteapi.WriteV2MessageType)
)

// Prometheus metrics for monitoring the collector's ingestion and classification performance.
// These counters provide visibility into the volume and categorization of incoming metrics.
var (
	// metricsReceived tracks the total number of metrics ingested from all Prometheus remote_write requests.
	// This counter helps monitor the overall collection volume and detect ingestion issues.
	metricsReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "metrics_received_total",
			Help: "Total number of metrics received",
		},
		[]string{},
	)

	// metricsReceivedCost tracks metrics classified as cost-related and sent to the cost storage pipeline.
	// These metrics support CloudZero's core cost allocation and billing analysis functionality.
	metricsReceivedCost = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "metrics_received_cost_total",
			Help: "Total number of cost metrics received",
		},
		[]string{},
	)

	// metricsReceivedObservability tracks metrics classified as observability-focused rather than cost-related.
	// These metrics are processed separately from cost data to optimize storage and processing efficiency.
	metricsReceivedObservability = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "metrics_received_observability_total",
			Help: "Total number of observability metrics received",
		},
		[]string{},
	)
)

// MetricCollector orchestrates the core metric collection and classification pipeline.
// This service receives Prometheus remote_write requests, classifies metrics into cost vs observability
// categories, and routes them to appropriate storage backends for the CloudZero platform.
type MetricCollector struct {
	// settings contains collector configuration including metric filtering rules and storage parameters.
	settings *config.Settings

	// costStore handles storage of metrics classified as cost-related for billing analysis.
	costStore types.WritableStore

	// observabilityStore handles storage of metrics classified as observability-focused.
	observabilityStore types.WritableStore

	// filter implements metric classification logic to separate cost from observability metrics.
	filter *MetricFilter

	// transformer handles vendor-specific metric transformation (e.g., DCGM GPU metrics).
	transformer types.MetricTransformer

	// clock provides time abstraction for testing and consistent timestamping.
	clock types.TimeProvider

	// cancelFunc enables graceful shutdown of background processing goroutines.
	cancelFunc context.CancelFunc

	// initialFlush indicates whether the first automatic flush cycle has completed.
	initialFlush bool
}

// Settings returns the collector configuration for external inspection and validation.
// This method provides read-only access to the collector's operational parameters.
func (d *MetricCollector) Settings() *config.Settings {
	return d.settings
}

// NewMetricCollector creates a MetricCollector and initializes the background flush cycle.
// The collector starts accepting Prometheus remote_write requests immediately and begins
// periodic flushing of buffered metrics to storage backends based on configuration.
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
		transformer:        transform.NewMetricTransformer(),
		clock:              clock,
		cancelFunc:         cancel,
	}
	go collector.rotateCachePeriodically(ctx)
	return collector, nil
}

// PutMetrics processes a Prometheus remote_write request and stores classified metrics.
// This method handles decompression, protocol version detection, metric classification,
// and routing to appropriate storage backends while maintaining compatibility statistics.
func (d *MetricCollector) PutMetrics(ctx context.Context, contentType, encodingType string, body []byte) (*WriteResponseStats, error) {
	var (
		metrics      []types.Metric
		stats        *WriteResponseStats
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
			return &WriteResponseStats{}, ErrJSONUnmarshal
		}
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}

	// Log complete DCGM metrics for debugging GPU transformation
	for _, metric := range metrics {
		if strings.HasPrefix(metric.MetricName, "DCGM_FI_DEV_") {
			log.Ctx(ctx).Info().
				Str("metricName", metric.MetricName).
				Str("value", metric.Value).
				Str("nodeName", metric.NodeName).
				Interface("labels", metric.Labels).
				Time("timestamp", metric.TimeStamp).
				Str("clusterName", metric.ClusterName).
				Str("cloudAccountID", metric.CloudAccountID).
				Msg("DCGM metric received")
		}
	}

	// Transform vendor-specific metrics (e.g., DCGM GPU metrics) before filtering
	metrics, err = d.transformer.Transform(ctx, metrics)
	if err != nil {
		return stats, fmt.Errorf("failed to transform metrics: %w", err)
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
			ret := remoteapi.WriteMessageType(pair[1])
			if err := ret.Validate(); err != nil {
				return "", fmt.Errorf("got %v content type; %w", contentType, err)
			}
			return string(ret), nil
		}
	}
	// No "proto=" parameter, assuming v1.
	return string(remoteapi.WriteV1MessageType), nil
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
func (d *MetricCollector) DecodeV2(data []byte) ([]types.Metric, *WriteResponseStats, error) {
	// Parse Protobuf v2 WriteRequest
	var writeReq writev2.Request
	if err := proto.Unmarshal(data, &writeReq); err != nil {
		return nil, &WriteResponseStats{}, err
	}

	// Initialize statistics
	stats := WriteResponseStats{}

	// Convert to []types.Metric and update stats
	var metrics []types.Metric
	for _, ts := range writeReq.Timeseries {
		labelsMap := make(map[string]string)

		// Decode labels from LabelsRefs using the symbols array
		for i := 0; i < len(ts.LabelsRefs); i += 2 {
			nameIdx := ts.LabelsRefs[i]
			valueIdx := ts.LabelsRefs[i+1]
			if int(nameIdx) >= len(writeReq.Symbols) || int(valueIdx) >= len(writeReq.Symbols) {
				return nil, &WriteResponseStats{}, errors.New("invalid label reference indices")
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
