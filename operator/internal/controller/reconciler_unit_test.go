// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/cloudzero/cloudzero-agent/app/domain/certificate"
	agentv1alpha1 "github.com/cloudzero/cloudzero-agent/operator/api/v1alpha1"
	certmocks "github.com/cloudzero/cloudzero-agent/operator/internal/certmanager/mocks"
)

// newTestReconciler creates a CloudZeroAgentReconciler with a fake client
// pre-populated with the given CloudZeroAgent and the provided CertManager mock.
func newTestReconciler(t *testing.T, agent *agentv1alpha1.CloudZeroAgent, cm *certmocks.MockCertManager) *CloudZeroAgentReconciler {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add client-go scheme: %v", err)
	}
	if err := agentv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add agentv1alpha1 scheme: %v", err)
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(agent).
		WithStatusSubresource(agent).
		Build()

	return &CloudZeroAgentReconciler{
		Client:      fakeClient,
		Scheme:      scheme,
		CertManager: cm,
	}
}

// defaultAgent returns a CloudZeroAgent in "managed" mode with sensible defaults for tests.
func defaultAgent(namespace, name string) *agentv1alpha1.CloudZeroAgent {
	return &agentv1alpha1.CloudZeroAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: agentv1alpha1.CloudZeroAgentSpec{
			TLS: agentv1alpha1.TLSSpec{
				Mode:             agentv1alpha1.TLSModeManaged,
				SecretName:       "cloudzero-agent-tls",
				WebhookName:      "cloudzero-agent-webhook",
				ServiceName:      "cloudzero-agent-webhook",
				Algorithm:        "ECDSA",
				KeySize:          256,
				ValidityDuration: "8760h",
				RenewalThreshold: "720h",
			},
		},
	}
}

func reconcileAgent(t *testing.T, r *CloudZeroAgentReconciler, namespace, name string) (ctrlruntime.Result, error) {
	t.Helper()
	return r.Reconcile(context.Background(), ctrlruntime.Request{
		NamespacedName: types.NamespacedName{Namespace: namespace, Name: name},
	})
}

func getCondition(t *testing.T, r *CloudZeroAgentReconciler, namespace, name string) *metav1.Condition {
	t.Helper()
	var updated agentv1alpha1.CloudZeroAgent
	if err := r.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, &updated); err != nil {
		t.Fatalf("failed to get CloudZeroAgent after reconcile: %v", err)
	}
	for _, c := range updated.Status.Conditions {
		if c.Type == ConditionCertificateValid {
			return &c
		}
	}
	return nil
}

func TestReconcile_ManagedMode_NoCertExists_GeneratesAndInstalls(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agent := defaultAgent("test-ns", "test-agent")
	mockCM := certmocks.NewMockCertManager(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), "test-ns", "cloudzero-agent-tls").Return(false, nil)
	mockCM.EXPECT().Generate(gomock.Any(), "cloudzero-agent-webhook", "test-ns", 256, 8760*time.Hour, "ECDSA").
		Return(&certificate.CertificateData{CABundle: "ca", TLSCrt: "crt", TLSKey: "key"}, nil)
	mockCM.EXPECT().UpdateResources(gomock.Any(), "test-ns", "cloudzero-agent-tls", "cloudzero-agent-webhook",
		&certificate.CertificateData{CABundle: "ca", TLSCrt: "crt", TLSKey: "key"}).Return(nil)

	r := newTestReconciler(t, agent, mockCM)
	result, err := reconcileAgent(t, r, "test-ns", "test-agent")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequeueAfter != defaultRequeueInterval {
		t.Errorf("expected RequeueAfter=%v, got %v", defaultRequeueInterval, result.RequeueAfter)
	}

	cond := getCondition(t, r, "test-ns", "test-agent")
	if cond == nil {
		t.Fatal("expected CertificateValid condition to be set")
	}
	if cond.Status != metav1.ConditionTrue {
		t.Errorf("expected condition status True, got %v", cond.Status)
	}
	if cond.Reason != "CertificateInstalled" {
		t.Errorf("expected reason CertificateInstalled, got %v", cond.Reason)
	}
}

func TestReconcile_ManagedMode_CertValidAndCurrent_NoGenerate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agent := defaultAgent("test-ns", "test-agent")
	mockCM := certmocks.NewMockCertManager(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), "test-ns", "cloudzero-agent-tls").Return(true, nil)
	mockCM.EXPECT().ValidateExpiry(gomock.Any(), "test-ns", "cloudzero-agent-tls", 720*time.Hour).Return(true, nil)
	// Generate and UpdateResources must NOT be called.

	r := newTestReconciler(t, agent, mockCM)
	result, err := reconcileAgent(t, r, "test-ns", "test-agent")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequeueAfter != defaultRequeueInterval {
		t.Errorf("expected RequeueAfter=%v, got %v", defaultRequeueInterval, result.RequeueAfter)
	}

	cond := getCondition(t, r, "test-ns", "test-agent")
	if cond == nil {
		t.Fatal("expected CertificateValid condition to be set")
	}
	if cond.Status != metav1.ConditionTrue {
		t.Errorf("expected condition status True, got %v", cond.Status)
	}
	if cond.Reason != "CertificateCurrent" {
		t.Errorf("expected reason CertificateCurrent, got %v", cond.Reason)
	}
}

