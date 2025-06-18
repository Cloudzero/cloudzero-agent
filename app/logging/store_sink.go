// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type storeWriter struct {
	store types.WritableStore
	ctx   context.Context

	clusterName    string
	cloudAccountID string
}

// Write implements io.Writer.
// Parses the raw json bytes from zerolog into a types.Metric
func (s *storeWriter) Write(p []byte) (n int, err error) {
	var logEntry map[string]interface{}
	if err := json.Unmarshal(p, &logEntry); err != nil {
		// Handle error
		return len(p), nil // Consume log even if unmarshal fails
	}

	// pull out the message itself
	var msg string
	if m, exists := logEntry[zerolog.MessageFieldName]; exists {
		// attempt to read as string or fallback
		if mStr, ok := m.(string); ok {
			msg = mStr
		} else {
			msg = fmt.Sprintf("%v", m)
		}
	}

	// parse the timestamp
	var ts time.Time
	if t, exists := logEntry[zerolog.TimestampFieldName]; exists {
		if tStr, ok := t.(string); ok {
			if parsed, err := time.Parse(zerolog.TimeFieldFormat, tStr); err == nil {
				ts = parsed
			}
		}
	}

	// convert the object blob to a string map
	stringMap := make(map[string]string)

	for key, value := range logEntry {
		if value == nil {
			stringMap[key] = "null"
			continue
		}

		switch v := value.(type) {
		case string:
			stringMap[key] = v
		case float64:
			stringMap[key] = fmt.Sprintf("%g", v)
		case bool:
			stringMap[key] = strconv.FormatBool(v)
		case map[string]interface{}, []interface{}:
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				stringMap[key] = fmt.Sprintf("%v", v)
			} else {
				stringMap[key] = string(jsonBytes)
			}
		default:
			stringMap[key] = fmt.Sprintf("%v", v)
		}
	}

	_ = s.store.Put(s.ctx, types.Metric{
		ID:             uuid.New(),
		ClusterName:    s.clusterName,
		CloudAccountID: s.cloudAccountID,
		MetricName:     "log",
		CreatedAt:      ts,
		TimeStamp:      ts,
		Labels:         stringMap,
		Value:          msg,
	})

	return len(p), nil
}

func StoreWriter(
	ctx context.Context,
	store types.WritableStore,
	clusterName string,
	cloudAccountID string,
) io.Writer {
	return &storeWriter{
		store:          store,
		ctx:            ctx,
		clusterName:    clusterName,
		cloudAccountID: cloudAccountID,
	}
}
