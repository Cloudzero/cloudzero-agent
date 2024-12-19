// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package pusher provides a mechanism for pushing metrics to a remote write endpoint.
// It periodically collects metrics from a resource store, formats them, and sends them to the specified endpoint.
// The package includes Prometheus metrics to monitor the performance and success of the remote write operations.
// It supports configuration for send intervals, timeouts, maximum payload sizes, and retry logic.
package pusher

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/prompb"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/rand"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/types"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/utils"
)

// -------------------- Prometheus Metrics --------------------
var (
	remoteWriteStatsOnce sync.Once
	// RemoteWriteTimeseriesSent counts the number of timeseries sent.
	RemoteWriteTimeseriesSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "remote_write_timeseries_total",
			Help: "Total number of timeseries attempted to be sent to remote write endpoint",
		},
		[]string{"endpoint"},
	)

	// RemoteWriteRequestDuration measures the duration of requests to remote write endpoint.
	RemoteWriteRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "remote_write_request_duration_seconds",
			Help:    "Histogram of request durations to remote write endpoint",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)

	// RemoteWriteResponseCodes counts the response codes returned by remote write endpoint.
	RemoteWriteResponseCodes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "remote_write_response_codes_total",
			Help: "Count of response codes from remote write endpoint",
		},
		[]string{"endpoint", "status_code"},
	)

	// RemoteWritePayloadSizeBytes measures the payload size of requests in bytes.
	RemoteWritePayloadSizeBytes = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "remote_write_payload_size_bytes",
			Help:    "Size of payloads sent to remote write endpoint in bytes",
			Buckets: prometheus.ExponentialBuckets(256, 2, 10),
		},
		[]string{"endpoint"},
	)

	// RemoteWriteFailures counts how many times the remote write fails.
	RemoteWriteFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "remote_write_failures_total",
			Help: "Total number of failed attempts to write metrics to the remote endpoint",
		},
		[]string{"endpoint"},
	)

	// RemoteWriteBacklog tracks how many records are waiting to be sent to the remote write endpoint.
	RemoteWriteBacklog = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "remote_write_backlog_records",
			Help: "Number of records that are currently waiting to be sent to the remote write endpoint",
		},
		[]string{"endpoint"},
	)

	// Tracks how many records have been successfully updated in the database after sending
	RemoteWriteRecordsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "remote_write_records_processed_total",
			Help: "Total number of records successfully processed (sent and marked as sent_at)",
		},
		[]string{"endpoint"},
	)

	// Tracks how many times updating sent_at in the database fails
	RemoteWriteDBFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "remote_write_db_failures_total",
			Help: "Total number of failures when updating sent_at for records in the database",
		},
		[]string{"endpoint"},
	)
)

// MetricsPusher is a runnable that periodically flushes metrics to a remote write endpoint.
type MetricsPusher struct {
	// interfaces
	clock types.TimeProvider
	store types.ResourceStore

	// settings
	sendTimeout  time.Duration
	sendInterval time.Duration
	sentMaxBytes int
	maxRetries   int
	settings     *config.Settings

	// flow controle
	originalCtx context.Context
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
	running     bool
	done        chan struct{}
}

func New(
	ctx context.Context,
	store types.ResourceStore,
	clock types.TimeProvider,
	settings *config.Settings,
) types.Runnable {
	remoteWriteStatsOnce.Do(func() {
		prometheus.MustRegister(
			RemoteWriteTimeseriesSent,
			RemoteWriteRequestDuration,
			RemoteWriteResponseCodes,
			RemoteWritePayloadSizeBytes,
			RemoteWriteFailures,
			RemoteWriteBacklog,
			RemoteWriteRecordsProcessed,
			RemoteWriteDBFailures,
		)
	})
	newCtx, cancel := context.WithCancel(ctx)
	return &MetricsPusher{
		settings:     settings,
		originalCtx:  ctx,
		ctx:          newCtx,
		cancel:       cancel,
		done:         make(chan struct{}),
		clock:        clock,
		store:        store,
		sendTimeout:  settings.RemoteWrite.SendTimeout,
		sendInterval: settings.RemoteWrite.SendInterval,
		sentMaxBytes: settings.RemoteWrite.MaxBytesPerSend,
		maxRetries:   settings.RemoteWrite.MaxRetries,
	}
}

func (h *MetricsPusher) ResetStats() {
	// reset all metrics on start
	RemoteWriteTimeseriesSent.Reset()
	RemoteWriteRequestDuration.Reset()
	RemoteWriteResponseCodes.Reset()
	RemoteWritePayloadSizeBytes.Reset()
	RemoteWriteFailures.Reset()
	RemoteWriteBacklog.Reset()
	RemoteWriteRecordsProcessed.Reset()
	RemoteWriteDBFailures.Reset()
}

