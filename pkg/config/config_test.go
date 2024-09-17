package config_test

import (
	"regexp"
	"testing"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := config.Filter(tt.tags, tt.patterns, tt.enabled)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
