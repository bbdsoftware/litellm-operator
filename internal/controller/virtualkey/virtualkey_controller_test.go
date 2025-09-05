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

package virtualkey

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/controller/base"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
	"github.com/bbdsoftware/litellm-operator/internal/util"
)

// mockLitellmVirtualKeyClient implements the LitellmVirtualKey interface for testing
type mockLitellmVirtualKeyClient struct {
	virtualKeys      map[string]*litellm.VirtualKeyResponse
	createError      error
	updateError      error
	deleteError      error
	getError         error
	checkExistsError error
	keyExists        bool
	updateNeeded     bool
}

func newMockLitellmVirtualKeyClient() *mockLitellmVirtualKeyClient {
	return &mockLitellmVirtualKeyClient{
		virtualKeys:  make(map[string]*litellm.VirtualKeyResponse),
		keyExists:    false,
		updateNeeded: false,
	}
}

func (m *mockLitellmVirtualKeyClient) GenerateVirtualKey(ctx context.Context, req *litellm.VirtualKeyRequest) (litellm.VirtualKeyResponse, error) {
	if m.createError != nil {
		return litellm.VirtualKeyResponse{}, m.createError
	}

	response := litellm.VirtualKeyResponse{
		KeyAlias:  req.KeyAlias,
		KeyName:   "key-" + req.KeyAlias,
		UserID:    req.UserID,
		TeamID:    req.TeamID,
		Key:       "sk-test-" + req.KeyAlias,
		TokenID:   "token-" + req.KeyAlias,
		MaxBudget: req.MaxBudget,
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	m.virtualKeys[req.KeyAlias] = &response
	return response, nil
}

func (m *mockLitellmVirtualKeyClient) DeleteVirtualKey(ctx context.Context, keyAlias string) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	delete(m.virtualKeys, keyAlias)
	return nil
}

func (m *mockLitellmVirtualKeyClient) GetVirtualKeyFromAlias(ctx context.Context, keyAlias string) ([]string, error) {
	if m.getError != nil {
		return []string{}, m.getError
	}
	return []string{m.virtualKeys[keyAlias].Key}, nil
}

func (m *mockLitellmVirtualKeyClient) GetVirtualKeyInfo(ctx context.Context, key string) (litellm.VirtualKeyResponse, error) {
	if m.getError != nil {
		return litellm.VirtualKeyResponse{}, m.getError
	}
	return *m.virtualKeys[key], nil
}

func (m *mockLitellmVirtualKeyClient) GetVirtualKey(ctx context.Context, key string) (litellm.VirtualKeyResponse, error) {
	if m.getError != nil {
		return litellm.VirtualKeyResponse{}, m.getError
	}

	// Find by key value
	for _, vk := range m.virtualKeys {
		if vk.Key == key {
			return *vk, nil
		}
	}

	return litellm.VirtualKeyResponse{}, fmt.Errorf("virtual key not found")
}

func (m *mockLitellmVirtualKeyClient) GetVirtualKeyID(ctx context.Context, keyAlias string) (string, error) {
	if vk, exists := m.virtualKeys[keyAlias]; exists {
		return vk.TokenID, nil
	}
	return "", fmt.Errorf("virtual key not found")
}

func (m *mockLitellmVirtualKeyClient) IsVirtualKeyUpdateNeeded(ctx context.Context, observed *litellm.VirtualKeyResponse, desired *litellm.VirtualKeyRequest) bool {
	return m.updateNeeded
}

func (m *mockLitellmVirtualKeyClient) UpdateVirtualKey(ctx context.Context, req *litellm.VirtualKeyRequest) (litellm.VirtualKeyResponse, error) {
	if m.updateError != nil {
		return litellm.VirtualKeyResponse{}, m.updateError
	}

	if existing, exists := m.virtualKeys[req.KeyAlias]; exists {
		updated := *existing
		updated.UserID = req.UserID
		updated.TeamID = req.TeamID
		updated.MaxBudget = req.MaxBudget
		updated.UpdatedAt = time.Now().Format(time.RFC3339)
		m.virtualKeys[req.KeyAlias] = &updated
		return updated, nil
	}

	return litellm.VirtualKeyResponse{}, fmt.Errorf("virtual key not found")
}

