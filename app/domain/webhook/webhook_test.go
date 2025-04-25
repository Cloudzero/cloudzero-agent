// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc.
// SPDX-License-Identifier: Apache-2.0
package webhook_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
)

func TestWebhookControllerReview(t *testing.T) {
	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	clock := mocks.NewMockClock(time.Now())
	store := mocks.NewMockResourceStore(mockCtl)
	settings := &config.Settings{}

	controller, err := webhook.NewWebhookFactory(store, settings, clock)
	assert.NoError(t, err)

	ar := makePodObjectRequest(metav1.ObjectMeta{
		Name:      "test-pod",
		Namespace: "default",
		Labels: map[string]string{
			"app": "test",
		},
		Annotations: map[string]string{
			"annotation-key": "annotation-value",
		},
	})

	result, err := controller.Review(context.Background(), ar)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)

	// Check unsupported types
	ar = makeUnsupportedRequest()
	result, err = controller.Review(context.Background(), ar)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
}

func TestWebhookController_GetGVKSupport(t *testing.T) {
	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	clock := mocks.NewMockClock(time.Now())
	store := mocks.NewMockResourceStore(mockCtl)
	settings := &config.Settings{}

	controller, err := webhook.NewWebhookFactory(store, settings, clock)
	assert.NoError(t, err)

	supportedGVK := controller.GetSupported()

	// Check if specific GVKs are present
	assert.Contains(t, supportedGVK, types.GroupApps)
	assert.Contains(t, supportedGVK[types.GroupApps], types.V1)
	assert.Contains(t, supportedGVK[types.GroupApps][types.V1], types.KindDeployment)
	assert.IsType(t, &appsv1.Deployment{}, supportedGVK[types.GroupApps][types.V1][types.KindDeployment])

	assert.Contains(t, supportedGVK, types.GroupCore)
	assert.Contains(t, supportedGVK[types.GroupCore], types.V1)
	assert.Contains(t, supportedGVK[types.GroupCore][types.V1], types.KindPod)
	assert.IsType(t, &corev1.Pod{}, supportedGVK[types.GroupCore][types.V1][types.KindPod])
}

func TestWebhookController_IsGVKGetSupported(t *testing.T) {
	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	clock := mocks.NewMockClock(time.Now())
	store := mocks.NewMockResourceStore(mockCtl)
	settings := &config.Settings{}

	controller, err := webhook.NewWebhookFactory(store, settings, clock)
	assert.NoError(t, err)

	// Test supported GVK
	assert.True(t, controller.IsSupported(types.GroupApps, types.V1, types.KindDeployment))
	assert.True(t, controller.IsSupported(types.GroupCore, types.V1, types.KindPod))

	// Test unsupported GVK
	assert.False(t, controller.IsSupported("unknownGroup", "v1", "unknownKind"))
	assert.False(t, controller.IsSupported(types.GroupApps, "v2", types.KindDeployment))
	assert.False(t, controller.IsSupported(types.GroupCore, types.V1, "unknownKind"))
}

func makePodObjectRequest(o metav1.ObjectMeta) *types.AdmissionReview {
	return &types.AdmissionReview{
		Operation: types.OperationCreate,
		RequestGVK: &metav1.GroupVersionKind{
			Group:   types.GroupCore,
			Version: types.V1,
			Kind:    types.KindPod,
		},
		RequestGVR: &metav1.GroupVersionResource{
			Group:    types.GroupCore,
			Version:  types.V1,
			Resource: "pods",
		},
		NewObjectRaw: getRawObject(appsv1.SchemeGroupVersion, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        o.Name,
				Namespace:   o.Namespace,
				Labels:      o.Labels,
				Annotations: o.Annotations,
			},
		}),
	}
}

func makeUnsupportedRequest() *types.AdmissionReview {
	return &types.AdmissionReview{
		RequestGVK: &metav1.GroupVersionKind{
			Group:   types.GroupCore,
			Version: types.V1,
			Kind:    "configmap",
		},
		RequestGVR: &metav1.GroupVersionResource{
			Group:    types.GroupCore,
			Version:  types.V1,
			Resource: "configmaps",
		},
		Operation: types.OperationCreate,
		NewObjectRaw: getRawObject(appsv1.SchemeGroupVersion, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{},
		}),
	}
}

func getRawObject(s schema.GroupVersion, o runtime.Object) []byte {
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	codecs := serializer.NewCodecFactory(scheme)
	encoder := codecs.LegacyCodec(s)
	raw, _ := runtime.Encode(encoder, o)
	return raw
}
