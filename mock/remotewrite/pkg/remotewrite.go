// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package remotewrite

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const (
	bucketName = "testbucket"
	region     = "us-east-1"
)

type RemoteWrite struct {
	files map[string]*file

	apiKey string

	// internal state
	s3Endpoint   string
	s3AccessKey  string
	s3PrivateKey string

	// minio
	minioClient *minio.Client

	// network delays
	uploadDelay time.Duration

	// an optional payload that will be sent over the replay request header
	// every time a file is uploaded
	replayRequestPayload string

	errorOnUpload bool
}

type file struct {
	refID   string
	url     string
	created time.Time
}

type NewRemoteWriteOpts struct {
	APIKey               string
	S3Endpoint           string
	S3AccessKey          string
	S3PrivateKey         string
	UploadDelay          time.Duration
	ReplayRequestPayload string
	ErrorOnUpload        bool
}

func NewRemoteWrite(
	ctx context.Context,
	opts *NewRemoteWriteOpts,
) (*RemoteWrite, error) {
	// create the minio client
	minioClient, err := minio.New(opts.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(opts.S3AccessKey, opts.S3PrivateKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}

	// create the bucket
	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: region})
	if err != nil {
		// Check to see if we already own this bucket (which happens if you run this twice)
		exists, errBucketExists := minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", bucketName)
		} else {
			return nil, fmt.Errorf("failed to create the bucket: %w", err)
		}
	}

	return &RemoteWrite{
		files:                make(map[string]*file),
		apiKey:               opts.APIKey,
		s3Endpoint:           opts.S3Endpoint,
		s3AccessKey:          opts.S3AccessKey,
		s3PrivateKey:         opts.S3PrivateKey,
		uploadDelay:          opts.UploadDelay,
		minioClient:          minioClient,
		replayRequestPayload: opts.ReplayRequestPayload,
		errorOnUpload:        opts.ErrorOnUpload,
	}, nil
}

func (rw *RemoteWrite) Handler(r chi.Router) {
	// add middleware
	r.Use(func(h http.Handler) http.Handler {
		return authMiddleware(h, rw.apiKey)
	})

	// Cluster status endpoint
	r.Get("/cluster_status", http.HandlerFunc(rw.status))
	// Upload endpoint for pre-signed URLs
	r.Post("/upload", http.HandlerFunc(rw.upload))
	// Abandon endpoint
	r.Post("/abandon", http.HandlerFunc(rw.abandon))

	// mock-specific endpoints
	r.Route("/mock", func(r chi.Router) {
		r.Get("/queryMinio", http.HandlerFunc(rw.QueryMinio))
	})
}
