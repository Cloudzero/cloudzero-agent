//go:build unit
// +build unit

package config_test

import (
	"os"
	"testing"

	"github.com/cloudzero/cirrus-remote-write/app/config"
	"github.com/stretchr/testify/assert"
)

func TestMetricServiceConfigLoad(t *testing.T) {
	t.Run("successful load", func(t *testing.T) {
		var config config.MetricServiceConfig
		err := config.Load()
		assert.NoError(t, err)
	})

	t.Run("load with missing environment variables", func(t *testing.T) {
		// Unset any environment variables that might interfere with the test
		os.Clearenv()

		var config config.MetricServiceConfig
		err := config.Load()
		assert.NoError(t, err)
	})
}
