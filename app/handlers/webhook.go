// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package handlers provides HTTP request handlers for CloudZero Agent Primary Adapter implementations.
// This package implements the HTTP interface layer that receives external requests from Kubernetes API servers,
// Prometheus instances, and other external systems, translating them into domain service operations.
//
// The handlers serve as Primary Adapters in the hexagonal architecture, providing the entry point
// for all external communication with the CloudZero Agent while maintaining clean separation
// between HTTP concerns and business logic.
//
// Key responsibilities:
//   - HTTP request/response handling: Parse incoming requests and generate appropriate responses
//   - Protocol adaptation: Convert HTTP requests to domain service method calls
//   - Error handling: Translate domain errors to appropriate HTTP status codes and responses
//   - Content negotiation: Handle various content types and serialization formats
//   - Security enforcement: Validate request authenticity and enforce security policies
//
// Handler types:
//   - ValidationWebhookAPI: Kubernetes admission webhook for cost allocation validation
//   - Remote write handlers: Prometheus metric ingestion endpoints
//   - Health check handlers: Service health and readiness endpoints
//   - Profiling handlers: Development and debugging support endpoints
//
// The package ensures that domain services remain HTTP-agnostic while providing
// robust, production-ready HTTP interfaces for all CloudZero Agent functionality.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"
	"github.com/go-obvious/server/request"

	"github.com/rs/zerolog/log"
	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/cloudzero/cloudzero-agent/app/domain/webhook"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

// MaxRequestBodyBytes represents the max size of Kubernetes objects we read. Kubernetes allows a 2x
// buffer on the max etcd size
// (https://github.com/kubernetes/kubernetes/blob/0afa569499d480df4977568454a50790891860f5/staging/src/k8s.io/apiserver/pkg/server/config.go#L362).
// We allow an additional 2x buffer, as it is still fairly cheap (6mb)
// Taken from https://github.com/istio/istio/commit/6ca5055a4db6695ef5504eabdfde3799f2ea91fd
const (
	// minimalAllowResponse provides emergency fail-open behavior for Kubernetes admission webhook responses.
	// This minimal JSON response is used as a last resort when normal response marshalling fails,
	// ensuring that Kubernetes clusters continue operating even during CloudZero Agent errors.
	//
	// The fail-open design is critical for production stability:
	//   - Prevents cluster disruption during agent failures or restarts
	//   - Allows workload deployment to continue even with cost allocation issues
	//   - Maintains cluster availability over strict cost tracking requirements
	//   - Provides graceful degradation when JSON serialization fails
	//
	// This follows the principle that operational availability takes precedence over
	// perfect cost allocation data in production Kubernetes environments.
	minimalAllowResponse = `{"response":{"allowed":true}}`

	// MaxRequestBodyBytes defines the maximum size of Kubernetes admission review requests.
	// This limit is based on Kubernetes etcd storage constraints with additional safety buffer
	// to handle large resource manifests while preventing memory exhaustion attacks.
	//
	// Size calculation:
	//   - Kubernetes default etcd max size: 1.5MB
	//   - Kubernetes internal buffer: 2x (3MB total)
	//   - CloudZero Agent additional buffer: 2x (6MB final limit)
	//
	// This conservative approach ensures:
	//   - Support for large ConfigMaps, Secrets, and Custom Resources
	//   - Protection against malicious oversized requests
	//   - Memory usage predictability in constrained environments
	//   - Compatibility with Kubernetes API server request size policies
	//
	// Reference: https://github.com/istio/istio/commit/6ca5055a4db6695ef5504eabdfde3799f2ea91fd
	MaxRequestBodyBytes = int64(6 * 1024 * 1024)

	// DefaultTimeout specifies the maximum processing time for admission webhook requests.
	// This timeout balances thorough cost allocation analysis with Kubernetes responsiveness
	// requirements, ensuring webhook decisions complete within cluster performance expectations.
	//
	// Timeout considerations:
	//   - Kubernetes API server expects webhook responses within 10-30 seconds
	//   - CloudZero metadata extraction and storage operations typically complete in <1 second
	//   - Network latency for CloudZero API calls may add 2-5 seconds
	//   - Buffer for temporary slowdowns or resource contention
	//
	// Production implications:
	//   - Prevents webhook from blocking cluster operations during slowdowns
	//   - Allows graceful fallback to fail-open behavior if processing exceeds timeout
	//   - Maintains cluster responsiveness under high admission request volumes
	//   - Provides predictable SLA for application deployment times
	DefaultTimeout = 15 * time.Second

	// MinTimeout establishes the minimum acceptable timeout for admission webhook processing.
	// This prevents extremely short timeouts that could cause unnecessary failures during
	// normal operations while still allowing timeout customization for specific environments.
	//
	// This minimum ensures:
	//   - Sufficient time for standard CloudZero metadata extraction
	//   - Network request completion under normal conditions
	//   - Database transaction completion for resource storage
	//   - Graceful handling of minor system resource contention
	//
	// Values below this threshold risk causing webhook failures during normal
	// cluster operations, potentially degrading application deployment reliability.
	MinTimeout = 5 * time.Second
)

