package domain

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
)

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
func (s *secretsMonitor) Run() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}

	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				s.running = false
				return
			case <-ticker.C:
				s.settings.SetAPIKey()
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
