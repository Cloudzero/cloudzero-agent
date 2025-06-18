// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package backfiller provides functionality to backfill Kubernetes Resource objects, and if enabled invokes the webhook domain logic
package backfiller

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"reflect"
	goruntime "runtime"
	"time"

	"github.com/golang/snappy"
	"github.com/google/uuid"
	"github.com/prometheus/prometheus/prompb"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/protoadapt"

	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	diag "github.com/cloudzero/cloudzero-agent/app/domain/diagnostic/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/utils/parallel"
)

const (
	MaxWorkersPerPool     = 10
	MaxConnectionAttempts = 10 // The most amount of sleep time possible across all attempts is 1033 seconds (approximately 17 minutes and 13 seconds).
	ConnectionTimeout     = 5 * time.Second
)

// KubernetesObjectEnumerator defines the interface for enumerating Kubernetes objects.
// It provides methods to start the enumeration process and to disable the waiting mechanism
// for dependent services, which can be useful for testing or specific scenarios.
type KubernetesObjectEnumerator interface {
	// Start begins the enumeration process for Kubernetes objects.
	// It takes a context for managing cancellation and deadlines.
	Start(ctx context.Context) error

	// DisableServiceWait disables the waiting mechanism for dependent services.
	// This can be useful for testing or scenarios where waiting is not required.
	DisableServiceWait()
}

type backfiller struct {
	k8sClient   kubernetes.Interface
	settings    *config.Settings
	controller  webhook.WebhookController
	disableWait bool
}

func NewKubernetesObjectEnumerator(k8sClient kubernetes.Interface, controller webhook.WebhookController, settings *config.Settings) KubernetesObjectEnumerator {
	return &backfiller{
		k8sClient:  k8sClient,
		settings:   settings,
		controller: controller,
	}
}

func (s *backfiller) DisableServiceWait() {
	s.disableWait = true
}

