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

package base

import (
	"context"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ============================================================================
// Common Condition Types
// ============================================================================

const (
	CondReady       = "Ready"       // Overall health indicator
	CondProgressing = "Progressing" // Actively reconciling / rolling out
	CondDegraded    = "Degraded"    // Partial failure state
	CondFailed      = "Failed"      // Complete failure state
)

// ============================================================================
// Common Reason Constants
// ============================================================================

const (
	ReasonReconciling         = "Reconciling"
	ReasonConfigError         = "ConfigError"
	ReasonChildCreateOrUpdate = "ChildResourcesUpdating"
	ReasonDependencyNotReady  = "DependencyNotReady"
	ReasonReconcileError      = "ReconcileError"
	ReasonReady               = "Ready"
	ReasonDeleted             = "Deleted"
	ReasonConnectionError     = "ConnectionError"
	ReasonDeleteFailed        = "DeleteFailed"
	ReasonCreateFailed        = "CreateFailed"
	ReasonConversionFailed    = "ConversionFailed"
	ReasonUpdateFailed        = "UpdateFailed"
	ReasonLitellmError        = "LitellmError"
	ReasonLitellmSuccess      = "LitellmSuccess"
	ReasonInvalidSpec         = "InvalidSpec"
)

// ============================================================================
// Base Controller Interface and Struct
// ============================================================================

// StatusManager defines the interface for managing resource status and conditions
type StatusManager[T StatusConditionObject] interface {
	// PatchStatus updates the status subresource
	PatchStatus(ctx context.Context, obj T) error
	// SetCondition sets a condition on the status
	SetCondition(obj T, condType string, status metav1.ConditionStatus, reason, message string)
	// SetConditions sets multiple conditions at once
	SetConditions(obj T, conditions []metav1.Condition)
	// SetSuccessConditions sets standard success conditions
	SetSuccessConditions(obj T, message string)
	// SetErrorConditions sets standard error conditions
	SetErrorConditions(obj T, reason, message string)
	// SetProgressingConditions sets standard progressing conditions
	SetProgressingConditions(obj T, message string)
}

// StatusConditionObject represents any Kubernetes object that has status conditions
type StatusConditionObject interface {
	client.Object
	GetConditions() []metav1.Condition
	SetConditions(conditions []metav1.Condition)
	GetGeneration() int64
}

// BaseController provides common controller functionality
type BaseController[T StatusConditionObject] struct {
	client.Client
	Scheme         *runtime.Scheme
	DefaultTimeout time.Duration
}

// ============================================================================
// Status Management Implementation
// ============================================================================

// PatchStatus updates the status subresource
// For most controller use cases, this method provides the correct behavior
func (b *BaseController[T]) PatchStatus(ctx context.Context, obj T) error {
	return b.Status().Update(ctx, obj)
}

// PatchStatusFrom updates the status subresource using a strategic merge patch
// This is useful when you have both the original and modified objects
func (b *BaseController[T]) PatchStatusFrom(ctx context.Context, original, modified T) error {
	patch := client.MergeFromWithOptions(original, client.MergeFromWithOptimisticLock{})
	return b.Status().Patch(ctx, modified, patch)
}

// SetCondition sets a single condition on the resource status
func (b *BaseController[T]) SetCondition(obj T, condType string, status metav1.ConditionStatus, reason, message string) {
	newCondition := metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: obj.GetGeneration(),
	}

	conditions := obj.GetConditions()
	meta.SetStatusCondition(&conditions, newCondition)
	obj.SetConditions(conditions)
}

// SetConditions sets multiple conditions at once
func (b *BaseController[T]) SetConditions(obj T, conditions []metav1.Condition) {
	existingConditions := obj.GetConditions()
	for _, condition := range conditions {
		condition.ObservedGeneration = obj.GetGeneration()
		meta.SetStatusCondition(&existingConditions, condition)
	}
	obj.SetConditions(existingConditions)
}

// SetSuccessConditions sets standard conditions for successful operations
func (b *BaseController[T]) SetSuccessConditions(obj T, message string) {
	b.SetCondition(obj, CondReady, metav1.ConditionTrue, ReasonReady, message)
	b.SetCondition(obj, CondProgressing, metav1.ConditionFalse, ReasonReady, message)
	b.SetCondition(obj, CondDegraded, metav1.ConditionFalse, ReasonReady, message)
}

// SetErrorConditions sets standard conditions for error states
func (b *BaseController[T]) SetErrorConditions(obj T, reason, message string) {
	b.SetCondition(obj, CondReady, metav1.ConditionFalse, reason, message)
	b.SetCondition(obj, CondProgressing, metav1.ConditionFalse, reason, message)
	b.SetCondition(obj, CondDegraded, metav1.ConditionTrue, reason, message)
}

// SetProgressingConditions sets standard conditions for progressing operations
func (b *BaseController[T]) SetProgressingConditions(obj T, message string) {
	b.SetCondition(obj, CondReady, metav1.ConditionFalse, ReasonReconciling, message)
	b.SetCondition(obj, CondProgressing, metav1.ConditionTrue, ReasonReconciling, message)
	b.SetCondition(obj, CondDegraded, metav1.ConditionFalse, ReasonReconciling, message)
}

