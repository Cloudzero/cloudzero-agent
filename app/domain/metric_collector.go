package domain

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	prom "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/prompb"
	writev2 "github.com/prometheus/prometheus/prompb/io/prometheus/write/v2"
	"github.com/prometheus/prometheus/storage/remote"

	"github.com/cloudzero/cirrus-remote-write/app/types"
)

var (
	ErrJsonUnmarshal    = errors.New("failed to parse metric from request body")
	ErrMetricIdMismatch = errors.New("metric ID in path does not match product ID in body")
)

const (
	SnappyBlockCompression = "snappy"
	appProtoContentType    = "application/x-protobuf"
)

var (
	v1ContentType = string(prom.RemoteWriteProtoMsgV1)
	v2ContentType = string(prom.RemoteWriteProtoMsgV2)
)

type MetricCollector struct {
	appendable types.Appendable
}

func NewMetricCollector(a types.Appendable) *MetricCollector {
	return &MetricCollector{
		appendable: a,
	}
}

func (d *MetricCollector) PutMetrics(ctx context.Context, contentType, encodingType string, body []byte) (*remote.WriteResponseStats, error) {
	var (
		metrics      []types.Metric
		stats        *remote.WriteResponseStats
		decompressed []byte = body
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
		metrics, err = DecodeV1(decompressed)
		if err != nil {
			return nil, ErrJsonUnmarshal
		}
	case v2ContentType:
		metrics, stats, err = DecodeV2(decompressed)
		if err != nil {
			return &remote.WriteResponseStats{}, ErrJsonUnmarshal
		}
	}

	if err := d.appendable.Put(ctx, metrics...); err != nil {
		return stats, err
	}
	return stats, nil
}

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

// DecodeV1 decompresses and decodes a Protobuf v1 WriteRequest, then converts it to a slice of Metric structs.
func DecodeV1(data []byte) ([]types.Metric, error) {
	// Parse Protobuf v1 WriteRequest
	var writeReq prompb.WriteRequest
	if err := proto.Unmarshal(data, &writeReq); err != nil {
		return nil, err
	}

	// Convert to []types.Metric
	var metrics []types.Metric
	for _, ts := range writeReq.Timeseries {
		labelsMap := make(map[string]string)
		var metricName string

		for _, label := range ts.Labels {
			labelsMap[label.Name] = label.Value
			if label.Name == "__name__" {
				metricName = label.Value
			}
		}

		for _, sample := range ts.Samples {
			metrics = append(metrics, types.NewMetric(
				metricName,
				sample.Timestamp,
				labelsMap,
				strconv.FormatFloat(sample.Value, 'f', -1, 64),
			))
		}
	}
	return metrics, nil
}

func DecodeV2(data []byte) ([]types.Metric, *remote.WriteResponseStats, error) {
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
		metricName := ""

		// Decode labels from LabelsRefs using the symbols array
		for i := 0; i < len(ts.LabelsRefs); i += 2 {
			nameIdx := ts.LabelsRefs[i]
			valueIdx := ts.LabelsRefs[i+1]
			if int(nameIdx) >= len(writeReq.Symbols) || int(valueIdx) >= len(writeReq.Symbols) {
				return nil, &remote.WriteResponseStats{}, fmt.Errorf("invalid label reference indices")
			}
			labelName := writeReq.Symbols[nameIdx]
			labelValue := writeReq.Symbols[valueIdx]
			labelsMap[labelName] = labelValue
			if labelName == "__name__" {
				metricName = labelValue
			}
		}

		// Process samples
		for _, sample := range ts.Samples {
			metric := types.Metric{
				Name:      metricName,
				TimeStamp: sample.Timestamp,
				Labels:    labelsMap,
				Value:     strconv.FormatFloat(sample.Value, 'f', -1, 64),
			}
			metrics = append(metrics, metric)
			stats.Samples++
		}

		// Process histograms
		stats.Histograms = len(ts.Histograms)
		// Process exemplars
		stats.Exemplars = len(ts.Exemplars)
	}

	// Set Confirmed to true, indicating that statistics are reliable
	stats.Confirmed = true

	return metrics, &stats, nil
}
