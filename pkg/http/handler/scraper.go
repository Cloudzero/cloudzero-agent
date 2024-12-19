// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"net/http"
	"sync"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
)

type ScraperHandler struct {
	http.Handler
	scraper  ClusterScraper
	settings *config.Settings
	mu       sync.Mutex
	ctx      context.Context
}

type ClusterScraper interface {
	// Start starts the scraper.
	Start(context.Context)
}

func NewScraperHandler(scraper ClusterScraper, settings *config.Settings) http.HandlerFunc {
	sh := &ScraperHandler{scraper: scraper, settings: settings, ctx: context.Background(), mu: sync.Mutex{}}
	return sh.StartScrape
}

func (sh *ScraperHandler) StartScrape(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "invalid method only POST requests are allowed", http.StatusMethodNotAllowed)
		return
	}
	sh.mu.Lock()
	defer sh.mu.Unlock()
	// Note - this is intentionally holding the caller
	sh.scraper.Start(r.Context())
	w.WriteHeader(http.StatusOK)
}
