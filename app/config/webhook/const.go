// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

type ResourceType int

const (
	Unknown ResourceType = iota
	Deployment
	StatefulSet
	Pod
	Node
	Namespace
	Job
	CronJob
	DaemonSet
	Ingress
	Service
	CustomResourceDefinition
	ReplicaSet
	PersistentVolume
	PersistentVolumeClaim
	Gateway
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
