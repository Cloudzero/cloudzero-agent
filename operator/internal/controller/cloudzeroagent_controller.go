// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	agentv1alpha1 "github.com/cloudzero/cloudzero-agent/operator/api/v1alpha1"
	"github.com/cloudzero/cloudzero-agent/operator/internal/certmanager"
	"github.com/cloudzero/cloudzero-agent/operator/internal/metricsapi"
)

const (
	// ConditionCertificateValid is the condition type used to report TLS certificate health.
	ConditionCertificateValid = "CertificateValid"

	// ConditionMemoryPressure is the condition type used to report component memory pressure.
	ConditionMemoryPressure = "MemoryPressure"

	// defaultRequeueInterval is how often the reconciler re-checks state.
	defaultRequeueInterval = 24 * time.Hour
)

// CloudZeroAgentReconciler reconciles a CloudZeroAgent object.
type CloudZeroAgentReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	CertManager   certmanager.CertManager
	MetricsClient metricsapi.MetricsClient // nil when metrics server is unavailable or disabled
	EventRecorder record.EventRecorder
}

// +kubebuilder:rbac:groups=agent.cloudzero.com,resources=cloudzeroagents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agent.cloudzero.com,resources=cloudzeroagents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agent.cloudzero.com,resources=cloudzeroagents/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile ensures the CloudZeroAgent's desired state matches the cluster's actual state.
// It reconciles TLS certificates and, if a MetricsClient is configured, memory pressure.
// All status mutations are accumulated in-memory and flushed in a single Status().Update() call.
func (r *CloudZeroAgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var agent agentv1alpha1.CloudZeroAgent
	if err := r.Get(ctx, req.NamespacedName, &agent); err != nil {
		if apierrors.IsNotFound(err) {
			// Resource was deleted — nothing to do.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Reconcile TLS certificate state (modifies agent.Status.Conditions in-memory).
	r.reconcileTLS(ctx, &agent)

	// Reconcile memory pressure if a metrics client is available and resource management is configured.
	logger.Info("resource management check",
		"metricsClientNil", r.MetricsClient == nil,
		"resourceManagementNil", agent.Spec.ResourceManagement == nil,
	)
	if r.MetricsClient != nil && agent.Spec.ResourceManagement != nil {
		r.reconcileResourceManagement(ctx, &agent)
	}

	// DEBUG: switched to Update to rule out MergeFrom patch being a no-op for componentMemory.
	logger.Info("status update: componentMemory count", "count", len(agent.Status.ComponentMemory))
	if err := r.Status().Update(ctx, &agent); err != nil {
		logger.Error(err, "Failed to update CloudZeroAgent status")
		return ctrl.Result{}, fmt.Errorf("failed to update CloudZeroAgent status: %w", err)
	}

	return ctrl.Result{RequeueAfter: defaultRequeueInterval}, nil
}

// reconcileTLS runs the TLS certificate state machine, setting the CertificateValid condition.
// All changes are applied to agent in-memory; no API calls are made.
func (r *CloudZeroAgentReconciler) reconcileTLS(ctx context.Context, agent *agentv1alpha1.CloudZeroAgent) {
	logger := log.FromContext(ctx)

	switch agent.Spec.TLS.Mode {
	case agentv1alpha1.TLSModeCertManager, agentv1alpha1.TLSModeUserSupplied:
		logger.Info("TLS certificates are externally managed, skipping cert reconciliation",
			"mode", agent.Spec.TLS.Mode)
		setCondition(agent, ConditionCertificateValid, metav1.ConditionUnknown,
			"ExternallyManaged",
			fmt.Sprintf("TLS certificates are managed externally (mode: %s)", agent.Spec.TLS.Mode),
		)

	case agentv1alpha1.TLSModeManaged:
		r.reconcileManagedTLS(ctx, agent)

	default:
		setCondition(agent, ConditionCertificateValid, metav1.ConditionFalse,
			"InvalidMode",
			fmt.Sprintf("Unknown TLS mode %q — valid values are: managed, cert-manager, user-supplied", agent.Spec.TLS.Mode),
		)
	}
}

// reconcileManagedTLS handles the "managed" TLS mode: checks cert presence and expiry,
// generates a new cert if needed, and updates the Secret and webhook config.
// All changes are applied to agent in-memory; no API calls are made except via CertManager.
func (r *CloudZeroAgentReconciler) reconcileManagedTLS(ctx context.Context, agent *agentv1alpha1.CloudZeroAgent) {
	logger := log.FromContext(ctx)
	tls := agent.Spec.TLS
	namespace := agent.Namespace

	renewalThreshold, err := time.ParseDuration(tls.RenewalThreshold)
	if err != nil {
		setCondition(agent, ConditionCertificateValid, metav1.ConditionFalse,
			"InvalidConfig",
			fmt.Sprintf("spec.tls.renewalThreshold %q is not a valid Go duration: %v", tls.RenewalThreshold, err),
		)
		return
	}

	validityDuration, err := time.ParseDuration(tls.ValidityDuration)
	if err != nil {
		setCondition(agent, ConditionCertificateValid, metav1.ConditionFalse,
			"InvalidConfig",
			fmt.Sprintf("spec.tls.validityDuration %q is not a valid Go duration: %v", tls.ValidityDuration, err),
		)
		return
	}

	// Check whether the secret already has cert fields populated.
	exists, err := r.CertManager.ValidateExisting(ctx, namespace, tls.SecretName)
	if err != nil {
		logger.Info("Could not validate existing certificate, will attempt to generate",
			"secret", tls.SecretName, "error", err)
		exists = false
	}

	if exists {
		valid, err := r.CertManager.ValidateExpiry(ctx, namespace, tls.SecretName, renewalThreshold)
		if err != nil {
			logger.Error(err, "Failed to parse existing certificate, will regenerate",
				"secret", tls.SecretName)
			// Treat unparseable cert as needing regeneration — fall through.
		} else if valid {
			logger.V(1).Info("TLS certificate is current, no action needed")
			setCondition(agent, ConditionCertificateValid, metav1.ConditionTrue,
				"CertificateCurrent",
				"TLS certificate exists and is not near expiry",
			)
			return
		} else {
			logger.Info("TLS certificate is expired or within renewal threshold, regenerating",
				"secret", tls.SecretName, "renewalThreshold", renewalThreshold)
		}
	}

	// Generate a new certificate.
	logger.Info("Generating new TLS certificate",
		"serviceName", tls.ServiceName, "namespace", namespace, "algorithm", tls.Algorithm)

	certData, err := r.CertManager.Generate(ctx, tls.ServiceName, namespace, tls.KeySize, validityDuration, tls.Algorithm)
	if err != nil {
		setCondition(agent, ConditionCertificateValid, metav1.ConditionFalse,
			"GenerationFailed",
			fmt.Sprintf("Failed to generate TLS certificate: %v", err),
		)
		return
	}

	if err := r.CertManager.UpdateResources(ctx, namespace, tls.SecretName, tls.WebhookName, certData); err != nil {
		setCondition(agent, ConditionCertificateValid, metav1.ConditionFalse,
			"InstallFailed",
			fmt.Sprintf("Failed to install TLS certificate: %v", err),
		)
		return
	}

	logger.Info("TLS certificate generated and installed successfully",
		"secret", tls.SecretName, "webhook", tls.WebhookName)
	setCondition(agent, ConditionCertificateValid, metav1.ConditionTrue,
		"CertificateInstalled",
		"TLS certificate generated and installed successfully",
	)
}

// setCondition sets a condition on agent.Status.Conditions in-memory.
// No API calls are made; the caller is responsible for flushing via Status().Update().
func setCondition(agent *agentv1alpha1.CloudZeroAgent, condType string, status metav1.ConditionStatus, reason, message string) {
	apimeta.SetStatusCondition(&agent.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: agent.Generation,
		Reason:             reason,
		Message:            message,
	})
}

// SetupWithManager registers the reconciler with the controller manager.
// GenerationChangedPredicate filters out watch events caused by status-only updates
// (status changes do not increment metadata.generation), which prevents the reconciler
// from triggering itself in a tight loop after each Status().Patch() call.
// Timer-based requeues (RequeueAfter) are unaffected by this predicate.
func (r *CloudZeroAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&agentv1alpha1.CloudZeroAgent{},
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Named("cloudzeroagent").
		Complete(r)
}
