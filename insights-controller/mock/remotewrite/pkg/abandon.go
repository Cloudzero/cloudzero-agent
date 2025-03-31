// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package remotewrite

import (
	"encoding/json"
	"net/http"
)

type abandonRequest []struct {
	ReferenceID string `json:"reference_id"`
	Reason      string `json:"reason"`
}

func (rw *RemoteWrite) abandon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check required query parameters
	err := checkRequiredParams(r, "cluster_name", "cloud_account_id", "region")
	if err != nil {
		writeAPIResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Parse the request body
	var req abandonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Process each reference ID to be abandoned
	for _, item := range req {
		// Remove the file reference
		delete(rw.files, item.ReferenceID)
		// Note: In a real implementation, we might want to track abandoned files
		// or return errors for non-existent reference IDs
	}

	// Return success response
	writeAPIResponse(w, http.StatusOK, "Abandon request processed successfully")
}
