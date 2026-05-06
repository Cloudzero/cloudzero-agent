// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"testing"

	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	agentv1alpha1 "github.com/cloudzero/cloudzero-agent/operator/api/v1alpha1"
	certmocks "github.com/cloudzero/cloudzero-agent/operator/internal/certmanager/mocks"
	"github.com/cloudzero/cloudzero-agent/operator/internal/metricsapi"
	metricsmocks "github.com/cloudzero/cloudzero-agent/operator/internal/metricsapi/mocks"
)

// newTestReconcilerWithMetrics creates a reconciler with both CertManager and MetricsClient mocks.
func newTestReconcilerWithMetrics(
	t *testing.T,
	agent *agentv1alpha1.CloudZeroAgent,
	cm *certmocks.MockCertManager,
	mc *metricsmocks.MockMetricsClient,
	extraObjects ...runtime.Object,
) (*CloudZeroAgentReconciler, *record.FakeRecorder) {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("add client-go scheme: %v", err)
	}
	if err := agentv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add agentv1alpha1 scheme: %v", err)
	}
	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add appsv1 scheme: %v", err)
	}

	builder := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).WithStatusSubresource(agent)
	for _, obj := range extraObjects {
		builder = builder.WithRuntimeObjects(obj)
	}
	fakeClient := builder.Build()

	fakeRecorder := record.NewFakeRecorder(16)

	return &CloudZeroAgentReconciler{
		Client:        fakeClient,
		Scheme:        scheme,
		CertManager:   cm,
		MetricsClient: mc,
		EventRecorder: fakeRecorder,
	}, fakeRecorder
}

// agentWithResourceManagement returns a CloudZeroAgent with the given ResourceManagementSpec.
func agentWithResourceManagement(ns, name string, rm *agentv1alpha1.ResourceManagementSpec) *agentv1alpha1.CloudZeroAgent {
	a := defaultAgent(ns, name)
	a.Spec.ResourceManagement = rm
	return a
}

// observeSpec returns a basic Observe-mode ResourceManagementSpec.
func observeSpec(threshold int) *agentv1alpha1.ResourceManagementSpec {
	return &agentv1alpha1.ResourceManagementSpec{
		Mode:                     agentv1alpha1.ResourceManagementModeObserve,
		PressureThresholdPercent: threshold,
	}
}

// ksmMetrics is a helper to create ComponentMetrics for the KSM component.
func ksmMetrics(usageBytes, limitBytes int64) *metricsapi.ComponentMetrics {
	return &metricsapi.ComponentMetrics{UsageBytes: usageBytes, LimitBytes: limitBytes}
}

// noMetricsForOtherComponents sets up AnyTimes expectations for all components except KSM,
// returning a zero-usage metric so they don't trigger pressure.
func noMetricsForOtherComponents(mc *metricsmocks.MockMetricsClient, namespace string) {
	for compName, cfg := range componentConfigs {
		if compName == agentv1alpha1.ComponentKubeStateMetrics {
			continue
		}
		mc.EXPECT().
			GetComponentMemory(gomock.Any(), namespace, cfg.LabelSelector, cfg.ContainerName).
			Return(&metricsapi.ComponentMetrics{UsageBytes: 100 * 1024 * 1024, LimitBytes: 1024 * 1024 * 1024}, nil).
			AnyTimes()
	}
}

// getMemoryPressureCondition returns the MemoryPressure condition from the agent, or nil.
func getMemoryPressureCondition(t *testing.T, r *CloudZeroAgentReconciler, namespace, name string) *metav1.Condition {
	t.Helper()
	var updated agentv1alpha1.CloudZeroAgent
	if err := r.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, &updated); err != nil {
		t.Fatalf("failed to get CloudZeroAgent: %v", err)
	}
	for _, c := range updated.Status.Conditions {
		if c.Type == ConditionMemoryPressure {
			return &c
		}
	}
	return nil
}

