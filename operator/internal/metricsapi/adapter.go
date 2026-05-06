// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metricsapi

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	versioned "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Adapter implements MetricsClient using the Kubernetes metrics API for usage data
// and the core pod API for limit data.
type Adapter struct {
	metricsClient versioned.Interface
	coreClient    kubernetes.Interface
}

// NewAdapter constructs an Adapter from a metrics clientset and a core kubernetes clientset.
func NewAdapter(metricsClient versioned.Interface, coreClient kubernetes.Interface) *Adapter {
	return &Adapter{metricsClient: metricsClient, coreClient: coreClient}
}

// GetComponentMemory returns aggregated memory usage and limit for pods matching labelSelector.
// If containerName is empty, all containers in matching pods are included.
//
// The Kubernetes aggregated metrics API (metrics.k8s.io) is routed through the API server's
// aggregation proxy. Under certain transport configurations (particularly Go's x/net/http2
// transport in KIND clusters), the underlying HTTP call can hang indefinitely even when a
// context deadline is set — the HTTP/2 SETTINGS goroutine inside x/net/http2 does not
// propagate context cancellation. This wrapper runs the fetch in a separate goroutine and
// returns as soon as either the call completes or the caller's context expires, guaranteeing
// the reconcile loop is never blocked longer than the caller's deadline.
func (a *Adapter) GetComponentMemory(ctx context.Context, namespace string, labelSelector map[string]string, containerName string) (*ComponentMetrics, error) {
	type result struct {
		metrics *ComponentMetrics
		err     error
	}

	// Give the inner HTTP call its own hard deadline independent of the caller's context.
	// This bounds goroutine lifetime: even if the HTTP transport ignores ctx cancellation,
	// the goroutine unblocks when httpCtx times out.
	httpCtx, httpCancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer httpCancel()

	ch := make(chan result, 1) // buffered so the goroutine never blocks on send
	go func() {
		m, err := a.fetchComponentMemory(httpCtx, namespace, labelSelector, containerName)
		select {
		case ch <- result{m, err}:
		default: // caller already timed out; discard result
		}
	}()

	select {
	case r := <-ch:
		return r.metrics, r.err
	case <-ctx.Done():
		return nil, fmt.Errorf("GetComponentMemory cancelled: %w", ctx.Err())
	case <-time.After(28 * time.Second):
		return nil, fmt.Errorf("GetComponentMemory hard timeout: 28s elapsed (ctx still active: %v)", ctx.Err())
	}
}

// fetchComponentMemory performs the actual Kubernetes API calls.
func (a *Adapter) fetchComponentMemory(ctx context.Context, namespace string, labelSelector map[string]string, containerName string) (*ComponentMetrics, error) {
	sel := labels.Set(labelSelector).String()
	listOpts := metav1.ListOptions{LabelSelector: sel}

	podMetricsList, err := a.metricsClient.MetricsV1beta1().PodMetricses(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("listing pod metrics (selector=%q): %w", sel, err)
	}

	podList, err := a.coreClient.CoreV1().Pods(namespace).List(ctx, listOpts)
	if err != nil {
		return nil, fmt.Errorf("listing pods for limits (selector=%q): %w", sel, err)
	}

	if len(podList.Items) == 0 {
		return nil, fmt.Errorf("no pods found matching selector %q in namespace %q", sel, namespace)
	}

	var totalUsage int64
	for i := range podMetricsList.Items {
		for j := range podMetricsList.Items[i].Containers {
			c := &podMetricsList.Items[i].Containers[j]
			if containerName == "" || c.Name == containerName {
				totalUsage += c.Usage.Memory().Value()
			}
		}
	}

	var totalLimit int64
	for i := range podList.Items {
		for j := range podList.Items[i].Spec.Containers {
			c := &podList.Items[i].Spec.Containers[j]
			if containerName == "" || c.Name == containerName {
				if limit, ok := c.Resources.Limits[corev1.ResourceMemory]; ok {
					totalLimit += limit.Value()
				}
			}
		}
	}

	return &ComponentMetrics{UsageBytes: totalUsage, LimitBytes: totalLimit}, nil
}
