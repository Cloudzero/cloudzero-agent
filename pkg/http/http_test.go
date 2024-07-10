// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package http_test

import (
	"context"
	net "net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/http"
	"github.com/cloudzero/cloudzero-agent-validator/test"
)

const (
	mockUrl = "http://example.com"
)

func TestHTTP_Do(t *testing.T) {
	ctx := context.Background()
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	queryParams := map[string]string{
		"key": "value with space",
	}

	mockClient := test.NewHTTPMock()
	mockClient.Expect("GET", "Hello World", net.StatusOK, nil)

	httpClient := mockClient.HTTPClient()
	code, err := http.Do(ctx, httpClient, net.MethodGet, headers, queryParams, mockUrl, nil)
	assert.NoError(t, err)
	assert.Equal(t, net.StatusOK, code)
}
