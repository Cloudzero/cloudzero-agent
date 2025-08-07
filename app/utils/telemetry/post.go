// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package telemetry provides functionality for collecting and transmitting operational
// telemetry data from the CloudZero agent to the CloudZero API for monitoring and analytics.
//
// This package handles the complete telemetry pipeline:
//
//   - Status collection: Gathering cluster and agent operational status
//   - Data serialization: Converting status to Protocol Buffer format
//   - Compression: Optimizing payload size for transmission (currently disabled)
//   - Secure transmission: HTTPS delivery with bearer token authentication
//   - Configuration respect: Honors telemetry disable flags for privacy
//
// Telemetry data includes:
//   - Agent health and performance metrics
//   - Cluster discovery and configuration status
//   - Error rates and operational statistics
//   - Resource utilization and processing metrics
//
// The telemetry system is designed to be:
//   - Privacy-aware: Can be completely disabled via configuration
//   - Lightweight: Minimal impact on agent performance
//   - Secure: Uses encrypted transmission with API key authentication
//   - Reliable: Includes timeout handling and error recovery
//
// API integration:
//   - Endpoint: /v1/container-metrics/status
//   - Format: Protocol Buffer over HTTPS
//   - Authentication: Bearer token with CloudZero API key
//   - Timeout: 15 seconds to match AWS API Gateway limits
//
// Usage:
//   err := telemetry.Post(ctx, httpClient, config, statusAccessor)
//   if err != nil {
//       log.Printf("Telemetry upload failed: %v", err)
//   }
package telemetry

import (
	"bytes"
	"context"
	"fmt"
	net "net/http"
	"strconv"
	"time"

	"google.golang.org/protobuf/proto"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	http "github.com/cloudzero/cloudzero-agent/app/http/client"
	pb "github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	// PostTimeout defines the maximum time allowed for individual HTTP request operations.
	// This timeout is shorter than the overall Timeout to allow for retries and connection setup.
	PostTimeout = 5 * time.Second
)

const (
	// URLPath defines the API endpoint path for telemetry status uploads.
	// This path is appended to the CloudZero host URL to form the complete endpoint.
	URLPath = "/v1/container-metrics/status"
)

const (
	// Timeout defines the maximum duration for telemetry upload operations.
	// This value matches AWS API Gateway timeout limits to prevent gateway-level
	// timeouts that would result in unclear error conditions.
	Timeout = 15 * time.Second
)

// Post uploads operational telemetry data to the CloudZero API for monitoring and analytics.
//
// This function handles the complete telemetry upload process:
//   1. Validates configuration and required parameters
//   2. Respects telemetry disable flags for privacy compliance
//   3. Serializes cluster status data using Protocol Buffers
//   4. Compresses data for efficient transmission (currently no-op)
//   5. Transmits data securely via HTTPS with authentication
//
// The function is designed to be non-blocking and fault-tolerant - failures in
// telemetry upload should not impact agent operations.
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - client: HTTP client for API requests (uses default client if nil)
//   - cfg: Agent configuration containing CloudZero API credentials and settings
//   - accessor: Interface for accessing cluster status data to upload
//
// Returns:
//   - nil: Telemetry successfully uploaded or telemetry disabled
//   - error: Configuration errors, serialization failures, or network errors
//
// Configuration requirements:
//   - cfg.Cloudzero.Host: CloudZero API base URL
//   - cfg.Cloudzero.Credential: Valid API key for authentication
//   - cfg.Deployment.AccountID: Account identifier for data routing
//   - cfg.Deployment.Region: Region information for context
//   - cfg.Deployment.ClusterName: Cluster identifier for data organization
//
// Privacy considerations:
//   - Returns immediately without action if cfg.Cloudzero.DisableTelemetry is true
//   - All data is transmitted over encrypted HTTPS connections
//   - Authentication uses secure bearer token mechanism
//
// Example:
//   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//   defer cancel()
//   
//   err := telemetry.Post(ctx, nil, agentConfig, statusAccessor)
//   if err != nil {
//       log.Printf("Telemetry upload failed (non-critical): %v", err)
//   }
func Post(ctx context.Context, client *net.Client, cfg *config.Settings, accessor pb.Accessor) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if accessor == nil {
		return errors.New("nil accessor")
	}

	if cfg.Cloudzero.Host == "" {
		return errors.New("missing cloudzero host")
	}

	if cfg.Cloudzero.Credential == "" {
		return errors.New("missing cloudzero api key")
	}

	// quietly exit
	if cfg.Cloudzero.DisableTelemetry {
		return nil
	}

	if client == nil {
		client = net.DefaultClient
	}

	var (
		err  error
		data []byte
	)
	accessor.ReadFromReport(func(cs *pb.ClusterStatus) {
		data, err = proto.Marshal(cs)
		logrus.Info("marshalled cluster status: " + strconv.Itoa(len(data)) + " bytes")
	})
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return errors.New("no data to post")
	}
	buf, err := compress(data)
	if err != nil {
		return err
	}

	logrus.Infof("compressed size is: %d bytes", buf.Len())

	endpoint := fmt.Sprintf("%s%s", cfg.Cloudzero.Host, URLPath)
	_, err = http.Do(ctx, client, net.MethodPost,
		map[string]string{
			http.HeaderAuthorization: "Bearer " + cfg.Cloudzero.Credential,
			http.HeaderContentType:   http.ContentTypeProtobuf,
		},
		map[string]string{
			http.QueryParamAccountID:   cfg.Deployment.AccountID,
			http.QueryParamRegion:      cfg.Deployment.Region,
			http.QueryParamClusterName: cfg.Deployment.ClusterName,
		},
		endpoint,
		&buf,
	)

	return err
}

// compress prepares telemetry data for transmission by applying compression algorithms.
//
// Currently, this function performs a no-op copy of the data without actual compression.
// The original Snappy compression implementation was disabled because it was increasing
// payload size rather than reducing it for typical telemetry data sizes.
//
// Future implementations may:
//   - Re-enable compression for larger payloads
//   - Use different compression algorithms (gzip, brotli)
//   - Apply compression conditionally based on payload size
//
// Parameters:
//   - data: Raw telemetry data to be compressed
//
// Returns:
//   - bytes.Buffer: Buffer containing the processed data
//   - error: Processing error (currently always nil)
//
// Note: The commented code shows the previous Snappy compression attempt
// which was disabled due to size increase rather than reduction.
func compress(data []byte) (bytes.Buffer, error) {
	var buf bytes.Buffer
	if _, err := buf.Write(data); err != nil {
		return bytes.Buffer{}, errors.Wrap(err, "write data to buffer")
	}

	// XXX: Compression is taking more space than the buffer alone! We gain nothing!
	// snappyWriter := snappy.NewBufferedWriter(&buf)
	// _, err := snappyWriter.Write(data)
	// if err != nil {
	// 	return bytes.Buffer{}, errors.Wrap(err, "data compress")
	// }

	// if err = snappyWriter.Close(); err != nil {
	// 	return bytes.Buffer{}, errors.Wrap(err, "close writer failed")
	// }

	return buf, nil
}
