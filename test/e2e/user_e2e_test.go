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

var _ = Describe("User E2E Tests", Ordered, func() {
	Context("User Lifecycle", func() {
		It("should create, update, and delete a user successfully", func() {
			userEmail := "e2e-test@example.com"
			userCRName := "e2e-test-user"

			By("creating a user CR")
			userCR := createUserCR(userCRName, userEmail)
			Expect(k8sClient.Create(context.Background(), userCR)).To(Succeed())

			By("verifying the user was created in LiteLLM")
			Eventually(func() error {
				return verifyUserExistsInLiteLLM(userEmail)
			}, testTimeout, testInterval).Should(Succeed())

			By("verifying user CR has ready status")
			Eventually(func() error {
				return verifyUserCRStatus(userCRName, "Ready")
			}, testTimeout, testInterval).Should(Succeed())

			By("updating the user CR")
			updatedUserCR := &authv1alpha1.User{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      userCRName,
				Namespace: modelTestNamespace,
			}, updatedUserCR)).To(Succeed())

			// Update user properties
			updatedUserCR.Spec.MaxBudget = "20"
			updatedUserCR.Spec.RPMLimit = 200
			Expect(k8sClient.Update(context.Background(), updatedUserCR)).To(Succeed())

			By("verifying the user was updated in LiteLLM")
			Eventually(func() error {
				return verifyUserUpdatedInLiteLLM(userEmail, "20", 200)
			}, testTimeout, testInterval).Should(Succeed())

			By("deleting the user CR")
			Expect(k8sClient.Delete(context.Background(), updatedUserCR)).To(Succeed())

			By("verifying the user was deleted from LiteLLM")
			Eventually(func() error {
				return verifyUserDeletedFromLiteLLM(userEmail)
			}, testTimeout, testInterval).Should(Succeed())
		})

		It("should handle user creation with invalid email", func() {
			invalidEmail := "invalid-email"
			userCRName := "invalid-email-user"

			By("creating a user CR with invalid email")
			invalidUserCR := createInvalidUserCR(userCRName, invalidEmail)
			Expect(k8sClient.Create(context.Background(), invalidUserCR)).To(Succeed())

			By("verifying the user CR shows error status")
			Eventually(func() error {
				return verifyUserCRStatusError(userCRName, "Error", "invalid email format")
			}, testTimeout, testInterval).Should(Succeed())

			By("cleaning up invalid user CR")
			Expect(k8sClient.Delete(context.Background(), invalidUserCR)).To(Succeed())
		})

		It("should handle user with auto-create key", func() {
			userEmail := "auto-key@example.com"
			userCRName := "auto-key-user"

			By("creating a user CR with auto-create key enabled")
			userCR := createUserWithAutoKey(userCRName, userEmail)
			Expect(k8sClient.Create(context.Background(), userCR)).To(Succeed())

			By("verifying the user was created with a key")
			Eventually(func() error {
				return verifyUserKeyCreated(userEmail)
			}, testTimeout, testInterval).Should(Succeed())

			By("cleaning up user CR")
			Expect(k8sClient.Delete(context.Background(), userCR)).To(Succeed())
		})
	})

	Context("User Validation", func() {
		It("should validate immutable userEmail field", func() {
			userEmail := "immutable@example.com"
			userCRName := "immutable-user"

			By("creating a user CR")
			userCR := createUserCR(userCRName, userEmail)
			Expect(k8sClient.Create(context.Background(), userCR)).To(Succeed())

			By("trying to update the immutable userEmail field")
			updatedUserCR := &authv1alpha1.User{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      userCRName,
				Namespace: modelTestNamespace,
			}, updatedUserCR)).To(Succeed())

			updatedUserCR.Spec.UserEmail = "new-email@example.com"
			err := k8sClient.Update(context.Background(), updatedUserCR)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("UserEmail is immutable"))

			By("cleaning up user CR")
			// Get the original CR again since update failed
			originalUserCR := &authv1alpha1.User{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      userCRName,
				Namespace: modelTestNamespace,
			}, originalUserCR)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), originalUserCR)).To(Succeed())
		})

		It("should validate user role enum", func() {
			userEmail := "role-test@example.com"
			userCRName := "role-test-user"

			By("creating a user CR with invalid role")
			invalidUserCR := createUserCR(userCRName, userEmail)
			invalidUserCR.Spec.UserRole = "invalid_role"
			err := k8sClient.Create(context.Background(), invalidUserCR)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("spec.userRole"))
		})
	})
})

