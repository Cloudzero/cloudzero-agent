package http

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/golang/snappy"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/prompb"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/rand"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/utils"
)

// -------------------- Prometheus Metrics --------------------
var (
	remoteWriteStatsOnce sync.Once
	// remoteWriteTimeseriesSent counts the number of timeseries sent.
	remoteWriteTimeseriesSent = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "remote_write_timeseries_total",
			Help: "Total number of timeseries attempted to be sent to remote write endpoint",
		},
		[]string{"endpoint"},
	)

	// remoteWriteRequestDuration measures the duration of requests to remote write endpoint.
	remoteWriteRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "remote_write_request_duration_seconds",
			Help:    "Histogram of request durations to remote write endpoint",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"endpoint"},
	)

	// remoteWriteResponseCodes counts the response codes returned by remote write endpoint.
	remoteWriteResponseCodes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "remote_write_response_codes_total",
			Help: "Count of response codes from remote write endpoint",
		},
		[]string{"endpoint", "status_code"},
	)

	// remoteWritePayloadSizeBytes measures the payload size of requests in bytes.
	remoteWritePayloadSizeBytes = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "remote_write_payload_size_bytes",
			Help:    "Size of payloads sent to remote write endpoint in bytes",
			Buckets: prometheus.ExponentialBuckets(256, 2, 10),
		},
		[]string{"endpoint"},
	)

	// remoteWriteFailures counts how many times the remote write fails.
	remoteWriteFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "remote_write_failures_total",
			Help: "Total number of failed attempts to write metrics to the remote endpoint",
		},
		[]string{"endpoint"},
	)

	// remoteWriteBacklog tracks how many records are waiting to be sent to the remote write endpoint.
	remoteWriteBacklog = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "remote_write_backlog_records",
			Help: "Number of records that are currently waiting to be sent to the remote write endpoint",
		},
		[]string{"endpoint"},
	)

	// Tracks how many records have been successfully updated in the database after sending
	remoteWriteRecordsProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "remote_write_records_processed_total",
			Help: "Total number of records successfully processed (sent and marked as sent_at)",
		},
		[]string{"endpoint"},
	)

	// Tracks how many times updating sent_at in the database fails
	remoteWriteDBFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "remote_write_db_failures_total",
			Help: "Total number of failures when updating sent_at for records in the database",
		},
		[]string{"endpoint"},
	)
)

// -------------------- RemoteWriter Struct --------------------
type RemoteWriter struct {
	writer   storage.DatabaseWriter
	reader   storage.DatabaseReader
	settings *config.Settings
	clock    utils.TimeProvider
}

func NewRemoteWriter(writer storage.DatabaseWriter, reader storage.DatabaseReader, settings *config.Settings) *RemoteWriter {
	remoteWriteStatsOnce.Do(func() {
		prometheus.MustRegister(
			remoteWriteTimeseriesSent,
			remoteWriteRequestDuration,
			remoteWriteResponseCodes,
			remoteWritePayloadSizeBytes,
			remoteWriteFailures,
			remoteWriteBacklog,
			remoteWriteRecordsProcessed,
			remoteWriteDBFailures,
		)
	})
	return &RemoteWriter{writer: writer, reader: reader, settings: settings, clock: &utils.Clock{}}
}

func (rw *RemoteWriter) StartRemoteWriter() time.Ticker {
	// reset all metrics on start
	remoteWriteTimeseriesSent.Reset()
	remoteWriteRequestDuration.Reset()
	remoteWriteResponseCodes.Reset()
	remoteWritePayloadSizeBytes.Reset()
	remoteWriteFailures.Reset()
	remoteWriteBacklog.Reset()
	remoteWriteRecordsProcessed.Reset()
	remoteWriteDBFailures.Reset()

	ticker := time.NewTicker(rw.settings.RemoteWrite.SendInterval)

	for range ticker.C {
		rw.Flush()
	}
	return *ticker
}

