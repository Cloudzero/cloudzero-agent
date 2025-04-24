// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc.
// SPDX-License-Identifier: Apache-2.0

package hook_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

func TestHandler_Execute(t *testing.T) {
	ctx := context.Background()

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "namespace",
			Name:      "deployment",
		},
	}

	tests := []struct {
		name        string
		operation   types.AdmissionReviewOp
		admitFunc   hook.AdmitFunc
		expectErr   bool
		expectMsg   string
		expectAllow bool
	}{
		{
			name:      "Create operation",
			operation: types.OperationCreate,
			admitFunc: func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
				return &types.AdmissionResponse{Allowed: true, Message: "Create operation successful"}, nil
			},
			expectErr:   false,
			expectMsg:   "Create operation successful",
			expectAllow: true,
		},
		{
			name:      "Update operation",
			operation: types.OperationUpdate,
			admitFunc: func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
				return &types.AdmissionResponse{Allowed: true, Message: "Update operation successful"}, nil
			},
			expectErr:   false,
			expectMsg:   "Update operation successful",
			expectAllow: true,
		},
		{
			name:      "Delete operation",
			operation: types.OperationDelete,
			admitFunc: func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
				return &types.AdmissionResponse{Allowed: true, Message: "Delete operation successful"}, nil
			},
			expectErr:   false,
			expectMsg:   "Delete operation successful",
			expectAllow: true,
		},
		{
			name:      "Connect operation",
			operation: types.OperationConnect,
			admitFunc: func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
				return &types.AdmissionResponse{Allowed: true, Message: "Connect operation successful"}, nil
			},
			expectErr:   false,
			expectMsg:   "Connect operation successful",
			expectAllow: true,
		},
		{
			name:        "Invalid operation",
			operation:   "Invalid",
			admitFunc:   nil, // No admit function for invalid operation
			expectErr:   false,
			expectMsg:   "Invalid operation: Invalid",
			expectAllow: true,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			// Initialize the handler with only the relevant AdmitFunc based on the operation
			h := &hook.Handler{
				ObjectCreator: helper.NewStaticObjectCreator(&appsv1.Deployment{}),
			}
			switch tt.operation {
			case types.OperationCreate:
				h.Create = tt.admitFunc
			case types.OperationUpdate:
				h.Update = tt.admitFunc
			case types.OperationDelete:
				h.Delete = tt.admitFunc
			case types.OperationConnect:
				h.Connect = tt.admitFunc
				// No default case needed; invalid operations will have no handler set
			}

			// Create a mock AdmissionRequest
			req := &types.AdmissionReview{
				Operation: tt.operation,
				// You can add more fields here if your handler uses them
				NewObjectRaw: getRawObject(appsv1.SchemeGroupVersion, deployment),
				OldObjectRaw: getRawObject(appsv1.SchemeGroupVersion, deployment),
			}

			// Execute the handler
			result, err := h.Execute(ctx, req)

			// Assert expectations
			if tt.expectErr {
				assert.Error(t, err, "Expected an error but got none")
			} else {
				assert.NoError(t, err, "Did not expect an error but got one")
				assert.Equal(t, tt.expectMsg, result.Message, "Unexpected message in result")
				assert.Equal(t, tt.expectAllow, result.Allowed, "Unexpected allowed status in result")
			}
		})
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
