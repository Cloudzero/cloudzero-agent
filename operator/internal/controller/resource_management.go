// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	agentv1alpha1 "github.com/cloudzero/cloudzero-agent/operator/api/v1alpha1"
	"github.com/cloudzero/cloudzero-agent/operator/internal/metricsapi"
)

// componentConfig maps each ComponentName to the label selector and container name
// used to identify its pods. ContainerName="" means aggregate all containers.
type componentConfig struct {
	LabelSelector map[string]string
	ContainerName string
}

// componentConfigs defines the static mapping from component name to its Kubernetes identity.
// Label values match the app.kubernetes.io/name labels set by the Helm chart.
var componentConfigs = map[agentv1alpha1.ComponentName]componentConfig{
	agentv1alpha1.ComponentKubeStateMetrics: {
		LabelSelector: map[string]string{"app.kubernetes.io/name": "ksm"},
		ContainerName: "ksm",
	},
	agentv1alpha1.ComponentCollector: {
		LabelSelector: map[string]string{"app.kubernetes.io/name": "server"},
		ContainerName: "cloudzero-agent-server",
	},
	agentv1alpha1.ComponentAggregator: {
		LabelSelector: map[string]string{"app.kubernetes.io/name": "aggregator"},
		ContainerName: "", // aggregator pod has multiple containers; aggregate all
	},
	agentv1alpha1.ComponentWebhook: {
		LabelSelector: map[string]string{"app.kubernetes.io/name": "webhook-server"},
		ContainerName: "webhook-server",
	},
}

// componentOrder defines the iteration order for deterministic status updates.
var componentOrder = []agentv1alpha1.ComponentName{
	agentv1alpha1.ComponentKubeStateMetrics,
	agentv1alpha1.ComponentCollector,
	agentv1alpha1.ComponentAggregator,
	agentv1alpha1.ComponentWebhook,
}

// reconcileResourceManagement observes pod memory metrics for each agent component and,
// depending on the configured mode, surfaces conditions, emits events, or patches Deployments.
// All status changes are applied to agent in-memory; the caller is responsible for flushing.
func (r *CloudZeroAgentReconciler) reconcileResourceManagement(ctx context.Context, agent *agentv1alpha1.CloudZeroAgent) {
	spec := agent.Spec.ResourceManagement
	if spec == nil {
		return
	}

	logger := log.FromContext(ctx)
	logger.Info("reconcileResourceManagement entered", "namespace", agent.Namespace)

	// Use a bounded context for metrics API calls to avoid hanging indefinitely
	// if the metrics server is slow or unresponsive.
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	threshold := spec.PressureThresholdPercent
	if threshold == 0 {
		threshold = 85
	}

	multiplier := spec.MaxMemoryMultiplier
	if multiplier == 0 {
		multiplier = 4
	}

	cooldown := 10 * time.Minute
	if spec.CooldownPeriod != "" {
		if d, err := time.ParseDuration(spec.CooldownPeriod); err == nil {
			cooldown = d
		}
	}

	now := metav1.Now()
	pressureDetected := false
	var updatedStatuses []agentv1alpha1.ComponentMemoryStatus

	for _, compName := range componentOrder {
		cfg := componentConfigs[compName]

		logger.Info("GetComponentMemory: starting", "component", compName)
		metrics, err := r.MetricsClient.GetComponentMemory(ctx, agent.Namespace, cfg.LabelSelector, cfg.ContainerName)
		logger.Info("GetComponentMemory: returned", "component", compName, "err", err)
		if metrics != nil {
			logger.Info("GetComponentMemory: values", "component", compName, "usageBytes", metrics.UsageBytes, "limitBytes", metrics.LimitBytes)
		}
		if err != nil {
			logger.Error(err, "Failed to get component memory metrics", "component", compName)
			if existing := findComponentStatus(agent.Status.ComponentMemory, compName); existing != nil {
				updatedStatuses = append(updatedStatuses, *existing)
			}
			continue
		}

		if metrics.LimitBytes == 0 {
			logger.V(1).Info("Component has no memory limit set, skipping", "component", compName)
			continue
		}

		usagePercent := int(float64(metrics.UsageBytes) / float64(metrics.LimitBytes) * 100)

		existing := findComponentStatus(agent.Status.ComponentMemory, compName)
		status := agentv1alpha1.ComponentMemoryStatus{
			ComponentName:     compName,
			CurrentUsageBytes: metrics.UsageBytes,
			LimitBytes:        metrics.LimitBytes,
			UsagePercent:      usagePercent,
			LastObservedAt:    now,
		}
		if existing != nil {
			status.LastPatchedAt = existing.LastPatchedAt
		}

		if usagePercent >= threshold {
			pressureDetected = true
			logger.Info("Memory pressure detected",
				"component", compName,
				"usagePercent", usagePercent,
				"threshold", threshold,
				"mode", spec.Mode,
			)

			switch spec.Mode {
			case agentv1alpha1.ResourceManagementModeRecommend:
				if r.EventRecorder != nil {
					r.EventRecorder.Eventf(agent, corev1.EventTypeNormal, "MemoryPressureDetected",
						"Component %s memory usage at %d%% of limit (%s/%s); consider increasing the memory limit",
						compName, usagePercent,
						resource.NewQuantity(metrics.UsageBytes, resource.BinarySI).String(),
						resource.NewQuantity(metrics.LimitBytes, resource.BinarySI).String(),
					)
				}

			case agentv1alpha1.ResourceManagementModeAutoRemediate:
				newLimit, patched, patchErr := r.autoRemediate(ctx, agent, compName, spec, metrics, existing, cooldown, multiplier)
				if patchErr != nil {
					logger.Error(patchErr, "AutoRemediate failed", "component", compName)
				} else if patched {
					patchTime := metav1.Now()
					status.LastPatchedAt = &patchTime
					status.LimitBytes = newLimit
					if r.EventRecorder != nil {
						r.EventRecorder.Eventf(agent, corev1.EventTypeNormal, "MemoryLimitIncreased",
							"Component %s memory limit increased to %s",
							compName, resource.NewQuantity(newLimit, resource.BinarySI).String(),
						)
					}
				}
			}
		}

		updatedStatuses = append(updatedStatuses, status)
	}

	agent.Status.ComponentMemory = updatedStatuses

	if pressureDetected {
		setCondition(agent, ConditionMemoryPressure, metav1.ConditionTrue,
			"ComponentMemoryPressure",
			fmt.Sprintf("One or more agent components are at or above the %d%% memory pressure threshold", threshold),
		)
	} else {
		setCondition(agent, ConditionMemoryPressure, metav1.ConditionFalse,
			"NoMemoryPressure",
			"All agent components are within normal memory usage bounds",
		)
	}
}

