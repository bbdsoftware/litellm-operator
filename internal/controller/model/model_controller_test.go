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

package model

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	litellmv1alpha1 "github.com/bbdsoftware/litellm-operator/api/litellm/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/controller/base"
	"github.com/bbdsoftware/litellm-operator/internal/controller/common"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
	"github.com/bbdsoftware/litellm-operator/internal/util"
)

// FakeLitellmModelClient implements litellm.LitellmModel for testing
type FakeLitellmModelClient struct {
	CreateModelFunc         func(ctx context.Context, req *litellm.ModelRequest) (litellm.ModelResponse, error)
	GetModelFunc            func(ctx context.Context, id string) (litellm.ModelResponse, error)
	UpdateModelFunc         func(ctx context.Context, req *litellm.ModelRequest) (litellm.ModelResponse, error)
	DeleteModelFunc         func(ctx context.Context, id string) error
	IsModelUpdateNeededFunc func(ctx context.Context, existing *litellm.ModelResponse, req *litellm.ModelRequest) (litellm.ModelUpdateNeeded, error)
	GetModelInfoFunc        func(ctx context.Context, id string) (litellm.ModelResponse, error)

	// call tracking
	CreateCalled bool
	UpdateCalled bool
	DeleteCalled bool
}

func (f *FakeLitellmModelClient) CreateModel(ctx context.Context, req *litellm.ModelRequest) (litellm.ModelResponse, error) {
	f.CreateCalled = true
	if f.CreateModelFunc == nil {
		return litellm.ModelResponse{}, nil
	}
	return f.CreateModelFunc(ctx, req)
}
func (f *FakeLitellmModelClient) GetModel(ctx context.Context, id string) (litellm.ModelResponse, error) {
	if f.GetModelFunc == nil {
		return litellm.ModelResponse{}, nil
	}
	return f.GetModelFunc(ctx, id)
}
func (f *FakeLitellmModelClient) UpdateModel(ctx context.Context, req *litellm.ModelRequest) (litellm.ModelResponse, error) {
	f.UpdateCalled = true
	if f.UpdateModelFunc == nil {
		return litellm.ModelResponse{}, nil
	}
	return f.UpdateModelFunc(ctx, req)
}
func (f *FakeLitellmModelClient) DeleteModel(ctx context.Context, id string) error {
	f.DeleteCalled = true
	if f.DeleteModelFunc == nil {
		return nil
	}
	return f.DeleteModelFunc(ctx, id)
}
func (f *FakeLitellmModelClient) IsModelUpdateNeeded(ctx context.Context, existing *litellm.ModelResponse, req *litellm.ModelRequest) (litellm.ModelUpdateNeeded, error) {
	return f.IsModelUpdateNeededFunc(ctx, existing, req)
}

func (f *FakeLitellmModelClient) GetModelInfo(ctx context.Context, id string) (litellm.ModelResponse, error) {
	if f.GetModelInfoFunc == nil {
		return litellm.ModelResponse{}, nil
	}
	return f.GetModelInfoFunc(ctx, id)
}

