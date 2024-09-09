package k8s

import (
	"context"
	"fmt"

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

// ListServices lists all Kubernetes services in all namespaces
func ListServices(ctx context.Context, clientset kubernetes.Interface) error {
	// List all services in all namespaces
	services, err := clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "listing services")
	}

	// Print the names and namespaces of the services
	fmt.Println("Services in all namespaces:")
	for _, service := range services.Items {
		fmt.Printf(" - %s (Namespace: %s)\n", service.Name, service.Namespace)
	}

	return nil
}
