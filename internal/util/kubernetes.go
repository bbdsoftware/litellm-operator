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

// Package util provides utility functions for Kubernetes operations.
package util

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetMapFromSecret(ctx context.Context, c client.Client, secretRef client.ObjectKey) (map[string]interface{}, error) {
	secret := &corev1.Secret{}
	err := c.Get(ctx, secretRef, secret)
	if err != nil {
		return nil, err
	}
	secretMap := make(map[string]interface{})
	for key, value := range secret.Data {
		secretMap[key] = string(value) // Convert []byte to string
	}
	return secretMap, nil
}

// CreateOrUpdateWithRetry creates or updates a Kubernetes resource with retry logic.
// It implements optimistic concurrency control with exponential backoff to handle
// resource conflicts in high-concurrency environments.
func CreateOrUpdateWithRetry(ctx context.Context, c client.Client, scheme *runtime.Scheme, obj client.Object, owner client.Object) (bool, error) {
	const maxRetries = 5
	restart := false
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Try to get the existing object
		existing := obj.DeepCopyObject().(client.Object)
		err := c.Get(ctx, client.ObjectKeyFromObject(obj), existing)

		if err != nil {
			if client.IgnoreNotFound(err) == nil {
				// Object doesn't exist, create it
				if err := ctrl.SetControllerReference(owner, obj, scheme); err != nil {
					return false, err
				}
				return false, c.Create(ctx, obj)
			}
			return false, err
		}

		// Object exists, check if update is needed
		needsUpdate, restart := needsUpdate(existing, obj)
		if !needsUpdate {
			return restart, nil // No update needed
		}

		// Object exists and needs update
		// Preserve the existing resource version and other metadata
		obj.SetResourceVersion(existing.GetResourceVersion())
		obj.SetUID(existing.GetUID())

		// Set controller reference for the update
		if err := ctrl.SetControllerReference(owner, obj, scheme); err != nil {
			return restart, err
		}

		// Try to update
		err = c.Update(ctx, obj)
		if err == nil {
			return restart, nil // Success
		}

		// Check if it's a conflict error
		if isConflictError(err) {
			if attempt < maxRetries-1 {
				// Wait a bit before retrying (exponential backoff)
				time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
				continue
			}
		}

		return restart, err
	}

	return restart, fmt.Errorf("failed to update after %d attempts", maxRetries)
}

// cause a restart of the deployment
func RestartDeployment(ctx context.Context, c client.Client, name, namespace string) error {
	deployment := &appsv1.Deployment{}
	err := c.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, deployment)
	if err != nil {
		return err
	}

	// Add a restart annotation to force pod recreation
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	return c.Update(ctx, deployment)
}

// needsUpdate checks if the resource needs to be updated by comparing existing and desired states.
// It implements resource-specific comparison logic to determine whether an update is necessary.
func needsUpdate(existing, desired client.Object) (bool, restart bool) {
	// For ConfigMaps, compare the data
	if existingConfigMap, ok := existing.(*corev1.ConfigMap); ok {
		if desiredConfigMap, ok := desired.(*corev1.ConfigMap); ok {
			if len(existingConfigMap.Data) != len(desiredConfigMap.Data) {
				return true, true
			}
			for key, value := range desiredConfigMap.Data {
				if existingConfigMap.Data[key] != value {
					return true, true
				}
			}
			return false, false
		}
	}

	// For Deployments, compare specific fields that matter for our use case
	if existingDeployment, ok := existing.(*appsv1.Deployment); ok {
		if desiredDeployment, ok := desired.(*appsv1.Deployment); ok {
			// Compare replicas
			if existingDeployment.Spec.Replicas != nil && desiredDeployment.Spec.Replicas != nil {
				if *existingDeployment.Spec.Replicas != *desiredDeployment.Spec.Replicas {
					return true, false
				}
			}

			// Compare container image
			if len(existingDeployment.Spec.Template.Spec.Containers) > 0 && len(desiredDeployment.Spec.Template.Spec.Containers) > 0 {
				if existingDeployment.Spec.Template.Spec.Containers[0].Image != desiredDeployment.Spec.Template.Spec.Containers[0].Image {
					return true, false
				}
			}

			// Compare container args
			if len(existingDeployment.Spec.Template.Spec.Containers) > 0 && len(desiredDeployment.Spec.Template.Spec.Containers) > 0 {
				existingArgs := existingDeployment.Spec.Template.Spec.Containers[0].Args
				desiredArgs := desiredDeployment.Spec.Template.Spec.Containers[0].Args
				if len(existingArgs) != len(desiredArgs) {
					return true, false
				}
				for i, arg := range existingArgs {
					if i >= len(desiredArgs) || arg != desiredArgs[i] {
						return true, false
					}
				}
			}

			// For now, we'll be conservative and update if we're not sure
			// In a production environment, you might want more sophisticated comparison
			return false, false
		}
	}

	// Default to updating if we can't determine the type
	return true, false
}

// isConflictError checks if the error is a Kubernetes conflict error.
// Conflict errors occur when a resource has been modified by another process.
func isConflictError(err error) bool {
	if err == nil {
		return false
	}

	// Check if it's a Kubernetes API error
	if statusErr, ok := err.(*errors.StatusError); ok {
		return statusErr.Status().Code == 409 // HTTP 409 Conflict
	}

	// Check error message for conflict indicators
	errMsg := err.Error()
	return len(errMsg) > 0 && (errMsg == "Operation cannot be fulfilled" ||
		errMsg == "the object has been modified; please apply your changes to the latest version and try again")
}

// HandleConflictError handles conflict errors by returning a short requeue delay.
// This allows the controller to retry with the latest resource version.
func HandleConflictError(err error) (ctrl.Result, error) {
	if isConflictError(err) {
		// Return a short requeue delay for conflict errors
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}
	return ctrl.Result{}, err
}

// FromInt converts an integer to an IntOrString type used by Kubernetes.
func FromInt(val int) intstr.IntOrString {
	return intstr.FromInt32(int32(val))
}

// Int32Ptr returns a pointer to an int32 value.
func Int32Ptr(val int32) *int32 {
	return &val
}

// IsAlreadyExists checks if the error indicates that a resource already exists.
func IsAlreadyExists(err error) bool {
	return err != nil && client.IgnoreAlreadyExists(err) == nil
}
