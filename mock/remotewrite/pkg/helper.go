// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package remotewrite provides a mock remote write server.
package remotewrite

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
)

type apiResponse struct {
	StatusCode int    `json:"statusCode"`
	Body       string `json:"body"`
}

// Generate a random reference ID
func generateRefID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Helper to check required query parameters
func checkRequiredParams(r *http.Request, params ...string) error {
	for _, param := range params {
		if r.URL.Query().Get(param) == "" {
			return fmt.Errorf("missing required parameter: %s", param)
		}
	}
	return nil
}

// Helper to write API response format
func writeAPIResponse(w http.ResponseWriter, statusCode int, message string) {
	response := apiResponse{
		StatusCode: statusCode,
		Body:       fmt.Sprintf("{\"message\": \"%s\"}", message),
	}
	writeJSONResponse(w, statusCode, response)
}

// Helper to write JSON response
func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
