package domain

import (
	"context"
	"sync"

	"github.com/cloudzero/cirrus-remote-write/app/config"
	"github.com/cloudzero/cirrus-remote-write/app/store"
	"github.com/cloudzero/cirrus-remote-write/app/types"
)

type secretsMonitor struct {
	ctx          context.Context
	cancel       context.CancelFunc
	settings     *config.Settings
	bus          types.Bus
	subscription *types.Subscription
	running      bool
	mu           sync.Mutex
}

func NewSecretMonitor(ctx context.Context, settings *config.Settings) (types.Runnable, error) {
	ctx, cancel := context.WithCancel(ctx)
	return &secretsMonitor{
		settings: settings,
		ctx:      ctx,
		bus:      store.NewBus(),
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

	fm, err := NewFileMonitor(s.ctx, s.bus, s.settings.Cloudzero.APIKeyPath)
	if err != nil {
		return err
	}

	go func() {
		fm.Start()
		defer fm.Close()

		s.subscription = s.bus.Subscribe()
		defer s.bus.Unsubscribe(s.subscription)

		for {
			select {
			case <-s.ctx.Done():
				s.running = false
				return
			case event := <-s.subscription.Events():
				if event.Type == FileChanged {
					s.settings.SetAPIKey()
				}
			}
		}
	}()
	s.running = true
	return nil
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
