// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//coverage:ignore
package types_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var simpleMetric = types.Metric{
	ID:             uuid.MustParse("d64271ef-46af-4ef9-94b6-c537a186b01d"),
	ClusterName:    "aws-cirrus-brahms",
	CloudAccountID: "8675309",
	MetricName:     "container_network_transmit_bytes_total",
	NodeName:       "ip-192-168-62-22.ec2.internal",
	CreatedAt:      time.UnixMilli(1740671645978).UTC(),
	TimeStamp:      time.UnixMilli(1740671634889).UTC(),
	Labels: map[string]string{
		"image":                     "602401143452.dkr.ecr.us-east-1.amazonaws.com/eks/pause:3.5",
		"instance":                  "ip-192-168-62-22.ec2.internal",
		"k8s_io_cloud_provider_aws": "eb707f9bdba15de05a26c5a3b4a909ee",
		"name":                      "340166e10e91263f42abc91459ed3523ced66250f87df0f945a5816dea321452",
		"namespace":                 "kube-system",
		"pod":                       "kube-proxy-9bnjh",
	},
	Value: "990",
}

func TestMetric_JSON(t *testing.T) {
	tests := []struct {
		name   string
		metric types.Metric
		want   map[string]interface{}
	}{
		{
			name:   "basic",
			metric: simpleMetric,
			want: map[string]interface{}{
				"id":               "d64271ef-46af-4ef9-94b6-c537a186b01d",
				"cluster_name":     "aws-cirrus-brahms",
				"cloud_account_id": "8675309",
				"metric_name":      "container_network_transmit_bytes_total",
				"node_name":        "ip-192-168-62-22.ec2.internal",
				"created_at":       "1740671645978",
				"timestamp":        "1740671634889",
				"labels": map[string]any{
					"image":                     "602401143452.dkr.ecr.us-east-1.amazonaws.com/eks/pause:3.5",
					"instance":                  "ip-192-168-62-22.ec2.internal",
					"k8s_io_cloud_provider_aws": "eb707f9bdba15de05a26c5a3b4a909ee",
					"name":                      "340166e10e91263f42abc91459ed3523ced66250f87df0f945a5816dea321452",
					"namespace":                 "kube-system",
					"pod":                       "kube-proxy-9bnjh",
				},
				"value": "990",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serialized, err := json.Marshal(tt.metric.JSON())
			assert.NoError(t, err)

			got := map[string]interface{}{}
			err = json.Unmarshal(serialized, &got)
			assert.NoError(t, err)

			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf("Metrics.JSON() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMetric_MarshalJSON_RoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		metric  types.Metric
		wantErr bool
	}{
		{
			name:    "basic",
			metric:  simpleMetric,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.metric
			got, err := m.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("Metric.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got2 := types.Metric{}
			err = got2.UnmarshalJSON(got)
			if err != nil {
				t.Errorf("Metric.UnmarshalJSON() error = %v", err)
				return
			}
			if diff := cmp.Diff(m, got2); diff != "" {
				t.Errorf("Metric.UnmarshalJSON() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMetric_ImportLabels(t *testing.T) {
	tests := []struct {
		name   string
		metric types.Metric
		labels map[string]string
		want   types.Metric
	}{
		{
			name:   "basic",
			metric: simpleMetric,
			labels: map[string]string{
				"__name__": "foo",
			},
			want: func() types.Metric {
				m := simpleMetric
				m.MetricName = "foo"
				m.Labels["__name__"] = "foo"
				return m
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.metric
			m.ImportLabels(tt.labels)

			if diff := cmp.Diff(tt.want, m); diff != "" {
				t.Errorf("Metric.ImportLabels() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMetric_ToParquet_RoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		metric  types.Metric
		wantErr bool
	}{
		{
			name:    "basic",
			metric:  simpleMetric,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.metric
			pm := m.Parquet()
			m2 := pm.Metric()
			require.Equal(t, m.MetricName, m2.MetricName)
		})
	}
}
