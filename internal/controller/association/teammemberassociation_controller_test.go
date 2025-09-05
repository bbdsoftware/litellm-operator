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

package association

import (
	"context"
	"errors"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/controller/base"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
	"github.com/bbdsoftware/litellm-operator/internal/util"
)

// mockLitellmTeamMemberAssociationClient implements the LitellmTeamMemberAssociation interface for testing
type mockLitellmTeamMemberAssociationClient struct {
	associations map[string]map[string]*litellm.TeamMemberWithRole // teamAlias -> userEmail -> member info
	teams        map[string]*litellm.TeamResponse                  // teamAlias -> team info
	createError  error
	deleteError  error
	getTeamError error
	getIDError   error
}

func newMockLitellmTeamMemberAssociationClient() *mockLitellmTeamMemberAssociationClient {
	return &mockLitellmTeamMemberAssociationClient{
		associations: make(map[string]map[string]*litellm.TeamMemberWithRole),
		teams:        make(map[string]*litellm.TeamResponse),
	}
}

func (m *mockLitellmTeamMemberAssociationClient) CreateTeamMemberAssociation(ctx context.Context, req *litellm.TeamMemberAssociationRequest) (litellm.TeamMemberAssociationResponse, error) {
	if m.createError != nil {
		return litellm.TeamMemberAssociationResponse{}, m.createError
	}

	// Initialize team associations map if needed
	if m.associations[req.TeamAlias] == nil {
		m.associations[req.TeamAlias] = make(map[string]*litellm.TeamMemberWithRole)
	}

	// Create member
	member := &litellm.TeamMemberWithRole{
		UserID:    "user-" + req.UserEmail,
		UserEmail: req.UserEmail,
		Role:      req.Role,
	}
	m.associations[req.TeamAlias][req.UserEmail] = member

	return litellm.TeamMemberAssociationResponse{
		TeamAlias: req.TeamAlias,
		TeamID:    "team-" + req.TeamAlias,
		UserEmail: req.UserEmail,
		UserID:    member.UserID,
	}, nil
}

func (m *mockLitellmTeamMemberAssociationClient) DeleteTeamMemberAssociation(ctx context.Context, teamAlias string, userEmail string) error {
	if m.deleteError != nil {
		return m.deleteError
	}

	if m.associations[teamAlias] != nil {
		delete(m.associations[teamAlias], userEmail)
	}
	return nil
}

func (m *mockLitellmTeamMemberAssociationClient) GetTeam(ctx context.Context, teamID string) (litellm.TeamResponse, error) {
	if m.getTeamError != nil {
		return litellm.TeamResponse{}, m.getTeamError
	}

	// Find team by ID (simplistic mapping for test)
	for _, team := range m.teams {
		if team.TeamID == teamID {
			// Update members from associations
			var members []litellm.TeamMemberWithRole
			if m.associations[team.TeamAlias] != nil {
				for _, member := range m.associations[team.TeamAlias] {
					members = append(members, *member)
				}
			}
			team.MembersWithRole = members
			return *team, nil
		}
	}

	// Create default team if not found
	teamAlias := teamID[5:] // Remove "team-" prefix
	team := &litellm.TeamResponse{
		TeamID:    teamID,
		TeamAlias: teamAlias,
	}
	if m.associations[teamAlias] != nil {
		var members []litellm.TeamMemberWithRole
		for _, member := range m.associations[teamAlias] {
			members = append(members, *member)
		}
		team.MembersWithRole = members
	}
	m.teams[teamAlias] = team
	return *team, nil
}

func (m *mockLitellmTeamMemberAssociationClient) GetTeamID(ctx context.Context, teamAlias string) (string, error) {
	if m.getIDError != nil {
		return "", m.getIDError
	}
	return "team-" + teamAlias, nil
}

func (m *mockLitellmTeamMemberAssociationClient) GetUserID(ctx context.Context, userEmail string) (string, error) {
	if m.getIDError != nil {
		return "", m.getIDError
	}
	return "user-" + userEmail, nil
}

