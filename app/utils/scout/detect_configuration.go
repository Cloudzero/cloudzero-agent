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
// It is designed for use when loading configuration. If any fields are empty,
// an auto scout (see the auto subpackage) will be used to attempt to detect the
// correct values. If the fields are not empty, they will be treated as
// overrides and left intact, but a warning will be logged if they don't match
// the detected values.
//
// If any of the fields are unable to be auto-detected AND are required (empty),
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
		return fmt.Errorf("failed to detect cloud provider: %w", innerErr)
	}

	if ei.CloudProvider == types.CloudProviderUnknown {
		// If detection fails but no values need to be filled, just log a warning
		if !needsDetection {
			if logger != nil {
				logger.Warn().Msg("cloud provider could not be auto-detected, but all required values are already provided")
			}
			return nil
		}
		return errors.New("cloud provider could not be auto-detected, manual configuration may be required")
	}

	// Handle region configuration
	if region != nil {
		if *region == "" {
			// Empty field - set with detected value
			if ei.Region == "" {
				return errors.New("region could not be auto-detected, manual configuration may be required")
			}
			*region = ei.Region
		} else if ei.Region != "" && *region != ei.Region {
			// Non-empty field that doesn't match detected value - log warning
			if logger != nil {
				logger.Warn().
					Str("provided", *region).
					Str("detected", ei.Region).
					Msg("provided region does not match detected region")
			}
		}
	}

	// Handle account ID configuration
	if accountID != nil {
		if *accountID == "" {
			// Empty field - set with detected value
			if ei.AccountID == "" {
				return errors.New("account ID could not be auto-detected, manual configuration may be required")
			}
			*accountID = ei.AccountID
		} else if ei.AccountID != "" && *accountID != ei.AccountID {
			// Non-empty field that doesn't match detected value - log warning
			if logger != nil {
				logger.Warn().
					Str("provided", *accountID).
					Str("detected", ei.AccountID).
					Msg("provided account ID does not match detected account ID")
			}
		}
	}

	// Handle cluster name configuration
	if clusterName != nil {
		if *clusterName == "" {
			// Empty field - set with detected value
			if ei.ClusterName == "" {
				return errors.New("cluster name could not be auto-detected, manual configuration may be required")
			}
			*clusterName = ei.ClusterName
		} else if ei.ClusterName != "" && *clusterName != ei.ClusterName {
			// Non-empty field that doesn't match detected value - log warning
			if logger != nil {
				logger.Warn().
					Str("provided", *clusterName).
					Str("detected", ei.ClusterName).
					Msg("provided cluster name does not match detected cluster name")
			}
		}
	}

	return nil
}
