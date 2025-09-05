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
	"os/exec"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/test/utils"
)

var _ = Describe("Team E2E Tests", Ordered, func() {
	Context("Team Lifecycle", func() {
		It("should create, update, and delete a team successfully", func() {
			teamAlias := "e2e-test-team"
			teamCRName := "e2e-test-team-cr"

			By("creating a team CR")
			teamCR := createTeamCR(teamCRName, teamAlias)
			Expect(k8sClient.Create(context.Background(), teamCR)).To(Succeed())

			By("verifying the team was created in LiteLLM")
			Eventually(func() error {
				return verifyTeamExistsInLiteLLM(teamAlias)
			}, testTimeout, testInterval).Should(Succeed())

			By("verifying team CR has ready status")
			Eventually(func() error {
				return verifyTeamCRStatus(teamCRName, "Ready")
			}, testTimeout, testInterval).Should(Succeed())

			By("updating the team CR")
			updatedTeamCR := &authv1alpha1.Team{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      teamCRName,
				Namespace: modelTestNamespace,
			}, updatedTeamCR)).To(Succeed())

			// Update team properties
			updatedTeamCR.Spec.MaxBudget = "50"
			updatedTeamCR.Spec.RPMLimit = 500
			updatedTeamCR.Spec.Models = []string{"gpt-4o", "gpt-3.5-turbo"}
			Expect(k8sClient.Update(context.Background(), updatedTeamCR)).To(Succeed())

			By("verifying the team was updated in LiteLLM")
			Eventually(func() error {
				return verifyTeamUpdatedInLiteLLM(teamAlias, "50", 500)
			}, testTimeout, testInterval).Should(Succeed())

			By("deleting the team CR")
			Expect(k8sClient.Delete(context.Background(), updatedTeamCR)).To(Succeed())

			By("verifying the team was deleted from LiteLLM")
			Eventually(func() error {
				return verifyTeamDeletedFromLiteLLM(teamAlias)
			}, testTimeout, testInterval).Should(Succeed())
		})

		It("should handle team with blocked status", func() {
			teamAlias := "blocked-team"
			teamCRName := "blocked-team-cr"

			By("creating a team CR with blocked status")
			teamCR := createBlockedTeamCR(teamCRName, teamAlias)
			Expect(k8sClient.Create(context.Background(), teamCR)).To(Succeed())

			By("verifying the team was created with blocked status")
			Eventually(func() error {
				return verifyTeamBlockedStatus(teamAlias, true)
			}, testTimeout, testInterval).Should(Succeed())

			By("unblocking the team")
			updatedTeamCR := &authv1alpha1.Team{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      teamCRName,
				Namespace: modelTestNamespace,
			}, updatedTeamCR)).To(Succeed())

			updatedTeamCR.Spec.Blocked = false
			Expect(k8sClient.Update(context.Background(), updatedTeamCR)).To(Succeed())

			By("verifying the team is no longer blocked")
			Eventually(func() error {
				return verifyTeamBlockedStatus(teamAlias, false)
			}, testTimeout, testInterval).Should(Succeed())

			By("cleaning up team CR")
			Expect(k8sClient.Delete(context.Background(), updatedTeamCR)).To(Succeed())
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
			Eventually(func() error {
				if err := verifyTeamExistsInLiteLLM(team1Alias); err != nil {
					return err
				}
				return verifyTeamExistsInLiteLLM(team2Alias)
			}, testTimeout, testInterval).Should(Succeed())

			By("verifying both team CRs have ready status")
			Eventually(func() error {
				if err := verifyTeamCRStatus(team1CRName, "Ready"); err != nil {
					return err
				}
				return verifyTeamCRStatus(team2CRName, "Ready")
			}, testTimeout, testInterval).Should(Succeed())

			By("cleaning up both team CRs")
			Expect(k8sClient.Delete(context.Background(), team1CR)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), team2CR)).To(Succeed())

			By("verifying both teams were deleted from LiteLLM")
			Eventually(func() error {
				if err := verifyTeamDeletedFromLiteLLM(team1Alias); err != nil {
					return err
				}
				return verifyTeamDeletedFromLiteLLM(team2Alias)
			}, testTimeout, testInterval).Should(Succeed())
		})
	})

	Context("Team Validation", func() {
		It("should validate immutable teamAlias field", func() {
			teamAlias := "immutable-team"
			teamCRName := "immutable-team-cr"

			By("creating a team CR")
			teamCR := createTeamCR(teamCRName, teamAlias)
			Expect(k8sClient.Create(context.Background(), teamCR)).To(Succeed())

			By("trying to update the immutable teamAlias field")
			updatedTeamCR := &authv1alpha1.Team{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      teamCRName,
				Namespace: modelTestNamespace,
			}, updatedTeamCR)).To(Succeed())

			updatedTeamCR.Spec.TeamAlias = "new-team-alias"
			err := k8sClient.Update(context.Background(), updatedTeamCR)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("TeamAlias is immutable"))

			By("cleaning up team CR")
			originalTeamCR := &authv1alpha1.Team{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      teamCRName,
				Namespace: modelTestNamespace,
			}, originalTeamCR)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), originalTeamCR)).To(Succeed())
		})

		It("should handle team with budget duration", func() {
			teamAlias := "budget-duration-team"
			teamCRName := "budget-duration-team-cr"

			By("creating a team CR with budget duration")
			teamCR := createTeamWithBudgetDuration(teamCRName, teamAlias)
			Expect(k8sClient.Create(context.Background(), teamCR)).To(Succeed())

			By("verifying the team has budget duration set")
			Eventually(func() error {
				return verifyTeamBudgetDuration(teamAlias, "1h")
			}, testTimeout, testInterval).Should(Succeed())

			By("cleaning up team CR")
			Expect(k8sClient.Delete(context.Background(), teamCR)).To(Succeed())
		})
	})
})

