// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package auto

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types/mocks"
)

func TestNewScout(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockScout1 := mocks.NewMockScout(ctrl)
	mockScout2 := mocks.NewMockScout(ctrl)

	// Test creating auto scout with variadic arguments
	autoScout := NewScout(mockScout1, mockScout2)

	if autoScout == nil {
		t.Fatal("Expected non-nil auto scout")
	}

	if autoScout.scouts == nil {
		t.Fatal("Expected non-nil scouts slice")
	}

	if len(autoScout.scouts) != 2 {
		t.Errorf("Expected 2 scouts, got %d", len(autoScout.scouts))
	}
}

func TestScoutWithSingleProvider(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInfo := &types.EnvironmentInfo{
		CloudProvider: types.CloudProviderAWS,
		Region:        "us-east-1",
		AccountID:     "123456789012",
	}

	// Create a mock scout that successfully detects AWS
	awsScout := mocks.NewMockScout(ctrl)
	awsScout.EXPECT().
		Detect(gomock.Any()).
		Return(types.CloudProviderAWS, nil)
	awsScout.EXPECT().
		EnvironmentInfo(gomock.Any()).
		Return(mockInfo, nil)

	// Create auto scout with just AWS
	autoScout := NewScout(awsScout)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := autoScout.EnvironmentInfo(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if info.CloudProvider != types.CloudProviderAWS {
		t.Errorf("Expected AWS, got: %s", info.CloudProvider)
	}
}

func TestScoutWithMultipleProviders(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInfo := &types.EnvironmentInfo{
		CloudProvider: types.CloudProviderAWS,
		Region:        "us-west-2",
		AccountID:     "987654321098",
	}

	// First scout doesn't detect
	scout1 := mocks.NewMockScout(ctrl)
	scout1.EXPECT().
		Detect(gomock.Any()).
		Return(types.CloudProviderUnknown, nil)

	// Second scout (AWS) successfully detects
	awsScout := mocks.NewMockScout(ctrl)
	awsScout.EXPECT().
		Detect(gomock.Any()).
		Return(types.CloudProviderAWS, nil)
	awsScout.EXPECT().
		EnvironmentInfo(gomock.Any()).
		Return(mockInfo, nil)

	// Create auto scout with first scout that fails, then AWS that succeeds
	autoScout := NewScout(scout1, awsScout)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := autoScout.EnvironmentInfo(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if info.CloudProvider != types.CloudProviderAWS {
		t.Errorf("Expected AWS, got: %s", info.CloudProvider)
	}
}
