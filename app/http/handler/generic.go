// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudzero/cloudzero-agent/app/instr"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/rs/zerolog/log"
)

// writeDataToStorage takes a record and writes it to the database
// with some more advanced tracing
func genericWriteDataToStorage(
	ctx context.Context,
	store types.ResourceStore,
	clock types.TimeProvider,
	record types.ResourceTags,
) {
	err := instr.RunSpan(ctx, "writeDataToStorage", func(ctx context.Context, span *instr.Span) error {
		conditions := []interface{}{}
		if record.Namespace != nil {
			conditions = append(conditions, "type = ? AND name = ? AND namespace = ?", record.Type, record.Name, *record.Namespace)
		} else {
			conditions = append(conditions, "type = ? AND name = ?", record.Type, record.Name)
		}

		log.Ctx(ctx).Debug().Msg("Searching for an existing record ...")
		var found *types.ResourceTags
		var err error
		err = instr.RunSpan(ctx, "writeDataToStorage_findRecord", func(ctx context.Context, span *instr.Span) error {
			found, err = store.FindFirstBy(ctx, conditions...)
			return err
		})

		switch {
		case (err != nil && errors.Is(err, types.ErrNotFound)) || found == nil:
			log.Ctx(ctx).Debug().Msg("No existing record found")
			log.Ctx(ctx).Debug().Msg("Creating record ...")
			err = instr.RunSpan(ctx, "writeDataToStorage_createRecord", func(ctx context.Context, span *instr.Span) error {
				return store.Tx(ctx, func(txCtx context.Context) error {
					return store.Create(txCtx, &record)
				})
			})
			if err != nil {
				return fmt.Errorf("failed to create the resource: %w", err)
			}
		case found != nil:
			log.Ctx(ctx).Debug().Msg("Existing record found")
			log.Ctx(ctx).Debug().Msg("Updating record ...")
			err = instr.RunSpan(ctx, "writeDataToStorage_updateRecord", func(ctx context.Context, span *instr.Span) error {
				return store.Tx(ctx, func(txCtx context.Context) error {
					record.ID = found.ID
					record.RecordCreated = found.RecordCreated
					record.RecordUpdated = clock.GetCurrentTime()
					record.SentAt = nil // reset send
					return store.Update(txCtx, &record)
				})
			})
			if err != nil {
				return fmt.Errorf("failed to update the resource: %w", err)
			}
		default:
			return fmt.Errorf("there was an issue searching for the resource: %w", err)
		}

		return nil
	})

	if err != nil {
		log.Err(err).Msg("failed to write data to storage")
	} else {
		log.Ctx(ctx).Debug().Msg("Successfully wrote data to storage")
	}
}
