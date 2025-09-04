// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package http_test

import (
	"context"
	net "net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	http "github.com/cloudzero/cloudzero-agent/app/http/client"
	"github.com/cloudzero/cloudzero-agent/tests/utils"
)

const (
	mockURL = "http://example.com"
)

func TestHTTP_Do(t *testing.T) {
	ctx := context.Background()
	headers := map[string]string{
		"Content-Type": "application/json",
	}
	queryParams := map[string]string{
		"key": "value with space",
	}

	mockClient := utils.NewHTTPMock()
	mockClient.Expect("GET", "Hello World", net.StatusOK, nil)

	httpClient := mockClient.HTTPClient()
	code, err := http.Do(ctx, httpClient, net.MethodGet, headers, queryParams, mockURL, nil)
	assert.NoError(t, err)
	assert.Equal(t, net.StatusOK, code)
}
