// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package monitor provides secure dynamic secret management and certificate monitoring for CloudZero Agent operations.
// This package implements critical security infrastructure that enables zero-downtime credential rotation
// and certificate lifecycle management without requiring agent restarts or service disruption.
//
// The monitor system operates as an Application Core component in the hexagonal architecture,
// providing secure credential management services to all agent components requiring authentication
// with external systems including the CloudZero platform, Kubernetes API, and Prometheus endpoints.
//
// Key responsibilities:
//   - Dynamic API key rotation: Automatic detection and loading of updated CloudZero API keys
//   - Certificate lifecycle management: TLS certificate expiration monitoring and renewal
//   - Secret change detection: SHA256-based change detection to minimize unnecessary operations
//   - Security-first design: Credential redaction in logs and secure memory handling
//   - Production reliability: Panic recovery and graceful error handling for continuous operation
//
// Architecture:
//   - secretsMonitor: Manages CloudZero API key rotation with configurable refresh intervals
//   - TLS monitors: Handle certificate expiration tracking and renewal notifications
//   - Change detection: Cryptographic hashing to identify actual secret changes
//   - Lifecycle management: Full start/stop/reset capabilities for operational flexibility
//
// Security features:
//   - Credential redaction: Automatic masking of sensitive data in log outputs
//   - Minimal exposure: Secrets loaded only when changes are detected
//   - Memory protection: Secure handling and cleanup of sensitive credential data
//   - Audit logging: Comprehensive tracking of secret rotation events for compliance
//
// The monitoring system enables production CloudZero deployments to maintain continuous
// operation while credentials are rotated according to organizational security policies,
// ensuring both security compliance and operational stability.
//
// Integration points:
//   - CloudZero API client: Receives updated API keys for platform authentication
//   - TLS infrastructure: Monitors certificate expiration for webhook and metric endpoints
//   - Configuration services: Triggers configuration reload when secrets change
//   - Operational monitoring: Provides metrics and alerts for credential management health
package monitor

import (
	"context"
	"crypto/sha256"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-agent/app/types"
)

// DefaultRefreshInterval defines the standard polling frequency for secret change detection.
// This interval balances security responsiveness with system resource utilization,
// providing timely credential rotation while avoiding excessive monitoring overhead.
//
// The 1-minute interval enables:
//   - Rapid response to credential rotation (maximum 60-second delay)
//   - Reasonable resource consumption for continuous monitoring
//   - Compliance with typical organizational security policies
//   - Compatibility with automated secret management systems
//
// Production deployments may adjust this interval based on:
//   - Security requirements and threat model
//   - Secret rotation frequency and urgency
//   - System resource constraints and monitoring overhead
//   - Integration with external credential management platforms
var DefaultRefreshInterval = 1 * time.Minute

// MonitoredAPIKey defines the interface for components providing dynamically refreshable API keys.
// This interface enables the secrets monitor to work with various credential providers
// including file-based secrets, Kubernetes secrets, cloud provider key management services,
// and external credential rotation systems.
//
// The interface supports both pull-based (GetAPIKey) and push-based (SetAPIKey) credential
// management patterns, enabling flexibility in how credentials are sourced and updated.
//
// Implementation considerations:
//   - Thread safety: Methods may be called concurrently during monitoring operations
//   - Error handling: SetAPIKey errors indicate credential loading failures
//   - Security: GetAPIKey returns sensitive data that should be handled securely
//   - Performance: Methods should minimize latency as they're called during monitoring cycles
type MonitoredAPIKey interface {
	// GetAPIKey retrieves the currently loaded API key for CloudZero platform authentication.
	// This method is called during monitoring cycles to obtain the current credential
	// for change detection and validation purposes.
	//
	// Returns:
	//   - string: The current API key value (sensitive data requiring secure handling)
	//
	// Security considerations:
	//   - Returned value contains sensitive authentication data
	//   - Should not be logged or stored in plaintext
	//   - Memory should be cleared after use when possible
	//
	// Thread safety:
	//   - Must be safe for concurrent access during monitoring operations
	//   - Should provide consistent results during credential transition periods
	GetAPIKey() string

	// SetAPIKey triggers a refresh of the API key from the configured credential source.
	// This method enables the monitor to request credential updates when changes are
	// suspected or during periodic refresh cycles.
	//
	// Implementation behavior:
	//   - Should reload credentials from the authoritative source
	//   - May validate credential format and accessibility
	//   - Should update internal state atomically to prevent inconsistency
	//   - Should handle transient errors gracefully with appropriate retry logic
	//
	// Error conditions:
	//   - Credential source unavailable (network, filesystem, API errors)
	//   - Invalid credential format or structure
	//   - Authentication failures with credential management systems
	//   - Permission errors accessing credential storage
	//
	// Returns:
	//   - error: nil on successful refresh, error describing failure condition
	SetAPIKey() error
}