func (h *MetricsPusher) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.running {
		return nil
	}

	ticker := time.NewTicker(h.sendInterval)
	go func() {
		defer ticker.Stop()
		defer close(h.done)
		h.ResetStats()
		defer func() {
			if r := recover(); r != nil {
				log.Info().Msgf("Recovered from panic in stale data removal: %v", r)
			}
		}()

		for {
			select {
			case <-h.ctx.Done():
				// ensure a final flush on shutdown
				h.Flush()
				h.running = false
				return
			case <-ticker.C:
				h.Flush()
			}
		}
	}()
	h.running = true
	return nil
}

func (h *MetricsPusher) Shutdown() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.running {
		return nil
	}
	h.cancel()
	<-h.done
	h.reset()
	return nil
}

func (h *MetricsPusher) reset() {
	h.running = false
	ctx, cancel := context.WithCancel(h.originalCtx)
	h.ctx = ctx
	h.cancel = cancel
	h.done = make(chan struct{})
}

func (h *MetricsPusher) IsRunning() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.running
}

func (h *MetricsPusher) sendBatch(batch []*types.ResourceTags) error {
	if len(batch) == 0 {
		return nil
	}

	endpoint := h.settings.RemoteWrite.Host
	apiToken := h.settings.GetAPIKey()
	if apiToken == "" {
		RemoteWriteFailures.WithLabelValues(endpoint).Inc()
		return errors.New("API key is empty")
	}

	ts := h.formatMetrics(batch)
	log.Debug().Msgf("Pushing %d records to remote write endpoint", len(ts))

	if err := h.pushMetrics(h.settings.RemoteWrite.Host, apiToken, ts); err != nil {
		RemoteWriteFailures.WithLabelValues(endpoint).Inc()
		return fmt.Errorf("failed to push metrics to remote write: %v", err)
	}

	RemoteWriteRecordsProcessed.WithLabelValues(endpoint).Add(float64(len(batch)))
	RemoteWriteTimeseriesSent.WithLabelValues(endpoint).Add(float64(len(ts)))
	return nil
}

func (h *MetricsPusher) Flush() error {
	currentTime := h.clock.GetCurrentTime()
	ctf := utils.FormatForStorage(currentTime)
	whereClause := fmt.Sprintf(`
		(record_updated < '%[1]s' AND record_created < '%[1]s' AND sent_at IS NULL)
		OR
		(sent_at IS NOT NULL AND record_updated > sent_at)
		`, ctf)
	found, err := h.store.FindAllBy(h.ctx, whereClause)
	if err != nil {
		RemoteWriteDBFailures.WithLabelValues(h.settings.RemoteWrite.Host).Inc()
		return fmt.Errorf("failed to find records to send: %v", err)
	}

	totalSize := 0
	batch := []*types.ResourceTags{}
	completed := []*types.ResourceTags{}
	for len(found) > 0 {
		next := found[0]
		found = found[1:] // pop
		RemoteWriteBacklog.WithLabelValues(h.settings.RemoteWrite.Host).Set(float64(len(found)))

		if next.Size+totalSize > h.sentMaxBytes && len(batch) > 0 {
			// Send the current batch
			if err := h.sendBatch(batch); err != nil {
				log.Error().Msgf("failed to send batch: %v", err)
				return err
			}

			// Reset totalSize and batch
			batch = []*types.ResourceTags{}
			totalSize = 0
		}

		completed = append(completed, next)
		batch = append(batch, next)
		totalSize += next.Size
	}

	// Send the last batch if it exists
	if len(batch) > 0 {
		if err := h.sendBatch(batch); err != nil {
			log.Error().Msgf("failed to send partial batch: %v", err)
			return err
		}
	}

	if len(completed) > 0 {
		if err := h.store.Tx(h.ctx, func(txCtx context.Context) error {
			for _, record := range completed {
				record.SentAt = &currentTime
				if err := h.store.Update(txCtx, record); err != nil {
					RemoteWriteDBFailures.WithLabelValues(h.settings.RemoteWrite.Host).Inc()
					return fmt.Errorf("failed to update sent_at for record: %v", err)
				}
			}
			return nil
		}); err != nil {
			log.Error().Msgf("failed to update sent_at for records: %v", err)
			RemoteWriteDBFailures.WithLabelValues(h.settings.RemoteWrite.Host).Inc()
			return fmt.Errorf("failed to update sent_at for records: %v", err)
		}
	}
	return nil
}

