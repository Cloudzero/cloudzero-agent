// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package types defines core metric data structures for the CloudZero Agent's cost allocation pipeline.
//
//coverage:ignore
package types

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Label represents a Prometheus-style label with name-value pair for metric metadata.
// Used throughout the metric collection and processing pipeline to maintain compatibility
// with Prometheus remote_write protocol and CloudZero's cost allocation requirements.
type Label struct {
	// Name is the label key, following Prometheus naming conventions (__name__, node, etc.).
	Name string `json:"name"`

	// Value is the label value, containing the actual metadata for cost allocation analysis.
	Value string `json:"value"`
}

// Sample represents a single metric sample with timestamp and value.
// Follows the Prometheus TimeSeries format for remote_write protocol compatibility.
type Sample struct {
	// Value is the metric value as a float64 pointer to support null values in time series data.
	Value *float64 `json:"value"`

	// Timestamp is the Unix timestamp in milliseconds when the metric was observed, stored as string for JSON compatibility.
	Timestamp string `json:"timestamp"`
}

// TimeSeries represents a complete Prometheus time series with labels and samples.
// This structure matches the Prometheus remote_write TimeSeries protobuf format
// for seamless integration with Prometheus collectors and the CloudZero platform.
type TimeSeries struct {
	// Labels contains all metric labels including __name__, resource identifiers, and custom tags.
	Labels []Label `json:"labels"`

	// Samples contains the time series data points with timestamps and values.
	Samples []Sample `json:"samples"`
}

// InputData represents the top-level structure for Prometheus remote_write requests.
// Received by the collector from Prometheus and other metric sources for processing
// into CloudZero's cost allocation format.
type InputData struct {
	// TimeSeries contains all metric series in the remote_write batch.
	TimeSeries []TimeSeries `json:"timeseries"` //nolint:tagliatelle // matches Prometheus remote_write format
}

// Metric represents the internal CloudZero metric format used throughout the processing pipeline.
// This is the canonical format after conversion from Prometheus TimeSeries, containing
// all necessary data for cost allocation analysis and upstream transmission.
type Metric struct {
	// ID is a unique identifier for this metric record, generated during collection.
	ID uuid.UUID

	// ClusterName identifies the Kubernetes cluster this metric originated from.
	ClusterName string

	// CloudAccountID identifies the cloud provider account for cost allocation.
	CloudAccountID string

	// MetricName is the Prometheus metric name, typically extracted from the __name__ label.
	MetricName string

	// NodeName identifies the specific node where this metric was collected, if applicable.
	NodeName string

	// CreatedAt is when this metric record was created in the CloudZero Agent system.
	CreatedAt time.Time

	// TimeStamp is the original metric observation time from the Prometheus sample.
	TimeStamp time.Time

	// Labels contains all metric labels excluding those hoisted to dedicated fields (like __name__, node).
	Labels map[string]string

	// Value is the metric value as a string to preserve precision during serialization.
	Value string
}

// ParquetMetric represents the optimized format for long-term storage and analytics.
// This format partitions data by time components (year, month, day, hour) for efficient
// querying and storage in columnar format, with labels serialized as JSON for flexibility.
type ParquetMetric struct {
	// ClusterName identifies the source Kubernetes cluster for cost allocation.
	ClusterName string `parquet:"cluster_name"`

	// CloudAccountID identifies the cloud provider account for billing analysis.
	CloudAccountID string `parquet:"cloud_account_id"`

	// Year provides time-based partitioning for efficient data warehouse queries.
	Year string `parquet:"year"`

	// Month provides monthly partitioning for billing cycle analysis.
	Month string `parquet:"month"`

	// Day provides daily partitioning for fine-grained cost tracking.
	Day string `parquet:"day"`

	// Hour provides hourly partitioning for detailed usage analysis.
	Hour string `parquet:"hour"`

	// MetricName is the Prometheus metric identifier for data classification.
	MetricName string `parquet:"metric_name"`

	// NodeName identifies the source node for infrastructure cost attribution.
	NodeName string `parquet:"node_name"`

	// CreatedAt is the metric creation timestamp in Unix milliseconds for efficient storage.
	CreatedAt int64 `parquet:"created_at,timestamp"`

	// TimeStamp is the original observation timestamp in Unix milliseconds.
	TimeStamp int64 `parquet:"timestamp,timestamp"`

	// Labels contains all metric metadata serialized as JSON for flexible querying.
	Labels string `parquet:"labels"`

	// Value is the metric measurement preserved as string for precision.
	Value string `parquet:"value"`
}

