// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package backfiller provides functionality to backfill Kubernetes resources and store them in a specified storage.
// This package is designed to gather data from various Kubernetes resources such as namespaces, pods, deployments,
// statefulsets, daemonsets, jobs, cronjobs, and nodes. The gathered data is then formatted and stored using a
// resource store interface. This business logic layer is essential for maintaining an up-to-date inventory of
// Kubernetes resources, which can be used for monitoring, auditing, and analysis purposes.
//
// The Backfiller struct is the main component of this package, which is initialized with a Kubernetes client,
// resource store, and configuration settings. The Start method begins the scraping process, iterating through
// all namespaces and collecting data from the specified resources based on the provided filters.
//
// The package also includes helper functions such as writeResources and writeNodes to handle the listing and
// storing of resources in a paginated manner, ensuring efficient data retrieval and storage.
//
// This package is valuable for organizations that need to keep track of their Kubernetes resources and ensure
// that their inventory is always up-to-date. It provides a robust and flexible solution for scraping and storing
// Kubernetes resource data.
package backfiller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	config "github.com/cloudzero/cloudzero-agent/app/config/insights-controller"
	"github.com/cloudzero/cloudzero-agent/app/http/handler"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

type Backfiller struct {
	k8sClient kubernetes.Interface
	settings  *config.Settings
	store     types.ResourceStore
}

func NewBackfiller(k8sClient kubernetes.Interface, store types.ResourceStore, settings *config.Settings) *Backfiller {
	return &Backfiller{
		k8sClient: k8sClient,
		settings:  settings,
		store:     store,
	}
}

func (s *Backfiller) Start(ctx context.Context) {
	var _continue string
	allNamespaces := []corev1.Namespace{}
	log.Info().
		Time("currentTime", time.Now().UTC()).
		Msg("Starting backfill of existing resources")

	// write all nodes in the cluster storage
	s.writeNodes(ctx)

	for {
		// List all namespaces
		namespaces, err := s.k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
			Limit:    s.settings.K8sClient.PaginationLimit,
			Continue: _continue,
		})
		if err != nil {
			log.Err(err).Msg("Error listing namespaces")
			return
		}
		allNamespaces = append(allNamespaces, namespaces.Items...)

		// For each namespace, gather all resources
		for _, ns := range namespaces.Items {
			log.Info().Str("namespace", ns.Name).Msg("Scraping data from namespace")
			// write namespace record
			nr := handler.FormatNamespaceData(&ns, s.settings)
			if err := s.store.Create(ctx, &nr); err != nil {
				log.Err(err).Msg("failed to write data to storage")
			}

			// write all pods in the namespace storage
			if s.settings.Filters.Labels.Resources.Pods || s.settings.Filters.Annotations.Resources.Pods { //nolint:dupl // code is similar, but not duplicated
				writeResources(ctx, s.store, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.CoreV1().Pods(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) (types.ResourceTags, error) {
					data, ok := obj.(*corev1.Pod)
					if !ok {
						return types.ResourceTags{}, fmt.Errorf("type mismatch: wanted corev1.Pod, got %s", reflect.TypeOf(obj))
					}
					return handler.FormatPodData(data, settings), nil
				}, s.settings)
			}

			// write all deployments in the namespace storage
			if s.settings.Filters.Labels.Resources.Deployments || s.settings.Filters.Annotations.Resources.Deployments { //nolint:dupl // code is similar, but not duplicated
				writeResources(ctx, s.store, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.AppsV1().Deployments(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) (types.ResourceTags, error) {
					data, ok := obj.(*appsv1.Deployment)
					if !ok {
						return types.ResourceTags{}, fmt.Errorf("type mismatch: wanted appsv1.Deployment, got %s", reflect.TypeOf(obj))
					}
					return handler.FormatDeploymentData(data, settings), nil
				}, s.settings)
			}

			// write all statefulsets in the namespace storage
			if s.settings.Filters.Labels.Resources.StatefulSets || s.settings.Filters.Annotations.Resources.StatefulSets { //nolint:dupl // code is similar, but not duplicated
				writeResources(ctx, s.store, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.AppsV1().StatefulSets(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) (types.ResourceTags, error) {
					data, ok := obj.(*appsv1.StatefulSet)
					if !ok {
						return types.ResourceTags{}, fmt.Errorf("type mismatch: wanted appsv1.StatefulSet, got %s", reflect.TypeOf(obj))
					}
					return handler.FormatStatefulsetData(data, settings), nil
				}, s.settings)
			}

			// write all daemonsets in the namespace storage
			if s.settings.Filters.Labels.Resources.DaemonSets || s.settings.Filters.Annotations.Resources.DaemonSets { //nolint:dupl // code is similar, but not duplicated
				writeResources(ctx, s.store, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.AppsV1().DaemonSets(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) (types.ResourceTags, error) {
					data, ok := obj.(*appsv1.DaemonSet)
					if !ok {
						return types.ResourceTags{}, fmt.Errorf("type mismatch: wanted appsv1.DaemonSet, got %s", reflect.TypeOf(obj))
					}
					return handler.FormatDaemonSetData(data, settings), nil
				}, s.settings)
			}

			// write all jobs in the namespace storage
			if s.settings.Filters.Labels.Resources.Jobs || s.settings.Filters.Annotations.Resources.Jobs { //nolint:dupl // code is similar, but not duplicated
				writeResources(ctx, s.store, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.BatchV1().Jobs(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) (types.ResourceTags, error) {
					data, ok := obj.(*batchv1.Job)
					if !ok {
						return types.ResourceTags{}, fmt.Errorf("type mismatch: wanted batchv1.Job, got %s", reflect.TypeOf(obj))
					}
					return handler.FormatJobData(data, settings), nil
				}, s.settings)
			}

			// write all cronjobs in the namespace storage
			if s.settings.Filters.Labels.Resources.CronJobs || s.settings.Filters.Annotations.Resources.CronJobs { //nolint:dupl // code is similar, but not duplicated
				writeResources(ctx, s.store, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.BatchV1().CronJobs(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) (types.ResourceTags, error) {
					data, ok := obj.(*batchv1.CronJob)
					if !ok {
						return types.ResourceTags{}, fmt.Errorf("type mismatch: wanted batchv1.CronJob, got %s", reflect.TypeOf(obj))
					}
					return handler.FormatCronJobData(data, settings), nil
				}, s.settings)
			}

		}
		if namespaces.GetContinue() == "" {
			log.Info().
				Time("currentTime", time.Now().UTC()).
				Int("namespacesCount", len(allNamespaces)).
				Msg("Backfill operation completed")
			break
		}
		_continue = namespaces.GetContinue()
	}
}