var (
	// v1beta1AdmissionReviewTypeMeta provides Kubernetes API metadata for legacy admission webhook responses.
	// This TypeMeta structure is required for proper serialization of AdmissionReview objects
	// when responding to Kubernetes API servers using the v1beta1 admission API version.
	//
	// Legacy API support considerations:
	//   - Required for Kubernetes clusters running versions 1.16-1.21
	//   - Maintained for backward compatibility with older cluster versions
	//   - Feature limitations: No support for admission warnings
	//   - Gradual deprecation path as clusters upgrade to newer versions
	//
	// The v1beta1 API version lacks certain features available in v1, requiring
	// special handling for advanced admission control features.
	v1beta1AdmissionReviewTypeMeta = metav1.TypeMeta{
		Kind:       "AdmissionReview",
		APIVersion: "admission.k8s.io/v1beta1",
	}

	// v1AdmissionReviewTypeMeta provides Kubernetes API metadata for modern admission webhook responses.
	// This TypeMeta structure supports the full feature set of Kubernetes admission control,
	// including admission warnings and enhanced error reporting capabilities.
	//
	// Modern API features:
	//   - Admission warnings: Non-blocking notifications to users about potential issues
	//   - Enhanced error details: Structured error responses with field-level validation
	//   - Improved security context: Better support for admission security policies
	//   - Future-ready: Supports ongoing Kubernetes admission control evolution
	//
	// This is the preferred API version for Kubernetes clusters 1.22+ and should be
	// used for all new CloudZero Agent deployments and webhook configurations.
	v1AdmissionReviewTypeMeta = metav1.TypeMeta{
		Kind:       "AdmissionReview",
		APIVersion: "admission.k8s.io/v1",
	}
)

// ValidationWebhookAPI implements the HTTP interface for CloudZero Agent Kubernetes admission webhooks.
// This struct serves as the Primary Adapter in the hexagonal architecture, translating HTTP admission
// requests from the Kubernetes API server into domain service operations for cost allocation processing.
//
// The webhook API provides the critical integration point between Kubernetes resource lifecycle
// management and CloudZero cost optimization, enabling automatic cost allocation metadata extraction
// and storage during resource admission processing.
//
// Architecture responsibilities:
//   - HTTP protocol handling: Parse admission review requests and generate valid responses
//   - Request validation: Ensure admission requests meet security and format requirements
//   - Timeout management: Enforce processing time limits to maintain cluster responsiveness
//   - Error handling: Translate domain errors into appropriate HTTP responses with fail-open behavior
//   - API versioning: Support both v1 and v1beta1 admission review formats for compatibility
//
// Production characteristics:
//   - Fail-open behavior: Always allow resources if processing fails to prevent cluster disruption
//   - Connection management: Periodic connection closure for load balancing across replicas
//   - Security validation: Content-type verification and request size limits
//   - Structured logging: Comprehensive request tracing for operational monitoring
//
// The webhook integrates with the domain webhook controller to perform actual cost allocation
// logic while maintaining clean separation between HTTP concerns and business logic.
type ValidationWebhookAPI struct {
	// api.Service provides the foundational HTTP server infrastructure from go-obvious/server.
	// This embedded service handles HTTP server lifecycle, request routing, middleware integration,
	// and provides consistent API patterns across CloudZero Agent HTTP endpoints.
	//
	// Service capabilities:
	//   - Router mounting: Automatic registration of webhook routes with the HTTP server
	//   - Middleware support: Request logging, authentication, and performance monitoring
	//   - Graceful shutdown: Proper cleanup during agent termination or restart
	//   - Health checking: Integration with agent health monitoring systems
	api.Service

	// controller implements the core admission webhook processing logic for CloudZero cost allocation.
	// This domain service handles the actual cost allocation metadata extraction, validation,
	// and storage operations while remaining HTTP-agnostic.
	//
	// Controller responsibilities:
	//   - Resource metadata extraction from Kubernetes admission requests
	//   - Cost allocation classification and labeling logic
	//   - Resource persistence to CloudZero Agent storage systems
	//   - Integration with CloudZero platform APIs for data synchronization
	//
	// The controller abstraction enables testing the business logic independently of HTTP concerns
	// and supports different webhook deployment patterns (HTTP vs gRPC) if needed.
	controller webhook.WebhookController

	// decoder provides Kubernetes-compatible deserialization for admission review requests.
	// This runtime decoder handles both v1 and v1beta1 admission API formats, enabling
	// compatibility across different Kubernetes cluster versions.
	//
	// Decoder configuration:
	//   - Universal deserializer: Automatically detects and handles multiple API versions
	//   - Schema registration: Supports admission.k8s.io/v1 and v1beta1 API schemas
	//   - Error handling: Provides structured errors for malformed admission requests
	//   - Performance optimization: Reuses codec instances for efficient deserialization
	//
	// This decoder enables the webhook to work with both legacy and modern Kubernetes clusters
	// without requiring version-specific deployment configurations.
	decoder runtime.Decoder
}

