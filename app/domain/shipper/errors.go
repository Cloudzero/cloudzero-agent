// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"errors"
	"fmt"
)

// ShipperError is a wrapper around an error that includes a status code `Code()` function
// to use in prometheus metrics.
type ShipperError interface {
	error
	// Code returns the associated error code as a string.
	Code() string
}

// shipperError implements the ShipperError interface.
type shipperError struct {
	code string
	msg  string

	// allow for error wrapping
	err error
}

// Error implements the error interface.
func (se *shipperError) Error() string {
	// If there is a wrapped error, include it.
	if se.err != nil {
		return fmt.Sprintf("%s: %v", se.msg, se.err)
	}
	return se.msg
}

// Code returns the shipper error code.
func (se *shipperError) Code() string {
	return se.code
}

// Unwrap allows errors.Unwrap to work with shipperError.
func (se *shipperError) Unwrap() error {
	return se.err
}

// NewShipperError creates a new shipperError instance.
// If err is non-nil, it will be wrapped.
func NewShipperError(code, msg string) ShipperError {
	return &shipperError{
		code: code,
		msg:  msg,
	}
}

// Define sentinel errors for each expected condition. These errors
// can be used with errors.Is and errors.As to detect specific issues.
var (
	// HTTP errors
	ErrUnauthorized      = NewShipperError("err-unauthorized", "unauthorized request - possible invalid API key")
	ErrNoURLs            = NewShipperError("err-no-urls", "no presigned URLs returned")
	ErrInvalidShipperID  = NewShipperError("err-invalid-shipper-id", "failed to get the shipper id")
	ErrEncodeBody        = NewShipperError("err-encode-body", "failed to encode the body into a foreign format")
	ErrGetRemoteBase     = NewShipperError("err-get-remote-base", "failed to get the remote endpoint api base from the config file")
	ErrHTTPRequestFailed = NewShipperError("err-http-request-failed", "the http request failed")
	ErrHTTPUnknown       = NewShipperError("err-http-unknown", "there was an unknown issue with the http request")
	ErrInvalidBody       = NewShipperError("err-invalid-body", "decoding a response/object failed")
	ErrExpiredURL        = NewShipperError("err-expired-url", "the requested url has expired")

	ErrCreateDirectory = NewShipperError("err-dir-create", "failed to create the requested directory")
	ErrCreateLock      = NewShipperError("err-lock-create", "failed to create or acquire the lock")
	ErrReleaseLock     = NewShipperError("err-lock-release", "failed to release the lock")

	ErrFilesList  = NewShipperError("err-files-walk", "failed to list/walk the files")
	ErrFileRemove = NewShipperError("err-file-remove", "failed to remove a file")
	ErrFileCreate = NewShipperError("err-file-create", "failed to create a file")
	ErrFileRead   = NewShipperError("err-file-read", "failed to read a file")

	ErrStorageCleanup = NewShipperError("err-storage-cleanup", "failed to clean up the disk")
	ErrGetDiskUsage   = NewShipperError("err-get-disk-usage", "failed to get the disk usage")
)

// ShipperErrorDefault is the default code given when the specific error type is not found
const ShipperErrorDefault = "unknown error"

// GetErrStatusCode extracts the error code from any wrapped ShipperError.
// If no matching ShipperError is found in the chain, "-1" is returned to indicate an unknown error.
func GetErrStatusCode(err error) string {
	var se ShipperError
	if errors.As(err, &se) {
		return se.Code()
	}

	return ShipperErrorDefault
}
