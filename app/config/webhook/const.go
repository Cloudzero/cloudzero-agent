// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
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
	IngressClass
	Ingress
	Service
	CustomResourceDefinition
	ReplicaSet
	StorageClass
	PersistentVolume
	PersistentVolumeClaim
	GatewayClass
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
	IngressClass:             "ingressclass",
	Ingress:                  "ingress",
	Service:                  "service",
	CustomResourceDefinition: "crd",
	ReplicaSet:               "replicaset",
	StorageClass:             "storageclass",
	PersistentVolume:         "pv",
	PersistentVolumeClaim:    "pcv",
	GatewayClass:             "gatewayclass",
	Gateway:                  "gateway",
}

const (
	FieldNamespace    = "namespace"
	FieldNode         = "node"
	FieldPod          = "pod"
	FieldResourceType = "resource_type"
	FieldWorkload     = "workload"
)
