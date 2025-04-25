// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package backfiller provides functionality to backfill Kubernetes Resource objects, and if enabled invokes the webhook domain logic
package backfiller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	goruntime "runtime"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/utils/parallel"
)

type Backfiller struct {
	k8sClient  kubernetes.Interface
	settings   *config.Settings
	controller webhook.WebhookController
}

func NewBackfiller(k8sClient kubernetes.Interface, controller webhook.WebhookController, settings *config.Settings) *Backfiller {
	return &Backfiller{
		k8sClient:  k8sClient,
		settings:   settings,
		controller: controller,
	}
}

func (s *Backfiller) Start(ctx context.Context) {
	log.Info().
		Time("currentTime", time.Now().UTC()).
		Msg("Starting backfill of existing resources")

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
		networkingv1Client      = s.k8sClient.NetworkingV1()
		networkingv1beta1Client = s.k8sClient.NetworkingV1beta1()
	)

	// NOTE: some types are not supported here - this must be updated as we add more Resource/object support
	catalog := []BackFillJobDescription[metav1.Object]{
		{
			types.GroupApps, types.V1, types.KindDeployment,
			ConvertObject[*appsv1.Deployment],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1Client.Deployments(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1Beta2, types.KindDeployment,
			ConvertObject[*appsv1beta2.Deployment],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1beta2Client.Deployments(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1Beta1, types.KindDeployment,
			ConvertObject[*appsv1beta1.Deployment],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1beta1Client.Deployments(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1, types.KindStatefulSet,
			ConvertObject[*appsv1.StatefulSet],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1Client.StatefulSets(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1Beta2, types.KindStatefulSet,
			ConvertObject[*appsv1beta2.StatefulSet],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1beta2Client.StatefulSets(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1Beta1, types.KindStatefulSet,
			ConvertObject[*appsv1beta1.StatefulSet],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1beta1Client.StatefulSets(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1, types.KindDaemonSet,
			ConvertObject[*appsv1.DaemonSet],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1Client.DaemonSets(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1Beta2, types.KindDaemonSet,
			ConvertObject[*appsv1beta2.DaemonSet],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1beta2Client.DaemonSets(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1, types.KindReplicaSet,
			ConvertObject[*appsv1.ReplicaSet],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1Client.ReplicaSets(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupApps, types.V1Beta2, types.KindReplicaSet,
			ConvertObject[*appsv1beta2.ReplicaSet],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return appsv1beta2Client.ReplicaSets(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupCore, types.V1, types.KindPod,
			ConvertObject[*corev1.Pod],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return corev1Client.Pods(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupCore, types.V1, types.KindService,
			ConvertObject[*corev1.Service],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return corev1Client.Services(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupCore, types.V1, types.KindPersistentVolume,
			ConvertObject[*corev1.PersistentVolume],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return corev1Client.PersistentVolumes().List(ctx, opts)
			},
		},
		{
			types.GroupCore, types.V1, types.KindPersistentVolumeClaim,
			ConvertObject[*corev1.PersistentVolumeClaim],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return corev1Client.PersistentVolumeClaims(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupBatch, types.V1, types.KindJob,
			ConvertObject[*batchv1.Job],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return batchv1Client.Jobs(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupBatch, types.V1, types.KindCronJob,
			ConvertObject[*batchv1.CronJob],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return batchv1Client.CronJobs(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupBatch, types.V1Beta1, types.KindCronJob,
			ConvertObject[*batchv1beta1.CronJob],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return batchv1beta1Client.CronJobs(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupNet, types.V1, types.KindIngress,
			ConvertObject[*networkingv1.Ingress],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return networkingv1Client.Ingresses(namespace).List(ctx, opts)
			},
		},
		{
			types.GroupNet, types.V1Beta1, types.KindIngress,
			ConvertObject[*networkingv1beta1.Ingress],
			func(namespace string, opts metav1.ListOptions) (metav1.ListInterface, error) {
				return networkingv1beta1Client.Ingresses(namespace).List(ctx, opts)
			},
		},
	}

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
			return
		}
		allNamespaces = append(allNamespaces, namespaces.Items...)

		// worker pool ensures we don't have unbounded growth
		pool := parallel.New(min(goruntime.NumCPU(), 10))
		defer pool.Close()
		waiter := parallel.NewWaiter()

		// For each namespace, gather all resources
		for _, ns := range namespaces.Items {
			// dispatch job post namespace validation AdmissionReview
			namespace := ns.GetName()
			pool.Run(
				func() error {
					log.Info().Str("namespace", namespace).Msg("namespace discovered")
					if ar, err2 := resourceReview(ns.GroupVersionKind(), &ns); err2 == nil {
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

				// If not enabled, skip enumeration task
				if !((cfg.LabelsEnabled() && cfg.LabelsEnabledForType()) && (cfg.AnnotationsEnabled() && cfg.AnnotationsEnabledForType())) {
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
									Str("g", g).Str("v", v).Str("k", k).
									Msg("error listing resources")
								break
							}

							items := reflect.ValueOf(resources).Elem().FieldByName("Items")
							count := items.Len()
							for i := range count {
								obj := items.Index(i).Addr().Interface()
								if resource := task.Convert(obj); resource != nil {
									if ar, err := resourceReview(schema.GroupVersionKind{Group: g, Version: v, Kind: k}, resource); err == nil {
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

		// Don't process the next namespace
		// until we have run all the jobs for that namespace
		waiter.Wait()

		// Escape
		if namespaces.GetContinue() != "" {
			_continue = namespaces.GetContinue()
			continue
		}

		log.Info().
			Time("currentTime", time.Now().UTC()).
			Int("namespacesCount", len(allNamespaces)).
			Msg("Backfill operation completed")
		break
	}
}

func (s *Backfiller) enumerateNodes(ctx context.Context) {
	// Check if node labels or annotations are enabled; if not, skip processing nodes
	nodeConfigAccessor := handler.NewNodeConfigAccessor(s.settings)
	if !nodeConfigAccessor.LabelsEnabledForType() && !nodeConfigAccessor.AnnotationsEnabledForType() {
		return
	}

	// Create a worker pool to limit concurrency and avoid unbounded growth
	pool := parallel.New(min(goruntime.NumCPU(), 10))
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
					// Create an AdmissionReview for the node and post it to the controller
					if ar, err2 := resourceReview(o.GroupVersionKind(), &o); err2 == nil {
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

		// Wait for all jobs in the current batch to complete
		waiter.Wait()

		// If there are no more nodes to process, exit the loop
		if nodes.Continue == "" {
			break
		}
		_continue = nodes.Continue
	}
}

// resourceReview is a generic function that creates an AdmissionReview object for a given resource.
// It takes a resource of any type T and a GroupVersionKind (GVK) as input parameters.
// The function encodes the resource into raw bytes and constructs an AdmissionReview object.
func resourceReview[T metav1.Object](gvk schema.GroupVersionKind, o T) (*types.AdmissionReview, error) {
	// Encode the resource object into raw bytes
	raw, err := encodeToRawBytes(o)
	if err != nil {
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

// registerSchemes registers the API schemes for various Kubernetes resource types.
// It ensures that the runtime.Scheme is aware of the resource types used in the application.
func registerSchemes(scheme *runtime.Scheme) error {
	// List of functions to add resource types to the scheme
	schemeFuncs := []func(*runtime.Scheme) error{
		appsv1.AddToScheme,
		appsv1beta1.AddToScheme,
		appsv1beta2.AddToScheme,
		batchv1.AddToScheme,
		batchv1beta1.AddToScheme,
		corev1.AddToScheme,
		networkingv1.AddToScheme,
		networkingv1beta1.AddToScheme,
	}

	// Register each scheme function
	for _, addToScheme := range schemeFuncs {
		if err := addToScheme(scheme); err != nil {
			return fmt.Errorf("failed to add scheme: %w", err)
		}
	}
	return nil
}

// encodeToRawBytes encodes a Kubernetes resource object into raw bytes.
// It uses the runtime.Scheme and CodecFactory to serialize the object.
func encodeToRawBytes(obj metav1.Object) ([]byte, error) {
	// Ensure the object implements runtime.Object
	runtimeObj, ok := obj.(runtime.Object)
	if !ok {
		return nil, errors.New("failed to convert metav1.Object to runtime.Object")
	}

	// Create a new runtime.Scheme and register the necessary schemes
	scheme := runtime.NewScheme()
	_ = registerSchemes(scheme)

	// Create a CodecFactory and an encoder for the registered schemes
	codecs := serializer.NewCodecFactory(scheme)
	encoder := codecs.LegacyCodec(
		appsv1.SchemeGroupVersion,
		appsv1beta1.SchemeGroupVersion,
		appsv1beta2.SchemeGroupVersion,
		batchv1.SchemeGroupVersion,
		batchv1beta1.SchemeGroupVersion,
		corev1.SchemeGroupVersion,
		networkingv1.SchemeGroupVersion,
		networkingv1beta1.SchemeGroupVersion,
	)

	// Encode the runtime object into raw bytes
	raw, err := runtime.Encode(encoder, runtimeObj)
	if err != nil {
		return nil, fmt.Errorf("failed to encode object: %w", err)
	}
	return raw, nil
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

// ConvertObject attempts to cast an object to the specified metav1.Object type.
// If the cast fails, it logs an error and returns nil.
func ConvertObject[T metav1.Object](o any) metav1.Object {
	if obj, ok := o.(T); ok {
		return obj
	}
	log.Error().Msgf("failed to cast object to %T", new(T))
	return nil
}
