// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc.
// SPDX-License-Identifier: Apache-2.0

package hook_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/admission/v1"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/http/hook"
)

func TestHandler_Execute(t *testing.T) {

	ctx := context.Background()
	tests := []struct {
		name        string
		operation   v1.Operation
		admitFunc   hook.AdmitFunc
		expectErr   bool
		expectMsg   string
		expectAllow bool
	}{
		{
			name:      "Create operation",
			operation: v1.Create,
			admitFunc: func(ctx context.Context, r *hook.Request) (*hook.Result, error) {
				return &hook.Result{Allowed: true, Msg: "Create operation successful"}, nil
			},
			expectErr:   false,
			expectMsg:   "Create operation successful",
			expectAllow: true,
		},
		{
			name:      "Update operation",
			operation: v1.Update,
			admitFunc: func(ctx context.Context, r *hook.Request) (*hook.Result, error) {
				return &hook.Result{Allowed: true, Msg: "Update operation successful"}, nil
			},
			expectErr:   false,
			expectMsg:   "Update operation successful",
			expectAllow: true,
		},
		{
			name:      "Delete operation",
			operation: v1.Delete,
			admitFunc: func(ctx context.Context, r *hook.Request) (*hook.Result, error) {
				return &hook.Result{Allowed: true, Msg: "Delete operation successful"}, nil
			},
			expectErr:   false,
			expectMsg:   "Delete operation successful",
			expectAllow: true,
		},
		{
			name:      "Connect operation",
			operation: v1.Connect,
			admitFunc: func(ctx context.Context, r *hook.Request) (*hook.Result, error) {
				return &hook.Result{Allowed: true, Msg: "Connect operation successful"}, nil
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
			expectAllow: false,
		},
	}

	for _, tt := range tests {
		tt := tt // Capture range variable
		t.Run(tt.name, func(t *testing.T) {
			// Initialize the handler with only the relevant AdmitFunc based on the operation
			h := &hook.Handler{}
			switch tt.operation {
			case v1.Create:
				h.Create = tt.admitFunc
			case v1.Update:
				h.Update = tt.admitFunc
			case v1.Delete:
				h.Delete = tt.admitFunc
			case v1.Connect:
				h.Connect = tt.admitFunc
				// No default case needed; invalid operations will have no handler set
			}

			// Create a mock AdmissionRequest
			req := &hook.Request{
				Operation: tt.operation,
				// You can add more fields here if your handler uses them
			}

			// Execute the handler
			result, err := h.Execute(ctx, req)

			// Assert expectations
			if tt.expectErr {
				assert.Error(t, err, "Expected an error but got none")
			} else {
				assert.NoError(t, err, "Did not expect an error but got one")
				assert.Equal(t, tt.expectMsg, result.Msg, "Unexpected message in result")
				assert.Equal(t, tt.expectAllow, result.Allowed, "Unexpected allowed status in result")
			}
		})
	}
}
