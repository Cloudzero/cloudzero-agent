// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package domain provides domain logic.
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

var (
	ErrJSONUnmarshal    = errors.New("failed to parse metric from request body")
	ErrMetricIDMismatch = errors.New("metric ID in path does not match product ID in body")
)

const (
	SnappyBlockCompression = "snappy"
	appProtoContentType    = "application/x-protobuf"
)

var (
	v1ContentType = string(prom.RemoteWriteProtoMsgV1)
	v2ContentType = string(prom.RemoteWriteProtoMsgV2)
)

var (
	metricsReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "metrics_received_total",
			Help: "Total number of metrics received",
		},
		[]string{},
	)
	metricsReceivedCost = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "metrics_received_cost_total",
			Help: "Total number of cost metrics received",
		},
		[]string{},
	)
	metricsReceivedObservability = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "metrics_received_observability_total",
			Help: "Total number of observability metrics received",
		},
		[]string{},
	)
)

// MetricCollector is responsible for collecting and flushing metrics.
type MetricCollector struct {
	settings           *config.Settings
	costStore          types.WritableStore
	observabilityStore types.WritableStore
	filter             *MetricFilter
	clock              types.TimeProvider
	cancelFunc         context.CancelFunc
}

// NewMetricCollector creates a new MetricCollector and starts the flushing goroutine.
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

	costMetrics, observabilityMetrics := d.filter.Filter(metrics)

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

		log.Ctx(ctx).Debug().
			Interface("metricCounts", metricsCount).
			Int("metrics", len(metrics)).
			Int("costMetrics", len(costMetrics)).
			Int("observabilityMetrics", len(observabilityMetrics)).
			Msg("metrics received")
	}

	if costMetrics != nil && d.costStore != nil {
		if err := d.costStore.Put(ctx, costMetrics...); err != nil {
			return stats, err
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
