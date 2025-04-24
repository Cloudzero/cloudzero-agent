// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // There is currently substantial duplication in the handlers :(
package handler

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/rs/zerolog/log"
)

// ConfigAccessor defines the interface for accessing settings related to labels and annotations
type ConfigAccessor interface {
	LabelsEnabled() bool
	AnnotationsEnabled() bool
	LabelsEnabledForType() bool
	AnnotationsEnabledForType() bool
	ResourceType() string
	Settings() *config.Settings
}

type DataFormatter func(accessor ConfigAccessor, obj metav1.Object) types.ResourceTags

type GenericHandler[T metav1.Object] struct {
	hook.Handler
	settings   *config.Settings
	clock      types.TimeProvider
	accessor   ConfigAccessor
	formatData DataFormatter
}

func NewGenericHandler[T metav1.Object](
	store types.ResourceStore,
	settings *config.Settings,
	clock types.TimeProvider,
	objectCreator metav1.Object,
	accessor ConfigAccessor,
	formatData DataFormatter,
) *hook.Handler {
	h := &GenericHandler[T]{
		settings:   settings,
		clock:      clock,
		formatData: formatData,
	}
	h.ObjectCreator = helper.NewStaticObjectCreator(objectCreator)
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Delete = h.Delete()
	h.Handler.Store = store
	return &h.Handler
}

func (h *GenericHandler[T]) Create() hook.AdmitFunc {
	return h.admitFunc("create")
}

func (h *GenericHandler[T]) Update() hook.AdmitFunc {
	return h.admitFunc("update")
}

func (h *GenericHandler[T]) Delete() hook.AdmitFunc {
	return h.admitFunc("delete")
}

func (h *GenericHandler[T]) admitFunc(action string) hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(T)
		if !ok {
			log.Warn().Msgf("unable to cast to object instance of type %T", o)
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, action)

		if h.accessor.LabelsEnabledForType() || h.accessor.AnnotationsEnabledForType() {
			genericWriteDataToStorage(ctx, h.Store, h.clock, h.formatData(h.accessor, o))
		}

		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func WorkloadDataFormatter(accessor ConfigAccessor, obj metav1.Object) types.ResourceTags {
	var (
		labels      = config.MetricLabelTags{}
		annotations = config.MetricLabelTags{}
		namespace   = obj.GetNamespace()
		objectName  = obj.GetName()
	)
	if accessor.LabelsEnabled() {
		labels = config.Filter(obj.GetLabels(), accessor.Settings().LabelMatches, accessor.LabelsEnabledForType(), accessor.Settings())
	}
	if accessor.AnnotationsEnabled() {
		annotations = config.Filter(obj.GetAnnotations(), accessor.Settings().AnnotationMatches, accessor.AnnotationsEnabledForType(), accessor.Settings())
	}
	metricLabels := config.MetricLabels{
		"workload":      objectName,
		"namespace":     namespace,
		"resource_type": accessor.ResourceType(),
	}
	return types.ResourceTags{
		Type:         config.Pod,
		Name:         objectName,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