// Helper functions for testing
func setupTestVirtualKeyReconciler(objects ...client.Object) *VirtualKeyReconciler {
	scheme := runtime.NewScheme()
	_ = authv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(&authv1alpha1.VirtualKey{}).
		Build()

	reconciler := NewVirtualKeyReconciler(fakeClient, scheme)
	reconciler.LitellmClient = newMockLitellmVirtualKeyClient()
	reconciler.litellmResourceNaming = util.NewLitellmResourceNaming(&authv1alpha1.ConnectionRef{})

	return reconciler
}

func createTestVirtualKey(name, namespace string) *authv1alpha1.VirtualKey {
	return &authv1alpha1.VirtualKey{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			Generation: 1,
		},
		Spec: authv1alpha1.VirtualKeySpec{
			ConnectionRef: authv1alpha1.ConnectionRef{
				SecretRef: &authv1alpha1.SecretRef{
					Name: "test-connection",
					Keys: authv1alpha1.SecretKeys{
						MasterKey: "masterkey",
						URL:       "url",
					},
				},
			},
			KeyAlias: fmt.Sprintf("%s-alias", name),
			UserID:   "test-user-id",
		},
	}
}

func findCondition(conditions []metav1.Condition, condType string) *metav1.Condition {
	for _, cond := range conditions {
		if cond.Type == condType {
			return &cond
		}
	}
	return nil
}

func assertCondition(conditions []metav1.Condition, condType string, reason string) {
	condition := findCondition(conditions, condType)
	Expect(condition).NotTo(BeNil(), "condition %s should exist", condType)
	if condition != nil {
		Expect(condition.Status).To(Equal(metav1.ConditionTrue), "condition %s status", condType)
		Expect(condition.Reason).To(Equal(reason), "condition %s reason", condType)
	}
}

