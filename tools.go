// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build tools

package tools

import (
	_ "go.uber.org/mock/mockgen"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	_ "honnef.co/go/tools/cmd/staticcheck"
	_ "mvdan.cc/gofumpt"
)
