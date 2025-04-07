// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package gh_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cloudzero/cloudzero-agent/pkg/util/gh"
	"github.com/stretchr/testify/assert"
)

func TestGetLatestRelease(t *testing.T) {
	// Create a mock server to simulate the GitHub API response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/owner/repo/releases/latest", r.URL.Path)
		assert.Equal(t, "application/vnd.github+json", r.Header.Get("Accept"))
		assert.Equal(t, "2022-11-28", r.Header.Get("X-GitHub-Api-Version"))

		response := `{"name": "v1.0.0"}`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer mockServer.Close()

	// Call the method under test
	latestRelease, err := gh.GetLatestRelease(mockServer.URL, "owner", "repo")

	// Assert the expected result
	assert.NoError(t, err)
	assert.Equal(t, "v1.0.0", latestRelease)
}