func (s *backfiller) Start(ctx context.Context) error {
	// put behind flag for testing
	if !s.disableWait {
		log.Info().Time("currentTime", time.Now().UTC()).Msg("Waiting for dependent services to become available")

		// Ensure the webhook service is ready before proceeding
		// to avoid missing any events during the enumeration process
		if err := AwaitWebhookService(ctx, s.settings.Server.Namespace, s.settings.Server.Domain, MaxConnectionAttempts, ConnectionTimeout); err != nil {
			log.Error().Err(err).Msg("Failed to await webhook service")
			return errors.New("failed to await webhook service")
		}

		// Wait for the collector service to become available to ensure discovered data can be sent successfully
		if err := AwaitCollectorService(ctx, s.settings.Destination, MaxConnectionAttempts, ConnectionTimeout); err != nil {
			log.Error().Err(err).Msg("Failed to await collector service")
			return errors.New("failed to await collector service")
		}
		log.Info().Time("currentTime", time.Now().UTC()).Msg("Dependent services are available.")
	}

	log.Info().Time("currentTime", time.Now().UTC()).Msg("Initiating backfill of existing Kubernetes resources")

	// write all nodes in the cluster storage
	s.enumerateNodes(ctx)

	var (
		// shorthand clients to make code below simpler to read
		corev1Client            = s.k8sClient.CoreV1()
		appsv1Client            = s.k8sClient.AppsV1()
		appsv1beta1Client       = s.k8sClient.AppsV1beta1()
		appsv1beta2Client       = s.k8sClient.AppsV1beta2()
		batchv1Client           = s.k8sClient.BatchV1()
		batchv1beta1Client      = s.k8sClient.BatchV1beta1()
		networkingv1Client      = s.k8sClient.NetworkingV1() // s.k8sClient.NetworkingV1().IngressClasses()
		networkingv1beta1Client = s.k8sClient.NetworkingV1beta1()
		storagev1Client         = s.k8sClient.StorageV1()
		storagev1betav1Client   = s.k8sClient.StorageV1beta1()
	)

	// NOTE: some types are not supported here - this must be updated as we add more Resource/object support
	catalog := []BackFillJobDescription[metav1.Object]{
		{
			types.GroupApps, types.V1, types.KindDeployment,
			helper.ConvertObject[*appsv1.Deployment],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1Client.Deployments(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1Beta2, types.KindDeployment,
			helper.ConvertObject[*appsv1beta2.Deployment],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1beta2Client.Deployments(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1Beta1, types.KindDeployment,
			helper.ConvertObject[*appsv1beta1.Deployment],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1beta1Client.Deployments(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1, types.KindStatefulSet,
			helper.ConvertObject[*appsv1.StatefulSet],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1Client.StatefulSets(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1Beta2, types.KindStatefulSet,
			helper.ConvertObject[*appsv1beta2.StatefulSet],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1beta2Client.StatefulSets(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1Beta1, types.KindStatefulSet,
			helper.ConvertObject[*appsv1beta1.StatefulSet],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1beta1Client.StatefulSets(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1, types.KindDaemonSet,
			helper.ConvertObject[*appsv1.DaemonSet],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1Client.DaemonSets(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1Beta2, types.KindDaemonSet,
			helper.ConvertObject[*appsv1beta2.DaemonSet],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1beta2Client.DaemonSets(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1, types.KindReplicaSet,
			helper.ConvertObject[*appsv1.ReplicaSet],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1Client.ReplicaSets(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1Beta2, types.KindReplicaSet,
			helper.ConvertObject[*appsv1beta2.ReplicaSet],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1beta2Client.ReplicaSets(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupCore, types.V1, types.KindPod,
			helper.ConvertObject[*corev1.Pod],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return corev1Client.Pods(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupCore, types.V1, types.KindService,
			helper.ConvertObject[*corev1.Service],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return corev1Client.Services(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupCore, types.V1, types.KindPersistentVolume,
			helper.ConvertObject[*corev1.PersistentVolume],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return corev1Client.PersistentVolumes().List(ctx, opts)
			},
		},
		{
			types.GroupCore, types.V1, types.KindPersistentVolumeClaim,
			helper.ConvertObject[*corev1.PersistentVolumeClaim],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return corev1Client.PersistentVolumeClaims(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupBatch, types.V1, types.KindJob,
			helper.ConvertObject[*batchv1.Job],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return batchv1Client.Jobs(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupBatch, types.V1, types.KindCronJob,
			helper.ConvertObject[*batchv1.CronJob],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return batchv1Client.CronJobs(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupBatch, types.V1Beta1, types.KindCronJob,
			helper.ConvertObject[*batchv1beta1.CronJob],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return batchv1beta1Client.CronJobs(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupNet, types.V1, types.KindIngress,
			helper.ConvertObject[*networkingv1.Ingress],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return networkingv1Client.Ingresses(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupNet, types.V1Beta1, types.KindIngress,
			helper.ConvertObject[*networkingv1beta1.Ingress],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return networkingv1beta1Client.Ingresses(namespace).List(ctx, opts)
			},
		},
		// Classes
		// Storage storagev1Client
		{
			types.GroupStorage, types.V1, types.KindStorageClass,
			helper.ConvertObject[*storagev1.StorageClass],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return storagev1Client.StorageClasses().List(ctx, opts)
			},
		},
		{
			types.GroupStorage, types.V1Beta1, types.KindStorageClass,
			helper.ConvertObject[*storagev1beta1.StorageClass],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return storagev1betav1Client.StorageClasses().List(ctx, opts)
			},
		},
		// Network networkingv1Client
		{
			types.GroupNet, types.V1, types.KindIngressClass,
			helper.ConvertObject[*networkingv1.IngressClass],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return networkingv1Client.IngressClasses().List(ctx, opts)
			},
		},
		{
			types.GroupNet, types.V1Beta1, types.KindIngressClass,
			helper.ConvertObject[*networkingv1beta1.IngressClass],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return networkingv1beta1Client.IngressClasses().List(ctx, opts)
			},
		},
		// Gateway gateway

	}

	// Notify use of enabled/disabled objects
	for _, task := range catalog {
		g, v, k := task.g, task.v, task.k
		cfg := s.controller.GetConfigurationAccessor(g, v, k)
		if !((cfg.LabelsEnabled() && cfg.LabelsEnabledForType()) || (cfg.AnnotationsEnabled() && cfg.AnnotationsEnabledForType())) {
			log.Info().
				Str("group", g).Str("version", v).Str("kind", k).
				Bool("labelsEnabled", cfg.LabelsEnabled()).
				Bool("labelsEnabledForType", cfg.LabelsEnabledForType()).
				Bool("annotationsEnabled", cfg.AnnotationsEnabled()).
				Bool("annotationsEnabledForType", cfg.AnnotationsEnabledForType()).
				Msg("scanning disabled")
			continue
		}
		log.Info().
			Str("group", g).Str("version", v).Str("kind", k).
			Bool("labelsEnabled", cfg.LabelsEnabled()).
			Bool("labelsEnabledForType", cfg.LabelsEnabledForType()).
			Bool("annotationsEnabled", cfg.AnnotationsEnabled()).
			Bool("annotationsEnabledForType", cfg.AnnotationsEnabledForType()).
			Msg("scanning enabled")
	}

	// worker pool ensures we don't have unbounded growth
	pool := parallel.New(min(goruntime.NumCPU(), MaxWorkersPerPool))
	defer pool.Close()
	waiter := parallel.NewWaiter()

	// cursor values
	limit := s.settings.K8sClient.PaginationLimit
	var _continue string
	allNamespaces := []corev1.Namespace{}
	for {
		// List all namespaces
		namespaces, err := s.k8sClient.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
			Limit:    s.settings.K8sClient.PaginationLimit,
			Continue: _continue,
		})
		if err != nil {
			log.Err(err).Msg("Error listing namespaces")
			return errors.New("failed to list namespaces")
		}
		allNamespaces = append(allNamespaces, namespaces.Items...)

		// For each namespace, gather all resources
		for _, ns := range namespaces.Items {
			// dispatch job post namespace validation AdmissionReview
			namespace := ns.GetName()
			pool.Run(
				func() error {
					log.Info().Str("namespace", namespace).Str("group", "").Str("version", "v1").Str("kind", "namespace").Msg("discovered")
					if ar, err2 := buildAdmissionReview(ns.GroupVersionKind(), &ns); err2 == nil {
						log.Info().Str("namespace", namespace).Str("group", "").Str("version", "v1").Str("kind", "namespace").Msg("published")
						_, _ = s.controller.Review(context.Background(), ar)
						return nil
					}
					log.Err(err).Str("namespace", namespace).Msg("failed to post namespace review")
					return nil // Don't return error, we are not going to retry
				},
				waiter,
			)

			// For Supported and Enabled GVR types - enumerate those resources and capture the resource metadata (labels/annotation)
			for _, task := range catalog {
				g, v, k := task.g, task.v, task.k
				cfg := s.controller.GetConfigurationAccessor(g, v, k)

				if !((cfg.LabelsEnabled() && cfg.LabelsEnabledForType()) || (cfg.AnnotationsEnabled() && cfg.AnnotationsEnabledForType())) {
					// we notified above - so just skip enumeration task
					continue
				}

				pool.Run(
					func() error {
						var cursor string
						for {
							resources, err := task.List(namespace, metav1.ListOptions{Limit: limit, Continue: cursor})
							if err != nil {
								log.Err(err).
									Str("namespace", namespace).
									Str("group", g).Str("version", v).Str("kind", k).
									Msg("error listing resources")
								break
							}

							items := reflect.ValueOf(resources).Elem().FieldByName("Items")
							count := items.Len()
							log.Info().
								Str("group", g).Str("version", v).Str("kind", k).
								Str("namespace", namespace).
								Int("count", count).
								Msg("discovered")

							for i := range count {
								obj := items.Index(i).Addr().Interface()
								if resource := task.Convert(obj); resource != nil {
									name := resource.GetName()
									if ar, err := buildAdmissionReview(schema.GroupVersionKind{Group: g, Version: v, Kind: k}, resource); err == nil {
										log.Info().Str("group", g).Str("version", v).Str("kind", k).Str("namespace", namespace).Str("name", name).Msg("published")
										_, _ = s.controller.Review(context.Background(), ar) // Post the review
										continue
									}
								}
							}

							if resources.GetContinue() != "" {
								cursor = resources.GetContinue()
								continue
							}

							break
						}
						return nil
					},
					waiter,
				)
			}
		}

		// Escape
		if namespaces.GetContinue() != "" {
			_continue = namespaces.GetContinue()
			continue
		}
		break
	}
	waiter.Wait()
	log.Info().
		Time("currentTime", time.Now().UTC()).
		Int("namespacesCount", len(allNamespaces)).
		Msg("Backfill operation completed")
	return nil
}

func (s *backfiller) enumerateNodes(ctx context.Context) {
	// Check if node labels or annotations are enabled; if not, skip processing nodes
	nodeConfigAccessor := handler.NewNodeConfigAccessor(s.settings)
	if !nodeConfigAccessor.LabelsEnabledForType() && !nodeConfigAccessor.AnnotationsEnabledForType() {
		return
	}

	// Create a worker pool to limit concurrency and avoid unbounded growth
	pool := parallel.New(min(goruntime.NumCPU(), MaxWorkersPerPool))
	defer pool.Close()
	waiter := parallel.NewWaiter()

	client := s.k8sClient.CoreV1()
	var _continue string
	log.Info().Msg("enumerating current cluster nodes")

	for {
		// List nodes in the cluster with pagination
		nodes, err := client.Nodes().List(ctx, metav1.ListOptions{
			Limit:    s.settings.K8sClient.PaginationLimit,
			Continue: _continue,
		})
		if err != nil {
			log.Printf("Error listing nodes: %v", err)
			continue
		}

		// Process each node in the current batch
		for _, o := range nodes.Items {
			pool.Run(
				func() error {
					log.Info().Str("group", "").Str("version", "v1").Str("kind", "node").Str("name", o.GetName()).Msg("published")
					// Create an AdmissionReview for the node and post it to the controller
					if ar, err2 := buildAdmissionReview(o.GroupVersionKind(), &o); err2 == nil {
						_, _ = s.controller.Review(context.Background(), ar)
						return nil
					}
					// Log an error if the review fails, but do not retry
					log.Err(err).Str("name", o.GetName()).Msg("failed to post node review")
					return nil
				},
				waiter,
			)
		}
		// If there are no more nodes to process, exit the loop
		if nodes.Continue == "" {
			break
		}
		_continue = nodes.Continue
	}

	// Wait for all jobs in the current batch to complete
	waiter.Wait()
}

// buildAdmissionReview is a generic function that creates an AdmissionReview object for a given resource.
// It takes a resource of any type T and a GroupVersionKind (GVK) as input parameters.
// The function encodes the resource into raw bytes and constructs an AdmissionReview object.
func buildAdmissionReview[T metav1.Object](gvk schema.GroupVersionKind, o T) (*types.AdmissionReview, error) {
	// Encode the resource object into raw bytes
	raw, err := helper.EncodeToRawBytes(o)
	if err != nil {
		log.Err(err).Str("group", gvk.Group).Str("version", gvk.Version).Str("kind", gvk.Kind).Msg("encode failure")
		return nil, err
	}

	// Extract the name and namespace of the resource
	name := o.GetName()
	namespace := o.GetNamespace()

	// Construct and return the AdmissionReview object
	return &types.AdmissionReview{
		ID:           uuid.New().String(),            // Generate a unique ID for the review
		Name:         name,                           // Resource name
		Namespace:    namespace,                      // Resource namespace
		Version:      types.AdmissionReviewVersionV1, // AdmissionReview version
		Operation:    types.OperationCreate,          // Operation type (e.g., Create)
		NewObjectRaw: raw,                            // Encoded resource object
		RequestGVK: &metav1.GroupVersionKind{ // GroupVersionKind of the resource
			Group:   gvk.Group,
			Version: gvk.Version,
			Kind:    gvk.Kind,
		},
	}, nil
}

// ObjectConverter is a type alias for a function that converts an object of any type to a specific metav1.Object type.
type ObjectConverter[T metav1.Object] func(o any) T

// ListFunc is a type alias for a function that lists resources in a namespace with specific list options.
type ListFunc func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error)

// BackFillJobDescription represents a job description for backfilling resources.
// It includes the group, version, kind of the resource, a converter function, and a list function.
type BackFillJobDescription[T metav1.Object] struct {
	g       string             // Group of the resource
	v       string             // Version of the resource
	k       string             // Kind of the resource
	Convert ObjectConverter[T] // Function to convert an object to the desired type
	List    ListFunc           // Function to list resources of this type
}

// AwaitWebhookService waits for the webhook service to be ready by sending validation requests.
// It retries the validation for a specified number of attempts with exponential backoff and jitter.
func AwaitWebhookService(ctx context.Context, namespace, serviceName string, maxRetries int, timeout time.Duration) error {
	url := diag.ValidateURLPathProtocol + "://" + serviceName + "." + namespace + ".svc.cluster.local" + diag.ValidateURLPath

	for attempt := range maxRetries {
		// Create a context with a timeout for the current attempt
		ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Attempt to send a pod to the validating webhook
		_, err := diag.SendPodToValidatingWebhook(ctxWithTimeout, url)
		if err == nil {
			return nil // Validation succeeded
		}
		// Apply exponential backoff with jitter
		backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
		jitter := time.Duration(rand.Int63n(int64(time.Second))) // #nosec G404
		log.Warn().
			Int("attempt", attempt+1).
			Str("url", url).
			Err(err).
			Msgf("still awaiting webhook API availability, next attempt in %v seconds", (backoff + jitter).Seconds())
		time.Sleep(backoff + jitter)
	}

	return fmt.Errorf("received non-2xx response: after %d retries", maxRetries)
}

// AwaitCollectorService attempts to send a WriteRequest to the specified collector service endpoint
// using an HTTP POST request. It retries the request up to a specified number of times with exponential
// backoff and jitter in case of failures.
func AwaitCollectorService(ctx context.Context, endpoint string, maxRetries int, timeout time.Duration) error {
	data, err := proto.Marshal(protoadapt.MessageV2Of(&prompb.WriteRequest{
		Timeseries: []prompb.TimeSeries{},
	}))
	if err != nil {
		log.Err(err).Msg("error marshaling WriteRequest for collector service")
		return fmt.Errorf("error marshaling WriteRequest: %v", err)
	}

	compressed := snappy.Encode(nil, data)

	var resp *http.Response
	var req *http.Request

	for attempt := range maxRetries {
		timedCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		req, err = http.NewRequestWithContext(timedCtx, "POST", endpoint, bytes.NewBuffer(compressed))
		if err != nil {
			return fmt.Errorf("error creating HTTP request: %v", err)
		}

		req.Header.Set("Content-Type", "application/x-protobuf")
		req.Header.Set("Content-Encoding", "snappy")

		client := &http.Client{}
		resp, err = client.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
		if err == nil && resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			return nil
		}

		errCode := 0
		if resp != nil {
			errCode = resp.StatusCode
		}

		backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
		jitter := time.Duration(rand.Int63n(int64(time.Second))) // #nosec G404
		log.Warn().
			Int("attempt", attempt+1).
			Str("url", endpoint).
			Err(err).
			Int("statusCode", errCode).
			Msgf("still awaiting collector API availability, next attempt in %v seconds", (backoff + jitter).Seconds())
		time.Sleep(backoff + jitter)
	}

	return fmt.Errorf("received non-2xx response: after %d retries", maxRetries)
}