var _ = Describe("TeamMemberAssociation Controller", func() {
	var reconciler *TeamMemberAssociationReconciler
	var mockClient *mockLitellmTeamMemberAssociationClient

	BeforeEach(func() {
		reconciler = setupTestTeamMemberAssociationReconciler()
		mockClient = reconciler.LitellmClient.(*mockLitellmTeamMemberAssociationClient)
	})

	Context("When reconciling a resource", func() {
		DescribeTable("should handle different scenarios correctly",
			func(
				name string,
				existingAssociation *authv1alpha1.TeamMemberAssociation,
				existingMember *litellm.TeamMemberWithRole,
				createError error,
				expectedResult ctrl.Result,
				expectedError string,
				expectedConditionStatus metav1.ConditionStatus,
				expectedConditionReason string,
			) {
				// Setup existing member if specified
				if existingMember != nil {
					if mockClient.associations[existingAssociation.Spec.TeamAlias] == nil {
						mockClient.associations[existingAssociation.Spec.TeamAlias] = make(map[string]*litellm.TeamMemberWithRole)
					}
					mockClient.associations[existingAssociation.Spec.TeamAlias][existingMember.UserEmail] = existingMember
				}

				// Setup error conditions
				mockClient.createError = createError

				// Create connection secret first
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"masterkey": []byte("test-key"),
						"url":       []byte("http://test-url"),
					},
				}
				err := reconciler.Create(context.Background(), secret)
				Expect(err).NotTo(HaveOccurred())

				// Create the association resource
				err = reconciler.Create(context.Background(), existingAssociation)
				Expect(err).NotTo(HaveOccurred())

				// Execute reconcile
				result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      existingAssociation.Name,
						Namespace: existingAssociation.Namespace,
					},
				})

				// Verify result
				if expectedError != "" {
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring(expectedError))
				} else {
					Expect(err).NotTo(HaveOccurred())
				}
				Expect(result).To(Equal(expectedResult))

				// Verify status conditions
				updatedAssociation := &authv1alpha1.TeamMemberAssociation{}
				err = reconciler.Get(context.Background(), types.NamespacedName{
					Name:      existingAssociation.Name,
					Namespace: existingAssociation.Namespace,
				}, updatedAssociation)
				Expect(err).NotTo(HaveOccurred())

				// Check condition exists and has correct values
				condition := findCondition(updatedAssociation.Status.Conditions, base.CondReady)
				if expectedConditionStatus != "" {
					Expect(condition).NotTo(BeNil())
					if condition != nil {
						Expect(condition.Status).To(Equal(expectedConditionStatus))
						Expect(condition.Reason).To(Equal(expectedConditionReason))
						Expect(condition.ObservedGeneration).To(Equal(existingAssociation.Generation))
					}
				}
			},
			Entry("create new association successfully",
				"create new association successfully",
				createTestTeamMemberAssociation("test-association", "default"),
				nil, // User not in team yet
				nil, // No create error
				ctrl.Result{RequeueAfter: 60 * time.Second},
				"", // No expected error
				metav1.ConditionTrue,
				base.ReasonReady,
			),
			Entry("user already correctly in team",
				"user already correctly in team",
				createTestTeamMemberAssociation("test-association", "default"),
				&litellm.TeamMemberWithRole{
					UserID:    "user-test-association@example.com",
					UserEmail: "test-association@example.com",
					Role:      "user",
				},
				nil, // No create error
				ctrl.Result{RequeueAfter: 60 * time.Second},
				"", // No expected error
				metav1.ConditionTrue,
				base.ReasonReady,
			),
			Entry("litellm creation error",
				"litellm creation error",
				createTestTeamMemberAssociation("test-association", "default"),
				nil, // No existing member
				errors.New("litellm service unavailable"),
				ctrl.Result{RequeueAfter: 30 * time.Second},
				"", // No expected error (handled gracefully)
				metav1.ConditionFalse,
				base.ReasonLitellmError,
			),
		)
	})

	Context("When handling finalizer lifecycle", func() {
		It("should successfully delete external resources and remove finalizer", func() {
			association := &authv1alpha1.TeamMemberAssociation{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-association",
					Namespace:  "default",
					Finalizers: []string{util.FinalizerName},
				},
				Status: authv1alpha1.TeamMemberAssociationStatus{
					TeamAlias: "test-team",
					UserEmail: "test@example.com",
				},
			}

			// Create the association in the fake client first
			err := reconciler.Create(context.Background(), association)
			Expect(err).NotTo(HaveOccurred())

			// Setup existing association in mock client
			if mockClient.associations["test-team"] == nil {
				mockClient.associations["test-team"] = make(map[string]*litellm.TeamMemberWithRole)
			}
			mockClient.associations["test-team"]["test@example.com"] = &litellm.TeamMemberWithRole{
				UserEmail: "test@example.com",
				Role:      "user",
			}

			// Now mark the resource for deletion
			err = reconciler.Delete(context.Background(), association)
			Expect(err).NotTo(HaveOccurred())

			// Test deletion via full reconcile (which will call reconcileDelete internally)
			result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      association.Name,
					Namespace: association.Namespace,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))

			// Verify external resource was deleted
			_, exists := mockClient.associations["test-team"]["test@example.com"]
			Expect(exists).To(BeFalse())

			// Verify finalizer was removed by checking the resource in the client
			updatedAssociation := &authv1alpha1.TeamMemberAssociation{}
			err = reconciler.Get(context.Background(), types.NamespacedName{
				Name:      association.Name,
				Namespace: association.Namespace,
			}, updatedAssociation)
			// The resource should be gone after successful deletion
			Expect(err).To(HaveOccurred())
		})
	})

	Context("When handling errors", func() {
		DescribeTable("should handle different error scenarios correctly",
			func(
				name string,
				createError error,
				getTeamError error,
				expectedCondition string,
				expectedReason string,
				shouldRequeue bool,
			) {
				reconcilerForError := setupTestTeamMemberAssociationReconciler()
				mockClientForError := reconcilerForError.LitellmClient.(*mockLitellmTeamMemberAssociationClient)
				mockClientForError.createError = createError
				mockClientForError.getIDError = getTeamError

				// Create connection secret first
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "default",
					},
					Data: map[string][]byte{
						"masterkey": []byte("test-key"),
						"url":       []byte("http://test-url"),
					},
				}
				err := reconcilerForError.Create(context.Background(), secret)
				Expect(err).NotTo(HaveOccurred())

				association := createTestTeamMemberAssociation("test", "default")
				err = reconcilerForError.Create(context.Background(), association)
				Expect(err).NotTo(HaveOccurred())

				result, err := reconcilerForError.Reconcile(context.Background(), ctrl.Request{
					NamespacedName: types.NamespacedName{Name: "test", Namespace: "default"},
				})

				if shouldRequeue {
					Expect(err).NotTo(HaveOccurred())
					Expect(result.RequeueAfter).To(BeNumerically(">", 0))
				}

				// Check degraded condition is set
				updatedAssociation := &authv1alpha1.TeamMemberAssociation{}
				err = reconcilerForError.Get(context.Background(), types.NamespacedName{Name: "test", Namespace: "default"}, updatedAssociation)
				Expect(err).NotTo(HaveOccurred())

				degradedCondition := findCondition(updatedAssociation.Status.Conditions, expectedCondition)
				Expect(degradedCondition).NotTo(BeNil())
				if degradedCondition != nil {
					Expect(degradedCondition.Status).To(Equal(metav1.ConditionTrue))
					Expect(degradedCondition.Reason).To(Equal(expectedReason))
				}
			},
			Entry("litellm connection error",
				"litellm connection error",
				errors.New("connection refused"),
				nil,
				base.CondDegraded,
				base.ReasonLitellmError,
				true,
			),
			Entry("team validation error",
				"team validation error",
				nil,
				errors.New("team not found"),
				base.CondDegraded,
				base.ReasonConfigError,
				true,
			),
		)
	})
})

// Test utilities

func setupTestTeamMemberAssociationReconciler() *TeamMemberAssociationReconciler {
	scheme := runtime.NewScheme()
	_ = authv1alpha1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&authv1alpha1.TeamMemberAssociation{}).
		Build()

	return &TeamMemberAssociationReconciler{
		BaseController: &base.BaseController[*authv1alpha1.TeamMemberAssociation]{
			Client:         fakeClient,
			Scheme:         scheme,
			DefaultTimeout: 20 * time.Second,
		},
		LitellmClient: newMockLitellmTeamMemberAssociationClient(),
	}
}

func createTestTeamMemberAssociation(name, namespace string) *authv1alpha1.TeamMemberAssociation {
	return &authv1alpha1.TeamMemberAssociation{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			Generation: 1,
		},
		Spec: authv1alpha1.TeamMemberAssociationSpec{
			TeamAlias: "test-team",
			UserEmail: fmt.Sprintf("%s@example.com", name),
			Role:      "user",
			ConnectionRef: authv1alpha1.ConnectionRef{
				SecretRef: &authv1alpha1.SecretRef{
					Name: "test-secret",
					Keys: authv1alpha1.SecretKeys{
						MasterKey: "masterkey",
						URL:       "url",
					},
				},
			},
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
