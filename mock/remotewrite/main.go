// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	remotewrite "github.com/cloudzero/cloudzero-agent/mock/remotewrite/pkg"
	"github.com/go-chi/chi/v5"
	"golang.org/x/exp/slog"
)

func main() {
	// check for an api key
	apiKey, exists := os.LookupEnv("API_KEY")
	if !exists {
		apiKey = "ak-test"
	}

	// check for a port
	port, exists := os.LookupEnv("PORT")
	if !exists {
		port = "8081"
	}

	// get s3 credentials
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	if s3Endpoint == "" {
		log.Fatalf("Please pass `S3_ENDPOINT` env variable")
	}
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	if s3AccessKey == "" {
		log.Fatalf("Please pass `S3_ACCESS_KEY` env variable")
	}
	s3PrivateKey := os.Getenv("S3_PRIVATE_KEY")
	if s3PrivateKey == "" {
		log.Fatalf("Please pass `S3_PRIVATE_KEY` env variable")
	}

	// check for mock options
	uploadDelay := time.Second * 0
	uploadDelayMsStr := os.Getenv("UPLOAD_DELAY_MS")
	uploadDelayMs, err := strconv.Atoi(uploadDelayMsStr)
	if err == nil {
		uploadDelay = time.Millisecond * time.Duration(uploadDelayMs)
	}

	// create the client
	rw, err := remotewrite.NewRemoteWrite(context.Background(), &remotewrite.NewRemoteWriteOpts{
		APIKey:               apiKey,
		S3Endpoint:           s3Endpoint,
		S3AccessKey:          s3AccessKey,
		S3PrivateKey:         s3PrivateKey,
		UploadDelay:          uploadDelay,
		ReplayRequestPayload: os.Getenv("REPLAY_REQUEST_PAYLOAD"),
		ErrorOnUpload:        os.Getenv("ERROR_ON_UPLOAD") == "true",
	})
	if err != nil {
		log.Fatalf("failed to create the remotewrite client: %s", err.Error())
	}

	// create the http mux
	r := chi.NewRouter()

	// add routes
	r.Route("/v1/container-metrics", rw.Handler)

	slog.Default().Info(fmt.Sprintf("Mock remotewrite is listening on: 'localhost:%s'", port))
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), r); err != nil {
		slog.Default().Error("Error starting server", "error", err)
	}
}
