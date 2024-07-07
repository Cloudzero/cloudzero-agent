package config_test

import (
	"testing"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestVersions_Validate(t *testing.T) {
	versions := &config.Versions{
		ChartVersion: "1.0.0",
		AgentVersion: "2.0.0",
	}

	err := versions.Validate()
	assert.NoError(t, err)
}
