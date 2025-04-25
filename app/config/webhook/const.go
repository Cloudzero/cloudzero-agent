// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

type ResourceType int

const (
	Unknown                  ResourceType = 0
	Deployment               ResourceType = 1
	StatefulSet              ResourceType = 2
	Pod                      ResourceType = 3
	Node                     ResourceType = 4
	Namespace                ResourceType = 5
	Job                      ResourceType = 6
	CronJob                  ResourceType = 7
	DaemonSet                ResourceType = 8
	Ingress                  ResourceType = 9
	Service                  ResourceType = 10
	CustomResourceDefinition ResourceType = 11
	ReplicaSet               ResourceType = 12
	PersistentVolume         ResourceType = 13
	PersistentVolumeClaim    ResourceType = 14
	Gateway                  ResourceType = 15
)

var ResourceTypeToMetricName = map[ResourceType]string{
	Unknown:                  "unknown",
	Deployment:               "deployment",
	StatefulSet:              "statefulset",
	Pod:                      "pod",
	Node:                     "node",
	Namespace:                "namespace",
	Job:                      "job",
	CronJob:                  "cronjob",
	DaemonSet:                "daemonset",
	Ingress:                  "ingress",
	Service:                  "service",
	CustomResourceDefinition: "crd",
	ReplicaSet:               "replicaset",
	PersistentVolume:         "pv",
	PersistentVolumeClaim:    "pcv",
	Gateway:                  "gateway",
}

const (
	FieldNamespace    = "namespace"
	FieldNode         = "node"
	FieldPod          = "pod"
	FieldResourceType = "resource_type"
	FieldWorkload     = "workload"
)