var _ = Describe("VirtualKey Controller", func() {
	var (
		ctx        context.Context
		reconciler *VirtualKeyReconciler
		virtualKey *authv1alpha1.VirtualKey
		mockClient *mockLitellmVirtualKeyClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		virtualKey = createTestVirtualKey("test-vk", "default")
		// Create reconciler with the VirtualKey object
		reconciler = setupTestVirtualKeyReconciler(virtualKey)
		mockClient = reconciler.LitellmClient.(*mockLitellmVirtualKeyClient)
	})

	Describe("Reconcile", func() {
		Context("when creating a new virtual key", func() {
			BeforeEach(func() {
				// Setup mock to indicate key doesn't exist yet
				mockClient.keyExists = false
			})

			It("should successfully create virtual key in LiteLLM", func() {
				result, err := reconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      virtualKey.Name,
						Namespace: virtualKey.Namespace,
					},
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(60 * time.Second))

				// Verify virtual key was created in mock
				Expect(mockClient.virtualKeys).To(HaveKey(virtualKey.Spec.KeyAlias))

				// Verify status was updated
				updatedVK := &authv1alpha1.VirtualKey{}
				err = reconciler.Get(ctx, types.NamespacedName{
					Name:      virtualKey.Name,
					Namespace: virtualKey.Namespace,
				}, updatedVK)
				Expect(err).NotTo(HaveOccurred())

				Expect(updatedVK.Status.KeyAlias).To(Equal(virtualKey.Spec.KeyAlias))
				Expect(updatedVK.Status.ObservedGeneration).To(Equal(virtualKey.Generation))
				assertCondition(updatedVK.Status.Conditions, base.CondReady, base.ReasonReady)

				// Verify finalizer was added
				Expect(updatedVK.Finalizers).To(ContainElement(util.FinalizerName))
			})

			It("should create a secret for the virtual key", func() {
				_, err := reconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      virtualKey.Name,
						Namespace: virtualKey.Namespace,
					},
				})

				Expect(err).NotTo(HaveOccurred())

				// Verify secret was created
				secretName := reconciler.litellmResourceNaming.GenerateSecretName(virtualKey.Spec.KeyAlias)
				secret := &corev1.Secret{}
				err = reconciler.Get(ctx, types.NamespacedName{
					Name:      secretName,
					Namespace: virtualKey.Namespace,
				}, secret)
				Expect(err).NotTo(HaveOccurred())

				// Verify secret data
				Expect(secret.Data).To(HaveKey("key"))
				Expect(string(secret.Data["key"])).To(Equal("sk-test-" + virtualKey.Spec.KeyAlias))

				// Verify owner reference
				Expect(secret.OwnerReferences).To(HaveLen(1))
				Expect(secret.OwnerReferences[0].Kind).To(Equal("VirtualKey"))
				Expect(secret.OwnerReferences[0].Name).To(Equal(virtualKey.Name))
			})
		})

		Context("when virtual key already exists", func() {
			BeforeEach(func() {
				// Setup existing virtual key in mock
				mockClient.keyExists = true
				mockClient.virtualKeys[virtualKey.Spec.KeyAlias] = &litellm.VirtualKeyResponse{
					KeyAlias: virtualKey.Spec.KeyAlias,
					Key:      "sk-existing-key",
					UserID:   virtualKey.Spec.UserID,
				}

				// Create the secret
				secretName := reconciler.litellmResourceNaming.GenerateSecretName(virtualKey.Spec.KeyAlias)
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: virtualKey.Namespace,
					},
					Data: map[string][]byte{
						"key": []byte("sk-existing-key"),
					},
				}
				Expect(reconciler.Create(ctx, secret)).To(Succeed())

				// Update status to reflect existing key
				virtualKey.Status.KeyAlias = virtualKey.Spec.KeyAlias
				virtualKey.Status.KeySecretRef = secretName
				Expect(reconciler.Status().Update(ctx, virtualKey)).To(Succeed())
			})

			It("should sync without update when no changes needed", func() {
				mockClient.updateNeeded = false

				result, err := reconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      virtualKey.Name,
						Namespace: virtualKey.Namespace,
					},
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(60 * time.Second))

				// Verify conditions are set correctly
				updatedVK := &authv1alpha1.VirtualKey{}
				err = reconciler.Get(ctx, types.NamespacedName{
					Name:      virtualKey.Name,
					Namespace: virtualKey.Namespace,
				}, updatedVK)
				Expect(err).NotTo(HaveOccurred())

				assertCondition(updatedVK.Status.Conditions, base.CondReady, base.ReasonReady)
			})

			It("should update virtual key when drift detected", func() {
				mockClient.updateNeeded = true

				result, err := reconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      virtualKey.Name,
						Namespace: virtualKey.Namespace,
					},
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(60 * time.Second))

				// Verify update was called in mock
				vk := mockClient.virtualKeys[virtualKey.Spec.KeyAlias]
				Expect(vk.UpdatedAt).NotTo(BeEmpty())
			})
		})

		Context("when handling deletion", func() {
			var deletingVK *authv1alpha1.VirtualKey

			BeforeEach(func() {
				// Create a separate VirtualKey object for deletion that includes finalizer and deletion timestamp
				now := metav1.Now()
				deletingVK = &authv1alpha1.VirtualKey{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "deleting-vk",
						Namespace:         "default",
						Generation:        1,
						Finalizers:        []string{util.FinalizerName},
						DeletionTimestamp: &now,
					},
					Spec: authv1alpha1.VirtualKeySpec{
						ConnectionRef: authv1alpha1.ConnectionRef{
							SecretRef: &authv1alpha1.SecretRef{
								Name: "test-connection",
								Keys: authv1alpha1.SecretKeys{
									MasterKey: "masterkey",
									URL:       "url",
								},
							},
						},
						KeyAlias: "deleting-vk-alias",
						UserID:   "test-user-id",
					},
					Status: authv1alpha1.VirtualKeyStatus{
						KeyAlias: "deleting-vk-alias",
					},
				}

				// Create new reconciler with the deleting VirtualKey
				reconciler = setupTestVirtualKeyReconciler(deletingVK)
				mockClient = reconciler.LitellmClient.(*mockLitellmVirtualKeyClient)

				// Setup existing virtual key in mock
				mockClient.virtualKeys[deletingVK.Spec.KeyAlias] = &litellm.VirtualKeyResponse{
					KeyAlias: deletingVK.Spec.KeyAlias,
				}
			})

			It("should delete virtual key from LiteLLM and remove finalizer", func() {
				result, err := reconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      deletingVK.Name,
						Namespace: deletingVK.Namespace,
					},
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))

				// Verify virtual key was deleted from mock
				Expect(mockClient.virtualKeys).NotTo(HaveKey(deletingVK.Spec.KeyAlias))

				// Note: After successful deletion and finalizer removal, the VirtualKey object
				// may be completely removed from the cluster, so we don't attempt to fetch it
			})
		})

		Context("when handling errors", func() {
			BeforeEach(func() {
				// VirtualKey is already created in outer BeforeEach
			})

			It("should handle LiteLLM creation errors", func() {
				mockClient.createError = fmt.Errorf("LiteLLM API error")
				mockClient.keyExists = false

				result, err := reconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      virtualKey.Name,
						Namespace: virtualKey.Namespace,
					},
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(30 * time.Second))

				// Verify error condition is set
				updatedVK := &authv1alpha1.VirtualKey{}
				err = reconciler.Get(ctx, types.NamespacedName{
					Name:      virtualKey.Name,
					Namespace: virtualKey.Namespace,
				}, updatedVK)
				Expect(err).NotTo(HaveOccurred())

				assertCondition(updatedVK.Status.Conditions, base.CondDegraded, base.ReasonLitellmError)
			})

			It("should handle connection errors", func() {
				mockClient.checkExistsError = fmt.Errorf("connection error")

				result, err := reconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      virtualKey.Name,
						Namespace: virtualKey.Namespace,
					},
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(result.RequeueAfter).To(Equal(30 * time.Second))

				// Verify error condition is set
				updatedVK := &authv1alpha1.VirtualKey{}
				err = reconciler.Get(ctx, types.NamespacedName{
					Name:      virtualKey.Name,
					Namespace: virtualKey.Namespace,
				}, updatedVK)
				Expect(err).NotTo(HaveOccurred())

				assertCondition(updatedVK.Status.Conditions, base.CondDegraded, base.ReasonLitellmError)
			})
		})

		Context("when resource is not found", func() {
			It("should return without error", func() {
				result, err := reconciler.Reconcile(ctx, ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      "non-existent",
						Namespace: "default",
					},
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(result).To(Equal(ctrl.Result{}))
			})
		})
	})

	Describe("convertToVirtualKeyRequest", func() {
		It("should correctly convert VirtualKey to VirtualKeyRequest", func() {
			virtualKey.Spec.MaxBudget = "100.50"
			virtualKey.Spec.SoftBudget = "80.25"

			request, err := reconciler.convertToVirtualKeyRequest(virtualKey)

			Expect(err).NotTo(HaveOccurred())
			Expect(request.KeyAlias).To(Equal(virtualKey.Spec.KeyAlias))
			Expect(request.UserID).To(Equal(virtualKey.Spec.UserID))
			Expect(request.MaxBudget).To(Equal(100.50))
			Expect(request.SoftBudget).To(Equal(80.25))
		})

		It("should handle invalid budget values", func() {
			virtualKey.Spec.MaxBudget = "invalid"

			_, err := reconciler.convertToVirtualKeyRequest(virtualKey)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("maxBudget"))
		})
	})
})