// getComponentStatuses returns ComponentMemory from the status.
func getComponentStatuses(t *testing.T, r *CloudZeroAgentReconciler, namespace, name string) []agentv1alpha1.ComponentMemoryStatus {
	t.Helper()
	var updated agentv1alpha1.CloudZeroAgent
	if err := r.Get(context.Background(), types.NamespacedName{Namespace: namespace, Name: name}, &updated); err != nil {
		t.Fatalf("failed to get CloudZeroAgent: %v", err)
	}
	return updated.Status.ComponentMemory
}

// ─── Tests ───────────────────────────────────────────────────────────────────

func TestResourceManagement_NilSpec_NoMemoryPressureCondition(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agent := defaultAgent("test-ns", "test-agent")
	// ResourceManagement is nil — no metric calls expected.
	mockCM := certmocks.NewMockCertManager(ctrl)
	mockMC := metricsmocks.NewMockMetricsClient(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	mockCM.EXPECT().ValidateExpiry(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)

	r, _ := newTestReconcilerWithMetrics(t, agent, mockCM, mockMC)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cond := getMemoryPressureCondition(t, r, "test-ns", "test-agent")
	if cond != nil {
		t.Errorf("expected no MemoryPressure condition when ResourceManagement is nil, got %+v", cond)
	}
}

func TestResourceManagement_Observe_BelowThreshold_ConditionFalse(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agent := agentWithResourceManagement("test-ns", "test-agent", observeSpec(85))
	mockCM := certmocks.NewMockCertManager(ctrl)
	mockMC := metricsmocks.NewMockMetricsClient(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	mockCM.EXPECT().ValidateExpiry(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)

	// All components at 50% usage — below 85% threshold.
	for _, cfg := range componentConfigs {
		mockMC.EXPECT().
			GetComponentMemory(gomock.Any(), "test-ns", cfg.LabelSelector, cfg.ContainerName).
			Return(&metricsapi.ComponentMetrics{UsageBytes: 512 * 1024 * 1024, LimitBytes: 1024 * 1024 * 1024}, nil)
	}

	r, _ := newTestReconcilerWithMetrics(t, agent, mockCM, mockMC)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cond := getMemoryPressureCondition(t, r, "test-ns", "test-agent")
	if cond == nil {
		t.Fatal("expected MemoryPressure condition to be set")
	}
	if cond.Status != metav1.ConditionFalse {
		t.Errorf("expected MemoryPressure=False, got %v", cond.Status)
	}
	if cond.Reason != "NoMemoryPressure" {
		t.Errorf("expected reason NoMemoryPressure, got %v", cond.Reason)
	}
}

func TestResourceManagement_Observe_AboveThreshold_ConditionTrue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agent := agentWithResourceManagement("test-ns", "test-agent", observeSpec(85))
	mockCM := certmocks.NewMockCertManager(ctrl)
	mockMC := metricsmocks.NewMockMetricsClient(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	mockCM.EXPECT().ValidateExpiry(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)

	// KSM at 92% — above threshold.
	ksmCfg := componentConfigs[agentv1alpha1.ComponentKubeStateMetrics]
	mockMC.EXPECT().
		GetComponentMemory(gomock.Any(), "test-ns", ksmCfg.LabelSelector, ksmCfg.ContainerName).
		Return(ksmMetrics(471*1024*1024, 512*1024*1024), nil) // ~92%

	noMetricsForOtherComponents(mockMC, "test-ns")

	r, _ := newTestReconcilerWithMetrics(t, agent, mockCM, mockMC)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cond := getMemoryPressureCondition(t, r, "test-ns", "test-agent")
	if cond == nil {
		t.Fatal("expected MemoryPressure condition to be set")
	}
	if cond.Status != metav1.ConditionTrue {
		t.Errorf("expected MemoryPressure=True, got %v", cond.Status)
	}
	if cond.Reason != "ComponentMemoryPressure" {
		t.Errorf("expected reason ComponentMemoryPressure, got %v", cond.Reason)
	}
}

func TestResourceManagement_Observe_ComponentStatusPopulated(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agent := agentWithResourceManagement("test-ns", "test-agent", observeSpec(85))
	mockCM := certmocks.NewMockCertManager(ctrl)
	mockMC := metricsmocks.NewMockMetricsClient(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	mockCM.EXPECT().ValidateExpiry(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)

	for _, cfg := range componentConfigs {
		mockMC.EXPECT().
			GetComponentMemory(gomock.Any(), "test-ns", cfg.LabelSelector, cfg.ContainerName).
			Return(&metricsapi.ComponentMetrics{UsageBytes: 256 * 1024 * 1024, LimitBytes: 512 * 1024 * 1024}, nil)
	}

	r, _ := newTestReconcilerWithMetrics(t, agent, mockCM, mockMC)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	statuses := getComponentStatuses(t, r, "test-ns", "test-agent")
	if len(statuses) != len(componentConfigs) {
		t.Errorf("expected %d component statuses, got %d", len(componentConfigs), len(statuses))
	}
	for _, s := range statuses {
		if s.UsagePercent != 50 {
			t.Errorf("component %s: expected UsagePercent=50, got %d", s.ComponentName, s.UsagePercent)
		}
		if s.LimitBytes != 512*1024*1024 {
			t.Errorf("component %s: unexpected LimitBytes=%d", s.ComponentName, s.LimitBytes)
		}
	}
}

func TestResourceManagement_Recommend_AboveThreshold_EmitsEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agent := agentWithResourceManagement("test-ns", "test-agent", &agentv1alpha1.ResourceManagementSpec{
		Mode:                     agentv1alpha1.ResourceManagementModeRecommend,
		PressureThresholdPercent: 85,
	})
	mockCM := certmocks.NewMockCertManager(ctrl)
	mockMC := metricsmocks.NewMockMetricsClient(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	mockCM.EXPECT().ValidateExpiry(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)

	ksmCfg := componentConfigs[agentv1alpha1.ComponentKubeStateMetrics]
	mockMC.EXPECT().
		GetComponentMemory(gomock.Any(), "test-ns", ksmCfg.LabelSelector, ksmCfg.ContainerName).
		Return(ksmMetrics(471*1024*1024, 512*1024*1024), nil) // ~92%

	noMetricsForOtherComponents(mockMC, "test-ns")

	r, recorder := newTestReconcilerWithMetrics(t, agent, mockCM, mockMC)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case event := <-recorder.Events:
		if event == "" {
			t.Error("expected a non-empty event")
		}
	default:
		t.Error("expected a Kubernetes event to be emitted for memory pressure, got none")
	}
}

func TestResourceManagement_Recommend_BelowThreshold_NoEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agent := agentWithResourceManagement("test-ns", "test-agent", &agentv1alpha1.ResourceManagementSpec{
		Mode:                     agentv1alpha1.ResourceManagementModeRecommend,
		PressureThresholdPercent: 85,
	})
	mockCM := certmocks.NewMockCertManager(ctrl)
	mockMC := metricsmocks.NewMockMetricsClient(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	mockCM.EXPECT().ValidateExpiry(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)

	for _, cfg := range componentConfigs {
		mockMC.EXPECT().
			GetComponentMemory(gomock.Any(), "test-ns", cfg.LabelSelector, cfg.ContainerName).
			Return(&metricsapi.ComponentMetrics{UsageBytes: 100 * 1024 * 1024, LimitBytes: 512 * 1024 * 1024}, nil) // ~20%
	}

	r, recorder := newTestReconcilerWithMetrics(t, agent, mockCM, mockMC)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case event := <-recorder.Events:
		t.Errorf("expected no event for below-threshold usage, got: %s", event)
	default:
		// Good — no event.
	}
}

func TestResourceManagement_AutoRemediate_AboveThreshold_PatchesDeployment(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initialLimit := resource.MustParse("512Mi")
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ksm",
			Namespace: "test-ns",
			Labels:    map[string]string{"app.kubernetes.io/name": "ksm"},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "ksm"},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "ksm",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: initialLimit,
								},
							},
						},
					},
				},
			},
		},
	}

	agent := agentWithResourceManagement("test-ns", "test-agent", &agentv1alpha1.ResourceManagementSpec{
		Mode:                     agentv1alpha1.ResourceManagementModeAutoRemediate,
		PressureThresholdPercent: 85,
		ScaleUpStepPercent:       25,
		MaxMemoryMultiplier:      4,
	})
	mockCM := certmocks.NewMockCertManager(ctrl)
	mockMC := metricsmocks.NewMockMetricsClient(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	mockCM.EXPECT().ValidateExpiry(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)

	ksmCfg := componentConfigs[agentv1alpha1.ComponentKubeStateMetrics]
	mockMC.EXPECT().
		GetComponentMemory(gomock.Any(), "test-ns", ksmCfg.LabelSelector, ksmCfg.ContainerName).
		Return(ksmMetrics(471*1024*1024, initialLimit.Value()), nil) // ~92%

	noMetricsForOtherComponents(mockMC, "test-ns")

	r, recorder := newTestReconcilerWithMetrics(t, agent, mockCM, mockMC, deploy)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the deployment was patched.
	var patchedDeploy appsv1.Deployment
	if err := r.Get(context.Background(), types.NamespacedName{Namespace: "test-ns", Name: "ksm"}, &patchedDeploy); err != nil {
		t.Fatalf("failed to get patched deployment: %v", err)
	}
	newLimit := patchedDeploy.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory]
	expectedNewLimit := int64(float64(initialLimit.Value()) * 1.25)
	if newLimit.Value() != expectedNewLimit {
		t.Errorf("expected new limit %d bytes, got %d bytes", expectedNewLimit, newLimit.Value())
	}

	// Verify a MemoryLimitIncreased event was emitted.
	select {
	case event := <-recorder.Events:
		if event == "" {
			t.Error("expected a non-empty event")
		}
	default:
		t.Error("expected a MemoryLimitIncreased event, got none")
	}
}

