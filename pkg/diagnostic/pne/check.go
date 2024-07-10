package pne

import (
	"context"
	"fmt"
	net "net/http"
	"time"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/diagnostic"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/http"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/logging"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/status"
	"github.com/sirupsen/logrus"
)

const DiagnosticNodeExporter = config.DiagnosticNodeExporter

var (
	// Exported so that it can be overridden in tests
	MaxRetry      = 12
	RetryInterval = 10 * time.Second
)

type checker struct {
	cfg    *config.Settings
	logger *logrus.Entry
}

func NewProvider(ctx context.Context, cfg *config.Settings) diagnostic.Provider {
	return &checker{
		cfg: cfg,
		logger: logging.NewLogger().
			WithContext(ctx).WithField(logging.OpField, "pne"),
	}
}

func (c *checker) Check(ctx context.Context, client *net.Client, accessor status.Accessor) error {
	var (
		err              error
		retriesRemaining = MaxRetry
		url              = fmt.Sprintf("%s/", c.cfg.Prometheus.PrometheusNodeExporterServiceEndpoint)
	)

	// We need to build in a retry here because the prometheus-node-exporter
	// service can take a few seconds to start up
	// If it is deploying with the cloudzero-agent chart
	for {
		_, err = http.Do(ctx, client, net.MethodGet, nil, nil, url, nil)
		if err == nil {
			break
		}
		if retriesRemaining == 0 {
			break
		}

		retriesRemaining--
		time.Sleep(RetryInterval)
	}

	if err != nil {
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticNodeExporter, Passing: false, Error: err.Error()})
		return nil
	}

	accessor.AddCheck(&status.StatusCheck{Name: DiagnosticNodeExporter, Passing: true})
	return nil

}
