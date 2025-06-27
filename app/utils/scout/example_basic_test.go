// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package scout_test

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types/mocks"
)

// Example_basic demonstrates basic usage of Scout for automatic cloud environment detection.
// This example uses a mock scout to provide deterministic output.
func Example_basic() {
	// Create gomock controller
	ctrl := gomock.NewController(nil) // In real tests, pass testing.T
	defer ctrl.Finish()

	// Create mock environment info for deterministic output
	mockInfo := &types.EnvironmentInfo{
		CloudProvider: types.CloudProviderAWS,
		Region:        "us-east-1",
		AccountID:     "123456789012",
	}

	// Create mock scout with expectations
	mockScout := mocks.NewMockScout(ctrl)
	mockScout.EXPECT().
		EnvironmentInfo(gomock.Any()).
		Return(mockInfo, nil)

	// Use mock scout directly (for deterministic examples)
	s := mockScout

	// Set timeout for metadata retrieval
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get environment information
	info, err := s.EnvironmentInfo(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Cloud Provider: %s\n", info.CloudProvider)
	fmt.Printf("Region: %s\n", info.Region)
	fmt.Printf("Account ID: %s\n", info.AccountID)

	// Output:
	// Cloud Provider: aws
	// Region: us-east-1
	// Account ID: 123456789012
}
