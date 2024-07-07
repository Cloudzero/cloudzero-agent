// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package http_test

import (
	net "net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/http"
)

func TestHTTP_ToError(t *testing.T) {
	tests := []struct {
		code int
		err  error
	}{
		{net.StatusOK, nil},
		{net.StatusBadRequest, http.ErrStatusBadRequest},
		{net.StatusUnauthorized, http.ErrStatusUnauthorized},
		{net.StatusPaymentRequired, http.ErrStatusPaymentRequired},
		{net.StatusForbidden, http.ErrStatusForbidden},
		{net.StatusNotFound, http.ErrStatusNotFound},
		{net.StatusMethodNotAllowed, http.ErrStatusMethodNotAllowed},
		{net.StatusNotAcceptable, http.ErrStatusNotAcceptable},
		{net.StatusProxyAuthRequired, http.ErrStatusProxyAuthRequired},
		{net.StatusRequestTimeout, http.ErrStatusRequestTimeout},
		{net.StatusConflict, http.ErrStatusConflict},
		{net.StatusGone, http.ErrStatusGone},
		{net.StatusLengthRequired, http.ErrStatusLengthRequired},
		{net.StatusPreconditionFailed, http.ErrStatusPreconditionFailed},
		{net.StatusRequestEntityTooLarge, http.ErrStatusRequestEntityTooLarge},
		{net.StatusRequestURITooLong, http.ErrStatusRequestURITooLong},
		{net.StatusUnsupportedMediaType, http.ErrStatusUnsupportedMediaType},
		{net.StatusRequestedRangeNotSatisfiable, http.ErrStatusRequestedRangeNotSatisfiable},
		{net.StatusExpectationFailed, http.ErrStatusExpectationFailed},
		{net.StatusTeapot, http.ErrStatusTeapot},
		{net.StatusMisdirectedRequest, http.ErrStatusMisdirectedRequest},
		{net.StatusUnprocessableEntity, http.ErrStatusUnprocessableEntity},
		{net.StatusLocked, http.ErrStatusLocked},
		{net.StatusFailedDependency, http.ErrStatusFailedDependency},
		{net.StatusTooEarly, http.ErrStatusTooEarly},
		{net.StatusUpgradeRequired, http.ErrStatusUpgradeRequired},
		{net.StatusPreconditionRequired, http.ErrStatusPreconditionRequired},
		{net.StatusTooManyRequests, http.ErrStatusTooManyRequests},
		{net.StatusRequestHeaderFieldsTooLarge, http.ErrStatusRequestHeaderFieldsTooLarge},
		{net.StatusInternalServerError, http.ErrStatusInternalServerError},
		{net.StatusNotImplemented, http.ErrStatusNotImplemented},
		{net.StatusBadGateway, http.ErrStatusBadGateway},
		{net.StatusServiceUnavailable, http.ErrStatusServiceUnavailable},
		{net.StatusGatewayTimeout, http.ErrStatusGatewayTimeout},
		{net.StatusHTTPVersionNotSupported, http.ErrStatusHTTPVersionNotSupported},
		{net.StatusVariantAlsoNegotiates, http.ErrStatusVariantAlsoNegotiates},
		{net.StatusInsufficientStorage, http.ErrStatusInsufficientStorage},
		{net.StatusLoopDetected, http.ErrStatusLoopDetected},
		{net.StatusNotExtended, http.ErrStatusNotExtended},
		{net.StatusNetworkAuthenticationRequired, http.ErrStatusNetworkAuthenticationRequired},
	}

	for _, tt := range tests {
		err := http.ToError(tt.code)
		assert.ErrorIs(t, err, tt.err)
	}
}
