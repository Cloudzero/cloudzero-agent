// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package scout provides cloud environment detection and metadata retrieval
// capabilities for cloud environments.
package scout

import (
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/auto"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/aws"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/azure"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/google"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
)

// NewScout creates a new Scout implementation with auto-detection capabilities.
func NewScout() types.Scout {
	return auto.NewScout(
		aws.NewScout(),
		azure.NewScout(),
		google.NewScout(),
	)
}
