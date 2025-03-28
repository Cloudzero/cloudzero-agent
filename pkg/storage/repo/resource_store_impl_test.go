// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package repo_test

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/cloudzero/cloudzero-insights-controller/app/types/mocks"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage/repo"
)

// setupTestRepo initializes a new in-memory repository with the provided TimeProvider.
func setupTestRepo(t *testing.T, clock types.TimeProvider) types.ResourceStore {
	repo, err := repo.NewInMemoryResourceRepository(clock)
	require.NoError(t, err)
	require.NotNil(t, repo)

	// make sure nothing sticks arround since last test
	repo.DeleteAll(context.Background())
	return repo
}

// createTestResource creates a resource in the repository and returns the created resource.
func createTestResource(t *testing.T, repo types.ResourceStore, ctx context.Context, resource types.ResourceTags) types.ResourceTags {
	err := repo.Create(ctx, &resource)
	require.NoError(t, err)
	assert.NotEmpty(t, resource.ID)
	assert.NotEmpty(t, resource.RecordCreated)
	return resource
}

func TestResourceRepoImpl_Create(t *testing.T) {
	// Initialize MockClock with a fixed current time
	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	repo := setupTestRepo(t, mockClock)
	ctx := context.Background()

	defaultNamespace := "default"

	// Create test records
	records := []types.ResourceTags{
		{Type: config.Deployment, Name: "Deployment", Namespace: nil},
		{Type: config.StatefulSet, Name: "StatefulSet", Namespace: nil},
		{Type: config.Pod, Name: "Pod", Namespace: nil},
		{Type: config.Node, Name: "Node", Namespace: nil},
		{Type: config.Namespace, Name: "Namespace", Namespace: nil},
		{Type: config.Job, Name: "Job", Namespace: nil},
		{Type: config.CronJob, Name: "CronJob", Namespace: nil},
		{Type: config.DaemonSet, Name: "DaemonSet", Namespace: nil},
		{Type: config.Deployment, Name: "Deployment", Namespace: &defaultNamespace},
		{Type: config.StatefulSet, Name: "StatefulSet", Namespace: &defaultNamespace},
		{Type: config.Pod, Name: "Pod", Namespace: &defaultNamespace},
		{Type: config.Node, Name: "Node", Namespace: &defaultNamespace},
		{Type: config.Namespace, Name: "Namespace", Namespace: &defaultNamespace},
		{Type: config.Job, Name: "Job", Namespace: &defaultNamespace},
		{Type: config.CronJob, Name: "CronJob", Namespace: &defaultNamespace},
		{Type: config.DaemonSet, Name: "DaemonSet", Namespace: &defaultNamespace},
	}

	for _, record := range records {
		createdRecord := createTestResource(t, repo, ctx, record)
		assert.NotEmpty(t, createdRecord.ID)
		assert.Equal(t, initialTime, createdRecord.RecordCreated)
		assert.Equal(t, initialTime, createdRecord.RecordUpdated)
	}
}

