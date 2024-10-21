// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package hook

import (
	"fmt"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"

	v1 "k8s.io/api/admission/v1"
)

type Request = v1.AdmissionRequest

// Result contains the result of an admission request
type Result struct {
	Allowed bool
	Msg     string
}

// AdmitFunc defines how to process an admission request
type AdmitFunc func(r *Request) (*Result, error)

// Handler represents the set of functions for each operation in an admission webhook.
type Handler struct {
	Create    AdmitFunc
	Delete    AdmitFunc
	Update    AdmitFunc
	Connect   AdmitFunc
	Writer    storage.DatabaseWriter
	ErrorChan chan<- error
}

// Execute evaluates the request and try to execute the function for operation specified in the request.
func (h *Handler) Execute(r *Request) (*Result, error) {
	switch r.Operation {
	case v1.Create:
		return middleware(h.Create, r)
	case v1.Update:
		return middleware(h.Update, r)
	case v1.Delete:
		return middleware(h.Delete, r)
	case v1.Connect:
		return middleware(h.Connect, r)
	}

	return &Result{Msg: fmt.Sprintf("Invalid operation: %s", r.Operation)}, nil
}

func middleware(fn AdmitFunc, r *Request) (*Result, error) {
	// This is a setup which would allow registration of middleware functions
	// which we could invoke before finally invoking the actual function.
	if fn == nil {
		return nil, fmt.Errorf("operation %s is not registered", r.Operation)
	}
	return fn(r)
}
