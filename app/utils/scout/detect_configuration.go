// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package scout

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
)

// DetectConfiguration provides an easy way to automatically detect cloud
// provider information from configuration variables which may or may not
// already contain values.
//
// It is designed for use when loading configuration. If any fields are empty,
// an auto scout (see the auto subpackage) will be used to attempt to detect the
// correct values. If the fields are not empty, they will be treated as
// overrides and left intact.
//
// If any of the fields are unable to be auto-detected, an error will be
// returned.
func DetectConfiguration(ctx context.Context, scout types.Scout, region *string, accountID *string, clusterName *string) error {
	if scout == nil {
		scout = NewScout()
	}

	if (region != nil && *region == "") || (accountID != nil && *accountID == "") || (clusterName != nil && *clusterName == "") {
		ei, innerErr := scout.EnvironmentInfo(ctx)
		if innerErr != nil {
			return fmt.Errorf("failed to detect cloud provider: %w", innerErr)
		} else if ei.CloudProvider == types.CloudProviderUnknown {
			return errors.New("cloud provider could not be auto-detected, manual configuration may be required")
		}

		if region != nil && *region == "" {
			if ei.Region == "" {
				return errors.New("region could not be auto-detected, manual configuration may be required")
			}
			*region = ei.Region
		}

		if accountID != nil && *accountID == "" {
			if ei.AccountID == "" {
				return errors.New("account ID could not be auto-detected, manual configuration may be required")
			}
			*accountID = ei.AccountID
		}

		if clusterName != nil && *clusterName == "" {
			if ei.ClusterName == "" {
				return errors.New("cluster name could not be auto-detected, manual configuration may be required")
			}
			*clusterName = ei.ClusterName
		}
	}

	return nil
}
