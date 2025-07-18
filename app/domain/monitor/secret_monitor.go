// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package monitor provides functionality to manage and reload secrets dynamically.
// This package is designed to enhance the security and flexibility of secret management by
// allowing dynamic reloading and rotation of secrets. It supports various methods
// for triggering secret reloads, such as file system changes, SIGHUP signals, and
// periodic intervals.
//
// The main component of this package is the secretsMonitor struct, which manages the secret
// configuration and ensures that the latest secrets are always used. The secretsMonitor
// can be customized using functional options to fit different use cases.
//
// This package is valuable for applications that require robust and flexible secret management,
// ensuring that the secret configuration stays up-to-date with the latest secrets
// without requiring application restarts.
package monitor

import (
	"context"
	"crypto/sha256"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-agent/app/types"
)

var DefaultRefreshInterval = 1 * time.Minute

// MonitoredAPIKey is an interface that allows the API key to be monitored.
type MonitoredAPIKey interface {
	// GetAPIKey returns the current API key.
	GetAPIKey() string
	// SetAPIKey refreshes the API key.
	SetAPIKey() error
}

type secretsMonitor struct {
	settings    MonitoredAPIKey
	originalCtx context.Context
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.Mutex
	lastHash    [32]byte
	running     bool
	done        chan struct{}
}

func NewSecretMonitor(ctx context.Context, settings MonitoredAPIKey) types.Runnable {
	newCtx, cancel := context.WithCancel(ctx)
	return &secretsMonitor{
		settings:    settings,
		originalCtx: ctx,
		ctx:         newCtx,
		cancel:      cancel,
		done:        make(chan struct{}),
	}
}

// Run implements types.Runnable.
func (s *secretsMonitor) Run() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}

	ticker := time.NewTicker(DefaultRefreshInterval)
	go func() {
		defer ticker.Stop()
		defer close(s.done)
		defer func() {
			if r := recover(); r != nil {
				log.Info().Interface("panic", r).Msg("Recovered from panic in secret monitor")
			}
		}()

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
					log.Info().Str("secret", redactSecret(newSecret)).Msg("discovered new secret")
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
		return secret[:2] + "***"
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
	<-s.done
	s.reset()
	return nil
}

func (s *secretsMonitor) reset() {
	s.running = false
	ctx, cancel := context.WithCancel(s.originalCtx)
	s.ctx = ctx
	s.cancel = cancel
	s.done = make(chan struct{})
}

func (s *secretsMonitor) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
