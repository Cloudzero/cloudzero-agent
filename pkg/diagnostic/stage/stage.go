package stage

import (
	"context"
	net "net/http"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/diagnostic"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/logging"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/status"
	"github.com/sirupsen/logrus"
)

const DiagnosticScrapeConfig = config.DiagnosticScrapeConfig

type checker struct {
	cfg    *config.Settings
	logger *logrus.Entry
	stage  status.StatusType
}

func NewProvider(ctx context.Context, cfg *config.Settings, stage status.StatusType) diagnostic.Provider {
	return &checker{
		cfg: cfg,
		logger: logging.NewLogger().
			WithContext(ctx).
			WithField(logging.OpField, "stage"),
		stage: stage,
	}
}

func (c *checker) Check(_ context.Context, _ *net.Client, accessor status.Accessor) error {
	accessor.WriteToReport(func(s *status.ClusterStatus) {
		s.State = c.stage
	})
	return nil
}
