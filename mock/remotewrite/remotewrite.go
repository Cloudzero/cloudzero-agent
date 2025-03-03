package main

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

type remoteWrite struct {
	files map[string]*file

	// internal state
	s3Endpoint   string
	s3AccessKey  string
	s3PrivateKey string

	// minio
	minioClient *minio.Client
}

type file struct {
	refID   string
	url     string
	created time.Time
}

func newRemoteWrite(
	ctx context.Context,
	s3Endpoint string,
	s3AccessKey string,
	s3PrivateKey string,
) (*remoteWrite, error) {
	// create the minio client
	minioClient, err := minio.New(s3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3AccessKey, s3PrivateKey, ""),
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

	return &remoteWrite{
		files:        make(map[string]*file),
		s3Endpoint:   s3Endpoint,
		s3AccessKey:  s3AccessKey,
		s3PrivateKey: s3PrivateKey,
		minioClient:  minioClient,
	}, nil
}

func (rw *remoteWrite) Handler(r chi.Router) {
	// Cluster status endpoint
	r.Get("/cluster_status", http.HandlerFunc(rw.status))
	// Upload endpoint for pre-signed URLs
	r.Post("/upload", http.HandlerFunc(rw.upload))
	// Abandon endpoint
	r.Post("/abandon", http.HandlerFunc(rw.abandon))
}