func TestReconcile_ManagedMode_CertNearExpiry_Regenerates(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agent := defaultAgent("test-ns", "test-agent")
	mockCM := certmocks.NewMockCertManager(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), "test-ns", "cloudzero-agent-tls").Return(true, nil)
	mockCM.EXPECT().ValidateExpiry(gomock.Any(), "test-ns", "cloudzero-agent-tls", 720*time.Hour).Return(false, nil)
	mockCM.EXPECT().Generate(gomock.Any(), "cloudzero-agent-webhook", "test-ns", 256, 8760*time.Hour, "ECDSA").
		Return(&certificate.CertificateData{CABundle: "ca", TLSCrt: "crt", TLSKey: "key"}, nil)
	mockCM.EXPECT().UpdateResources(gomock.Any(), "test-ns", "cloudzero-agent-tls", "cloudzero-agent-webhook", gomock.Any()).Return(nil)

	r := newTestReconciler(t, agent, mockCM)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cond := getCondition(t, r, "test-ns", "test-agent")
	if cond == nil {
		t.Fatal("expected CertificateValid condition to be set")
	}
	if cond.Status != metav1.ConditionTrue {
		t.Errorf("expected condition status True, got %v", cond.Status)
	}
	if cond.Reason != "CertificateInstalled" {
		t.Errorf("expected reason CertificateInstalled, got %v", cond.Reason)
	}
}

func TestReconcile_ManagedMode_GenerateFails_ConditionFalse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agent := defaultAgent("test-ns", "test-agent")
	mockCM := certmocks.NewMockCertManager(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)
	mockCM.EXPECT().Generate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, fmt.Errorf("crypto failure"))

	r := newTestReconciler(t, agent, mockCM)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cond := getCondition(t, r, "test-ns", "test-agent")
	if cond == nil {
		t.Fatal("expected CertificateValid condition to be set")
	}
	if cond.Status != metav1.ConditionFalse {
		t.Errorf("expected condition status False, got %v", cond.Status)
	}
	if cond.Reason != "GenerationFailed" {
		t.Errorf("expected reason GenerationFailed, got %v", cond.Reason)
	}
}

func TestReconcile_ManagedMode_InstallFails_ConditionFalse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agent := defaultAgent("test-ns", "test-agent")
	mockCM := certmocks.NewMockCertManager(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), gomock.Any(), gomock.Any()).Return(false, nil)
	mockCM.EXPECT().Generate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&certificate.CertificateData{CABundle: "ca", TLSCrt: "crt", TLSKey: "key"}, nil)
	mockCM.EXPECT().UpdateResources(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(fmt.Errorf("secret patch failed"))

	r := newTestReconciler(t, agent, mockCM)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cond := getCondition(t, r, "test-ns", "test-agent")
	if cond == nil {
		t.Fatal("expected CertificateValid condition to be set")
	}
	if cond.Status != metav1.ConditionFalse {
		t.Errorf("expected condition status False, got %v", cond.Status)
	}
	if cond.Reason != "InstallFailed" {
		t.Errorf("expected reason InstallFailed, got %v", cond.Reason)
	}
}

func TestReconcile_CertManagerMode_SetsUnknown_NoGenerate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agent := defaultAgent("test-ns", "test-agent")
	agent.Spec.TLS.Mode = agentv1alpha1.TLSModeCertManager
	mockCM := certmocks.NewMockCertManager(ctrl)
	// No expectations — CertManager must not be called at all.

	r := newTestReconciler(t, agent, mockCM)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cond := getCondition(t, r, "test-ns", "test-agent")
	if cond == nil {
		t.Fatal("expected CertificateValid condition to be set")
	}
	if cond.Status != metav1.ConditionUnknown {
		t.Errorf("expected condition status Unknown, got %v", cond.Status)
	}
	if cond.Reason != "ExternallyManaged" {
		t.Errorf("expected reason ExternallyManaged, got %v", cond.Reason)
	}
}

func TestReconcile_UserSuppliedMode_SetsUnknown_NoGenerate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agent := defaultAgent("test-ns", "test-agent")
	agent.Spec.TLS.Mode = agentv1alpha1.TLSModeUserSupplied
	mockCM := certmocks.NewMockCertManager(ctrl)
	// No expectations — CertManager must not be called at all.

	r := newTestReconciler(t, agent, mockCM)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cond := getCondition(t, r, "test-ns", "test-agent")
	if cond == nil {
		t.Fatal("expected CertificateValid condition to be set")
	}
	if cond.Status != metav1.ConditionUnknown {
		t.Errorf("expected condition status Unknown, got %v", cond.Status)
	}
}

func TestReconcile_InvalidRenewalThreshold_ConditionFalse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agent := defaultAgent("test-ns", "test-agent")
	agent.Spec.TLS.RenewalThreshold = "not-a-duration"
	mockCM := certmocks.NewMockCertManager(ctrl)
	// No CertManager calls expected.

	r := newTestReconciler(t, agent, mockCM)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cond := getCondition(t, r, "test-ns", "test-agent")
	if cond == nil {
		t.Fatal("expected CertificateValid condition to be set")
	}
	if cond.Status != metav1.ConditionFalse {
		t.Errorf("expected condition status False, got %v", cond.Status)
	}
	if cond.Reason != "InvalidConfig" {
		t.Errorf("expected reason InvalidConfig, got %v", cond.Reason)
	}
}

func TestReconcile_ResourceNotFound_NoError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create reconciler with no objects in the fake client.
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = agentv1alpha1.AddToScheme(scheme)

	mockCM := certmocks.NewMockCertManager(ctrl)
	// No CertManager calls expected.

	r := &CloudZeroAgentReconciler{
		Client:      fake.NewClientBuilder().WithScheme(scheme).Build(),
		Scheme:      scheme,
		CertManager: mockCM,
	}

	result, err := r.Reconcile(context.Background(), ctrlruntime.Request{
		NamespacedName: types.NamespacedName{Namespace: "test-ns", Name: "does-not-exist"},
	})

	if err != nil {
		t.Fatalf("expected no error for not-found resource, got: %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Errorf("expected no requeue for not-found resource, got RequeueAfter=%v", result.RequeueAfter)
	}
}
