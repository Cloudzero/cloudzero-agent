// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package webhook_server_test

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/diagnostic/webhook_server"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/status"
	"github.com/stretchr/testify/assert"
)

func makeReport() status.Accessor {
	return status.NewAccessor(&status.ClusterStatus{})
}

func TestCheck(t *testing.T) {
	tests := []struct {
		name          string
		serverFunc    func() *httptest.Server
		wantPass      bool
		expectedError string
	}{
		{
			name: "TLS enabled endpoint",
			serverFunc: func() *httptest.Server {
				return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			wantPass:      true,
			expectedError: "",
		},
		{
			name: "Non-TLS endpoint",
			serverFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			wantPass:      true,
			expectedError: "",
		},
		{
			name: "Failing endpoint",
			serverFunc: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			wantPass:      false,
			expectedError: "internal server error",
		},
		{
			name: "Unreachable endpoint",
			serverFunc: func() *httptest.Server {
				return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusServiceUnavailable)
				}))
			},
			wantPass:      false,
			expectedError: "service unavailable error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.serverFunc()
			defer server.Close()

			cfg := &config.Settings{
				Deployment: config.Deployment{
					WebhookServer: config.WebhookServer{
						ServiceAddress: server.URL,
						CACert:         server.TLS.Certificates[0].Certificate[0],
					},
				},
			}

			provider := webhook_server.NewProvider(context.Background(), cfg)
			client := &http.Client{
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
				},
			}

			accessor := makeReport()

			err := provider.Check(context.Background(), client, accessor)

			assert.NoError(t, err)
			accessor.ReadFromReport(func(s *status.ClusterStatus) {
				assert.Len(t, s.Checks, 1)
				assert.Equal(t, tt.expectedError, s.Checks[0].Error)
				assert.Equal(t, tt.wantPass, s.Checks[0].Passing)
			})
		})
	}
}
