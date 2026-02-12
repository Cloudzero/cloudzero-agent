// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// AdmissionReviewVersion represents the API version used for Kubernetes admission webhook communication.
// This type enables the CloudZero Agent to support multiple admission webhook API versions
// while maintaining compatibility across different Kubernetes cluster versions.
//
// The CloudZero Agent webhook must handle both v1beta1 (legacy) and v1 (current) admission
// review protocols to ensure broad compatibility with Kubernetes clusters running different
// API server versions. Version selection impacts request/response format validation and
// field availability during cost allocation metadata extraction.
type AdmissionReviewVersion string

const (
	// AdmissionReviewVersionV1beta1 identifies the deprecated v1beta1 admission review API.
	// Used for compatibility with older Kubernetes clusters (versions 1.16-1.18).
	// The CloudZero Agent maintains support for this version to ensure cost allocation
	// functionality works across diverse customer environments with legacy clusters.
	//
	// Key limitations:
	//   - Some fields may be missing or have different formats compared to v1
	//   - Deprecated status requires careful handling for future Kubernetes compatibility
	//   - Warning generation may behave differently than v1 API
	AdmissionReviewVersionV1beta1 AdmissionReviewVersion = "v1beta1"

	// AdmissionReviewVersionV1 identifies the current stable v1 admission review API.
	// This is the preferred version for Kubernetes clusters 1.19+ and provides the most
	// complete and stable admission review functionality for cost allocation processing.
	//
	// Benefits over v1beta1:
	//   - Complete field set for resource metadata extraction
	//   - Stable API contract with guaranteed backward compatibility
	//   - Enhanced warning support for user feedback during validation
	//   - Improved authentication and authorization information access
	AdmissionReviewVersionV1 AdmissionReviewVersion = "v1"
)

// AdmissionReviewOp represents the type of Kubernetes operation being performed during admission review.
// This type enables the CloudZero Agent webhook to apply different cost allocation logic
// based on whether resources are being created, updated, or deleted within the cluster.
//
// Different operations require different cost allocation strategies:
//   - CREATE: Initial cost tag validation and metadata injection
//   - UPDATE: Cost tag change validation and billing impact assessment
//   - DELETE: Cost allocation cleanup and resource lifecycle tracking
//   - CONNECT: Proxy request validation (typically not used for cost allocation)
type AdmissionReviewOp string

const (
	// OperationUnknown represents an unrecognized or unsupported admission operation.
	// Used as a fallback when the Kubernetes API server sends an operation type
	// that the CloudZero Agent doesn't recognize, enabling graceful error handling
	// and preventing admission failures due to unknown operation types.
	//
	// Typically results in:
	//   - Admission allowed with warning
	//   - Operational monitoring alert for unknown operation encountered
	//   - No cost allocation metadata processing
	OperationUnknown AdmissionReviewOp = "unknown"

	// OperationCreate represents a resource creation operation in Kubernetes.
	// This is the most common operation processed by the CloudZero Agent webhook
	// for cost allocation metadata validation and injection.
	//
	// Processing includes:
	//   - Validating required cost allocation tags are present
	//   - Injecting missing cost center or team identification labels
	//   - Establishing initial resource cost attribution
	//   - Recording resource creation for billing lifecycle tracking
	OperationCreate AdmissionReviewOp = "create"

	// OperationUpdate represents a resource modification operation in Kubernetes.
	// The CloudZero Agent processes updates to detect cost allocation changes
	// and ensure billing accuracy when resource metadata changes.
	//
	// Processing includes:
	//   - Detecting changes to cost allocation tags and labels
	//   - Validating cost center transfers and ownership changes
	//   - Recording cost attribution modifications for audit purposes
	//   - Updating stored resource metadata for accurate billing
	OperationUpdate AdmissionReviewOp = "update"

	// OperationDelete represents a resource deletion operation in Kubernetes.
	// The CloudZero Agent processes deletions to complete cost allocation
	// lifecycle tracking and ensure proper resource cleanup.
	//
	// Processing includes:
	//   - Recording final cost allocation state before deletion
	//   - Triggering cost allocation cleanup procedures
	//   - Updating resource lifecycle tracking for billing accuracy
	//   - Generally allows deletion with metadata cleanup
	OperationDelete AdmissionReviewOp = "delete"

	// OperationConnect represents a proxy connection operation in Kubernetes.
	// This operation type is rarely used in admission control and typically
	// doesn't require cost allocation processing by the CloudZero Agent.
	//
	// Processing includes:
	//   - Generally allowed without cost allocation validation
	//   - May log connection attempts for security auditing
	//   - Doesn't affect resource cost attribution or billing
	OperationConnect AdmissionReviewOp = "connect"
)

