package catalog_test

import (
	"context"
	"testing"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/diagnostic/catalog"
	"github.com/stretchr/testify/assert"
)

func TestRegistry_Get(t *testing.T) {
	ctx := context.Background()
	c := &config.Settings{}
	r := catalog.NewCatalog(ctx, c)

	// Test getting providers with existing IDs
	providers := r.Get(config.DiagnosticAPIKey, config.DiagnosticK8sVersion)
	assert.Len(t, providers, 2)

	// Test getting providers with non-existing IDs
	providers = r.Get("non-existing-id")
	assert.Empty(t, providers)

	// Test getting providers with empty IDs
	providers = r.Get()
	assert.Empty(t, providers)
}

func TestRegistry_Has(t *testing.T) {
	ctx := context.Background()
	c := &config.Settings{}
	r := catalog.NewCatalog(ctx, c)

	// Test checking for existing ID
	has := r.Has(config.DiagnosticAPIKey)
	assert.True(t, has)

	// Test checking for non-existing ID
	has = r.Has("non-existing-id")
	assert.False(t, has)
}

func TestRegistry_List(t *testing.T) {
	ctx := context.Background()
	c := &config.Settings{}
	r := catalog.NewCatalog(ctx, c)

	// Test listing providers
	providers := r.List()
	assert.Len(t, providers, 6)
}
