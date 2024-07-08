// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	connectTimeout = 15 * time.Second
)

func Do(
	ctx context.Context,
	client *http.Client,
	method string,
	headers map[string]string,
	queryParams map[string]string,
	uri string,
	body io.Reader,
) (int, error) {
	if client == nil {
		client = http.DefaultClient
	}

	// make sure each request doesn't hang for too long
	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, uri, body)
	if err != nil {
		return http.StatusInternalServerError, errors.Wrap(err, "create request")
	}

	// Set the headers
	for header, value := range headers {
		req.Header.Set(header, value)
	}
	setUserAgent(headers)

	// Set the query parameters
	values := req.URL.Query()
	for key, value := range queryParams {
		values.Add(key, value)
	}
	// make sure they are http encoded
	req.URL.RawQuery = values.Encode()

	resp, err := client.Do(req)
	if resp == nil {
		if msg := classifyNetworkError(err); msg != "" {
			logrus.WithError(err).WithField("message", msg).Error("network error")
			return http.StatusInternalServerError, errors.Wrap(err, fmt.Sprintf("network error: %s", req.URL.String()))
		}
		logrus.WithError(err).Error("Failed to make request")
		return http.StatusInternalServerError, err
	}
	return resp.StatusCode, ToError(resp.StatusCode)
}
