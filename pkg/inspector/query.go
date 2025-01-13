// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package inspector

import (
	"fmt"
	"sync"

	"github.com/itchyny/gojq"
)

// gojqQueryCache contains a cache of parsed gojq queries.
type gojqQueryCache struct {
	cache map[string]*gojq.Query
	lock  sync.Mutex
}

// gojqCache is the global cache of parsed gojq queries.
var gojqCache = gojqQueryCache{
	cache: map[string]*gojq.Query{},
}

// Get returns a cached gojq.Query for the given query string. If the query is
// not cached, it will be parsed and added to the cache.
func (cache *gojqQueryCache) Get(query string) (*gojq.Query, error) {
	// If the query is already cached, return it.
	if jq, ok := cache.cache[query]; ok {
		return jq, nil
	}

	// Parse the query.
	newQuery, err := gojq.Parse(query)
	if err != nil {
		return nil, err
	}

	// Add the query to the cache.
	func() {
		cache.lock.Lock()
		defer cache.lock.Unlock()

		cache.cache[query] = newQuery
	}()

	return newQuery, nil
}

// JSONQuery evaluates the query against the data.
func (cache *gojqQueryCache) JSONQuery(query string, data any) (gojq.Iter, error) {
	jq, err := cache.Get(query)
	if err != nil {
		return nil, fmt.Errorf("failed to run JQ query: %w", err)
	}

	return jq.Run(data), nil
}

// JSONMatch tests if tha data matches the query, which must return a boolean value.
func (cache *gojqQueryCache) JSONMatch(query string, data any) (bool, error) {
	iter, err := cache.JSONQuery(query, data)
	if err != nil {
		return false, err
	}

	v, ok := iter.Next()
	if !ok {
		return false, nil
	}

	if err, ok := v.(error); ok {
		return false, fmt.Errorf("JQ query returned error: %w", err)
	}

	resultValue, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("non-boolean JQ result: %#v", v)
	}

	return resultValue, nil
}
