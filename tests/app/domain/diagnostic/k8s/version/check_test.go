// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package version_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic/k8s/version"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/cloudzero/cloudzero-agent/tests/utils"
)

var k8sAPIResponseBody = `{
  "major": "1",
  "minor": "29",
  "gitVersion": "v1.29.6+k3s1",
  "gitCommit": "83ae095ab9197f168a6bd3f6bd355f89bce39a9c",
  "gitTreeState": "clean",
  "buildDate": "2024-06-25T18:24:29Z",
  "goVersion": "go1.21.11",
  "compiler": "gc",
  "platform": "linux/arm64"
}`

func makeReport() status.Accessor {
	return status.NewAccessor(&status.ClusterStatus{})
}

func TestChecker_CheckOK(t *testing.T) {
	t.Skip("Skipping test - comment this out to manually run if you have a k8s cluster running locally")
	cfg := &config.Settings{}

	// IMPORTANT:
	// 1. CI/CD will require a known K8s (kind) versionm
	// 2. If you are running thislocal, I suggest deploying Rancher Desktop
	//
	// Ideally we improve our MockTransport to handle this
	// Allowing us to overide the client config.Transport
	//
	// XXX: Replace with the expected version
	expectedVersion := "1.29"

	provider := version.NewProvider(context.Background(), cfg)

	mock := utils.NewHTTPMock()
	mock.Expect(http.MethodGet, k8sAPIResponseBody, 200, nil)
	client := mock.HTTPClient()

	accessor := makeReport()
	err := provider.Check(context.Background(), client, accessor)
	assert.NoError(t, err)

	accessor.ReadFromReport(func(s *status.ClusterStatus) {
		assert.Len(t, s.Checks, 1)
		for _, c := range s.Checks {
			assert.True(t, c.Passing)
		}
		assert.Equal(t, expectedVersion, s.K8SVersion)
	})
}

// func TestK8s
