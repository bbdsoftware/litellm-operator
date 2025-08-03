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

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
)

type FakeLitellmTeamMemberAssociationClient struct{}

func (l *FakeLitellmTeamMemberAssociationClient) CreateTeamMemberAssociation(ctx context.Context, req *litellm.TeamMemberAssociationRequest) (litellm.TeamMemberAssociationResponse, error) {
	return litellm.TeamMemberAssociationResponse{}, nil
}

func (l *FakeLitellmTeamMemberAssociationClient) DeleteTeamMemberAssociation(ctx context.Context, teamAlias string, userEmail string) error {
	return nil
}

func (l *FakeLitellmTeamMemberAssociationClient) GetTeamMemberAssociation(ctx context.Context, teamAlias string, userEmail string) (litellm.TeamMemberAssociationResponse, error) {
	return litellm.TeamMemberAssociationResponse{}, nil
}

func (l *FakeLitellmTeamMemberAssociationClient) IsTeamMemberAssociationUpdateNeeded(ctx context.Context, teamMemberAssociationResponse *litellm.TeamMemberAssociationResponse, teamMemberAssociationRequest *litellm.TeamMemberAssociationRequest) bool {
	return false
}

func (l *FakeLitellmTeamMemberAssociationClient) UpdateTeamMemberAssociation(ctx context.Context, req *litellm.TeamMemberAssociationRequest) (litellm.TeamMemberAssociationResponse, error) {
	return litellm.TeamMemberAssociationResponse{}, nil
}

func (l *FakeLitellmTeamMemberAssociationClient) GetTeam(ctx context.Context, teamAlias string) (litellm.TeamResponse, error) {
	return litellm.TeamResponse{}, nil
}

func (l *FakeLitellmTeamMemberAssociationClient) GetTeamID(ctx context.Context, teamAlias string) (string, error) {
	return "test-team-id", nil
}

func (l *FakeLitellmTeamMemberAssociationClient) GetUserID(ctx context.Context, userEmail string) (string, error) {
	return "test-user-id", nil
}

var _ = Describe("TeamMemberAssociation Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		teammemberassociation := &authv1alpha1.TeamMemberAssociation{}

		BeforeEach(func() {
			By("creating the test connection secret")
			connectionSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"masterkey": []byte("test-master-key"),
					"url":       []byte("http://test-url"),
				},
			}
			Expect(k8sClient.Create(ctx, connectionSecret)).To(Succeed())

			By("creating the custom resource for the Kind TeamMemberAssociation")
			err := k8sClient.Get(ctx, typeNamespacedName, teammemberassociation)
			if err != nil && errors.IsNotFound(err) {
				resource := &authv1alpha1.TeamMemberAssociation{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: authv1alpha1.TeamMemberAssociationSpec{
						ConnectionRef: authv1alpha1.ConnectionRef{
							SecretRef: &authv1alpha1.SecretRef{
								Name: "test-secret",
								Keys: authv1alpha1.SecretKeys{
									MasterKey: "masterkey",
									URL:       "url",
								},
							},
						},
						TeamAlias: "test-team-alias",
						UserEmail: "test-user-email",
						Role:      "user",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &authv1alpha1.TeamMemberAssociation{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance TeamMemberAssociation")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			By("Cleanup the test connection secret")
			connectionSecret := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-secret", Namespace: "default"}, connectionSecret)
			if err == nil {
				Expect(k8sClient.Delete(ctx, connectionSecret)).To(Succeed())
			}
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &TeamMemberAssociationReconciler{
				Client:                       k8sClient,
				Scheme:                       k8sClient.Scheme(),
				LitellmTeamMemberAssociation: &FakeLitellmTeamMemberAssociationClient{},
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
