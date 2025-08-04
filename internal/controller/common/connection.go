/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package common

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	litellmv1alpha1 "github.com/bbdsoftware/litellm-operator/api/litellm/v1alpha1"
	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
	corev1 "k8s.io/api/core/v1"
)

// ConnectionDetails holds the connection information for LiteLLM
type ConnectionDetails struct {
	MasterKey string
	URL       string
}

// ConnectionHandler provides methods to handle connection references
type ConnectionHandler struct {
	client.Client
}

// NewConnectionHandler creates a new connection handler
func NewConnectionHandler(client client.Client) *ConnectionHandler {
	return &ConnectionHandler{
		Client: client,
	}
}

// GetConnectionDetails retrieves connection details from either a secret or LiteLLM instance
func (h *ConnectionHandler) GetConnectionDetails(ctx context.Context, connectionRef authv1alpha1.ConnectionRef, namespace string) (*ConnectionDetails, error) {
	if connectionRef.SecretRef != nil {
		return h.getConnectionDetailsFromSecret(ctx, connectionRef.SecretRef, namespace)
	} else if connectionRef.InstanceRef != nil {
		return h.getConnectionDetailsFromInstance(ctx, connectionRef.InstanceRef, namespace)
	}

	return nil, fmt.Errorf("neither SecretRef nor InstanceRef is specified in ConnectionRef")
}

// getConnectionDetailsFromSecret retrieves connection details from a secret
func (h *ConnectionHandler) getConnectionDetailsFromSecret(ctx context.Context, secretRef *authv1alpha1.SecretRef, namespace string) (*ConnectionDetails, error) {
	secret := &corev1.Secret{}
	secretKey := types.NamespacedName{
		Name:      secretRef.Name,
		Namespace: namespace,
	}

	if err := h.Get(ctx, secretKey, secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %s: %w", secretRef.Name, err)
	}

	masterKeyBytes, exists := secret.Data[secretRef.Keys.MasterKey]
	if !exists {
		return nil, fmt.Errorf("secret %s does not contain key %s", secretRef.Name, secretRef.Keys.MasterKey)
	}

	urlBytes, exists := secret.Data[secretRef.Keys.URL]
	if !exists {
		return nil, fmt.Errorf("secret %s does not contain key %s", secretRef.Name, secretRef.Keys.URL)
	}

	return &ConnectionDetails{
		MasterKey: string(masterKeyBytes),
		URL:       string(urlBytes),
	}, nil
}

// getConnectionDetailsFromInstance retrieves connection details from a LiteLLM instance
func (h *ConnectionHandler) getConnectionDetailsFromInstance(ctx context.Context, instanceRef *authv1alpha1.InstanceRef, namespace string) (*ConnectionDetails, error) {
	// Determine namespace for the instance
	instanceNamespace := instanceRef.Namespace
	if instanceNamespace == "" {
		instanceNamespace = namespace
	}

	// Get the LiteLLM instance
	instance := &litellmv1alpha1.LiteLLMInstance{}
	instanceKey := types.NamespacedName{
		Name:      instanceRef.Name,
		Namespace: instanceNamespace,
	}

	if err := h.Get(ctx, instanceKey, instance); err != nil {
		return nil, fmt.Errorf("failed to get LiteLLM instance %s: %w", instanceRef.Name, err)
	}

	// Get the master key from the instance's secret
	secretName := fmt.Sprintf("%s-secrets", instance.Name)
	secret := &corev1.Secret{}
	secretKey := types.NamespacedName{
		Name:      secretName,
		Namespace: instanceNamespace,
	}

	if err := h.Get(ctx, secretKey, secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %s for LiteLLM instance: %w", secretName, err)
	}

	masterKeyBytes, exists := secret.Data["masterkey"]
	if !exists {
		return nil, fmt.Errorf("secret %s does not contain masterkey", secretName)
	}

	// Construct the URL from the service
	serviceName := fmt.Sprintf("%s-service", instance.Name)
	service := &corev1.Service{}
	serviceKey := types.NamespacedName{
		Name:      serviceName,
		Namespace: instanceNamespace,
	}

	if err := h.Get(ctx, serviceKey, service); err != nil {
		return nil, fmt.Errorf("failed to get service %s for LiteLLM instance: %w", serviceName, err)
	}

	// Construct the URL
	url := fmt.Sprintf("http://%s:%d", service.Name, service.Spec.Ports[0].Port)
	return &ConnectionDetails{
		MasterKey: strings.TrimSpace(string(masterKeyBytes)),
		URL:       strings.TrimSpace(url),
	}, nil
}

// ConfigureLitellmClient configures a LiteLLM client with connection details
func ConfigureLitellmClient(connectionDetails *ConnectionDetails) *litellm.LitellmClient {
	return litellm.NewLitellmClient(connectionDetails.URL, connectionDetails.MasterKey)
}
