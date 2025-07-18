// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package cz_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic/cz"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/cloudzero/cloudzero-agent/tests/utils"
)

const (
	mockURL = "http://example.com"
)

func makeReport() status.Accessor {
	return status.NewAccessor(&status.ClusterStatus{})
}

func TestChecker_CheckOK(t *testing.T) {
	cfg := &config.Settings{
		Cloudzero: config.Cloudzero{
			Host:       mockURL,
			Credential: "your-api-key",
		},
	}

	provider := cz.NewProvider(context.Background(), cfg)

	mock := utils.NewHTTPMock()
	mock.Expect(http.MethodGet, "Hello World", http.StatusOK, nil)
	client := mock.HTTPClient()

	accessor := makeReport()

	err := provider.Check(context.Background(), client, accessor)
	assert.NoError(t, err)

	accessor.ReadFromReport(func(s *status.ClusterStatus) {
		assert.Len(t, s.Checks, 1)
		assert.True(t, s.Checks[0].Passing)
		assert.Empty(t, s.Checks[0].Error)
	})
}

func TestChecker_CheckBadKey(t *testing.T) {
	cfg := &config.Settings{
		Cloudzero: config.Cloudzero{
			Host:       mockURL,
			Credential: "your-api-key",
		},
	}

	provider := cz.NewProvider(context.Background(), cfg)

	mock := utils.NewHTTPMock()
	mock.Expect(http.MethodGet, "", http.StatusUnauthorized, nil)
	client := mock.HTTPClient()

	accessor := makeReport()
	err := provider.Check(context.Background(), client, accessor)
	assert.NoError(t, err)

	accessor.ReadFromReport(func(s *status.ClusterStatus) {
		assert.Len(t, s.Checks, 1)
		assert.False(t, s.Checks[0].Passing)
		assert.NotEmpty(t, s.Checks[0].Error)
	})
}
