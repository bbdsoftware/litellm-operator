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

package controller

// finalizerName is the name of the finalizer used by the litellm operator
const finalizerName = "litellm-operator.litellm.ai/finalizer"

// ensureMetadata ensures that the metadata contains the managed_by metadata
func ensureMetadata(metadata map[string]string) map[string]string {
	operatorMetadata := map[string]string{
		"managed_by": "litellm-operator",
	}
	for k, v := range metadata {
		operatorMetadata[k] = v
	}
	return operatorMetadata
}

// getSecretName returns the name of the secret for a given alias
func getSecretName(alias string) string {
	return "litellm-key-" + alias
}
