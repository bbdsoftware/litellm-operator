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

package team

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
)

type FakeLitellmTeamClient struct {
	teamExists   bool
	updateNeeded bool
}

var fakeTeamResponse = litellm.TeamResponse{
	CreatedAt:      "2024-03-20T10:00:00Z",
	UpdatedAt:      "2024-03-20T10:00:00Z",
	TeamID:         "test-team-id",
	TeamAlias:      "test-alias",
	OrganizationID: "test-org",
	MembersWithRole: []litellm.TeamMemberWithRole{
		{
			UserID:    "user1",
			UserEmail: "user1@example.com",
			Role:      "admin",
		},
	},
}

func (l *FakeLitellmTeamClient) CreateTeam(ctx context.Context, req *litellm.TeamRequest) (litellm.TeamResponse, error) {
	fakeTeamResponse.TeamAlias = req.TeamAlias
	fakeTeamResponse.OrganizationID = req.OrganizationID
	return fakeTeamResponse, nil
}

func (l *FakeLitellmTeamClient) DeleteTeam(ctx context.Context, teamID string) error {
	return nil
}

func (l *FakeLitellmTeamClient) GetTeam(ctx context.Context, teamID string) (litellm.TeamResponse, error) {
	if l.teamExists {
		return fakeTeamResponse, nil
	}
	return litellm.TeamResponse{}, errors.New("team not found")
}

func (l *FakeLitellmTeamClient) GetTeamID(ctx context.Context, teamAlias string) (string, error) {
	if l.teamExists {
		return "test-team-id", nil
	}
	return "", nil
}

func (l *FakeLitellmTeamClient) IsTeamUpdateNeeded(ctx context.Context, teamResponse *litellm.TeamResponse, teamRequest *litellm.TeamRequest) bool {
	return l.updateNeeded
}

func (l *FakeLitellmTeamClient) UpdateTeam(ctx context.Context, req *litellm.TeamRequest) (litellm.TeamResponse, error) {
	return fakeTeamResponse, nil
}

var _ = Describe("Team Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-team"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		team := &authv1alpha1.Team{}

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

			By("creating the custom resource for the Kind Team")
			err := k8sClient.Get(ctx, typeNamespacedName, team)
			if err != nil && apierrors.IsNotFound(err) {
				resource := &authv1alpha1.Team{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: authv1alpha1.TeamSpec{
						ConnectionRef: authv1alpha1.ConnectionRef{
							SecretRef: &authv1alpha1.SecretRef{
								Name: "test-secret",
								Keys: authv1alpha1.SecretKeys{
									MasterKey: "masterkey",
									URL:       "url",
								},
							},
						},
						TeamAlias:      "test-alias",
						OrganizationID: "test-org",
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &authv1alpha1.Team{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance Team")
			By("Deleting the resource from k8s")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			By("Deleting the resource from litellm")
			controllerReconciler := &TeamReconciler{
				Client:      k8sClient,
				Scheme:      k8sClient.Scheme(),
				LitellmTeam: &FakeLitellmTeamClient{teamExists: true},
			}
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the test connection secret")
			connectionSecret := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-secret", Namespace: "default"}, connectionSecret)
			if err == nil {
				Expect(k8sClient.Delete(ctx, connectionSecret)).To(Succeed())
			}
		})

		Context("that does not already exist in litellm", func() {
			It("should successfully reconcile the resource", func() {
				By("Reconciling the created resource")
				controllerReconciler := &TeamReconciler{
					Client:      k8sClient,
					Scheme:      k8sClient.Scheme(),
					LitellmTeam: &FakeLitellmTeamClient{teamExists: false},
				}

				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())
				// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
				// Example: If you expect a certain status condition after reconciliation, verify it here.
				By("Verifying the team status was updated")
				team := &authv1alpha1.Team{}
				err = k8sClient.Get(ctx, typeNamespacedName, team)
				Expect(err).NotTo(HaveOccurred())
				Expect(team.Status.TeamID).To(Equal("test-team-id"))
				Expect(team.Status.TeamAlias).To(Equal("test-alias"))
				Expect(team.Status.OrganizationID).To(Equal("test-org"))
				Expect(team.Status.MembersWithRole).To(Equal([]authv1alpha1.TeamMemberWithRole{
					{
						UserID:    "user1",
						UserEmail: "user1@example.com",
						Role:      "admin",
					},
				}))

				By("Verifying the conditions were updated")
				Expect(team.Status.Conditions).To(HaveLen(1))
				Expect(team.Status.Conditions[0].Type).To(Equal("Ready"))
				Expect(team.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
				Expect(team.Status.Conditions[0].Reason).To(Equal("LitellmSuccess"))
				Expect(team.Status.Conditions[0].Message).To(Equal("Team created in Litellm"))
			})
		})

		Context("that already exists in litellm", func() {
			It("should add an error condition to the status", func() {
				By("Reconciling the created resource")
				controllerReconciler := &TeamReconciler{
					Client:      k8sClient,
					Scheme:      k8sClient.Scheme(),
					LitellmTeam: &FakeLitellmTeamClient{teamExists: true},
				}

				_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
					NamespacedName: typeNamespacedName,
				})
				Expect(err).NotTo(HaveOccurred())

				By("Verifying the conditions were updated")
				team := &authv1alpha1.Team{}
				err = k8sClient.Get(ctx, typeNamespacedName, team)
				Expect(err).NotTo(HaveOccurred())
				Expect(team.Status.Conditions).To(HaveLen(1))
				Expect(team.Status.Conditions[0].Type).To(Equal("Ready"))
				Expect(team.Status.Conditions[0].Status).To(Equal(metav1.ConditionFalse))
				Expect(team.Status.Conditions[0].Reason).To(Equal("DuplicateAlias"))
				Expect(team.Status.Conditions[0].Message).To(Equal("Team with this alias already exists"))
			})
		})
	})
})
