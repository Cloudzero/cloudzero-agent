// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package types defines Kubernetes API resource constants and interfaces for the CloudZero Agent webhook system.
package types

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Kubernetes API group constants used for admission webhook resource filtering.
// These groups represent the primary API categories that the CloudZero Agent monitors
// for label and annotation collection to support cost allocation.
const (
	// GroupApps represents the apps/v1 API group containing workload resources like Deployments and StatefulSets.
	GroupApps = "apps"

	// GroupBatch represents the batch/v1 API group containing job-related resources like Jobs and CronJobs.
	GroupBatch = "batch"

	// GroupCore represents the core/v1 API group (empty string) containing fundamental resources like Pods and Nodes.
	GroupCore = ""

	// GroupExt represents the apiextensions.k8s.io API group containing CustomResourceDefinitions.
	GroupExt = "apiextensions.k8s.io"

	// GroupNet represents the networking.k8s.io API group containing Ingress and IngressClass resources.
	GroupNet = "networking.k8s.io"

	// GroupGateway represents the gateway.networking.k8s.io API group containing Gateway API resources.
	GroupGateway = "gateway.networking.k8s.io"

	// GroupStorage represents the storage.k8s.io API group containing storage-related resources.
	GroupStorage = "storage.k8s.io"

	// V1 represents the stable v1 API version used by most core Kubernetes resources.
	V1 = "v1"

	// V1Beta2 represents the v1beta2 API version for resources in beta stability.
	V1Beta2 = "v1beta2"

	// V1Beta1 represents the v1beta1 API version for resources in beta stability.
	V1Beta1 = "v1beta1"

	// Important: when adding a new resource type, please remember to also
	// update the list in the webhook-validating-config.yaml template in the
	// Helm chart.

	// Resource kind constants for Kubernetes objects monitored by the CloudZero Agent admission webhook.
	// These lowercase string constants match the Kubernetes API resource names and are used for
	// webhook admission rule configuration and resource type identification.

	// KindDeployment represents Deployment workload resources that manage ReplicaSets and Pods.
	KindDeployment = "deployment"

	// KindStatefulSet represents StatefulSet workload resources that manage stateful applications.
	KindStatefulSet = "statefulset"

	// KindDaemonSet represents DaemonSet workload resources that ensure Pods run on all/selected nodes.
	KindDaemonSet = "daemonset"

	// KindReplicaSet represents ReplicaSet resources that maintain a stable set of replica Pods.
	KindReplicaSet = "replicaset"

	// KindPod represents Pod resources, the smallest deployable units containing one or more containers.
	KindPod = "pod"

	// KindNamespace represents Namespace resources that provide resource isolation and naming scope.
	KindNamespace = "namespace"

	// KindNode represents Node resources, the worker machines (physical or virtual) in the cluster.
	KindNode = "node"

	// KindService represents Service resources that expose Pods as network services.
	KindService = "service"

	// KindStorageClass represents StorageClass resources that define different types of storage.
	KindStorageClass = "storageclass"

	// KindPersistentVolume represents PersistentVolume resources that provide cluster-level storage.
	KindPersistentVolume = "persistentvolume"

	// KindPersistentVolumeClaim represents PersistentVolumeClaim resources that request storage.
	KindPersistentVolumeClaim = "persistentvolumeclaim"

	// KindJob represents Job resources that create one or more Pods to run a task to completion.
	KindJob = "job"

	// KindCronJob represents CronJob resources that create Jobs on a schedule.
	KindCronJob = "cronjob"

	// KindCRD represents CustomResourceDefinition resources that extend the Kubernetes API.
	KindCRD = "customresourcedefinition"

	// KindIngress represents Ingress resources that manage external access to cluster services.
	KindIngress = "ingress"

	// KindIngressClass represents IngressClass resources that define different ingress implementations.
	KindIngressClass = "ingressclass"

	// KindGateway represents Gateway resources from the Gateway API for advanced traffic management.
	KindGateway = "gateway"

	// KindGatewayClass represents GatewayClass resources that define different gateway implementations.
	KindGatewayClass = "gatewayclass"
)

// Groups contains the complete list of Kubernetes API groups monitored by the CloudZero Agent.
// Used for webhook admission controller configuration and resource filtering logic.
var Groups = []string{
	GroupApps,
	GroupBatch,
	GroupCore,
	GroupExt,
	GroupNet,
	GroupGateway,
}

// Versions contains the supported Kubernetes API versions for webhook operations.
// These versions define the API compatibility matrix for admission webhook rules.
var Versions = []string{
	V1,
	V1Beta2,
	V1Beta1,
}

// Kinds contains all Kubernetes resource kinds that the CloudZero Agent webhook monitors.
// This list must be kept synchronized with the webhook-validating-config.yaml template
// to ensure proper admission controller functionality for cost allocation data collection.
var Kinds = []string{
	KindDeployment,
	KindStatefulSet,
	KindDaemonSet,
	KindReplicaSet,
	KindPod,
	KindNamespace,
	KindNode,
	KindService,
	KindPersistentVolume,
	KindPersistentVolumeClaim,
	KindJob,
	KindCronJob,
	KindCRD,
	KindIngress,
	KindGateway,
}

// K8sObject defines a Kubernetes object that implements both metav1.Object and runtime.Object interfaces.
// For unknown or unsupported Kubernetes objects, use `unstructured.Unstructured` instead
// (e.g., objects like `corev1.PodExecOptions` that do not satisfy both interfaces).
type K8sObject interface {
	metav1.Object
	runtime.Object
}

// ObjectCreator defines an interface for creating Kubernetes objects from raw encoded data (json, or yaml).
// It provides a method to decode and transform raw runtime-encoded bytes into a K8sObject
// while capturing relevant metadata.
type ObjectCreator interface {
	// NewObject decodes the provided raw runtime-encoded byte slice into a K8sObject.
	// It extracts and captures the necessary metadata from the object.
	// Returns the constructed K8sObject or an error if decoding fails.
	NewObject(raw []byte) (K8sObject, error)
}
