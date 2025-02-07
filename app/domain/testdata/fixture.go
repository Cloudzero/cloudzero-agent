package testdata

import (
	"context"
	"fmt"
	"math"

	"github.com/cloudzero/cloudzero-insights-controller/app/domain"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/histogram"
	"github.com/prometheus/prometheus/model/metadata"
	"github.com/prometheus/prometheus/prompb"
	writev2 "github.com/prometheus/prometheus/prompb/io/prometheus/write/v2"

	"github.com/rs/zerolog/log"
)

var (
	// Borrowed from https://github.com/prometheus/prometheus/blob/main/storage/remote/codec_test.go#L96
	TestHistogram = histogram.Histogram{
		Schema:          2,
		ZeroThreshold:   1e-128,
		ZeroCount:       0,
		Count:           0,
		Sum:             20,
		PositiveSpans:   []histogram.Span{{Offset: 0, Length: 1}},
		PositiveBuckets: []int64{1},
		NegativeSpans:   []histogram.Span{{Offset: 0, Length: 1}},
		NegativeBuckets: []int64{-1},
	}

	WriteRequestFixture = &prompb.WriteRequest{
		Timeseries: []prompb.TimeSeries{
			{
				Labels: []prompb.Label{
					{Name: "__name__", Value: "test_metric1"},
					{Name: "b", Value: "c"},
					{Name: "baz", Value: "qux"},
					{Name: "d", Value: "e"},
					{Name: "foo", Value: "bar"},
				},
				Samples: []prompb.Sample{{Value: 1, Timestamp: 1}},
				Exemplars: []prompb.Exemplar{
					{Labels: []prompb.Label{{Name: "f", Value: "g"}}, Value: 1, Timestamp: 1},
				},
				Histograms: []prompb.Histogram{
					prompb.FromIntHistogram(1, &TestHistogram),
					prompb.FromFloatHistogram(2, TestHistogram.ToFloat(nil)),
				},
			},
			{
				Labels: []prompb.Label{
					{Name: "__name__", Value: "test_metric1"},
					{Name: "b", Value: "c"},
					{Name: "baz", Value: "qux"},
					{Name: "d", Value: "e"},
					{Name: "foo", Value: "bar"},
				},
				Samples: []prompb.Sample{
					{Value: 2, Timestamp: 2},
				},
				Exemplars: []prompb.Exemplar{
					{Labels: []prompb.Label{{Name: "h", Value: "i"}}, Value: 2, Timestamp: 2},
				},
				Histograms: []prompb.Histogram{
					prompb.FromIntHistogram(3, &TestHistogram),
					prompb.FromFloatHistogram(4, TestHistogram.ToFloat(nil)),
				},
			},
		},
	}

	WriteV2RequestSeries1Metadata = metadata.Metadata{
		Type: model.MetricTypeGauge,
		Help: "Test gauge for test purposes",
		Unit: "Maybe op/sec who knows (:",
	}
	WriteV2RequestSeries2Metadata = metadata.Metadata{
		Type: model.MetricTypeCounter,
		Help: "Test counter for test purposes",
	}

	// WriteV2RequestFixture represents the same request as writeRequestFixture,
	// but using the v2 representation, plus includes writeV2RequestSeries1Metadata and writeV2RequestSeries2Metadata.
	// NOTE: Use TestWriteV2RequestFixture and copy the diff to regenerate if needed.
	WriteV2RequestFixture = &writev2.Request{
		Symbols: []string{"", "__name__", "test_metric1", "b", "c", "baz", "qux", "d", "e", "foo", "bar", "f", "g", "h", "i", "Test gauge for test purposes", "Maybe op/sec who knows (:", "Test counter for test purposes"},
		Timeseries: []writev2.TimeSeries{
			{
				LabelsRefs: []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, // Symbolized writeRequestFixture.Timeseries[0].Labels
				Metadata: writev2.Metadata{
					Type: writev2.Metadata_METRIC_TYPE_GAUGE, // writeV2RequestSeries1Metadata.Type.

					HelpRef: 15, // Symbolized writeV2RequestSeries1Metadata.Help.
					UnitRef: 16, // Symbolized writeV2RequestSeries1Metadata.Unit.
				},
				Samples:    []writev2.Sample{{Value: 1, Timestamp: 1}},
				Exemplars:  []writev2.Exemplar{{LabelsRefs: []uint32{11, 12}, Value: 1, Timestamp: 1}},
				Histograms: []writev2.Histogram{writev2.FromIntHistogram(1, &TestHistogram), writev2.FromFloatHistogram(2, TestHistogram.ToFloat(nil))},
			},
			{
				LabelsRefs: []uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, // Same series as first.
				Metadata: writev2.Metadata{
					Type: writev2.Metadata_METRIC_TYPE_COUNTER, // writeV2RequestSeries2Metadata.Type.

					HelpRef: 17, // Symbolized writeV2RequestSeries2Metadata.Help.
					// No unit.
				},
				Samples:    []writev2.Sample{{Value: 2, Timestamp: 2}},
				Exemplars:  []writev2.Exemplar{{LabelsRefs: []uint32{13, 14}, Value: 2, Timestamp: 2}},
				Histograms: []writev2.Histogram{writev2.FromIntHistogram(3, &TestHistogram), writev2.FromFloatHistogram(4, TestHistogram.ToFloat(nil))},
			},
		},
	}
)

