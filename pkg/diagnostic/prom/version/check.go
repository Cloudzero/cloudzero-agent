package version

import (
	"bytes"
	"context"
	"fmt"
	net "net/http"
	"os"
	"os/exec"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/diagnostic"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/logging"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/status"
	"github.com/sirupsen/logrus"
)

const DiagnosticPrometheusVersion = config.DiagnosticPrometheusVersion

type checker struct {
	cfg    *config.Settings
	logger *logrus.Entry
}

func NewProvider(ctx context.Context, cfg *config.Settings) diagnostic.Provider {
	return &checker{
		cfg: cfg,
		logger: logging.NewLogger().
			WithContext(ctx).WithField(logging.OpField, "prom"),
	}
}

func (c *checker) Check(ctx context.Context, _ *net.Client, accessor status.Accessor) error {
	if len(c.cfg.Prometheus.Executable) == 0 {
		accessor.AddCheck(&status.StatusCheck{
			Name:  DiagnosticPrometheusVersion,
			Error: "no prometheus binary available at configured location",
		})
		return nil
	}

	versionData, err := c.GetVersion(ctx)
	if err != nil {
		accessor.AddCheck(
			&status.StatusCheck{
				Name:  DiagnosticPrometheusVersion,
				Error: err.Error(),
			})
		return nil
	}

	accessor.WriteToReport(func(s *status.ClusterStatus) {
		s.AgentVersion = string(versionData)
		s.Checks = append(s.Checks, &status.StatusCheck{Name: DiagnosticPrometheusVersion, Passing: true})
	})
	return nil
}

func (c *checker) GetVersion(ctx context.Context) ([]byte, error) {
	executable := c.cfg.Prometheus.Executable
	if len(executable) == 0 {
		return nil, fmt.Errorf("no prometheus binary available at configured location")
	}

	fi, err := os.Stat(executable)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("prometheus executable not found: %w", err)
	}
	if fi.Mode()&0111 == 0 {
		return nil, fmt.Errorf("prometheus executable is not executable: %s", executable)
	}

	// create the raw output fle
	rawOutput, err := os.CreateTemp(os.TempDir(), ".promver.*")
	if err != nil {
		return nil, fmt.Errorf("failed to create raw prometheus version output file: %w", err)
	}
	defer func() {
		_ = rawOutput.Close()
		_ = os.Remove(rawOutput.Name())
	}()

	// Build the command and Exec the scanner
	cmd := exec.CommandContext(ctx, executable, "--version")

	// capture the output
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = rawOutput

	// Now run the app
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to run prometheus: %w", err)
	}

	// make sure all bytes are written from the standard output
	if err := rawOutput.Sync(); err != nil {
		return nil, fmt.Errorf("failed to sync raw prometheus output file: %w", err)
	}

	// seek to the beginning of the file for reading.
	if _, err := rawOutput.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek to the beginning of the raw prometheus output file: %w", err)
	}

	// read the results into a byte slice
	return os.ReadFile(rawOutput.Name())
}
