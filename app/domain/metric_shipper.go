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
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cirrus-remote-write/app/compress"
	"github.com/cloudzero/cirrus-remote-write/app/config"
	"github.com/cloudzero/cirrus-remote-write/app/parallel"
	"github.com/cloudzero/cirrus-remote-write/app/types"
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
	// First, ship existing .tgz files
	tgzFiles, err := filepath.Glob(filepath.Join(m.setting.Database.StoragePath, "*.tgz"))
	if err != nil {
		return fmt.Errorf("failed to list .tgz files: %w", err)
	}

	pm := parallel.New(10)
	defer pm.Close()
	waiter := parallel.NewWaiter()

	// Ship existing .tgz files first
	for _, tgzFile := range tgzFiles {
		filePath := tgzFile
		fn := func() error {
			// Give them at least 10 minutes - they may be in flight
			if fi, err := os.Stat(filePath); err != nil || time.Since(fi.ModTime()) < 10*time.Minute {
				return nil
			}

			if err := m.ShipFile(filePath); err != nil {
				log.Error().Err(err).Msgf("Failed to ship and remove %s", filePath)
				return nil
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

	// Process new files in parallel
	files, err := m.lister.GetFiles()
	if err != nil {
		return fmt.Errorf("failed to get shippable files: %w", err)
	}

	for _, file := range files {
		filePath := file
		fn := func() error {
			locked, err := m.LockFile(filePath)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to lock file %s", filePath)
			}
			if !locked {
				return nil
			}
			defer m.UnlockFile(filePath)

			destFilePath, err := compress.File(filePath)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to compress file %s", filePath)
				return nil
			}
			if destFilePath == nil {
				return nil
			}

			if err = m.ShipFile(*destFilePath); err != nil {
				log.Error().Err(err).Msgf("Failed to ship and remove %s", *destFilePath)
				return nil
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

// ShipFile handles the shipping of a single .tgz file.
func (m *MetricShipper) ShipFile(filePath string) error {
	presignedURL, err := m.AllocatePresignedURL(filePath)
	if err != nil {
		return fmt.Errorf("failed to allocate presigned URL for %s: %w", filePath, err)
	}

	if err := m.UploadFile(presignedURL, filePath); err != nil {
		return fmt.Errorf("failed to upload %s: %w", filePath, err)
	}
	return nil
}

// AllocatePresignedURL requests a presigned S3 URL from the Cloudzero API for the given file.
func (m *MetricShipper) AllocatePresignedURL(filePath string) (string, error) {
	uploadEndpoint := m.setting.Cloudzero.Host

	// Prepare the request payload
	payloadBytes, err := json.Marshal(map[string]string{
		"file_name":        filepath.Base(filePath),
		"cluster_name":     m.setting.ClusterName,
		"cloud_account_id": m.setting.CloudAccountID,
		"region":           m.setting.Region,
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create a new HTTP request
	req, err := http.NewRequestWithContext(m.ctx, "POST", uploadEndpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set necessary headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", m.setting.Cloudzero.APIKey))

	// Send the request
	resp, err := m.HttpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response
	var respData struct {
		PresignedURL string `json:"presigned_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if respData.PresignedURL == "" {
		return "", errors.New("presigned_url is empty in response")
	}

	return respData.PresignedURL, nil
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

// LockFile attempts to lock the specified file. Returns true if locked, false otherwise.
func (m *MetricShipper) LockFile(srcFilePath string) (bool, error) {
	lockFilePath := srcFilePath + ".lock"

	// Attempt to create the lock file atomically
	lockFile, err := os.OpenFile(lockFilePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		if os.IsExist(err) {
			// Lock file already exists. Check if it's stale.
			info, statErr := os.Stat(lockFilePath)
			if statErr != nil {
				return false, fmt.Errorf("failed to stat lock file: %w", statErr)
			}

			// Determine the age of the lock file
			age := time.Since(info.ModTime())
			if age > m.setting.Cloudzero.LockStaleDuration {
				// Lock file is stale. Remove it and retry creating the lock file.
				log.Warn().Msgf("Stale lock file detected for %s (age: %v). Removing stale lock.", srcFilePath, age)
				if removeErr := os.Remove(lockFilePath); removeErr != nil {
					return false, fmt.Errorf("failed to remove stale lock file: %w", removeErr)
				}

				// Retry creating the lock file
				lockFile, err = os.OpenFile(lockFilePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
				if err != nil {
					if os.IsExist(err) {
						// Another process might have created the lock file after removal
						log.Warn().Msgf("Lock file for %s was recreated by another process. Skipping compression.", srcFilePath)
						return false, nil
					}
					return false, fmt.Errorf("failed to create lock file after removing stale lock: %w", err)
				}
			} else {
				// Lock file is not stale. Skip processing the file.
				return false, nil
			}
		} else {
			// An unexpected error occurred while creating the lock file
			return false, fmt.Errorf("failed to create lock file: %w", err)
		}
	}
	// Close the lock file
	if err := lockFile.Close(); err != nil {
		return false, fmt.Errorf("failed to close lock file: %w", err)
	}

	return true, nil
}

// UnlockFile removes the lock file for the specified source file.
func (m *MetricShipper) UnlockFile(srcFilePath string) {
	lockFilePath := srcFilePath + ".lock"
	if err := os.Remove(lockFilePath); err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Msgf("Failed to remove lock file %s", lockFilePath)
	}
}