// Metric converts a ParquetMetric back to the standard internal Metric format.
// This is used when reading data from long-term storage for processing or transmission.
// The conversion reconstructs timestamps from Unix milliseconds and deserializes
// the JSON labels back into a map structure.
func (pm *ParquetMetric) Metric() Metric {
	m := Metric{
		ClusterName:    pm.ClusterName,
		CloudAccountID: pm.CloudAccountID,
		MetricName:     pm.MetricName,
		NodeName:       pm.NodeName,
		CreatedAt:      time.UnixMilli(pm.CreatedAt).UTC(),
		TimeStamp:      time.UnixMilli(pm.TimeStamp).UTC(),
		Value:          pm.Value,
	}

	labels := map[string]string{}
	if err := json.Unmarshal([]byte(pm.Labels), &labels); err != nil {
		log.Ctx(context.Background()).Fatal().Err(err).Msg("failed to unmarshal labels")
	}

	m.ImportLabels(labels)
	return m
}

// Parquet converts a Metric to ParquetMetric format for efficient columnar storage.
// This transformation partitions the data by time components (year, month, day, hour)
// and serializes labels as JSON for flexible querying in data warehouse systems.
func (m *Metric) Parquet() ParquetMetric {
	labelsData, err := json.Marshal(m.FullLabels())
	if err != nil {
		log.Ctx(context.Background()).Fatal().Err(err).Msg("failed to marshal labels")
	}

	return ParquetMetric{
		ClusterName:    m.ClusterName,
		CloudAccountID: m.CloudAccountID,
		Year:           m.TimeStamp.Format("2006"),
		Month:          m.TimeStamp.Format("01"),
		Day:            m.TimeStamp.Format("02"),
		Hour:           m.TimeStamp.Format("15"),
		MetricName:     m.MetricName,
		NodeName:       m.NodeName,
		CreatedAt:      m.CreatedAt.UnixMilli(),
		TimeStamp:      m.TimeStamp.UnixMilli(),
		Labels:         string(labelsData),
		Value:          m.Value,
	}
}

// jsonMetric is an internal struct for JSON serialization/deserialization of Metric.
// This intermediate format handles the conversion between Go's type-safe fields
// and JSON's string-based representations, particularly for UUIDs and timestamps.
type jsonMetric struct {
	// ID is the metric UUID serialized as string for JSON compatibility.
	ID string `json:"id"`

	// ClusterName identifies the source Kubernetes cluster.
	ClusterName string `json:"cluster_name"` //nolint:tagliatelle // matches CloudZero API format

	// CloudAccountID identifies the cloud provider account.
	CloudAccountID string `json:"cloud_account_id"` //nolint:tagliatelle // matches CloudZero API format

	// MetricName is the Prometheus metric identifier.
	MetricName string `json:"metric_name"` //nolint:tagliatelle // matches CloudZero API format

	// NodeName identifies the source Kubernetes node.
	NodeName string `json:"node_name"` //nolint:tagliatelle // matches CloudZero API format

	// CreatedAt is the creation timestamp as Unix milliseconds string.
	CreatedAt string `json:"created_at"` //nolint:tagliatelle // matches CloudZero API format

	// TimeStamp is the observation timestamp as Unix milliseconds string.
	TimeStamp string `json:"timestamp"` //nolint:tagliatelle // matches CloudZero API format

	// Labels contains all metric metadata for cost allocation.
	Labels map[string]string `json:"labels"`

	// Value is the metric measurement preserved as string.
	Value string `json:"value"`
}

