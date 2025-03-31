// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package instr

import "context"

// span key is the value that is embeded into a context
type spanCtxKey string

// the value embeded as a key into the context
const spanIDKey spanCtxKey = "currentSpanID"

func getParentID(ctx context.Context) string {
	if ps, ok := ctx.Value(spanIDKey).(string); ok {
		return ps
	}
	return ""
}
