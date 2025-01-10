// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package gh

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Release struct {
	Name string `json:"name"`
}

func GetLatestRelease(baseURL, owner, repo string) (string, error) {
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}

	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", baseURL, owner, repo)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get latest release: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var release Release
	if err := json.Unmarshal(body, &release); err != nil {
		return "", err
	}

	return release.Name, nil
}
