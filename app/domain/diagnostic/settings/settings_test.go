package settings_test

import (
	"encoding/base64"
	"os"
	"testing"

	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic/settings"

	"github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnit_Diagnostic_Settings_CheckOK(t *testing.T) {
	// create the settings
	wd, err := os.Getwd()
	require.NoError(t, err, "failed to get current directory")

	// settings locations
	validatorFile := wd + "/testdata/config-validator.yaml"
	webhookFile := wd + "/testdata/config-webhook.yaml"
	gatorFile := wd + "/testdata/config-gator.yaml"

	// create the provider
	provider := settings.NewProvider(t.Context(), []string{validatorFile}, []string{webhookFile}, []string{gatorFile})

	// run the check
	accessor := status.NewAccessor(&status.ClusterStatus{})
	err = provider.Check(t.Context(), nil, accessor)
	require.NoError(t, err, "failed to run the provider")

	// validate results
	accessor.ReadFromReport(func(cs *status.ClusterStatus) {
		assert.Len(t, cs.Checks, 1)
		for _, c := range cs.Checks {
			assert.True(t, c.Passing)
		}
		assert.NotEmpty(t, cs.ConfigValidatorBase64)
		assert.NotEmpty(t, cs.ConfigWebhookServerBase64)
		assert.NotEmpty(t, cs.ConfigAggregatorBase64)

		// ensure the configs are base64 compatible
		_, err = base64.StdEncoding.DecodeString(cs.ConfigValidatorBase64)
		assert.NoError(t, err, "invalid validator config")
		_, err = base64.StdEncoding.DecodeString(cs.ConfigWebhookServerBase64)
		assert.NoError(t, err, "invalid webhook config")
		_, err = base64.StdEncoding.DecodeString(cs.ConfigAggregatorBase64)
		assert.NoError(t, err, "invalid aggregator config")
	})
}
