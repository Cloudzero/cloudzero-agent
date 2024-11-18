//go:build unit
// +build unit

package domain_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/cloudzero/cirrus-remote-write/app/config"
	"github.com/cloudzero/cirrus-remote-write/app/domain"
	"github.com/cloudzero/cirrus-remote-write/app/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockSecretBus struct {
	mock.Mock
}

func (m *MockSecretBus) Subscribe() *types.Subscription {
	args := m.Called()
	return args.Get(0).(*types.Subscription)
}

func (m *MockSecretBus) Unsubscribe(sub *types.Subscription) {
	m.Called(sub)
}

type MockFileMonitor struct {
	mock.Mock
}

func (m *MockFileMonitor) Start() {
	m.Called()
}

func (m *MockFileMonitor) Close() {
	m.Called()
}

func TestSecretsMonitor_Run(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// make a temp file
	file, err := os.CreateTemp(t.TempDir(), "apikey.txt")
	assert.NoError(t, err)
	defer os.Remove(file.Name())

	// write initial value as foo
	_, err = file.WriteString("foo")
	assert.NoError(t, err)

	settings := &config.Settings{
		Cloudzero: config.Cloudzero{
			APIKeyPath: file.Name(),
		},
	}

	monitor, err := domain.NewSecretMonitor(ctx, settings)
	assert.NoError(t, err)
	defer monitor.Shutdown()

	err = monitor.Run()
	assert.NoError(t, err)

	// update file content to bar
	_, err = file.Seek(0, 0)
	assert.NoError(t, err)
	_, err = file.WriteString("bar")
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// validate our settings has the right value
	assert.Equal(t, "bar", settings.GetAPIKey())
}
