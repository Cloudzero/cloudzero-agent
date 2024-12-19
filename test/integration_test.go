//go:build integration
// +build integration

// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"testing"
	"time"

	netHttp "net/http"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
	"gotest.tools/v3/assert"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/domain/pusher"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/http"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/http/handler"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage/repo"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/types"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/utils"
)

func TestIntegration(t *testing.T) {
	settings := &config.Settings{}

	clock := utils.Clock{}
	store, err := repo.NewInMemoryResourceRepository(&clock)
	if err != nil {
		t.Fatalf("Failed to create storage system: %v", err)
	}

	errChan := make(chan error)

	server := http.NewServer(settings,
		[]http.RouteSegment{},
		[]http.AdmissionRouteSegment{
			{Route: "/validate/pod", Hook: handler.NewPodHandler(store, settings, errChan)},
			{Route: "/validate/deployment", Hook: handler.NewDeploymentHandler(store, settings, errChan)},
			{Route: "/validate/statefulset", Hook: handler.NewStatefulsetHandler(store, settings, errChan)},
			{Route: "/validate/namespace", Hook: handler.NewNamespaceHandler(store, settings, errChan)},
			{Route: "/validate/node", Hook: handler.NewNodeHandler(store, settings, errChan)},
			{Route: "/validate/job", Hook: handler.NewJobHandler(store, settings, errChan)},
			{Route: "/validate/cronjob", Hook: handler.NewCronJobHandler(store, settings, errChan)},
			{Route: "/validate/daemonset", Hook: handler.NewDaemonSetHandler(store, settings, errChan)},
		}...,
	)

	// Create a listener
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	go func() {
		server.Serve(listener)
	}()
	defer server.Close()

	// Wait for the server to start
	time.Sleep(1 * time.Second)

	serverAddr := "http://" + listener.Addr().String()

	// Test cases
	tests := []struct {
		route string
		body  interface{}
	}{
		{route: "/validate/pod", body: CreatePodAdmissionReviewRequest()},
	}

	for _, test := range tests {
		admissionReview := test.body
		payload, err := json.Marshal(admissionReview)
		if err != nil {
			fmt.Println("Failed to marshal JSON:", err)
			os.Exit(1)
		}
		req, err := netHttp.NewRequest("POST", serverAddr+test.route, bytes.NewBuffer(payload))
		assert.NilError(t, err)
		req.Header.Set("Content-Type", "application/json")
		client := &netHttp.Client{}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println("Failed to send HTTP request:", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if err != nil {
			t.Fatalf("Failed to send POST request: %v", err)
		}
		if resp.StatusCode != netHttp.StatusOK {
			t.Errorf("Expected status code 200, got %d", resp.StatusCode)
		}
	}
	results, err := store.FindAllBy(context.Background(), "1=1")
	assert.Equal(t, len(results), 1)
	assert.Equal(t, results[0].Type, config.Pod)
	assert.Equal(t, results[0].Name, "example-pod")
	assert.Equal(t, (*results[0].MetricLabels)["namespace"], "default")
	assert.Equal(t, (*results[0].MetricLabels)["pod"], "example-pod")
	assert.Equal(t, (*results[0].MetricLabels)["resource_type"], "pod")
}

// createAPIKeyFile creates a temporary API key file with the given content.
// It returns the file path and a cleanup function to remove the file.
func createAPIKeyFile(t *testing.T, apiKeyContent string) string {
	apiKeyFile, err := os.CreateTemp("", "api_key-*.txt")
	require.NoError(t, err, "Failed to create temp API key file")

	_, err = apiKeyFile.Write([]byte(apiKeyContent))
	require.NoError(t, err, "Failed to write to temp API key file")

	err = apiKeyFile.Close()
	require.NoError(t, err, "Failed to close temp API key file")

	t.Cleanup(func() {
		_ = os.Remove(apiKeyFile.Name())
	})

	return apiKeyFile.Name()
}

func TestRemoteWrite(t *testing.T) {
	apiKeyPath := createAPIKeyFile(t, os.Getenv("CLOUDZERO_DEV_API_KEY"))

	// Set up the configuration settings directly
	settings := &config.Settings{
		APIKeyPath:     apiKeyPath,
		CloudAccountID: os.Getenv("CLOUD_ACCOUNT_ID"),
		Region:         os.Getenv("CSP_REGION"),
		ClusterName:    os.Getenv("CLUSTER_NAME"),
		Host:           os.Getenv("CLOUDZERO_HOST"),
		RemoteWrite: config.RemoteWrite{
			MaxBytesPerSend: 10000000,
			SendInterval:    60 * time.Second,
			SendTimeout:     10 * time.Second,
			MaxRetries:      3,
		},
		Database: config.Database{
			BatchUpdateSize: 100, // Set the batch update size
		},
	}
	settings.SetAPIKey()

	// Manually set the RemoteWrite URL
	baseURL, err := url.Parse(fmt.Sprintf("https://%s", settings.Host))
	if err != nil {
		t.Fatalf("Malformed URL: %v", err)
	}
	baseURL.Path += "/v1/container-metrics"
	params := url.Values{}
	params.Add("cluster_name", settings.ClusterName)
	params.Add("cloud_account_id", settings.CloudAccountID)
	params.Add("region", settings.Region)
	baseURL.RawQuery = params.Encode()
	settings.RemoteWrite.Host = baseURL.String()

	clock := utils.Clock{}
	store, err := repo.NewInMemoryResourceRepository(&clock)
	if err != nil {
		t.Fatalf("Failed to create storage system: %v", err)
	}

	// Insert dummy data into the database
	insertDummyData(store)

	remoteWriter := pusher.New(context.Background(), store, &clock, settings)
	writer := remoteWriter.(*pusher.MetricsPusher)

	// Call the Flush method and check for errors
	err = writer.Flush()
	assert.NilError(t, err)
}

func insertDummyData(writer types.ResourceStore) {
	dummyRecords := []types.ResourceTags{
		{
			Type:          config.Pod,
			Name:          "example-pod-with-namespace",
			Namespace:     stringPtr("default"),
			Labels:        &config.MetricLabelTags{"example-label": "pod-label-1", "env": "production"},
			MetricLabels:  &config.MetricLabels{"namespace": "default", "pod": "example-pod-with-namespace", "resource_type": "pod"},
			Annotations:   &config.MetricLabelTags{"example-annotation": "pod-annotation-1"},
			RecordCreated: time.Now(),
			RecordUpdated: time.Now(),
		},
	}

	for _, record := range dummyRecords {
		log.Printf("Inserting dummy record: %+v", record)
		err := writer.Create(context.Background(), &record) // Use the WriteData method to insert data
		if err != nil {
			log.Printf("failed to insert dummy data: %v", err)
		}
	}
}

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	return &s
}