// JSON converts a Metric to a generic map for flexible JSON serialization.
// This method is used internally by MarshalJSON to create the CloudZero API format
// with consistent field naming and timestamp formatting.
func (m *Metric) JSON() map[string]interface{} {
	return map[string]interface{}{
		"id":               m.ID.String(),
		"cluster_name":     m.ClusterName,
		"cloud_account_id": m.CloudAccountID,
		"metric_name":      m.MetricName,
		"node_name":        m.NodeName,
		"created_at":       strconv.FormatInt(m.CreatedAt.UnixMilli(), 10),
		"timestamp":        strconv.FormatInt(m.TimeStamp.UnixMilli(), 10),
		"labels":           m.Labels,
		"value":            m.Value,
	}
}

// MarshalJSON implements the json.Marshaler interface for Metric.
// Uses the JSON() method to ensure consistent serialization format
// compatible with the CloudZero API specification.
func (m Metric) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.JSON())
}

// UnmarshalJSON implements the json.Unmarshaler interface for Metric.
// Handles conversion from string-based JSON fields back to proper Go types,
// including UUID parsing and timestamp conversion from Unix milliseconds.
func (m *Metric) UnmarshalJSON(data []byte) error {
	var aux jsonMetric
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	{
		var err error
		m.ID, err = uuid.Parse(aux.ID)
		if err != nil {
			return fmt.Errorf("failed to parse id: %w", err)
		}
	}

	m.ClusterName = aux.ClusterName
	m.CloudAccountID = aux.CloudAccountID
	m.MetricName = aux.MetricName
	m.NodeName = aux.NodeName
	m.Value = aux.Value

	if createdAt, err := strconv.ParseInt(aux.CreatedAt, 10, 64); err == nil {
		m.CreatedAt = time.UnixMilli(createdAt).UTC()
	} else {
		return fmt.Errorf("failed to parse created_at: %w", err)
	}
	if timestamp, err := strconv.ParseInt(aux.TimeStamp, 10, 64); err == nil {
		m.TimeStamp = time.UnixMilli(timestamp).UTC()
	} else {
		return fmt.Errorf("failed to parse timestamp: %w", err)
	}

	m.ImportLabels(aux.Labels)

	return nil
}

// MetricRange represents a paginated collection of metrics for API responses.
// This structure supports efficient streaming and pagination of large metric datasets
// from storage systems to the CloudZero platform.
type MetricRange struct {
	// Metrics contains the current page of metric records.
	Metrics []Metric `json:"metrics"`

	// Next is an optional pagination token for retrieving the next page of results.
	Next *string `json:"next,omitempty"`
}

// ImportLabels imports labels from a map. This is similar to setting the Labels
// field to labels, except for special-case labels are used to set fields on the
// metric.
//
// Note that the fields will only be set if the label is present in the map, so
// it will not overwrite existing values unless the relevant label is actually
// found.
func (m *Metric) ImportLabels(labels map[string]string) {
	dest := map[string]string{}
	if m.Labels != nil {
		maps.Copy(dest, m.Labels)
	}

	for k, v := range labels {
		switch k {
		case "__name__":
			m.MetricName = v
			continue
		case "node":
			m.NodeName = v
			continue
		}
		dest[k] = v
	}

	m.Labels = dest
}

// FullLabels returns a map of all labels, including ones which have been
// hoisted out to fields.
func (m *Metric) FullLabels() map[string]string {
	labels := map[string]string{}

	maps.Copy(labels, m.Labels)
	if m.MetricName != "" {
		labels["__name__"] = m.MetricName
	}
	if m.NodeName != "" {
		labels["node"] = m.NodeName
	}

	return labels
}