func (h *MetricsPusher) formatMetrics(records []*types.ResourceTags) []prompb.TimeSeries {
	timeSeries := []prompb.TimeSeries{}
	for _, record := range records {
		metricName := h.constructMetricTagName(record, "labels")
		recordCreatedOrUpdated := h.maxTime(record.RecordUpdated, record.RecordCreated)
		timeSeries = append(timeSeries, h.createTimeseries(metricName, *record.Labels, *record.MetricLabels, recordCreatedOrUpdated))
		if record.Annotations != nil {
			metricName := h.constructMetricTagName(record, "annotations")
			timeSeries = append(timeSeries, h.createTimeseries(metricName, *record.Annotations, *record.MetricLabels, recordCreatedOrUpdated))
		}
	}
	return timeSeries
}

func (h *MetricsPusher) constructMetricTagName(record *types.ResourceTags, metricType string) string {
	return fmt.Sprintf("cloudzero_%s_%s", config.ResourceTypeToMetricName[record.Type], metricType)
}

func (h *MetricsPusher) createTimeseries(
	metricName string, metricTags config.MetricLabelTags,
	additionalMetricLabels config.MetricLabels,
	recordCreatedOrUpdated time.Time,
) prompb.TimeSeries {
	ts := prompb.TimeSeries{
		Labels: []prompb.Label{
			{
				Name:  "__name__",
				Value: metricName,
			},
		},
		Samples: []prompb.Sample{
			{
				Value:     1,
				Timestamp: recordCreatedOrUpdated.UnixNano() / int64(time.Millisecond),
			},
		},
	}
	for labelKey, labelValue := range additionalMetricLabels {
		ts.Labels = append(ts.Labels, prompb.Label{
			Name:  labelKey,
			Value: labelValue,
		})
	}
	for labelKey, labelValue := range metricTags {
		ts.Labels = append(ts.Labels, prompb.Label{
			Name:  "label_" + labelKey,
			Value: labelValue,
		})
	}

	return ts
}

func (h *MetricsPusher) pushMetrics(remoteWriteURL string, apiKey string, timeSeries []prompb.TimeSeries) error {
	writeRequest := &prompb.WriteRequest{
		Timeseries: timeSeries,
	}

	data, err := proto.Marshal(protoadapt.MessageV2Of(writeRequest))
	if err != nil {
		return fmt.Errorf("error marshaling WriteRequest: %v", err)
	}

	compressed := snappy.Encode(nil, data)

	endpoint := remoteWriteURL
	start := time.Now()

	// Instrument: Observe payload size
	RemoteWritePayloadSizeBytes.WithLabelValues(endpoint).Observe(float64(len(compressed)))

	var resp *http.Response
	var req *http.Request

	for attempt := range h.maxRetries {
		ctx, cancel := context.WithTimeout(h.ctx, h.sendTimeout)
		defer cancel()

		req, err = http.NewRequestWithContext(ctx, "POST", remoteWriteURL, bytes.NewBuffer(compressed))
		if err != nil {
			return fmt.Errorf("error creating HTTP request: %v", err)
		}

		req.Header.Set("Content-Type", "application/x-protobuf")
		req.Header.Set("Content-Encoding", "snappy")
		req.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{}
		resp, err = client.Do(req)

		// Instrument: measure duration after each attempt
		duration := time.Since(start).Seconds()
		RemoteWriteRequestDuration.WithLabelValues(endpoint).Observe(duration)

		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			// Instrument: response code 200
			RemoteWriteResponseCodes.WithLabelValues(endpoint, "200").Inc()
			return nil
		}

		if resp != nil {
			statusCode := strconv.Itoa(resp.StatusCode)
			RemoteWriteResponseCodes.WithLabelValues(endpoint, statusCode).Inc()
			resp.Body.Close()
			log.Error().Msgf("received non-200 response: %v, retrying...", resp.StatusCode)
		} else {
			// If resp is nil, we can track it as a failure as well
			RemoteWriteResponseCodes.WithLabelValues(endpoint, "no_response").Inc()
		}

		backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
		jitter := time.Duration(rand.Int63n(int64(time.Second)))
		time.Sleep(backoff + jitter)
	}

	return fmt.Errorf("received non-200 response: %v after %d retries", err, h.maxRetries)
}

func (h *MetricsPusher) maxTime(t1, t2 time.Time) time.Time {
	if t1.After(t2) {
		return t1
	}
	return t2
}
