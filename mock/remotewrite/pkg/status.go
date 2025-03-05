// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package remotewrite

import (
	"encoding/json"
	"net/http"
)

// Handle cluster status data upload
func (rw *RemoteWrite) status(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check required query parameters
	err := checkRequiredParams(r, "cluster_name", "cloud_account_id")
	if err != nil {
		writeAPIResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Check if there's a body
	if r.Body == nil {
		writeAPIResponse(w, http.StatusBadRequest, "No body in request")
		return
	}

	// Parse the JSON body
	var requestBody struct {
		Body string `json:"body"`
	}

	err = json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		writeAPIResponse(w, http.StatusBadRequest, "Unable to decode status data")
		return
	}

	// Mock successful response
	writeAPIResponse(w, http.StatusOK, "Cluster status accepted")
}
