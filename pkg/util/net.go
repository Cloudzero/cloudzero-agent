// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package util contains generic utility methods
package util

import (
	"errors"
	"fmt"
	"net/url"
)

// ExtractHostnameFromURL parses a URL string and returns the hostname part
func ExtractHostnameFromURL(rawURL string) (string, error) {
	if len(rawURL) == 0 {
		return "", errors.New("URL is empty")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("%s invalid URL: %w", rawURL, err)
	}
	return parsedURL.Hostname(), nil
}
