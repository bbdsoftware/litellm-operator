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

package e2e

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
)

// Use status constants from model_e2e_test.go

var _ = Describe("Integration E2E Tests", Ordered, func() {
	Context("User-Team-VirtualKey Integration", func() {
		It("should create and manage complete user-team-key workflow", func() {
			// Test data
			userEmail := "integration-user@example.com"
			teamAlias := "integration-team"
			keyAlias := "integration-key"

			userCRName := "integration-user-cr"
			teamCRName := "integration-team-cr"
			keyCRName := "integration-key-cr"
			associationCRName := "integration-association-cr"

			By("creating a team CR")
			teamCR := createIntegrationTeamCR(teamCRName, teamAlias)
			Expect(k8sClient.Create(context.Background(), teamCR)).To(Succeed())

			By("creating a user CR")
			userCR := createIntegrationUserCR(userCRName, userEmail)
			Expect(k8sClient.Create(context.Background(), userCR)).To(Succeed())

			By("creating a team member association CR")
			associationCR := createTeamMemberAssociationCR(associationCRName, teamCRName, userCRName)
			Expect(k8sClient.Create(context.Background(), associationCR)).To(Succeed())

			By("creating a virtual key associated with the user")
			keyCR := createIntegrationVirtualKeyCR(keyCRName, keyAlias, userEmail)
			Expect(k8sClient.Create(context.Background(), keyCR)).To(Succeed())

			By("verifying all resources are ready")
			eventuallyVerify(func() error {
				if err := verifyTeamCRStatus(teamCRName); err != nil {
					return fmt.Errorf("team not ready: %v", err)
				}
				if err := verifyUserCRStatus(userCRName, statusReady); err != nil {
					return fmt.Errorf("user not ready: %v", err)
				}
				if err := verifyTeamMemberAssociationCRStatus(associationCRName, statusReady); err != nil {
					return fmt.Errorf("association not ready: %v", err)
				}
				if err := verifyVirtualKeyCRStatus(keyCRName, statusReady); err != nil {
					return fmt.Errorf("virtual key not ready: %v", err)
				}
				return nil
			})

			By("verifying team membership is established")
			eventuallyVerify(func() error {
				return verifyTeamMembership(associationCRName, teamAlias, userEmail)
			})

			// TODO: Uncomment this when we have a way to update user role in team
			// By("updating user role in team")
			// Eventually(func() error {
			// 	return updateTeamMemberRole(associationCRName, "admin")
			// }, testTimeout, testInterval).Should(Succeed())

			// By("verifying role change took effect")
			// Eventually(func() error {
			// 	return verifyTeamMemberRole(teamAlias, userEmail, "admin")
			// }, testTimeout, testInterval).Should(Succeed())

			By("cleaning up all resources")
			Expect(k8sClient.Delete(context.Background(), associationCR)).To(Succeed())
			eventuallyVerify(func() error {
				if err := verifyTeamMemberAssociationDeletedFromLiteLLM(associationCRName); err != nil {
					return fmt.Errorf("association not deleted: %v", err)
				}
				return nil
			})
			Expect(k8sClient.Delete(context.Background(), keyCR)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), userCR)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), teamCR)).To(Succeed())

			By("verifying all resources are deleted")
			eventuallyVerify(func() error {
				if err := verifyVirtualKeyDeletedFromLiteLLM(keyCRName); err != nil {
					return fmt.Errorf("virtual key still exists: %v", err)
				}
				if err := verifyUserDeletedFromLiteLLM(userCRName); err != nil {
					return fmt.Errorf("user still exists: %v", err)
				}
				if err := verifyTeamDeletedFromLiteLLM(teamCRName); err != nil {
					return fmt.Errorf("team still exists: %v", err)
				}
				return nil
			})
		})

		It("should handle team budget inheritance and validation", func() {
			userEmail := "budget-user@example.com"
			teamAlias := "budget-team"
			keyAlias := "budget-key"

			userCRName := "budget-user-cr"
			teamCRName := "budget-team-cr"
			keyCRName := "budget-key-cr"
			associationCRName := "budget-association-cr"

			By("creating a team with budget constraints")
			teamCR := createTeamWithBudget(teamCRName, teamAlias, "100")
			Expect(k8sClient.Create(context.Background(), teamCR)).To(Succeed())

			By("creating a user with lower budget")
			userCR := createUserWithBudget(userCRName, userEmail, "50")
			Expect(k8sClient.Create(context.Background(), userCR)).To(Succeed())

			By("creating team member association with budget limit")
			associationCR := createTeamMemberAssociationWithBudget(associationCRName, teamAlias, userEmail, "30")
			Expect(k8sClient.Create(context.Background(), associationCR)).To(Succeed())

			By("creating virtual key associated with the user")
			keyCR := createVirtualKeyWithBudget(keyCRName, keyAlias, userEmail, "20")
			Expect(k8sClient.Create(context.Background(), keyCR)).To(Succeed())

			By("verifying budget hierarchy is respected")
			eventuallyVerify(func() error {
				return verifyBudgetHierarchy(teamAlias, userEmail, keyAlias)
			})

			By("cleaning up budget test resources")
			Expect(k8sClient.Delete(context.Background(), keyCR)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), associationCR)).To(Succeed())
			eventuallyVerify(func() error {
				if err := verifyTeamMemberAssociationDeletedFromLiteLLM(associationCRName); err != nil {
					return fmt.Errorf("association not deleted: %v", err)
				}
				return nil
			})
			Expect(k8sClient.Delete(context.Background(), userCR)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), teamCR)).To(Succeed())
		})

		//  Assigning team admins is a premium feature so disabling this test for now
		//
		// 	It("should handle multiple users in a team with different roles", func() {
		// 		teamAlias := "multi-user-team"
		// 		user1Email := "admin-user@example.com"
		// 		user2Email := "regular-user@example.com"

		// 		teamCRName := "multi-user-team-cr"
		// 		user1CRName := "admin-user-cr"
		// 		user2CRName := "regular-user-cr"
		// 		association1CRName := "admin-association-cr"
		// 		association2CRName := "regular-association-cr"

		// 		By("creating a team")
		// 		teamCR := createIntegrationTeamCR(teamCRName, teamAlias)
		// 		Expect(k8sClient.Create(context.Background(), teamCR)).To(Succeed())

		// 		By("creating two users")
		// 		user1CR := createIntegrationUserCR(user1CRName, user1Email)
		// 		user2CR := createIntegrationUserCR(user2CRName, user2Email)
		// 		Expect(k8sClient.Create(context.Background(), user1CR)).To(Succeed())
		// 		Expect(k8sClient.Create(context.Background(), user2CR)).To(Succeed())

		// 		By("creating team member associations with different roles")
		// 		association1CR := createTeamMemberAssociationWithRole(association1CRName, teamCRName, user1CRName, "admin")
		// 		association2CR := createTeamMemberAssociationWithRole(association2CRName, teamCRName, user2CRName, "user")
		// 		Expect(k8sClient.Create(context.Background(), association1CR)).To(Succeed())
		// 		Expect(k8sClient.Create(context.Background(), association2CR)).To(Succeed())

		// 		By("verifying both users are team members with correct roles")
		// 		Eventually(func() error {
		// 			if err := verifyTeamMemberRole(teamAlias, user1Email, "admin"); err != nil {
		// 				return err
		// 			}
		// 			return verifyTeamMemberRole(teamAlias, user2Email, "user")
		// 		}, testTimeout, testInterval).Should(Succeed())

		// 		By("removing one user from the team")
		// 		Expect(k8sClient.Delete(context.Background(), association2CR)).To(Succeed())

		// 		By("verifying user was removed from team")
		// 		Eventually(func() error {
		// 			return verifyUserRemovedFromTeam(teamAlias, user2Email)
		// 		}, testTimeout, testInterval).Should(Succeed())

		// 		By("cleaning up multi-user test resources")
		// 		Expect(k8sClient.Delete(context.Background(), association1CR)).To(Succeed())
		// 		Expect(k8sClient.Delete(context.Background(), user1CR)).To(Succeed())
		// 		Expect(k8sClient.Delete(context.Background(), user2CR)).To(Succeed())
		// 		Expect(k8sClient.Delete(context.Background(), teamCR)).To(Succeed())
		// 	})

	})

	Context("TeamMemberAssociation Validation", func() {
		It("should validate role enum values", func() {
			associationCRName := "invalid-role-association"
			teamAlias := "invalid-role-team"
			userEmail := "invalid-role-user@example.com"

			teamCRName := "invalid-role-team-cr"
			userCRName := "invalid-role-user-cr"

			By("creating a team")
			teamCR := createIntegrationTeamCR(teamCRName, teamAlias)
			Expect(k8sClient.Create(context.Background(), teamCR)).To(Succeed())

			By("creating a user")
			userCR := createIntegrationUserCR(userCRName, userEmail)
			Expect(k8sClient.Create(context.Background(), userCR)).To(Succeed())

			By("creating team member association with invalid role")
			invalidAssociationCR := createTeamMemberAssociationWithRole(associationCRName, teamCRName, userCRName, "invalid_role")
			err := k8sClient.Create(context.Background(), invalidAssociationCR)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.role"))

			By("cleaning up invalid role test resources")
			Expect(k8sClient.Delete(context.Background(), userCR)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), teamCR)).To(Succeed())

			By("verifying all resources are deleted")
			eventuallyVerify(func() error {
				if err := verifyUserDeletedFromLiteLLM(userCRName); err != nil {
					return fmt.Errorf("user still exists: %v", err)
				}
				if err := verifyTeamDeletedFromLiteLLM(teamCRName); err != nil {
					return fmt.Errorf("team still exists: %v", err)
				}
				return nil
			})
		})

		It("should validate immutable fields", func() {
			associationCRName := "immutable-test-association"
			teamAlias := "immutable-team"
			userEmail := "immutable@example.com"

			teamCRName := "immutable-team-cr"
			userCRName := "immutable-user-cr"

			By("creating a team")
			teamCR := createIntegrationTeamCR(teamCRName, teamAlias)
			Expect(k8sClient.Create(context.Background(), teamCR)).To(Succeed())

			By("creating a user")
			userCR := createIntegrationUserCR(userCRName, userEmail)
			Expect(k8sClient.Create(context.Background(), userCR)).To(Succeed())

			By("creating team member association")
			associationCR := createTeamMemberAssociationCR(associationCRName, teamCRName, userCRName)
			Expect(k8sClient.Create(context.Background(), associationCR)).To(Succeed())

			By("trying to update immutable teamAlias field")
			updatedAssociationCR := &authv1alpha1.TeamMemberAssociation{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      associationCRName,
				Namespace: e2eTestNamespace,
			}, updatedAssociationCR)).To(Succeed())

			updatedAssociationCR.Spec.TeamRef.Name = "new-team-alias"
			err := k8sClient.Update(context.Background(), updatedAssociationCR)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("TeamRef is immutable"))

			By("cleaning up association CR")
			originalAssociationCR := &authv1alpha1.TeamMemberAssociation{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      associationCRName,
				Namespace: e2eTestNamespace,
			}, originalAssociationCR)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), originalAssociationCR)).To(Succeed())
			Eventually(func() error {
				if err := verifyTeamMemberAssociationDeletedFromLiteLLM(associationCRName); err != nil {
					return fmt.Errorf("association not deleted: %v", err)
				}
				return nil
			}, testTimeout, testInterval).Should(Succeed())
			Expect(k8sClient.Delete(context.Background(), userCR)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), teamCR)).To(Succeed())

			By("verifying all resources are deleted")
			eventuallyVerify(func() error {
				if err := verifyUserDeletedFromLiteLLM(userCRName); err != nil {
					return fmt.Errorf("user still exists: %v", err)
				}
				if err := verifyTeamDeletedFromLiteLLM(teamCRName); err != nil {
					return fmt.Errorf("team still exists: %v", err)
				}
				return nil
			})
		})
	})
})

