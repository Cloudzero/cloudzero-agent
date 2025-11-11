// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package helmless provides embedded default Helm values for override extraction.
package helmless

import (
	_ "embed"
)

// DefaultValues contains the default Helm chart values, embedded at build time.
//
//go:embed default-values.yaml
var DefaultValues []byte
