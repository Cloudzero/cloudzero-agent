// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"fmt"
	"time"

	"github.com/cloudzero/cloudzero-agent-validator/app/types"
	"github.com/google/uuid"
)

func GenerateNodeRecords(
	input *MetricRecordInput,
	nodeName, region string,
	maxCPU, maxMemory int64,
	totalPods int,
) []types.Metric {
	metrics := make([]types.Metric, 0)

	currTime := input.GetStartTime()
	for currTime.Before(input.GetEndTime()) {
		// add an info metric every 2 minutes
		metrics = append(metrics, types.Metric{
			ID:             uuid.New(),
			ClusterName:    input.ClusterName,
			CloudAccountID: input.CloudAccountID,
			MetricName:     "kube_node_info",
			NodeName:       nodeName,
			CreatedAt:      currTime.UTC(),
			TimeStamp:      currTime.UTC(),
			Labels: map[string]string{
				"__name__":    "kube_node_info",
				"node":        nodeName,
				"provider_id": fmt.Sprintf("aws:///%sa/i-instance-1", region),
				"system_uuid": "k8s-node-id-0001",
			},
			Value: "1",
		})

		// add status capacity records every 2 minutes
		metrics = append(metrics, types.Metric{
			ID:             uuid.New(),
			ClusterName:    input.ClusterName,
			CloudAccountID: input.CloudAccountID,
			MetricName:     "kube_node_status_capacity",
			NodeName:       nodeName,
			CreatedAt:      currTime.UTC(),
			TimeStamp:      currTime.UTC(),
			Labels: map[string]string{
				"__name__": "kube_node_status_capacity",
				"node":     nodeName,
				"resource": "cpu",
				"unit":     "core",
			},
			Value: fmt.Sprintf("%d", maxCPU),
		})
		metrics = append(metrics, types.Metric{
			ID:             uuid.New(),
			ClusterName:    input.ClusterName,
			CloudAccountID: input.CloudAccountID,
			MetricName:     "kube_node_status_capacity",
			NodeName:       nodeName,
			CreatedAt:      currTime.UTC(),
			TimeStamp:      currTime.UTC(),
			Labels: map[string]string{
				"__name__": "kube_node_status_capacity",
				"node":     nodeName,
				"resource": "memory",
				"unit":     "byte",
			},
			Value: fmt.Sprintf("%d", maxMemory),
		})

		currTime = currTime.Add(time.Minute * 2)
	}

	// generate the pod data
	for i := range totalPods {
		metrics = append(metrics, GeneratePodRecords(input, nodeName, fmt.Sprintf("pod-%d", i), fmt.Sprintf("namespace-%d", i), "", maxCPU/int64(totalPods), maxMemory/int64(totalPods), i+1)...)
	}

	return metrics
}