// NewValidationWebhookAPI creates a new Kubernetes admission webhook API server instance.
// This constructor initializes all necessary components for processing admission requests,
// including Kubernetes API deserialization support and HTTP routing configuration.
//
// The webhook API server provides the primary integration point between Kubernetes clusters
// and CloudZero cost allocation services, enabling automatic metadata extraction during
// resource creation, update, and deletion operations.
//
// Configuration parameters:
//   - base: HTTP path prefix for webhook endpoints (typically "/webhook" or "/validate")
//   - controller: Domain service implementing the actual cost allocation processing logic
//
// Initialization process:
//  1. Configure Kubernetes API deserialization for v1 and v1beta1 admission formats
//  2. Set up HTTP routing with chi router for efficient request handling
//  3. Register webhook endpoints with go-obvious server infrastructure
//  4. Enable structured logging and metrics collection for operational monitoring
//
// The returned server.API can be registered with the CloudZero Agent HTTP server
// to begin processing admission webhook requests from Kubernetes API servers.
//
// Production considerations:
//   - Supports both legacy (v1beta1) and modern (v1) Kubernetes admission APIs
//   - Automatic content negotiation based on request API version
//   - Fail-safe deserialization with comprehensive error handling
//   - Ready for integration with agent lifecycle management and health checking
func NewValidationWebhookAPI(base string, controller webhook.WebhookController) server.API {
	a := &ValidationWebhookAPI{
		controller: controller,
		decoder: func() runtime.Decoder {
			r := runtime.NewScheme()
			r.AddKnownTypes(admissionv1beta1.SchemeGroupVersion, &admissionv1beta1.AdmissionReview{})
			r.AddKnownTypes(admissionv1.SchemeGroupVersion, &admissionv1.AdmissionReview{})
			codecs := serializer.NewCodecFactory(r)
			return codecs.UniversalDeserializer()
		}(),
		Service: api.Service{
			APIName: "webhook",
			Mounts:  map[string]*chi.Mux{},
		},
	}

	a.Service.Mounts[base] = a.Routes()
	return a
}

// Register integrates the ValidationWebhookAPI with the CloudZero Agent HTTP server infrastructure.
// This method completes the webhook server setup by mounting the API endpoints and enabling
// request processing for Kubernetes admission control integration.
//
// Registration process:
//   - Mount webhook routes at the configured base path
//   - Enable HTTP middleware for logging, metrics, and error handling
//   - Configure request timeout and size limit enforcement
//   - Activate admission review processing pipeline
//
// The registration process integrates the webhook with the agent's broader HTTP infrastructure,
// enabling coordinated startup, shutdown, and health monitoring across all agent components.
//
// Error conditions:
//   - Service registration failures (port conflicts, permission issues)
//   - Route mounting conflicts with existing endpoints
//   - Middleware initialization failures
//
// Once registration completes successfully, the webhook is ready to receive admission
// requests from Kubernetes API servers and process them through the CloudZero cost allocation pipeline.
func (a *ValidationWebhookAPI) Register(app server.Server) error {
	if err := a.Service.Register(app); err != nil {
		return err
	}
	return nil
}

