// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package controller provides a mock insights controller.
package controller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/parallel"
	"github.com/cloudzero/cloudzero-agent/app/utils"
	"github.com/cloudzero/cloudzero-agent/mock/metrics"
	"github.com/golang/snappy"
	"github.com/google/uuid"
	"github.com/prometheus/prometheus/prompb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
)

type MockInsightsController struct {
	CollectorEndpoint string // endpoint to send metrics to
	APIKey            string // api key to use as auth for the collector
	TotalHours        int    // number of hours calculated as time.Now() - (time.Hour * `env.TOTAL_HOURS`)
	NumNodes          int    // total number of nodes
	PodsPerNode       int    // number of pods to generate data per node
	CPUPerNode        int64  // number of CPUs to use for each node
	MemPerNode        int64  // number of memory in bytes for each node
	NumBatches        int    // number of times to send the data
	ChunkSize         int    // size to break the metrics into when sending to the collector
}

func (c *MockInsightsController) Run() error {
	slog.Default().Info("Starting run for mock insights controller")

	if c.NumBatches == 0 {
		c.NumBatches = 1
	}

	totalSent := 0

	for i := range c.NumBatches {
		slog.Default().With("batch", i).Info("Processing batch")

		// create the time
		end := time.Now()
		start := end.Add(-time.Hour * time.Duration(c.TotalHours))

		// generate the metrics
		m := metrics.GenerateClusterMetrics("org-mock-controller", uuid.NewString()[:10], fmt.Sprintf("cluster-%s", uuid.NewString()[:6]), start, end, c.CPUPerNode, c.MemPerNode, c.NumNodes, c.PodsPerNode)

		// chunk
		chunks := utils.Chunk(m, c.ChunkSize)

		slog.Default().With("chunks", len(chunks)).Info("Chunked the generated metrics")

		// process each chunk in a thread pool
		pm := parallel.New(10)
		defer pm.Close()
		waiter := parallel.NewWaiter()

		slog.Default().With("chunks", len(chunks)).Info("Processing chunks")
		for i, chunk := range chunks {
			fn := func() error {
				slog.Default().With("chunk", i).Info("Processing chunk")
				// encode into a prom requrest
				req, err := metrics.EncodeV1(chunk)
				if err != nil {
					return fmt.Errorf("failed to encode into prom metrics: %w", err)
				}

				// send the request
				if err := c.pushMetrics(req.Timeseries); err != nil {
					return fmt.Errorf("failed to push metrics: %w", err)
				}

				slog.Default().With("chunk", i).Info("Successfully processed chunk")
				return nil
			}

			pm.Run(fn, waiter)
		}
		waiter.Wait()

		// check for errors in the waiter
		for err := range waiter.Err() {
			if err != nil {
				return fmt.Errorf("failed to upload files; %w", err)
			}
		}

		slog.Default().With("batch", i).Info("Finished batch")
		totalSent += len(m)
	}

	slog.Default().With("totalSent", totalSent).Info("Successfully sent all metrics")

	return nil
}

func (c *MockInsightsController) pushMetrics(timeSeries []prompb.TimeSeries) error {
	writeRequest := &prompb.WriteRequest{
		Timeseries: timeSeries,
	}

	data, err := proto.Marshal(protoadapt.MessageV2Of(writeRequest))
	if err != nil {
		return fmt.Errorf("failed to encode the metrics")
	}

	compressed := snappy.Encode(nil, data)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", c.CollectorEndpoint, bytes.NewBuffer(compressed))
	if err != nil {
		return fmt.Errorf("error creating HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send the request: %w", err)
	}

	if resp.StatusCode == http.StatusNoContent {
		slog.Default().Info("Successfully sent request")
		return nil
	}

	slog.Default().
		With("statusCode", resp.StatusCode).
		With("status", resp.Status).
		Error("ERROR IN REQUEST")

	// read the body
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read the body")
	}
	fmt.Println(string(raw))

	return errors.New("unknown error")
}