func TestResourceRepoImpl_Update(t *testing.T) {
	// Initialize MockClock with a fixed current time
	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	repo := setupTestRepo(t, mockClock)
	ctx := context.Background()
	// Create a test resource
	resource := types.ResourceTags{
		Type:          config.Job,
		Name:          "TestJob",
		Namespace:     nil,
		RecordCreated: mockClock.GetCurrentTime(),
		RecordUpdated: mockClock.GetCurrentTime(),
	}
	createdResource := createTestResource(t, repo, ctx, resource)

	// Test updating the existing resource
	t.Run("Update RecordUpdated", func(t *testing.T) {
		// Advance the mock clock to simulate time passage
		newTime := initialTime.Add(2 * time.Hour)
		mockClock.SetCurrentTime(newTime)

		err := repo.Update(ctx, &createdResource)
		require.NoError(t, err)

		// Retrieve the updated resource
		got, err := repo.Get(ctx, createdResource.ID)
		require.NoError(t, err)
		assert.Equal(t, initialTime, got.RecordCreated)
		assert.Equal(t, newTime, got.RecordUpdated)
		assert.Nil(t, got.SentAt)
	})

	t.Run("Update SentAt", func(t *testing.T) {
		// Advance the mock clock to simulate time passage
		newTime := initialTime.Add(2 * time.Hour)
		mockClock.SetCurrentTime(newTime)
		createdResource.SentAt = &newTime

		err := repo.Update(ctx, &createdResource)
		require.NoError(t, err)

		// Retrieve the updated resource
		got, err := repo.Get(ctx, createdResource.ID)
		require.NoError(t, err)
		assert.Equal(t, initialTime, got.RecordCreated)
		assert.Equal(t, newTime, got.RecordUpdated)
		assert.Equal(t, newTime, *got.SentAt)
	})

	t.Run("Update Labels, Annotations, and MetricsLabels", func(t *testing.T) {
		// Advance the mock clock to simulate time passage
		newTime := initialTime.Add(3 * time.Hour)
		mockClock.SetCurrentTime(newTime)

		createdResource.Labels = &config.MetricLabelTags{"env": "production"}
		createdResource.Annotations = &config.MetricLabelTags{"owner": "team-a"}
		createdResource.MetricLabels = &config.MetricLabels{"app": "my-app"}

		err := repo.Update(ctx, &createdResource)
		require.NoError(t, err)

		// Now find it
		got, err := repo.Get(ctx, createdResource.ID)
		require.NoError(t, err)
		assert.Equal(t, initialTime, got.RecordCreated)
		assert.Equal(t, newTime, got.RecordUpdated)
		assert.Equal(t, config.MetricLabelTags{"env": "production"}, *got.Labels)
		assert.Equal(t, config.MetricLabelTags{"owner": "team-a"}, *got.Annotations)
		assert.Equal(t, config.MetricLabels{"app": "my-app"}, *got.MetricLabels)
	})

	t.Run("Remove Annotations", func(t *testing.T) {
		// Advance the mock clock to simulate time passage
		newTime := initialTime.Add(4 * time.Hour)
		mockClock.SetCurrentTime(newTime)

		createdResource.Annotations = nil

		err := repo.Update(ctx, &createdResource)
		require.NoError(t, err)

		// Now find it
		got, err := repo.Get(ctx, createdResource.ID)
		require.NoError(t, err)
		assert.Equal(t, initialTime, got.RecordCreated)
		assert.Equal(t, newTime, got.RecordUpdated)
		assert.Nil(t, got.Annotations)
	})
}

func TestResourceRepoImpl_Get(t *testing.T) {
	// Initialize MockClock with a fixed current time
	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	repo := setupTestRepo(t, mockClock)
	ctx := context.Background()

	// Create a test resource
	resource := types.ResourceTags{
		Type:          config.Job,
		Name:          "TestJob",
		Namespace:     nil,
		RecordCreated: mockClock.GetCurrentTime(),
	}
	createdResource := createTestResource(t, repo, ctx, resource)

	t.Run("Get existing resource", func(t *testing.T) {
		got, err := repo.Get(ctx, createdResource.ID)
		require.NoError(t, err)
		assert.Equal(t, createdResource.ID, got.ID)
		assert.Equal(t, createdResource.Type, got.Type)
		assert.Equal(t, createdResource.Name, got.Name)
		assert.Equal(t, createdResource.Namespace, got.Namespace)
		assert.Equal(t, createdResource.RecordCreated, got.RecordCreated)
		assert.Equal(t, createdResource.RecordCreated, got.RecordUpdated)
		assert.Nil(t, got.SentAt)
	})

	t.Run("Get non-existing resource", func(t *testing.T) {
		_, err := repo.Get(ctx, "non-existing-id")
		assert.Error(t, err)
		assert.Equal(t, types.ErrNotFound, err)
	})
}

func TestResourceRepoImpl_Delete(t *testing.T) {
	// Initialize MockClock with a fixed current time
	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	repo := setupTestRepo(t, mockClock)
	ctx := context.Background()

	// Create a test resource
	resource := types.ResourceTags{
		Type:          config.Job,
		Name:          "TestJob",
		Namespace:     nil,
		RecordCreated: mockClock.GetCurrentTime(),
	}
	createdResource := createTestResource(t, repo, ctx, resource)

	t.Run("Delete existing resource", func(t *testing.T) {
		err := repo.Delete(ctx, createdResource.ID)
		require.NoError(t, err)

		// Try to retrieve the deleted resource
		_, err = repo.Get(ctx, createdResource.ID)
		assert.Error(t, err)
		assert.Equal(t, types.ErrNotFound, err)
	})

	t.Run("Delete non-existing resource", func(t *testing.T) {
		err := repo.Delete(ctx, "non-existing-id")
		assert.NoError(t, err)
	})
}

