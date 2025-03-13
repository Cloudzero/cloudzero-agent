// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

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

type Label struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Sample struct {
	Value     *float64 `json:"value"`
	Timestamp string   `json:"timestamp"`
}

type TimeSeries struct {
	Labels  []Label  `json:"labels"`
	Samples []Sample `json:"samples"`
}

type InputData struct {
	TimeSeries []TimeSeries `json:"timeseries"` //nolint:tagliatelle // "time_series" might be right; does this need to match something else?
}

type Metric struct {
	ID             uuid.UUID
	ClusterName    string
	CloudAccountID string
	MetricName     string
	NodeName       string
	CreatedAt      time.Time
	TimeStamp      time.Time
	Labels         map[string]string
	Value          string
}

type ParquetMetric struct {
	ClusterName    string `parquet:"cluster_name"`
	CloudAccountID string `parquet:"cloud_account_id"`
	Year           string `parquet:"year"`
	Month          string `parquet:"month"`
	Day            string `parquet:"day"`
	Hour           string `parquet:"hour"`
	MetricName     string `parquet:"metric_name"`
	NodeName       string `parquet:"node_name"`
	CreatedAt      int64  `parquet:"created_at,timestamp"`
	TimeStamp      int64  `parquet:"timestamp,timestamp"`
	Labels         string `parquet:"labels"`
	Value          string `parquet:"value"`
}

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

type jsonMetric struct {
	ID             string            `json:"id"`
	ClusterName    string            `json:"cluster_name"`     //nolint:tagliatelle // we should keep these consistent
	CloudAccountID string            `json:"cloud_account_id"` //nolint:tagliatelle // we should keep these consistent
	MetricName     string            `json:"metric_name"`      //nolint:tagliatelle // we should keep these consistent
	NodeName       string            `json:"node_name"`        //nolint:tagliatelle // we should keep these consistent
	CreatedAt      string            `json:"created_at"`       //nolint:tagliatelle // we should keep these consistent
	TimeStamp      string            `json:"timestamp"`        //nolint:tagliatelle // we should keep these consistent
	Labels         map[string]string `json:"labels"`
	Value          string            `json:"value"`
}

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

func (m Metric) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.JSON())
}

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

type MetricRange struct {
	Metrics []Metric `json:"metrics"`
	Next    *string  `json:"next,omitempty"`
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
