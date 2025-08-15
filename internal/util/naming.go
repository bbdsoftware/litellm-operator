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

// Package util provides utility functions for resource naming.
package util

import (
	"fmt"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
)

const (
	// Resource name suffixes used for creating child resources
	ConfigMapSuffix              = "-config"      // Suffix for ConfigMap resources
	SecretSuffix                 = "-secrets"     // Suffix for Secret resources
	DeploymentSuffix             = "-deployment"  // Suffix for Deployment resources
	ServiceSuffix                = "-service"     // Suffix for Service resources
	IngressSuffix                = "-ingress"     // Suffix for Ingress resources
	ServiceAccountSuffix         = "-sa"          // Suffix for ServiceAccount resources
	RoleSuffix                   = "-role"        // Suffix for Role resources
	RoleBindingSuffix            = "-rolebinding" // Suffix for RoleBinding resources
	DefaultLLMName               = "litellm"
	DefaultUserSecretAlias       = "user-secrets"
	DefaultVirtualKeySecretAlias = "key"
)

type LitellmResourceNaming struct {
	litellmInstanceName string
}

// NewLitellmResourceNaming creates a new LitellmResourceNaming instance
// It accepts either a string (instance name) or a ConnectionRef
func NewLitellmResourceNaming[T string | *authv1alpha1.ConnectionRef](param T) *LitellmResourceNaming {
	var litellmInstanceName string

	switch v := any(param).(type) {
	case string:
		litellmInstanceName = v
	case *authv1alpha1.ConnectionRef:
		if v.InstanceRef != nil {
			litellmInstanceName = v.InstanceRef.Name
		} else {
			litellmInstanceName = DefaultLLMName
		}
	}

	return &LitellmResourceNaming{
		litellmInstanceName: litellmInstanceName,
	}
}

// GetConfigMapName generates the name for a ConfigMap resource based on the LiteLLM instance name.
func (n *LitellmResourceNaming) GetConfigMapName() string {
	return n.litellmInstanceName + ConfigMapSuffix
}

// GetSecretName generates the name for a Secret resource based on the LiteLLM instance name.
func (n *LitellmResourceNaming) GetSecretName() string {
	return n.litellmInstanceName + SecretSuffix
}

func (n *LitellmResourceNaming) GenerateSecretName(alias string) string {
	return n.litellmInstanceName + "-key-" + alias
}

// GetDeploymentName generates the name for a Deployment resource based on the LiteLLM instance name.
func (n *LitellmResourceNaming) GetDeploymentName() string {
	return n.litellmInstanceName + DeploymentSuffix
}

// GetServiceName generates the name for a Service resource based on the LiteLLM instance name.
func (n *LitellmResourceNaming) GetServiceName() string {
	return n.litellmInstanceName + ServiceSuffix
}

// GetIngressName generates the name for an Ingress resource based on the LiteLLM instance name.
func (n *LitellmResourceNaming) GetIngressName() string {
	return n.litellmInstanceName + IngressSuffix
}

// GetServiceAccountName generates the name for a ServiceAccount resource based on the LiteLLM instance name.
func (n *LitellmResourceNaming) GetServiceAccountName() string {
	return n.litellmInstanceName + ServiceAccountSuffix
}

// GetRoleName generates the name for a Role resource based on the LiteLLM instance name.
func (n *LitellmResourceNaming) GetRoleName() string {
	return n.litellmInstanceName + RoleSuffix
}

// GetRoleBindingName generates the name for a RoleBinding resource based on the LiteLLM instance name.
func (n *LitellmResourceNaming) GetRoleBindingName() string {
	return n.litellmInstanceName + RoleBindingSuffix
}

// GetAppLabels generates the standard application labels for LiteLLM resources.
// These labels are used for resource selection and organisation.
func (n *LitellmResourceNaming) GetAppLabels() map[string]string {
	return map[string]string{
		"app": fmt.Sprintf("litellm-%s", n.litellmInstanceName),
	}
}
