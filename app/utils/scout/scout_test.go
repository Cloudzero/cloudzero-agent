package scout

import (
	"context"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/cloudzero/cloudzero-agent/app/utils/scout/auto"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types/mocks"
)

func TestNewScout(t *testing.T) {
	// Test default auto-detection scout
	scout := NewScout()
	autoScout, ok := scout.(*auto.Scout)
	if !ok {
		t.Error("Expected auto.Scout implementation")
	}
	// We can't easily test the internal scouts array without exposing it,
	// but we can test that it returns a valid auto.Scout
	if autoScout == nil {
		t.Error("Expected non-nil auto.Scout")
	}
}

func TestAutoScoutEnvironmentInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInfo := &types.EnvironmentInfo{
		CloudProvider: types.CloudProviderAWS,
		Region:        "test-region",
		AccountID:     "test-account",
	}

	// Create mock scout that returns detection and info
	mockScout := mocks.NewMockScout(ctrl)
	mockScout.EXPECT().
		Detect(gomock.Any()).
		Return(types.CloudProviderAWS, nil)
	mockScout.EXPECT().
		EnvironmentInfo(gomock.Any()).
		Return(mockInfo, nil)

	// Create auto scout with single mock scout
	autoScout := auto.NewScout(mockScout)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := autoScout.EnvironmentInfo(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if info.CloudProvider != types.CloudProviderAWS {
		t.Errorf("Expected cloud provider 'aws', got: %s", info.CloudProvider)
	}

	if info.Region != "test-region" {
		t.Errorf("Expected region 'test-region', got: %s", info.Region)
	}

	if info.AccountID != "test-account" {
		t.Errorf("Expected account ID 'test-account', got: %s", info.AccountID)
	}
}

func TestAutoScoutDetect(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test successful detection
	mockScout := mocks.NewMockScout(ctrl)
	mockScout.EXPECT().
		Detect(gomock.Any()).
		Return(types.CloudProviderAWS, nil)

	autoScout := auto.NewScout(mockScout)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	detected, err := autoScout.Detect(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if detected != types.CloudProviderAWS {
		t.Errorf("Expected cloud provider 'aws', got: %s", detected)
	}
}

func TestAutoScoutNoDetection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock scouts that don't detect anything
	mockScout1 := mocks.NewMockScout(ctrl)
	mockScout1.EXPECT().
		Detect(gomock.Any()).
		Return(types.CloudProviderUnknown, nil).
		AnyTimes() // Allow multiple calls since EnvironmentInfo also calls Detect

	mockScout2 := mocks.NewMockScout(ctrl)
	mockScout2.EXPECT().
		Detect(gomock.Any()).
		Return(types.CloudProviderUnknown, nil).
		AnyTimes() // Allow multiple calls since EnvironmentInfo also calls Detect

	autoScout := auto.NewScout(mockScout1, mockScout2)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test Detect method
	detected, err := autoScout.Detect(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if detected != types.CloudProviderUnknown {
		t.Errorf("Expected cloud provider 'unknown', got: %s", detected)
	}

	// Test EnvironmentInfo method
	_, err = autoScout.EnvironmentInfo(ctx)
	if err != auto.ErrNoCloudProviderDetected {
		t.Errorf("Expected auto.ErrNoCloudProviderDetected, got: %v", err)
	}
}