func TestResourceManagement_AutoRemediate_AtCeiling_NoPatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Current limit is already at 4× the initial, which is the ceiling.
	currentLimit := resource.MustParse("2Gi") // if initial was 512Mi, 4× = 2Gi
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ksm",
			Namespace: "test-ns",
			Labels:    map[string]string{"app.kubernetes.io/name": "ksm"},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "ksm"},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "ksm",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{corev1.ResourceMemory: currentLimit},
							},
						},
					},
				},
			},
		},
	}

	agent := agentWithResourceManagement("test-ns", "test-agent", &agentv1alpha1.ResourceManagementSpec{
		Mode:                     agentv1alpha1.ResourceManagementModeAutoRemediate,
		PressureThresholdPercent: 85,
		ScaleUpStepPercent:       25,
		MaxMemoryMultiplier:      1, // ceiling = 1× current = current, so no scale up possible
	})
	mockCM := certmocks.NewMockCertManager(ctrl)
	mockMC := metricsmocks.NewMockMetricsClient(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	mockCM.EXPECT().ValidateExpiry(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)

	ksmCfg := componentConfigs[agentv1alpha1.ComponentKubeStateMetrics]
	mockMC.EXPECT().
		GetComponentMemory(gomock.Any(), "test-ns", ksmCfg.LabelSelector, ksmCfg.ContainerName).
		Return(ksmMetrics(int64(float64(currentLimit.Value())*0.92), currentLimit.Value()), nil) // 92%

	noMetricsForOtherComponents(mockMC, "test-ns")

	r, recorder := newTestReconcilerWithMetrics(t, agent, mockCM, mockMC, deploy)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Deployment limit should be unchanged.
	var patchedDeploy appsv1.Deployment
	if err := r.Get(context.Background(), types.NamespacedName{Namespace: "test-ns", Name: "ksm"}, &patchedDeploy); err != nil {
		t.Fatalf("failed to get deployment: %v", err)
	}
	gotLimit := patchedDeploy.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory]
	if gotLimit.Value() != currentLimit.Value() {
		t.Errorf("expected limit unchanged at %d, got %d", currentLimit.Value(), gotLimit.Value())
	}

	select {
	case event := <-recorder.Events:
		t.Errorf("expected no event when at ceiling, got: %s", event)
	default:
		// Good.
	}
}

