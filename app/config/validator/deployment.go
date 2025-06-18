// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import "github.com/pkg/errors"

type Deployment struct {
	AccountID   string `yaml:"account_id" env:"ACCOUNT_ID" required:"true" env-description:"AWS Account ID"`
	ClusterName string `yaml:"cluster_name" env:"CLUSTER_NAME" required:"true" env-description:"Cluster Name"`
	Region      string `yaml:"region" env:"REGION" required:"true" env-description:"AWS Region"`
}

func (s *Deployment) Validate() error {
	if s.AccountID == "" {
		return errors.New(ErrNoAccountIDMsg)
	}

	if s.ClusterName == "" {
		return errors.New(ErrNoClusterNameMsg)
	}

	if s.Region == "" {
		return errors.New(ErrNoRegionMsg)
	}

	return nil
}
