// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net"
	"net/http"
	"syscall"

	"github.com/pkg/errors"
)

var (
	ErrStatusBadRequest                    = errors.New("badrequest error")
	ErrStatusUnauthorized                  = errors.New("unauthorized error")
	ErrStatusPaymentRequired               = errors.New("paymentrequired error")
	ErrStatusForbidden                     = errors.New("forbidden error")
	ErrStatusNotFound                      = errors.New("not found error")
	ErrStatusMethodNotAllowed              = errors.New("method not allowed error")
	ErrStatusNotAcceptable                 = errors.New("not acceptable error")
	ErrStatusProxyAuthRequired             = errors.New("proxy authrequired error")
	ErrStatusRequestTimeout                = errors.New("request timeout error")
	ErrStatusConflict                      = errors.New("conflict error")
	ErrStatusGone                          = errors.New("gone error")
	ErrStatusLengthRequired                = errors.New("length required error")
	ErrStatusPreconditionFailed            = errors.New("precondition failed error")
	ErrStatusRequestEntityTooLarge         = errors.New("requestentity too large error")
	ErrStatusRequestURITooLong             = errors.New("request uri too long error")
	ErrStatusUnsupportedMediaType          = errors.New("unsupported media type error")
	ErrStatusRequestedRangeNotSatisfiable  = errors.New("requested range not satisfiable error")
	ErrStatusExpectationFailed             = errors.New("expectation failed error")
	ErrStatusTeapot                        = errors.New("teapot error")
	ErrStatusMisdirectedRequest            = errors.New("misdirected request error")
	ErrStatusUnprocessableEntity           = errors.New("unprocessable entity error")
	ErrStatusLocked                        = errors.New("locked error")
	ErrStatusFailedDependency              = errors.New("failed dependency error")
	ErrStatusTooEarly                      = errors.New("too early error")
	ErrStatusUpgradeRequired               = errors.New("upgrade required error")
	ErrStatusPreconditionRequired          = errors.New("precondition required error")
	ErrStatusTooManyRequests               = errors.New("too many requests error")
	ErrStatusRequestHeaderFieldsTooLarge   = errors.New("request header fields too large error")
	ErrStatusInternalServerError           = errors.New("internal server error")
	ErrStatusNotImplemented                = errors.New("not implemented error")
	ErrStatusBadGateway                    = errors.New("bad gateway error")
	ErrStatusServiceUnavailable            = errors.New("service unavailable error")
	ErrStatusGatewayTimeout                = errors.New("gateway timeout error")
	ErrStatusHTTPVersionNotSupported       = errors.New("http version not supported error")
	ErrStatusVariantAlsoNegotiates         = errors.New("variant also negotiates error")
	ErrStatusInsufficientStorage           = errors.New("insufficient storage error")
	ErrStatusLoopDetected                  = errors.New("loop detected error")
	ErrStatusNotExtended                   = errors.New("not extended error")
	ErrStatusNetworkAuthenticationRequired = errors.New("network authentication required error")
)

func classifyNetworkError(err error) string {
	cause := err
	for {
		// Unwrap was added in Go 1.13.
		// See https://github.com/golang/go/issues/36781
		if unwrap, ok := cause.(interface{ Unwrap() error }); ok {
			cause = unwrap.Unwrap()
			continue
		}
		break
	}

	// DNSError.IsNotFound was added in Go 1.13.
	// See https://github.com/golang/go/issues/28635
	if cause, ok := cause.(*net.DNSError); ok && cause.Err == "no such host" {
		return "name not found"
	}

	if cause, ok := cause.(syscall.Errno); ok {
		if cause == 10061 || cause == syscall.ECONNREFUSED { //nolint:revive // The source for 10061 is unknown
			return "connection refused"
		}
	}

	if cause, ok := cause.(net.Error); ok && cause.Timeout() {
		return "timeout"
	}

	return ""
}

func ToError(code int) error {
	switch code {
	case http.StatusBadRequest:
		return ErrStatusBadRequest
	case http.StatusUnauthorized:
		return ErrStatusUnauthorized
	case http.StatusPaymentRequired:
		return ErrStatusPaymentRequired
	case http.StatusForbidden:
		return ErrStatusForbidden
	case http.StatusNotFound:
		return ErrStatusNotFound
	case http.StatusMethodNotAllowed:
		return ErrStatusMethodNotAllowed
	case http.StatusNotAcceptable:
		return ErrStatusNotAcceptable
	case http.StatusProxyAuthRequired:
		return ErrStatusProxyAuthRequired
	case http.StatusRequestTimeout:
		return ErrStatusRequestTimeout
	case http.StatusConflict:
		return ErrStatusConflict
	case http.StatusGone:
		return ErrStatusGone
	case http.StatusLengthRequired:
		return ErrStatusLengthRequired
	case http.StatusPreconditionFailed:
		return ErrStatusPreconditionFailed
	case http.StatusRequestEntityTooLarge:
		return ErrStatusRequestEntityTooLarge
	case http.StatusRequestURITooLong:
		return ErrStatusRequestURITooLong
	case http.StatusUnsupportedMediaType:
		return ErrStatusUnsupportedMediaType
	case http.StatusRequestedRangeNotSatisfiable:
		return ErrStatusRequestedRangeNotSatisfiable
	case http.StatusExpectationFailed:
		return ErrStatusExpectationFailed
	case http.StatusTeapot:
		return ErrStatusTeapot
	case http.StatusMisdirectedRequest:
		return ErrStatusMisdirectedRequest
	case http.StatusUnprocessableEntity:
		return ErrStatusUnprocessableEntity
	case http.StatusLocked:
		return ErrStatusLocked
	case http.StatusFailedDependency:
		return ErrStatusFailedDependency
	case http.StatusTooEarly:
		return ErrStatusTooEarly
	case http.StatusUpgradeRequired:
		return ErrStatusUpgradeRequired
	case http.StatusPreconditionRequired:
		return ErrStatusPreconditionRequired
	case http.StatusTooManyRequests:
		return ErrStatusTooManyRequests
	case http.StatusRequestHeaderFieldsTooLarge:
		return ErrStatusRequestHeaderFieldsTooLarge
	case http.StatusInternalServerError:
		return ErrStatusInternalServerError
	case http.StatusNotImplemented:
		return ErrStatusNotImplemented
	case http.StatusBadGateway:
		return ErrStatusBadGateway
	case http.StatusServiceUnavailable:
		return ErrStatusServiceUnavailable
	case http.StatusGatewayTimeout:
		return ErrStatusGatewayTimeout
	case http.StatusHTTPVersionNotSupported:
		return ErrStatusHTTPVersionNotSupported
	case http.StatusVariantAlsoNegotiates:
		return ErrStatusVariantAlsoNegotiates
	case http.StatusInsufficientStorage:
		return ErrStatusInsufficientStorage
	case http.StatusLoopDetected:
		return ErrStatusLoopDetected
	case http.StatusNotExtended:
		return ErrStatusNotExtended
	case http.StatusNetworkAuthenticationRequired:
		return ErrStatusNetworkAuthenticationRequired
	}
	return nil
}
