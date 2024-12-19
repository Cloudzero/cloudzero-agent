// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
// Package scraper provides functionality to scrape Kubernetes resources and store them in a specified storage.
// This package is designed to gather data from various Kubernetes resources such as namespaces, pods, deployments,
// statefulsets, daemonsets, jobs, cronjobs, and nodes. The gathered data is then formatted and stored using a
// resource store interface. This business logic layer is essential for maintaining an up-to-date inventory of
// Kubernetes resources, which can be used for monitoring, auditing, and analysis purposes.
//
// The Scraper struct is the main component of this package, which is initialized with a Kubernetes client,
// resource store, and configuration settings. The Start method begins the scraping process, iterating through
// all namespaces and collecting data from the specified resources based on the provided filters.
//
// The package also includes helper functions such as writeResources and writeNodes to handle the listing and
// storing of resources in a paginated manner, ensuring efficient data retrieval and storage.
//
// This package is valuable for organizations that need to keep track of their Kubernetes resources and ensure
// that their inventory is always up-to-date. It provides a robust and flexible solution for scraping and storing
// Kubernetes resource data.
package scraper

import (
	"context"
	"reflect"
	"time"

	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/http/handler"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/types"
)

type Scraper struct {
	k8sClient kubernetes.Interface
	settings  *config.Settings
	store     types.ResourceStore
}

func NewScraper(k8sClient kubernetes.Interface, store types.ResourceStore, settings *config.Settings) *Scraper {
	return &Scraper{
		k8sClient: k8sClient,
		settings:  settings,
		store:     store,
	}
}

func (s *Scraper) Start(ctx context.Context) {
	var _continue string
	allNamespaces := []corev1.Namespace{}
	log.Info().Msgf("Starting scrape of existing resources at: %v", time.Now().UTC())

	// write all nodes in the cluster storage
	s.writeNodes(ctx)

	for {
		// List all namespaces
		namespaces, err := s.k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
			Limit:    s.settings.K8sClient.PaginationLimit,
			Continue: _continue,
		})
		if err != nil {
			log.Error().Msgf("Error listing namespaces: %v", err)
			return
		}
		allNamespaces = append(allNamespaces, namespaces.Items...)

		// For each namespace, gather all resources
		for _, ns := range namespaces.Items {
			log.Info().Msgf("Scraping data from namespace: %s", ns.Name)
			// write namespace record
			ns := ns
			nr := handler.FormatNamespaceData(&ns, s.settings)
			if err := s.store.Create(ctx, &nr); err != nil {
				log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
			}

			// write all pods in the namespace storage
			if s.settings.Filters.Labels.Resources.Pods || s.settings.Filters.Annotations.Resources.Pods { // nolint
				writeResources(ctx, s.store, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.CoreV1().Pods(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) types.ResourceTags {
					return handler.FormatPodData(obj.(*corev1.Pod), settings) // nolint
				}, s.settings)
			}

			// write all deployments in the namespace storage
			if s.settings.Filters.Labels.Resources.Deployments || s.settings.Filters.Annotations.Resources.Deployments { // nolint
				writeResources(ctx, s.store, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.AppsV1().Deployments(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) types.ResourceTags {
					return handler.FormatDeploymentData(obj.(*appsv1.Deployment), settings) // nolint
				}, s.settings)
			}

			// write all statefulsets in the namespace storage
			if s.settings.Filters.Labels.Resources.StatefulSets || s.settings.Filters.Annotations.Resources.StatefulSets { // nolint
				writeResources(ctx, s.store, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.AppsV1().StatefulSets(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) types.ResourceTags {
					return handler.FormatStatefulsetData(obj.(*appsv1.StatefulSet), settings) // nolint
				}, s.settings)
			}

			// write all daemonsets in the namespace storage
			if s.settings.Filters.Labels.Resources.DaemonSets || s.settings.Filters.Annotations.Resources.DaemonSets { // nolint
				writeResources(ctx, s.store, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.AppsV1().DaemonSets(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) types.ResourceTags {
					return handler.FormatDaemonSetData(obj.(*appsv1.DaemonSet), settings) // nolint
				}, s.settings)
			}

			// write all jobs in the namespace storage
			if s.settings.Filters.Labels.Resources.Jobs || s.settings.Filters.Annotations.Resources.Jobs { // nolint
				writeResources(ctx, s.store, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.BatchV1().Jobs(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) types.ResourceTags {
					return handler.FormatJobData(obj.(*batchv1.Job), settings) // nolint
				}, s.settings)
			}

			// write all cronjobs in the namespace storage
			if s.settings.Filters.Labels.Resources.CronJobs || s.settings.Filters.Annotations.Resources.CronJobs { // nolint
				writeResources(ctx, s.store, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.BatchV1().CronJobs(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) types.ResourceTags {
					return handler.FormatCronJobData(obj.(*batchv1.CronJob), settings) // nolint
				}, s.settings)
			}

		}
		if namespaces.GetContinue() == "" {
			log.Info().Msgf("Scrape operation completed at: %v, scraped data from %d namespaces", time.Now().UTC(), len(allNamespaces))
			break
		}
		_continue = namespaces.GetContinue()
	}
}

func writeResources[T metav1.ListInterface](ctx context.Context, store types.ResourceStore, namespace string,
	listFunc func(string, metav1.ListOptions) (T, error),
	formatFunc func(any, *config.Settings) types.ResourceTags, settings *config.Settings) {
	var _continue string
	for {
		resources, err := listFunc(namespace, metav1.ListOptions{
			Limit:    settings.K8sClient.PaginationLimit,
			Continue: _continue,
		})

		if err != nil {
			log.Error().Msgf("Error listing resources in namespace %s: %v", namespace, err)
			break
		}

		items := reflect.ValueOf(resources).Elem().FieldByName("Items")
		for i := 0; i < items.Len(); i++ {
			resource := items.Index(i).Addr().Interface()
			record := formatFunc(resource, settings)
			if err := store.Create(ctx, &record); err != nil {
				log.Error().Err(err).Msg("Failed to write data to storage")
			}
		}

		if resources.GetContinue() == "" {
			return
		}
		_continue = resources.GetContinue()
	}
}

func (s *Scraper) writeNodes(ctx context.Context) {
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
			node := node
			record := handler.FormatNodeData(&node, s.settings)
			if err := s.store.Create(ctx, &record); err != nil {
				log.Error().Err(err).Msgf("failed to write node data to storage: %v", err)
			}
		}
		if nodes.Continue == "" {
			break
		}
		_continue = nodes.Continue
	}
}
