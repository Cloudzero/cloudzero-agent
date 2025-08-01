// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package helper contains decode helper methods for transforming kubernetes metav1.Objects into K8sObjects
package helper

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/cloudzero/cloudzero-agent/app/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	clientsetscheme "k8s.io/client-go/kubernetes/scheme"
)

// newK8sObj returns a new object of a Kubernetes type based on the type.
func newK8sObj(t reflect.Type) metav1.Object {
	// Create a new object of the webhook resource type
	// convert to ptr and typeassert to Kubernetes Object.
	var obj interface{}
	newObj := reflect.New(t)
	obj = newObj.Interface()
	o, _ := obj.(metav1.Object)
	return o
}

// getK8sObjType returns the type (not the pointer type) of a kubernetes object.
func getK8sObjType(obj metav1.Object) reflect.Type {
	// Object is an interface, is safe to assume that is a pointer.
	// Get the indirect type of the object.
	return reflect.Indirect(reflect.ValueOf(obj)).Type()
}

type staticObjectCreator struct {
	objType      reflect.Type
	deserializer runtime.Decoder
}

// NewStaticObjectCreator doesn't need to infer the type, it will create a new schema and create a new
// object with the same type from the received object type.
func NewStaticObjectCreator(obj metav1.Object) types.ObjectCreator {
	codecs := serializer.NewCodecFactory(runtime.NewScheme())
	return staticObjectCreator{
		objType:      getK8sObjType(obj),
		deserializer: codecs.UniversalDeserializer(),
	}
}

func (s staticObjectCreator) NewObject(raw []byte) (types.K8sObject, error) {
	obj, ok := newK8sObj(s.objType).(types.K8sObject)
	if !ok {
		return nil, errors.New("could not type assert metav1.Object and runtime.Object")
	}

	_, _, err := s.deserializer.Decode(raw, nil, obj)
	if err != nil {
		return nil, fmt.Errorf("error deseralizing request raw object: %s", err)
	}

	return obj, nil
}

type dynamicObjectCreator struct {
	universalDecoder    runtime.Decoder
	unstructuredDecoder runtime.Decoder
}

// NewDynamicObjectCreator returns a object creator that knows how to return objects from raw
// JSON data without the need of knowing the type.
//
// To be able to infer the types the types need to be registered on the global client Scheme.
// Normally when a user tries casting the metav1.Object to a specific type, the object is already
// registered. In case the type is not registered and the object can't be created it will fallback
// to an `Unstructured` type.
//
// Some types like pod/exec (`corev1.PodExecOptions`) implement `runtime.Object` however they don't
// implement `metav1.Object`. In that case we also fallback to `Unstructured`.
//
// Useful to make dynamic webhooks that expect multiple or unknown types.
func NewDynamicObjectCreator() types.ObjectCreator {
	return dynamicObjectCreator{
		universalDecoder:    clientsetscheme.Codecs.UniversalDeserializer(),
		unstructuredDecoder: unstructured.UnstructuredJSONScheme,
	}
}

func (d dynamicObjectCreator) NewObject(raw []byte) (types.K8sObject, error) {
	runtimeObj, _, err := d.universalDecoder.Decode(raw, nil, nil)
	if err == nil {
		// Some types like pod/exec (`corev1.PodExecOptions`) implement `runtime.Object` however
		// they don't implement `metav1.Object`. i.e. A Validator APIs should give the
		// user a runtime.Object instead of a metav1.Object; if this kind of object in encountered, fallback to Unstructured.
		obj, ok := runtimeObj.(types.K8sObject)
		if ok {
			return obj, nil
		}
	}

	// Fallback to unstructured.
	runtimeObj, _, err = d.unstructuredDecoder.Decode(raw, nil, nil)
	obj, ok := runtimeObj.(types.K8sObject)
	if !ok {
		return nil, errors.New("could not type assert metav1.Object and runtime.Object")
	}

	return obj, err
}
