package domain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cirrus-remote-write/app/config"
	"github.com/cloudzero/cirrus-remote-write/app/parallel"
	"github.com/cloudzero/cirrus-remote-write/app/types"
)

var (
	ErrUnauthorized = errors.New("unauthorized request - possible invalid API key")
	ErrNoURLs       = errors.New("no presigned URLs returned")
)

// Ensure MetricShipper implements types.Runnable
var _ types.Runnable = (*MetricShipper)(nil)

// Status represents the current status of the MetricShipper.
type Status struct {
	ShippableFiles int    `json:"shippable_files"`
	ShippedFiles   uint64 `json:"shipped_files"`
}

// MetricShipper handles the periodic shipping of metrics to Cloudzero.
type MetricShipper struct {
	setting *config.Settings
	lister  types.AppendableFiles

	// Internal fields
	ctx          context.Context
	cancel       context.CancelFunc
	HttpClient   *http.Client
	shippedFiles uint64 // Counter for shipped files
}

// NewMetricShipper initializes a new MetricShipper.
func NewMetricShipper(ctx context.Context, s *config.Settings, f types.AppendableFiles) *MetricShipper {
	ctx, cancel := context.WithCancel(ctx)

	// Initialize an HTTP client with the specified timeout
	httpClient := &http.Client{
		Timeout: time.Duration(s.Cloudzero.SendTimeout) * time.Second,
	}

	return &MetricShipper{
		setting:    s,
		lister:     f,
		ctx:        ctx,
		cancel:     cancel,
		HttpClient: httpClient,
	}
}

// Run starts the MetricShipper service and blocks until a shutdown signal is received.
func (m *MetricShipper) Run() error {
	// Set up channel to listen for OS signals
	sigChan := make(chan os.Signal, 1)
	// Listen for interrupt and termination signals
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Initialize ticker for periodic shipping
	ticker := time.NewTicker(time.Duration(m.setting.Cloudzero.SendInterval) * time.Second)
	defer ticker.Stop()

	log.Info().Msg("Shipper service starting")

	for {
		select {
		case <-m.ctx.Done():
			log.Info().Msg("Shipper service stopping")
			return nil

		case sig := <-sigChan:
			log.Info().Msgf("Received signal %s. Initiating shutdown.", sig)
			m.Shutdown()
			return nil

		case <-ticker.C:
			if err := m.performShipping(); err != nil {
				log.Error().Err(err).Msg("Failed to ship metrics")
			}
		}
	}
}

// GetStatus returns the current status of the MetricShipper, including
// the number of shippable files and the number of files shipped.
func (m *MetricShipper) GetStatus() (Status, error) {
	// Retrieve the current list of shippable files
	files, err := m.lister.GetFiles()
	if err != nil {
		return Status{}, fmt.Errorf("failed to get shippable files: %w", err)
	}

	// Atomically load the number of shipped files
	shipped := atomic.LoadUint64(&m.shippedFiles)

	return Status{
		ShippableFiles: len(files),
		ShippedFiles:   shipped,
	}, nil
}

// performShipping encapsulates the steps to ship existing .tgz files and new metrics.
func (m *MetricShipper) performShipping() error {
	pm := parallel.New(10)
	defer pm.Close()

	// Process new files in parallel
	files, err := m.lister.GetFiles()
	if err != nil {
		return fmt.Errorf("failed to get shippable files: %w", err)
	}

	// Get the presigned URLs for the files in a batch
	presignedURLs, err := m.AllocatePresignedURLs(len(files))
	if err != nil {
		return fmt.Errorf("failed to allocate presigned URLs: %w", err)
	}
	if len(presignedURLs) == 0 {
		return nil
	}

	waiter := parallel.NewWaiter()
	for i, file := range files {
		filePath := file
		presignedURL := presignedURLs[i]
		fn := func() error {
			// Upload the file
			if err := m.UploadFile(presignedURL, filePath); err != nil {
				return fmt.Errorf("failed to upload %s: %w", filePath, err)
			}

			// Delete the local file after successful upload
			if err := os.Remove(filePath); err != nil {
				log.Error().Err(err).Msgf("Failed to delete local file %s", filePath)
				return nil
			}

			atomic.AddUint64(&m.shippedFiles, 1)
			return nil
		}
		pm.Run(fn, waiter)
	}

	waiter.Wait()
	return nil
}

// AllocatePresignedURL requests a presigned S3 URL from the Cloudzero API for the given file.
func (m *MetricShipper) AllocatePresignedURLs(count int) ([]string, error) {
	uploadEndpoint := m.setting.Cloudzero.Host
	if count <= 0 {
		return nil, nil
	}

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(m.ctx, "POST", uploadEndpoint, bytes.NewBuffer([]byte{}))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set necessary headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.setting.GetAPIKey()))

	// Make sure we set the query parameters for count, expiration, cloud_account_id, region, cluster_name
	q := req.URL.Query()
	q.Add("count", fmt.Sprintf("%d", count))
	q.Add("expiration", fmt.Sprintf("%d", 3600))
	q.Add("cloud_account_id", m.setting.CloudAccountID)
	q.Add("region", m.setting.Region)
	q.Add("cluster_name", m.setting.ClusterName)
	req.URL.RawQuery = q.Encode()

	// Send the request
	resp, err := m.HttpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response
	var respData struct {
		URLs []string `json:"urls"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(respData.URLs) == 0 {
		return nil, ErrNoURLs
	}

	return respData.URLs, nil
}

// UploadFile uploads the specified file to S3 using the provided presigned URL.
func (m *MetricShipper) UploadFile(presignedURL, filePath string) error {
	// Open the file to upload
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for upload: %w", err)
	}
	defer file.Close()

	// Create a unique context with a timeout for the upload
	ctx, cancel := context.WithTimeout(m.ctx, time.Duration(m.setting.Cloudzero.SendTimeout)*time.Second)
	defer cancel()

	// Create a new HTTP PUT request with the file as the body
	req, err := http.NewRequestWithContext(ctx, "PUT", presignedURL, file)
	if err != nil {
		return fmt.Errorf("failed to create upload HTTP request: %w", err)
	}

	// Set the appropriate Content-Type if required by the presigned URL
	req.Header.Set("Content-Type", "application/octet-stream")

	// Send the request
	resp, err := m.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("file upload HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful upload
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected upload status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// Shutdown gracefully stops the MetricShipper service.
func (m *MetricShipper) Shutdown() error {
	m.cancel()
	return nil
}
