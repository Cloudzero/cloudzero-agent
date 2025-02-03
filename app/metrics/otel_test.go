package metrics

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func TestPrometheusHandler(t *testing.T) {
	// create a mock http service
	srv := httptest.NewServer(Handler())
	defer srv.Close()

	// record a sample metric
	testHistogram.Record(context.Background(), 12.34)

	// check the result
	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "test_histogram")
	require.Contains(t, string(body), "12.34")
}

func TestMetrics(t *testing.T) {
	// create mock http server
	srv := httptest.NewServer(Handler())
	defer srv.Close()

	getBody := func(t *testing.T) string {
		resp, err := http.Get(srv.URL)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return string(body)
	}

	t.Run("RemoteWriteRequestCount", func(t *testing.T) {
		RemoteWriteRequestCount.Add(context.Background(), 1,
			metric.WithAttributes(
				attribute.String("endpoint", "https://test-endpoint-1.com"),
			),
		)

		RemoteWriteRequestCount.Add(context.Background(), 2,
			metric.WithAttributes(
				attribute.String("endpoint", "https://test-endpoint-2.com"),
			),
		)

		body := getBody(t)

		// ensure that things are being recorded correctly
		for _, line := range strings.Split(string(body), "\n") {
			if strings.Contains(line, "https://test-endpoint-1.com") {
				require.Contains(t, line, "1")
			}
			if strings.Contains(line, "https://test-endpoint-2.com") {
				require.Contains(t, line, "2")
			}
		}
	})

	t.Run("RemoteWriteRequestDurationSeconds", func(t *testing.T) {
		RemoteWriteRequestDurationSeconds.Record(context.Background(), 0.02)
		RemoteWriteRequestDurationSeconds.Record(context.Background(), 2.3)
		RemoteWriteRequestDurationSeconds.Record(context.Background(), 10)

		body := getBody(t)
		fmt.Println(body)
	})
}