func TestResourceManagement_AutoRemediate_ExplicitMaxBound_Respected(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Current limit is 512Mi. A 25% step would propose 640Mi.
	// Explicit max is set to 600Mi — operator should clamp to 600Mi.
	initialLimit := resource.MustParse("512Mi")
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ksm",
			Namespace: "test-ns",
			Labels:    map[string]string{"app.kubernetes.io/name": "ksm"},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "ksm"},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "ksm",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{corev1.ResourceMemory: initialLimit},
							},
						},
					},
				},
			},
		},
	}

	agent := agentWithResourceManagement("test-ns", "test-agent", &agentv1alpha1.ResourceManagementSpec{
		Mode:                     agentv1alpha1.ResourceManagementModeAutoRemediate,
		PressureThresholdPercent: 85,
		ScaleUpStepPercent:       25,
		MaxMemoryMultiplier:      4,
		Components: map[agentv1alpha1.ComponentName]agentv1alpha1.ComponentResourceSpec{
			agentv1alpha1.ComponentKubeStateMetrics: {
				Memory: agentv1alpha1.MemoryBounds{Max: "600Mi"},
			},
		},
	})
	mockCM := certmocks.NewMockCertManager(ctrl)
	mockMC := metricsmocks.NewMockMetricsClient(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	mockCM.EXPECT().ValidateExpiry(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)

	ksmCfg := componentConfigs[agentv1alpha1.ComponentKubeStateMetrics]
	mockMC.EXPECT().
		GetComponentMemory(gomock.Any(), "test-ns", ksmCfg.LabelSelector, ksmCfg.ContainerName).
		Return(ksmMetrics(int64(float64(initialLimit.Value())*0.92), initialLimit.Value()), nil) // 92%

	noMetricsForOtherComponents(mockMC, "test-ns")

	r, _ := newTestReconcilerWithMetrics(t, agent, mockCM, mockMC, deploy)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var patchedDeploy appsv1.Deployment
	if err := r.Get(context.Background(), types.NamespacedName{Namespace: "test-ns", Name: "ksm"}, &patchedDeploy); err != nil {
		t.Fatalf("failed to get deployment: %v", err)
	}
	gotLimit := patchedDeploy.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory]
	maxQ := resource.MustParse("600Mi")
	if gotLimit.Value() != maxQ.Value() {
		t.Errorf("expected limit clamped to 600Mi (%d), got %d", maxQ.Value(), gotLimit.Value())
	}
}