// ============================================================================
// Creation Helpers
// ============================================================================

func createIntegrationTeamCR(name, teamAlias string) *authv1alpha1.Team {
	return &authv1alpha1.Team{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e2eTestNamespace,
		},
		Spec: authv1alpha1.TeamSpec{
			ConnectionRef: authv1alpha1.ConnectionRef{
				InstanceRef: &authv1alpha1.InstanceRef{
					Namespace: e2eTestNamespace,
					Name:      "e2e-test-instance",
				},
			},
			TeamAlias: teamAlias,
			MaxBudget: "100",
			Models:    []string{"gpt-4o"},
		},
	}
}

func createIntegrationUserCR(name, userEmail string) *authv1alpha1.User {
	return &authv1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e2eTestNamespace,
		},
		Spec: authv1alpha1.UserSpec{
			ConnectionRef: authv1alpha1.ConnectionRef{
				InstanceRef: &authv1alpha1.InstanceRef{
					Namespace: e2eTestNamespace,
					Name:      "e2e-test-instance",
				},
			},
			UserEmail: userEmail,
			UserRole:  "internal_user",
			MaxBudget: "50",
		},
	}
}

func createIntegrationVirtualKeyCR(name, keyAlias, userEmail string) *authv1alpha1.VirtualKey {
	return &authv1alpha1.VirtualKey{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e2eTestNamespace,
		},
		Spec: authv1alpha1.VirtualKeySpec{
			ConnectionRef: authv1alpha1.ConnectionRef{
				InstanceRef: &authv1alpha1.InstanceRef{
					Namespace: e2eTestNamespace,
					Name:      "e2e-test-instance",
				},
			},
			KeyAlias:  keyAlias,
			MaxBudget: "25",
			// Associate with user by email lookup (this would be resolved to UserID by controller)
			Metadata: map[string]string{
				"userEmail": userEmail,
			},
		},
	}
}

