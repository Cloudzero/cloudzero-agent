// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates.
// SPDX-License-Identifier: Apache-2.0

package backfiller_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/homedir"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/backfiller"
	"github.com/cloudzero/cloudzero-agent/app/storage/repo"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
	"github.com/cloudzero/cloudzero-agent/app/utils"
	"github.com/cloudzero/cloudzero-agent/app/utils/k8s"
)

// TestBackfiller_FakeK8s_Start tests the Backfiller.Start method with various Kubernetes resources using a fake client.
func TestBackfiller_FakeK8s_Start(t *testing.T) {
	ctx := context.Background()
	settings := getDefaultSettings()

	testCases := []struct {
		name         string
		setupObjects []runtime.Object
		expectations func(store *mocks.MockResourceStore)
	}{
		{
			name: "Node",
			setupObjects: []runtime.Object{
				&apiv1.NodeList{
					Items: []apiv1.Node{
						{
							TypeMeta: metav1.TypeMeta{
								Kind:       "Node",
								APIVersion: "v1",
							},
							ObjectMeta: metav1.ObjectMeta{
								Name: "demo",
							},
							Status: apiv1.NodeStatus{
								Addresses: []apiv1.NodeAddress{
									{
										Type:    apiv1.NodeInternalIP,
										Address: "10.0.0.1",
									},
								},
							},
						},
					},
				},
			},
			expectations: func(store *mocks.MockResourceStore) {
				store.EXPECT().FindFirstBy(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
				store.EXPECT().Tx(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				store.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, r *types.ResourceTags) error {
					if r.Type != config.Node || r.Name != "demo" || r.Namespace != nil {
						t.Errorf("Unexpected resource tags: %+v", r)
					}
					return nil
				}).Times(1)
			},
		},
		{
			name: "Namespace",
			setupObjects: []runtime.Object{
				&apiv1.NamespaceList{Items: []apiv1.Namespace{{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Namespace",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace",
					},
				}}},
			},
			expectations: func(store *mocks.MockResourceStore) {
				store.EXPECT().FindFirstBy(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
				store.EXPECT().Tx(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				store.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, r *types.ResourceTags) error {
						if r.Type != config.Namespace || r.Name != "namespace" || r.Namespace != nil {
							t.Errorf("Unexpected resource tags: %+v", r)
						}
						return nil
					},
				).Times(1)
			},
		},
		{
			name: "Multiple Pods",
			setupObjects: []runtime.Object{
				&apiv1.NamespaceList{Items: []apiv1.Namespace{{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Namespace",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
				}}},
				&apiv1.PodList{Items: []apiv1.Pod{
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "pod-1",
							Namespace: "default",
						},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "pod-2",
							Namespace: "default",
						},
					},
					{
						TypeMeta: metav1.TypeMeta{
							Kind:       "Pod",
							APIVersion: "v1",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name:      "pod-3",
							Namespace: "default",
						},
					},
				}},
			},
			expectations: func(store *mocks.MockResourceStore) {
				store.EXPECT().FindFirstBy(gomock.Any(), gomock.Any()).Return(nil, nil).Times(4)
				store.EXPECT().Tx(gomock.Any(), gomock.Any()).Return(nil).AnyTimes().Times(4)
				store.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(
					func(ctx context.Context, r *types.ResourceTags) error {
						switch r.Type {
						case config.Namespace:
							if r.Name == "default" && r.Namespace == nil {
								return nil
							}
						case config.Pod:
							if r.Namespace != nil && *r.Namespace == "default" {
								if r.Name == "pod-1" || r.Name == "pod-2" || r.Name == "pod-3" {
									return nil
								}
							}
						}
						t.Errorf("Unexpected resource tags: %+v", r)
						return nil
					},
				).Times(4)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			clock := mocks.NewMockClock(time.Now())
			store := mocks.NewMockResourceStore(mockCtl)
			tc.expectations(store)

			controller, err := webhook.NewWebhookFactory(store, settings, clock)
			assert.NoError(t, err)

			clientset := fake.NewClientset(tc.setupObjects...)
			s := backfiller.NewKubernetesObjectEnumerator(clientset, controller, settings)
			s.DisableServiceWait()
			s.Start(ctx)
		})
	}

	// Integration Test
	t.Run("with real client; integration test", func(t *testing.T) {
		if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
			t.Skip("Skipping integration test as RUN_INTEGRATION_TESTS is not set to true")
		}
		clock := &utils.Clock{}

		store, err := repo.NewInMemoryResourceRepository(clock)
		require.NoError(t, err)
		require.NotNil(t, store)

		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
		settings.K8sClient.PaginationLimit = 3

		k8sClient, err := k8s.NewClient(kubeconfig)
		require.NoError(t, err)

		controller, err := webhook.NewWebhookFactory(store, settings, clock)
		require.NoError(t, err)
		s := backfiller.NewKubernetesObjectEnumerator(k8sClient, controller, settings)
		s.DisableServiceWait()
		s.Start(context.Background())

		// Wait for Backfiller to process resources
		// Consider using synchronization mechanisms instead of sleep in real tests
		time.Sleep(5 * time.Second)

		found, err := store.FindAllBy(context.Background(), "1=1")
		require.NoError(t, err)
		assert.NotEmpty(t, found)
	})
}

// getDefaultSettings returns a default configuration settings for the Backfiller.
func getDefaultSettings() *config.Settings {
	return &config.Settings{
		Filters: config.Filters{
			Labels: config.Labels{
				Enabled: true,
				Resources: config.Resources{
					Pods:         true,
					Namespaces:   true,
					Deployments:  true,
					Jobs:         true,
					CronJobs:     true,
					StatefulSets: true,
					DaemonSets:   true,
					Nodes:        true,
				},
				Patterns: []string{".*"},
			},
			Annotations: config.Annotations{
				Enabled: true,
				Resources: config.Resources{
					Pods:         true,
					Namespaces:   true,
					Deployments:  true,
					Jobs:         true,
					CronJobs:     true,
					StatefulSets: true,
					DaemonSets:   true,
					Nodes:        true,
				},
				Patterns: []string{".*"},
			},
		},
	}
}
