// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package scout_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/cloudzero/cloudzero-agent/app/utils/scout"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types/mocks"
)

func TestDetectConfiguration_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name                string
		inputRegion         *string
		inputAccountID      *string
		inputClusterName    *string
		expectedRegion      string
		expectedAccountID   string
		expectedClusterName string
		environmentInfo     *types.EnvironmentInfo
		envInfoError        error
	}{
		{
			name:                "detect all three values",
			inputRegion:         stringPtr(""),
			inputAccountID:      stringPtr(""),
			inputClusterName:    stringPtr(""),
			expectedRegion:      "us-west-2",
			expectedAccountID:   "123456789012",
			expectedClusterName: "test-cluster",
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "us-west-2",
				AccountID:     "123456789012",
				ClusterName:   "test-cluster",
			},
		},
		{
			name:                "detect only region and cluster when account is already set",
			inputRegion:         stringPtr(""),
			inputAccountID:      stringPtr("existing-account"),
			inputClusterName:    stringPtr(""),
			expectedRegion:      "eu-west-1",
			expectedAccountID:   "existing-account",
			expectedClusterName: "detected-cluster",
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "eu-west-1",
				AccountID:     "987654321098",
				ClusterName:   "detected-cluster",
			},
		},
		{
			name:                "detect only account and cluster when region is already set",
			inputRegion:         stringPtr("existing-region"),
			inputAccountID:      stringPtr(""),
			inputClusterName:    stringPtr(""),
			expectedRegion:      "existing-region",
			expectedAccountID:   "detected-account",
			expectedClusterName: "production-cluster",
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "us-central1",
				AccountID:     "detected-account",
				ClusterName:   "production-cluster",
			},
		},
		{
			name:                "detect only cluster when region and account are already set",
			inputRegion:         stringPtr("existing-region"),
			inputAccountID:      stringPtr("existing-account"),
			inputClusterName:    stringPtr(""),
			expectedRegion:      "existing-region",
			expectedAccountID:   "existing-account",
			expectedClusterName: "staging-cluster",
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "us-west-2",
				AccountID:     "123456789012",
				ClusterName:   "staging-cluster",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockScout := mocks.NewMockScout(ctrl)

			// Set up expectations for EnvironmentInfo if needed
			if (tt.inputRegion != nil && *tt.inputRegion == "") || (tt.inputAccountID != nil && *tt.inputAccountID == "") || (tt.inputClusterName != nil && *tt.inputClusterName == "") {
				mockScout.EXPECT().
					EnvironmentInfo(gomock.Any()).
					Return(tt.environmentInfo, tt.envInfoError)
			}

			// Call the actual function with mock scout
			err := scout.DetectConfiguration(context.Background(), mockScout, tt.inputRegion, tt.inputAccountID, tt.inputClusterName)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			// Verify results
			if tt.inputRegion != nil && *tt.inputRegion != tt.expectedRegion {
				t.Errorf("Expected region %s, got %s", tt.expectedRegion, *tt.inputRegion)
			}

			if tt.inputAccountID != nil && *tt.inputAccountID != tt.expectedAccountID {
				t.Errorf("Expected accountID %s, got %s", tt.expectedAccountID, *tt.inputAccountID)
			}

			if tt.inputClusterName != nil && *tt.inputClusterName != tt.expectedClusterName {
				t.Errorf("Expected clusterName %s, got %s", tt.expectedClusterName, *tt.inputClusterName)
			}
		})
	}
}

