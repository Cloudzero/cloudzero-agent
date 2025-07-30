// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // Duplication is acceptable we expect to extend the definitions later
package handler

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"

	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

var (
	storageOnce      sync.Once
	StorageInfoTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: types.ObservabilityMetric("storage_types_total"),
			Help: "Tracks the total number of storage class events, categorized by name and provisioner.",
		},
		[]string{"name", "provisioner"},
	)
)

type StorageClassConfigAccessor struct {
	settings *config.Settings
}

func NewStorageClassConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &StorageClassConfigAccessor{settings: settings}
}

func (s *StorageClassConfigAccessor) LabelsEnabled() bool {
	return s.settings.Filters.Labels.Enabled
}

func (s *StorageClassConfigAccessor) AnnotationsEnabled() bool {
	return s.settings.Filters.Annotations.Enabled
}

func (s *StorageClassConfigAccessor) LabelsEnabledForType() bool {
	return true
}

func (s *StorageClassConfigAccessor) AnnotationsEnabledForType() bool {
	return true
}

func (s *StorageClassConfigAccessor) ResourceType() config.ResourceType {
	return config.StorageClass
}

func (s *StorageClassConfigAccessor) Settings() *config.Settings {
	return s.settings
}

type StorageClassHandler struct {
	hook.Handler
	settings   *config.Settings
	clock      types.TimeProvider
	formatData DataFormatter
}

func NewStorageClassHandler(
	store types.ResourceStore,
	settings *config.Settings,
	clock types.TimeProvider,
	objTemplate metav1.Object,
) *hook.Handler {
	storageOnce.Do(func() {
		prometheus.MustRegister(StorageInfoTotal)
	})

	h := &StorageClassHandler{
		settings:   settings,
		clock:      clock,
		formatData: WorkloadDataFormatter, // TODO: stub
	}
	h.Accessor = NewStorageClassConfigAccessor(settings)
	h.ObjectType = objTemplate
	h.ObjectCreator = helper.NewStaticObjectCreator(objTemplate)
	h.Handler.Store = store
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Delete = h.Delete()
	h.Handler.Connect = h.Connect()
	return &h.Handler
}

func (h *StorageClassHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		// Use a type assertion to handle both v1 and v1beta1 StorageClass
		switch o := obj.(type) {
		case *storagev1.StorageClass:
			StorageInfoTotal.WithLabelValues(o.GetName(), o.Provisioner).Inc()
			debugPrintObject(o, "create "+config.ResourceTypeToMetricName[h.Accessor.ResourceType()])
		case *storagev1beta1.StorageClass:
			StorageInfoTotal.WithLabelValues(o.GetName(), o.Provisioner).Inc()
			debugPrintObject(o, "create "+config.ResourceTypeToMetricName[h.Accessor.ResourceType()])
		default:
			log.Warn().Msgf("unable to cast to object instance of type %T", obj)
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *StorageClassHandler) Update() hook.AdmitFunc {
	return hook.AllowAlways
}

func (h *StorageClassHandler) Delete() hook.AdmitFunc {
	return hook.AllowAlways
}

func (h *StorageClassHandler) Connect() hook.AdmitFunc {
	return hook.AllowAlways
}
