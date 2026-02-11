// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
)

// ResourceStore defines the specialized storage interface for Kubernetes resource metadata persistence.
// This interface extends the generic Storage pattern with resource-specific query capabilities
// essential for the CloudZero Agent's cost allocation and resource tracking functionality.
//
// The ResourceStore manages ResourceTags entities which contain cost allocation metadata
// extracted from Kubernetes resources during admission webhook processing. This data enables
// CloudZero's cost optimization platform to attribute infrastructure costs to specific
// applications, teams, and business units.
//
// Key responsibilities:
//   - Persist resource metadata for cost allocation analysis
//   - Support efficient lookup of resources by various attributes
//   - Enable bulk operations for resource processing pipelines
//   - Maintain referential integrity for cost calculation accuracy
//
// Integration points:
//   - Webhook handlers store new resource metadata during admission processing
//   - Cost allocation services query stored metadata for billing attribution
//   - Monitoring systems track resource metadata volume and processing efficiency
type ResourceStore interface {
	// StorageCommon provides transaction support and bulk operations for resource metadata.
	// These operations ensure data consistency across multiple resource updates and
	// enable efficient batch processing during resource discovery and validation.
	StorageCommon

	// Storage provides CRUD operations for ResourceTags entities with string-based identifiers.
	// The string ID typically represents the unique resource identifier within Kubernetes
	// (namespace/name format) enabling efficient resource lookups and updates.
	Storage[ResourceTags, string]

	// FindFirstBy retrieves the first ResourceTags entity matching the specified query conditions.
	// This method supports complex queries for resource discovery and validation operations.
	//
	// Common usage patterns:
	//   - Finding resources by specific label combinations for cost attribution
	//   - Locating resources within specific namespaces for scope validation
	//   - Discovering resources with specific ownership or team assignments
	//
	// The conditions parameter accepts ORM-style query conditions (field=value, field IN (values))
	// enabling flexible resource lookup without complex SQL construction.
	//
	// Returns nil if no matching resource is found, enabling safe handling of optional lookups.
	// Returns error if the query fails due to database connectivity or malformed conditions.
	FindFirstBy(ctx context.Context, conds ...interface{}) (*ResourceTags, error)

	// FindAllBy retrieves all ResourceTags entities matching the specified query conditions.
	// This method enables bulk resource discovery and batch processing operations essential
	// for cost allocation calculations and resource lifecycle management.
	//
	// Common usage patterns:
	//   - Finding all resources belonging to a specific team or cost center
	//   - Discovering resources with missing or incomplete cost allocation tags
	//   - Retrieving resources for bulk cost calculation and reporting operations
	//   - Implementing resource cleanup based on lifecycle policies
	//
	// The conditions parameter supports the same query syntax as FindFirstBy but returns
	// multiple results in a slice format for efficient batch processing.
	//
	// Returns empty slice if no matching resources are found, enabling safe iteration.
	// Returns error if the query fails due to database issues or invalid query conditions.
	//
	// Performance considerations:
	//   - Large result sets may require pagination in production environments
	//   - Consider using transaction context for consistency during bulk operations
	//   - Index key fields used in conditions for optimal query performance
	FindAllBy(ctx context.Context, conds ...interface{}) ([]*ResourceTags, error)
}
