package runner

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/diagnostic/catalog"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/status"
	"github.com/stretchr/testify/assert"
)

type mockProvider struct {
	Test func(ctx context.Context, client *http.Client, recorder status.Accessor) error
}

func (m *mockProvider) Check(ctx context.Context, client *http.Client, recorder status.Accessor) error {
	if m.Test != nil {
		return m.Test(ctx, client, recorder)
	}
	return nil
}

func TestRunner_Run_Error(t *testing.T) {
	cfg := &config.Settings{
		Deployment: config.Deployment{
			AccountID:   "test-account",
			Region:      "test-region",
			ClusterName: "test-cluster",
		},
	}

	reg := catalog.NewCatalog(context.Background(), cfg)
	stage := config.ContextStageInit

	r := NewRunner(cfg, reg, stage)
	engine := r.(*runner)

	// Add mock providers
	mockProvider1 := &mockProvider{}
	mockProvider2 := &mockProvider{}
	engine.AddPreStep(mockProvider1)
	engine.AddStep(mockProvider2)
	engine.AddPostStep(mockProvider1)

	// Simulate an error in one of the providers
	mockProvider2.Test = func(ctx context.Context, client *http.Client, recorder status.Accessor) error {
		return errors.New("provider error")
	}

	recorder, err := r.Run(context.Background())

	assert.Error(t, err)
	assert.NotNil(t, recorder)
}

func TestRunner_Run(t *testing.T) {
	cfg := &config.Settings{
		Deployment: config.Deployment{
			AccountID:   "test-account",
			Region:      "test-region",
			ClusterName: "test-cluster",
		},
	}

	reg := catalog.NewCatalog(context.Background(), cfg)
	stage := config.ContextStageInit

	r := NewRunner(cfg, reg, stage)
	engine := r.(*runner)

	// Add mock providers
	mockProvider1 := &mockProvider{}
	mockProvider2 := &mockProvider{}
	engine.AddPreStep(mockProvider1)
	engine.AddStep(mockProvider2)
	engine.AddPostStep(mockProvider1)

	recorder, err := r.Run(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, recorder)
}
