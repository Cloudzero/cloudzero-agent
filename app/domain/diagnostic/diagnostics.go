// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package diagnostic contains an interface to be implemented by diagnostics providers.
package diagnostic

import (
	"context"
	"net/http"

	"github.com/cloudzero/cloudzero-agent/app/types/status"
)

// Provider is the interface that must be implemented by a diagnostics provider
// to run a targeted check(s) returning one or more results
type Provider interface {
	// Check will perform a targeted check(s) setting meaningful values on the status object
	// and only will return an error if the condition is unrecoverable
	Check(_ context.Context, _ *http.Client, _ status.Accessor) error
}