type AdmissionReviewRequest struct {
	ApiVersion string           `json:"apiVersion"`
	Kind       string           `json:"kind"`
	Request    AdmissionRequest `json:"request"`
}

type AdmissionRequest struct {
	UID       string                 `json:"uid"`
	Kind      ObjectReference        `json:"kind"`
	Resource  ObjectReference        `json:"resource"`
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace"`
	Operation string                 `json:"operation"`
	UserInfo  UserInfo               `json:"userInfo"`
	Object    map[string]interface{} `json:"object"`
}

type ObjectReference struct {
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
}

type UserInfo struct {
	Username string   `json:"username"`
	UID      string   `json:"uid"`
	Groups   []string `json:"groups"`
}

func CreatePodAdmissionReviewRequest() AdmissionReviewRequest {
	return AdmissionReviewRequest{
		ApiVersion: "admission.k8s.io/v1",
		Kind:       "AdmissionReview",
		Request: AdmissionRequest{
			UID:       "12345",
			Kind:      ObjectReference{Group: "", Version: "v1", Kind: "Pod"},
			Resource:  ObjectReference{Group: "", Version: "v1", Kind: "pods"},
			Name:      "example-pod",
			Namespace: "default",
			Operation: "CREATE",
			UserInfo:  UserInfo{Username: "system:serviceaccount:default:default", UID: "user-uid", Groups: []string{"system:authenticated"}},
			Object: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":      "example-pod",
					"namespace": "default",
					"labels": map[string]string{
						"example-label": "pod-label-1",
						"env":           "production",
					},
					"annotations": map[string]string{
						"example-annotation": "pod-annotation-1",
					},
				},
			},
		},
	}
}

func CreateDeploymentAdmissionReviewRequest() AdmissionReviewRequest {
	return AdmissionReviewRequest{
		ApiVersion: "admission.k8s.io/v1",
		Kind:       "AdmissionReview",
		Request: AdmissionRequest{
			UID:       "12345",
			Kind:      ObjectReference{Group: "apps", Version: "v1", Kind: "Deployment"},
			Resource:  ObjectReference{Group: "apps", Version: "v1", Kind: "deployments"},
			Name:      "example-deployment",
			Namespace: "default",
			Operation: "CREATE",
			UserInfo:  UserInfo{Username: "system:serviceaccount:default:default", UID: "user-uid", Groups: []string{"system:authenticated"}},
			Object:    map[string]interface{}{"metadata": map[string]interface{}{"name": "example-deployment", "namespace": "default"}},
		},
	}
}
