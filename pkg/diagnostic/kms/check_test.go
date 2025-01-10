// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package kms_test

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/diagnostic/kms"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/status"
	"github.com/cloudzero/cloudzero-agent-validator/test"
)

const (
	mockURL = "http://example.com"
)

func makeReport() status.Accessor {
	return status.NewAccessor(&status.ClusterStatus{})
}

// createMockEndpoints creates mock endpoints and adds them to the fake clientset
func createMockEndpoints(clientset *fake.Clientset) {
	clientset.PrependReactor("get", "endpoints", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &corev1.Endpoints{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cz-prom-agent-kube-state-metrics",
				Namespace: "prom-agent",
			},
			Subsets: []corev1.EndpointSubset{
				{
					Addresses: []corev1.EndpointAddress{
						{IP: "192.168.1.1"},
					},
					Ports: []corev1.EndpointPort{
						{Name: "http", Port: 8080},
					},
				},
			},
		}, nil
	})
}

func TestChecker_CheckOK(t *testing.T) {
	cfg := &config.Settings{
		Prometheus: config.Prometheus{
			KubeStateMetricsServiceEndpoint: mockURL,
			KubeMetrics:                     []string{"kube_pod_info", "kube_node_info"},
		},
	}
	clientset := fake.NewSimpleClientset()
	createMockEndpoints(clientset)
	provider := kms.NewProvider(context.Background(), cfg, clientset)

	mock := test.NewHTTPMock()
	mock.Expect(http.MethodGet, "kube_pod_info\nkube_node_info\n", http.StatusOK, nil)
	client := mock.HTTPClient()

	accessor := makeReport()

	err := provider.Check(context.Background(), client, accessor)
	assert.NoError(t, err)

	accessor.ReadFromReport(func(s *status.ClusterStatus) {
		assert.Len(t, s.Checks, 1)
		for _, c := range s.Checks {
			assert.True(t, c.Passing)
		}
	})
}

func TestChecker_CheckRetry(t *testing.T) {
	cfg := &config.Settings{
		Prometheus: config.Prometheus{
			KubeStateMetricsServiceEndpoint: mockURL,
			KubeMetrics:                     []string{"kube_pod_info", "kube_node_info"},
		},
	}
	clientset := fake.NewSimpleClientset()
	createMockEndpoints(clientset)
	provider := kms.NewProvider(context.Background(), cfg, clientset)

	// Update the test sleep interval to accelerate the test
	kms.RetryInterval = 10 * time.Millisecond
	kms.MaxRetry = 3

	mock := test.NewHTTPMock()
	for i := 0; i < kms.MaxRetry-1; i++ {
		mock.Expect(http.MethodGet, "", http.StatusNotFound, nil)
	}
	mock.Expect(http.MethodGet, "kube_pod_info\nkube_node_info\n", http.StatusOK, nil)
	client := mock.HTTPClient()

	accessor := makeReport()

	err := provider.Check(context.Background(), client, accessor)
	assert.NoError(t, err)

	accessor.ReadFromReport(func(s *status.ClusterStatus) {
		assert.Len(t, s.Checks, 1)
		for _, c := range s.Checks {
			assert.True(t, c.Passing)
		}
	})
}

func TestChecker_CheckRetryFailure(t *testing.T) {
	cfg := &config.Settings{
		Prometheus: config.Prometheus{
			KubeStateMetricsServiceEndpoint: mockURL,
			KubeMetrics:                     []string{"kube_pod_info", "kube_node_info"},
		},
	}
	clientset := fake.NewSimpleClientset()
	createMockEndpoints(clientset)
	provider := kms.NewProvider(context.Background(), cfg, clientset)

	// Update the test sleep interval to accelerate the test
	kms.RetryInterval = 10 * time.Millisecond
	kms.MaxRetry = 3

	mock := test.NewHTTPMock()
	for i := 0; i < kms.MaxRetry; i++ {
		mock.Expect(http.MethodGet, "", http.StatusNotFound, nil)
	}
	client := mock.HTTPClient()

	accessor := makeReport()

	err := provider.Check(context.Background(), client, accessor)
	assert.NoError(t, err)

	accessor.ReadFromReport(func(s *status.ClusterStatus) {
		assert.Len(t, s.Checks, 1)
		for _, c := range s.Checks {
			assert.False(t, c.Passing)
		}
	})
}

