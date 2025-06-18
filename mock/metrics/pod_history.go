// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metrics

import (
	"fmt"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/google/uuid"
)

func GeneratePodRecords(
	input *MetricRecordInput,
	nodeName, podName, namespace, workload string,
	maxCPU, maxMemory int64,
	numContainers int,
) []types.Metric {
	labels := createPodInfoLabels(workload, nodeName, namespace, podName)
	metrics := make([]types.Metric, 0)

	// add lifecycle records every 2 minutes
	currTime := input.GetStartTime()
	for currTime.Before(input.GetEndTime()) {
		metrics = append(metrics, types.Metric{
			ID:             uuid.New(),
			ClusterName:    input.ClusterName,
			CloudAccountID: input.CloudAccountID,
			MetricName:     "kube_pod_info",
			NodeName:       nodeName,
			CreatedAt:      currTime.UTC(),
			TimeStamp:      currTime.UTC(),
			Labels:         labels,
			Value:          "1",
		})

		// generate some container requests
		for i := range numContainers {
			metrics = append(metrics, GenerateContainerRecords(input, currTime, fmt.Sprintf("container-%d", i), nodeName, podName, namespace, workload, maxCPU/int64(numContainers)/2, maxCPU/int64(numContainers), maxMemory/int64(numContainers)/2, maxMemory/int64(numContainers))...)
		}

		currTime = currTime.Add(time.Minute * 2)
	}

	// generate cpu and memory requests for this container
	metrics = append(metrics, GenerateCPUUsageRecords(input, int(maxCPU), 0.6, 60, nodeName, namespace, podName)...)
	metrics = append(metrics, GenerateMemoryUsageRecords(input, maxMemory, 0.85, 60, nodeName, namespace, podName)...)

	return metrics
}

func GenerateContainerRecords(
	input *MetricRecordInput,
	currentTime time.Time,
	containerName string,
	nodeName, podName, namespace, workload string,
	cpuReq, cpuLimit, memReq, memLimit int64,
) []types.Metric {
	metrics := make([]types.Metric, 0)

	// generate resource requests
	metrics = append(metrics, types.Metric{
		ID:             uuid.New(),
		ClusterName:    input.ClusterName,
		CloudAccountID: input.CloudAccountID,
		MetricName:     "kube_pod_container_resource_requests",
		NodeName:       nodeName,
		CreatedAt:      currentTime.UTC(),
		TimeStamp:      currentTime.UTC(),
		Labels: map[string]string{
			"__name__":  "kube_pod_container_resource_requests",
			"node":      nodeName,
			"namespace": namespace,
			"pod":       podName,
			"uid":       podName,
			"container": containerName,
			"resource":  "cpu",
			"unit":      "core",
		},
		Value: fmt.Sprintf("%d", cpuReq),
	})
	metrics = append(metrics, types.Metric{
		ID:             uuid.New(),
		ClusterName:    input.ClusterName,
		CloudAccountID: input.CloudAccountID,
		MetricName:     "kube_pod_container_resource_requests",
		NodeName:       nodeName,
		CreatedAt:      currentTime.UTC(),
		TimeStamp:      currentTime.UTC(),
		Labels: map[string]string{
			"__name__":  "kube_pod_container_resource_requests",
			"node":      nodeName,
			"namespace": namespace,
			"pod":       podName,
			"uid":       podName,
			"container": containerName,
			"resource":  "memory",
			"unit":      "unit",
		},
		Value: fmt.Sprintf("%d", memReq),
	})

	// generate resource limits
	metrics = append(metrics, types.Metric{
		ID:             uuid.New(),
		ClusterName:    input.ClusterName,
		CloudAccountID: input.CloudAccountID,
		MetricName:     "kube_pod_container_resource_limits",
		NodeName:       nodeName,
		CreatedAt:      currentTime.UTC(),
		TimeStamp:      currentTime.UTC(),
		Labels: map[string]string{
			"__name__":  "kube_pod_container_resource_limits",
			"node":      nodeName,
			"namespace": namespace,
			"pod":       podName,
			"uid":       podName,
			"container": containerName,
			"resource":  "cpu",
			"unit":      "core",
		},
		Value: fmt.Sprintf("%d", cpuLimit),
	})
	metrics = append(metrics, types.Metric{
		ID:             uuid.New(),
		ClusterName:    input.ClusterName,
		CloudAccountID: input.CloudAccountID,
		MetricName:     "kube_pod_container_resource_limits",
		NodeName:       nodeName,
		CreatedAt:      currentTime.UTC(),
		TimeStamp:      currentTime.UTC(),
		Labels: map[string]string{
			"__name__":  "kube_pod_container_resource_limits",
			"node":      nodeName,
			"namespace": namespace,
			"pod":       podName,
			"uid":       podName,
			"container": containerName,
			"resource":  "memory",
			"unit":      "unit",
		},
		Value: fmt.Sprintf("%d", memLimit),
	})

	return metrics
}

func createPodInfoLabels(workload string, node string, namespace string, pod string) map[string]string {
	m := map[string]string{
		"__name__":        "kube_pod_info",
		"node":            node,
		"namespace":       namespace,
		"pod":             pod,
		"uid":             pod,
		"created_by_kind": workload,
	}

	switch workload {
	case "StatefulSet":
		m["created_by_name"] = "jl"
	case "Job":
		m["created_by_name"] = "k8s-cronjob-nonsense-18840387"
	case "ReplicaSet":
		m["created_by_name"] = "customer-app-18840387"
	case "SparkApplication":
		m["created_by_name"] = "usage-compute"
	case "Deployment":
		m["created_by_name"] = "usage-stamper"
	case "Pod":
		m["created_by_name"] = "usage-pod"
	}

	return m
}