func TestResourceRepoImpl_FindFirstBy(t *testing.T) {
	// Initialize MockClock with a fixed current time
	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	repo := setupTestRepo(t, mockClock)
	ctx := context.Background()

	// Create test resources
	resources := []types.ResourceTags{
		{Type: config.Deployment, Name: "Deployment1", Namespace: nil, RecordCreated: mockClock.GetCurrentTime()},
		{Type: config.Deployment, Name: "Deployment2", Namespace: nil, RecordCreated: mockClock.GetCurrentTime()},
		{Type: config.Pod, Name: "Pod1", Namespace: nil, RecordCreated: mockClock.GetCurrentTime()},
	}

	for _, resource := range resources {
		createTestResource(t, repo, ctx, resource)
	}

	t.Run("FindFirstBy existing resource", func(t *testing.T) {
		got, err := repo.FindFirstBy(ctx, "type = ? AND name = ?", config.Deployment, "Deployment1")
		require.NoError(t, err)
		assert.Equal(t, config.Deployment, got.Type)
		assert.Equal(t, "Deployment1", got.Name)
		assert.Nil(t, got.Namespace)
		assert.Equal(t, initialTime, got.RecordCreated)
		assert.Equal(t, initialTime, got.RecordUpdated)
		assert.Nil(t, got.SentAt)
	})

	t.Run("FindFirstBy non-existing resource", func(t *testing.T) {
		_, err := repo.FindFirstBy(ctx, "type = ? AND name = ?", config.Namespace, "NonExistingNamespace")
		assert.Error(t, err)
		assert.Equal(t, types.ErrNotFound, err)
	})
}

func TestResourceRepoImpl_FindAllBy(t *testing.T) {
	// Initialize MockClock with a fixed current time
	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	repo := setupTestRepo(t, mockClock)
	ctx := context.Background()

	// Create test resources
	resources := []types.ResourceTags{
		{Type: config.Deployment, Name: "TestResourceRepoImpl_FindAllBy1", Namespace: nil, RecordCreated: mockClock.GetCurrentTime()},
		{Type: config.Deployment, Name: "TestResourceRepoImpl_FindAllBy2", Namespace: nil, RecordCreated: mockClock.GetCurrentTime()},
		{Type: config.Pod, Name: "Pod1", Namespace: nil, RecordCreated: mockClock.GetCurrentTime()},
	}

	for _, resource := range resources {
		createTestResource(t, repo, ctx, resource)
	}

	t.Run("FindAllBy existing resources", func(t *testing.T) {
		got, err := repo.FindAllBy(ctx, "type = ? AND name like ?", config.Deployment, "TestResourceRepoImpl_FindAllBy%")
		require.NoError(t, err)
		assert.Len(t, got, 2)

		for _, resource := range got {
			assert.Equal(t, config.Deployment, resource.Type)
			assert.Nil(t, resource.Namespace)
			assert.Equal(t, initialTime, resource.RecordCreated)
			assert.Equal(t, initialTime, resource.RecordUpdated)
			assert.Nil(t, resource.SentAt)
		}
	})

	t.Run("FindAllBy non-existing resources", func(t *testing.T) {
		got, err := repo.FindAllBy(ctx, "type = ?", "undefined")
		require.NoError(t, err)
		assert.Len(t, got, 0)
	})

	t.Run("FindAllBy records with not sent but updated date greater than", func(t *testing.T) {
		// Advance the mock clock to simulate time passage
		newTime := initialTime.Add(2 * time.Hour)
		mockClock.SetCurrentTime(newTime)

		// Create test resources with updated dates
		resources := []types.ResourceTags{
			{Type: config.Deployment, Name: "Deployment3", Namespace: nil},
			{Type: config.Deployment, Name: "Deployment4", Namespace: nil},
		}

		for _, resource := range resources {
			createTestResource(t, repo, ctx, resource)
		}

		// Find records with no sent date but updated date greater than initialTime
		got, err := repo.FindAllBy(ctx, "sent_at IS NULL AND record_updated > ?", initialTime)
		require.NoError(t, err)
		assert.Len(t, got, 2)

		for _, resource := range got {
			assert.Equal(t, config.Deployment, resource.Type)
			assert.Nil(t, resource.Namespace)
			assert.Equal(t, newTime, resource.RecordCreated)
			assert.Equal(t, newTime, resource.RecordUpdated)
			assert.Nil(t, resource.SentAt)
		}
	})

	t.Run("Update SentAt and find not sent resources", func(t *testing.T) {
		// Advance the mock clock to simulate time passage
		newTime := initialTime.Add(10 * time.Hour)
		mockClock.SetCurrentTime(newTime)

		// Create test resources
		notSent := createTestResource(
			t, repo, ctx,
			types.ResourceTags{
				Type: config.Deployment, Name: "Deployment5", Namespace: nil,
			},
		)

		sentResource := createTestResource(
			t, repo, ctx,
			types.ResourceTags{
				Type: config.Deployment, Name: "Deployment6", Namespace: nil,
			},
		)
		advTime := initialTime.Add(11 * time.Hour)
		mockClock.SetCurrentTime(advTime)
		// make it sent
		sentResource.SentAt = &advTime
		err := repo.Update(ctx, &sentResource)
		require.NoError(t, err)

		// Find records with no sent date
		got, err := repo.FindAllBy(ctx, "sent_at IS NULL AND record_updated between ? AND ?", newTime, advTime)
		require.NoError(t, err)
		assert.Len(t, got, 1)
		assert.Equal(t, notSent.ID, got[0].ID)
	})
}