func (rw *RemoteWriter) Flush() error {
	ctx, cancel := context.WithTimeout(context.Background(), rw.settings.RemoteWrite.SendTimeout)
	defer cancel()
	currentTime := rw.clock.GetCurrentTime()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Flush operation timed out: %v", ctx.Err())
			return ctx.Err()
		default:
			log.Debug().Msgf("Starting data upload at %v", currentTime)

			recordsToProcess, err := rw.reader.ReadData(currentTime)
			if err != nil {
				log.Error().Msgf("failed to read data from storage: %v", err)
				return err
			}

			// Update the backlog gauge
			endpoint := rw.settings.RemoteWrite.Host
			remoteWriteBacklog.WithLabelValues(endpoint).Set(float64(len(recordsToProcess)))

			// if there are no records to process, stop processing
			if len(recordsToProcess) == 0 {
				log.Debug().Msg("Done remote writing records")
				return nil
			}

			apiToken := rw.settings.GetAPIKey()
			if apiToken == "" {
				log.Error().Msg("API key is empty")
				remoteWriteFailures.WithLabelValues(endpoint).Inc()
				return fmt.Errorf("API key is empty")
			}

			ts := rw.formatMetrics(recordsToProcess)
			log.Debug().Msgf("Pushing %d records to remote write endpoint", len(ts))

			// Attempt to push metrics
			err = rw.pushMetrics(rw.settings.RemoteWrite.Host, apiToken, ts)
			if err != nil {
				log.Error().Msgf("failed to push metrics to remote write: %v", err)
				// Increment the failure counter since it was not successful
				remoteWriteFailures.WithLabelValues(endpoint).Inc()
				return err
			}

			// If we reach here, pushMetrics was successful, so increment the timeseries sent metric
			remoteWriteTimeseriesSent.WithLabelValues(endpoint).Add(float64(len(ts)))

			// Update sent_at for the records
			updatedRecordCount, err := rw.writer.UpdateSentAtForRecords(recordsToProcess, currentTime)
			if err != nil || updatedRecordCount != int64(len(recordsToProcess)) {
				log.Error().Msgf("failed to update sent_at for records: %v", err)

				// Increment the database failure counter
				remoteWriteDBFailures.WithLabelValues(endpoint).Inc()

				return err
			}

			// If sent_at update succeeded for all records, increment the processed counter
			remoteWriteRecordsProcessed.WithLabelValues(endpoint).Add(float64(updatedRecordCount))

		}
	}
}

func (rw *RemoteWriter) formatMetrics(records []storage.ResourceTags) []prompb.TimeSeries {
	timeSeries := []prompb.TimeSeries{}
	for _, record := range records {
		metricName := rw.constructMetricTagName(record, "labels")
		recordCreatedOrUpdated := rw.maxTime(record.RecordUpdated, record.RecordCreated)
		timeSeries = append(timeSeries, rw.createTimeseries(metricName, *record.Labels, *record.MetricLabels, recordCreatedOrUpdated))
		if record.Annotations != nil {
			metricName := rw.constructMetricTagName(record, "annotations")
			timeSeries = append(timeSeries, rw.createTimeseries(metricName, *record.Annotations, *record.MetricLabels, recordCreatedOrUpdated))
		}
	}
	return timeSeries
}

func (rw *RemoteWriter) constructMetricTagName(record storage.ResourceTags, metricType string) string {
	return fmt.Sprintf("cloudzero_%s_%s", config.ResourceTypeToMetricName[record.Type], metricType)
}

func (rw *RemoteWriter) createTimeseries(metricName string, metricTags config.MetricLabelTags, additionalMetricLabels config.MetricLabels, recordCreatedOrUpdated time.Time) prompb.TimeSeries {
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
			Name:  fmt.Sprintf("label_%s", labelKey),
			Value: labelValue,
		})
	}

	return ts
}

func (rw *RemoteWriter) pushMetrics(remoteWriteURL string, apiKey string, timeSeries []prompb.TimeSeries) error {
	writeRequest := &prompb.WriteRequest{
		Timeseries: timeSeries,
	}

	data, err := proto.Marshal(writeRequest)
	if err != nil {
		return fmt.Errorf("error marshaling WriteRequest: %v", err)
	}

	compressed := snappy.Encode(nil, data)

	endpoint := remoteWriteURL
	start := time.Now()

	// Instrument: Observe payload size
	remoteWritePayloadSizeBytes.WithLabelValues(endpoint).Observe(float64(len(compressed)))

	var resp *http.Response
	var req *http.Request

	for attempt := 0; attempt < rw.settings.RemoteWrite.MaxRetries; attempt++ {
		req, err = http.NewRequest("POST", remoteWriteURL, bytes.NewBuffer(compressed))
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
		remoteWriteRequestDuration.WithLabelValues(endpoint).Observe(duration)

		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			// Instrument: response code 200
			remoteWriteResponseCodes.WithLabelValues(endpoint, "200").Inc()
			return nil
		}

		if resp != nil {
			statusCode := fmt.Sprintf("%d", resp.StatusCode)
			remoteWriteResponseCodes.WithLabelValues(endpoint, statusCode).Inc()
			resp.Body.Close()
			log.Error().Msgf("received non-200 response: %v, retrying...", resp.StatusCode)
		} else {
			// If resp is nil, we can track it as a failure as well
			remoteWriteResponseCodes.WithLabelValues(endpoint, "no_response").Inc()
		}

		backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
		jitter := time.Duration(rand.Int63n(int64(time.Second)))
		time.Sleep(backoff + jitter)
	}

	return fmt.Errorf("received non-200 response: %v after %d retries", err, rw.settings.RemoteWrite.MaxRetries)
}

func (rw *RemoteWriter) maxTime(t1, t2 time.Time) time.Time {
	if t1.After(t2) {
		return t1
	}
	return t2
}
