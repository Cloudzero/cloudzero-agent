// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package diagnostic provides a comprehensive diagnostic framework for monitoring and
// validating the operational health of CloudZero agent components and their dependencies.
//
// This package defines the core interface for diagnostic providers that perform targeted
// health checks across different aspects of the agent's runtime environment:
//
//   - Environment validation: Configuration and runtime environment checks
//   - External service connectivity: CloudZero API, Kubernetes API, Prometheus connectivity
//   - Resource availability: Memory, disk space, network access
//   - Component functionality: Collector, shipper, webhook server health
//   - Security validation: TLS certificates, authentication tokens, permissions
//
// The diagnostic system is designed around a pluggable provider model where each
// provider focuses on a specific aspect of system health:
//
// Diagnostic categories:
//   - CloudZero integration: API connectivity, authentication, data upload
//   - Kubernetes integration: API access, permissions, resource discovery
//   - Prometheus integration: Remote write connectivity, configuration validation
//   - System resources: Disk space, memory usage, network connectivity
//   - Security components: Certificate validity, encryption status
//
// Provider implementation pattern:
//   Each diagnostic provider implements the Provider interface to perform
//   targeted health checks and populate status information that can be
//   consumed by monitoring systems, APIs, and operational dashboards.
//
// Integration with status system:
//   Diagnostic results are aggregated into a unified status structure that
//   provides a comprehensive view of agent health and can be exposed through
//   various channels (HTTP endpoints, logs, telemetry).
//
// Usage:
//   providers := []diagnostic.Provider{
//       cz.NewCheck(config),
//       k8s.NewVersionCheck(),
//       prom.NewConfigCheck(config),
//   }
//   
//   for _, provider := range providers {
//       if err := provider.Check(ctx, httpClient, statusAccessor); err != nil {
//           log.Printf("Diagnostic check failed: %v", err)
//       }
//   }
package diagnostic

import (
	"context"
	"net/http"

	"github.com/cloudzero/cloudzero-agent/app/types/status"
)

// Provider defines the interface that must be implemented by diagnostic providers
// to perform health checks and populate status information.
//
// Each provider is responsible for checking a specific aspect of system health
// and updating the status accessor with relevant diagnostic information. Providers
// should be focused, reliable, and fail gracefully.
//
// Implementation requirements:
//   - Targeted scope: Each provider should check a specific system aspect
//   - Error handling: Only return errors for unrecoverable conditions
//   - Status population: Use the accessor to record check results
//   - Context respect: Honor context cancellation and timeouts
//   - Resource efficiency: Minimize resource usage and execution time
//
// Design principles:
//   - Fail-safe: Diagnostic failures should not impact agent functionality
//   - Informative: Provide meaningful status information for operations teams
//   - Efficient: Execute quickly to avoid blocking critical operations
//   - Isolated: Each provider operates independently of others
type Provider interface {
	// Check performs diagnostic validation and updates status information.
	//
	// This method executes the provider's health check logic and populates
	// the status accessor with diagnostic results. The check should be:
	//   - Non-blocking: Complete quickly to avoid impacting agent operations
	//   - Informative: Provide useful diagnostic information via status accessor
	//   - Resilient: Handle failures gracefully and continue when possible
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout control
	//   - client: HTTP client for external service connectivity checks
	//   - accessor: Interface for updating diagnostic status information
	//
	// Returns:
	//   - nil: Check completed successfully (results recorded in accessor)
	//   - error: Unrecoverable error that prevents check execution
	//
	// Error handling:
	//   - Recoverable issues: Record in status accessor, return nil
	//   - Unrecoverable issues: Return error to halt diagnostic execution
	//   - Network timeouts: Generally recoverable, record and continue
	//   - Configuration errors: Usually unrecoverable, return error
	//
	// Example implementation:
	//   func (p *MyProvider) Check(ctx context.Context, client *http.Client, accessor status.Accessor) error {
	//       result := p.performCheck(ctx, client)
	//       accessor.UpdateDiagnostic("my-check", result)
	//       return nil // Only return error for unrecoverable conditions
	//   }
	Check(_ context.Context, _ *http.Client, _ status.Accessor) error
}
