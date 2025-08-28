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

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
	litellmv1alpha1 "github.com/bbdsoftware/litellm-operator/api/litellm/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/interfaces"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
	corev1 "k8s.io/api/core/v1"
)

// ConnectionDetails holds the connection information for LiteLLM
type ConnectionDetails struct {
	MasterKey string
	URL       string
}

// NilKeys implements KeysInterface for types that don't have keys
type NilKeys struct{}

// GetMasterKey returns empty string for nil keys
func (n NilKeys) GetMasterKey() string {
	return ""
}

// GetURL returns empty string for nil keys
func (n NilKeys) GetURL() string {
	return ""
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
// This is now a generic function that can handle different ConnectionRef types
func (h *ConnectionHandler) GetConnectionDetails(ctx context.Context, connectionRef interfaces.ConnectionRefInterface, namespace string) (*ConnectionDetails, error) {
	if connectionRef.HasSecretRef() {
		secretRef := connectionRef.GetSecretRef()
		if secretRefInterface, ok := secretRef.(interfaces.SecretRefInterface); ok {
			return h.getConnectionDetailsFromSecretRef(ctx, secretRefInterface, namespace)
		}
		return nil, fmt.Errorf("SecretRef does not implement SecretRefInterface")
	} else if connectionRef.HasInstanceRef() {
		instanceRef := connectionRef.GetInstanceRef()
		if instanceRefInterface, ok := instanceRef.(interfaces.InstanceRefInterface); ok {
			return h.getConnectionDetailsFromInstanceRef(ctx, instanceRefInterface, namespace)
		}
		return nil, fmt.Errorf("InstanceRef does not implement InstanceRefInterface")
	}

	return nil, fmt.Errorf("neither SecretRef nor InstanceRef is specified in ConnectionRef")
}

// getConnectionDetailsFromSecretRef handles different SecretRef types
func (h *ConnectionHandler) getConnectionDetailsFromSecretRef(ctx context.Context, secretRef interfaces.SecretRefInterface, namespace string) (*ConnectionDetails, error) {
	secret := &corev1.Secret{}
	secretNamespace := secretRef.GetNamespace()
	if secretNamespace == "" {
		secretNamespace = namespace
	}

	secretKey := types.NamespacedName{
		Name:      secretRef.GetSecretName(),
		Namespace: secretNamespace,
	}

	if err := h.Get(ctx, secretKey, secret); err != nil {
		return nil, fmt.Errorf("failed to get secret %s: %w", secretRef.GetSecretName(), err)
	}

	// Handle different key structures
	if secretRef.HasKeys() {
		// For SecretRef with Keys structure
		keys := secretRef.GetKeys()
		// GetKeys already returns a KeysInterface; no need for a type assertion
		masterKeyField := keys.GetMasterKey()
		urlField := keys.GetURL()

		if masterKeyField == "" || urlField == "" {
			return nil, fmt.Errorf("secret %s has invalid keys structure", secretRef.GetSecretName())
		}

		masterKeyBytes, exists := secret.Data[masterKeyField]
		if !exists {
			return nil, fmt.Errorf("secret %s does not contain key %s", secretRef.GetSecretName(), masterKeyField)
		}

		urlBytes, exists := secret.Data[urlField]
		if !exists {
			return nil, fmt.Errorf("secret %s does not contain key %s", secretRef.GetSecretName(), urlField)
		}

		return &ConnectionDetails{
			MasterKey: string(masterKeyBytes),
			URL:       string(urlBytes),
		}, nil
	} else {
		// For SecretRef with standard key names
		masterKeyBytes, exists := secret.Data["masterkey"]
		if !exists {
			return nil, fmt.Errorf("secret %s does not contain key masterkey", secretRef.GetSecretName())
		}

		urlBytes, exists := secret.Data["url"]
		if !exists {
			return nil, fmt.Errorf("secret %s does not contain key url", secretRef.GetSecretName())
		}

		return &ConnectionDetails{
			MasterKey: string(masterKeyBytes),
			URL:       string(urlBytes),
		}, nil
	}
}

// getConnectionDetailsFromInstanceRef handles different InstanceRef types
func (h *ConnectionHandler) getConnectionDetailsFromInstanceRef(ctx context.Context, instanceRef interfaces.InstanceRefInterface, namespace string) (*ConnectionDetails, error) {
	// Determine namespace for the instance
	instanceNamespace := instanceRef.GetNamespace()
	if instanceNamespace == "" {
		instanceNamespace = namespace
	}

	// Get the LiteLLM instance
	instance := &litellmv1alpha1.LiteLLMInstance{}
	instanceKey := types.NamespacedName{
		Name:      instanceRef.GetInstanceName(),
		Namespace: instanceNamespace,
	}

	if err := h.Get(ctx, instanceKey, instance); err != nil {
		return nil, fmt.Errorf("failed to get LiteLLM instance %s: %w", instanceRef.GetInstanceName(), err)
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
	url := fmt.Sprintf("http://%s.%s.svc.cluster.local", service.Name, instanceNamespace)

	return &ConnectionDetails{
		MasterKey: strings.TrimSpace(string(masterKeyBytes)),
		URL:       strings.TrimSpace(url),
	}, nil
}

// ConfigureLitellmClient configures a LiteLLM client with connection details
func ConfigureLitellmClient(connectionDetails *ConnectionDetails) *litellm.LitellmClient {
	return litellm.NewLitellmClient(connectionDetails.URL, connectionDetails.MasterKey)
}

// GetConnectionDetailsFromAuthRef is a convenience function for authv1alpha1.ConnectionRef
func (h *ConnectionHandler) GetConnectionDetailsFromAuthRef(ctx context.Context, connectionRef authv1alpha1.ConnectionRef, namespace string) (*ConnectionDetails, error) {
	return h.GetConnectionDetails(ctx, connectionRef, namespace)
}

// GetConnectionDetailsFromLitellmRef is a convenience function for litellmv1alpha1.ConnectionRef
func (h *ConnectionHandler) GetConnectionDetailsFromLitellmRef(ctx context.Context, connectionRef litellmv1alpha1.ConnectionRef, namespace string) (*ConnectionDetails, error) {
	return h.GetConnectionDetails(ctx, connectionRef, namespace)
}

/*
Usage Examples:

// For authv1alpha1.ConnectionRef (used in Team, User, TeamMemberAssociation, VirtualKey controllers)
authConnectionRef := authv1alpha1.ConnectionRef{
    SecretRef: &authv1alpha1.SecretRef{
        Name: "my-secret",
        Keys: authv1alpha1.SecretKeys{
            MasterKey: "masterkey",
            URL: "url",
        },
    },
}
connectionDetails, err := handler.GetConnectionDetailsFromAuthRef(ctx, authConnectionRef, namespace)

// For litellmv1alpha1.ConnectionRef (used in Model controller)
litellmConnectionRef := litellmv1alpha1.ConnectionRef{
    SecretRef: litellmv1alpha1.SecretRef{
        Namespace:  "my-namespace",
        SecretName: "my-secret",
    },
}
connectionDetails, err := handler.GetConnectionDetailsFromLitellmRef(ctx, litellmConnectionRef, namespace)

// Using the generic interface directly (no adapters needed!)
connectionDetails, err := handler.GetConnectionDetails(ctx, authConnectionRef, namespace)
connectionDetails, err := handler.GetConnectionDetails(ctx, litellmConnectionRef, namespace)

// The system is now completely generic and not tied to any specific types:
// - ConnectionRefInterface for different ConnectionRef types
// - SecretRefInterface for different SecretRef types
// - InstanceRefInterface for different InstanceRef types
// - KeysInterface for different key structures
//
// To add a new type, just implement the interfaces on your types:
//
// type MySecretKeys struct {
//     MasterKeyField string `json:"masterKeyField"`
//     URLField       string `json:"urlField"`
// }
//
// func (m MySecretKeys) GetMasterKey() string { return m.MasterKeyField }
// func (m MySecretKeys) GetURL() string { return m.URLField }
//
// No adapters needed - the original types implement the interfaces directly!
*/
