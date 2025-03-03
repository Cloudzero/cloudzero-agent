package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

type uploadRequest struct {
	Files []struct {
		ReferenceID string `json:"reference_id"`
	} `json:"files"`
}

type uploadResponse map[string]string

// generate a list of pre-signed urls
func (rw *remoteWrite) upload(w http.ResponseWriter, r *http.Request) {
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

	// Parse the count parameter if provided
	count := 1
	countParam := r.URL.Query().Get("count")
	if countParam != "" {
		count, err = strconv.Atoi(countParam)
		if err != nil || count <= 0 {
			writeAPIResponse(w, http.StatusBadRequest, "Invalid count parameter")
			return
		}
	}

	// Parse the request body
	var req uploadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// If files are provided in the request, use those reference IDs
	// Otherwise, generate the requested number of reference IDs
	var refIDs []string
	if len(req.Files) > 0 {
		for _, file := range req.Files {
			refIDs = append(refIDs, file.ReferenceID)
		}
	} else {
		for i := 0; i < count; i++ {
			refIDs = append(refIDs, generateRefID())
		}
	}

	// Generate response with pre-signed URLs
	response := make(uploadResponse)
	for _, refID := range refIDs {
		// create a pre-signed url with the minio client
		presignedURL, err := rw.minioClient.PresignedPutObject(r.Context(), bucketName, refID, time.Minute*10)
		if err != nil {
			writeAPIResponse(w, http.StatusBadRequest, "Failed to create the pre-signed url")
			return
		}

		// Store the file reference
		rw.files[refID] = &file{
			refID:   refID,
			url:     presignedURL.String(),
			created: time.Now(),
		}

		// Add to response
		response[refID] = presignedURL.String()
	}

	// TODO -- add replay header information
	// should use a mock request header to see if we should activate this or not
	// Set the X-CloudZero-Replay header

	// Return the response
	writeJSONResponse(w, http.StatusOK, response)
}
