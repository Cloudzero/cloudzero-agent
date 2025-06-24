// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build tools

package tools

import (
	_ "github.com/itchyny/gojq/cmd/gojq"
	_ "github.com/yannh/kubeconform/cmd/kubeconform"
	_ "go.uber.org/mock/mockgen"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	_ "helm.sh/helm/v3/cmd/helm"
	_ "honnef.co/go/tools/cmd/staticcheck"
	_ "mvdan.cc/gofumpt"
)