func TestResourceManagement_AutoRemediate_WithinCooldown_NoPatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initialLimit := resource.MustParse("512Mi")
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ksm",
			Namespace: "test-ns",
			Labels:    map[string]string{"app.kubernetes.io/name": "ksm"},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "ksm"},
			},
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "ksm",
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{corev1.ResourceMemory: initialLimit},
							},
						},
					},
				},
			},
		},
	}

	// Pre-populate status with a very recent LastPatchedAt.
	recentPatch := metav1.Now()
	agent := agentWithResourceManagement("test-ns", "test-agent", &agentv1alpha1.ResourceManagementSpec{
		Mode:                     agentv1alpha1.ResourceManagementModeAutoRemediate,
		PressureThresholdPercent: 85,
		ScaleUpStepPercent:       25,
		MaxMemoryMultiplier:      4,
		CooldownPeriod:           "10m",
	})
	agent.Status.ComponentMemory = []agentv1alpha1.ComponentMemoryStatus{
		{
			ComponentName: agentv1alpha1.ComponentKubeStateMetrics,
			LimitBytes:    initialLimit.Value(),
			LastPatchedAt: &recentPatch, // just patched — still within cooldown
		},
	}

	mockCM := certmocks.NewMockCertManager(ctrl)
	mockMC := metricsmocks.NewMockMetricsClient(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	mockCM.EXPECT().ValidateExpiry(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)

	ksmCfg := componentConfigs[agentv1alpha1.ComponentKubeStateMetrics]
	mockMC.EXPECT().
		GetComponentMemory(gomock.Any(), "test-ns", ksmCfg.LabelSelector, ksmCfg.ContainerName).
		Return(ksmMetrics(int64(float64(initialLimit.Value())*0.92), initialLimit.Value()), nil)

	noMetricsForOtherComponents(mockMC, "test-ns")

	r, recorder := newTestReconcilerWithMetrics(t, agent, mockCM, mockMC, deploy)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Limit should be unchanged.
	var patchedDeploy appsv1.Deployment
	if err := r.Get(context.Background(), types.NamespacedName{Namespace: "test-ns", Name: "ksm"}, &patchedDeploy); err != nil {
		t.Fatalf("failed to get deployment: %v", err)
	}
	gotLimit := patchedDeploy.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory]
	if gotLimit.Value() != initialLimit.Value() {
		t.Errorf("expected limit unchanged during cooldown, got %d", gotLimit.Value())
	}

	select {
	case event := <-recorder.Events:
		t.Errorf("expected no event during cooldown, got: %s", event)
	default:
		// Good.
	}
}

