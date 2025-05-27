// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package provider contains code for checking the Kubernetes configuration.
package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic/common"
	logging "github.com/cloudzero/cloudzero-agent/app/logging/validator"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

const DiagnosticK8sProvider = config.DiagnosticK8sProvider

type checker struct {
	cfg    *config.Settings
	logger *logrus.Entry
}

func NewProvider(ctx context.Context, cfg *config.Settings) diagnostic.Provider {
	return &checker{
		cfg: cfg,
		logger: logging.NewLogger().
			WithContext(ctx).WithField(logging.OpField, "k8s_provider"),
	}
}

func (c *checker) Check(ctx context.Context, client *http.Client, accessor status.Accessor) error {
	// get the pid
	pid, err := c.getProviderID(ctx)
	if err != nil {
		c.logger.Error(err.Error())
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticK8sProvider, Error: err.Error()})
		return nil
	}

	// write the status
	accessor.WriteToReport(func(cs *status.ClusterStatus) {
		cs.ProviderId = pid
	})
	accessor.AddCheck(&status.StatusCheck{Name: DiagnosticK8sProvider, Passing: true})
	return nil
}

func (c *checker) getProviderID(ctx context.Context) (string, error) {
	// ensure the correct env variables were injected
	ns, exists := os.LookupEnv("NAMESPACE")
	if !exists {
		return "", errors.New("the env variable `NAMESPACE` must exist")
	}
	name, exists := os.LookupEnv("POD_NAME")
	if !exists {
		return "", errors.New("the env variable `POD_NAME` must exist")
	}

	// create the k8s client
	cfg, err := c.getConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get the k8s client config: %w", err)
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create a k8s client: %w", err)
	}

	// get the pod
	pod, err := client.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to query the pod: %w", err)
	}

	// get the node
	node, err := client.CoreV1().Nodes().Get(ctx, pod.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get the node: %w", err)
	}

	return node.Spec.ProviderID, nil
}

// getConfig returns a k8s config based on the environment
// detecting if we are on the prometheus pod or running
// on a machine with a kubeconfig file
func (c *checker) getConfig() (*rest.Config, error) {
	if common.InPod() {
		return rest.InClusterConfig()
	}

	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}