func TestDetectConfiguration_DetectionFailures(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name             string
		inputRegion      *string
		inputAccountID   *string
		inputClusterName *string
		environmentInfo  *types.EnvironmentInfo
		envInfoError     error
		expectedErrorMsg string
	}{
		{
			name:             "environment info fails",
			inputRegion:      stringPtr(""),
			inputAccountID:   stringPtr(""),
			inputClusterName: stringPtr(""),
			envInfoError:     errors.New("environment info failed"),
			expectedErrorMsg: "failed to detect cloud provider: environment info failed",
		},
		{
			name:             "cloud provider unknown after detection",
			inputRegion:      stringPtr(""),
			inputAccountID:   nil,
			inputClusterName: stringPtr(""),
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderUnknown,
				Region:        "us-west-2",
				ClusterName:   "test-cluster",
			},
			expectedErrorMsg: "cloud provider could not be auto-detected, manual configuration may be required",
		},
		{
			name:             "region detection fails when required",
			inputRegion:      stringPtr(""),
			inputAccountID:   nil,
			inputClusterName: nil,
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "", // Empty region should cause error
				AccountID:     "123456789012",
				ClusterName:   "test-cluster",
			},
			expectedErrorMsg: "region could not be auto-detected, manual configuration may be required",
		},
		{
			name:             "account ID detection fails when required",
			inputRegion:      nil,
			inputAccountID:   stringPtr(""),
			inputClusterName: nil,
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "us-west-2",
				AccountID:     "", // Empty account ID should cause error
				ClusterName:   "test-cluster",
			},
			expectedErrorMsg: "account ID could not be auto-detected, manual configuration may be required",
		},
		{
			name:             "cluster name detection fails when required",
			inputRegion:      nil,
			inputAccountID:   nil,
			inputClusterName: stringPtr(""),
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "us-west-2",
				AccountID:     "123456789012",
				ClusterName:   "", // Empty cluster name should cause error
			},
			expectedErrorMsg: "cluster name could not be auto-detected, manual configuration may be required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockScout := mocks.NewMockScout(ctrl)

			// Set up expectations
			mockScout.EXPECT().
				EnvironmentInfo(gomock.Any()).
				Return(tt.environmentInfo, tt.envInfoError)

			// Call the actual function with mock scout
			err := scout.DetectConfiguration(context.Background(), mockScout, tt.inputRegion, tt.inputAccountID, tt.inputClusterName)

			if err == nil {
				t.Errorf("Expected error, but got none")
				return
			}

			if err.Error() != tt.expectedErrorMsg {
				t.Errorf("Expected error message %q, got %q", tt.expectedErrorMsg, err.Error())
			}
		})
	}
}

func TestDetectConfiguration_EnsurePropertiesAlwaysSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name                string
		inputRegion         *string
		inputAccountID      *string
		inputClusterName    *string
		environmentInfo     *types.EnvironmentInfo
		expectError         bool
		expectedRegion      string
		expectedAccountID   string
		expectedClusterName string
	}{
		{
			name:             "all values must be set on success",
			inputRegion:      stringPtr(""),
			inputAccountID:   stringPtr(""),
			inputClusterName: stringPtr(""),
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "us-east-1",
				AccountID:     "111111111111",
				ClusterName:   "production-cluster",
			},
			expectError:         false,
			expectedRegion:      "us-east-1",
			expectedAccountID:   "111111111111",
			expectedClusterName: "production-cluster",
		},
		{
			name:             "region and cluster must be set when requested",
			inputRegion:      stringPtr(""),
			inputAccountID:   nil,
			inputClusterName: stringPtr(""),
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "us-west-1",
				AccountID:     "222222222222",
				ClusterName:   "staging-cluster",
			},
			expectError:         false,
			expectedRegion:      "us-west-1",
			expectedClusterName: "staging-cluster",
		},
		{
			name:             "account and cluster must be set when requested",
			inputRegion:      nil,
			inputAccountID:   stringPtr(""),
			inputClusterName: stringPtr(""),
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "us-west-2",
				AccountID:     "333333333333",
				ClusterName:   "development-cluster",
			},
			expectError:         false,
			expectedAccountID:   "333333333333",
			expectedClusterName: "development-cluster",
		},
		{
			name:             "cluster only must be set when requested",
			inputRegion:      nil,
			inputAccountID:   nil,
			inputClusterName: stringPtr(""),
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "eu-central-1",
				AccountID:     "444444444444",
				ClusterName:   "test-cluster",
			},
			expectError:         false,
			expectedClusterName: "test-cluster",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockScout := mocks.NewMockScout(ctrl)

			// Set up expectations
			if (tt.inputRegion != nil && *tt.inputRegion == "") || (tt.inputAccountID != nil && *tt.inputAccountID == "") || (tt.inputClusterName != nil && *tt.inputClusterName == "") {
				mockScout.EXPECT().
					EnvironmentInfo(gomock.Any()).
					Return(tt.environmentInfo, nil)
			}

			// Call the actual function with mock scout
			err := scout.DetectConfiguration(context.Background(), mockScout, tt.inputRegion, tt.inputAccountID, tt.inputClusterName)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			// CRITICAL: Verify that region, accountID, and clusterName are ALWAYS set when non-nil pointers are provided
			if tt.inputRegion != nil {
				if *tt.inputRegion == "" {
					t.Error("Region pointer was provided but not set after DetectConfiguration call")
				}
				if tt.expectedRegion != "" && *tt.inputRegion != tt.expectedRegion {
					t.Errorf("Expected region %q, got %q", tt.expectedRegion, *tt.inputRegion)
				}
			}

			if tt.inputAccountID != nil {
				if *tt.inputAccountID == "" {
					t.Error("AccountID pointer was provided but not set after DetectConfiguration call")
				}
				if tt.expectedAccountID != "" && *tt.inputAccountID != tt.expectedAccountID {
					t.Errorf("Expected accountID %q, got %q", tt.expectedAccountID, *tt.inputAccountID)
				}
			}

			if tt.inputClusterName != nil {
				if *tt.inputClusterName == "" {
					t.Error("ClusterName pointer was provided but not set after DetectConfiguration call")
				}
				if tt.expectedClusterName != "" && *tt.inputClusterName != tt.expectedClusterName {
					t.Errorf("Expected clusterName %q, got %q", tt.expectedClusterName, *tt.inputClusterName)
				}
			}
		})
	}
}

