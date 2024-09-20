package k8s

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// UpdateConfigMap updates the specified ConfigMap
func UpdateConfigMap(ctx context.Context, clientset kubernetes.Interface, namespace, name string, data map[string]string) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}

	_, err := clientset.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
	if err == nil {
		return nil
	}

	// If the ConfigMap does not exist, create it
	if k8serrors.IsNotFound(err) {
		_, createErr := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
		if createErr != nil {
			return errors.Wrap(createErr, "creating configmap")
		}
		return nil
	}

	return errors.Wrap(err, "updating configmap")
}

// BuildKubeClient builds a Kubernetes clientset from the kubeconfig file
func BuildKubeClient(kubeconfigPath string) (kubernetes.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, errors.Wrap(err, "building kubeconfig")
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "building clientset")
	}
	return clientset, nil
}

// GetServiceURLs fetches the URLs for services containing the substrings "kube-state-metrics" and "node-exporter"
func GetServiceURLs(ctx context.Context, clientset kubernetes.Interface, namespace string) (string, string, error) {
	services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", "", errors.Wrap(err, "listing services")
	}

	var kubeStateMetricsURL, nodeExporterURL string

	for _, service := range services.Items {
		if strings.Contains(service.Name, "kube-state-metrics") {
			kubeStateMetricsURL = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", service.Name, service.Namespace, service.Spec.Ports[0].Port)
		}
		if strings.Contains(service.Name, "node-exporter") {
			nodeExporterURL = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", service.Name, service.Namespace, service.Spec.Ports[0].Port)
		}
	}

	if kubeStateMetricsURL == "" || nodeExporterURL == "" {
		return "", "", errors.New("required services not found")
	}

	return kubeStateMetricsURL, nodeExporterURL, nil
}
