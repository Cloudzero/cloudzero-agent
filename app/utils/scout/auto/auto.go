// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package auto provides auto-detection capabilities for the CloudZero Scout.
//
// The auto package orchestrates multiple cloud provider Scouts to provide
// automatic detection.
package auto

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
)

// ErrNoCloudProviderDetected is returned when no cloud provider is detected
var ErrNoCloudProviderDetected = errors.New("no cloud provider detected")

// Scout implements the types.Scout interface with auto-detection capabilities.
type Scout struct {
	environmentInfo      types.EnvironmentInfo
	environmentInfoError error
	scout                types.Scout
	detectOnce           sync.Once
	environmentInfoOnce  sync.Once

	scouts []types.Scout
}

// NewScout creates a new auto-detection Scout that tries the provided scouts.
//
// Each scout is tried concurrently, but the first successful detection cancels
// the others to avoid unnecessary work.
func NewScout(scouts ...types.Scout) *Scout {
	return &Scout{
		environmentInfo: types.EnvironmentInfo{
			CloudProvider: types.CloudProviderUnknown,
		},
		scouts: scouts,
	}
}

// Detect iterates through all provided scouts concurrently and returns the
// first cloud provider detected. Returns CloudProviderUnknown if no cloud
// provider is detected by any scout.
//
// Network errors during detection are treated as "not detected" and do not
// prevent other scouts from running.
func (s *Scout) Detect(ctx context.Context) (types.CloudProvider, error) {
	s.detectOnce.Do(func() {
		var wg sync.WaitGroup

		cancellableCtx, cancel := context.WithCancel(ctx)
		defer cancel()

		for _, scout := range s.scouts {
			wg.Add(1)

			go func(currentScout types.Scout) {
				defer wg.Done()

				detected, err := currentScout.Detect(cancellableCtx)
				if err != nil {
					_ = err // Ignore error, continue to next scout
					return
				}

				if detected == types.CloudProviderUnknown {
					return
				}

				// We have a match, cancel the context to stop the other scouts
				cancel()

				// Technically there could be a race condition here, but only if
				// two Scouts return a non-unknown cloud provider at the same
				// time, which shouldn't happen.
				s.environmentInfo.CloudProvider = detected
				s.scout = currentScout
			}(scout)
		}

		wg.Wait()

		s.scouts = nil
	})

	return s.environmentInfo.CloudProvider, s.environmentInfoError
}

// EnvironmentInfo attempts to retrieve environment information.
func (s *Scout) EnvironmentInfo(ctx context.Context) (*types.EnvironmentInfo, error) {
	s.environmentInfoOnce.Do(func() {
		if s.scout == nil {
			detected, err := s.Detect(ctx)
			if err != nil {
				s.environmentInfoError = err
				return
			}

			if detected != s.environmentInfo.CloudProvider {
				s.environmentInfoError = fmt.Errorf("detected cloud provider %s does not match expected cloud provider %s", detected, s.environmentInfo.CloudProvider)
				return
			}

		}

		if s.scout == nil {
			s.environmentInfoError = ErrNoCloudProviderDetected
			return
		}

		ei, err := s.scout.EnvironmentInfo(ctx)
		if err != nil {
			s.environmentInfoError = err
			return
		}

		s.environmentInfo = *ei
	})

	return &s.environmentInfo, s.environmentInfoError
}
