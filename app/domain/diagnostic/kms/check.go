// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package kms contains code for testing access the Kubernetes Management Service.
package kms

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic"
	logging "github.com/cloudzero/cloudzero-agent/app/logging/validator"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const DiagnosticKMS = config.DiagnosticKMS

var (
	// Exported so that it can be overridden in tests
	MaxRetry      = 12
	RetryInterval = 10 * time.Second
)

type checker struct {
	cfg       *config.Settings
	logger    *logrus.Entry
	clientset kubernetes.Interface
}

var NewProvider = func(ctx context.Context, cfg *config.Settings, clientset ...kubernetes.Interface) diagnostic.Provider {
	var cs kubernetes.Interface
	if len(clientset) > 0 {
		cs = clientset[0]
	} else {
		// Use the in-cluster config if running inside a cluster, otherwise use the default kubeconfig
		config, err := rest.InClusterConfig()
		if err != nil {
			kubeconfig := clientcmd.RecommendedHomeFile
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				panic(err.Error())
			}
		}

		// Create the clientset
		cs, err = kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
	}

	return &checker{
		cfg: cfg,
		logger: logging.NewLogger().
			WithContext(ctx).WithField(logging.OpField, "ksm"),
		clientset: cs,
	}
}

func (c *checker) Check(_ context.Context, client *http.Client, accessor status.Accessor) error {
	var (
		retriesRemaining = MaxRetry
		endpointURL      = c.cfg.Prometheus.KubeStateMetricsServiceEndpoint + "/metrics"
	)

	c.logger.Infof("Using endpoint URL: %s", endpointURL)

	// Retry logic to handle transient issues
	for attempt := 1; retriesRemaining > 0; attempt++ {
		resp, err := client.Get(endpointURL)
		if err != nil {
			c.logger.Errorf("Failed to fetch metrics on attempt %d: %v", attempt, err)
			retriesRemaining--
			time.Sleep(RetryInterval)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			c.logger.Errorf("Unexpected status code on attempt %d: %d", attempt, resp.StatusCode)
			retriesRemaining--
			time.Sleep(RetryInterval)
			continue
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.logger.Errorf("Failed to read metrics on attempt %d: %v", attempt, err)
			accessor.AddCheck(&status.StatusCheck{Name: DiagnosticKMS, Passing: false, Error: "Failed to read metrics: " + err.Error()})
			return nil
		}

		metrics := string(body)
		allMetricsFound := true
		for _, metric := range c.cfg.Prometheus.KubeMetrics {
			if strings.Contains(metrics, metric) {
				c.logger.Infof("Found required metric %s on attempt %d", metric, attempt)
			} else {
				c.logger.Errorf("Required metric %s not found on attempt %d", metric, attempt)
				accessor.AddCheck(&status.StatusCheck{Name: DiagnosticKMS, Passing: false, Error: fmt.Sprintf("Required metric %s not found", metric)})
				allMetricsFound = false
			}
		}

		if allMetricsFound {
			accessor.AddCheck(&status.StatusCheck{Name: DiagnosticKMS, Passing: true})
			return nil
		}

		retriesRemaining--
		time.Sleep(RetryInterval)
	}

	accessor.AddCheck(&status.StatusCheck{Name: DiagnosticKMS, Passing: false, Error: fmt.Sprintf("Failed to fetch metrics after %d retries", MaxRetry)})
	return nil
}