// autoRemediate calculates and applies a memory limit increase to the Deployment for compName.
// Returns the new limit in bytes, whether a patch was applied, and any error.
func (r *CloudZeroAgentReconciler) autoRemediate(
	ctx context.Context,
	agent *agentv1alpha1.CloudZeroAgent,
	compName agentv1alpha1.ComponentName,
	spec *agentv1alpha1.ResourceManagementSpec,
	metrics *metricsapi.ComponentMetrics,
	existing *agentv1alpha1.ComponentMemoryStatus,
	cooldown time.Duration,
	multiplier int,
) (newLimit int64, patched bool, err error) {
	// Check cooldown: if we patched recently, skip.
	if existing != nil && existing.LastPatchedAt != nil {
		if time.Since(existing.LastPatchedAt.Time) < cooldown {
			return 0, false, nil
		}
	}

	scaleStep := spec.ScaleUpStepPercent
	if scaleStep == 0 {
		scaleStep = 25
	}

	currentLimit := metrics.LimitBytes
	proposed := int64(float64(currentLimit) * (1 + float64(scaleStep)/100))

	// Determine ceiling: explicit per-component max overrides the multiplier default.
	ceiling := int64(multiplier) * currentLimit
	if compSpec, ok := spec.Components[compName]; ok && compSpec.Memory.Max != "" {
		if maxQ, parseErr := resource.ParseQuantity(compSpec.Memory.Max); parseErr == nil {
			ceiling = maxQ.Value()
		}
	}

	if currentLimit >= ceiling {
		// Already at or above ceiling — do not patch.
		return 0, false, nil
	}

	if proposed > ceiling {
		proposed = ceiling
	}

	// Find the Deployment for this component.
	cfg := componentConfigs[compName]
	var deployList appsv1.DeploymentList
	if listErr := r.List(ctx, &deployList,
		client.InNamespace(agent.Namespace),
		client.MatchingLabels(cfg.LabelSelector),
	); listErr != nil {
		return 0, false, fmt.Errorf("listing deployments for %s: %w", compName, listErr)
	}

	if len(deployList.Items) == 0 {
		return 0, false, fmt.Errorf("no deployment found for component %s", compName)
	}

	deploy := &deployList.Items[0]
	newLimitQ := resource.NewQuantity(proposed, resource.BinarySI)

	patch := client.MergeFrom(deploy.DeepCopy())
	containerPatched := false
	for i := range deploy.Spec.Template.Spec.Containers {
		c := &deploy.Spec.Template.Spec.Containers[i]
		if cfg.ContainerName == "" || c.Name == cfg.ContainerName {
			if c.Resources.Limits == nil {
				c.Resources.Limits = corev1.ResourceList{}
			}
			c.Resources.Limits[corev1.ResourceMemory] = *newLimitQ
			containerPatched = true
		}
	}

	if !containerPatched {
		return 0, false, fmt.Errorf("container %q not found in deployment %s/%s",
			cfg.ContainerName, deploy.Namespace, deploy.Name)
	}

	if patchErr := r.Patch(ctx, deploy, patch); patchErr != nil {
		return 0, false, fmt.Errorf("patching deployment %s/%s: %w", deploy.Namespace, deploy.Name, patchErr)
	}

	return proposed, true, nil
}

// findComponentStatus looks up the existing status entry for a component, or nil if not present.
func findComponentStatus(statuses []agentv1alpha1.ComponentMemoryStatus, name agentv1alpha1.ComponentName) *agentv1alpha1.ComponentMemoryStatus {
	for i := range statuses {
		if statuses[i].ComponentName == name {
			return &statuses[i]
		}
	}
	return nil
}
