// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package types defines Kubernetes resource metadata structures for cost allocation.
package types

import (
	"time"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
)

// Type aliases for Kubernetes metadata used throughout the cost allocation system.
// These aliases provide semantic clarity while maintaining compatibility with
// standard Go map types for label and annotation processing.
type (
	// Labels represents Kubernetes resource labels used for cost allocation and resource identification.
	Labels = map[string]string

	// Annotations represents Kubernetes resource annotations containing additional metadata for billing analysis.
	Annotations = map[string]string
)

// ResourceTags represents a database record of Kubernetes resource metadata for cost allocation.
// This structure stores labels, annotations, and metrics from admission webhook events,
// enabling the CloudZero platform to perform detailed cost analysis and resource attribution.
type ResourceTags struct {
	// ID is a unique database identifier for this resource metadata record.
	ID string `gorm:"unique;autoIncrement"`

	// Type identifies the Kubernetes resource type (deployment, pod, node, etc.) for cost categorization.
	Type config.ResourceType `gorm:"primaryKey"`

	// Name is the Kubernetes resource name, forming part of the composite primary key.
	Name string `gorm:"primaryKey"`

	// Namespace is the Kubernetes namespace containing this resource, nullable for cluster-scoped resources.
	Namespace *string `gorm:"primaryKey"`

	// MetricLabels contains Prometheus-style labels extracted from the resource for metric correlation.
	MetricLabels *config.MetricLabels `gorm:"serializer:json"`

	// Labels contains the Kubernetes resource labels used for cost allocation and resource grouping.
	Labels *config.MetricLabelTags `gorm:"serializer:json"`

	// Annotations contains the Kubernetes resource annotations with additional cost allocation metadata.
	Annotations *config.MetricLabelTags `gorm:"serializer:json"`

	// RecordCreated is when this metadata record was first created in the local database.
	RecordCreated time.Time

	// RecordUpdated is when this record was last updated due to resource label/annotation changes.
	RecordUpdated time.Time

	// SentAt tracks when this record was successfully transmitted to the CloudZero API, null if pending.
	SentAt *time.Time

	// Size is a computed field showing the total byte size of this record for storage monitoring.
	Size int `gorm:"->;type:GENERATED ALWAYS AS (octet_length(name) + IFNULL(octet_length(namespace), 0) + IFNULL(octet_length(labels), 0) + IFNULL(octet_length(annotations), 0)) VIRTUAL;"`
}

// RemoteWriteHistory tracks the timestamp of the most recent Prometheus remote_write request.
// This structure is used to monitor metric collection activity and detect collection gaps
// that might indicate issues with the Prometheus integration or network connectivity.
type RemoteWriteHistory struct {
	// LastRemoteWriteTime is the timestamp of the most recent successful remote_write request.
	// Used for monitoring metric collection continuity and detecting collection interruptions.
	LastRemoteWriteTime time.Time
}
