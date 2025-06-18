// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package helper_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
)

func TestEncodeRuntimeObject(t *testing.T) {
	// Create a sample Kubernetes object
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "test-container",
					Image: "nginx:latest",
				},
			},
		},
	}

	// Encode the object
	encodedBytes, err := helper.EncodeRuntimeObject(pod)
	require.NoError(t, err, "failed to encode runtime object")
	require.NotNil(t, encodedBytes, "encoded bytes should not be nil")

	// Decode the encoded bytes back to a runtime.Object
	scheme := runtime.NewScheme()
	err = helper.RegisterSchemes(scheme)
	require.NoError(t, err, "failed to register schemes")
	codecs := serializer.NewCodecFactory(scheme)
	decoder := codecs.UniversalDeserializer()

	obj, _, err := decoder.Decode(encodedBytes, nil, nil)
	require.NoError(t, err, "failed to decode encoded bytes")
	require.NotNil(t, obj, "decoded object should not be nil")

	// Assert the decoded object is of the same type and has the same metadata
	decodedPod, ok := obj.(*v1.Pod)
	require.True(t, ok, "decoded object is not of type *v1.Pod")
	assert.Equal(t, pod.Name, decodedPod.Name, "pod name mismatch")
	assert.Equal(t, pod.Namespace, decodedPod.Namespace, "pod namespace mismatch")
	assert.Equal(t, pod.Spec.Containers[0].Name, decodedPod.Spec.Containers[0].Name, "container name mismatch")
	assert.Equal(t, pod.Spec.Containers[0].Image, decodedPod.Spec.Containers[0].Image, "container image mismatch")
}

func TestEncodeToRawBytes(t *testing.T) {
	// Create a sample Kubernetes object
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "test-container",
					Image: "nginx:latest",
				},
			},
		},
	}

	// Encode the object using EncodeToRawBytes
	encodedBytes, err := helper.EncodeToRawBytes(pod)
	require.NoError(t, err, "failed to encode object to raw bytes")
	require.NotNil(t, encodedBytes, "encoded bytes should not be nil")

	// Decode the encoded bytes back to a runtime.Object
	scheme := runtime.NewScheme()
	err = helper.RegisterSchemes(scheme)
	require.NoError(t, err, "failed to register schemes")
	codecs := serializer.NewCodecFactory(scheme)
	decoder := codecs.UniversalDeserializer()

	obj, _, err := decoder.Decode(encodedBytes, nil, nil)
	require.NoError(t, err, "failed to decode encoded bytes")
	require.NotNil(t, obj, "decoded object should not be nil")

	// Assert the decoded object is of the same type and has the same metadata
	decodedPod, ok := obj.(*v1.Pod)
	require.True(t, ok, "decoded object is not of type *v1.Pod")
	assert.Equal(t, pod.Name, decodedPod.Name, "pod name mismatch")
	assert.Equal(t, pod.Namespace, decodedPod.Namespace, "pod namespace mismatch")
	assert.Equal(t, pod.Spec.Containers[0].Name, decodedPod.Spec.Containers[0].Name, "container name mismatch")
	assert.Equal(t, pod.Spec.Containers[0].Image, decodedPod.Spec.Containers[0].Image, "container image mismatch")
}

func TestConvertObject(t *testing.T) {
	// Test case: Successful conversion
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
	}
	convertedObj := helper.ConvertObject[*v1.Pod](pod)
	require.NotNil(t, convertedObj, "converted object should not be nil")
	assert.Equal(t, pod.Name, convertedObj.GetName(), "pod name mismatch")
	assert.Equal(t, pod.Namespace, convertedObj.GetNamespace(), "pod namespace mismatch")

	// Test case: Failed conversion
	service := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-service",
			Namespace: "default",
		},
	}
	convertedObj = helper.ConvertObject[*v1.Pod](service)
	assert.Nil(t, convertedObj, "converted object should be nil for incompatible types")
}