var _ = Describe("ModelReconciler", func() {
	const (
		namespace       = "default"
		resourceName    = "test-model"
		secretName      = "test-secret"
		modelSecretName = "test-model-secret"
	)

	createConnectionSecret := func() {
		// create or update so tests are idempotent across runs
		secret := &corev1.Secret{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret)
		if errors.IsNotFound(err) {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secretName,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"masterkey": []byte("dummy-master-key"),
					"url":       []byte("http://dummy-url"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			return
		} else if err != nil {
			Fail("unexpected error getting secret: " + err.Error())
		}
		// update existing
		secret.Data = map[string][]byte{
			"masterkey": []byte("dummy-master-key"),
			"url":       []byte("http://dummy-url"),
		}
		Expect(k8sClient.Update(ctx, secret)).To(Succeed())
	}

	createModelSecret := func() {
		// create or update model secret (Azure-compatible) so tests are idempotent
		secret := &corev1.Secret{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: modelSecretName, Namespace: namespace}, secret)
		if errors.IsNotFound(err) {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      modelSecretName,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					"apiKey":  []byte("sk-azure-test-123"),
					"apiBase": []byte("https://my-resource.openai.azure.com"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
			return
		} else if err != nil {
			Fail("unexpected error getting model secret: " + err.Error())
		}
		// update existing
		secret.Data = map[string][]byte{
			"apiKey":  []byte("sk-azure-test-123"),
			"apiBase": []byte("https://my-resource.openai.azure.com"),
		}
		Expect(k8sClient.Update(ctx, secret)).To(Succeed())
	}

	cleanupSecret := func() {
		secret := &corev1.Secret{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, secret)
		if err == nil {
			_ = k8sClient.Delete(ctx, secret)
		} else if !errors.IsNotFound(err) {
			Fail("unexpected error getting secret: " + err.Error())
		}
	}

	cleanupModelSecret := func() {
		secret := &corev1.Secret{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: modelSecretName, Namespace: namespace}, secret)
		if err == nil {
			_ = k8sClient.Delete(ctx, secret)
		} else if !errors.IsNotFound(err) {
			Fail("unexpected error getting model secret: " + err.Error())
		}
	}

	newModelCR := func() *litellmv1alpha1.Model {
		return &litellmv1alpha1.Model{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespace,
			},
			Spec: litellmv1alpha1.ModelSpec{
				ModelName: resourceName,
				ConnectionRef: litellmv1alpha1.ConnectionRef{
					SecretRef: litellmv1alpha1.SecretRef{
						Namespace:  namespace,
						SecretName: secretName,
					},
				},
				LiteLLMParams: litellmv1alpha1.LiteLLMParams{
					Model: strPtr("azure/gpt-4"),
				},
				ModelSecretRef: litellmv1alpha1.SecretRef{
					Namespace:  namespace,
					SecretName: modelSecretName,
				},
			},
		}
	}

	Context("Model reconciliation behaviour", func() {
		var fakeClient *FakeLitellmModelClient
		var reconciler *ModelReconciler

		BeforeEach(func() {
			createConnectionSecret()
			createModelSecret()
			fakeClient = &FakeLitellmModelClient{}
			reconciler = NewModelReconciler(k8sClient, k8sClient.Scheme())
			reconciler.LitellmModelClient = fakeClient
		})

		AfterEach(func() {
			// cleanup model if exists
			res := &litellmv1alpha1.Model{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, res)
			if err == nil {
				// remove finalizers to allow garbage collection
				res.Finalizers = nil
				_ = k8sClient.Update(ctx, res)
				_ = k8sClient.Delete(ctx, res)
				// wait until it's gone
				Eventually(func() bool {
					r := &litellmv1alpha1.Model{}
					e := k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, r)
					return errors.IsNotFound(e)
				}, time.Second*5, time.Millisecond*200).Should(BeTrue())
			} else if !errors.IsNotFound(err) {
				Fail("unexpected get error: " + err.Error())
			}
			cleanupSecret()
			cleanupModelSecret()
		})

		It("creates model via LiteLLM and correctly updates status", func() {
			// Arrange: create CR without status
			model := newModelCR()
			Expect(k8sClient.Create(ctx, model)).To(Succeed())

			// fake create returns ID and params
			id := "model-123"
			fakeClient.CreateModelFunc = func(ctx context.Context, req *litellm.ModelRequest) (litellm.ModelResponse, error) {
				Expect(req.ModelName).To(Equal(common.AppendModelSourceTag(resourceName, common.ModelTagCRD)))
				return litellm.ModelResponse{
					ModelName: resourceName,
					LiteLLMParams: &litellm.UpdateLiteLLMParams{
						Model: strPtr("gpt-4"),
					},
					ModelInfo: &litellm.ModelInfo{ID: strPtr(id)},
				}, nil
			}

			// Act
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: resourceName, Namespace: namespace}})
			Expect(err).NotTo(HaveOccurred())

			// Assert: fetch updated resource
			fetched := &litellmv1alpha1.Model{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, fetched)).To(Succeed())
			Expect(fakeClient.CreateCalled).To(BeTrue())
			Expect(fetched.Status.ModelId).NotTo(BeNil())
			Expect(*fetched.Status.ModelId).To(Equal(id))
			Expect(fetched.Status.ModelName).NotTo(BeNil())
			Expect(*fetched.Status.ModelName).To(Equal(resourceName))

			// Verify Ready condition is set correctly
			Eventually(func() bool {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, fetched)
				readyCondition := findCondition(fetched.Status.Conditions, base.CondReady)
				return readyCondition != nil && readyCondition.Status == metav1.ConditionTrue
			}, time.Second*5, time.Millisecond*200).Should(BeTrue())

			// Verify ObservedGeneration is set
			Expect(fetched.Status.ObservedGeneration).To(Equal(fetched.Generation))
		})

		It("correctly recognises when existing model needs updating", func() {
			// Arrange: create CR with existing status ID
			model := newModelCR()
			// set initial status to simulate existing model
			id := "model-abc"
			Expect(k8sClient.Create(ctx, model)).To(Succeed())

			created := &litellmv1alpha1.Model{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, created)).To(Succeed())

			created.Status.ModelId = &id
			created.Status.Conditions = append(created.Status.Conditions, metav1.Condition{
				Type:               base.CondReady,
				Status:             metav1.ConditionTrue,
				Reason:             base.ReasonReady,
				Message:            "Model created successfully",
				LastTransitionTime: metav1.Now(),
				ObservedGeneration: created.Generation,
			})
			Expect(k8sClient.Status().Update(ctx, created)).To(Succeed())

			// fake GetModelInfo and update-needed
			fakeClient.GetModelInfoFunc = func(ctx context.Context, gotID string) (litellm.ModelResponse, error) {
				Expect(gotID).To(Equal(id))
				return litellm.ModelResponse{
					ModelName: resourceName,
					ModelInfo: &litellm.ModelInfo{ID: &id},
				}, nil
			}
			fakeClient.IsModelUpdateNeededFunc = func(ctx context.Context, existing *litellm.ModelResponse, req *litellm.ModelRequest) (litellm.ModelUpdateNeeded, error) {
				return litellm.ModelUpdateNeeded{NeedsUpdate: true}, nil
			}
			fakeClient.UpdateModelFunc = func(ctx context.Context, req *litellm.ModelRequest) (litellm.ModelResponse, error) {
				Expect(req.ModelName).To(Equal(common.AppendModelSourceTag(resourceName, common.ModelTagCRD)))
				return litellm.ModelResponse{
					ModelName: common.AppendModelSourceTag(resourceName, common.ModelTagCRD),
					ModelInfo: &litellm.ModelInfo{ID: &id},
				}, nil
			}

			// Act
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: resourceName, Namespace: namespace}})
			Expect(err).NotTo(HaveOccurred())

			// Assert
			Expect(fakeClient.UpdateCalled).To(BeTrue())
		})

		It("properly finalises model deletion when finalizer is present", func() {
			// Arrange: create CR and add finalizer + status id, then mark for deletion
			model := newModelCR()
			id := "to-delete"
			Expect(k8sClient.Create(ctx, model)).To(Succeed())
			// set status id
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, model)).To(Succeed())
			model.Status.ModelId = &id
			Expect(k8sClient.Status().Update(ctx, model)).To(Succeed())

			// add finalizer (creation path normally adds it, but we set it directly for this isolated test)
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: resourceName, Namespace: namespace}, model)).To(Succeed())
			model.Finalizers = append(model.Finalizers, util.FinalizerName)
			Expect(k8sClient.Update(ctx, model)).To(Succeed())

			// delete the resource - API server will set DeletionTimestamp and keep object due to finalizer
			Expect(k8sClient.Delete(ctx, model)).To(Succeed())

			// fake delete
			fakeClient.DeleteModelFunc = func(ctx context.Context, got string) error {
				Expect(got).To(Equal(id))
				return nil
			}

			// Act
			_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: resourceName, Namespace: namespace}})
			Expect(err).NotTo(HaveOccurred())
			Expect(fakeClient.DeleteCalled).To(BeTrue())
		})
	})
})

// helpers
func strPtr(s string) *string { return &s }

// findCondition finds a condition by type in the conditions slice
func findCondition(conditions []metav1.Condition, condType string) *metav1.Condition {
	for _, cond := range conditions {
		if cond.Type == condType {
			return &cond
		}
	}
	return nil
}
