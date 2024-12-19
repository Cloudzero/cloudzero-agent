// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"

	"github.com/stretchr/testify/assert"
)

type MockClusterScraper struct{}

func (m *MockClusterScraper) Start(context.Context) {}

func TestNewScraperHandler(t *testing.T) {
	settings := &config.Settings{}
	scraper := &MockClusterScraper{}
	handler := NewScraperHandler(scraper, settings)

	req, err := http.NewRequest("POST", "/scrape", nil)
	assert.NoError(t, err)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	// handler.StartScrape(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

}
