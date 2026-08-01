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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
)

func init() {
	// Add the auth scheme
	err := authv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
}

var _ = Describe("Team E2E Tests", Ordered, func() {
	BeforeAll(func() {
		// Initialize k8sClient (reinitialize to ensure auth scheme is registered)
		cfg := config.GetConfigOrDie()
		var err error
		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred())
	})

	Context("Team Lifecycle", func() {
		It("should create, update, and delete a team successfully", func() {
			teamAlias := "e2e-test-team"
			teamCRName := "e2e-test-team-cr"

			By("creating a team CR")
			teamCR := createTeamCR(teamCRName, teamAlias)
			Expect(k8sClient.Create(context.Background(), teamCR)).To(Succeed())

			By("verifying the team was created in LiteLLM")
			eventuallyVerify(func() error {
				return verifyTeamExistsInLiteLLM(teamCRName)
			})

			By("verifying team CR has ready status")
			eventuallyVerify(func() error {
				return verifyTeamCRStatus(teamCRName)
			})

			By("updating the team CR")
			updatedTeamCR := &authv1alpha1.Team{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      teamCRName,
				Namespace: e2eTestNamespace,
			}, updatedTeamCR)).To(Succeed())

			// Update team properties
			newBudget := "50"
			newRPM := 500
			updatedTeamCR.Spec.MaxBudget = newBudget
			updatedTeamCR.Spec.RPMLimit = newRPM
			Expect(k8sClient.Update(context.Background(), updatedTeamCR)).To(Succeed())

			By("verifying the team was updated in LiteLLM")
			eventuallyVerify(func() error {
				return verifyTeamUpdatedInLiteLLM(teamCRName, newBudget, newRPM)
			})

			By("deleting the team CR")
			Expect(k8sClient.Delete(context.Background(), updatedTeamCR)).To(Succeed())

			By("verifying the team was deleted from LiteLLM")
			eventuallyVerify(func() error {
				return verifyTeamDeletedFromLiteLLM(teamCRName)
			})
		})

		It("should handle blocking and unblocking a team", func() {
			teamAlias := "blocked-team"
			teamCRName := "blocked-team-cr"

			By("creating a team CR")
			teamCR := createTeamCR(teamCRName, teamAlias)
			Expect(k8sClient.Create(context.Background(), teamCR)).To(Succeed())

			By("verifying team CR has ready status")
			eventuallyVerify(func() error {
				return verifyTeamCRStatus(teamCRName)
			})

			By("blocking the team")
			updatedTeamCR, err := getTeamCR(teamCRName)
			Expect(err).NotTo(HaveOccurred())
			updatedTeamCR.Spec.Blocked = true
			Expect(k8sClient.Update(context.Background(), updatedTeamCR)).To(Succeed())

			By("verifying the team is blocked")
			eventuallyVerify(func() error {
				return verifyTeamBlockedStatus(teamCRName, true)
			})

			By("unblocking the team")
			updatedTeamCR, err = getTeamCR(teamCRName)
			Expect(err).NotTo(HaveOccurred())
			updatedTeamCR.Spec.Blocked = false
			Expect(k8sClient.Update(context.Background(), updatedTeamCR)).To(Succeed())

			By("verifying the team is unblocked")
			eventuallyVerify(func() error {
				return verifyTeamBlockedStatus(teamCRName, false)
			})

			By("cleaning up team CR")
			Expect(deleteTeamCR(teamCRName)).To(Succeed())
		})

		It("should handle multiple teams in the same namespace", func() {
			team1Alias := "multi-test-team-1"
			team2Alias := "multi-test-team-2"
			team1CRName := "multi-test-team-1-cr"
			team2CRName := "multi-test-team-2-cr"

			By("creating first team CR")
			team1CR := createTeamCR(team1CRName, team1Alias)
			Expect(k8sClient.Create(context.Background(), team1CR)).To(Succeed())

			By("creating second team CR")
			team2CR := createTeamCR(team2CRName, team2Alias)
			Expect(k8sClient.Create(context.Background(), team2CR)).To(Succeed())

			By("verifying both teams were created in LiteLLM")
			eventuallyVerify(func() error {
				if err := verifyTeamExistsInLiteLLM(team1CRName); err != nil {
					return err
				}
				return verifyTeamExistsInLiteLLM(team2CRName)
			})

			By("verifying both team CRs have ready status")
			eventuallyVerify(func() error {
				if err := verifyTeamCRStatus(team1CRName); err != nil {
					return err
				}
				return verifyTeamCRStatus(team2CRName)
			})

			By("cleaning up both team CRs")
			Expect(deleteTeamCR(team1CRName)).To(Succeed())
			Expect(deleteTeamCR(team2CRName)).To(Succeed())

			By("verifying both teams were deleted from LiteLLM")
			eventuallyVerify(func() error {
				if err := verifyTeamDeletedFromLiteLLM(team1CRName); err != nil {
					return err
				}
				return verifyTeamDeletedFromLiteLLM(team2CRName)
			})
		})
	})

	Context("Team Validation", func() {
		It("should validate immutable teamAlias field", func() {
			teamAlias := "immutable-team"
			teamCRName := "immutable-team-cr"

			By("creating a team CR")
			teamCR := createTeamCR(teamCRName, teamAlias)
			Expect(k8sClient.Create(context.Background(), teamCR)).To(Succeed())

			By("waiting for team CR to be ready")
			eventuallyVerify(func() error {
				return verifyTeamCRStatus(teamCRName)
			})

			By("trying to update the immutable teamAlias field")
			updatedTeamCR := &authv1alpha1.Team{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      teamCRName,
				Namespace: e2eTestNamespace,
			}, updatedTeamCR)).To(Succeed())

			updatedTeamCR.Spec.TeamAlias = "new-team-alias"
			err := k8sClient.Update(context.Background(), updatedTeamCR)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("TeamAlias is immutable"))

			By("cleaning up team CR")
			Expect(deleteTeamCR(teamCRName)).To(Succeed())
		})

		It("should handle team with budget duration", func() {
			teamAlias := "budget-duration-team"
			teamCRName := "budget-duration-team-cr"

			duration := "1h"

			By("creating a team CR with budget duration")
			teamCR := createTeamWithBudgetDuration(teamCRName, teamAlias, duration)
			Expect(k8sClient.Create(context.Background(), teamCR)).To(Succeed())

			By("verifying the team has correct budget duration set")
			eventuallyVerify(func() error {
				return verifyTeamBudgetDuration(teamCRName, duration)
			})

			By("cleaning up team CR")
			Expect(deleteTeamCR(teamCRName)).To(Succeed())
		})
	})
})

