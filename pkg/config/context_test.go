package config_test

import (
	"strings"
	"testing"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestIsValidStage(t *testing.T) {
	assert.True(t, config.IsValidStage(config.ContextStageInit))
	assert.True(t, config.IsValidStage(config.ContextStageStart))
	assert.True(t, config.IsValidStage(config.ContextStageStop))

	assert.True(t, config.IsValidStage(strings.ToUpper(config.ContextStageInit)))
	assert.True(t, config.IsValidStage(strings.ToUpper(config.ContextStageStart)))
	assert.True(t, config.IsValidStage(strings.ToUpper(config.ContextStageStop)))

	assert.False(t, config.IsValidStage("bogus"))
}

func TestContext_Validate(t *testing.T) {
	tcases := []struct {
		name     string
		input    *config.Context
		expected *config.Context
	}{
		{
			name: "Valid context",
			input: &config.Context{
				Stage: config.ContextStageInit,
			},
			expected: &config.Context{
				Stage: config.ContextStageInit,
			},
		},
	}
	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NoError(t, tc.input.Validate())
			assert.Equal(t, tc.expected, tc.input)
		})
	}
}
