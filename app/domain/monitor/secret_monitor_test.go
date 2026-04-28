// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package monitor_test

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/cloudzero/cloudzero-agent/app/domain/monitor"
)

type MockFileMonitor struct {
	mock.Mock
}

func (m *MockFileMonitor) Run() {
	m.Called()
}

func (m *MockFileMonitor) Close() {
	m.Called()
}

// fileBackedAPIKey is a minimal monitor.MonitoredAPIKey that reads from a
// file on disk, used here to exercise the secrets monitor's refresh loop.
type fileBackedAPIKey struct {
	path string
	mu   sync.Mutex
	key  string
}

func (f *fileBackedAPIKey) GetAPIKey() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.key
}

func (f *fileBackedAPIKey) SetAPIKey() error {
	data, err := os.ReadFile(f.path)
	if err != nil {
		return err
	}
	f.mu.Lock()
	f.key = strings.TrimSpace(string(data))
	f.mu.Unlock()
	return nil
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

	settings := &fileBackedAPIKey{path: file.Name()}

	err = settings.SetAPIKey()
	assert.NoError(t, err)
	assert.Equal(t, "foo", settings.GetAPIKey())

	m := monitor.NewSecretMonitor(ctx, settings)
	defer m.Shutdown()

	// update the interval to cause faster refresh
	monitor.DefaultRefreshInterval = 100 * time.Millisecond

	err = m.Run()
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
