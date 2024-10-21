package config

import (
	"regexp"

	"github.com/rs/zerolog/log"

	"github.com/microcosm-cc/bluemonday"
)

type Filters struct {
	Labels      Labels      `yaml:"labels"`      //nolint:tagliatelle
	Annotations Annotations `yaml:"annotations"` //nolint:tagliatelle
	Policy      bluemonday.Policy
}

// Represents any metric label that can be added to a metric; "pod", "namespace", "label_foo" etc.
type MetricLabels = map[string]string

// Represents metric labels attached to a metric that represent annotations or labels; value must be prefixed with "label_"
type MetricLabelTags = map[string]string

func Filter(tags map[string]string, patterns []regexp.Regexp, enabled bool, settings Settings) MetricLabelTags {
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

func evalTag(key string, value string, patterns []regexp.Regexp, settings Settings) bool {
	if settings.Filters.Policy.Sanitize(key) != key {
		log.Warn().Msgf("tag: %s does not satisfy filter policy", key)
		return false
	} else if settings.Filters.Policy.Sanitize(value) != value {
		log.Warn().Msgf("tag value: %s does not satisfy filter policy", key)
		return false
	}
	for _, pattern := range patterns {
		if pattern.MatchString(key) {
			return true
		}
	}
	return false
}
