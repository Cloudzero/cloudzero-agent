package namespace_test

import (
	"context"
	"os"
	"testing"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic/k8s/namespace"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/stretchr/testify/assert"
)

func TestUnit_Diagnostic_K8s_Namespace_CheckOK(t *testing.T) {
	cfg := &config.Settings{}

	// set the namespace for the test
	expectedNs := "test-ns"
	os.Setenv("NAMESPACE", expectedNs)

	provider := namespace.NewProvider(context.Background(), cfg)

	accessor := status.NewAccessor(&status.ClusterStatus{})
	err := provider.Check(t.Context(), nil, accessor)
	assert.NoError(t, err)

	accessor.ReadFromReport(func(cs *status.ClusterStatus) {
		assert.Len(t, cs.Checks, 1)
		for _, c := range cs.Checks {
			assert.True(t, c.Passing)
		}
		assert.Equal(t, expectedNs, cs.Namespace)
	})
}
