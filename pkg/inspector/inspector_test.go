// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package inspector_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/cloudzero/cloudzero-agent/pkg/inspector"
	"github.com/google/go-cmp/cmp"

	"github.com/rs/zerolog"
)

func TestInspector_Inspect(t *testing.T) {
	tests := []struct {
		name          string
		i             *inspector.Inspector
		resp          *http.Response
		want          map[string]any
		wantErr       bool
		wantDecodeErr bool
	}{
		{
			name: "200 OK",
			resp: &http.Response{StatusCode: http.StatusOK},
			want: map[string]any{
				"level":   "debug",
				"message": "successful HTTP response",
				"status":  float64(http.StatusOK),
			},
		},
		{
			name: "404 Not Found",
			resp: &http.Response{StatusCode: http.StatusNotFound},
			want: map[string]any{
				"status":  float64(http.StatusNotFound),
				"level":   "warn",
				"message": "Unknown HTTP error",
			},
		},
		{
			name: "403 Forbidden generic",
			resp: &http.Response{
				StatusCode: http.StatusForbidden,
				Header: http.Header{
					"X-Foo": []string{"bar"},
				},
			},
			want: map[string]any{
				"status":  float64(http.StatusForbidden),
				"level":   "warn",
				"message": "Unknown HTTP 403 Forbidden error",
				"body":    "",
				"headers": map[string]any{
					"X-Foo": []any{"bar"},
				},
			},
		},
		{
			name: "invalid-json",
			resp: &http.Response{
				StatusCode: http.StatusForbidden,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(`{"foo": "bar"`)),
			},
			wantErr:       true,
			wantDecodeErr: true,
		},
		{
			name: "Invalid API key",
			resp: &http.Response{
				StatusCode: http.StatusForbidden,
				Header: http.Header{
					"Content-Type": []string{"application/json"},
				},
				Body: io.NopCloser(bytes.NewBufferString(`{"message": "User is not authorized to access this resource"}`)),
			},
			want: map[string]any{
				"status":       float64(http.StatusForbidden),
				"Content-Type": "application/json",
				"level":        "error",
				"message":      "Invalid CloudZero API key",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logOutput := bytes.Buffer{}
			ctx := context.Background()
			logger := zerolog.New(&logOutput)

			i := inspector.New()
			if err := i.Inspect(ctx, tt.resp, logger); (err != nil) != tt.wantErr {
				t.Errorf("Inspector.Inspect() error = %v, wantErr %v", err, tt.wantErr)
			}

			if logOutput.Len() == 0 && tt.want != nil {
				t.Errorf("Inspector.Inspect() logOutput = %v, want %v", logOutput.String(), tt.want)
			}

			logData := map[string]any{}
			if err := json.Unmarshal(logOutput.Bytes(), &logData); (err != nil) != tt.wantDecodeErr {
				t.Errorf("Inspector.Inspect() failed to decode log output: %v", err)
			} else if err != nil {
				return
			}

			if diff := cmp.Diff(logData, tt.want); diff != "" {
				t.Errorf("Inspector.Inspect() log output mismatch (-got +want):\n%s", diff)
			}
		})
	}
}
