// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"testing"

	"github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/stretchr/testify/assert"
)

func TestVersions_Validate(t *testing.T) {
	versions := &config.Versions{
		ChartVersion: "1.0.0",
		AgentVersion: "2.0.0",
	}

	err := versions.Validate()
	assert.NoError(t, err)
}
