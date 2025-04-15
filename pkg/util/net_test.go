// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractHostnameFromURL(t *testing.T) {
	tests := []struct {
		name        string
		rawURL      string
		expected    string
		expectError bool
	}{
		{
			name:        "Valid URL with hostname",
			rawURL:      "https://example.com/path",
			expected:    "example.com",
			expectError: false,
		},
		{
			name:        "Valid URL with subdomain",
			rawURL:      "https://sub.example.com/path",
			expected:    "sub.example.com",
			expectError: false,
		},
		{
			name:        "Invalid URL",
			rawURL:      "://invalid-url",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Empty URL",
			rawURL:      "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "URL with IP address",
			rawURL:      "http://192.168.1.1/path",
			expected:    "192.168.1.1",
			expectError: false,
		},
		{
			name:        "URL with port",
			rawURL:      "http://example.com:8080/path",
			expected:    "example.com",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractHostnameFromURL(tt.rawURL)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
