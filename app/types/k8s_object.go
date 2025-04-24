// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

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
