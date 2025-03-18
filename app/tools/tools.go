// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build tools
// +build tools

package tools

import (
	// Tools we use during development.
	_ "honnef.co/go/tools/cmd/staticcheck"
)
