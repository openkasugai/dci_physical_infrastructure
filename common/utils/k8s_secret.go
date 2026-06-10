// Copyright 2026 NTT, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// GetSecretData retrieves a value from a Kubernetes Secret
// Parameters:
//   - secretName: the name of the Secret resource
//   - namespace: the namespace where the Secret is located
//   - key: the key within the Secret's data field
// Returns:
//   - string: the decoded value (Kubernetes client-go automatically decodes base64)
//   - error: any error that occurred during retrieval
func GetSecretData(secretName, namespace, key string) (string, error) {
	if secretName == "" {
		return "", fmt.Errorf("secret name cannot be empty")
	}
	if namespace == "" {
		return "", fmt.Errorf("namespace cannot be empty")
	}
	if key == "" {
		return "", fmt.Errorf("key cannot be empty")
	}

	// Create in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get in-cluster config: %w", err)
	}

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Get Secret
	secret, err := clientset.CoreV1().Secrets(namespace).Get(context.Background(), secretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s/%s: %w", namespace, secretName, err)
	}

	// Get value from secret data
	value, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("key '%s' not found in secret %s/%s", key, namespace, secretName)
	}

	// Kubernetes Secret data is already base64 decoded when accessed via client-go
	return string(value), nil
}
