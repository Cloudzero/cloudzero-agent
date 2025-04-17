// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package catalog contains the registry of diagnostics.
package catalog

import (
	"context"
	"sync"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/diagnostic"
	"github.com/cloudzero/cloudzero-agent/app/diagnostic/cz"
	"github.com/cloudzero/cloudzero-agent/app/diagnostic/k8s"
	"github.com/cloudzero/cloudzero-agent/app/diagnostic/kms"
	promcfg "github.com/cloudzero/cloudzero-agent/app/diagnostic/prom/config"
	promver "github.com/cloudzero/cloudzero-agent/app/diagnostic/prom/version"
	"github.com/cloudzero/cloudzero-agent/app/diagnostic/stage"
	"github.com/cloudzero/cloudzero-agent/app/diagnostic/webhook"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
)

type Registry interface {
	// Has checks if the specified diagnostic is registered
	Has(id string) bool
	// Get retrieves the diagnostic providers for the given IDs
	Get(ids ...string) []diagnostic.Provider
	// List returns the list of registered diagnostics
	List() []string
}

type providerInfo struct {
	private bool
	handler diagnostic.Provider
}

type registry struct {
	mu        sync.Mutex
	providers map[string]providerInfo
}

func NewCatalog(ctx context.Context, c *config.Settings) Registry {
	r := &registry{
		providers: make(map[string]providerInfo),
	}
	// Register checks
	r.add(config.DiagnosticAPIKey, false, cz.NewProvider(ctx, c))
	r.add(config.DiagnosticK8sVersion, false, k8s.NewProvider(ctx, c))
	r.add(config.DiagnosticKMS, false, kms.NewProvider(ctx, c))
	r.add(config.DiagnosticScrapeConfig, false, promcfg.NewProvider(ctx, c))
	r.add(config.DiagnosticPrometheusVersion, false, promver.NewProvider(ctx, c))
	r.add(config.DiagnosticInsightsIngress, false, webhook.NewProvider(ctx, c))

	// Internal diagnostics emitted based on stage
	r.add(config.DiagnosticInternalInitStart, true, stage.NewProvider(ctx, c, status.StatusType_STATUS_TYPE_INIT_STARTED))
	r.add(config.DiagnosticInternalInitStop, true, stage.NewProvider(ctx, c, status.StatusType_STATUS_TYPE_INIT_OK))
	r.add(config.DiagnosticInternalInitFailed, true, stage.NewProvider(ctx, c, status.StatusType_STATUS_TYPE_INIT_FAILED))
	r.add(config.DiagnosticInternalPodStart, true, stage.NewProvider(ctx, c, status.StatusType_STATUS_TYPE_POD_STARTED))
	r.add(config.DiagnosticInternalPodStop, true, stage.NewProvider(ctx, c, status.StatusType_STATUS_TYPE_POD_STOPPING))

	return r
}

func (r *registry) Get(ids ...string) []diagnostic.Provider {
	providers := []diagnostic.Provider{}
	if len(ids) == 0 {
		return providers
	}

	needed := []string{}
	for _, id := range ids {
		if r.Has(id) {
			needed = append(needed, id)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, id := range needed {
		providers = append(providers, r.providers[id].handler)
	}
	return providers
}

func (r *registry) Has(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.providers[id]
	return ok
}

func (r *registry) List() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	ids := []string{}
	for id, p := range r.providers {
		if p.private {
			continue
		}
		ids = append(ids, id)
	}
	return ids
}

func (r *registry) add(name string, private bool, provider diagnostic.Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if provider == nil {
		panic("diagnostics: Register provider is nil")
	}
	if _, dup := r.providers[name]; dup {
		panic("diagnostics: Register called twice for provider " + name)
	}
	r.providers[name] = providerInfo{private, provider}
}