// AdmissionReview represents a normalized admission webhook request for CloudZero cost allocation processing.
// This struct provides a unified interface for handling admission requests from both v1 and v1beta1
// Kubernetes APIs, enabling consistent cost allocation logic regardless of cluster version.
//
// The CloudZero Agent uses this structure to:
//   - Extract resource metadata for cost allocation validation
//   - Store admission context for audit and troubleshooting
//   - Process resource changes for billing impact assessment
//   - Generate appropriate admission responses with cost allocation guidance
//
// This abstraction simplifies webhook logic by providing a single interface for both
// admission API versions while preserving all necessary information for cost allocation decisions.
type AdmissionReview struct {
	// OriginalAdmissionReview holds the raw Kubernetes admission review object (v1 or v1beta1).
	// This field preserves the original request for debugging, audit logging, and response
	// generation that requires format-specific handling. Essential for troubleshooting
	// admission failures and maintaining compatibility with different API versions.
	OriginalAdmissionReview runtime.Object

	// ID is the unique identifier for this admission request from the Kubernetes API server.
	// Used for tracking admission requests through the CloudZero Agent processing pipeline,
	// correlating logs, and generating audit trails for cost allocation decisions.
	// Corresponds to the UID field in the original admission request.
	ID string

	// Name is the name of the Kubernetes resource being processed in this admission request.
	// Combined with Namespace, this provides the unique resource identifier within the cluster
	// for cost allocation tracking and resource metadata storage. May be empty for
	// cluster-scoped resources or during resource generation.
	Name string

	// Namespace is the namespace containing the resource being processed in this admission request.
	// Essential for cost allocation as many organizations use namespace-based cost attribution
	// and billing strategies. Empty for cluster-scoped resources like nodes, persistent volumes,
	// and custom resource definitions that don't belong to specific namespaces.
	Namespace string

	// Operation identifies the type of Kubernetes operation being performed (CREATE, UPDATE, DELETE, CONNECT).
	// The CloudZero Agent uses this field to apply appropriate cost allocation logic:
	// CREATE operations validate and inject cost tags, UPDATE operations track cost changes,
	// DELETE operations cleanup cost allocation metadata.
	Operation AdmissionReviewOp

	// Version identifies which admission review API version this request originated from (v1 or v1beta1).
	// Used for generating properly formatted responses and handling version-specific field availability.
	// Critical for maintaining compatibility across different Kubernetes cluster versions.
	Version AdmissionReviewVersion

	// RequestGVR identifies the Group/Version/Resource being operated on in this admission request.
	// Used by the CloudZero Agent to determine which cost allocation rules and validation
	// logic to apply based on resource type. Different resource types may require different
	// cost attribution strategies (e.g., pods vs services vs ingresses).
	RequestGVR *metav1.GroupVersionResource

	// RequestGVK identifies the Group/Version/Kind being operated on in this admission request.
	// Complements RequestGVR by providing the Kind information needed for resource-specific
	// cost allocation processing. Used for routing requests to appropriate resource handlers.
	RequestGVK *metav1.GroupVersionKind

	// OldObjectRaw contains the raw JSON of the resource before modification (for UPDATE operations).
	// The CloudZero Agent uses this field to detect changes in cost allocation metadata
	// and calculate billing impact when resources are modified. Empty for CREATE operations.
	// Essential for tracking cost center transfers and ownership changes.
	OldObjectRaw []byte

	// NewObjectRaw contains the raw JSON of the resource after modification or creation.
	// This is the primary field used by the CloudZero Agent to extract cost allocation
	// metadata, validate required tags, and inject missing cost attribution information.
	// Contains the complete resource specification for cost allocation analysis.
	NewObjectRaw []byte

	// DryRun indicates whether this is a dry-run request (kubectl apply --dry-run).
	// When true, the CloudZero Agent performs all validation and cost allocation analysis
	// but doesn't store resource metadata or trigger billing system updates.
	// Used for testing cost allocation policies without affecting production data.
	DryRun bool

	// UserInfo contains authentication information about the user or service account
	// making the resource request. The CloudZero Agent may use this information for:
	// - Audit logging of cost allocation changes
	// - Applying user-specific cost allocation policies
	// - Attribution of cost allocation decisions to specific users or teams
	UserInfo authenticationv1.UserInfo
}

