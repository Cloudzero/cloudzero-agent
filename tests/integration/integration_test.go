//go:build integration
// +build integration

// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestIntegrationValidResponses(t *testing.T) {
	tests := []struct {
		name           string
		requests       []Request
		route          string
		expectedStatus int
	}{
		{
			name: "Valid route request should return 200",
			requests: []Request{
				{
					QueryParams: map[string]string{
						"cluster_name":     ValidClusterName,
						"cloud_account_id": ValidAccountID,
						"region":           ValidRegion,
					}, Body: []byte{},
					Route:  PodRoute,
					Method: http.MethodPost,
				},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, req := range tt.requests {
				log.Info().Msg("Running test")

				admissionReq := NewAdmissionRequest()
				if admissionReq == nil {
					t.Fatalf("Failed to create test admission request")
				}
				httpReq, err := GenerateRequest(req.Method, req.Route, BaseURL, Request{Body: admissionReq, QueryParams: req.QueryParams})
				if err != nil {
					t.Fatalf("Failed to generate request: %v", err)
				}

				resp, err := http.DefaultClient.Do(httpReq)
				if err != nil {
					t.Fatalf("Failed to send request: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
				}

				if resp.StatusCode == http.StatusOK {
					body, err := io.ReadAll(resp.Body)
					if err != nil {
						t.Fatalf("Failed to read response body: %v", err)
					}
					t.Logf("Response body: %s", body)
				}
				time.Sleep(10 * time.Second)
				filePath := filepath.Join(TestOutputDir, "output.json")
				_, err = os.Stat(filePath)
				assert.NoError(t, err)

				file, err := os.Open(filePath)
				if err != nil {
					t.Fatalf("Failed to open file: %v", err)
				}
				defer file.Close()

				content, err := io.ReadAll(file)
				assert.NoError(t, err)

				var result map[string]string
				err = json.Unmarshal(content, &result)
				assert.NoError(t, err)
			}
		})
	}
	t.Cleanup(func() {
		files, err := os.ReadDir(TestOutputDir)
		if err != nil {
			t.Fatalf("Failed to read directory: %v", err)
		}
		for _, file := range files {
			err := os.Remove(fmt.Sprintf("%s/%s", TestOutputDir, file.Name()))
			if err != nil {
				t.Fatalf("Failed to remove test file: %v", err)
			}
		}
	})
}

func TestIntegrationInvalidResponses(t *testing.T) {
	tests := []struct {
		name           string
		requests       []Request
		method         string
		route          string
		expectedStatus int
	}{
		{
			name:   "Invalid route request should return 404",
			method: http.MethodPost,
			requests: []Request{
				{
					QueryParams: map[string]string{}, Body: []byte{},
					Route: "/invalid-route",
				},
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, req := range tt.requests {
				log.Info().Msg("Running test")

				foo := NewAdmissionRequest()
				if foo == nil {
					t.Fatalf("Failed to create fake request")
				}
				httpReq, err := GenerateRequest(req.Method, req.Route, BaseURL, Request{Body: foo, QueryParams: req.QueryParams})
				if err != nil {
					t.Fatalf("Failed to generate request: %v", err)
				}

				resp, err := http.DefaultClient.Do(httpReq)
				if err != nil {
					t.Fatalf("Failed to send request: %v", err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != tt.expectedStatus {
					t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
				}

				if resp.StatusCode == http.StatusOK {
					body, err := io.ReadAll(resp.Body)
					if err != nil {
						t.Fatalf("Failed to read response body: %v", err)
					}
					t.Logf("Response body: %s", body)
				}
			}
		})
	}
}
