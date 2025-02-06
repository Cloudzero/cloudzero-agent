package shipper

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/cloudzero/cloudzero-insights-controller/app/instr"
	"github.com/cloudzero/cloudzero-insights-controller/app/parallel"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/rs/zerolog/log"
)

const (
	shipperWorkerCount = 10
	expirationTime     = 3600
)

var (
	ErrUnauthorized = errors.New("unauthorized request - possible invalid API key")
	ErrNoURLs       = errors.New("no presigned URLs returned")
)

// MetricShipper handles the periodic shipping of metrics to Cloudzero.
type MetricShipper struct {
	setting *config.Settings
	lister  types.AppendableFiles

	// Internal fields
	ctx          context.Context
	cancel       context.CancelFunc
	HTTPClient   *http.Client
	shippedFiles uint64 // Counter for shipped files
	metrics      *instr.PrometheusMetrics
}

// NewMetricShipper initializes a new MetricShipper.
func NewMetricShipper(ctx context.Context, s *config.Settings, f types.AppendableFiles) (*MetricShipper, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Initialize an HTTP client with the specified timeout
	httpClient := &http.Client{
		Timeout: s.Cloudzero.SendTimeout,
	}

	// create the metrics listener
	metrics, err := InitMetrics()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to init metrics: %s", err)
	}

	return &MetricShipper{
		setting:    s,
		lister:     f,
		ctx:        ctx,
		cancel:     cancel,
		HTTPClient: httpClient,
		metrics:    metrics,
	}, nil
}

func (m *MetricShipper) GetMetricHandler() http.Handler {
	return m.metrics.Handler()
}

// Run starts the MetricShipper service and blocks until a shutdown signal is received.
func (m *MetricShipper) Run() error {
	// Set up channel to listen for OS signals
	sigChan := make(chan os.Signal, 1)
	// Listen for interrupt and termination signals
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Initialize ticker for periodic shipping
	ticker := time.NewTicker(m.setting.Cloudzero.SendInterval)
	defer ticker.Stop()

	log.Info().Msg("Shipper service starting")

	for {
		select {
		case <-m.ctx.Done():
			log.Info().Msg("Shipper service stopping")
			return nil

		case sig := <-sigChan:
			log.Info().Msgf("Received signal %s. Initiating shutdown.", sig)
			err := m.Shutdown()
			if err != nil {
				log.Error().Err(err).Msg("Failed to shutdown shipper service")
			}
			return nil

		case <-ticker.C:
			if err := m.Ship(); err != nil {
				log.Error().Err(err).Msg("Failed to ship metrics")
			}
		}
	}
}

func (m *MetricShipper) Ship() error {
	pm := parallel.New(shipperWorkerCount)
	defer pm.Close()

	// Process new files in parallel
	paths, err := m.lister.GetFiles()
	if err != nil {
		return fmt.Errorf("failed to get shippable files: %w", err)
	}

	// create the files object
	files := NewFilesFromPaths(paths)
	if len(files) == 0 {
		return nil
	}

	// Assign pre-signed urls to each of the file references
	files, err = m.AllocatePresignedURLs(files)
	if err != nil {
		return fmt.Errorf("failed to allocate presigned URLs: %w", err)
	}

	waiter := parallel.NewWaiter()
	for _, file := range files {
		fn := func() error {
			// Upload the file
			if err := m.UploadFile(file); err != nil {
				return fmt.Errorf("failed to upload %s: %w", file.ReferenceID, err)
			}

			// TODO - mark the file as uploaded

			atomic.AddUint64(&m.shippedFiles, 1)
			return nil
		}
		pm.Run(fn, waiter)
	}

	return nil
}

// Shutdown gracefully stops the MetricShipper service.
func (m *MetricShipper) Shutdown() error {
	m.cancel()
	return nil
}