// NewAdmissionReviewV1Beta1 creates a normalized AdmissionReview from a v1beta1 Kubernetes admission request.
// This function converts the deprecated v1beta1 admission review format into the unified
// CloudZero Agent internal format, enabling consistent cost allocation processing across
// different Kubernetes API versions.
//
// Used primarily for backward compatibility with older Kubernetes clusters (1.16-1.18)
// that only support the v1beta1 admission webhook API. The function handles field mapping
// and provides safe defaults for fields that may not exist in the v1beta1 format.
//
// Key transformations:
//   - Maps v1beta1 operation types to internal AdmissionReviewOp constants
//   - Extracts GroupVersionResource and GroupVersionKind with fallback logic
//   - Handles optional DryRun field with safe default (false)
//   - Preserves original admission review for format-specific response generation
//
// Returns a fully populated AdmissionReview ready for cost allocation processing.
func NewAdmissionReviewV1Beta1(ar *admissionv1beta1.AdmissionReview) AdmissionReview {
	// Default false.
	dryRun := false
	if ar.Request.DryRun != nil {
		dryRun = *ar.Request.DryRun
	}

	return AdmissionReview{
		OriginalAdmissionReview: ar,
		ID:                      string(ar.Request.UID),
		Name:                    ar.Request.Name,
		Version:                 AdmissionReviewVersionV1beta1,
		Namespace:               ar.Request.Namespace,
		Operation:               v1Beta1OperationToModel(ar.Request.Operation),
		OldObjectRaw:            ar.Request.OldObject.Raw,
		NewObjectRaw:            ar.Request.Object.Raw,
		RequestGVR:              v1Beta1ResourceToModel(ar),
		RequestGVK:              v1Beta1KindToModel(ar),
		UserInfo:                ar.Request.UserInfo,
		DryRun:                  dryRun,
	}
}

func v1Beta1ResourceToModel(ar *admissionv1beta1.AdmissionReview) *metav1.GroupVersionResource {
	if ar.Request.RequestResource != nil {
		return ar.Request.RequestResource
	}

	return &metav1.GroupVersionResource{
		Group:    ar.Request.Resource.Group,
		Version:  ar.Request.Resource.Version,
		Resource: ar.Request.Resource.Resource,
	}
}

func v1Beta1KindToModel(ar *admissionv1beta1.AdmissionReview) *metav1.GroupVersionKind {
	if ar.Request.RequestKind != nil {
		return ar.Request.RequestKind
	}

	return &metav1.GroupVersionKind{
		Group:   ar.Request.Kind.Group,
		Version: ar.Request.Kind.Version,
		Kind:    ar.Request.Kind.Kind,
	}
}

func v1Beta1OperationToModel(op admissionv1beta1.Operation) AdmissionReviewOp {
	switch op {
	case admissionv1beta1.Create:
		return OperationCreate
	case admissionv1beta1.Update:
		return OperationUpdate
	case admissionv1beta1.Delete:
		return OperationDelete
	case admissionv1beta1.Connect:
		return OperationConnect
	}

	return OperationUnknown
}

