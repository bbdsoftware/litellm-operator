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

import "fmt"

const (
	// Resource name suffixes used for creating child resources
	ConfigMapSuffix      = "-config"      // Suffix for ConfigMap resources
	SecretSuffix         = "-secrets"     // Suffix for Secret resources
	DeploymentSuffix     = "-deployment"  // Suffix for Deployment resources
	ServiceSuffix        = "-service"     // Suffix for Service resources
	IngressSuffix        = "-ingress"     // Suffix for Ingress resources
	ServiceAccountSuffix = "-sa"          // Suffix for ServiceAccount resources
	RoleSuffix           = "-role"        // Suffix for Role resources
	RoleBindingSuffix    = "-rolebinding" // Suffix for RoleBinding resources
)

// GetConfigMapName generates the name for a ConfigMap resource based on the LiteLLM instance name.
func GetConfigMapName(llmName string) string {
	return llmName + ConfigMapSuffix
}

// GetSecretName generates the name for a Secret resource based on the LiteLLM instance name.
func GetSecretName(llmName string) string {
	return llmName + SecretSuffix
}

// GetDeploymentName generates the name for a Deployment resource based on the LiteLLM instance name.
func GetDeploymentName(llmName string) string {
	return llmName + DeploymentSuffix
}

// GetServiceName generates the name for a Service resource based on the LiteLLM instance name.
func GetServiceName(llmName string) string {
	return llmName + ServiceSuffix
}

// GetIngressName generates the name for an Ingress resource based on the LiteLLM instance name.
func GetIngressName(llmName string) string {
	return llmName + IngressSuffix
}

// GetServiceAccountName generates the name for a ServiceAccount resource based on the LiteLLM instance name.
func GetServiceAccountName(llmName string) string {
	return llmName + ServiceAccountSuffix
}

// GetRoleName generates the name for a Role resource based on the LiteLLM instance name.
func GetRoleName(llmName string) string {
	return llmName + RoleSuffix
}

// GetRoleBindingName generates the name for a RoleBinding resource based on the LiteLLM instance name.
func GetRoleBindingName(llmName string) string {
	return llmName + RoleBindingSuffix
}

// GetAppLabels generates the standard application labels for LiteLLM resources.
// These labels are used for resource selection and organisation.
func GetAppLabels(llmName string) map[string]string {
	return map[string]string{
		"app": fmt.Sprintf("litellm-%s", llmName),
	}
}
