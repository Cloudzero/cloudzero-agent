// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceManagementMode controls how the operator responds to memory pressure.
// +kubebuilder:validation:Enum=Observe;Recommend;AutoRemediate
type ResourceManagementMode string

const (
	// ResourceManagementModeObserve surfaces memory pressure as status conditions only.
	ResourceManagementModeObserve ResourceManagementMode = "Observe"
	// ResourceManagementModeRecommend emits Kubernetes Events with sizing recommendations.
	ResourceManagementModeRecommend ResourceManagementMode = "Recommend"
	// ResourceManagementModeAutoRemediate patches Deployment resource limits automatically.
	ResourceManagementModeAutoRemediate ResourceManagementMode = "AutoRemediate"
)

// ComponentName identifies a CloudZero agent component whose memory the operator can manage.
// +kubebuilder:validation:Enum=kubeStateMetrics;collector;aggregator;webhook
type ComponentName string

const (
	ComponentKubeStateMetrics ComponentName = "kubeStateMetrics"
	ComponentCollector        ComponentName = "collector"
	ComponentAggregator       ComponentName = "aggregator"
	ComponentWebhook          ComponentName = "webhook"
)

// MemoryBounds defines an explicit upper bound the operator may set in AutoRemediate mode.
// If not set for a component, the operator derives the ceiling from the component's current
// limit multiplied by ResourceManagementSpec.MaxMemoryMultiplier.
type MemoryBounds struct {
	// Max is the upper bound for the memory limit (e.g. "2Gi").
	// +kubebuilder:validation:Pattern=`^[0-9]+(\.[0-9]+)?(Ki|Mi|Gi|Ti|Pi|Ei|k|M|G|T|P|E)?$`
	// +optional
	Max string `json:"max,omitempty"`
}

// ComponentResourceSpec allows overriding the AutoRemediate ceiling for a single component.
type ComponentResourceSpec struct {
	// Memory defines the explicit upper bound for AutoRemediate limit patches.
	Memory MemoryBounds `json:"memory,omitempty"`
}

// ResourceManagementSpec configures memory pressure observation and optional remediation.
type ResourceManagementSpec struct {
	// Mode controls operator behavior when memory pressure is detected.
	// +kubebuilder:default=Observe
	Mode ResourceManagementMode `json:"mode"`

	// PressureThresholdPercent is the memory usage percentage (of limit) that triggers action.
	// +kubebuilder:default=85
	// +kubebuilder:validation:Minimum=50
	// +kubebuilder:validation:Maximum=99
	PressureThresholdPercent int `json:"pressureThresholdPercent,omitempty"`

	// ScaleUpStepPercent is how much to increase the limit by on each AutoRemediate patch,
	// expressed as a percentage of the current limit.
	// +kubebuilder:default=25
	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=100
	ScaleUpStepPercent int `json:"scaleUpStepPercent,omitempty"`

	// CooldownPeriod is the minimum time between successive AutoRemediate patches for the same
	// component (Go duration string, e.g. "10m").
	// +kubebuilder:default="10m"
	CooldownPeriod string `json:"cooldownPeriod,omitempty"`

	// MaxMemoryMultiplier is the ceiling multiplier applied to the component's current memory limit
	// when no explicit per-component Max is configured. For example, a value of 4 means the operator
	// will not scale a component beyond 4× its Helm-deployed limit.
	// +kubebuilder:default=4
	// +kubebuilder:validation:Minimum=2
	// +kubebuilder:validation:Maximum=32
	MaxMemoryMultiplier int `json:"maxMemoryMultiplier,omitempty"`

	// Components optionally overrides the AutoRemediate ceiling for specific components.
	// If a component is not listed here, the ceiling is derived from its current limit
	// multiplied by MaxMemoryMultiplier.
	// +optional
	Components map[ComponentName]ComponentResourceSpec `json:"components,omitempty"`
}

// ComponentMemoryStatus tracks observed memory usage for a single agent component.
type ComponentMemoryStatus struct {
	// ComponentName identifies the component.
	ComponentName ComponentName `json:"componentName"`
	// CurrentUsageBytes is the most recently observed memory usage in bytes.
	CurrentUsageBytes int64 `json:"currentUsageBytes"`
	// LimitBytes is the current memory limit in bytes.
	LimitBytes int64 `json:"limitBytes"`
	// UsagePercent is CurrentUsageBytes / LimitBytes * 100.
	UsagePercent int `json:"usagePercent"`
	// LastPatchedAt is the timestamp of the last AutoRemediate patch, if any.
	// +optional
	LastPatchedAt *metav1.Time `json:"lastPatchedAt,omitempty"`
	// LastObservedAt is the timestamp of the last metrics observation.
	LastObservedAt metav1.Time `json:"lastObservedAt"`
}

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

	// ResourceManagement configures memory pressure observation and optional remediation.
	// +optional
	ResourceManagement *ResourceManagementSpec `json:"resourceManagement,omitempty"`
}

// CloudZeroAgentStatus defines the observed state of a CloudZeroAgent installation.
type CloudZeroAgentStatus struct {
	// Conditions represent the latest observations of the CloudZeroAgent's state.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// ComponentMemory tracks observed memory usage for each managed agent component.
	// +optional
	ComponentMemory []ComponentMemoryStatus `json:"componentMemory,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=cza
// +kubebuilder:printcolumn:name="TLS Mode",type=string,JSONPath=`.spec.tls.mode`
// +kubebuilder:printcolumn:name="Certificate Valid",type=string,JSONPath=`.status.conditions[?(@.type=="CertificateValid")].status`
// +kubebuilder:printcolumn:name="Memory Pressure",type=string,JSONPath=`.status.conditions[?(@.type=="MemoryPressure")].status`
// +kubebuilder:printcolumn:name="Resource Mgmt",type=string,JSONPath=`.spec.resourceManagement.mode`
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
