// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"errors"
	"regexp"
	"strings"
)

var (
	validCloudAccountID = regexp.MustCompile(`^[a-zA-Z0-9-]+$`)
	// Regex Explanation:
	// ^[a-zA-Z0-9]            => Must start with an alphanumeric character.
	// ([a-zA-Z0-9-]{0,251}    => Can have 0 to 251 alphanumeric or hyphen characters in the middle.
	// [a-zA-Z0-9])?$          => Must end with an alphanumeric character.
	validClusterName = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,251}[a-zA-Z0-9])?$`)
)

// isValidCloudAccountID checks if the cloudAccountID contains only alphanumeric and hyphen characters.
func isValidCloudAccountID(cloudAccountID string) bool {
	return validCloudAccountID.MatchString(cloudAccountID)
}

// removeSurroundingQuotes removes any surrounding double quotes from the input string.
func removeSurroundingQuotes(s string) string {
	return strings.Trim(s, `"`)
}

// ValidateCloudAccountID validates the cloud_account_id.
// It attempts to clean the input by removing surrounding quotes if the initial validation fails.
// Returns the cleaned cloudAccountID and an error if validation fails.
func ValidateCloudAccountID(cloudAccountID string) (string, error) {
	if !isValidCloudAccountID(cloudAccountID) {
		cleanedAccountID := removeSurroundingQuotes(cloudAccountID)
		if isValidCloudAccountID(cleanedAccountID) {
			return cleanedAccountID, nil
		} else {
			message := "invalid format for 'cloud_account_id', can only include alphanumeric and hyphen characters"
			return "", errors.New(message)
		}
	}

	// If the original cloudAccountID is valid, return it as is.
	return cloudAccountID, nil
}

// isValidClusterName checks if the clusterName meets the specified format requirements.
// It must:
// - Contain only alphanumeric and hyphen characters.
// - Begin and end with an alphanumeric character.
// - Be no more than 253 characters long.
func isValidClusterName(clusterName string) bool {
	const maxLength = 253
	return validClusterName.MatchString(clusterName) && len(clusterName) <= maxLength
}

// ValidateClusterName validates the cluster_name.
// It returns an error if the clusterName does not meet the format requirements.
func ValidateClusterName(clusterName string) error {
	if !isValidClusterName(clusterName) {
		message := "invalid format for 'cluster_name', can only include alphanumeric and hyphen characters, " +
			"must begin and end with an alphanumeric, and contain no more than 253 characters"
		return errors.New(message)
	}
	return nil
}