func createTeamCR(name, teamAlias string) *authv1alpha1.Team {
	return &authv1alpha1.Team{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: modelTestNamespace,
		},
		Spec: authv1alpha1.TeamSpec{
			ConnectionRef: authv1alpha1.ConnectionRef{
				InstanceRef: &authv1alpha1.InstanceRef{
					Namespace: modelTestNamespace,
					Name:      "e2e-test-instance",
				},
			},
			TeamAlias: teamAlias,
			MaxBudget: "25",
			RPMLimit:  250,
			TPMLimit:  2500,
			Models:    []string{"gpt-4o"},
			Blocked:   false,
		},
	}
}

func createBlockedTeamCR(name, teamAlias string) *authv1alpha1.Team {
	teamCR := createTeamCR(name, teamAlias)
	teamCR.Spec.Blocked = true
	return teamCR
}

func createTeamWithBudgetDuration(name, teamAlias string) *authv1alpha1.Team {
	teamCR := createTeamCR(name, teamAlias)
	teamCR.Spec.BudgetDuration = "1h"
	return teamCR
}

func verifyTeamExistsInLiteLLM(teamAlias string) error {
	cmd := exec.Command("kubectl", "get", "team", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.teamAlias=='"+teamAlias+"')].status.teamID}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(output)) == "" {
		return fmt.Errorf("team %s has empty status.teamID", teamAlias)
	}

	return nil
}

func verifyTeamUpdatedInLiteLLM(teamAlias, expectedBudget string, expectedRPM int) error {
	cmd := exec.Command("kubectl", "get", "team", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.teamAlias=='"+teamAlias+"')].spec.maxBudget}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(output)) != expectedBudget {
		return fmt.Errorf("expected budget %s, got %s", expectedBudget, string(output))
	}

	return nil
}

func verifyTeamDeletedFromLiteLLM(teamAlias string) error {
	cmd := exec.Command("kubectl", "get", "team", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.teamAlias=='"+teamAlias+"')].metadata.name}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if string(output) != "" {
		return fmt.Errorf("team %s still exists in Kubernetes", teamAlias)
	}

	return nil
}

func verifyTeamBlockedStatus(teamAlias string, expectedBlocked bool) error {
	cmd := exec.Command("kubectl", "get", "team", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.teamAlias=='"+teamAlias+"')].spec.blocked}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	actualBlocked := strings.TrimSpace(string(output)) == "true"
	if actualBlocked != expectedBlocked {
		return fmt.Errorf("expected blocked status %v, got %v", expectedBlocked, actualBlocked)
	}

	return nil
}

func verifyTeamBudgetDuration(teamAlias, expectedDuration string) error {
	cmd := exec.Command("kubectl", "get", "team", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.teamAlias=='"+teamAlias+"')].spec.budgetDuration}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(output)) != expectedDuration {
		return fmt.Errorf("expected budget duration %s, got %s", expectedDuration, string(output))
	}

	return nil
}

func verifyTeamCRStatus(teamCRName, expectedStatus string) error {
	cmd := exec.Command("kubectl", "get", "team", teamCRName,
		"-n", modelTestNamespace,
		"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	got := strings.TrimSpace(string(output))
	var expectedConditionStatus string
	switch expectedStatus {
	case "Ready":
		expectedConditionStatus = condStatusTrue
	case "Error":
		expectedConditionStatus = condStatusFalse
	default:
		expectedConditionStatus = expectedStatus
	}

	if got != expectedConditionStatus {
		return fmt.Errorf("expected status %s (condition status %s), got %s", expectedStatus, expectedConditionStatus, got)
	}

	return nil
}