func writeResources[T metav1.ListInterface](
	ctx context.Context,
	store types.ResourceStore,
	namespace string,
	listFunc func(string, metav1.ListOptions) (T, error),
	formatFunc func(any, *config.Settings) (types.ResourceTags, error),
	settings *config.Settings,
) {
	var _continue string
	for {
		resources, err := listFunc(namespace, metav1.ListOptions{
			Limit:    settings.K8sClient.PaginationLimit,
			Continue: _continue,
		})
		if err != nil {
			log.Err(err).Str("namespace", namespace).Msg("Error listing resources")
			break
		}

		items := reflect.ValueOf(resources).Elem().FieldByName("Items")
		for i := range items.Len() {
			resource := items.Index(i).Addr().Interface()
			record, err := formatFunc(resource, settings)
			if err != nil {
				log.Err(err).Msg("Failed to format data")
				continue
			}
			if err = store.Create(ctx, &record); err != nil {
				log.Err(err).Msg("Failed to write data to storage")
			}
		}

		if resources.GetContinue() == "" {
			return
		}
		_continue = resources.GetContinue()
	}
}

func (s *Backfiller) writeNodes(ctx context.Context) {
	// if nodes are not enabled, skip the work
	if !s.settings.Filters.Labels.Resources.Nodes && !s.settings.Filters.Annotations.Resources.Nodes {
		return
	}

	log.Info().Msg("Writing nodes to storage")
	var _continue string
	for {
		nodes, err := s.k8sClient.CoreV1().Nodes().List(ctx, metav1.ListOptions{
			Limit:    s.settings.K8sClient.PaginationLimit,
			Continue: _continue,
		})
		if err != nil {
			log.Printf("Error listing nodes: %v", err)
			continue
		}
		for _, node := range nodes.Items {
			record := handler.FormatNodeData(&node, s.settings)
			if err := s.store.Create(ctx, &record); err != nil {
				log.Err(err).Msg("failed to write node data to storage")
			}
		}
		if nodes.Continue == "" {
			break
		}
		_continue = nodes.Continue
	}
}
