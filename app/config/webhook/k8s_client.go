// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

type K8sClient struct {
	KubeConfig      string `yaml:"kube_config" env:"KUBE_CONFIG" default:"false" env-description:"path to the kubeconfig file"`
	PaginationLimit int64  `yaml:"pagination_limit" env:"KUBE_PAGINATION_LIMIT" default:"500" env-description:"limit for pagination"`
}