func TestDetectConfiguration_NilPointers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Test that nil pointers are handled correctly and don't cause detection attempts
	mockScout := mocks.NewMockScout(ctrl)

	// Should not call any mock methods since all pointers are nil
	err := scout.DetectConfiguration(context.Background(), mockScout, nil, nil, nil)
	if err != nil {
		t.Errorf("Expected no error when all pointers are nil, got: %v", err)
	}
}

func TestDetectConfiguration_PartialDetection(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockScout := mocks.NewMockScout(ctrl)

	region := "existing-region"       // Already set
	accountID := "existing-account"   // Already set
	clusterName := "existing-cluster" // Already set

	err := scout.DetectConfiguration(context.Background(), mockScout, &region, &accountID, &clusterName)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Values should remain unchanged
	if region != "existing-region" {
		t.Errorf("Expected region to remain unchanged, got: %s", region)
	}
	if accountID != "existing-account" {
		t.Errorf("Expected accountID to remain unchanged, got: %s", accountID)
	}
	if clusterName != "existing-cluster" {
		t.Errorf("Expected clusterName to remain unchanged, got: %s", clusterName)
	}
}

func TestDetectConfiguration_CriticalRequirement(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name                  string
		inputRegion           *string
		inputAccountID        *string
		environmentInfo       *types.EnvironmentInfo
		envInfoError          error
		expectError           bool
		expectRegionSet       bool
		expectAccountIDSet    bool
		expectedErrorContains string
	}{
		{
			name:           "SUCCESS: both region and accountID detected and set",
			inputRegion:    stringPtr(""),
			inputAccountID: stringPtr(""),
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "detected-region",
				AccountID:     "detected-account",
			},
			expectError:        false,
			expectRegionSet:    true,
			expectAccountIDSet: true,
		},
		{
			name:           "SUCCESS: only region detected and set",
			inputRegion:    stringPtr(""),
			inputAccountID: nil,
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "detected-region",
				AccountID:     "some-account",
			},
			expectError:        false,
			expectRegionSet:    true,
			expectAccountIDSet: false,
		},
		{
			name:           "SUCCESS: only accountID detected and set",
			inputRegion:    nil,
			inputAccountID: stringPtr(""),
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "some-region",
				AccountID:     "detected-account",
			},
			expectError:        false,
			expectRegionSet:    false,
			expectAccountIDSet: true,
		},
		{
			name:           "FAIL: region required but cannot be detected - MUST return error",
			inputRegion:    stringPtr(""),
			inputAccountID: nil,
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "", // Empty - cannot be detected
				AccountID:     "some-account",
			},
			expectError:           true,
			expectRegionSet:       false,
			expectAccountIDSet:    false,
			expectedErrorContains: "region could not be auto-detected",
		},
		{
			name:           "FAIL: accountID required but cannot be detected - MUST return error",
			inputRegion:    nil,
			inputAccountID: stringPtr(""),
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "some-region",
				AccountID:     "", // Empty - cannot be detected
			},
			expectError:           true,
			expectRegionSet:       false,
			expectAccountIDSet:    false,
			expectedErrorContains: "account ID could not be auto-detected",
		},
		{
			name:           "FAIL: both required but region cannot be detected - MUST return error",
			inputRegion:    stringPtr(""),
			inputAccountID: stringPtr(""),
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "", // Empty - cannot be detected
				AccountID:     "detected-account",
			},
			expectError:           true,
			expectRegionSet:       false,
			expectAccountIDSet:    false,
			expectedErrorContains: "region could not be auto-detected",
		},
		{
			name:           "FAIL: both required but accountID cannot be detected - MUST return error",
			inputRegion:    stringPtr(""),
			inputAccountID: stringPtr(""),
			environmentInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAWS,
				Region:        "detected-region",
				AccountID:     "", // Empty - cannot be detected
			},
			expectError:           true,
			expectRegionSet:       true, // Region gets set BEFORE accountID check fails
			expectAccountIDSet:    false,
			expectedErrorContains: "account ID could not be auto-detected",
		},
		{
			name:                  "FAIL: environment info fails completely - MUST return error",
			inputRegion:           stringPtr(""),
			inputAccountID:        stringPtr(""),
			envInfoError:          errors.New("metadata service unavailable"),
			expectError:           true,
			expectRegionSet:       false,
			expectAccountIDSet:    false,
			expectedErrorContains: "failed to detect cloud provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockScout := mocks.NewMockScout(ctrl)

			// Set up mock expectations
			mockScout.EXPECT().
				EnvironmentInfo(gomock.Any()).
				Return(tt.environmentInfo, tt.envInfoError)

			// Create copies of the input values for testing
			var regionCopy, accountIDCopy, clusterNameCopy *string
			if tt.inputRegion != nil {
				regionCopy = stringPtr(*tt.inputRegion)
			}
			if tt.inputAccountID != nil {
				accountIDCopy = stringPtr(*tt.inputAccountID)
			}
			// Note: We don't test clusterName in this critical test since it focuses on the original region/account requirements
			// but we need to provide a parameter for the function signature

			// Call the actual function with mock scout
			err := scout.DetectConfiguration(context.Background(), mockScout, regionCopy, accountIDCopy, clusterNameCopy)

			// CRITICAL VALIDATION: Error handling
			if tt.expectError {
				if err == nil {
					t.Fatalf("CRITICAL FAILURE: Expected error when detection fails, but got success. This violates the requirement that errors must be returned when properties cannot be detected.")
				}
				if tt.expectedErrorContains != "" && !contains(err.Error(), tt.expectedErrorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.expectedErrorContains, err.Error())
				}

				// Validate that properties are set/not set according to expectations
				// Note: The actual implementation may set some properties before failing on others
				if tt.expectRegionSet {
					if regionCopy == nil || *regionCopy == "" {
						t.Errorf("Expected region to be set during partial success, got %q", getStringValue(regionCopy))
					}
				} else {
					if regionCopy != nil && *regionCopy != "" {
						t.Errorf("Expected region to remain empty when detection fails early, got %q", *regionCopy)
					}
				}

				if tt.expectAccountIDSet {
					if accountIDCopy == nil || *accountIDCopy == "" {
						t.Errorf("Expected accountID to be set during partial success, got %q", getStringValue(accountIDCopy))
					}
				} else {
					if accountIDCopy != nil && *accountIDCopy != "" {
						t.Errorf("Expected accountID to remain empty when detection fails, got %q", *accountIDCopy)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("Expected success but got error: %v", err)
				}

				// CRITICAL VALIDATION: Properties must ALWAYS be set on success
				if tt.expectRegionSet {
					if regionCopy == nil || *regionCopy == "" {
						t.Errorf("CRITICAL FAILURE: Region pointer was provided but not set after successful DetectConfiguration call. Expected non-empty value, got %q", getStringValue(regionCopy))
					}
				}

				if tt.expectAccountIDSet {
					if accountIDCopy == nil || *accountIDCopy == "" {
						t.Errorf("CRITICAL FAILURE: AccountID pointer was provided but not set after successful DetectConfiguration call. Expected non-empty value, got %q", getStringValue(accountIDCopy))
					}
				}
			}
		})
	}
}

