// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metricio

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSVReader_ReadCSVFile(t *testing.T) {
	tests := []struct {
		name      string
		csv       string
		batchSize int
		expected  int
		wantErr   bool
		errMsg    string
	}{
		{
			name: "basic csv with uppercase headers",
			csv: `USAGE_DATE,VALUE,LABELS
2024-01-01 12:00:00.000 Z,42.5,"{""__name__"":""test_metric"",""node"":""test-node""}"
`,
			batchSize: 100,
			expected:  1,
		},
		{
			name: "lowercase headers",
			csv: `usage_date,value,labels
2024-01-01 12:00:00.000 Z,100,"{""__name__"":""lowercase_metric""}"
`,
			batchSize: 100,
			expected:  1,
		},
		{
			name: "mixed case headers",
			csv: `Usage_Date,Value,Labels
2024-01-01 12:00:00.000 Z,50,"{""__name__"":""mixed_case_metric""}"
`,
			batchSize: 100,
			expected:  1,
		},
		{
			name: "multiple rows",
			csv: `USAGE_DATE,VALUE,LABELS
2024-01-01 12:00:00.000 Z,1,"{""__name__"":""m1""}"
2024-01-01 12:01:00.000 Z,2,"{""__name__"":""m2""}"
2024-01-01 12:02:00.000 Z,3,"{""__name__"":""m3""}"
`,
			batchSize: 100,
			expected:  3,
		},
		{
			name: "batching behavior",
			csv: `USAGE_DATE,VALUE,LABELS
2024-01-01 12:00:00.000 Z,1,"{""__name__"":""m1""}"
2024-01-01 12:01:00.000 Z,2,"{""__name__"":""m2""}"
2024-01-01 12:02:00.000 Z,3,"{""__name__"":""m3""}"
`,
			batchSize: 2,
			expected:  3,
		},
		{
			name: "date format without Z",
			csv: `USAGE_DATE,VALUE,LABELS
2024-01-01 12:00:00.000,42,"{""__name__"":""no_z_metric""}"
`,
			batchSize: 100,
			expected:  1,
		},
		{
			name: "missing required columns",
			csv: `DATE,VAL,INFO
2024-01-01,42,test
`,
			batchSize: 100,
			wantErr:   true,
			errMsg:    "must have USAGE_DATE, VALUE, and LABELS columns",
		},
		{
			name: "extra columns ignored",
			csv: `EXTRA1,USAGE_DATE,EXTRA2,VALUE,LABELS,EXTRA3
x,2024-01-01 12:00:00.000 Z,y,42,"{""__name__"":""extra_metric""}",z
`,
			batchSize: 100,
			expected:  1,
		},
		{
			name: "skip invalid rows",
			csv: `USAGE_DATE,VALUE,LABELS
2024-01-01 12:00:00.000 Z,42,"{""__name__"":""valid""}"
invalid_date,100,"{""__name__"":""invalid""}"
2024-01-01 12:02:00.000 Z,99,"{""__name__"":""valid2""}"
`,
			batchSize: 100,
			expected:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			csvPath := filepath.Join(tmpDir, "test.csv")
			err := os.WriteFile(csvPath, []byte(tt.csv), 0o644)
			require.NoError(t, err)

			reader := NewCSVReader(tt.batchSize)

			var timeSeries []prompb.TimeSeries
			err = reader.ReadCSVFile(csvPath, func(batch []prompb.TimeSeries) error {
				timeSeries = append(timeSeries, batch...)
				return nil
			})

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.Len(t, timeSeries, tt.expected)
		})
	}
}

func TestCSVReader_CaseInsensitiveColumns(t *testing.T) {
	tmpDir := t.TempDir()

	// Test various case combinations
	cases := []struct {
		name   string
		header string
	}{
		{"all uppercase", "USAGE_DATE,VALUE,LABELS"},
		{"all lowercase", "usage_date,value,labels"},
		{"mixed case 1", "Usage_Date,Value,Labels"},
		{"mixed case 2", "USAGE_date,value,LABELS"},
		{"mixed case 3", "usage_DATE,VALUE,labels"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			csv := tc.header + "\n" + `2024-01-01 12:00:00.000 Z,42,"{""__name__"":""test""}"` + "\n"
			csvPath := filepath.Join(tmpDir, tc.name+".csv")
			err := os.WriteFile(csvPath, []byte(csv), 0o644)
			require.NoError(t, err)

			reader := NewCSVReader(100)
			var count int
			err = reader.ReadCSVFile(csvPath, func(batch []prompb.TimeSeries) error {
				count += len(batch)
				return nil
			})

			require.NoError(t, err, "header: %s", tc.header)
			assert.Equal(t, 1, count, "header: %s", tc.header)
		})
	}
}

func TestCSVReader_FileNotFound(t *testing.T) {
	reader := NewCSVReader(100)

	err := reader.ReadCSVFile("/nonexistent/file.csv", func(batch []prompb.TimeSeries) error {
		return nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open")
}

func TestCSVReader_CallbackError(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")
	csv := `USAGE_DATE,VALUE,LABELS
2024-01-01 12:00:00.000 Z,42,"{""__name__"":""test""}"
`
	err := os.WriteFile(csvPath, []byte(csv), 0o644)
	require.NoError(t, err)

	reader := NewCSVReader(100)

	err = reader.ReadCSVFile(csvPath, func(batch []prompb.TimeSeries) error {
		return assert.AnError
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "callback error")
}

func TestCSVReader_LabelsParsing(t *testing.T) {
	tmpDir := t.TempDir()
	csvPath := filepath.Join(tmpDir, "test.csv")
	csv := `USAGE_DATE,VALUE,LABELS
2024-01-01 12:00:00.000 Z,42,"{""__name__"":""cpu_usage"",""node"":""worker-1"",""pod"":""myapp-123"",""namespace"":""default""}"
`
	err := os.WriteFile(csvPath, []byte(csv), 0o644)
	require.NoError(t, err)

	reader := NewCSVReader(100)

	var timeSeries []prompb.TimeSeries
	err = reader.ReadCSVFile(csvPath, func(batch []prompb.TimeSeries) error {
		timeSeries = append(timeSeries, batch...)
		return nil
	})

	require.NoError(t, err)
	require.Len(t, timeSeries, 1)

	ts := timeSeries[0]
	labelMap := make(map[string]string)
	for _, l := range ts.Labels {
		labelMap[l.Name] = l.Value
	}

	assert.Equal(t, "cpu_usage", labelMap["__name__"])
	assert.Equal(t, "worker-1", labelMap["node"])
	assert.Equal(t, "myapp-123", labelMap["pod"])
	assert.Equal(t, "default", labelMap["namespace"])

	// Check labels are sorted
	for i := 1; i < len(ts.Labels); i++ {
		assert.True(t, ts.Labels[i-1].Name <= ts.Labels[i].Name,
			"labels should be sorted")
	}
}

func TestNewCSVReader_DefaultBatchSize(t *testing.T) {
	reader := NewCSVReader(0)
	assert.NotNil(t, reader)
}

func TestNewCSVReader_NegativeBatchSize(t *testing.T) {
	reader := NewCSVReader(-5)
	assert.NotNil(t, reader)
}
