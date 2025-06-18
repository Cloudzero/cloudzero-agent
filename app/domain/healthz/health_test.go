// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package healthz_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudzero/cloudzero-agent/app/domain/healthz"
)

func TestEndpointHandler(t *testing.T) {
	t.Run("should return 200 OK when all checks pass", func(t *testing.T) {
		healthz.Register("check1", func() error { return nil })

		h := healthz.NewHealthz()

		req, err := http.NewRequest("GET", "/healthz", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := h.EndpointHandler()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "ok", rr.Body.String())
	})

	t.Run("should return 500 Internal Server Error when a check fails", func(t *testing.T) {
		healthz.Register("check1", func() error { return nil })
		healthz.Register("check2", func() error { return assert.AnError })
		h := healthz.NewHealthz()

		req, err := http.NewRequest("GET", "/healthz", nil)
		require.NoError(t, err)

		rr := httptest.NewRecorder()
		handler := h.EndpointHandler()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "check2 failed: assert.AnError general error for testing")
	})
}