func CompressPayload(tmpbuf *[]byte, inp []byte, enc string) (compressed []byte, _ error) {
	switch enc {
	case domain.SnappyBlockCompression:
		compressed = snappy.Encode(*tmpbuf, inp)
		if n := snappy.MaxEncodedLen(len(inp)); n > len(*tmpbuf) {
			// grow the buffer for the next time
			*tmpbuf = make([]byte, n)
		}
		return compressed, nil
	default:
		return compressed, fmt.Errorf("Unknown compression scheme [%v]", enc)
	}
}

func BuildTimeSeries(
	timeSeries []prompb.TimeSeries,
	filter func(prompb.TimeSeries) bool,
) (int64, int64, []prompb.TimeSeries, int, int, int) {
	var highest int64
	var lowest int64
	var droppedSamples, droppedExemplars, droppedHistograms int

	keepIdx := 0
	lowest = math.MaxInt64
	for i, ts := range timeSeries {
		if filter != nil && filter(ts) {
			if len(ts.Samples) > 0 {
				droppedSamples++
			}
			if len(ts.Exemplars) > 0 {
				droppedExemplars++
			}
			if len(ts.Histograms) > 0 {
				droppedHistograms++
			}
			continue
		}

		// At the moment we only ever append a TimeSeries with a single sample or exemplar in it.
		if len(ts.Samples) > 0 && ts.Samples[0].Timestamp > highest {
			highest = ts.Samples[0].Timestamp
		}
		if len(ts.Exemplars) > 0 && ts.Exemplars[0].Timestamp > highest {
			highest = ts.Exemplars[0].Timestamp
		}
		if len(ts.Histograms) > 0 && ts.Histograms[0].Timestamp > highest {
			highest = ts.Histograms[0].Timestamp
		}

		// Get lowest timestamp
		if len(ts.Samples) > 0 && ts.Samples[0].Timestamp < lowest {
			lowest = ts.Samples[0].Timestamp
		}
		if len(ts.Exemplars) > 0 && ts.Exemplars[0].Timestamp < lowest {
			lowest = ts.Exemplars[0].Timestamp
		}
		if len(ts.Histograms) > 0 && ts.Histograms[0].Timestamp < lowest {
			lowest = ts.Histograms[0].Timestamp
		}
		if i != keepIdx {
			// We have to swap the kept timeseries with the one which should be dropped.
			// Copying any elements within timeSeries could cause data corruptions when reusing the slice in a next batch (shards.populateTimeSeries).
			timeSeries[keepIdx], timeSeries[i] = timeSeries[i], timeSeries[keepIdx]
		}
		keepIdx++
	}

	timeSeries = timeSeries[:keepIdx]
	return highest, lowest, timeSeries, droppedSamples, droppedExemplars, droppedHistograms
}

func BuildWriteRequest(
	timeSeries []prompb.TimeSeries,
	metadata []prompb.MetricMetadata,
	pBuf *proto.Buffer,
	buf *[]byte,
	filter func(prompb.TimeSeries) bool,
	enc string,
) (compressed []byte, highest, lowest int64, _ error) {
	highest, lowest, timeSeries,
		droppedSamples, droppedExemplars, droppedHistograms := BuildTimeSeries(timeSeries, filter)

	if droppedSamples > 0 || droppedExemplars > 0 || droppedHistograms > 0 {
		log.Ctx(context.TODO()).Debug().
			Str("message", "dropped data due to their age").
			Int("droppedSamples", droppedSamples).
			Int("droppedExemplars", droppedExemplars).
			Int("droppedHistograms", droppedHistograms).
			Send()
	}

	req := &prompb.WriteRequest{
		Timeseries: timeSeries,
		Metadata:   metadata,
	}

	if pBuf == nil {
		pBuf = proto.NewBuffer(nil) // For convenience in tests. Not efficient.
	} else {
		pBuf.Reset()
	}
	err := pBuf.Marshal(req)
	if err != nil {
		return nil, highest, lowest, err
	}

	// snappy uses len() to see if it needs to allocate a new slice. Make the
	// buffer as long as possible.
	if buf != nil {
		*buf = (*buf)[0:cap(*buf)]
	} else {
		buf = &[]byte{}
	}

	compressed, err = CompressPayload(buf, pBuf.Bytes(), enc)
	if err != nil {
		return nil, highest, lowest, err
	}
	return compressed, highest, lowest, nil
}

