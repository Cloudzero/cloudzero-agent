// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/app/domain"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"
)

func TestUnit_Metrics_GenerateCluster(t *testing.T) {
	startTime := time.Now()
	numHours := 4
	numNodes := 3
	podsPerNode := 3
	cpus := 128
	var mem int64 = (1 << 30) * 196 // 196gb

	// calculate the expected number of records
	expectedRecords := 0
	expectedRecords += numNodes * (numHours * 30)               // node status every 2 minutes
	expectedRecords += numNodes * (numHours * 30 * 2)           // node capacity every 2 minutes (cpu+mem)
	expectedRecords += numNodes * podsPerNode * (numHours * 30) // records every 2 minutes for the pod lifecycle
	for i := range podsPerNode {
		expectedRecords += numNodes * (i + 1) * 4 * (numHours * 30) // container resource requests
	}
	expectedRecords += numNodes * podsPerNode * numHours * 60 // cpu usage records
	expectedRecords += numNodes * podsPerNode * numHours * 60 // mem usage records

	// generate the metrics
	metrics := GenerateClusterMetrics("org-id", "id-123", "test-cluster", startTime.Add(-time.Hour*time.Duration(numHours)), startTime, int64(cpus), mem, numNodes, podsPerNode)

	// encode into json
	enc, err := json.Marshal(metrics)
	require.NoError(t, err, "failed to encode into json")

	fmt.Println("--- Generation Stats:")
	fmt.Println("* Expected Records:            ", expectedRecords)
	fmt.Println("* Generated Records:           ", len(metrics))
	fmt.Println("* Size of Records List (JSON) :", float64(len(enc))/1_000_000, "MB")

	require.Equal(t, expectedRecords, len(metrics))
}

func TestUnit_Metrics_ToPromb(t *testing.T) {
	startTime := time.Now()
	numHours := 4
	numNodes := 3
	podsPerNode := 3
	cpus := 128
	var mem int64 = (1 << 30) * 196 // 196gb

	// generate the metrics
	metrics := GenerateClusterMetrics("org-id", "id-123", "test-cluster", startTime.Add(-time.Hour*time.Duration(numHours)), startTime, int64(cpus), mem, numNodes, podsPerNode)

	// encode to a promb write request
	wr, err := EncodeV1(metrics)
	require.NoError(t, err, "failed to encode the metrics into a promb write request")

	// encode with proto
	data, err := proto.Marshal(protoadapt.MessageV2Of(wr))
	require.NoError(t, err, "failed to encode the prometheus metrics to proto")

	// create a metrics collector
	cfg := config.Settings{
		CloudAccountID: "123456789012",
		Region:         "us-west-2",
		ClusterName:    "testcluster",
		Cloudzero: config.Cloudzero{
			Host:           "api.cloudzero.com",
			RotateInterval: 10 * time.Second,
		},
	}
	ctrl := gomock.NewController(t)
	storage := mocks.NewMockStore(ctrl)
	mockClock := mocks.NewMockClock(time.Now())
	require.NoError(t, err, "failed to create the store")
	collector, err := domain.NewMetricCollector(&cfg, mockClock, storage, nil)
	require.NoError(t, err, "failed to create the metric collector")

	// decode the metrics
	decodedMetrics, err := collector.DecodeV1(data)
	require.NoError(t, err, "failed to decode the metrics")

	fmt.Println("--- Generation Stats:")
	fmt.Println("* Generated Records: ", len(metrics))
	fmt.Println("* Decoded Records:   ", len(decodedMetrics))

	// ensure they are the same length
	require.Equal(t, len(metrics), len(decodedMetrics))
}
