// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/pkg/errors"
)

// Settings represents the configuration settings for the application.
type Settings struct {
	CloudAccountId string      `yaml:"cloud_account_id" env:"CLOUD_ACCOUNT_ID" env-description:"CSP account ID"`
	Region         string      `yaml:"region" env:"CSP_REGION" env-description:"cloud service provider region"`
	ClusterName    string      `yaml:"cluster_name" env:"CLUSTER_NAME" env-description:"name of the cluster to monitor"`
	Server         Server      `yaml:"server"`
	Certificate    Certificate `yaml:"certificate"`
	Logging        Logging     `yaml:"logging"`
	Database       Database    `yaml:"database"`
	Annotations    Annotations `yaml:"annotations"`
	Labels         Labels      `yaml:"labels"`
}

func NewSettings(configFiles ...string) (*Settings, error) {
	var cfg Settings
	for _, cfgFile := range configFiles {
		if cfgFile == "" {
			continue
		}

		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			return nil, errors.Wrap(err, fmt.Sprintf("no config %s", cfgFile))
		}

		err := cleanenv.ReadConfig(cfgFile, &cfg)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("config read %s", cfgFile))
		}
	}
	return &cfg, nil
}