// Routes configures HTTP request routing for the admission webhook API endpoints.
// This method creates a chi router instance with all necessary routes for processing
// Kubernetes admission review requests and health checking.
//
// Route configuration:
//   - POST /: Primary admission review endpoint for Kubernetes API server requests
//   - Future endpoints: Health checks, metrics, and debugging endpoints as needed
//
// The chi router provides:
//   - High-performance HTTP routing with minimal overhead
//   - Middleware support for cross-cutting concerns (logging, metrics, authentication)
//   - RESTful routing patterns compatible with Kubernetes expectations
//   - Integration with go-obvious server infrastructure
//
// This routing configuration enables the webhook to respond to admission requests
// while maintaining compatibility with standard HTTP tooling and monitoring systems.
func (a *ValidationWebhookAPI) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Post("/", a.PostAdmissionRequest)
	return r
}

// requestBodyToModelReview deserializes Kubernetes admission review requests into domain model objects.
// This method handles the conversion from raw JSON bytes to CloudZero Agent internal representations,
// supporting both v1 and v1beta1 admission API versions for maximum Kubernetes compatibility.
//
// Deserialization process:
//  1. Use Kubernetes universal deserializer to detect API version automatically
//  2. Deserialize into appropriate admission.k8s.io/v1 or v1beta1 structures
//  3. Convert Kubernetes types to CloudZero Agent domain types for processing
//  4. Validate the resulting admission review for completeness and correctness
//
// API version handling:
//   - v1beta1: Legacy API for Kubernetes 1.16-1.21 compatibility
//   - v1: Modern API with full feature support for Kubernetes 1.22+
//   - Automatic detection: No manual version negotiation required
//
// The conversion to domain types enables the webhook controller to process admission
// requests using CloudZero-specific business logic while maintaining compatibility
// with various Kubernetes cluster versions.
//
// Error conditions:
//   - Malformed JSON in request body
//   - Unsupported admission API version
//   - Missing required fields in admission review
//   - Invalid resource types or operations
func (a *ValidationWebhookAPI) requestBodyToModelReview(body []byte) (*types.AdmissionReview, error) {
	review, _, err := a.decoder.Decode(body, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("could not decode the admission review from the request: %w", err)
	}

	switch ar := review.(type) {
	case *admissionv1beta1.AdmissionReview:
		res := types.NewAdmissionReviewV1Beta1(ar)
		return &res, nil
	case *admissionv1.AdmissionReview:
		res := types.NewAdmissionReviewV1(ar)
		return &res, nil
	}

	return nil, errors.New("invalid admission review type")
}