func TestResourceManagement_MetricsClientError_SkipsComponent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agent := agentWithResourceManagement("test-ns", "test-agent", observeSpec(85))
	mockCM := certmocks.NewMockCertManager(ctrl)
	mockMC := metricsmocks.NewMockMetricsClient(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	mockCM.EXPECT().ValidateExpiry(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)

	ksmCfg := componentConfigs[agentv1alpha1.ComponentKubeStateMetrics]
	mockMC.EXPECT().
		GetComponentMemory(gomock.Any(), "test-ns", ksmCfg.LabelSelector, ksmCfg.ContainerName).
		Return(nil, fmt.Errorf("metrics server unavailable"))

	// Other components succeed.
	for compName, cfg := range componentConfigs {
		if compName == agentv1alpha1.ComponentKubeStateMetrics {
			continue
		}
		mockMC.EXPECT().
			GetComponentMemory(gomock.Any(), "test-ns", cfg.LabelSelector, cfg.ContainerName).
			Return(&metricsapi.ComponentMetrics{UsageBytes: 100 * 1024 * 1024, LimitBytes: 1024 * 1024 * 1024}, nil)
	}

	r, _ := newTestReconcilerWithMetrics(t, agent, mockCM, mockMC)
	_, err := reconcileAgent(t, r, "test-ns", "test-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// KSM status should not be present (was skipped due to error).
	statuses := getComponentStatuses(t, r, "test-ns", "test-agent")
	for _, s := range statuses {
		if s.ComponentName == agentv1alpha1.ComponentKubeStateMetrics {
			t.Errorf("expected KSM status to be absent after metrics error, found: %+v", s)
		}
	}
}

func TestResourceManagement_NilMetricsClient_NoMemoryPressureCondition(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	agent := agentWithResourceManagement("test-ns", "test-agent", observeSpec(85))
	mockCM := certmocks.NewMockCertManager(ctrl)

	mockCM.EXPECT().ValidateExisting(gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)
	mockCM.EXPECT().ValidateExpiry(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(true, nil)

	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = agentv1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).WithStatusSubresource(agent).Build()

	// MetricsClient is nil — resource management should be skipped even if spec is set.
	r := &CloudZeroAgentReconciler{
		Client:        fakeClient,
		Scheme:        scheme,
		CertManager:   mockCM,
		MetricsClient: nil,
	}

	result, err := r.Reconcile(context.Background(), ctrlruntime.Request{
		NamespacedName: types.NamespacedName{Namespace: "test-ns", Name: "test-agent"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RequeueAfter != defaultRequeueInterval {
		t.Errorf("expected requeue after %v, got %v", defaultRequeueInterval, result.RequeueAfter)
	}

	cond := getMemoryPressureCondition(t, r, "test-ns", "test-agent")
	if cond != nil {
		t.Errorf("expected no MemoryPressure condition when MetricsClient is nil, got %+v", cond)
	}
}