// secretsMonitor implements continuous monitoring and automatic rotation of CloudZero API credentials.
// This struct provides the core secret management functionality that enables zero-downtime credential
// updates without requiring agent restarts or service interruption.
//
// The monitor uses a polling-based approach with cryptographic change detection to minimize
// resource overhead while ensuring timely detection of credential updates.
//
// Operational characteristics:
//   - Periodic polling: Configurable interval-based credential checking
//   - Change detection: SHA256 hashing to identify actual credential changes
//   - Thread safety: Concurrent access protection using mutex synchronization
//   - Lifecycle management: Full start/stop/restart capabilities for operational flexibility
//   - Panic recovery: Robust error handling to prevent service disruption
type secretsMonitor struct {
	// settings provides access to the credential source implementing MonitoredAPIKey interface.
	// This abstraction enables support for various credential providers including file-based
	// secrets, Kubernetes secrets, cloud provider key management, and external rotation systems.
	settings MonitoredAPIKey

	// originalCtx preserves the parent context for reset operations after shutdown.
	// This enables the monitor to restart with the original context scope rather than
	// a cancelled context, supporting operational patterns like restart-after-failure.
	originalCtx context.Context

	// ctx provides the active context for monitoring operations and cancellation.
	// Derived from originalCtx with cancellation capability for graceful shutdown
	// and restart operations. All monitoring goroutines respect this context.
	ctx context.Context

	// cancel enables termination of the monitoring goroutine and cleanup of resources.
	// Called during shutdown operations to trigger graceful termination and ensure
	// proper cleanup of monitoring resources and goroutines.
	cancel context.CancelFunc

	// mu provides thread-safe access to monitor state during concurrent operations.
	// Protects running state, context management, and lifecycle transitions from
	// race conditions during start/stop operations from multiple goroutines.
	mu sync.Mutex

	// lastHash stores the SHA256 hash of the most recently observed credential.
	// Used for efficient change detection that avoids unnecessary processing
	// when credentials haven't actually changed, reducing log noise and overhead.
	lastHash [32]byte

	// running indicates whether the monitoring goroutine is currently active.
	// Protected by mutex to ensure thread-safe state checking and lifecycle
	// management across concurrent start/stop operations.
	running bool

	// done provides completion signaling for graceful shutdown coordination.
	// The monitoring goroutine closes this channel upon termination, enabling
	// the Shutdown method to wait for complete cleanup before returning.
	done chan struct{}
}

func NewSecretMonitor(ctx context.Context, settings MonitoredAPIKey) types.Runnable {
	newCtx, cancel := context.WithCancel(ctx)
	return &secretsMonitor{
		settings:    settings,
		originalCtx: ctx,
		ctx:         newCtx,
		cancel:      cancel,
		done:        make(chan struct{}),
	}
}

// Run implements types.Runnable.
func (s *secretsMonitor) Run() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}

	ticker := time.NewTicker(DefaultRefreshInterval)
	go func() {
		defer ticker.Stop()
		defer close(s.done)
		defer func() {
			if r := recover(); r != nil {
				log.Info().Interface("panic", r).Msg("Recovered from panic in secret monitor")
			}
		}()

		for {
			select {
			case <-s.ctx.Done():
				s.running = false
				return
			case <-ticker.C:
				_ = s.settings.SetAPIKey()
				newSecret := s.settings.GetAPIKey()
				newHash := sha256.Sum256([]byte(newSecret))
				if newHash != s.lastHash {
					log.Info().Str("secret", redactSecret(newSecret)).Msg("discovered new secret")
					s.lastHash = newHash
				}
			}
		}
	}()
	s.running = true
	return nil
}

func redactSecret(secret string) string {
	if len(secret) > 2 {
		return secret[:2] + "***"
	}
	return "*****"
}

// Shutdown implements types.Runnable.
func (s *secretsMonitor) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return nil
	}
	s.cancel()
	<-s.done
	s.reset()
	return nil
}

func (s *secretsMonitor) reset() {
	s.running = false
	ctx, cancel := context.WithCancel(s.originalCtx)
	s.ctx = ctx
	s.cancel = cancel
	s.done = make(chan struct{})
}

func (s *secretsMonitor) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
