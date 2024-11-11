package domain

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	prom "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/prompb"
	writev2 "github.com/prometheus/prometheus/prompb/io/prometheus/write/v2"
	"github.com/prometheus/prometheus/storage/remote"
	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cirrus-remote-write/app/config"
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

// MetricCollector is responsible for collecting and flushing metrics.
type MetricCollector struct {
	settings   *config.Settings
	appendable types.Appendable
	cancelFunc context.CancelFunc
}

// NewMetricCollector creates a new MetricCollector and starts the flushing goroutine.
func NewMetricCollector(s *config.Settings, a types.Appendable) *MetricCollector {

	if s.Cloudzero.RotateInterval <= 0 {
		s.Cloudzero.RotateInterval = 10 * time.Minute
	}

	ctx, cancel := context.WithCancel(context.Background())
	collector := &MetricCollector{
		settings:   s,
		appendable: a,
		cancelFunc: cancel,
	}
	go collector.rotateCachePeriodically(ctx)
	return collector
}

// PutMetrics appends metrics and returns write response stats.
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
		metrics, err = d.DecodeV1(decompressed)
		if err != nil {
			return nil, ErrJsonUnmarshal
		}
	case v2ContentType:
		metrics, stats, err = d.DecodeV2(decompressed)
		if err != nil {
			return &remote.WriteResponseStats{}, ErrJsonUnmarshal
		}
	default:
		return nil, fmt.Errorf("unsupported content type: %s", contentType)
	}

	if err := d.appendable.Put(ctx, metrics...); err != nil {
		return stats, err
	}
	return stats, nil
}

// Flush triggers the flushing of accumulated metrics.
func (d *MetricCollector) Flush(ctx context.Context) error {
	return d.appendable.Flush()
}

// Close stops the flushing goroutine gracefully.
func (d *MetricCollector) Close() {
	d.cancelFunc()
}

// rotateCachePeriodically runs a background goroutine that flushes metrics at regular intervals.
func (d *MetricCollector) rotateCachePeriodically(ctx context.Context) {
	ticker := time.NewTicker(d.settings.Cloudzero.RotateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			flushCtx, cancel := context.WithTimeout(ctx, d.settings.Cloudzero.RotateInterval)
			if err := d.Flush(flushCtx); err != nil {
				log.Err(err).Msg("Error during flush")
			}
			cancel()
		case <-ctx.Done():
			// Perform a final flush before exiting
			// flushCtx, cancel := context.WithTimeout(context.Background(), d.flushInterval)
			// if err := d.Flush(flushCtx); err != nil {
			// 	log.Err(err).Msg("Error during final flush")
			// }
			// cancel()
			return
		}
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
		var metricName string

		for _, label := range ts.Labels {
			labelsMap[label.Name] = label.Value
			if label.Name == "__name__" {
				metricName = label.Value
			}
		}

		for _, sample := range ts.Samples {
			if len(metricName) == 0 { // don't save garbage metrics
				continue
			}
			metrics = append(metrics, types.NewMetric(
				d.settings.OrganizationID,
				d.settings.CloudAccountID,
				d.settings.ClusterName,
				metricName,
				sample.Timestamp,
				labelsMap,
				strconv.FormatFloat(sample.Value, 'f', -1, 64),
			))
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
		stats.Histograms += len(ts.Histograms)
		// Process exemplars
		stats.Exemplars += len(ts.Exemplars)
	}

	// Set Confirmed to true, indicating that statistics are reliable
	stats.Confirmed = true

	return metrics, &stats, nil
}