// ============================================================================
// Creation Helpers
// ============================================================================

func createTeamCR(name, teamAlias string) *authv1alpha1.Team {
	return &authv1alpha1.Team{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e2eTestNamespace,
		},
		Spec: authv1alpha1.TeamSpec{
			ConnectionRef: authv1alpha1.ConnectionRef{
				InstanceRef: &authv1alpha1.InstanceRef{
					Namespace: e2eTestNamespace,
					Name:      e2eTestInstance,
				},
			},
			TeamAlias: teamAlias,
			MaxBudget: "25",
			RPMLimit:  250,
			TPMLimit:  2500,
			Models:    []string{e2eTestModelGPT4o},
			Blocked:   false,
		},
	}
}

func createTeamWithBudgetDuration(name, teamAlias, duration string) *authv1alpha1.Team {
	teamCR := createTeamCR(name, teamAlias)
	teamCR.Spec.BudgetDuration = duration
	return teamCR
}

// ============================================================================
// Retrieval Helpers
// ============================================================================

func getTeamCR(teamCRName string) (*authv1alpha1.Team, error) {
	teamCR := &authv1alpha1.Team{}
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      teamCRName,
		Namespace: e2eTestNamespace,
	}, teamCR)
	if err != nil {
		return nil, fmt.Errorf("failed to get team %s: %w", teamCRName, err)
	}
	return teamCR, nil
}

// ============================================================================
// Verification Helpers - LiteLLM
// ============================================================================

func verifyTeamExistsInLiteLLM(teamCRName string) error {
	teamCR, err := getTeamCR(teamCRName)
	if err != nil {
		return err
	}

	if teamCR.Status.TeamID == "" {
		return fmt.Errorf("team %s has empty status.teamID", teamCRName)
	}

	return nil
}

func verifyTeamUpdatedInLiteLLM(teamCRName, expectedBudget string, expectedRPM int) error {
	teamCR, err := getTeamCR(teamCRName)
	if err != nil {
		return err
	}

	// Verify budget
	if err := compareBudgetStrings(expectedBudget, teamCR.Status.MaxBudget); err != nil {
		return err
	}

	// Verify RPM limit
	if teamCR.Status.RPMLimit != expectedRPM {
		return fmt.Errorf("expected RPM limit %d, got %d", expectedRPM, teamCR.Status.RPMLimit)
	}

	return nil
}

func verifyTeamDeletedFromLiteLLM(teamCRName string) error {
	teamCR := &authv1alpha1.Team{}
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      teamCRName,
		Namespace: e2eTestNamespace,
	}, teamCR)

	if err != nil {
		// If the CR is not found, that's acceptable - it means it was deleted
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get team %s: %w", teamCRName, err)
	}

	if teamCR.GetDeletionTimestamp().IsZero() {
		return fmt.Errorf("team %s is not marked for deletion", teamCRName)
	}

	return fmt.Errorf("team %s is not deleted", teamCRName)
}

func verifyTeamBudgetDuration(teamCRName, expectedDuration string) error {
	teamCR, err := getTeamCR(teamCRName)
	if err != nil {
		return err
	}

	if teamCR.Status.BudgetDuration != expectedDuration {
		return fmt.Errorf("expected budget duration %s, got %s", expectedDuration, teamCR.Status.BudgetDuration)
	}

	return nil
}

// ============================================================================
// Verification Helpers - Status
// ============================================================================

func verifyTeamCRStatus(teamCRName string) error {
	teamCR, err := getTeamCR(teamCRName)
	if err != nil {
		return err
	}

	return verifyReady(teamCR.GetConditions(), statusReady)
}

func verifyTeamBlockedStatus(teamCRName string, expectedBlocked bool) error {
	teamCR, err := getTeamCR(teamCRName)
	if err != nil {
		return err
	}
	if teamCR.Status.Blocked != expectedBlocked {
		return fmt.Errorf("expected blocked status %t, got %t", expectedBlocked, teamCR.Status.Blocked)
	}
	return nil
}

// ============================================================================
// Utility Helpers
// ============================================================================

func deleteTeamCR(teamCRName string) error {
	teamCR, err := getTeamCR(teamCRName)
	if err != nil {
		return err
	}
	return k8sClient.Delete(context.Background(), teamCR)
}
