// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metricio

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang/snappy"
	"github.com/prometheus/prometheus/prompb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
)

const (
	// DefaultTimeout is the default HTTP request timeout.
	DefaultTimeout = 30 * time.Second
	// DefaultMaxRetries is the default number of retries for failed requests.
	DefaultMaxRetries = 3
)

// RemoteWriter sends metrics to a Prometheus remote_write endpoint.
type RemoteWriter struct {
	url        string
	client     *http.Client
	timeout    time.Duration
	maxRetries int
}

// NewRemoteWriter creates a new RemoteWriter.
func NewRemoteWriter(url string) *RemoteWriter {
	return &RemoteWriter{
		url: url,
		client: &http.Client{
			Timeout: DefaultTimeout,
		},
		timeout:    DefaultTimeout,
		maxRetries: DefaultMaxRetries,
	}
}

// Write sends a batch of TimeSeries to the remote_write endpoint.
func (w *RemoteWriter) Write(ctx context.Context, timeSeries []prompb.TimeSeries) error {
	if len(timeSeries) == 0 {
		return nil
	}

	writeRequest := &prompb.WriteRequest{
		Timeseries: timeSeries,
	}

	data, err := proto.Marshal(protoadapt.MessageV2Of(writeRequest))
	if err != nil {
		return fmt.Errorf("failed to marshal WriteRequest: %w", err)
	}

	compressed := snappy.Encode(nil, data)

	var lastErr error
	for attempt := range w.maxRetries {
		if attempt > 0 {
			// Exponential backoff
			time.Sleep(time.Duration(1<<attempt) * time.Second)
		}

		err = w.doRequest(ctx, compressed)
		if err == nil {
			return nil
		}
		lastErr = err
	}

	return fmt.Errorf("failed after %d retries: %w", w.maxRetries, lastErr)
}

func (w *RemoteWriter) doRequest(ctx context.Context, data []byte) error {
	ctx, cancel := context.WithTimeout(ctx, w.timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", w.url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
