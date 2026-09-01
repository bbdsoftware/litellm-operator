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
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

func GetMapFromSecret(ctx context.Context, c client.Client, secretRef client.ObjectKey) (map[string]string, error) {
	secret := &corev1.Secret{}
	err := c.Get(ctx, secretRef, secret)
	if err != nil {
		return nil, err
	}
	secretMap := make(map[string]string)
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
	log := logf.FromContext(ctx)

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Try to get the existing object
		existing := obj.DeepCopyObject().(client.Object)
		err := c.Get(ctx, client.ObjectKeyFromObject(obj), existing)

		if err != nil {
			if client.IgnoreNotFound(err) == nil {
				if err := ctrl.SetControllerReference(owner, obj, scheme); err != nil {
					log.Error(err, "failed to set controller reference for new object")
					return false, err
				}
				if createErr := c.Create(ctx, obj); createErr != nil {
					log.Error(createErr, "failed to create object", "name", obj.GetName(), "namespace", obj.GetNamespace())
					return false, fmt.Errorf("create %s/%s: %w", obj.GetNamespace(), obj.GetName(), createErr)
				}
				log.Info("Created new object", "name", obj.GetName(), "namespace", obj.GetNamespace())
				return false, nil
			}
			// Any other get error (e.g. RBAC/forbidden) should be surfaced
			log.Error(err, "Error getting object")
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
			log.Error(err, "Error setting controller reference")
			return restart, err
		}

		// Try to update
		err = c.Update(ctx, obj)
		if err == nil {
			return restart, nil // Success
		}

		// Check if it's a conflict error
		if isConflictError(err) {
			log.Info("Conflict detected, retrying...", "attempt", attempt+1)
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
//
// The controller registers every child type it manages with Owns(), so writing a
// child object queues another reconcile of the owner. Reporting an update for an
// object that has not actually changed therefore makes the controller drive
// itself, and only a type this function does not recognise falls back to
// updating unconditionally.
//
// The second return value reports whether the change also requires the LiteLLM
// deployment to be restarted.
func needsUpdate(existing, desired client.Object) (update, restart bool) {
	switch desiredObj := desired.(type) {
	case *corev1.ConfigMap:
		existingObj, ok := existing.(*corev1.ConfigMap)
		if !ok {
			return true, false
		}
		if configMapDataNeedsUpdate(existingObj, desiredObj) {
			// A configuration change requires the proxy to pick up the new config.
			return true, true
		}
		// Metadata drift alone does not warrant restarting the proxy.
		return metadataNeedsUpdate(existingObj, desiredObj), false

	case *appsv1.Deployment:
		existingObj, ok := existing.(*appsv1.Deployment)
		if !ok {
			return true, false
		}
		return deploymentNeedsUpdate(existingObj, desiredObj), false

	case *corev1.Service:
		existingObj, ok := existing.(*corev1.Service)
		if !ok {
			return true, false
		}
		return serviceNeedsUpdate(existingObj, desiredObj), false

	case *corev1.Secret:
		existingObj, ok := existing.(*corev1.Secret)
		if !ok {
			return true, false
		}
		return secretNeedsUpdate(existingObj, desiredObj), false

	case *corev1.ServiceAccount:
		existingObj, ok := existing.(*corev1.ServiceAccount)
		if !ok {
			return true, false
		}
		// A ServiceAccount carries no spec of its own, so metadata is all there is.
		return metadataNeedsUpdate(existingObj, desiredObj), false

	case *rbacv1.Role:
		existingObj, ok := existing.(*rbacv1.Role)
		if !ok {
			return true, false
		}
		return roleNeedsUpdate(existingObj, desiredObj), false

	case *rbacv1.RoleBinding:
		existingObj, ok := existing.(*rbacv1.RoleBinding)
		if !ok {
			return true, false
		}
		return roleBindingNeedsUpdate(existingObj, desiredObj), false

	case *networkingv1.Ingress:
		existingObj, ok := existing.(*networkingv1.Ingress)
		if !ok {
			return true, false
		}
		return ingressNeedsUpdate(existingObj, desiredObj), false
	}

	// Default to updating if we can't determine the type
	return true, false
}

// configMapDataNeedsUpdate reports whether the desired ConfigMap data differs
// from the existing data. Only the data is considered, because it is the data
// that the LiteLLM proxy has to be restarted to pick up.
func configMapDataNeedsUpdate(existing, desired *corev1.ConfigMap) bool {
	if len(existing.Data) != len(desired.Data) {
		return true
	}
	for key, value := range desired.Data {
		if existing.Data[key] != value {
			return true
		}
	}
	return false
}

// deploymentNeedsUpdate compares the Deployment fields that this operator sets.
func deploymentNeedsUpdate(existing, desired *appsv1.Deployment) bool {
	if existing.Spec.Replicas != nil && desired.Spec.Replicas != nil {
		if *existing.Spec.Replicas != *desired.Spec.Replicas {
			return true
		}
	}

	existingContainers := existing.Spec.Template.Spec.Containers
	desiredContainers := desired.Spec.Template.Spec.Containers
	if len(existingContainers) > 0 && len(desiredContainers) > 0 {
		if existingContainers[0].Image != desiredContainers[0].Image {
			return true
		}
		if !apiequality.Semantic.DeepEqual(existingContainers[0].Args, desiredContainers[0].Args) {
			return true
		}
	}

	return metadataNeedsUpdate(existing, desired)
}

// serviceNeedsUpdate compares the Service fields that this operator sets. Fields
// the API server populates, such as clusterIP and any assigned node port, are
// deliberately not compared.
func serviceNeedsUpdate(existing, desired *corev1.Service) bool {
	if desired.Spec.Type != "" && existing.Spec.Type != desired.Spec.Type {
		return true
	}
	if !apiequality.Semantic.DeepEqual(existing.Spec.Selector, desired.Spec.Selector) {
		return true
	}
	if len(existing.Spec.Ports) != len(desired.Spec.Ports) {
		return true
	}
	for i, desiredPort := range desired.Spec.Ports {
		existingPort := existing.Spec.Ports[i]
		if existingPort.Name != desiredPort.Name ||
			existingPort.Port != desiredPort.Port ||
			existingPort.TargetPort != desiredPort.TargetPort {
			return true
		}
		// Protocol is defaulted to TCP by the API server when left unset.
		if desiredPort.Protocol != "" && existingPort.Protocol != desiredPort.Protocol {
			return true
		}
	}

	return metadataNeedsUpdate(existing, desired)
}

// secretNeedsUpdate compares the Secret type and data.
func secretNeedsUpdate(existing, desired *corev1.Secret) bool {
	if desired.Type != "" && existing.Type != desired.Type {
		return true
	}
	if !apiequality.Semantic.DeepEqual(existing.Data, desired.Data) {
		return true
	}

	return metadataNeedsUpdate(existing, desired)
}

// roleNeedsUpdate compares the Role rules.
func roleNeedsUpdate(existing, desired *rbacv1.Role) bool {
	if !apiequality.Semantic.DeepEqual(existing.Rules, desired.Rules) {
		return true
	}

	return metadataNeedsUpdate(existing, desired)
}

// roleBindingNeedsUpdate compares the RoleBinding subjects and role reference.
func roleBindingNeedsUpdate(existing, desired *rbacv1.RoleBinding) bool {
	if !apiequality.Semantic.DeepEqual(existing.RoleRef, desired.RoleRef) {
		return true
	}
	if !apiequality.Semantic.DeepEqual(existing.Subjects, desired.Subjects) {
		return true
	}

	return metadataNeedsUpdate(existing, desired)
}

// ingressNeedsUpdate compares the Ingress fields that this operator sets. The
// ingress class name is only compared when the desired object sets it, because
// the API server fills it in from the default IngressClass otherwise.
func ingressNeedsUpdate(existing, desired *networkingv1.Ingress) bool {
	if desired.Spec.IngressClassName != nil {
		if existing.Spec.IngressClassName == nil ||
			*existing.Spec.IngressClassName != *desired.Spec.IngressClassName {
			return true
		}
	}
	if !apiequality.Semantic.DeepEqual(existing.Spec.Rules, desired.Spec.Rules) {
		return true
	}
	if !apiequality.Semantic.DeepEqual(existing.Spec.TLS, desired.Spec.TLS) {
		return true
	}
	if !apiequality.Semantic.DeepEqual(existing.Spec.DefaultBackend, desired.Spec.DefaultBackend) {
		return true
	}

	return metadataNeedsUpdate(existing, desired)
}

// metadataNeedsUpdate reports whether the labels and annotations this operator
// declares are missing or stale on the existing object. Keys the operator does
// not declare are ignored, so labels and annotations written by other
// controllers do not count as drift.
func metadataNeedsUpdate(existing, desired client.Object) bool {
	return !isSubset(desired.GetLabels(), existing.GetLabels()) ||
		!isSubset(desired.GetAnnotations(), existing.GetAnnotations())
}

// isSubset reports whether every key in want is present in have with the same value.
func isSubset(want, have map[string]string) bool {
	for key, value := range want {
		if have[key] != value {
			return false
		}
	}
	return true
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
