// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package hook contains structures and interfaces for implementing admission webhooks handlers.
package hook

import (
	"context"
	"fmt"

	v1 "k8s.io/api/admission/v1"

	"github.com/cloudzero/cloudzero-agent-validator/app/instr"
	"github.com/cloudzero/cloudzero-agent-validator/app/types"
)

type Request = v1.AdmissionRequest

// Result contains the result of an admission request
type Result struct {
	Allowed bool
	Msg     string
}

// AdmitFunc defines how to process an admission request
type AdmitFunc func(ctx context.Context, r *Request) (*Result, error)

// Handler represents the set of functions for each operation in an admission webhook.
type Handler struct {
	Create    AdmitFunc
	Delete    AdmitFunc
	Update    AdmitFunc
	Connect   AdmitFunc
	ErrorChan chan<- error
	Store     types.ResourceStore
}

// Execute evaluates the request and try to execute the function for operation specified in the request.
func (h *Handler) Execute(ctx context.Context, r *Request) (*Result, error) {
	var res *Result
	var err error
	switch r.Operation {
	case v1.Create:
		err = instr.RunSpan(ctx, "executeAdmissionsReviewRequest_Create", func(ctx context.Context, span *instr.Span) error {
			res, err = middleware(ctx, h.Create, r)
			return err
		})
	case v1.Update:
		err = instr.RunSpan(ctx, "executeAdmissionsReviewRequest_Update", func(ctx context.Context, span *instr.Span) error {
			res, err = middleware(ctx, h.Update, r)
			return err
		})
	case v1.Delete:
		err = instr.RunSpan(ctx, "executeAdmissionsReviewRequest_Delete", func(ctx context.Context, span *instr.Span) error {
			res, err = middleware(ctx, h.Delete, r)
			return err
		})
	case v1.Connect:
		err = instr.RunSpan(ctx, "executeAdmissionsReviewRequest_Connect", func(ctx context.Context, span *instr.Span) error {
			res, err = middleware(ctx, h.Connect, r)
			return err
		})
	default:
		return &Result{Msg: fmt.Sprintf("Invalid operation: %s", r.Operation)}, nil
	}

	return res, err
}

func middleware(ctx context.Context, fn AdmitFunc, r *Request) (*Result, error) {
	// This is a setup which would allow registration of middleware functions
	// which we could invoke before finally invoking the actual function.
	if fn == nil {
		return nil, fmt.Errorf("operation %s is not registered", r.Operation)
	}
	return fn(ctx, r)
}