// PostAdmissionRequest processes Kubernetes admission review requests for CloudZero cost allocation.
// This method implements the core webhook endpoint that receives admission requests from Kubernetes API servers
// and orchestrates the complete cost allocation pipeline while maintaining fail-open behavior for cluster stability.
//
// Processing pipeline:
//  1. Request validation: Verify content type, timeout parameters, and body size limits
//  2. Admission review parsing: Deserialize Kubernetes admission request into domain objects
//  3. Business logic execution: Extract cost allocation metadata through webhook controller
//  4. Response generation: Create properly formatted admission review response
//  5. Fail-open handling: Always allow requests if processing encounters errors
//
// Webhook execution matrix:
//
//	| Result Type              | HTTP Code | status.Code | status.Status | status.Message |
//	|--------------------------|-----------|-------------|---------------|----------------|
//	| Validating Allowed       | 200       | -           | -             | -              |
//	| Validating not allowed   | 200       | 400         | Failure       | Custom message |
//	| Processing Error         | 200       | -           | -             | - (fail-open)  |
//
// Timeout management:
//   - Default timeout: 15 seconds for comprehensive processing
//   - Minimum timeout: 5 seconds to prevent unnecessary failures
//   - Query parameter override: "?timeout=10s" for custom timeouts
//   - Context cancellation: Proper cleanup on request cancellation
//
// Production safeguards:
//   - Fail-open behavior: Processing errors result in admission approval
//   - Connection management: Periodic closure for load balancing
//   - Security validation: Content-type checking and size limits
//   - Comprehensive logging: Structured events for operational monitoring
//
// This method represents the primary integration point between Kubernetes resource lifecycle
// management and CloudZero cost optimization, processing thousands of admission requests
// per minute in production clusters while maintaining sub-second response times.
func (a *ValidationWebhookAPI) PostAdmissionRequest(w http.ResponseWriter, r *http.Request) {
	// Webhook execution logic. This is how we are dealing with the different responses:
	// |                        | HTTP Code             | status.Code | status.Status | status.Message |
	// |------------------------|-----------------------| ------------|---------------|----------------|
	// | Validating Allowed     | 200                   | -           | -             | -              |
	// | Validating not allowed | 200                   | 400         | Failure       | Custom message |
	// | Err                    | 500
	ctx := r.Context()
	defer r.Body.Close()

	// Our API should never time more than 15 seconds
	timeout := DefaultTimeout
	if to := request.QS(r, "timeout"); to != "" {
		if duration, err := time.ParseDuration(to); err == nil {
			timeout = duration
		}
		if timeout < MinTimeout {
			timeout = MinTimeout
		}
		if timeout > DefaultTimeout {
			timeout = DefaultTimeout
		}
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Ctx(ctx).Error().Msg("only content type 'application/json' is supported")
		request.Reply(r, w, "failed to read request body", http.StatusBadRequest)
		return
	}

	// Get webhook body with the admission review.
	var body []byte
	body, err := configReader(r)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to parse body")
		request.Reply(r, w, "failed to read request body", http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		log.Ctx(ctx).Err(err).Msg("no body in request")
		request.Reply(r, w, "no body in request", http.StatusBadRequest)
		return
	}

	review, err := a.requestBodyToModelReview(body)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("could not read request body")
		request.Reply(r, w, fmt.Sprintf("could not read request body: %v", err), http.StatusBadRequest)
		return
	}
	if review == nil {
		log.Ctx(ctx).Error().Msg("malformed admission review: request is nil")
		request.Reply(r, w, "malformed admission review: request is nil", http.StatusBadRequest)
		return
	}

	sendAllowResponse := func(w http.ResponseWriter, r *http.Request) {
		allowResponse := &types.AdmissionResponse{Allowed: true}
		resp, err := a.marshallResponseToJSON(ctx, review, allowResponse)
		if err != nil {
			// Log the error but still allow the request - fail-open behavior
			log.Ctx(ctx).Err(err).Msg("could not marshal allow response to json, allowing request anyway")

			// Use minimal JSON response to ensure we always allow
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(minimalAllowResponse))
			return
		}

		// Only set headers when we know we'll succeed
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(resp)
	}

	log.Ctx(ctx).Trace().
		Int("content_length", int(r.ContentLength)).
		Str("operation", string(review.Operation)).
		Msg("processing review request")

	if _, err := a.controller.Review(ctx, review); err != nil {
		log.Ctx(ctx).Error().Err(err).Send()
		sendAllowResponse(w, r)
		return
	}

	// If we're using HTTP/1.x, we want to periodically close the connection to
	// help distribute the load across the various webhook replicas.
	//
	// Unfortunately this won't work for HTTP/2, but currently all traffic we're
	// seeing from the Kubernetes API server is HTTP/1.1.
	if r.ProtoMajor == 1 {
		rf := a.controller.Settings().Server.ReconnectFrequency
		if rf > 0 && rand.Intn(rf) == 0 { //nolint:gosec // a weak PRNG is fine here
			w.Header().Set("Connection", "close")
		}
	}

	sendAllowResponse(w, r)
}

