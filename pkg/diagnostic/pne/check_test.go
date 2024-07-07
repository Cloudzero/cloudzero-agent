package pne_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/diagnostic/pne"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/status"
	"github.com/cloudzero/cloudzero-agent-validator/test"
)

const (
	mockURL = "http://example.com"
)

func makeReport() status.Accessor {
	return status.NewAccessor(&status.ClusterStatus{})
}

func TestChecker_CheckOK(t *testing.T) {
	cfg := &config.Settings{
		Prometheus: config.Prometheus{
			PrometheusNodeExporterServiceEndpoint: mockURL,
		},
	}
	provider := pne.NewProvider(context.Background(), cfg)

	mock := test.NewHTTPMock()
	mock.Expect(http.MethodGet, "Hello World", http.StatusOK, nil)
	client := mock.HTTPClient()

	accessor := makeReport()

	err := provider.Check(context.Background(), client, accessor)
	assert.NoError(t, err)

	accessor.ReadFromReport(func(s *status.ClusterStatus) {
		assert.Len(t, s.Checks, 1)
		for _, c := range s.Checks {
			assert.True(t, c.Passing)
		}
	})
}

func TestChecker_CheckRetry(t *testing.T) {
	cfg := &config.Settings{
		Prometheus: config.Prometheus{
			PrometheusNodeExporterServiceEndpoint: mockURL,
		},
	}
	provider := pne.NewProvider(context.Background(), cfg)

	// Update the test sleep interval to accellerate the test
	pne.RetryInterval = 10 * time.Millisecond
	pne.MaxRetry = 1

	mock := test.NewHTTPMock()
	mock.Expect(http.MethodGet, "", http.StatusNotFound, nil)
	mock.Expect(http.MethodGet, "Hello World", http.StatusOK, nil)
	client := mock.HTTPClient()

	accessor := makeReport()

	err := provider.Check(context.Background(), client, accessor)
	assert.NoError(t, err)

	accessor.ReadFromReport(func(s *status.ClusterStatus) {
		assert.Len(t, s.Checks, 1)
		for _, c := range s.Checks {
			assert.True(t, c.Passing)
		}
	})
}

func TestChecker_CheckRetryFailure(t *testing.T) {
	cfg := &config.Settings{
		Prometheus: config.Prometheus{
			PrometheusNodeExporterServiceEndpoint: mockURL,
		},
	}
	provider := pne.NewProvider(context.Background(), cfg)

	// Update the test sleep interval to accellerate the test
	pne.RetryInterval = 10 * time.Millisecond
	pne.MaxRetry = 0

	mock := test.NewHTTPMock()
	mock.Expect(http.MethodGet, "", http.StatusNotFound, nil)
	client := mock.HTTPClient()

	accessor := makeReport()

	err := provider.Check(context.Background(), client, accessor)
	assert.NoError(t, err)

	accessor.ReadFromReport(func(s *status.ClusterStatus) {
		assert.Len(t, s.Checks, 1)
		for _, c := range s.Checks {
			assert.False(t, c.Passing)
		}
	})
}
