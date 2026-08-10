// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasUsableConfig(t *testing.T) {
	tests := []struct {
		name        string
		configFiles []string
		want        bool
	}{
		{name: "nil slice", configFiles: nil, want: false},
		{name: "empty slice", configFiles: []string{}, want: false},
		// The regression this guards: the webhook server launched with
		// --config "" gets a non-empty slice of one empty string. NewSettings
		// would treat it as a valid empty config (CP-45528), so main must reject
		// it here instead of starting an unusable server.
		{name: "single empty string", configFiles: []string{""}, want: false},
		{name: "whitespace only", configFiles: []string{"   "}, want: false},
		{name: "real path", configFiles: []string{"/etc/config/server-config.yaml"}, want: true},
		{name: "empty then real", configFiles: []string{"", "/etc/config/server-config.yaml"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, hasUsableConfig(tt.configFiles))
		})
	}
}
