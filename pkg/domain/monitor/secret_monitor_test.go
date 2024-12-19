// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package monitor_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/domain/monitor"
)

type MockFileMonitor struct {
	mock.Mock
}

func (m *MockFileMonitor) Start() {
	m.Called()
}

func (m *MockFileMonitor) Close() {
	m.Called()
}

func TestSecretsMonitor_Start(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// make a temp file
	file, err := os.CreateTemp(t.TempDir(), "apikey.txt")
	assert.NoError(t, err)
	defer func() {
		_ = file.Close()
		_ = os.Remove(file.Name())
	}()

	// write initial value as foo
	_, err = file.WriteString("foo")
	assert.NoError(t, err)

	settings := &config.Settings{
		APIKeyPath: file.Name(),
	}

	err = settings.SetAPIKey()
	assert.NoError(t, err)
	assert.Equal(t, "foo", settings.GetAPIKey())

	m := monitor.NewSecretMonitor(ctx, settings)
	defer m.Shutdown()

	// update the interval to cause faster refresh
	monitor.DefaultRefreshInterval = 100 * time.Millisecond

	err = m.Start()
	assert.NoError(t, err)

	// update file content to bar
	_, err = file.Seek(0, 0)
	assert.NoError(t, err)
	_, err = file.WriteString("bar")
	assert.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	// validate our settings has the right value
	assert.Equal(t, "bar", settings.GetAPIKey())
}
