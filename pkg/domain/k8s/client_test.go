//go:build integration
// +build integration

// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package k8s_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/domain/k8s"
)

func TestBuildKubeClient(t *testing.T) {
	t.Run("Valid kubeconfig path", func(t *testing.T) {
		// Create a fake kubeconfig
		kubeconfig := clientcmd.NewDefaultClientConfigLoadingRules().GetDefaultFilename()

		clientset, err := k8s.NewClient(kubeconfig)
		assert.NoError(t, err)
		assert.NotNil(t, clientset)
	})

	t.Run("Invalid kubeconfig path", func(t *testing.T) {
		clientset, err := k8s.NewClient("/invalid/path/to/kubeconfig")
		assert.Error(t, err)
		assert.Nil(t, clientset)
	})
}
