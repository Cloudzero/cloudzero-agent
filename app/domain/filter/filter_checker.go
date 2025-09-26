// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package filter provides high-performance metric filtering utilities for CloudZero Agent cost allocation.
// This package implements the critical filtering logic that separates cost-related metrics from
// observability metrics, enabling proper routing to financial analysis versus operational monitoring systems.
//
// The filter system operates as part of the Application Core in the hexagonal architecture,
// implementing business logic for metric classification that determines billing attribution
// and cost optimization processing paths.
//
// Key responsibilities:
//   - Metric classification: Distinguish cost metrics from observability metrics
//   - Pattern matching: Efficient multi-pattern filtering with various match types
//   - Performance optimization: Fast filtering for high-volume metric streams
//   - Configuration flexibility: Support for exact, prefix, suffix, contains, and regex patterns
//
// Architecture:
//   - FilterChecker: High-performance pattern matching engine with optimized data structures
//   - FilterEntry: Configuration structure for individual filter rules
//   - FilterMatchType: Enumeration of supported pattern matching algorithms
//
// The filtering system is designed for production throughput requirements where millions
// of metrics may need classification during peak collection periods. The implementation
// uses optimized data structures and algorithms to minimize processing overhead while
// maintaining comprehensive pattern matching capabilities.
//
// Integration points:
//   - Metric collector: Uses filters during initial metric ingestion
//   - Storage routing: Directs metrics to appropriate storage destinations
//   - Cost allocation: Identifies metrics requiring financial analysis
//   - Monitoring systems: Routes observability metrics to operational dashboards
package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// FilterMatchType defines the algorithm used for pattern matching during metric filtering.
// This enumeration enables flexible configuration of filtering rules with different
// performance characteristics optimized for various metric naming patterns.
//
// Match types are ordered by performance (fastest to slowest) to guide configuration
// decisions in high-throughput environments where filtering performance is critical.
type FilterMatchType string

const (
	// FilterMatchTypeExact performs O(1) hash-based exact string matching.
	// This is the fastest matching algorithm and should be preferred when
	// complete metric names are known. Used for filtering specific metrics
	// like "cloudzero_cost_total" or "prometheus_notifications_total".
	//
	// Performance: O(1) lookup time, optimal memory usage with map storage
	// Use cases: Known metric names, specific cost allocation identifiers
	FilterMatchTypeExact FilterMatchType = "exact"

	// FilterMatchTypePrefix performs O(n) prefix matching against metric names.
	// Efficient for filtering metric families or namespaces with common prefixes.
	// Used for filtering all CloudZero metrics ("cloudzero_") or Kubernetes
	// metrics ("kube_") during cost allocation processing.
	//
	// Performance: O(n) where n is prefix length, memory efficient
	// Use cases: Metric namespaces, cost allocation by service prefix
	FilterMatchTypePrefix FilterMatchType = "prefix"

	// FilterMatchTypeSuffix performs O(n) suffix matching against metric names.
	// Useful for filtering metrics by their measurement type or unit suffix.
	// Used for filtering all counter metrics ("_total") or rate metrics ("_rate")
	// during cost versus observability classification.
	//
	// Performance: O(n) where n is suffix length, memory efficient
	// Use cases: Metric types, measurement units, cost allocation by metric kind
	FilterMatchTypeSuffix FilterMatchType = "suffix"

	// FilterMatchTypeContains performs O(n*m) substring matching within metric names.
	// More expensive than prefix/suffix but enables flexible pattern matching.
	// Used for filtering metrics containing specific keywords like "cost", "billing",
	// or "allocation" regardless of their position in the metric name.
	//
	// Performance: O(n*m) where n is string length, m is pattern length
	// Use cases: Keyword-based filtering, complex metric name patterns
	FilterMatchTypeContains FilterMatchType = "contains"

	// FilterMatchTypeRegex performs full regular expression matching against metric names.
	// Most flexible but also most expensive matching algorithm. Should be used sparingly
	// in high-throughput scenarios. Used for complex pattern matching that cannot
	// be expressed with simpler algorithms.
	//
	// Performance: O(n) to O(n^2) depending on regex complexity, highest memory usage
	// Use cases: Complex patterns, advanced metric classification rules
	FilterMatchTypeRegex FilterMatchType = "regex"
)

// FilterEntry represents a single filtering rule with its pattern and matching algorithm.
// This structure enables configuration-driven filtering where rules can be loaded
// from configuration files, environment variables, or dynamic updates from the
// CloudZero platform without requiring code changes.
//
// Filter entries are compiled into optimized data structures by FilterChecker
// for high-performance pattern matching during metric processing.
type FilterEntry struct {
	// Pattern specifies the string pattern to match against metric names.
	// The interpretation depends on the Match type:
	//   - Exact: Complete metric name to match exactly
	//   - Prefix: String that metric names must start with
	//   - Suffix: String that metric names must end with
	//   - Contains: Substring that must appear anywhere in metric name
	//   - Regex: Regular expression pattern for complex matching
	//
	// Examples:
	//   - Exact: "cloudzero_cost_total"
	//   - Prefix: "cloudzero_"
	//   - Suffix: "_total"
	//   - Contains: "cost"
	//   - Regex: "^(cloudzero|kube)_.*_(total|gauge)$"
	Pattern string

	// Match specifies the algorithm to use when testing this pattern against metric names.
	// The choice affects both performance and matching behavior, enabling optimization
	// of filter configurations for specific deployment scenarios and metric volumes.
	//
	// Performance considerations:
	//   - Exact matches should be preferred when possible for O(1) performance
	//   - Prefix/Suffix matches are efficient for namespace-based filtering
	//   - Contains matches are more expensive but offer flexibility
	//   - Regex matches should be used sparingly in high-throughput scenarios
	Match FilterMatchType
}