func createUserCR(name, email string) *authv1alpha1.User {
	return &authv1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: modelTestNamespace,
		},
		Spec: authv1alpha1.UserSpec{
			ConnectionRef: authv1alpha1.ConnectionRef{
				InstanceRef: &authv1alpha1.InstanceRef{
					Namespace: modelTestNamespace,
					Name:      "e2e-test-instance",
				},
			},
			UserEmail:     email,
			UserAlias:     name,
			UserRole:      "internal_user",
			MaxBudget:     "10",
			RPMLimit:      100,
			TPMLimit:      1000,
			Models:        []string{"gpt-4o"},
			AutoCreateKey: false,
		},
	}
}

func createInvalidUserCR(name, email string) *authv1alpha1.User {
	return &authv1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: modelTestNamespace,
		},
		Spec: authv1alpha1.UserSpec{
			ConnectionRef: authv1alpha1.ConnectionRef{
				InstanceRef: &authv1alpha1.InstanceRef{
					Namespace: modelTestNamespace,
					Name:      "e2e-test-instance",
				},
			},
			UserEmail: email, // Invalid email format
			UserAlias: name,
			UserRole:  "internal_user",
		},
	}
}

func createUserWithAutoKey(name, email string) *authv1alpha1.User {
	userCR := createUserCR(name, email)
	userCR.Spec.AutoCreateKey = true
	userCR.Spec.KeyAlias = name + "-key"
	return userCR
}

func verifyUserExistsInLiteLLM(userEmail string) error {
	cmd := exec.Command("kubectl", "get", "user", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.userEmail=='"+userEmail+"')].status.userID}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(output)) == "" {
		return fmt.Errorf("user %s has empty status.userID", userEmail)
	}

	return nil
}

func verifyUserUpdatedInLiteLLM(userEmail, expectedBudget string, expectedRPM int) error {
	cmd := exec.Command("kubectl", "get", "user", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.userEmail=='"+userEmail+"')].spec.maxBudget}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(output)) != expectedBudget {
		return fmt.Errorf("expected budget %s, got %s", expectedBudget, string(output))
	}

	return nil
}

func verifyUserDeletedFromLiteLLM(userEmail string) error {
	cmd := exec.Command("kubectl", "get", "user", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.userEmail=='"+userEmail+"')].metadata.name}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if string(output) != "" {
		return fmt.Errorf("user %s still exists in Kubernetes", userEmail)
	}

	return nil
}

func verifyUserKeyCreated(userEmail string) error {
	cmd := exec.Command("kubectl", "get", "user", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.userEmail=='"+userEmail+"')].status.keySecretRef}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(output)) == "" {
		return fmt.Errorf("user %s has no key secret reference", userEmail)
	}

	return nil
}

func verifyUserCRStatus(userCRName, expectedStatus string) error {
	cmd := exec.Command("kubectl", "get", "user", userCRName,
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

func verifyUserCRStatusError(userCRName, expectedStatus string, errorMsg string) error {
	var expectedConditionStatus string
	switch expectedStatus {
	case "Ready":
		expectedConditionStatus = condStatusTrue
	case "Error":
		expectedConditionStatus = condStatusFalse
	default:
		expectedConditionStatus = expectedStatus
	}

	cmdStatus := exec.Command("kubectl", "get", "user", userCRName,
		"-n", modelTestNamespace,
		"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")
	outStatus, err := utils.Run(cmdStatus)
	if err != nil {
		return err
	}

	got := strings.TrimSpace(string(outStatus))
	if got != expectedConditionStatus {
		return fmt.Errorf("expected status %s (condition status %s), got %s", expectedStatus, expectedConditionStatus, got)
	}

	cmdMsg := exec.Command("kubectl", "get", "user", userCRName,
		"-n", modelTestNamespace,
		"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].message}")
	outMsg, err := utils.Run(cmdMsg)
	if err != nil {
		return err
	}

	if !strings.Contains(string(outMsg), errorMsg) {
		return fmt.Errorf("expected error message '%s' not found in status message: %s", errorMsg, string(outMsg))
	}

	return nil
}
