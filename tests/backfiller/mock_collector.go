// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package backfiller provides utilities for testing label backfiller integrations.
package backfiller

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"github.com/rs/zerolog/log"
)

// MockCollector captures and validates Prometheus RemoteWrite requests
type MockCollector struct {
	port           int
	server         *http.Server
	receivedData   []prompb.WriteRequest
	mu             sync.RWMutex
	outputDir      string
	expectedLabels map[string]string
}

// ReceivedMetrics holds captured metrics data for verification
type ReceivedMetrics struct {
	Timestamp time.Time
	Request   prompb.WriteRequest
}

// NewMockCollector creates a new mock collector for testing
func NewMockCollector(port int, outputDir string) *MockCollector {
	return &MockCollector{
		port:           port,
		receivedData:   make([]prompb.WriteRequest, 0),
		outputDir:      outputDir,
		expectedLabels: make(map[string]string),
	}
}

// SetExpectedLabels sets the labels we expect to see in the metrics
func (mc *MockCollector) SetExpectedLabels(labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.expectedLabels = labels
}

// Start starts the mock collector HTTP server
func (mc *MockCollector) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/container-metrics", mc.handleContainerMetrics)
	mux.HandleFunc("/health", mc.handleHealth)

	mc.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", mc.port),
		Handler: mux,
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(mc.outputDir, 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	go func() {
		log.Info().Int("port", mc.port).Msg("Mock collector starting")
		if err := mc.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("Mock collector server error")
		}
	}()

	return nil
}

// Stop stops the mock collector server
func (mc *MockCollector) Stop() error {
	if mc.server != nil {
		return mc.server.Close()
	}
	return nil
}

// handleContainerMetrics handles the container metrics endpoint
func (mc *MockCollector) handleContainerMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Validate query parameters
	query := r.URL.Query()
	if query.Get("cluster_name") == "" || query.Get("cloud_account_id") == "" || query.Get("region") == "" {
		http.Error(w, "Missing required query parameters", http.StatusBadRequest)
		return
	}

	// Read and decompress the request body
	compressedData, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Decompress with snappy
	decompressedData, err := snappy.Decode(nil, compressedData)
	if err != nil {
		http.Error(w, "Failed to decompress data", http.StatusInternalServerError)
		return
	}

	// Parse protobuf WriteRequest
	var writeRequest prompb.WriteRequest
	if err := proto.Unmarshal(decompressedData, &writeRequest); err != nil {
		http.Error(w, "Failed to parse protobuf", http.StatusInternalServerError)
		return
	}

	// Store the received data
	mc.mu.Lock()
	mc.receivedData = append(mc.receivedData, writeRequest)
	writeRequestCount := len(mc.receivedData)
	mc.mu.Unlock()

	// Debug logging for each write request
	log.Info().Msg("=== MOCK COLLECTOR DEBUG ===")
	log.Info().Int("writeRequestNumber", writeRequestCount).Int("timeseriesCount", len(writeRequest.Timeseries)).Msg("Received WriteRequest")

	// Debug each timeseries in the write request
	for i, ts := range writeRequest.Timeseries {
		log.Info().Int("timeseriesIndex", i+1).Int("labelCount", len(ts.Labels)).Msg("Processing TimeSeries")

		// Log labels
		for _, label := range ts.Labels {
			log.Info().Str("labelName", label.Name).Str("labelValue", label.Value).Msg("TimeSeries label")
		}

		// Log samples
		log.Info().Int("sampleCount", len(ts.Samples)).Msg("TimeSeries samples")
		for j, sample := range ts.Samples {
			log.Info().Int("sampleIndex", j+1).Float64("value", sample.Value).Int64("timestamp", sample.Timestamp).Msg("Sample data")
		}
	}
	log.Info().Msg("=== END DEBUG ===")

	// Save to file for debugging
	timestamp := time.Now().Unix()
	fileName := fmt.Sprintf("received_metrics_%d.json", timestamp)
	filePath := fmt.Sprintf("%s/%s", mc.outputDir, fileName)

	if err := mc.saveMetricsToFile(filePath, writeRequest); err != nil {
		log.Error().Err(err).Str("filePath", filePath).Msg("Failed to save metrics to file")
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleHealth handles health check requests
func (mc *MockCollector) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("healthy"))
}

// saveMetricsToFile saves metrics to a JSON file for debugging
func (mc *MockCollector) saveMetricsToFile(filePath string, writeRequest prompb.WriteRequest) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write a human-readable representation
	for i, ts := range writeRequest.Timeseries {
		fmt.Fprintf(file, "TimeSeries %d:\n", i)
		fmt.Fprintf(file, "  Labels:\n")
		for _, label := range ts.Labels {
			fmt.Fprintf(file, "    %s: %s\n", label.Name, label.Value)
		}
		fmt.Fprintf(file, "  Samples:\n")
		for _, sample := range ts.Samples {
			fmt.Fprintf(file, "    Value: %f, Timestamp: %d\n", sample.Value, sample.Timestamp)
		}
		fmt.Fprintf(file, "\n")
	}

	return nil
}

// GetReceivedData returns all received WriteRequests
func (mc *MockCollector) GetReceivedData() []prompb.WriteRequest {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make([]prompb.WriteRequest, len(mc.receivedData))
	copy(result, mc.receivedData)
	return result
}

// GetNamespaceMetrics returns only the namespace-related metrics
func (mc *MockCollector) GetNamespaceMetrics() []prompb.TimeSeries {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var namespaceMetrics []prompb.TimeSeries

	for _, writeRequest := range mc.receivedData {
		for _, ts := range writeRequest.Timeseries {
			// Look for namespace metrics
			for _, label := range ts.Labels {
				if label.Name == "__name__" &&
					(label.Value == "cloudzero_namespace_labels" ||
						label.Value == "cloudzero_namespace_annotations") {
					namespaceMetrics = append(namespaceMetrics, ts)
					break
				}
			}
		}
	}

	return namespaceMetrics
}

// ValidateNamespaceMetrics validates that expected namespace metrics were received
func (mc *MockCollector) ValidateNamespaceMetrics(expectedNamespaces []string, expectedLabels []string) error {
	namespaceMetrics := mc.GetNamespaceMetrics()

	if len(namespaceMetrics) == 0 {
		return fmt.Errorf("no namespace metrics received")
	}

	foundNamespaces := make(map[string]bool)

	for _, ts := range namespaceMetrics {
		var namespaceName string
		var metricType string

		// Extract namespace name and metric type from labels
		for _, label := range ts.Labels {
			if label.Name == "namespace" {
				namespaceName = label.Value
			}
			if label.Name == "__name__" {
				metricType = label.Value
			}
		}

		if namespaceName != "" {
			foundNamespaces[namespaceName] = true
		}

		fmt.Printf("Found metric: %s for namespace: %s\n", metricType, namespaceName)
	}

	// Check if all expected namespaces were found
	for _, expectedNs := range expectedNamespaces {
		if !foundNamespaces[expectedNs] {
			return fmt.Errorf("expected namespace %s not found in metrics", expectedNs)
		}
	}

	return nil
}

// Clear clears all received data
func (mc *MockCollector) Clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.receivedData = make([]prompb.WriteRequest, 0)
}
