// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cfg_gator "github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/app/types/clusterconfig"
)

func TestPost(t *testing.T) {
	t.Run("nil aggregator settings is reported, not dereferenced", func(t *testing.T) {
		// When every config fails to build, settingsAggregator is nil. post must
		// return a reportable error rather than dereferencing it into a SIGSEGV.
		// See CP-45528.
		err := post(context.Background(), nil, &clusterconfig.ClusterConfig{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil settings")
	})

	t.Run("nil clusterConfig is reported", func(t *testing.T) {
		err := post(context.Background(), &cfg_gator.Settings{}, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil clusterConfig")
	})
}
