// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build tools

// tools.go is used by the install-tools-go Makefile target to install Go tools.
// The install-tools-go target uses grep to extract import paths from this file
// and runs 'go install' on them. These should be main packages that we want
// to install as binaries in .tools/bin/.
//
// Usage: make install-tools-go
package tools

import (
	_ "github.com/homeport/dyff/cmd/dyff"
	_ "github.com/itchyny/gojq/cmd/gojq"
	_ "github.com/nektos/act"
	_ "github.com/yannh/kubeconform/cmd/kubeconform"
	_ "go.uber.org/mock/mockgen"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	_ "helm.sh/helm/v3/cmd/helm"
	_ "honnef.co/go/tools/cmd/staticcheck"
	_ "mvdan.cc/gofumpt"
	_ "sigs.k8s.io/kind/cmd/kind"
)
