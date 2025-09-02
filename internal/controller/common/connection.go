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
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
type LitellmConnectionHandler struct {
	client.Client
	litellmClient *litellm.LitellmClient
}

func (h *LitellmConnectionHandler) GetLitellmClient() *litellm.LitellmClient {
	return h.litellmClient
}

// NewConnectionHandler creates a new connection handler
func NewLitellmConnectionHandler(client client.Client, ctx context.Context, connectionRef interfaces.ConnectionRefInterface, namespace string) (*LitellmConnectionHandler, error) {

	h := &LitellmConnectionHandler{
		Client: client,
	}

	connectionDetails, err := h.GetConnectionDetails(ctx, connectionRef, namespace)
	if err != nil {
		return nil, err
	}

	h.litellmClient = litellm.NewLitellmClient(connectionDetails.URL, connectionDetails.MasterKey)
	h.litellmClient.TestConnection(ctx)
	if err != nil {
		return h, err
	}

	return h, nil

}

// GetConnectionDetails retrieves connection details from either a secret or LiteLLM instance
// This is now a generic function that can handle different ConnectionRef types
func (h *LitellmConnectionHandler) GetConnectionDetails(ctx context.Context, connectionRef interfaces.ConnectionRefInterface, namespace string) (*ConnectionDetails, error) {
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
func (h *LitellmConnectionHandler) getConnectionDetailsFromSecretRef(ctx context.Context, secretRef interfaces.SecretRefInterface, namespace string) (*ConnectionDetails, error) {
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
func (h *LitellmConnectionHandler) getConnectionDetailsFromInstanceRef(ctx context.Context, instanceRef interfaces.InstanceRefInterface, namespace string) (*ConnectionDetails, error) {
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
	url := ""
	if os.Getenv("LITELLM_URL_OVERRIDE") != "" {
		url = os.Getenv("LITELLM_URL_OVERRIDE")
	} else {
		url = fmt.Sprintf("http://%s.%s.svc.cluster.local", service.Name, instanceNamespace)
	}

	return &ConnectionDetails{
		MasterKey: strings.TrimSpace(string(masterKeyBytes)),
		URL:       strings.TrimSpace(url),
	}, nil
}
