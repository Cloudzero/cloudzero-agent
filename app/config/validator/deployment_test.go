// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types/mocks"
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
		setupMock  func(*gomock.Controller) *mocks.MockScout
	}{
		{
			name: "ValidDeployment",
			deployment: &config.Deployment{
				AccountID:   accountID,
				ClusterName: clusterID,
				Region:      region,
			},
			wantErr: false,
			setupMock: func(ctrl *gomock.Controller) *mocks.MockScout {
				// Valid deployment shouldn't need auto-detection
				return nil
			},
		},
		{
			name: "MissingAccountID",
			deployment: &config.Deployment{
				ClusterName: clusterID,
				Region:      region,
			},
			wantErr: true,
			setupMock: func(ctrl *gomock.Controller) *mocks.MockScout {
				mockScout := mocks.NewMockScout(ctrl)
				// Mock returns detection failure
				mockScout.EXPECT().EnvironmentInfo(gomock.Any()).Return(nil, fmt.Errorf("detection failed"))
				return mockScout
			},
		},
		{
			name: "MissingClusterName",
			deployment: &config.Deployment{
				AccountID: accountID,
				Region:    region,
			},
			wantErr: true,
			setupMock: func(ctrl *gomock.Controller) *mocks.MockScout {
				mockScout := mocks.NewMockScout(ctrl)
				// Mock returns detection failure
				mockScout.EXPECT().EnvironmentInfo(gomock.Any()).Return(nil, fmt.Errorf("detection failed"))
				return mockScout
			},
		},
		{
			name: "MissingRegion",
			deployment: &config.Deployment{
				AccountID:   accountID,
				ClusterName: clusterID,
			},
			wantErr: true,
			setupMock: func(ctrl *gomock.Controller) *mocks.MockScout {
				mockScout := mocks.NewMockScout(ctrl)
				// Mock returns detection failure
				mockScout.EXPECT().EnvironmentInfo(gomock.Any()).Return(nil, fmt.Errorf("detection failed"))
				return mockScout
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			if tt.setupMock != nil {
				if mockScout := tt.setupMock(ctrl); mockScout != nil {
					// Use reflection to set the private scout field for testing
					// Since scout field is private, we need to create a new deployment with the mock
					deploymentWithMock := &config.Deployment{
						AccountID:   tt.deployment.AccountID,
						ClusterName: tt.deployment.ClusterName,
						Region:      tt.deployment.Region,
					}
					// We need to set the scout field - let's add a method to do this
					deploymentWithMock.SetScout(mockScout)
					tt.deployment = deploymentWithMock
				}
			}

			err := tt.deployment.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
