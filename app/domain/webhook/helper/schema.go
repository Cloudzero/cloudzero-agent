// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package helper

import (
	"fmt"

	"github.com/rs/zerolog/log"

	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// ConvertObject attempts to cast an object to the specified metav1.Object type.
// If the cast fails, it logs an error and returns nil.
func ConvertObject[T metav1.Object](o any) metav1.Object {
	if obj, ok := o.(T); ok {
		return obj
	}
	log.Error().Msgf("failed to cast object to %T", new(T))
	return nil
}

// RegisterSchemes registers the API schemes for various Kubernetes resource types.
// It ensures that the runtime.Scheme is aware of the resource types used in the application.
func RegisterSchemes(scheme *runtime.Scheme) error {
	// List of functions to add resource types to the scheme
	schemeFuncs := []func(*runtime.Scheme) error{
		admissionv1.AddToScheme,
		appsv1.AddToScheme,
		appsv1beta1.AddToScheme,
		appsv1beta2.AddToScheme,
		batchv1.AddToScheme,
		batchv1beta1.AddToScheme,
		corev1.AddToScheme,
		networkingv1.AddToScheme,
		networkingv1beta1.AddToScheme,
	}

	// Register each scheme function
	for _, addToScheme := range schemeFuncs {
		if err := addToScheme(scheme); err != nil {
			return fmt.Errorf("failed to add scheme: %w", err)
		}
	}
	return nil
}

// EncodeToRawBytes encodes a Kubernetes resource object into raw bytes.
// It accepts any object that implements metav1.Object and runtime.Object.
func EncodeToRawBytes(obj metav1.Object) ([]byte, error) {
	// Ensure the object also implements runtime.Object
	runtimeObj, ok := obj.(runtime.Object)
	if !ok {
		return nil, fmt.Errorf("object does not implement runtime.Object")
	}

	// Use the helper function to encode the runtime.Object
	return EncodeRuntimeObject(runtimeObj)
}

// EncodeRuntimeObject encodes a given runtime.Object into raw bytes using the registered schemes.
func EncodeRuntimeObject(runtimeObj runtime.Object) ([]byte, error) {
	// Create a new runtime.Scheme and register the necessary schemes
	scheme := runtime.NewScheme()
	if err := RegisterSchemes(scheme); err != nil {
		return nil, fmt.Errorf("failed to register schemes: %w", err)
	}

	// Create a CodecFactory and an encoder for the registered schemes
	codecs := serializer.NewCodecFactory(scheme)
	encoder := codecs.LegacyCodec(
		admissionv1.SchemeGroupVersion,
		appsv1.SchemeGroupVersion,
		appsv1beta1.SchemeGroupVersion,
		appsv1beta2.SchemeGroupVersion,
		batchv1.SchemeGroupVersion,
		batchv1beta1.SchemeGroupVersion,
		corev1.SchemeGroupVersion,
		networkingv1.SchemeGroupVersion,
		networkingv1beta1.SchemeGroupVersion,
	)

	// Encode the runtime object into raw bytes
	raw, err := runtime.Encode(encoder, runtimeObj)
	if err != nil {
		return nil, fmt.Errorf("failed to encode object: %w", err)
	}

	return raw, nil
}
