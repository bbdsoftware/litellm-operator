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

package user

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
)

type FakeLitellmUserClient struct{}

var fakeUserResponse = litellm.UserResponse{
	UserID:    "test-user-id",
	UserEmail: "test-user-email",
	UserRole:  "admin",
}

func (l *FakeLitellmUserClient) CreateUser(ctx context.Context, req *litellm.UserRequest) (litellm.UserResponse, error) {
	return fakeUserResponse, nil
}

func (l *FakeLitellmUserClient) DeleteUser(ctx context.Context, userID string) error {
	return nil
}

func (l *FakeLitellmUserClient) CheckUserExists(ctx context.Context, userID string) (bool, error) {
	return true, nil
}

func (l *FakeLitellmUserClient) GetUser(ctx context.Context, userID string) (litellm.UserResponse, error) {
	return fakeUserResponse, nil
}

func (l *FakeLitellmUserClient) GetUserID(ctx context.Context, userEmail string) (string, error) {
	return "test-user-id", nil
}

func (l *FakeLitellmUserClient) GetTeam(ctx context.Context, teamID string) (litellm.TeamResponse, error) {
	return litellm.TeamResponse{
		TeamAlias: "test-team-alias",
		TeamID:    teamID,
	}, nil
}

func (l *FakeLitellmUserClient) IsUserUpdateNeeded(ctx context.Context, userResponse *litellm.UserResponse, userRequest *litellm.UserRequest) (litellm.UserUpdateNeeded, error) {
	return litellm.UserUpdateNeeded{
		NeedsUpdate: false,
		ChangedFields: []litellm.FieldChange{
			{
				FieldName:     "User is up to date",
				CurrentValue:  "User is up to date",
				ExpectedValue: "User is up to date",
			},
		},
	}, nil
}

func (l *FakeLitellmUserClient) UpdateUser(ctx context.Context, req *litellm.UserRequest) (litellm.UserResponse, error) {
	return fakeUserResponse, nil
}

var _ = Describe("User Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		user := &authv1alpha1.User{}

		BeforeEach(func() {
			By("creating the test secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"masterkey": []byte("test-master-key"),
					"url":       []byte("http://test-url"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())

			By("creating the custom resource for the Kind User")
			err := k8sClient.Get(ctx, typeNamespacedName, user)
			if err != nil && errors.IsNotFound(err) {
				resource := &authv1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: authv1alpha1.UserSpec{
						ConnectionRef: authv1alpha1.ConnectionRef{
							SecretRef: &authv1alpha1.SecretRef{
								Name: "test-secret",
								Keys: authv1alpha1.SecretKeys{
									MasterKey: "masterkey",
									URL:       "url",
								},
							},
						},
						UserEmail: "test-user-email",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &authv1alpha1.User{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance User")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			By("Cleanup the test secret")
			secret := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
			if err == nil {
				Expect(k8sClient.Delete(ctx, secret)).To(Succeed())
			}
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &UserReconciler{
				Client:      k8sClient,
				Scheme:      k8sClient.Scheme(),
				LitellmUser: &FakeLitellmUserClient{},
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
