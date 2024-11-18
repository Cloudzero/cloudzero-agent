//go:build unit
// +build unit

package domain_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudzero/cirrus-remote-write/app/domain"
	"github.com/cloudzero/cirrus-remote-write/app/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockBus struct {
	mock.Mock
}

func (m *MockBus) Subscribe() *types.Subscription {
	args := m.Called()
	return args.Get(0).(*types.Subscription)
}

func (m *MockBus) Unsubscribe(sub *types.Subscription) error {
	args := m.Called(sub)
	return args.Error(0)
}

func (m *MockBus) Publish(event types.Event) {
	m.Called(event)
}

func TestSecretMonitor_Start_FileNotExist(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockBus := new(MockBus)

	monitor, err := domain.NewFileMonitor(ctx, mockBus, "nonexistent.txt")
	assert.Error(t, err)
	assert.Nil(t, monitor)
}

func TestSecretMonitor_Start(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "secret.txt")
	err := os.WriteFile(tempFile, []byte("initial content"), 0644)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockBus := new(MockBus)
	mockBus.On("Publish", types.Event{
		Type: domain.FileChanged,
		Value: types.FileChanged{
			Name: tempFile,
		},
	}).Return()

	monitor, err := domain.NewFileMonitor(ctx, mockBus, tempFile)
	assert.NoError(t, err)
	assert.NotNil(t, monitor)

	monitor.Start()
	defer monitor.Close()

	// Simulate a file write event
	err = os.WriteFile(tempFile, []byte("new content"), 0644)
	assert.NoError(t, err)

	// Give some time for the event to be processed
	time.Sleep(100 * time.Millisecond)

	mockBus.AssertCalled(t, "Publish", types.Event{
		Type: domain.FileChanged,
		Value: types.FileChanged{
			Name: tempFile,
		},
	})
}

func TestSecretMonitor_Start_FileRename(t *testing.T) {
	// For example "mv /tmp/file /tmp/rename" will emit:
	//
	//   Event{Op: Rename, Name: "/tmp/file"}
	//   Event{Op: Create, Name: "/tmp/rename"}

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "secret.txt")
	newTempFile := filepath.Join(tempDir, "renamed_secret.txt")
	err := os.WriteFile(tempFile, []byte("initial content"), 0644)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockBus := new(MockBus)

	mockBus.On("Publish", types.Event{
		Type: domain.FileRenamed,
		Value: types.FileRenamed{
			Name: tempFile,
		},
	}).Return()

	monitor, err := domain.NewFileMonitor(ctx, mockBus, tempFile)
	assert.NoError(t, err)
	assert.NotNil(t, monitor)

	monitor.Start()
	defer monitor.Close()

	// Rename the file to simulate a rename event
	err = os.Rename(tempFile, newTempFile)
	assert.NoError(t, err)

	// Give some time for the event to be processed
	time.Sleep(100 * time.Millisecond)

	mockBus.AssertCalled(t, "Publish", types.Event{
		Type: domain.FileRenamed,
		Value: types.FileRenamed{
			Name: tempFile,
		},
	})
}

func TestSecretMonitor_Start_FileDeleted(t *testing.T) {
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "secret.txt")
	err := os.WriteFile(tempFile, []byte("initial content"), 0644)
	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mockBus := new(MockBus)
	mockBus.On("Publish", types.Event{
		Type: domain.FileDeleted,
		Value: types.FileDeleted{
			Name: tempFile,
		},
	}).Return()

	monitor, err := domain.NewFileMonitor(ctx, mockBus, tempFile)
	assert.NoError(t, err)
	assert.NotNil(t, monitor)

	monitor.Start()
	defer monitor.Close()

	// Delete the file to simulate an error
	err = os.Remove(tempFile)
	assert.NoError(t, err)

	// Give some time for the event to be processed
	time.Sleep(100 * time.Millisecond)

	mockBus.AssertCalled(t, "Publish", types.Event{
		Type: domain.FileDeleted,
		Value: types.FileDeleted{
			Name: tempFile,
		},
	})
}

func TestSecretMonitor_Start_Directory(t *testing.T) {
	tempDir := t.TempDir()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	newTempFile := filepath.Join(tempDir, "new_secret.txt")
	mockBus := new(MockBus)
	mockBus.On("Publish", types.Event{
		Type: domain.FileCreated,
		Value: types.FileCreated{
			Name: newTempFile,
		},
	}).Return()

	monitor, err := domain.NewFileMonitor(ctx, mockBus, tempDir)
	assert.NoError(t, err)
	assert.NotNil(t, monitor)

	monitor.Start()
	defer monitor.Close()

	// Create a new file in the directory to simulate a create event

	err = os.WriteFile(newTempFile, []byte("new content"), 0644)
	assert.NoError(t, err)

	// Give some time for the event to be processed
	time.Sleep(100 * time.Millisecond)

	mockBus.AssertCalled(t, "Publish", types.Event{
		Type: domain.FileCreated,
		Value: types.FileCreated{
			Name: newTempFile,
		},
	})
}
