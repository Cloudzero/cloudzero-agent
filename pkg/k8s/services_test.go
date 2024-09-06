package k8s_test

import (
    "testing"

    "github.com/stretchr/testify/assert"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes/fake"

    "github.com/cloudzero/cloudzero-agent-validator/pkg/k8s"
)

// TestListServices tests the ListServices function
func TestListServices(t *testing.T) {
	// Create a fake clientset with some services
	clientset := fake.NewSimpleClientset(
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "service1",
				Namespace: "default",
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "service2",
				Namespace: "kube-system",
			},
		},
	)

	// Test listing services
	err := k8s.ListServices(clientset)
	assert.NoError(t, err)
}
