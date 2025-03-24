// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package filter

import (
	"fmt"
	"regexp"
	"strings"
)

type FilterMatchType string

const (
	FilterMatchTypeExact    FilterMatchType = "exact"
	FilterMatchTypePrefix   FilterMatchType = "prefix"
	FilterMatchTypeSuffix   FilterMatchType = "suffix"
	FilterMatchTypeContains FilterMatchType = "contains"
	FilterMatchTypeRegex    FilterMatchType = "regex"
)

type FilterEntry struct {
	Pattern string
	Match   FilterMatchType
}

// filterChecker is a small utility which allows us to check if a value matches
// a test pattern, using various methods.
type FilterChecker struct {
	exactMatches    map[string]bool
	prefixMatches   []string
	suffixMatches   []string
	containsMatches []string
	regexMatches    []*regexp.Regexp
}

func NewFilterChecker(filters []FilterEntry) (*FilterChecker, error) {
	if len(filters) == 0 {
		return nil, nil //nolint:nilnil // methods handle nil properly, returning nil allows us to elide code
	}

	chk := &FilterChecker{
		exactMatches: map[string]bool{},
	}

	for _, filter := range filters {
		switch filter.Match {
		case FilterMatchTypeExact:
			chk.exactMatches[filter.Pattern] = true
		case FilterMatchTypePrefix:
			chk.prefixMatches = append(chk.prefixMatches, filter.Pattern)
		case FilterMatchTypeSuffix:
			chk.suffixMatches = append(chk.suffixMatches, filter.Pattern)
		case FilterMatchTypeContains:
			chk.containsMatches = append(chk.containsMatches, filter.Pattern)
		case FilterMatchTypeRegex:
			regex, err := regexp.Compile(filter.Pattern)
			if err != nil {
				return nil, fmt.Errorf("failed to compile regex: %w", err)
			}

			chk.regexMatches = append(chk.regexMatches, regex)
		default:
			return nil, fmt.Errorf("unknown filter match type: %s", filter.Match)
		}
	}

	return chk, nil
}

func (chk *FilterChecker) Test(value string) bool {
	if chk == nil {
		return true
	}

	if _, found := chk.exactMatches[value]; found {
		return true
	}

	for _, prefix := range chk.prefixMatches {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}

	for _, suffix := range chk.suffixMatches {
		if strings.HasSuffix(value, suffix) {
			return true
		}
	}

	for _, contains := range chk.containsMatches {
		if strings.Contains(value, contains) {
			return true
		}
	}

	for _, regex := range chk.regexMatches {
		if regex.MatchString(value) {
			return true
		}
	}

	return false
}
