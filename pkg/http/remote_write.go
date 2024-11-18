package http

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/golang/protobuf/proto" //nolint:staticcheck
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/rand"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/utils"
)

type RemoteWriter struct {
	writer   storage.DatabaseWriter
	reader   storage.DatabaseReader
	settings *config.Settings
	clock    utils.TimeProvider
}

func NewRemoteWriter(writer storage.DatabaseWriter, reader storage.DatabaseReader, settings *config.Settings) *RemoteWriter {
	return &RemoteWriter{writer: writer, reader: reader, settings: settings, clock: &utils.Clock{}}
}

func (rw *RemoteWriter) StartRemoteWriter() time.Ticker {
	ticker := time.NewTicker(rw.settings.RemoteWrite.SendInterval)

	for range ticker.C {
		rw.Flush()
	}
	return *ticker
}

func (rw *RemoteWriter) Flush() {
	ctx, cancel := context.WithTimeout(context.Background(), rw.settings.RemoteWrite.SendTimeout)
	defer cancel()
	currentTime := rw.clock.GetCurrentTime()
flushLoop:
	for {
		select {
		case <-ctx.Done():
			log.Printf("Flush operation timed out: %v", ctx.Err())
			return
		default:
			log.Debug().Msgf("Starting data upload at %v", currentTime)
			// get a chunk of data to process
			recordsToProcess, err := rw.reader.ReadData(currentTime)
			if err != nil {
				log.Error().Msgf("failed to read data from storage: %v", err)
				break flushLoop
			}
			// if there are no records to process, stop processing
			if len(recordsToProcess) == 0 {
				log.Debug().Msg("Done remote writing records")
				return
			}
			// format metrics to prometheus format
			ts := rw.formatMetrics(recordsToProcess)
			log.Debug().Msgf("Pushing %d records to remote write endpoint", len(ts))

			// push data to remote write
			err = rw.pushMetrics(rw.settings.RemoteWrite.Host, string(rw.settings.RemoteWrite.APIKey), ts)
			if err != nil {
				log.Error().Msgf("failed to push metrics to remote write: %v", err)
				break flushLoop
			}

			// mark records as processed
			updatedRecordCount, err := rw.writer.UpdateSentAtForRecords(recordsToProcess, currentTime)
			if err != nil || updatedRecordCount != int64(len(recordsToProcess)) {
				log.Error().Msgf("failed to update sent_at for records: %v", err)
				break flushLoop
			}
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
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			return nil
		}

		if resp != nil {
			resp.Body.Close()
			log.Error().Msgf("received non-200 response: %v, retrying...", resp.StatusCode)
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
