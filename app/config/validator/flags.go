// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

type Flags struct {
	Mode  string
	Stage string

	EnvFile    string
	ConfigFile string
}

const (
	FlagEnvFile      = "env-file"
	FlagDescEnvFile  = "environment variable configuration file"
	FlagConfigFile   = "config-file"
	FlagDescConfFile = "configuration file location"

	FlagAccountID       = "account"
	FlagDescAccountID   = "cloud account ID"
	FlagClusterName     = "cluster"
	FlagDescClusterName = "kubernetes cluster name"
	FlagRegion          = "region"
	FlagDescRegion      = "deployment region matching the dimension in the CloudZero dashboard"

	FlagConfigFileWebhook    = "config-webhook"
	FlagConfigFileAggregator = "config-aggregator"
)
