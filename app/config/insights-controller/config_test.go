// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"regexp"
	"testing"

	"github.com/cloudzero/cloudzero-agent/app/config/insights-controller"
	"github.com/stretchr/testify/assert"
)

func TestFilter(t *testing.T) {
	tests := []struct {
		name     string
		tags     map[string]string
		patterns []regexp.Regexp
		enabled  bool
		expected config.MetricLabels
	}{
		{
			name: "filter enabled with matching patterns",
			tags: map[string]string{
				"app":        "myapp",
				"appFoo":     "myappFoo",
				"namespace":  "default",
				"env":        "prod",
				"annotation": "test",
			},
			patterns: []regexp.Regexp{
				*regexp.MustCompile(`^app`),
			},
			enabled: true,
			expected: config.MetricLabels{
				"app":    "myapp",
				"appFoo": "myappFoo",
			},
		},
		{
			name: "filter enabled with multiple matching patterns",
			tags: map[string]string{
				"foo":        "bar",
				"bat":        "baz",
				"namespace":  "default",
				"env":        "prod",
				"annotation": "test",
			},
			patterns: []regexp.Regexp{
				*regexp.MustCompile(`^foo`),
				*regexp.MustCompile(`^bat`),
			},
			enabled: true,
			expected: config.MetricLabels{
				"foo": "bar",
				"bat": "baz",
			},
		},
		{
			name: "filter enabled with no matching patterns",
			tags: map[string]string{
				"app":        "myapp",
				"namespace":  "default",
				"label_env":  "prod",
				"annotation": "test",
			},
			patterns: []regexp.Regexp{
				*regexp.MustCompile(`^doesnotexist`),
			},
			enabled:  true,
			expected: config.MetricLabels{},
		},
		{
			name: "filter disabled",
			tags: map[string]string{
				"app":        "myapp",
				"namespace":  "default",
				"env":        "prod",
				"annotation": "test",
			},
			patterns: []regexp.Regexp{
				*regexp.MustCompile(`^app`),
			},
			enabled:  false,
			expected: config.MetricLabels{},
		},
		{
			name: "filter enabled with matching patterns and sanitized tags",
			tags: map[string]string{
				"app": `Hello<script>alert('You have been hacked!');</script> <b>World</b> <img src="http://malicious-site.com" onerror="alert('Malicious Image!')"/>`,
				`Hello <STYLE>.XSS{background-image:url("javascript:alert('XSS')");}</STYLE><A CLASS=XSS></A>World`: "default",
			},
			patterns: []regexp.Regexp{
				*regexp.MustCompile(`.*`),
			},
			enabled:  true,
			expected: config.MetricLabels{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := config.Filter(tt.tags, tt.patterns, tt.enabled, &config.Settings{})
			assert.Equal(t, tt.expected, actual)
		})
	}
}
