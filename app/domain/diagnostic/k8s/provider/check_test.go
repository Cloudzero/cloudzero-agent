// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package provider_test

import (
	"context"
	"os"
	"testing"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic/k8s/provider"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/stretchr/testify/require"
)

func TestUnit_Diagnostic_K8s_Provider_CheckOK(t *testing.T) {
	t.Skip("Skipping test - comment this out to manually run if you have a k8s cluster running locally")
	cfg := &config.Settings{}

	// WHEN RUNNING LOCALLY:
	// set this to the namespace and the pod name you have currently running within a
	// local kubernetes cluster.
	// TODO: we should run some tests inside of a k8s cluster
	os.Setenv("K8S_NAMESPACE", "flux-system")
	os.Setenv("K8S_POD_NAME", "flux-59dbfd5444-dlr5c")

	p := provider.NewProvider(t.Context(), cfg)
	accessor := status.NewAccessor(&status.ClusterStatus{})
	err := p.Check(context.Background(), nil, accessor)
	require.NoError(t, err)
}
