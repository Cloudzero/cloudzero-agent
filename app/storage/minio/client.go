// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package minio

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/domain/upload/ports"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Client wraps MinIO client for upload API operations
type Client struct {
	client     *minio.Client
	bucketName string
}

// Config holds MinIO client configuration
type Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	UseSSL          bool
}

// NewClient creates a new MinIO client
func NewClient(cfg Config) (*Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create MinIO client: %w", err)
	}

	return &Client{
		client:     client,
		bucketName: cfg.BucketName,
	}, nil
}

// GeneratePresignedURL creates a presigned URL for uploading a file
func (c *Client) GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error) {
	presignedURL, err := c.client.PresignedPutObject(ctx, c.bucketName, key, expiration)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL for key %s: %w", key, err)
	}

	return presignedURL.String(), nil
}

// BuildKey constructs the S3-compatible key path for a file
// Follows the Python implementation pattern:
// upload/organization_id={org}/year={Y}/month={M}/day={D}/hour={H}/cloud_account_id={account}/cluster_name={cluster}/shipper_id={shipper}/region={region}/{reference_id}.parquet
func (c *Client) BuildKey(params ports.KeyParams) string {
	return fmt.Sprintf(
		"upload/organization_id=%s/year=%d/month=%d/day=%d/hour=%d/cloud_account_id=%s/cluster_name=%s/shipper_id=%s/region=%s/%s.parquet",
		url.QueryEscape(params.OrganizationID),
		params.Year,
		params.Month,
		params.Day,
		params.Hour,
		url.QueryEscape(params.CloudAccountID),
		url.QueryEscape(params.ClusterName),
		url.QueryEscape(params.ShipperID),
		url.QueryEscape(params.Region),
		url.QueryEscape(params.ReferenceID),
	)
}

// EnsureBucket creates the bucket if it doesn't exist
func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.client.BucketExists(ctx, c.bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if !exists {
		err = c.client.MakeBucket(ctx, c.bucketName, minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket %s: %w", c.bucketName, err)
		}
	}

	return nil
}