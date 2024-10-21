package http

import (
	"fmt"
	"time"

	"github.com/prometheus/prometheus/prompb"
	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
)

type RemoteWriter struct {
	writer   storage.DatabaseWriter
	reader   storage.DatabaseReader
	settings *config.Settings
}

func NewRemoteWriter(writer storage.DatabaseWriter, reader storage.DatabaseReader, settings *config.Settings) *RemoteWriter {
	return &RemoteWriter{writer: writer, reader: reader, settings: settings}
}

func (rw *RemoteWriter) StartRemoteWriter() time.Ticker {
	ticker := time.NewTicker(rw.settings.RemoteWrite.SendInterval)

	for range ticker.C {
		currentTime := time.Now().UTC()
		for {
			log.Debug().Msgf("Starting data upload at %v", currentTime)
			// get a chunk of data to process
			recordsToProcess, err := rw.reader.ReadData(currentTime)
			if err != nil {
				log.Error().Msgf("failed to read data from storage: %v", err)
				break
			}
			// if there are no records to process, stop processing
			if len(recordsToProcess) == 0 {
				log.Debug().Msg("Done remote writing records")
				break
			}
			// format metrics to prometheus format
			ts := rw.formatMetrics(recordsToProcess)
			log.Debug().Msgf("Pushing %d records to remote write", len(ts))

			// push data to remote write - todo: make this part of this struct, add retry logic
			err = pushMetrics(rw.settings.RemoteWrite.Host, string(rw.settings.RemoteWrite.APIKey), ts)
			if err != nil {
				log.Error().Msgf("failed to push metrics to remote write: %v", err)
				break
			}

			// mark records as processed
			err = rw.writer.UpdateSentAtForRecords(recordsToProcess, currentTime)
			if err != nil {
				log.Error().Msgf("failed to update sent_at for records: %v", err)
			}
		}
	}
	return *ticker
}

func (rw *RemoteWriter) formatMetrics(records []storage.ResourceTags) []prompb.TimeSeries {
	timeSeries := []prompb.TimeSeries{}
	for _, record := range records {
		metricName := rw.constructMetricTagName(record, "labels")
		recordCreatedOrUpdated := rw.maxTime(record.UpdatedAt, record.CreatedAt)
		timeSeries = append(timeSeries, rw.createTimeseries(metricName, *record.Labels, *record.MetricLabels, recordCreatedOrUpdated))
		if record.Annotations != nil {
			metricName := rw.constructMetricTagName(record, "annotations")
			timeSeries = append(timeSeries, rw.createTimeseries(metricName, *record.Annotations, *record.MetricLabels, recordCreatedOrUpdated))
		}

	}
	return timeSeries
}

func (rw *RemoteWriter) constructMetricTagName(record storage.ResourceTags, metricType string) string {
	return fmt.Sprintf("kube_%s_%s", config.ResourceTypeToMetricName[record.Type], metricType)
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

func (rw *RemoteWriter) maxTime(t1, t2 time.Time) time.Time {
	if t1.After(t2) {
		return t1
	}
	return t2
}
