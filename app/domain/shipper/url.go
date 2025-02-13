// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/rs/zerolog/log"
)

type PresignedURLAPIPayload struct {
	Files []*PresignedURLAPIPayloadFile `json:"files"`
}

type PresignedURLAPIPayloadFile struct {
	ReferenceID string `json:"reference_id"`      //nolint:tagliatelle // downstream expects cammel case
	SHA256      string `json:"sha_256,omitempty"` //nolint:tagliatelle // downstream expects cammel case
	Size        int64  `json:"size,omitempty"`
}

type AbandonAPIPayloadFile struct {
	ReferenceID string `json:"reference_id"` //nolint:tagliatelle // downstream expects cammel case
	Reason      string `json:"reason"`
}

// Allocates a set of pre-signed urls for the passed file objects
// The passed `files` argument will be modified to add the `PresignedURL` field
// You can opt to consume the return value or allow for implicit modification.
func (m *MetricShipper) AllocatePresignedURLs(files []*File) ([]*File, error) {
	if len(files) == 0 {
		return nil, nil
	}

	// create the payload with the files
	bodyFiles := make([]*PresignedURLAPIPayloadFile, len(files))
	for i, file := range files {
		bodyFiles[i] = &PresignedURLAPIPayloadFile{
			ReferenceID: file.ReferenceID,
		}
	}

	// create the http request body
	body := PresignedURLAPIPayload{Files: bodyFiles}

	// marshal to json
	enc, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode the body into json: %w", err)
	}

	// Create a new HTTP request
	uploadEndpoint := m.setting.Cloudzero.Host + "/upload"
	req, err := http.NewRequestWithContext(m.ctx, "POST", uploadEndpoint, bytes.NewBuffer(enc))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set necessary headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", m.setting.GetAPIKey())

	// Make sure we set the query parameters for count, expiration, cloud_account_id, region, cluster_name
	q := req.URL.Query()
	q.Add("count", strconv.Itoa(len(files)))
	q.Add("expiration", strconv.Itoa(expirationTime))
	q.Add("cloud_account_id", m.setting.CloudAccountID)
	q.Add("cluster_name", m.setting.ClusterName)
	q.Add("region", m.setting.Region)
	req.URL.RawQuery = q.Encode()

	log.Info().Msgf("Requesting %d presigned URLs from '%s' with key '%s'", len(files), req.URL.String(), m.setting.GetAPIKey())

	// Send the request
	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("HTTP request failed")
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrUnauthorized
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response
	var response map[string]string // map of: {ReferenceId: PresignedURL}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// validation
	if len(response) == 0 {
		return nil, ErrNoURLs
	}

	// create a map of {ReferenceId: File} to match api response
	fileMap := make(map[string]*File)
	for _, item := range files {
		fileMap[item.ReferenceID] = item
	}

	// ensure the same length
	if len(response) != len(fileMap) {
		return nil, fmt.Errorf("the length of the response did not match the request! files (%d) != urls (%d)", len(fileMap), len(response))
	}

	// set the pre-signed url value of the file and recompose the list
	// setting this value on the file reference will affect the base list
	// so we do not need to re-create the list and can simply return the list
	// passed as an argument
	for refid, url := range response {
		if file, ok := fileMap[refid]; ok {
			file.PresignedURL = url
		}
	}

	// TODO -- check for replay requests

	// check the metadata header

	// write into a []string of reference ids

	// save the reference ids to disk in a file

	return files, nil
}

// sends an abandon request for a list of files with a given reason
func (m *MetricShipper) AbandonFiles(referenceIDs []string, reason string) error {
	if len(referenceIDs) == 0 {
		return errors.New("cannot send in an empty slice")
	}

	// create the body
	body := make([]*AbandonAPIPayloadFile, len(referenceIDs))
	for i, item := range referenceIDs {
		body[i] = &AbandonAPIPayloadFile{
			ReferenceID: item,
			Reason:      reason,
		}
	}

	// serialize the body
	enc, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to encode the body: %w", err)
	}

	// Create a new HTTP request
	abandonEndpoint, err := m.setting.GetRemoteAPIBase()
	if err != nil {
		return fmt.Errorf("failed to get the abandon endpoint: %w", err)
	}
	abandonEndpoint.Path += "/abandon"
	req, err := http.NewRequestWithContext(m.ctx, "POST", abandonEndpoint.String(), bytes.NewBuffer(enc))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set necessary headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", m.setting.GetAPIKey())

	// Make sure we set the query parameters for count, cloud_account_id, region, cluster_name
	q := req.URL.Query()
	q.Add("count", strconv.Itoa(len(referenceIDs)))
	// q.Add("cluster_name", m.setting.ClusterName)
	// q.Add("cloud_account_id", m.setting.CloudAccountID)
	// q.Add("region", m.setting.Region)
	req.URL.RawQuery = q.Encode()

	log.Info().Msgf("Abandoning %d files from '%s'", len(referenceIDs), req.URL.String())

	// Send the request
	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("HTTP request failed")
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return ErrUnauthorized
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// success
	return nil
}
