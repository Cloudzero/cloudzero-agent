// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package logging provides structured logging infrastructure for CloudZero Agent operations.
package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// storeWriter implements a logging sink that converts log events to CloudZero metrics for storage.
// This writer transforms structured log events into CloudZero metric format, enabling log events
// to be processed through the CloudZero cost allocation and analysis pipeline alongside other metrics.
//
// Log-to-metric transformation enables:
//   - Unified data pipeline: Log events processed with other CloudZero metrics
//   - Cost allocation: Log volume and patterns attributed to specific resources
//   - Operational analytics: Log metrics integrated with CloudZero insights
//   - Storage efficiency: Leverages existing metric storage and compression
//
// The writer maintains CloudZero context information (cluster, cloud account) to ensure
// proper attribution and correlation of log events within the broader cost optimization framework.
type storeWriter struct {
	// store handles metric persistence through CloudZero Agent storage infrastructure.
	// Log events are converted to metrics and stored using the same pipeline as other
	// CloudZero metrics, enabling unified storage, processing, and analysis.
	store types.WritableStore

	// ctx provides request context for storage operations and cancellation.
	// This context enables proper timeout handling and resource cleanup
	// during log event storage operations.
	ctx context.Context

	// clusterName identifies the Kubernetes cluster for cost allocation attribution.
	// This field ensures log events are properly attributed to specific clusters
	// in CloudZero cost optimization analysis.
	clusterName string

	// cloudAccountID identifies the cloud provider account for billing correlation.
	// This field enables correlation of log events with cloud provider billing data
	// for comprehensive cost optimization insights.
	cloudAccountID string
}

// Write implements io.Writer interface for CloudZero metric storage integration.
// This method converts structured JSON log events from Zerolog into CloudZero metrics,
// enabling log events to be processed through the CloudZero cost allocation pipeline.
//
// Transformation process:
//  1. Parse JSON log event from Zerolog output
//  2. Extract message content and timestamp information
//  3. Convert all log fields to string-based key-value pairs
//  4. Create CloudZero metric with "log" metric name and message as value
//  5. Store metric through CloudZero Agent storage infrastructure
//
// Field conversion:
//   - Strings: Direct assignment to string map
//   - Numbers: Formatted as string with appropriate precision
//   - Booleans: Converted to "true"/"false" strings
//   - Objects/Arrays: JSON-marshaled to string representation
//   - Null values: Represented as "null" string
//
// Error handling:
//   - JSON parse errors: Log event is consumed without storage (non-blocking)
//   - Storage errors: Ignored to prevent logging failures from disrupting operations
//   - Field conversion errors: Fallback to string representation
//
// This writer enables CloudZero Agent to treat log events as first-class metrics,
// providing unified visibility into both application metrics and operational logs
// within the CloudZero cost optimization and analysis framework.
func (s *storeWriter) Write(p []byte) (n int, err error) {
	var logEntry map[string]interface{}
	if err := json.Unmarshal(p, &logEntry); err != nil {
		// Handle error
		return len(p), nil // Consume log even if unmarshal fails
	}

	// pull out the message itself
	var msg string
	if m, exists := logEntry[zerolog.MessageFieldName]; exists {
		// attempt to read as string or fallback
		if mStr, ok := m.(string); ok {
			msg = mStr
		} else {
			msg = fmt.Sprintf("%v", m)
		}
	}

	// parse the timestamp
	var ts time.Time
	if t, exists := logEntry[zerolog.TimestampFieldName]; exists {
		if tStr, ok := t.(string); ok {
			if parsed, err := time.Parse(zerolog.TimeFieldFormat, tStr); err == nil {
				ts = parsed
			}
		}
	}

	// convert the object blob to a string map
	stringMap := make(map[string]string)

	for key, value := range logEntry {
		if value == nil {
			stringMap[key] = "null"
			continue
		}

		switch v := value.(type) {
		case string:
			stringMap[key] = v
		case float64:
			stringMap[key] = fmt.Sprintf("%g", v)
		case bool:
			stringMap[key] = strconv.FormatBool(v)
		case map[string]interface{}, []interface{}:
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				stringMap[key] = fmt.Sprintf("%v", v)
			} else {
				stringMap[key] = string(jsonBytes)
			}
		default:
			stringMap[key] = fmt.Sprintf("%v", v)
		}
	}

	_ = s.store.Put(s.ctx, types.Metric{
		ID:             uuid.New(),
		ClusterName:    s.clusterName,
		CloudAccountID: s.cloudAccountID,
		MetricName:     "log",
		CreatedAt:      ts,
		TimeStamp:      ts,
		Labels:         stringMap,
		Value:          msg,
	})

	return len(p), nil
}

// StoreWriter creates a CloudZero metric storage writer for log event integration.
// This constructor creates a writer that transforms structured log events into CloudZero metrics,
// enabling log data to be processed through the CloudZero cost allocation and analysis pipeline.
//
// Integration benefits:
//   - Unified data pipeline: Log events stored alongside other CloudZero metrics
//   - Cost attribution: Log volume attributed to specific clusters and cloud accounts
//   - Operational insights: Log patterns integrated with CloudZero analytics
//   - Storage efficiency: Leverages CloudZero metric storage and compression
//
// Parameters:
//   - ctx: Context for storage operations and lifecycle management
//   - store: WritableStore for CloudZero metric persistence
//   - clusterName: Kubernetes cluster identification for cost attribution
//   - cloudAccountID: Cloud provider account for billing correlation
//
// The returned writer transforms every log event into a CloudZero metric with:
//   - Metric name: "log"
//   - Value: Log message content
//   - Labels: All log event fields as key-value pairs
//   - Timestamps: Preserved from original log events
//   - Context: Cluster and cloud account attribution
//
// This enables CloudZero platform to analyze log patterns, volume, and costs
// alongside other infrastructure and application metrics.
func StoreWriter(
	ctx context.Context,
	store types.WritableStore,
	clusterName string,
	cloudAccountID string,
) io.Writer {
	return &storeWriter{
		store:          store,
		ctx:            ctx,
		clusterName:    clusterName,
		cloudAccountID: cloudAccountID,
	}
}
