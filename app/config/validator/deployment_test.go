// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"testing"

	"github.com/cloudzero/cloudzero-agent/app/config/validator"
)

// Ensure these match the testdata/cloudzero-agent-validator.yml file
const (
	accountID = "000000000000"
	clusterID = "test-cluster"
	region    = "us-west-2"
)

func TestDeployment_Validate(t *testing.T) {
	tests := []struct {
		name       string
		deployment *config.Deployment
		wantErr    bool
	}{
		{
			name: "ValidDeployment",
			deployment: &config.Deployment{
				AccountID:   accountID,
				ClusterName: clusterID,
				Region:      region,
			},
			wantErr: false,
		},
		{
			name: "MissingAccountID",
			deployment: &config.Deployment{
				ClusterName: clusterID,
				Region:      region,
			},
			wantErr: true,
		},
		{
			name: "MissingClusterName",
			deployment: &config.Deployment{
				AccountID: accountID,
				Region:    region,
			},
			wantErr: true,
		},
		{
			name: "MissingRegion",
			deployment: &config.Deployment{
				AccountID:   accountID,
				ClusterName: clusterID,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.deployment.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
