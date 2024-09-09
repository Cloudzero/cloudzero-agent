package k8s

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// BuildKubeClient builds a Kubernetes clientset from a kubeconfig path
func BuildKubeClient(kubeconfigPath string) (kubernetes.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, errors.Wrap(err, "building kubeconfig")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "creating clientset")
	}

	return clientset, nil
}

// GetServiceURLs retrieves the URLs for services containing 'kube-state-metrics' and 'node-exporter' substrings
func GetServiceURLs(ctx context.Context, clientset kubernetes.Interface) (string, string, error) {
    // List all services in all namespaces
    services, err := clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
    if err != nil {
        return "", "", errors.Wrap(err, "listing services")
    }

    var kubeStateMetricsURL, nodeExporterURL string

    // Filter services for substrings 'kube-state-metrics' and 'node-exporter' and generate URLs
    for _, service := range services.Items {
        if strings.Contains(service.Name, "kube-state-metrics") {
            if len(service.Spec.Ports) > 0 {
                kubeStateMetricsURL = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", service.Name, service.Namespace, service.Spec.Ports[0].Port)
            }
        } else if strings.Contains(service.Name, "node-exporter") {
            if len(service.Spec.Ports) > 0 {
                nodeExporterURL = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", service.Name, service.Namespace, service.Spec.Ports[0].Port)
            }
        }
    }

    if kubeStateMetricsURL == "" {
        return "", "", fmt.Errorf("kube-state-metrics service not found. Please install kube-state-metrics.")
    }

    if nodeExporterURL == "" {
        return "", "", fmt.Errorf("node-exporter service not found. Please install node-exporter.")
    }

    return kubeStateMetricsURL, nodeExporterURL, nil
}