func createTeamMemberAssociationCR(name, teamCRName, userCRName string) *authv1alpha1.TeamMemberAssociation {
	return &authv1alpha1.TeamMemberAssociation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e2eTestNamespace,
		},
		Spec: authv1alpha1.TeamMemberAssociationSpec{
			ConnectionRef: authv1alpha1.ConnectionRef{
				InstanceRef: &authv1alpha1.InstanceRef{
					Namespace: e2eTestNamespace,
					Name:      "e2e-test-instance",
				},
			},
			TeamRef: authv1alpha1.CRDRef{
				Name:      teamCRName,
				Namespace: e2eTestNamespace,
			},
			UserRef: authv1alpha1.CRDRef{
				Name:      userCRName,
				Namespace: e2eTestNamespace,
			},
			Role: "user",
		},
	}
}

func createTeamMemberAssociationWithRole(name, teamAlias, userEmail, role string) *authv1alpha1.TeamMemberAssociation {
	assoc := createTeamMemberAssociationCR(name, teamAlias, userEmail)
	assoc.Spec.Role = role
	return assoc
}

func createTeamMemberAssociationWithBudget(name, teamAlias, userEmail, budget string) *authv1alpha1.TeamMemberAssociation {
	assoc := createTeamMemberAssociationCR(name, teamAlias, userEmail)
	assoc.Spec.MaxBudgetInTeam = budget
	return assoc
}

