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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	agentv1alpha1 "github.com/cloudzero/cloudzero-agent/operator/api/v1alpha1"
	"github.com/cloudzero/cloudzero-agent/operator/internal/certmanager"
)

const (
	// ConditionCertificateValid is the condition type used to report TLS certificate health.
	ConditionCertificateValid = "CertificateValid"

	// defaultRequeueInterval is how often the reconciler re-checks certificate state.
	defaultRequeueInterval = 24 * time.Hour
)

// CloudZeroAgentReconciler reconciles a CloudZeroAgent object.
type CloudZeroAgentReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	CertManager certmanager.CertManager
}

// +kubebuilder:rbac:groups=agent.cloudzero.com,resources=cloudzeroagents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=agent.cloudzero.com,resources=cloudzeroagents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=agent.cloudzero.com,resources=cloudzeroagents/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=validatingwebhookconfigurations,verbs=get;list;watch;patch

// Reconcile ensures the CloudZeroAgent's TLS certificate state matches the desired spec.
// It runs on every change to a CloudZeroAgent resource and on a periodic requeue timer.
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

	switch agent.Spec.TLS.Mode {
	case agentv1alpha1.TLSModeCertManager, agentv1alpha1.TLSModeUserSupplied:
		logger.Info("TLS certificates are externally managed, skipping cert reconciliation",
			"mode", agent.Spec.TLS.Mode)
		return r.setCondition(ctx, &agent, metav1.ConditionUnknown,
			"ExternallyManaged",
			fmt.Sprintf("TLS certificates are managed externally (mode: %s)", agent.Spec.TLS.Mode),
		)

	case agentv1alpha1.TLSModeManaged:
		return r.reconcileManaged(ctx, &agent)

	default:
		return r.setCondition(ctx, &agent, metav1.ConditionFalse,
			"InvalidMode",
			fmt.Sprintf("Unknown TLS mode %q — valid values are: managed, cert-manager, user-supplied", agent.Spec.TLS.Mode),
		)
	}
}

// reconcileManaged handles the "managed" TLS mode: checks cert presence and expiry,
// generates a new cert if needed, and updates the Secret and webhook config.
func (r *CloudZeroAgentReconciler) reconcileManaged(ctx context.Context, agent *agentv1alpha1.CloudZeroAgent) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	tls := agent.Spec.TLS
	namespace := agent.Namespace

	renewalThreshold, err := time.ParseDuration(tls.RenewalThreshold)
	if err != nil {
		return r.setCondition(ctx, agent, metav1.ConditionFalse,
			"InvalidConfig",
			fmt.Sprintf("spec.tls.renewalThreshold %q is not a valid Go duration: %v", tls.RenewalThreshold, err),
		)
	}

	validityDuration, err := time.ParseDuration(tls.ValidityDuration)
	if err != nil {
		return r.setCondition(ctx, agent, metav1.ConditionFalse,
			"InvalidConfig",
			fmt.Sprintf("spec.tls.validityDuration %q is not a valid Go duration: %v", tls.ValidityDuration, err),
		)
	}

	// Check whether the secret already has cert fields populated.
	exists, err := r.CertManager.ValidateExisting(ctx, namespace, tls.SecretName)
	if err != nil {
		// Secret may not exist yet or is inaccessible — attempt to generate.
		logger.Info("Could not validate existing certificate, will attempt to generate",
			"secret", tls.SecretName, "error", err)
		exists = false
	}

	if exists {
		// Secret has cert fields — check whether the cert is still valid and not near expiry.
		valid, err := r.CertManager.ValidateExpiry(ctx, namespace, tls.SecretName, renewalThreshold)
		if err != nil {
			logger.Error(err, "Failed to parse existing certificate, will regenerate",
				"secret", tls.SecretName)
			// Treat unparseable cert as needing regeneration.
		} else if valid {
			logger.V(1).Info("TLS certificate is current, no action needed")
			return r.setCondition(ctx, agent, metav1.ConditionTrue,
				"CertificateCurrent",
				"TLS certificate exists and is not near expiry",
			)
		} else {
			logger.Info("TLS certificate is expired or within renewal threshold, regenerating",
				"secret", tls.SecretName, "renewalThreshold", renewalThreshold)
		}
	}

	// Generate a new certificate.
	logger.Info("Generating new TLS certificate",
		"serviceName", tls.ServiceName,
		"namespace", namespace,
		"algorithm", tls.Algorithm,
	)
	certData, err := r.CertManager.Generate(ctx, tls.ServiceName, namespace, tls.KeySize, validityDuration, tls.Algorithm)
	if err != nil {
		return r.setCondition(ctx, agent, metav1.ConditionFalse,
			"GenerationFailed",
			fmt.Sprintf("Failed to generate TLS certificate: %v", err),
		)
	}

	// Install the certificate into the Secret and patch the webhook config.
	if err := r.CertManager.UpdateResources(ctx, namespace, tls.SecretName, tls.WebhookName, certData); err != nil {
		return r.setCondition(ctx, agent, metav1.ConditionFalse,
			"InstallFailed",
			fmt.Sprintf("Failed to install TLS certificate: %v", err),
		)
	}

	logger.Info("TLS certificate generated and installed successfully",
		"secret", tls.SecretName, "webhook", tls.WebhookName)
	return r.setCondition(ctx, agent, metav1.ConditionTrue,
		"CertificateInstalled",
		"TLS certificate generated and installed successfully",
	)
}

// setCondition updates the CertificateValid condition on the CloudZeroAgent status
// and requeues after the default interval.
func (r *CloudZeroAgentReconciler) setCondition(
	ctx context.Context,
	agent *agentv1alpha1.CloudZeroAgent,
	status metav1.ConditionStatus,
	reason, message string,
) (ctrl.Result, error) {
	apimeta.SetStatusCondition(&agent.Status.Conditions, metav1.Condition{
		Type:               ConditionCertificateValid,
		Status:             status,
		ObservedGeneration: agent.Generation,
		Reason:             reason,
		Message:            message,
	})

	if err := r.Status().Update(ctx, agent); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update CloudZeroAgent status: %w", err)
	}

	return ctrl.Result{RequeueAfter: defaultRequeueInterval}, nil
}

// SetupWithManager registers the reconciler with the controller manager.
func (r *CloudZeroAgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&agentv1alpha1.CloudZeroAgent{}).
		Named("cloudzeroagent").
		Complete(r)
}
