// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package scout

import (
	"context"
	"errors"
	"fmt"

	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
)

// DetectConfiguration provides an easy way to automatically detect cloud provider
// information. It is designed for use when loading configuration. If any fields
// are empty (but not nil), a scout will be used to attempt to detect the
// correct values. If, however, non-empty values are provided, detection will be
// elided and the supplied values left unaltered.
func DetectConfiguration(ctx context.Context, scout types.Scout, region *string, accountID *string) error {
	if scout == nil {
		scout = NewScout()
	}

	if (region != nil && *region == "") || (accountID != nil && *accountID == "") {
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
	}

	return nil
}
