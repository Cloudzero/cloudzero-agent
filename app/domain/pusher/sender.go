// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package pusher

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"time"

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

// FormatMetrics converts ResourceTags records into Prometheus TimeSeries
// suitable for remote_write. Each record produces one labels timeseries
// and optionally one annotations timeseries.
func FormatMetrics(records []*types.ResourceTags) []prompb.TimeSeries {
	timeSeries := []prompb.TimeSeries{}
	for _, record := range records {
		metricName := fmt.Sprintf("cloudzero_%s_labels", config.ResourceTypeToMetricName[record.Type])
		recordTime := maxTime(record.RecordUpdated, record.RecordCreated)
		timeSeries = append(timeSeries, createTimeseries(metricName, *record.Labels, *record.MetricLabels, recordTime))
		if record.Annotations != nil {
			metricName := fmt.Sprintf("cloudzero_%s_annotations", config.ResourceTypeToMetricName[record.Type])
			timeSeries = append(timeSeries, createTimeseries(metricName, *record.Annotations, *record.MetricLabels, recordTime))
		}
	}
	return timeSeries
}

// PushMetrics serializes timeseries to a snappy-compressed protobuf and
// POSTs them to the remote_write endpoint with retry + exponential backoff.
func PushMetrics(ctx context.Context, url string, timeSeries []prompb.TimeSeries, maxRetries int, sendTimeout time.Duration) error {
	writeRequest := &prompb.WriteRequest{
		Timeseries: timeSeries,
	}

	data, err := proto.Marshal(protoadapt.MessageV2Of(writeRequest))
	if err != nil {
		return fmt.Errorf("error marshaling WriteRequest: %v", err)
	}

	compressed := snappy.Encode(nil, data)

	endpoint := url
	start := time.Now()

	RemoteWritePayloadSizeBytes.WithLabelValues(endpoint).Observe(float64(len(compressed)))

	var resp *http.Response
	var req *http.Request

	for attempt := range maxRetries {
		reqCtx, cancel := context.WithTimeout(ctx, sendTimeout)
		defer cancel()

		req, err = http.NewRequestWithContext(reqCtx, "POST", url, bytes.NewBuffer(compressed))
		if err != nil {
			return fmt.Errorf("error creating HTTP request: %v", err)
		}

		req.Header.Set("Content-Type", "application/x-protobuf")
		req.Header.Set("Content-Encoding", "snappy")

		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			log.Ctx(reqCtx).Err(err).Msg("post metric failure")
		}

		duration := time.Since(start).Seconds()
		RemoteWriteRequestDuration.WithLabelValues(endpoint).Observe(duration)

		if err == nil && (resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices) {
			defer resp.Body.Close()
			RemoteWriteResponseCodes.WithLabelValues(endpoint, "2xx").Inc()
			return nil
		}

		if resp != nil {
			statusCode := strconv.Itoa(resp.StatusCode)
			RemoteWriteResponseCodes.WithLabelValues(endpoint, statusCode).Inc()
			resp.Body.Close()
			log.Ctx(ctx).Error().
				Int("statusCode", resp.StatusCode).
				Str("statusText", resp.Status).
				Msg("Received non-2xx response, retrying...")
		} else {
			RemoteWriteResponseCodes.WithLabelValues(endpoint, "no_response").Inc()
		}
		backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
		jitter := time.Duration(rand.Int64N(int64(time.Second))) //nolint:gosec // cryptographically secure PRNG is not necessary for backoff jitter
		time.Sleep(backoff + jitter)
	}

	return fmt.Errorf("received non-2xx response: %v after %d retries", err, maxRetries)
}

func createTimeseries(
	metricName string, metricTags config.MetricLabelTags,
	additionalMetricLabels config.MetricLabels,
	recordTime time.Time,
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
				Timestamp: recordTime.UnixNano() / int64(time.Millisecond),
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

func maxTime(t1, t2 time.Time) time.Time {
	if t1.After(t2) {
		return t1
	}
	return t2
}
