// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package scout

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
)

// DetectConfiguration provides an easy way to automatically detect cloud
// provider information from configuration variables which may or may not
// already contain values.
//
// It is designed for use when loading configuration. When Scout is able to
// detect a value, it will override any customer-provided value (logging a
// warning if they differ). Customer-provided values are only used as a
// fallback when Scout cannot detect a value.
//
// If any of the fields are unable to be auto-detected AND are not provided,
// an error will be returned. If all required fields are already provided,
// detection failures will only result in warning logs.
//
// If all parameters are nil, no detection will be performed.
// If logger is nil, no warning logs will be emitted.
func DetectConfiguration(ctx context.Context, logger *zerolog.Logger, scout types.Scout, region *string, accountID *string, clusterName *string) error {
	// If all parameters are nil, there's nothing to detect or compare
	if region == nil && accountID == nil && clusterName == nil {
		return nil
	}

	if scout == nil {
		scout = NewScout()
	}

	// Check if any values need to be detected (are empty)
	needsDetection := (region != nil && *region == "") || (accountID != nil && *accountID == "") || (clusterName != nil && *clusterName == "")

	// Always try to run the scout to allow comparison and warning logs
	ei, innerErr := scout.EnvironmentInfo(ctx)
	if innerErr != nil {
		// If detection fails but no values need to be filled, just log a warning
		if !needsDetection {
			if logger != nil {
				logger.Warn().Str("detection_failure", innerErr.Error()).Msg("cloud provider detection failed, but all required values are already provided")
			}
			return nil
		}
		return fmt.Errorf("failed to detect cloud provider: %w. Manual configuration (setting cloudAccountId, region, and clusterName in the Helm chart) may be required", innerErr)
	}

	if ei.CloudProvider == types.CloudProviderUnknown {
		// If detection fails but no values need to be filled, just log a warning
		if !needsDetection {
			if logger != nil {
				logger.Warn().Msg("cloud provider could not be auto-detected, but all required values are already provided")
			}
			return nil
		}
		return errors.New("cloud provider could not be auto-detected, manual configuration (setting cloudAccountId, region, and clusterName in the Helm chart) may be required")
	}

	// Handle region configuration
	if region != nil {
		if ei.Region != "" {
			// Detected value available - use it (override customer-provided if different)
			if *region != "" && *region != ei.Region {
				// Log warning when overriding customer-provided value
				if logger != nil {
					logger.Warn().
						Str("provided", *region).
						Str("detected", ei.Region).
						Msg("provided region does not match detected region; using detected value")
				}
			}
			*region = ei.Region
		} else if *region == "" {
			// Not detected AND not provided - error
			return errors.New("region could not be auto-detected, manual configuration (setting region in the Helm chart) may be required")
		}
		// If detected is empty but customer provided: keep customer value (implicit)
	}

	// Handle account ID configuration
	if accountID != nil {
		if ei.AccountID != "" {
			// Detected value available - use it (override customer-provided if different)
			if *accountID != "" && *accountID != ei.AccountID {
				// Log warning when overriding customer-provided value
				if logger != nil {
					logger.Warn().
						Str("provided", *accountID).
						Str("detected", ei.AccountID).
						Msg("provided account ID does not match detected account ID; using detected value")
				}
			}
			*accountID = ei.AccountID
		} else if *accountID == "" {
			// Not detected AND not provided - error
			return errors.New("account ID could not be auto-detected, manual configuration (setting cloudAccountId in the Helm chart) may be required")
		}
		// If detected is empty but customer provided: keep customer value (implicit)
	}

	// Handle cluster name configuration
	if clusterName != nil {
		if ei.ClusterName != "" {
			// Detected value available - use it (override customer-provided if different)
			if *clusterName != "" && *clusterName != ei.ClusterName {
				// Log warning when overriding customer-provided value
				if logger != nil {
					logger.Warn().
						Str("provided", *clusterName).
						Str("detected", ei.ClusterName).
						Msg("provided cluster name does not match detected cluster name; using detected value")
				}
			}
			*clusterName = ei.ClusterName
		} else if *clusterName == "" {
			// Not detected AND not provided - error
			return errors.New("cluster name could not be auto-detected, manual configuration (setting clusterName in the Helm chart) may be required")
		}
		// If detected is empty but customer provided: keep customer value (implicit)
	}

	return nil
}
