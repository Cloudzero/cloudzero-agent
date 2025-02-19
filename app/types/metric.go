// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//coverage:ignore
package types

import (
	"time"

	"github.com/go-obvious/timestamp"
	"github.com/google/uuid"
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
	ID             string            `json:"id"               parquet:"-"`
	ClusterName    string            `json:"cluster_name"     parquet:"cluster_name"`     //nolint:tagliatelle // we should keep these consistent
	CloudAccountID string            `json:"cloud_account_id" parquet:"cloud_account_id"` //nolint:tagliatelle // we should keep these consistent
	Year           string            `json:"year"             parquet:"year"`
	Month          string            `json:"month"            parquet:"month"`
	Day            string            `json:"day"              parquet:"day"`
	Hour           string            `json:"hour"             parquet:"hour"`
	MetricName     string            `json:"metric_name"      parquet:"metric_name"`                       //nolint:tagliatelle // we should keep these consistent
	NodeName       string            `json:"node_name"        parquet:"node_name"`                         //nolint:tagliatelle // we should keep these consistent
	CreatedAt      int64             `json:"created_at"       parquet:"created_at,timestamp(microsecond)"` //nolint:tagliatelle // we should keep these consistent
	TimeStamp      int64             `json:"timestamp"        parquet:"timestamp,timestamp(microsecond)"`  //nolint:tagliatelle // "timestamp" is one word, tagliatelle wants timeStamp.
	Labels         map[string]string `json:"labels"           parquet:"labels"`
	Value          string            `json:"value"            parquet:"value"`
}

func NewMetric(cloudAccountID, clusterName, name, nodeName string, timeStamp int64, labels map[string]string, value string) Metric {
	if labels == nil {
		labels = make(map[string]string)
	}
	createAt := timestamp.Milli()
	t := time.Unix(0, timeStamp*int64(time.Millisecond))
	year := GetYear(t)
	month := GetMonth(t)
	day := GetDay(t)
	hour := GetHour(t)
	return Metric{
		ID:             uuid.New().String(),
		CloudAccountID: cloudAccountID,
		ClusterName:    clusterName,
		MetricName:     name,
		NodeName:       nodeName,
		CreatedAt:      createAt,
		Year:           year,
		Month:          month,
		Day:            day,
		Hour:           hour,
		TimeStamp:      timeStamp,
		Labels:         labels,
		Value:          value,
	}
}

type MetricRange struct {
	Metrics []Metric `json:"metrics"`
	Next    *string  `json:"next,omitempty"`
}

// GetYear extracts the year as a string with four digits.
func GetYear(t time.Time) string {
	return t.Format("2006")
}

// GetMonth extracts the month as a string with two digits (leading zero if necessary).
func GetMonth(t time.Time) string {
	return t.Format("01")
}

// GetDay extracts the day as a string with two digits (leading zero if necessary).
func GetDay(t time.Time) string {
	return t.Format("02")
}

// GetHour extracts the hour as a string with two digits (leading zero if necessary) in 24-hour format.
func GetHour(t time.Time) string {
	return t.Format("15")
}
