// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

type Versions struct {
	ChartVersion string `yaml:"chart_version" env:"CHART_VERSION" env-description:"Chart Version"`
	AgentVersion string `yaml:"agent_version" env:"AGENT_VERSION" env-description:"Agent Version"`
}

func (s *Versions) Validate() error {
	return nil
}