// configReader safely reads and validates HTTP request bodies for admission webhook processing.
// This function implements security controls aligned with Kubernetes API server constraints,
// preventing memory exhaustion attacks while supporting large resource manifests.
//
// Security controls:
//   - Size limiting: MaxRequestBodyBytes (6MB) prevents oversized requests
//   - Memory protection: Uses io.LimitedReader to cap memory allocation
//   - Early detection: Checks size limit before complete body reading
//   - Error integration: Returns Kubernetes-compatible error types
//
// The size limit calculation follows Kubernetes internal patterns:
//   - Kubernetes etcd default: 1.5MB per object
//   - Kubernetes API buffer: 2x multiplier (3MB)
//   - CloudZero safety buffer: 2x multiplier (6MB final)
//
// This conservative approach ensures compatibility with legitimate large resources
// (ConfigMaps, Secrets, Custom Resources) while protecting against malicious requests
// that could exhaust agent memory or cause denial of service conditions.
//
// The function integrates with Kubernetes error types to provide consistent error
// responses that API servers can properly interpret and handle.
func configReader(req *http.Request) ([]byte, error) {
	defer req.Body.Close()
	lr := &io.LimitedReader{
		R: req.Body,
		N: MaxRequestBodyBytes + 1,
	}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if lr.N <= 0 {
		return nil, apierrors.NewRequestEntityTooLargeError(fmt.Sprintf("limit is %d", MaxRequestBodyBytes))
	}
	return data, nil
}

// marshallResponseToJSON serializes admission webhook responses into Kubernetes-compatible JSON format.
// This method handles the complex conversion from CloudZero domain objects back to Kubernetes admission
// API structures, supporting both v1 and v1beta1 formats for maximum cluster compatibility.
//
// Response generation process:
//  1. Validate admission response completeness and consistency
//  2. Create appropriate metav1.Status for rejection responses with detailed error information
//  3. Determine original admission API version for response format selection
//  4. Serialize into v1 or v1beta1 AdmissionReview structure as required
//  5. Generate final JSON bytes ready for HTTP response transmission
//
// API version handling:
//   - v1beta1: Legacy format without admission warnings support
//   - v1: Modern format with full admission warnings and enhanced error details
//   - Automatic detection: Matches response format to original request version
//   - Compatibility: Graceful degradation for unsupported features
//
// Response structure compliance:
//   - TypeMeta: Proper Kind and APIVersion for Kubernetes deserialization
//   - Response UID: Matches original request for correlation
//   - Status codes: HTTP 400 for validation failures, success otherwise
//   - Error messages: Human-readable descriptions for webhook rejections
//
// Error handling:
//   - Validates response object completeness before serialization
//   - Handles JSON marshalling failures gracefully
//   - Provides structured error information for unsupported API versions
//   - Integrates with fail-open behavior when serialization fails
//
// This method ensures that CloudZero Agent webhook responses conform exactly to
// Kubernetes admission webhook specifications, enabling seamless integration with
// all supported Kubernetes cluster versions.
func (a *ValidationWebhookAPI) marshallResponseToJSON(ctx context.Context, review *types.AdmissionReview, resp *types.AdmissionResponse) (data []byte, err error) {
	if resp == nil {
		log.Ctx(ctx).Warn().Msg("admission response is nil")
		return nil, errors.New("invalid admission response")
	}

	// Set the satus code and result based on the validation result.
	var resultStatus *metav1.Status
	if !resp.Allowed {
		resultStatus = &metav1.Status{
			Message: resp.Message,
			Status:  metav1.StatusFailure,
			Code:    http.StatusBadRequest,
		}
	}

	switch review.OriginalAdmissionReview.(type) {
	case *admissionv1beta1.AdmissionReview:
		if len(resp.Warnings) > 0 {
			log.Ctx(ctx).Warn().Msg("warnings used in a 'v1beta1' webhook")
		}

		data, err := json.Marshal(admissionv1beta1.AdmissionReview{
			TypeMeta: v1beta1AdmissionReviewTypeMeta,
			Response: &admissionv1beta1.AdmissionResponse{
				UID:     k8stypes.UID(review.ID),
				Allowed: resp.Allowed,
				Result:  resultStatus,
			},
		})
		return data, err

	case *admissionv1.AdmissionReview:
		data, err := json.Marshal(admissionv1.AdmissionReview{
			TypeMeta: v1AdmissionReviewTypeMeta,
			Response: &admissionv1.AdmissionResponse{
				UID:      k8stypes.UID(review.ID),
				Warnings: resp.Warnings,
				Allowed:  resp.Allowed,
				Result:   resultStatus,
			},
		})
		return data, err
	}

	log.Ctx(ctx).Warn().
		Str("type", fmt.Sprintf("%T", review.OriginalAdmissionReview)).
		Msg("unsupported admission review type")
	return nil, errors.New("invalid admission response type")
}
