// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"regexp"

	"github.com/rs/zerolog/log"

	"github.com/microcosm-cc/bluemonday"
)

type Filters struct {
	Labels      Labels      `yaml:"labels"`
	Annotations Annotations `yaml:"annotations"`
	Policy      bluemonday.Policy
}

// MetricLabels represents any metric label that can be added to a metric; "pod", "namespace", "label_foo" etc.
type MetricLabels = map[string]string

// MetricLabelTags represents metric labels attached to a metric that represent annotations or labels; value must be prefixed with "label_"
type MetricLabelTags = map[string]string

func Filter(tags map[string]string, patterns []regexp.Regexp, enabled bool, settings *Settings) MetricLabelTags {
	filteredTags := make(MetricLabels)
	if !enabled {
		return filteredTags
	}
	for key, value := range tags {
		if evalTag(key, value, patterns, settings) {
			filteredTags[key] = value
		}
	}
	return filteredTags
}

func evalTag(key string, value string, patterns []regexp.Regexp, settings *Settings) bool {
	if settings.Filters.Policy.Sanitize(key) != key {
		log.Debug().Str("tag", key).Msg("tag does not satisfy filter policy")
		return false
	} else if settings.Filters.Policy.Sanitize(value) != value {
		log.Debug().Str("value", value).Msg("tag value does not satisfy filter policy")
		return false
	}
	for _, pattern := range patterns {
		if pattern.MatchString(key) {
			return true
		}
	}
	return false
}
