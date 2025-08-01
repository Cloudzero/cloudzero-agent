// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package runner contains tools for running diagnostics.
package runner

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/cloudzero/cloudzero-agent/app/build"
	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic/catalog"
	logging "github.com/cloudzero/cloudzero-agent/app/logging/validator"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
)

type Engine interface {
	// Run executes the engine
	Run(context.Context) (status.Accessor, error)
}

type runner struct {
	stage  string
	cfg    *config.Settings
	logger *logrus.Entry
	client *http.Client
	reg    catalog.Registry

	pre  []diagnostic.Provider
	plan []diagnostic.Provider
	post []diagnostic.Provider
}

func NewRunner(c *config.Settings, reg catalog.Registry, stage string) Engine {
	r := &runner{
		cfg:    c,
		stage:  stage,
		reg:    reg,
		logger: logging.NewLogger().WithField(logging.OpField, "runner"),
		client: http.DefaultClient,
	}

	// Add actions needed to run before the main diagnostic checks
	if stage == config.ContextStageInit {
		r.AddPreStep(reg.Get(config.DiagnosticInternalInitStart)...)
	}

	// Add main actions based on the configuration
	for _, s := range c.Diagnostics.Stages {
		if s.Name != stage {
			continue
		}
		r.AddStep(reg.Get(s.Checks...)...)
	}

	// Add actions needed to run after the main diagnostic checks
	switch stage {
	case config.ContextStageInit:
		r.AddPostStep(reg.Get(config.DiagnosticInternalInitStop)...)
	case config.ContextStageStart:
		r.AddPostStep(reg.Get(config.DiagnosticInternalPodStart)...)
	case config.ContextStageStop:
		r.AddPostStep(reg.Get(config.DiagnosticInternalPodStop)...)
	}
	return r
}

func (r *runner) AddPreStep(providers ...diagnostic.Provider) {
	r.pre = append(r.pre, providers...)
}

func (r *runner) AddStep(providers ...diagnostic.Provider) {
	r.plan = append(r.plan, providers...)
}

func (r *runner) AddPostStep(providers ...diagnostic.Provider) {
	r.post = append(r.post, providers...)
}

func (r *runner) Run(ctx context.Context) (status.Accessor, error) {
	recorder := status.NewAccessor(&status.ClusterStatus{})

	recorder.WriteToReport(func(cs *status.ClusterStatus) {
		cs.Account = r.cfg.Deployment.AccountID
		cs.Region = r.cfg.Deployment.Region
		cs.Name = r.cfg.Deployment.ClusterName
		cs.ValidatorVersion = build.GetVersion()
		cs.ChartVersion = r.cfg.Versions.ChartVersion
	})

	// Pre steps sequentially
	for _, pv := range r.pre {
		if err := pv.Check(ctx, r.client, recorder); err != nil {
			return recorder, err
		}
	}

	// Run steps in parallel

	errHistory := make([]error, len(r.plan))

	var wg sync.WaitGroup
	for i, p := range r.plan {
		wg.Add(1)
		go func(wgi *sync.WaitGroup, p diagnostic.Provider, i int) {
			defer wgi.Done()
			if err := p.Check(ctx, r.client, recorder); err != nil {
				errHistory[i] = err
			}
		}(&wg, p, i)
	}

	wg.Wait()

	if err := errors.Join(errHistory...); err != nil {
		return recorder, err
	}

	// Post steps sequentially
	for _, ps := range r.post {
		if err := ps.Check(ctx, r.client, recorder); err != nil {
			return recorder, err
		}
	}

	// check the results (and set correct end code)
	processFailures(ctx, recorder, r)()

	return recorder, nil
}

// this function returns a function which will set an error code if necessary
func processFailures(ctx context.Context, recorder status.Accessor, r *runner) func() {
	handleFailure := func() {}
	recorder.ReadFromReport(func(cs *status.ClusterStatus) {
		if r.stage != config.ContextStageInit {
			return
		}
		for _, c := range cs.Checks {
			if !c.Passing {
				if chkr := r.reg.Get(config.DiagnosticInternalInitFailed); len(chkr) > 0 {
					// set to read handler since we already hold the lock
					handleFailure = func() {
						_ = chkr[0].Check(ctx, r.client, recorder)
					}
				}
				break
			}
		}
	})
	return handleFailure
}