// ============================================================================
// Helper Methods
// ============================================================================

// TemporaryError marks an error as retryable
// Use the standard library's error wrapping and interface idioms for retriable errors.

// HandleCommonErrors implements standard error classification: conflicts → retry; not found → ignore; others → retryable
func (b *BaseController[T]) HandleCommonErrors(ctx context.Context, obj T, err error) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Handle conflicts by returning error for controller-runtime retry
	if kerrors.IsConflict(err) {
		log.V(1).Info("Conflict detected, will retry", "error", err)
		return ctrl.Result{}, err
	}

	// Handle not found errors gracefully
	if kerrors.IsNotFound(err) {
		log.V(1).Info("Resource not found, ignoring", "error", err)
		return ctrl.Result{}, nil
	}

	// For other errors, treat as retryable
	log.Error(err, "Handling error with retry")
	return b.HandleErrorRetryable(ctx, obj, err, ReasonReconcileError)
}

// HandleError is a common error handling pattern with status update
func (b *BaseController[T]) HandleErrorRetryable(ctx context.Context, obj T, err error, reason string) (ctrl.Result, error) {

	b.SetErrorConditions(obj, reason, err.Error())
	_ = b.PatchStatus(ctx, obj)

	return ctrl.Result{RequeueAfter: time.Second * 30}, nil
}

func (b *BaseController[T]) HandleErrorFinal(ctx context.Context, obj T, err error, reason string) (ctrl.Result, error) {
	b.SetErrorConditions(obj, reason, err.Error())
	_ = b.PatchStatus(ctx, obj)

	return ctrl.Result{}, nil
}

// HandleSuccess is a common success handling pattern with status update
func (b *BaseController[T]) HandleSuccess(ctx context.Context, obj T, message string) (ctrl.Result, error) {
	b.SetSuccessConditions(obj, message)
	_ = b.PatchStatus(ctx, obj)
	return ctrl.Result{}, nil
}

// HandleProgressing is a common progressing handling pattern with status update
func (b *BaseController[T]) HandleProgressing(ctx context.Context, obj T, message string) (ctrl.Result, error) {
	b.SetProgressingConditions(obj, message)
	_ = b.PatchStatus(ctx, obj)
	return ctrl.Result{}, nil
}

// ============================================================================
// Finalizer Management
// ============================================================================

func (b *BaseController[T]) WithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	to := b.DefaultTimeout
	if to == 0 {
		to = 20 * time.Second
	}
	return context.WithTimeout(ctx, to)
}

func (b *BaseController[T]) AddFinalizer(ctx context.Context, obj client.Object, name string) error {
	for _, f := range obj.GetFinalizers() {
		if f == name {
			return nil
		}
	}
	patch := client.MergeFrom(obj.DeepCopyObject().(client.Object))
	obj.SetFinalizers(append(obj.GetFinalizers(), name))
	return b.Client.Patch(ctx, obj, patch)
}

func (b *BaseController[T]) RemoveFinalizer(ctx context.Context, obj client.Object, name string) error {
	finalizers := obj.GetFinalizers()
	out := finalizers[:0]
	found := false
	for _, f := range finalizers {
		if f == name {
			found = true
			continue
		}
		out = append(out, f)
	}
	if !found {
		return nil
	}
	patch := client.MergeFrom(obj.DeepCopyObject().(client.Object))
	obj.SetFinalizers(out)
	return b.Client.Patch(ctx, obj, patch)
}

func (b *BaseController[T]) HasFinalizer(obj client.Object, name string) bool {
	for _, f := range obj.GetFinalizers() {
		if f == name {
			return true
		}
	}
	return false
}

// ============================================================================
// Resource Fetching Pattern
// ============================================================================

// FetchResource retrieves a resource and handles not-found errors gracefully
func (b *BaseController[T]) FetchResource(ctx context.Context, namespacedName client.ObjectKey, obj T) (T, error) {
	log := log.FromContext(ctx)

	err := b.Get(ctx, namespacedName, obj)
	if err != nil {
		if kerrors.IsNotFound(err) {
			log.V(1).Info("Resource not found. Ignoring since object must be deleted", "resource", namespacedName)
			var zero T
			return zero, nil
		}
		log.Error(err, "Failed to get resource", "resource", namespacedName)
		return obj, err
	}

	return obj, nil
}

// ============================================================================
// Status Update Helpers
// ============================================================================

// UpdateObservedGeneration updates the observed generation on the status
// This is a helper method that controllers can call to set ObservedGeneration
// Individual controllers should implement this based on their status structure
func (b *BaseController[T]) UpdateObservedGeneration(obj T) {
	// This method is intentionally empty as different CRDs have different status structures
	// Controllers should set obj.Status.ObservedGeneration = obj.GetGeneration() directly
}

// EnsureObservedGenerationAndUpdate sets ObservedGeneration and updates status
// This is a convenience method that combines setting ObservedGeneration with status update
func (b *BaseController[T]) EnsureObservedGenerationAndUpdate(ctx context.Context, obj T, setObservedGeneration func(T)) error {
	setObservedGeneration(obj)
	return b.PatchStatus(ctx, obj)
}
