// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"reflect"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/handler"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Scraper struct {
	k8sClient kubernetes.Interface
	settings  *config.Settings
	writer    storage.DatabaseWriter
}

func NewScraper(k8sClient kubernetes.Interface, writer storage.DatabaseWriter, settings *config.Settings) *Scraper {
	return &Scraper{
		k8sClient: k8sClient,
		settings:  settings,
		writer:    writer,
	}
}

func (s *Scraper) Start() {
	ctx := context.Background()
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
			if err := s.writer.WriteData(nr, false); err != nil {
				log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
			}

			// write all pods in the namespace storage
			if s.settings.Filters.Labels.Resources.Pods || s.settings.Filters.Annotations.Resources.Pods { // nolint
				writeResources(s.writer, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.CoreV1().Pods(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) storage.ResourceTags {
					return handler.FormatPodData(obj.(*corev1.Pod), settings) // nolint
				}, s.settings)
			}

			// write all deployments in the namespace storage
			if s.settings.Filters.Labels.Resources.Deployments || s.settings.Filters.Annotations.Resources.Deployments { // nolint
				writeResources(s.writer, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.AppsV1().Deployments(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) storage.ResourceTags {
					return handler.FormatDeploymentData(obj.(*appsv1.Deployment), settings) // nolint
				}, s.settings)
			}

			// write all statefulsets in the namespace storage
			if s.settings.Filters.Labels.Resources.StatefulSets || s.settings.Filters.Annotations.Resources.StatefulSets { // nolint
				writeResources(s.writer, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.AppsV1().StatefulSets(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) storage.ResourceTags {
					return handler.FormatStatefulsetData(obj.(*appsv1.StatefulSet), settings) // nolint
				}, s.settings)
			}

			// write all daemonsets in the namespace storage
			if s.settings.Filters.Labels.Resources.DaemonSets || s.settings.Filters.Annotations.Resources.DaemonSets { // nolint
				writeResources(s.writer, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.AppsV1().DaemonSets(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) storage.ResourceTags {
					return handler.FormatDaemonSetData(obj.(*appsv1.DaemonSet), settings) // nolint
				}, s.settings)
			}

			// write all jobs in the namespace storage
			if s.settings.Filters.Labels.Resources.Jobs || s.settings.Filters.Annotations.Resources.Jobs { // nolint
				writeResources(s.writer, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.BatchV1().Jobs(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) storage.ResourceTags {
					return handler.FormatJobData(obj.(*batchv1.Job), settings) // nolint
				}, s.settings)
			}

			// write all cronjobs in the namespace storage
			if s.settings.Filters.Labels.Resources.CronJobs || s.settings.Filters.Annotations.Resources.CronJobs { // nolint
				writeResources(s.writer, ns.Name, func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
					return s.k8sClient.BatchV1().CronJobs(namespace).List(ctx, opts)
				}, func(obj any, settings *config.Settings) storage.ResourceTags {
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

func writeResources[T metav1.ListInterface](writer storage.DatabaseWriter, namespace string,
	listFunc func(string, metav1.ListOptions) (T, error),
	formatFunc func(any, *config.Settings) storage.ResourceTags, settings *config.Settings) {
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
			if err := writer.WriteData(record, false); err != nil {
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
			if err := s.writer.WriteData(record, false); err != nil {
				log.Error().Err(err).Msgf("failed to write node data to storage: %v", err)
			}
		}
		if nodes.Continue == "" {
			break
		}
		_continue = nodes.Continue
	}
}
