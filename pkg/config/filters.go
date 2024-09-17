package config

import (
	"regexp"
)

type Filters struct {
	Labels      Labels      `yaml:"labels"`      //nolint:tagliatelle
	Annotations Annotations `yaml:"annotations"` //nolint:tagliatelle

}

// Represents any metric label that can be added to a metric; "pod", "namespace", "label_foo" etc.
type MetricLabels = map[string]string

// Represents metric labels attached to a metric that represent annotations or labels; value must be prefixed with "label_"
type MetricLabelTags = map[string]string

func Filter(tags map[string]string, patterns []regexp.Regexp, enabled bool) MetricLabels {
	filteredTags := make(MetricLabels)
	if !enabled {
		return filteredTags
	}
	for key, value := range tags {
		if evalTag(key, patterns) {
			filteredTags[key] = value
		}
	}
	return filteredTags
}

func evalTag(tag string, patterns []regexp.Regexp) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(tag) {
			return true
		}
	}
	return false
}
