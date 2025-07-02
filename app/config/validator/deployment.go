// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-agent/app/utils/scout"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
)

type Deployment struct {
	AccountID   string      `yaml:"account_id" env:"ACCOUNT_ID" required:"true" env-description:"AWS Account ID"`
	ClusterName string      `yaml:"cluster_name" env:"CLUSTER_NAME" required:"true" env-description:"Cluster Name"`
	Region      string      `yaml:"region" env:"REGION" required:"true" env-description:"AWS Region"`
	scout       types.Scout `yaml:"-" env:"-" env-description:"Scout"`
}

// SetScout provides a way to specify a scout. By default, an auto scout will be
// used.
func (s *Deployment) SetScout(scout types.Scout) {
	s.scout = scout
}

func (s *Deployment) Validate() error {
	// Trim whitespace from all fields
	s.AccountID = strings.TrimSpace(s.AccountID)
	s.ClusterName = strings.TrimSpace(s.ClusterName)
	s.Region = strings.TrimSpace(s.Region)

	// Auto-detect cloud account ID and region if needed
	logger := log.Logger.With().Str("component", "validator-deployment").Logger()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := scout.DetectConfiguration(ctx, &logger, s.scout, &s.Region, &s.AccountID, &s.ClusterName)
	if err != nil {
		return errors.Wrap(err, "failed to auto-detect cloud environment")
	}

	if s.ClusterName == "" {
		return errors.New(ErrNoClusterNameMsg)
	}

	return nil
}