func TestDetectConfiguration_NilScout(t *testing.T) {
	// Test that passing nil scout uses the default scout
	region := ""
	accountID := ""
	clusterName := ""

	// This should not panic and should use the default scout
	// We don't expect it to succeed since there's no real cloud environment,
	// but it should attempt detection without panicking
	err := scout.DetectConfiguration(context.Background(), nil, &region, &accountID, &clusterName)

	// We expect some kind of error since we're not in a real cloud environment
	// The important thing is that it doesn't panic
	if err == nil {
		t.Log("Unexpectedly succeeded - might be running in a cloud environment")
	} else {
		t.Logf("Expected error when using default scout: %v", err)
	}
}

func TestDetectConfiguration_NoDetectionWhenValuesProvided(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// This is the new test requested by the user:
	// Ensure Scout is not invoked when all values are already provided
	mockScout := mocks.NewMockScout(ctrl)

	// NO mock expectations should be set - if Scout methods are called, the test will fail

	region := "pre-set-region"
	accountID := "pre-set-account"
	clusterName := "pre-set-cluster"

	err := scout.DetectConfiguration(context.Background(), mockScout, &region, &accountID, &clusterName)
	if err != nil {
		t.Errorf("Expected no error when values are pre-set, got: %v", err)
	}

	// Verify values weren't changed
	if region != "pre-set-region" {
		t.Errorf("Expected region to remain unchanged, got: %s", region)
	}
	if accountID != "pre-set-account" {
		t.Errorf("Expected accountID to remain unchanged, got: %s", accountID)
	}
	if clusterName != "pre-set-cluster" {
		t.Errorf("Expected clusterName to remain unchanged, got: %s", clusterName)
	}
}

// Helper functions for creating pointers
func stringPtr(s string) *string {
	return &s
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}())
}

// Helper function to get string value for error messages
func getStringValue(ptr *string) string {
	if ptr == nil {
		return "<nil>"
	}
	return *ptr
}
