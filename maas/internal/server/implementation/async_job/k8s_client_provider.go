package async_job

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// K8sClientProvider abstracts Kubernetes client creation for easier testing.
type K8sClientProvider interface {
	GetClientset() (*kubernetes.Clientset, error)
}

// InClusterK8sClientProvider creates a clientset from in-cluster configuration.
type InClusterK8sClientProvider struct{}

func (p *InClusterK8sClientProvider) GetClientset() (*kubernetes.Clientset, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}
	return clientset, nil
}

func resolveNamespace(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if env := os.Getenv("POD_NAMESPACE"); env != "" {
		return env
	}
	return "default"
}
