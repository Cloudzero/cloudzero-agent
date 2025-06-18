// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package hook contains structures and interfaces for implementing admission webhooks handlers.
package hook

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/logging/instr"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

// AdmitFunc defines how to process an admission request
type AdmitFunc func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error)

// Handler represents the set of functions for each operation in an admission webhook.
type Handler struct {
	Accessor      config.ConfigAccessor
	ObjectCreator types.ObjectCreator
	ObjectType    metav1.Object
	Create        AdmitFunc
	Delete        AdmitFunc
	Update        AdmitFunc
	Connect       AdmitFunc
	Store         types.ResourceStore
}

// Execute evaluates the request and try to execute the function for operation specified in the request.
func (h *Handler) Execute(ctx context.Context, r *types.AdmissionReview) (*types.AdmissionResponse, error) {
	var res *types.AdmissionResponse
	var err error

	// No object creator? Don't handle the review
	if h.ObjectCreator == nil {
		return &types.AdmissionResponse{Allowed: true, Message: fmt.Sprintf("no object creator: %s", r.Operation)}, nil
	}

	raw := r.NewObjectRaw
	if r.Operation == types.OperationDelete {
		raw = r.OldObjectRaw
	}

	// Create a new object from the raw type.
	runtimeObj, err := h.ObjectCreator.NewObject(raw)
	if err != nil {
		// RULE: always return allow
		return &types.AdmissionResponse{Allowed: true, Message: fmt.Sprintf("Unable to create object from raw: %s: %s", r.Operation, err.Error())}, nil
	}

	validatingObj, ok := runtimeObj.(metav1.Object)
	// Get the object.
	if !ok {
		// RULE: always return allow
		return &types.AdmissionResponse{Allowed: true, Message: "Impossible to type assert the deep copy to metav1.Object"}, nil
	}

	switch r.Operation {
	case types.OperationCreate:
		err = instr.RunSpan(ctx, "executeAdmissionsReviewRequest_Create", func(ctx context.Context, span *instr.Span) error {
			res, err = middleware(ctx, h.Create, r, validatingObj)
			return err
		})
	case types.OperationUpdate:
		err = instr.RunSpan(ctx, "executeAdmissionsReviewRequest_Update", func(ctx context.Context, span *instr.Span) error {
			res, err = middleware(ctx, h.Update, r, validatingObj)
			return err
		})
	case types.OperationDelete:
		err = instr.RunSpan(ctx, "executeAdmissionsReviewRequest_Delete", func(ctx context.Context, span *instr.Span) error {
			res, err = middleware(ctx, h.Delete, r, validatingObj)
			return err
		})
	case types.OperationConnect:
		err = instr.RunSpan(ctx, "executeAdmissionsReviewRequest_Connect", func(ctx context.Context, span *instr.Span) error {
			res, err = middleware(ctx, h.Connect, r, validatingObj)
			return err
		})
	default:
		return &types.AdmissionResponse{Allowed: true, Message: fmt.Sprintf("Invalid operation: %s", r.Operation)}, nil
	}

	return res, err
}

func middleware(ctx context.Context, fn AdmitFunc, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
	// This is a setup which would allow registration of middleware functions
	// which we could invoke before finally invoking the actual function.
	if fn == nil {
		return nil, fmt.Errorf("operation %s is not registered", r.Operation)
	}
	return fn(ctx, r, obj)
}