func BuildV2WriteRequest(
	samples []writev2.TimeSeries,
	labels []string,
	pBuf, buf *[]byte,
	filter func(writev2.TimeSeries) bool,
	enc string,
) (compressed []byte, highest, lowest int64, _ error) {
	highest, lowest, timeSeries, droppedSamples, droppedExemplars, droppedHistograms := BuildV2TimeSeries(samples, filter)

	if droppedSamples > 0 || droppedExemplars > 0 || droppedHistograms > 0 {
		log.Ctx(context.TODO()).Debug().
			Str("message", "dropped data due to their age").
			Int("droppedSamples", droppedSamples).
			Int("droppedExemplars", droppedExemplars).
			Int("droppedHistograms", droppedHistograms).
			Send()
	}

	req := &writev2.Request{
		Symbols:    labels,
		Timeseries: timeSeries,
	}

	if pBuf == nil {
		pBuf = &[]byte{} // For convenience in tests. Not efficient.
	}

	data, err := req.OptimizedMarshal(*pBuf)
	if err != nil {
		return nil, highest, lowest, err
	}
	*pBuf = data

	// snappy uses len() to see if it needs to allocate a new slice. Make the
	// buffer as long as possible.
	if buf != nil {
		*buf = (*buf)[0:cap(*buf)]
	} else {
		buf = &[]byte{}
	}

	compressed, err = CompressPayload(buf, data, enc)
	if err != nil {
		return nil, highest, lowest, err
	}
	return compressed, highest, lowest, nil
}

func BuildV2TimeSeries(
	timeSeries []writev2.TimeSeries,
	filter func(writev2.TimeSeries) bool,
) (int64, int64, []writev2.TimeSeries, int, int, int) {
	var highest int64
	var lowest int64
	var droppedSamples, droppedExemplars, droppedHistograms int

	keepIdx := 0
	lowest = math.MaxInt64
	for i, ts := range timeSeries {
		if filter != nil && filter(ts) {
			if len(ts.Samples) > 0 {
				droppedSamples++
			}
			if len(ts.Exemplars) > 0 {
				droppedExemplars++
			}
			if len(ts.Histograms) > 0 {
				droppedHistograms++
			}
			continue
		}

		// At the moment we only ever append a TimeSeries with a single sample or exemplar in it.
		if len(ts.Samples) > 0 && ts.Samples[0].Timestamp > highest {
			highest = ts.Samples[0].Timestamp
		}
		if len(ts.Exemplars) > 0 && ts.Exemplars[0].Timestamp > highest {
			highest = ts.Exemplars[0].Timestamp
		}
		if len(ts.Histograms) > 0 && ts.Histograms[0].Timestamp > highest {
			highest = ts.Histograms[0].Timestamp
		}

		// Get the lowest timestamp.
		if len(ts.Samples) > 0 && ts.Samples[0].Timestamp < lowest {
			lowest = ts.Samples[0].Timestamp
		}
		if len(ts.Exemplars) > 0 && ts.Exemplars[0].Timestamp < lowest {
			lowest = ts.Exemplars[0].Timestamp
		}
		if len(ts.Histograms) > 0 && ts.Histograms[0].Timestamp < lowest {
			lowest = ts.Histograms[0].Timestamp
		}
		if i != keepIdx {
			// We have to swap the kept timeseries with the one which should be dropped.
			// Copying any elements within timeSeries could cause data corruptions when reusing the slice in a next batch (shards.populateTimeSeries).
			timeSeries[keepIdx], timeSeries[i] = timeSeries[i], timeSeries[keepIdx]
		}
		keepIdx++
	}

	timeSeries = timeSeries[:keepIdx]
	return highest, lowest, timeSeries, droppedSamples, droppedExemplars, droppedHistograms
}
