// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TLSMode defines how TLS certificates are managed for the webhook.
// +kubebuilder:validation:Enum=managed;cert-manager;user-supplied
type TLSMode string

const (
	// TLSModeManaged means the operator generates and rotates certificates automatically.
	TLSModeManaged TLSMode = "managed"
	// TLSModeCertManager means cert-manager is responsible for certificate lifecycle.
	TLSModeCertManager TLSMode = "cert-manager"
	// TLSModeUserSupplied means certificates are provided and managed externally.
	TLSModeUserSupplied TLSMode = "user-supplied"
)

// TLSSpec defines how TLS certificates are managed for the CloudZero Agent webhook.
type TLSSpec struct {
	// Mode determines how TLS certificates are managed.
	// "managed" means the operator generates and rotates certs automatically.
	// "cert-manager" or "user-supplied" means the operator skips cert management.
	// +kubebuilder:default=managed
	Mode TLSMode `json:"mode"`

	// SecretName is the name of the Kubernetes Secret holding TLS cert data.
	// +kubebuilder:default="cloudzero-agent-tls"
	SecretName string `json:"secretName,omitempty"`

	// WebhookName is the name of the ValidatingWebhookConfiguration to patch with the CA bundle.
	// +kubebuilder:default="cloudzero-agent-webhook"
	WebhookName string `json:"webhookName,omitempty"`

	// ServiceName is the DNS service name used for certificate Subject Alternative Names.
	// +kubebuilder:default="cloudzero-agent-webhook"
	ServiceName string `json:"serviceName,omitempty"`

	// Algorithm is the key generation algorithm: RSA, ECDSA, or Ed25519.
	// +kubebuilder:default=ECDSA
	// +kubebuilder:validation:Enum=RSA;ECDSA;Ed25519
	Algorithm string `json:"algorithm,omitempty"`

	// KeySize is the key size in bits for RSA (min 2048) or ECDSA (256, 384, 521).
	// Ignored for Ed25519.
	// +kubebuilder:default=256
	KeySize int `json:"keySize,omitempty"`

	// ValidityDuration is how long generated certificates are valid (Go duration string, e.g. "8760h").
	// +kubebuilder:default="8760h"
	ValidityDuration string `json:"validityDuration,omitempty"`

	// RenewalThreshold is how far before expiry the operator should renew the certificate
	// (Go duration string, e.g. "720h"). Must be less than ValidityDuration.
	// +kubebuilder:default="720h"
	RenewalThreshold string `json:"renewalThreshold,omitempty"`
}

// CloudZeroAgentSpec defines the desired state of a CloudZeroAgent installation.
type CloudZeroAgentSpec struct {
	// TLS configures webhook TLS certificate management.
	// +optional
	TLS TLSSpec `json:"tls,omitempty"`
}

// CloudZeroAgentStatus defines the observed state of a CloudZeroAgent installation.
type CloudZeroAgentStatus struct {
	// Conditions represent the latest observations of the CloudZeroAgent's state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cza
// +kubebuilder:printcolumn:name="TLS Mode",type=string,JSONPath=`.spec.tls.mode`
// +kubebuilder:printcolumn:name="Certificate Valid",type=string,JSONPath=`.status.conditions[?(@.type=="CertificateValid")].status`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// CloudZeroAgent is the schema for a CloudZero Agent installation.
// The operator watches this resource and reconciles the agent's state on the cluster.
type CloudZeroAgent struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +optional
	Spec CloudZeroAgentSpec `json:"spec,omitempty"`

	// +optional
	Status CloudZeroAgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudZeroAgentList contains a list of CloudZeroAgent resources.
type CloudZeroAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudZeroAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudZeroAgent{}, &CloudZeroAgentList{})
}