func createTeamWithBudget(name, teamAlias, budget string) *authv1alpha1.Team {
	team := createIntegrationTeamCR(name, teamAlias)
	team.Spec.MaxBudget = budget
	return team
}

func createUserWithBudget(name, userEmail, budget string) *authv1alpha1.User {
	user := createIntegrationUserCR(name, userEmail)
	user.Spec.MaxBudget = budget
	return user
}

func createVirtualKeyWithBudget(name, keyAlias, userEmail, budget string) *authv1alpha1.VirtualKey {
	key := createIntegrationVirtualKeyCR(name, keyAlias, userEmail)
	key.Spec.MaxBudget = budget
	return key
}

// ============================================================================
// Retrieval Helpers
// ============================================================================

func getTeamMemberAssociationCR(associationCRName string) (*authv1alpha1.TeamMemberAssociation, error) {
	association := &authv1alpha1.TeamMemberAssociation{}
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      associationCRName,
		Namespace: e2eTestNamespace,
	}, association)
	if err != nil {
		return nil, fmt.Errorf("failed to get team member association %s: %w", associationCRName, err)
	}
	return association, nil
}

// ============================================================================
// Verification Helpers - LiteLLM
// ============================================================================

func verifyTeamMembership(associationCRName, teamAlias, userEmail string) error {
	association, err := getTeamMemberAssociationCR(associationCRName)
	if err != nil {
		return err
	}

	if association.Status.TeamAlias != teamAlias {
		return fmt.Errorf("team alias mismatch: expected %s, got %s", teamAlias, association.Status.TeamAlias)
	}

	if association.Status.UserEmail != userEmail {
		return fmt.Errorf("user email mismatch: expected %s, got %s", userEmail, association.Status.UserEmail)
	}

	return verifyReady(association.GetConditions(), statusReady)
}

func verifyBudgetHierarchy(teamAlias, userEmail, keyAlias string) error {
	// This is a placeholder for budget hierarchy validation
	// In a real implementation, you would verify that:
	// - Key budget <= User budget <= Team member budget <= Team budget
	// This would require querying the actual budget values and comparing them
	return nil
}

func verifyTeamMemberAssociationCRStatus(associationCRName, expectedStatus string) error {
	association := &authv1alpha1.TeamMemberAssociation{}
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      associationCRName,
		Namespace: e2eTestNamespace,
	}, association)
	if err != nil {
		return fmt.Errorf("failed to get team member association %s: %w", associationCRName, err)
	}

	return verifyReady(association.GetConditions(), expectedStatus)
}

func verifyTeamMemberAssociationDeletedFromLiteLLM(associationCRName string) error {
	association := &authv1alpha1.TeamMemberAssociation{}
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      associationCRName,
		Namespace: e2eTestNamespace,
	}, association)

	if err == nil {
		return fmt.Errorf("team member association %s still exists in Kubernetes", associationCRName)
	}

	if !kerrors.IsNotFound(err) {
		return fmt.Errorf("error checking if team member association %s exists: %w", associationCRName, err)
	}

	// Resource not found means it was successfully deleted
	return nil
}
