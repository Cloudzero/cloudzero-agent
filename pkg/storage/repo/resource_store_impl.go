// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package repo provides implementations for resource repository interfaces.
// This package includes implementations for repositories that can be extended
// to fit specific use cases. It supports transaction management and context-based
// database operations.
package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage/core"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage/sqlite"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
)

// remoteWriteStatsOnce is used to ensure that the initialization of remote write statistics
// happens only once. This is useful to avoid race conditions and ensure thread-safe initialization
// of metrics or other related resources.
var (
	remoteWriteStatsOnce sync.Once
	StorageWriteFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "storage_write_failure_total",
			Help: "Total number of storage write failures.",
		},
		[]string{"resource_type", "namespace", "resource_name", "action"},
	)
)

// NewInMemoryResourceRepository creates a new in-memory resource repository.
func NewInMemoryResourceRepository(clock types.TimeProvider) (types.ResourceStore, error) {
	remoteWriteStatsOnce.Do(func() {
		prometheus.MustRegister(
			StorageWriteFailures,
		)
	})

	db, err := sqlite.NewSQLiteDriver(sqlite.MemorySharedCached)
	if err != nil {
		return nil, core.TranslateError(err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, core.TranslateError(err)
	}
	// Special configuration needed for in-memory, concurrent access, and shared cache
	// REFS:
	// https://github.com/mattn/go-sqlite3/issues/204
	// https://gorm.io/docs/connecting_to_the_database.html#Connection-Pool
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	// 5000 milliseconds
	if _, err := sqlDB.Exec("PRAGMA busy_timeout = 5000;"); err != nil {
		return nil, core.TranslateError(err)
	}

	return NewResourceRepository(clock, db)
}

// NewResourceRepository creates a new resource repository with the given clock and database connection.
func NewResourceRepository(clock types.TimeProvider, db *gorm.DB) (types.ResourceStore, error) {
	if err := db.AutoMigrate(&types.ResourceTags{}); err != nil {
		return nil, core.TranslateError(err)
	}

	return &resourceRepoImpl{
		clock: clock,
		BaseRepoImpl: core.NewBaseRepoImpl(
			db, &resourceRepoImpl{},
		),
	}, nil
}

// resourceRepoImpl is the concrete implementation of the ResourceRepository interface.
type resourceRepoImpl struct {
	core.BaseRepoImpl
	clock types.TimeProvider
}

// Create inserts a new resource tag instance with the ID and RecordCreated fields set.
func (r *resourceRepoImpl) Create(ctx context.Context, it *types.ResourceTags) error {
	it.ID = core.NewID()
	ct := r.clock.GetCurrentTime()
	it.RecordCreated, it.RecordUpdated = ct, ct

	err := r.DB(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "type"}, {Name: "name"}, {Name: "namespace"}},
			DoNothing: true,
		}).Create(it).Error
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("storage write create failure")
		StorageWriteFailures.With(prometheus.Labels{
			"action":        "create",
			"resource_type": fmt.Sprintf("%d", it.Type),
			"namespace":     *it.Namespace,
			"resource_name": it.Name,
		}).Inc()
		return core.TranslateError(err)
	}
	return nil
}

// Delete removes a resource tag instance by its ID.
func (r *resourceRepoImpl) Delete(ctx context.Context, id string) error {
	if err := r.DB(ctx).Delete(&types.ResourceTags{}, "id = ?", id).Error; err != nil {
		return core.TranslateError(err)
	}
	return nil
}

// Get retrieves a resource tag instance by its ID.
func (r *resourceRepoImpl) Get(ctx context.Context, id string) (*types.ResourceTags, error) {
	it := &types.ResourceTags{}
	if err := r.DB(ctx).First(it, "id = ?", id).Error; err != nil {
		return nil, core.TranslateError(err)
	}
	return it, nil
}

// Update modifies an existing resource tag instance.
func (r *resourceRepoImpl) Update(ctx context.Context, it *types.ResourceTags) error {
	if it.ID == "" {
		return types.ErrMissingKey
	}
	it.RecordUpdated = r.clock.GetCurrentTime()

	// Serialize MetricLabels
	var metricLabelsJSON []byte
	if it.MetricLabels != nil {
		var err error
		metricLabelsJSON, err = json.Marshal(it.MetricLabels)
		if err != nil {
			return fmt.Errorf("failed to serialize MetricLabels: %w", err)
		}
	}

	// Serialize Labels
	var labelsJSON []byte
	if it.Labels != nil {
		var err error
		labelsJSON, err = json.Marshal(it.Labels)
		if err != nil {
			return fmt.Errorf("failed to serialize Labels: %w", err)
		}
	}

	// Serialize Annotations
	var annotationsJSON []byte
	if it.Annotations != nil {
		var err error
		annotationsJSON, err = json.Marshal(it.Annotations)
		if err != nil {
			return fmt.Errorf("failed to serialize Annotations: %w", err)
		}
	}

	// Prepare the updates map with serialized JSON
	updates := map[string]interface{}{
		"metric_labels":  string(metricLabelsJSON),
		"labels":         string(labelsJSON),
		"annotations":    string(annotationsJSON),
		"sent_at":        it.SentAt,
		"record_updated": it.RecordUpdated,
	}

	// Perform the update
	if err := r.DB(ctx).Model(it).
		Where("id = ?", it.ID).
		Updates(updates).Error; err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("storage write update failure")
		StorageWriteFailures.With(prometheus.Labels{
			"action":        "update",
			"resource_type": fmt.Sprintf("%d", it.Type),
			"namespace":     *it.Namespace,
			"resource_name": it.Name,
		}).Inc()
		return core.TranslateError(err)
	}
	return nil
}

// FindFirstBy returns the first record that matches the provided conditions.
func (r *resourceRepoImpl) FindFirstBy(ctx context.Context, conds ...interface{}) (*types.ResourceTags, error) {
	it := &types.ResourceTags{}
	if err := r.DB(ctx).First(it, conds...).Error; err != nil {
		return nil, core.TranslateError(err)
	}
	return it, nil
}

// FindAllBy returns all records that match the provided conditions.
func (r *resourceRepoImpl) FindAllBy(ctx context.Context, conds ...interface{}) ([]*types.ResourceTags, error) {
	it := []*types.ResourceTags{}
	if err := r.DB(ctx).Find(&it, conds...).Error; err != nil {
		return nil, core.TranslateError(err)
	}
	return it, nil
}
