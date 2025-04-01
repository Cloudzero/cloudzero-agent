// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

type Resources struct {
	Pods         bool `yaml:"pods" default:"true"`
	Namespaces   bool `yaml:"namespaces" default:"true"`
	Deployments  bool `yaml:"deployments" default:"false"`
	Jobs         bool `yaml:"jobs" default:"false"`
	CronJobs     bool `yaml:"cronjobs" default:"false"`     //nolint:tagliatelle // compatibility
	StatefulSets bool `yaml:"statefulsets" default:"false"` //nolint:tagliatelle // compatibility
	DaemonSets   bool `yaml:"daemonsets" default:"false"`   //nolint:tagliatelle // compatibility
	Nodes        bool `yaml:"nodes" default:"false"`
}