// NewAdmissionReviewV1 creates a normalized AdmissionReview from a v1 Kubernetes admission request.
// This function converts the current stable v1 admission review format into the unified
// CloudZero Agent internal format, providing the preferred processing path for modern
// Kubernetes clusters (1.19+) with complete field availability.
//
// This is the primary constructor used for admission request processing as it handles
// the stable v1 API with full feature support including enhanced warning capabilities
// and complete authentication information.
//
// Key transformations:
//   - Maps v1 operation types to internal AdmissionReviewOp constants
//   - Extracts GroupVersionResource and GroupVersionKind with complete field support
//   - Handles DryRun field with proper nil pointer checking
//   - Preserves original admission review for generating properly formatted v1 responses
//
// Returns a fully populated AdmissionReview with access to all v1 API fields.
func NewAdmissionReviewV1(ar *admissionv1.AdmissionReview) AdmissionReview {
	// Default false.
	dryRun := false
	if ar.Request.DryRun != nil {
		dryRun = *ar.Request.DryRun
	}

	return AdmissionReview{
		OriginalAdmissionReview: ar,
		ID:                      string(ar.Request.UID),
		Name:                    ar.Request.Name,
		Namespace:               ar.Request.Namespace,
		Version:                 AdmissionReviewVersionV1,
		Operation:               v1OperationToModel(ar.Request.Operation),
		OldObjectRaw:            ar.Request.OldObject.Raw,
		NewObjectRaw:            ar.Request.Object.Raw,
		RequestGVR:              v1ResourceToModel(ar),
		RequestGVK:              v1KindToModel(ar),
		UserInfo:                ar.Request.UserInfo,
		DryRun:                  dryRun,
	}
}

func v1ResourceToModel(ar *admissionv1.AdmissionReview) *metav1.GroupVersionResource {
	if ar.Request.RequestResource != nil {
		return ar.Request.RequestResource
	}

	return &metav1.GroupVersionResource{
		Group:    ar.Request.Resource.Group,
		Version:  ar.Request.Resource.Version,
		Resource: ar.Request.Resource.Resource,
	}
}

func v1KindToModel(ar *admissionv1.AdmissionReview) *metav1.GroupVersionKind {
	if ar.Request.RequestKind != nil {
		return ar.Request.RequestKind
	}

	return &metav1.GroupVersionKind{
		Group:   ar.Request.Kind.Group,
		Version: ar.Request.Kind.Version,
		Kind:    ar.Request.Kind.Kind,
	}
}

func v1OperationToModel(op admissionv1.Operation) AdmissionReviewOp {
	switch op {
	case admissionv1.Create:
		return OperationCreate
	case admissionv1.Update:
		return OperationUpdate
	case admissionv1.Delete:
		return OperationDelete
	case admissionv1.Connect:
		return OperationConnect
	}

	return OperationUnknown
}

// AdmissionResponse represents the CloudZero Agent's decision on a Kubernetes admission request.
// This struct provides a simplified interface for generating admission webhook responses
// that communicate cost allocation validation results back to the Kubernetes API server.
//
// The CloudZero Agent uses this structure to:
//   - Indicate whether resources should be allowed into the cluster
//   - Provide detailed messages explaining cost allocation validation failures
//   - Generate warnings about cost allocation best practices
//   - Correlate responses with original admission requests for audit purposes
//
// This abstraction enables consistent response generation across different admission
// API versions while maintaining compatibility with both v1 and v1beta1 response formats.
type AdmissionResponse struct {
	// ID correlates this response with the original admission request identifier.
	// Used for tracking admission decisions through logs and audit trails, enabling
	// troubleshooting of cost allocation validation failures and policy enforcement.
	// Must match the ID from the corresponding AdmissionReview.
	ID string

	// Allowed indicates whether the CloudZero Agent permits this resource operation.
	// When false, the Kubernetes API server will reject the resource operation and
	// return the Message to the user. When true, the operation proceeds normally.
	//
	// Common reasons for rejection (Allowed=false):
	//   - Missing required cost allocation tags
	//   - Invalid cost center or team assignments
	//   - Policy violations (e.g., unapproved namespaces)
	//   - Resource quota or billing limit exceeded
	Allowed bool

	// Message provides detailed explanation for the admission decision.
	// When Allowed=false, this message is displayed to users explaining why
	// their resource was rejected and how to fix cost allocation issues.
	// When Allowed=true, this may contain informational text about cost allocation.
	//
	// Examples:
	//   - "Missing required cost allocation tag: cost-center"
	//   - "Invalid team assignment: 'marketing' not found in organization"
	//   - "Resource allowed with cost center 'engineering'"
	Message string

	// Warnings contain non-blocking advice about cost allocation best practices.
	// These are displayed to users even when Allowed=true, helping organizations
	// improve their cost allocation hygiene without blocking resource creation.
	//
	// Common warnings:
	//   - "Consider adding project-specific tags for better cost attribution"
	//   - "Namespace 'default' has no cost center assignment"
	//   - "Resource lacks owner information for cost allocation"
	Warnings []string
}