func TestChecker_CheckMetricsValidation(t *testing.T) {
	cfg := &config.Settings{
		Prometheus: config.Prometheus{
			KubeStateMetricsServiceEndpoint: mockURL,
			KubeMetrics:                     []string{"kube_pod_info", "kube_node_info"},
		},
	}
	clientset := fake.NewSimpleClientset()
	createMockEndpoints(clientset)
	provider := kms.NewProvider(context.Background(), cfg, clientset)

	mock := test.NewHTTPMock()
	mock.Expect(http.MethodGet, "kube_pod_info\nkube_node_info\n", http.StatusOK, nil)
	client := mock.HTTPClient()

	accessor := makeReport()

	err := provider.Check(context.Background(), client, accessor)
	assert.NoError(t, err)

	accessor.ReadFromReport(func(s *status.ClusterStatus) {
		assert.Len(t, s.Checks, 1)
		for _, c := range s.Checks {
			assert.True(t, c.Passing)
		}
	})
}

func TestChecker_CheckHandles500Error(t *testing.T) {
	cfg := &config.Settings{
		Prometheus: config.Prometheus{
			KubeStateMetricsServiceEndpoint: mockURL,
			KubeMetrics:                     []string{"kube_pod_info", "kube_node_info"},
		},
	}
	clientset := fake.NewSimpleClientset()
	createMockEndpoints(clientset)
	provider := kms.NewProvider(context.Background(), cfg, clientset)

	mock := test.NewHTTPMock()
	mock.Expect(http.MethodGet, "", http.StatusInternalServerError, nil)
	client := mock.HTTPClient()

	accessor := makeReport()

	err := provider.Check(context.Background(), client, accessor)
	assert.NoError(t, err)

	accessor.ReadFromReport(func(s *status.ClusterStatus) {
		assert.Len(t, s.Checks, 1)
		for _, c := range s.Checks {
			assert.False(t, c.Passing)
		}
	})
}

func TestChecker_CheckMissingMetrics(t *testing.T) {
	cfg := &config.Settings{
		Prometheus: config.Prometheus{
			KubeStateMetricsServiceEndpoint: mockURL,
			KubeMetrics:                     []string{"kube_pod_info", "kube_node_info", "missing_metric"},
		},
	}
	clientset := fake.NewSimpleClientset()
	createMockEndpoints(clientset)
	provider := kms.NewProvider(context.Background(), cfg, clientset)

	mock := test.NewHTTPMock()
	mock.Expect(http.MethodGet, "kube_pod_info\nkube_node_info\n", http.StatusOK, nil)
	client := mock.HTTPClient()

	accessor := makeReport()

	err := provider.Check(context.Background(), client, accessor)
	assert.NoError(t, err)

	accessor.ReadFromReport(func(s *status.ClusterStatus) {
		assert.Len(t, s.Checks, 2)
		foundMissingMetricError := false
		foundRetryError := false
		for _, c := range s.Checks {
			t.Logf("Check: %+v", c)
			assert.False(t, c.Passing)
			if strings.Contains(c.Error, "Required metric missing_metric not found") {
				foundMissingMetricError = true
			}
			if strings.Contains(c.Error, "Failed to fetch metrics after 3 retries") {
				foundRetryError = true
			}
		}
		assert.True(t, foundMissingMetricError, "Expected error for missing metric not found")
		assert.True(t, foundRetryError, "Expected error for failed retries not found")
	})
}
