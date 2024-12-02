package kms

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/diagnostic"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/logging"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/status"
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

func (c *checker) Check(ctx context.Context, client *http.Client, accessor status.Accessor) error {
	var (
		retriesRemaining = MaxRetry
		endpointURL      = fmt.Sprintf("%s/metrics", c.cfg.Prometheus.KubeStateMetricsServiceEndpoint)
	)

	c.logger.Infof("Using endpoint URL: %s", endpointURL)

	// Retry logic to handle transient issues
	for attempt := 1; retriesRemaining > 0; attempt++ {
		resp, err := client.Get(endpointURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				c.logger.Errorf("Failed to read metrics on attempt %d: %v", attempt, err)
				accessor.AddCheck(&status.StatusCheck{Name: DiagnosticKMS, Passing: false, Error: fmt.Sprintf("Failed to read metrics: %s", err.Error())})
				return nil
			}

			metrics := string(body)
			requiredMetrics := []string{"kube_pod_info", "kube_node_info"} // Add the required metrics here
			for _, metric := range requiredMetrics {
				if !strings.Contains(metrics, metric) {
					c.logger.Errorf("Required metric %s not found on attempt %d", metric, attempt)
					accessor.AddCheck(&status.StatusCheck{Name: DiagnosticKMS, Passing: false, Error: fmt.Sprintf("Required metric %s not found", metric)})
					return nil
				}
			}

			accessor.AddCheck(&status.StatusCheck{Name: DiagnosticKMS, Passing: true})
			return nil
		}

		c.logger.Errorf("Failed to fetch metrics on attempt %d: %v", attempt, err)
		retriesRemaining--
		time.Sleep(RetryInterval)
	}

	accessor.AddCheck(&status.StatusCheck{Name: DiagnosticKMS, Passing: false, Error: fmt.Sprintf("Failed to fetch metrics after %d retries", MaxRetry)})
	return nil
}
