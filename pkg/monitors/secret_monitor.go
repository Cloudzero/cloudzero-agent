// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package monitors

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/types"
)

var DefaultRefreshInterval = 1 * time.Minute

type secretsMonitor struct {
	ctx      context.Context
	cancel   context.CancelFunc
	settings *config.Settings
	running  bool
	mu       sync.Mutex
	lastHash [32]byte
}

func NewSecretMonitor(ctx context.Context, settings *config.Settings) (types.Runnable, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &secretsMonitor{
		settings: settings,
		ctx:      ctx,
		cancel:   cancel,
	}, nil
}

// Run implements types.Runnable.
func (s *secretsMonitor) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}

	ticker := time.NewTicker(DefaultRefreshInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				s.running = false
				return
			case <-ticker.C:
				_ = s.settings.SetAPIKey()
				newSecret := s.settings.GetAPIKey()
				newHash := sha256.Sum256([]byte(newSecret))
				if newHash != s.lastHash {
					log.Info().Msgf("discovered new secret %s", redactSecret(newSecret))
					s.lastHash = newHash
				}
			}
		}
	}()
	s.running = true
	return nil
}

func redactSecret(secret string) string {
	if len(secret) > 2 {
		return fmt.Sprintf("%s***", secret[:2])
	}
	return "*****"
}

// Shutdown implements types.Runnable.
func (s *secretsMonitor) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return nil
	}
	s.cancel()
	return nil
}
