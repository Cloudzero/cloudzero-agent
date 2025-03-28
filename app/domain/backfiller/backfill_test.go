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
	v1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/util/homedir"

	config "github.com/cloudzero/cloudzero-insights-controller/app/config/insights-controller"
	"github.com/cloudzero/cloudzero-insights-controller/app/domain/backfiller"
	"github.com/cloudzero/cloudzero-insights-controller/app/domain/k8s"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/cloudzero/cloudzero-insights-controller/app/types/mocks"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage/repo"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/utils"
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
				&apiv1.NodeList{Items: []apiv1.Node{{
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
				}}},
			},
			expectations: func(store *mocks.MockResourceStore) {
				store.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, r *types.ResourceTags) error {
					if r.Type != config.Node || r.Name != "demo" || r.Namespace != nil {
						t.Errorf("Unexpected resource tags: %+v", r)
					}
					return nil
				}).AnyTimes()
			},
		},
		{
			name: "Namespace",
			setupObjects: []runtime.Object{
				&apiv1.NamespaceList{Items: []apiv1.Namespace{{
					ObjectMeta: metav1.ObjectMeta{
						Name: "namespace",
					},
				}}},
			},
			expectations: func(store *mocks.MockResourceStore) {
				store.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, r *types.ResourceTags) error {
					if r.Type != config.Namespace || r.Name != "namespace" || r.Namespace != nil {
						t.Errorf("Unexpected resource tags: %+v", r)
					}
					return nil
				}).AnyTimes()
			},
		},
		{
			name: "Pod",
			setupObjects: []runtime.Object{
				&apiv1.NamespaceList{Items: []apiv1.Namespace{{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
				}}},
				&apiv1.PodList{Items: []apiv1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "pod",
							Namespace: "default",
						},
					},
				}},
			},
			expectations: func(store *mocks.MockResourceStore) {
				store.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, r *types.ResourceTags) error {
					if (r.Type == config.Namespace && r.Name == "default" && r.Namespace == nil) ||
						(r.Type == config.Pod && r.Name == "pod" && r.Namespace != nil && *r.Namespace == "default") {
						return nil
					}
					t.Errorf("Unexpected resource tags: %+v", r)
					return nil
				}).AnyTimes()
			},
		},
		{
			name: "Deployments",
			setupObjects: []runtime.Object{
				&apiv1.NamespaceList{Items: []apiv1.Namespace{{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
				}}},
				&v1.DeploymentList{Items: []v1.Deployment{{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "deployment",
						Namespace: "default",
					},
				}}},
			},
			expectations: func(store *mocks.MockResourceStore) {
				store.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, r *types.ResourceTags) error {
					if (r.Type == config.Namespace && r.Name == "default" && r.Namespace == nil) ||
						(r.Type == config.Deployment && r.Name == "deployment" && r.Namespace != nil && *r.Namespace == "default") {
						return nil
					}
					t.Errorf("Unexpected resource tags: %+v", r)
					return nil
				}).AnyTimes()
			},
		},
		{
			name: "StatefulSets",
			setupObjects: []runtime.Object{
				&apiv1.NamespaceList{Items: []apiv1.Namespace{{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
				}}},
				&v1.StatefulSetList{Items: []v1.StatefulSet{{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "statefulset",
						Namespace: "default",
					},
				}}},
			},
			expectations: func(store *mocks.MockResourceStore) {
				store.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, r *types.ResourceTags) error {
					if (r.Type == config.Namespace && r.Name == "default" && r.Namespace == nil) ||
						(r.Type == config.StatefulSet && r.Name == "statefulset" && r.Namespace != nil && *r.Namespace == "default") {
						return nil
					}
					t.Errorf("Unexpected resource tags: %+v", r)
					return nil
				}).AnyTimes()
			},
		},
		{
			name: "DaemonSets",
			setupObjects: []runtime.Object{
				&apiv1.NamespaceList{Items: []apiv1.Namespace{{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
				}}},
				&v1.DaemonSetList{Items: []v1.DaemonSet{{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "daemonset",
						Namespace: "default",
					},
				}}},
			},
			expectations: func(store *mocks.MockResourceStore) {
				store.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, r *types.ResourceTags) error {
					if (r.Type == config.Namespace && r.Name == "default" && r.Namespace == nil) ||
						(r.Type == config.DaemonSet && r.Name == "daemonset" && r.Namespace != nil && *r.Namespace == "default") {
						return nil
					}
					t.Errorf("Unexpected resource tags: %+v", r)
					return nil
				}).AnyTimes()
			},
		},
		{
			name: "Jobs",
			setupObjects: []runtime.Object{
				&apiv1.NamespaceList{Items: []apiv1.Namespace{{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
				}}},
				&batchv1.JobList{Items: []batchv1.Job{{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "job",
						Namespace: "default",
					},
				}}},
			},
			expectations: func(store *mocks.MockResourceStore) {
				store.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, r *types.ResourceTags) error {
					if (r.Type == config.Namespace && r.Name == "default" && r.Namespace == nil) ||
						(r.Type == config.Job && r.Name == "job" && r.Namespace != nil && *r.Namespace == "default") {
						return nil
					}
					t.Errorf("Unexpected resource tags: %+v", r)
					return nil
				}).AnyTimes()
			},
		},
		{
			name: "CronJobs",
			setupObjects: []runtime.Object{
				&apiv1.NamespaceList{Items: []apiv1.Namespace{{
					ObjectMeta: metav1.ObjectMeta{
						Name: "default",
					},
				}}},
				&batchv1.CronJobList{Items: []batchv1.CronJob{{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cronjob",
						Namespace: "default",
					},
				}}},
			},
			expectations: func(store *mocks.MockResourceStore) {
				store.EXPECT().Create(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, r *types.ResourceTags) error {
					if (r.Type == config.Namespace && r.Name == "default" && r.Namespace == nil) ||
						(r.Type == config.CronJob && r.Name == "cronjob" && r.Namespace != nil && *r.Namespace == "default") {
						return nil
					}
					t.Errorf("Unexpected resource tags: %+v", r)
					return nil
				}).AnyTimes()
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			ctlr := gomock.NewController(t)
			defer ctlr.Finish()

			mockStore := mocks.NewMockResourceStore(ctlr)
			tc.expectations(mockStore)

			clientset := fake.NewSimpleClientset(tc.setupObjects...)
			s := backfiller.NewBackfiller(clientset, mockStore, settings)
			s.Start(ctx)
		})
	}

	// Integration Test
	t.Run("with real client; integration test", func(t *testing.T) {
		if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
			t.Skip("Skipping integration test as RUN_INTEGRATION_TESTS is not set to true")
		}

		store, err := repo.NewInMemoryResourceRepository(&utils.Clock{})
		require.NoError(t, err)
		require.NotNil(t, store)

		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
		settings.K8sClient.PaginationLimit = 3

		k8sClient, err := k8s.NewClient(kubeconfig)
		require.NoError(t, err)

		s := backfiller.NewBackfiller(k8sClient, store, settings)
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