// FilterChecker provides high-performance pattern matching for metric filtering operations.
// This struct compiles FilterEntry configurations into optimized data structures that
// minimize memory usage and maximize throughput during metric classification processing.
//
// The checker uses specialized data structures for each match type to achieve optimal
// performance characteristics:
//   - Hash maps for O(1) exact matching
//   - Slices for efficient iteration over prefix/suffix/contains patterns
//   - Compiled regex objects for complex pattern matching
//
// Design principles:
//   - Performance optimization: Separate storage by match type for algorithmic efficiency
//   - Memory efficiency: Minimal overhead per pattern with appropriate data structures
//   - Nil safety: Graceful handling of empty filter configurations
//   - Immutable state: Thread-safe operation after initialization
//
// The FilterChecker is thread-safe after construction and can be used concurrently
// across multiple goroutines processing metrics in parallel without synchronization.
type FilterChecker struct {
	// exactMatches provides O(1) hash-based lookup for exact string matching.
	// Maps metric names to boolean values for fast membership testing.
	// This is the most efficient matching algorithm and should be preferred
	// when complete metric names are known in advance.
	exactMatches map[string]bool

	// prefixMatches stores prefix patterns for O(n) prefix matching operations.
	// Patterns are tested in order until a match is found, making pattern
	// ordering important for performance optimization in high-frequency scenarios.
	prefixMatches []string

	// suffixMatches stores suffix patterns for O(n) suffix matching operations.
	// Similar to prefixes but tests the end of metric names for pattern matches.
	// Useful for filtering by metric type or measurement unit suffixes.
	suffixMatches []string

	// containsMatches stores substring patterns for O(n*m) contains matching.
	// More expensive than prefix/suffix matching but provides flexibility
	// for keyword-based filtering regardless of position in metric names.
	containsMatches []string

	// regexMatches stores compiled regular expression objects for complex pattern matching.
	// Regex patterns are pre-compiled during construction to avoid compilation overhead
	// during metric processing. Most expensive matching option but most flexible.
	regexMatches []*regexp.Regexp
}

// NewFilterChecker constructs an optimized FilterChecker from a collection of filter rules.
// This constructor compiles FilterEntry configurations into high-performance data structures
// specialized for each matching algorithm, enabling efficient metric filtering operations.
//
// Optimization process:
//  1. Validates all filter entries and compiles regex patterns upfront
//  2. Groups patterns by match type for algorithmic efficiency
//  3. Pre-compiles regex patterns to avoid runtime compilation overhead
//  4. Initializes optimized data structures (maps for exact, slices for others)
//  5. Returns nil for empty configurations to enable efficient nil-checking
//
// Performance considerations:
//   - Regex compilation errors are caught during construction, not during filtering
//   - Data structure initialization minimizes allocation overhead during filtering
//   - Pattern grouping enables algorithm-specific optimizations
//   - Nil return for empty filters eliminates unnecessary processing overhead
//
// Error conditions:
//   - Invalid regex patterns in FilterMatchTypeRegex entries
//   - Unknown FilterMatchType values that don't map to supported algorithms
//
// The returned FilterChecker is immutable and thread-safe, suitable for concurrent
// use across multiple goroutines processing metrics in parallel.
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

// Test determines whether a metric name matches any of the configured filter patterns.
// This method implements the core filtering logic used throughout the CloudZero Agent
// to classify metrics for cost allocation versus observability routing.
//
// Algorithm execution order (optimized for performance):
//  1. Exact matching: O(1) hash lookup for known metric names
//  2. Prefix matching: O(n) iteration with early termination on match
//  3. Suffix matching: O(n) iteration with early termination on match
//  4. Contains matching: O(n*m) substring search with early termination
//  5. Regex matching: O(n) to O(n^2) pattern matching with early termination
//
// Performance optimization:
//   - Algorithms are ordered from fastest to slowest for optimal average performance
//   - Early termination on first match avoids unnecessary pattern testing
//   - Nil safety enables efficient handling of empty filter configurations
//   - No memory allocation during testing for minimal GC pressure
//
// Nil handling:
//
//	Returns true for nil FilterChecker instances, implementing a "pass-through"
//	behavior that allows metrics to proceed when no filtering is configured.
//	This design enables optional filtering without conditional code throughout the application.
//
// Thread safety:
//
//	This method is thread-safe and can be called concurrently from multiple
//	goroutines without synchronization, as it only reads immutable state.
//
// Use cases:
//   - Cost metric identification during collection pipeline
//   - Observability metric routing to monitoring systems
//   - Custom filtering rules for specific deployment environments
//   - Dynamic metric classification based on naming patterns
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
