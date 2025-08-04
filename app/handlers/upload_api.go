// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"
	"github.com/go-obvious/server/request"
	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-agent/app/domain/upload"
)

type UploadAPI struct {
	api.Service
	uploadService *upload.Service
}

func NewUploadAPI(base string, uploadService *upload.Service) *UploadAPI {
	a := &UploadAPI{
		uploadService: uploadService,
		Service: api.Service{
			APIName: "upload",
			Mounts:  map[string]*chi.Mux{},
		},
	}
	a.Service.Mounts[base] = a.Routes()
	return a
}

func (a *UploadAPI) Register(app server.Server) error {
	if err := a.Service.Register(app); err != nil {
		return err
	}
	return nil
}

func (a *UploadAPI) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Post("/", a.PostUpload)
	r.Get("/health", a.GetHealth)
	return r
}

type UploadAPIRequest struct {
	Files []FileRequest `json:"files"`
}

type FileRequest struct {
	ReferenceID string `json:"reference_id"`
}

type UploadAPIResponse struct {
	StatusCode int               `json:"statusCode"`
	Body       map[string]string `json:"body"`
	Headers    map[string]string `json:"headers,omitempty"`
}

func (a *UploadAPI) PostUpload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse JSON request body
	var req UploadAPIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to decode upload request")
		request.Reply(r, w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	// Extract query parameters
	query := r.URL.Query()
	cloudAccountID := query.Get("cloud_account_id")
	clusterName := query.Get("cluster_name")
	shipperID := query.Get("shipper_id")
	region := query.Get("region")

	// Validate required parameters
	if cloudAccountID == "" || clusterName == "" || shipperID == "" || region == "" {
		log.Ctx(ctx).Error().
			Str("cloud_account_id", cloudAccountID).
			Str("cluster_name", clusterName).
			Str("shipper_id", shipperID).
			Str("region", region).
			Msg("Missing required query parameters")
		request.Reply(r, w, "Missing required query parameters", http.StatusBadRequest)
		return
	}

	// For testing purposes, use a default organization ID
	organizationID := "test-org-123"

	// Convert request to domain model
	uploadReq := upload.UploadRequest{
		OrganizationID: organizationID,
		CloudAccountID: cloudAccountID,
		ClusterName:    clusterName,
		ShipperID:      shipperID,
		Region:         region,
		Files:          make([]upload.FileRequest, len(req.Files)),
	}

	for i, file := range req.Files {
		uploadReq.Files[i] = upload.FileRequest{
			ReferenceID: file.ReferenceID,
		}
	}

	// Generate upload URLs
	result, err := a.uploadService.GenerateUploadURLs(ctx, uploadReq)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("Failed to generate upload URLs")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Build response matching Python implementation format
	response := UploadAPIResponse{
		StatusCode: 200,
		Body:       result.URLs,
	}

	// Add replay header if there are replay URLs
	if len(result.Replay) > 0 {
		replayJSON, err := json.Marshal(result.Replay)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("Failed to marshal replay URLs")
		} else {
			response.Headers = map[string]string{
				"X-CloudZero-Replay": string(replayJSON),
			}
			w.Header().Set("X-CloudZero-Replay", string(replayJSON))
		}
	}

	log.Ctx(ctx).Info().
		Str("organization_id", organizationID).
		Str("cloud_account_id", cloudAccountID).
		Str("cluster_name", clusterName).
		Str("shipper_id", shipperID).
		Str("region", region).
		Int("file_count", len(req.Files)).
		Msg("Upload URLs generated successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (a *UploadAPI) GetHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"status": "healthy",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}