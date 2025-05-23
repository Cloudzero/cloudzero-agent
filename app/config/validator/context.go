// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"strings"
)

const (
	ContextStageNone       string = "none"
	ContextStageInit       string = "pre-start"
	ContextStageStart      string = "post-start"
	ContextStateConfigLoad string = "config-load"
	ContextStageStop       string = "pre-stop"
)

func IsValidStage(m string) bool {
	m = strings.ToLower(strings.TrimSpace(m))
	return m == ContextStageInit || m == ContextStageStart || m == ContextStageStop || m == ContextStateConfigLoad
}

type Context struct {
	Stage string `env:"CZ_CHECKER_STAGE" env-description:"Execution stage one of init, post-start, config-load, pre-stop, or all" default:"init"`
}

func (s *Context) Validate() error {
	s.Stage = strings.ToLower(strings.TrimSpace(s.Stage))
	return nil
}