func TestResourceRepoImpl_CreateWithTransaction(t *testing.T) {
	// Initialize MockClock with a fixed current time
	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	repo := setupTestRepo(t, mockClock)
	ctx := context.Background()

	// Create a test resource within a transaction
	t.Run("Create resource within a transaction", func(t *testing.T) {
		var id string
		err := repo.Tx(ctx, func(ctxTx context.Context) error {
			resource := types.ResourceTags{
				Type:          config.Job,
				Name:          "TestJobWithTransaction",
				Namespace:     nil,
				RecordCreated: mockClock.GetCurrentTime(),
				RecordUpdated: mockClock.GetCurrentTime(),
			}
			err := repo.Create(ctxTx, &resource)
			require.NoError(t, err)
			assert.NotEmpty(t, resource.ID)
			assert.Equal(t, initialTime, resource.RecordCreated)
			assert.Equal(t, initialTime, resource.RecordUpdated)
			id = resource.ID
			return nil
		})
		require.NoError(t, err)
		require.NotEmpty(t, id)

		// Verify the resource was created
		got, err := repo.Get(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, config.Job, got.Type)
	})

	t.Run("Nothing created when error occurs in transaction", func(t *testing.T) {
		err := repo.Tx(ctx, func(ctxTx context.Context) error {
			resource := types.ResourceTags{
				Type:          config.Job,
				Name:          "JobA",
				Namespace:     nil,
				RecordCreated: mockClock.GetCurrentTime(),
				RecordUpdated: mockClock.GetCurrentTime(),
			}
			err := repo.Create(ctxTx, &resource)
			require.NoError(t, err)
			assert.NotEmpty(t, resource.ID)
			assert.Equal(t, initialTime, resource.RecordCreated)
			assert.Equal(t, initialTime, resource.RecordUpdated)
			return fmt.Errorf("fake error")
		})
		require.Error(t, err)
		require.Equal(t, "fake error", err.Error())

		// Verify the resource was not created
		found, err := repo.FindFirstBy(ctx, "type = ? AND name = ?", config.Job, "JobA")
		require.Error(t, err)
		assert.Equal(t, types.ErrNotFound, err)
		require.Nil(t, found)
	})

	t.Run("Find works within transaction", func(t *testing.T) {
		err := repo.Tx(ctx, func(ctxTx context.Context) error {
			resource := types.ResourceTags{
				Type:          config.Job,
				Name:          "JobB",
				Namespace:     nil,
				RecordCreated: mockClock.GetCurrentTime(),
				RecordUpdated: mockClock.GetCurrentTime(),
			}
			err := repo.Create(ctxTx, &resource)
			require.NoError(t, err)
			assert.NotEmpty(t, resource.ID)
			assert.Equal(t, initialTime, resource.RecordCreated)
			assert.Equal(t, initialTime, resource.RecordUpdated)

			// Verify the resource was created
			got, err := repo.FindFirstBy(ctxTx, "type = ? AND name = ?", config.Job, "JobB")
			require.NoError(t, err)
			assert.Equal(t, config.Job, got.Type)
			return nil
		})
		require.NoError(t, err)
	})
}

// TestResourceRepoImpl_ConcurrentReadWrite tests concurrent read and write operations on the repository,
// specifically ensuring that resources can be retrieved by both name and type.
func TestResourceRepoImpl_ConcurrentReadWrite(t *testing.T) {
	// Initialize MockClock with a fixed current time
	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	repo := setupTestRepo(t, mockClock)
	ctx := context.Background()

	var wg sync.WaitGroup
	numWrites := 1000
	for i := 0; i < numWrites; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			// Generate a random int64 between 0 and 100 inclusive to allow a randomization for currency
			randomNumber := rand.Int63n(11) // 0 <= randomNumber <= 10
			time.Sleep(time.Duration(randomNumber) * time.Millisecond)

			resourceName := fmt.Sprintf("Resource_%d", i)
			resource := types.ResourceTags{
				Type:      config.CronJob,
				Name:      resourceName,
				Namespace: nil,
			}
			err := repo.Create(ctx, &resource)
			assert.NoError(t, err)

			found, err := repo.FindAllBy(ctx, "type = ? AND name = ?", config.CronJob, resourceName)
			assert.NoError(t, err)
			assert.Len(t, found, 1)
		}(i)
	}
	wg.Wait()
}
