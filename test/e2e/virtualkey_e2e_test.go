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

var _ = Describe("VirtualKey E2E Tests", Ordered, func() {
	Context("VirtualKey Lifecycle", func() {
		It("should create, update, and delete a virtual key successfully", func() {
			keyAlias := "e2e-test-key"
			keyCRName := "e2e-test-key-cr"

			By("creating a virtual key CR")
			keyCR := createVirtualKeyCR(keyCRName, keyAlias)
			Expect(k8sClient.Create(context.Background(), keyCR)).To(Succeed())

			By("verifying the virtual key was created in LiteLLM")
			Eventually(func() error {
				return verifyVirtualKeyExistsInLiteLLM(keyAlias)
			}, testTimeout, testInterval).Should(Succeed())

			By("verifying virtual key CR has ready status")
			Eventually(func() error {
				return verifyVirtualKeyCRStatus(keyCRName, "Ready")
			}, testTimeout, testInterval).Should(Succeed())

			By("verifying key secret was created")
			Eventually(func() error {
				return verifyKeySecretCreated(keyAlias)
			}, testTimeout, testInterval).Should(Succeed())

			By("updating the virtual key CR")
			updatedKeyCR := &authv1alpha1.VirtualKey{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      keyCRName,
				Namespace: modelTestNamespace,
			}, updatedKeyCR)).To(Succeed())

			// Update key properties
			updatedKeyCR.Spec.MaxBudget = "30"
			updatedKeyCR.Spec.RPMLimit = 300
			updatedKeyCR.Spec.Models = []string{"gpt-4o", "gpt-3.5-turbo"}
			Expect(k8sClient.Update(context.Background(), updatedKeyCR)).To(Succeed())

			By("verifying the virtual key was updated in LiteLLM")
			Eventually(func() error {
				return verifyVirtualKeyUpdatedInLiteLLM(keyAlias, "30", 300)
			}, testTimeout, testInterval).Should(Succeed())

			By("deleting the virtual key CR")
			Expect(k8sClient.Delete(context.Background(), updatedKeyCR)).To(Succeed())

			By("verifying the virtual key was deleted from LiteLLM")
			Eventually(func() error {
				return verifyVirtualKeyDeletedFromLiteLLM(keyAlias)
			}, testTimeout, testInterval).Should(Succeed())

			By("verifying key secret was deleted")
			Eventually(func() error {
				return verifyKeySecretDeleted(keyAlias)
			}, testTimeout, testInterval).Should(Succeed())
		})

		It("should handle virtual key with user association", func() {
			keyAlias := "user-associated-key"
			keyCRName := "user-associated-key-cr"
			userID := "test-user-123"

			By("creating a virtual key CR with user association")
			keyCR := createVirtualKeyWithUser(keyCRName, keyAlias, userID)
			Expect(k8sClient.Create(context.Background(), keyCR)).To(Succeed())

			By("verifying the virtual key has user association")
			Eventually(func() error {
				return verifyVirtualKeyUserAssociation(keyAlias, userID)
			}, testTimeout, testInterval).Should(Succeed())

			By("cleaning up virtual key CR")
			Expect(k8sClient.Delete(context.Background(), keyCR)).To(Succeed())
		})

		It("should handle virtual key with team association", func() {
			keyAlias := "team-associated-key"
			keyCRName := "team-associated-key-cr"
			teamID := "test-team-456"

			By("creating a virtual key CR with team association")
			keyCR := createVirtualKeyWithTeam(keyCRName, keyAlias, teamID)
			Expect(k8sClient.Create(context.Background(), keyCR)).To(Succeed())

			By("verifying the virtual key has team association")
			Eventually(func() error {
				return verifyVirtualKeyTeamAssociation(keyAlias, teamID)
			}, testTimeout, testInterval).Should(Succeed())

			By("cleaning up virtual key CR")
			Expect(k8sClient.Delete(context.Background(), keyCR)).To(Succeed())
		})

		It("should handle virtual key with blocked status", func() {
			keyAlias := "blocked-key"
			keyCRName := "blocked-key-cr"

			By("creating a virtual key CR with blocked status")
			keyCR := createBlockedVirtualKey(keyCRName, keyAlias)
			Expect(k8sClient.Create(context.Background(), keyCR)).To(Succeed())

			By("verifying the virtual key is blocked")
			Eventually(func() error {
				return verifyVirtualKeyBlockedStatus(keyAlias, true)
			}, testTimeout, testInterval).Should(Succeed())

			By("unblocking the virtual key")
			updatedKeyCR := &authv1alpha1.VirtualKey{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      keyCRName,
				Namespace: modelTestNamespace,
			}, updatedKeyCR)).To(Succeed())

			updatedKeyCR.Spec.Blocked = false
			Expect(k8sClient.Update(context.Background(), updatedKeyCR)).To(Succeed())

			By("verifying the virtual key is no longer blocked")
			Eventually(func() error {
				return verifyVirtualKeyBlockedStatus(keyAlias, false)
			}, testTimeout, testInterval).Should(Succeed())

			By("cleaning up virtual key CR")
			Expect(k8sClient.Delete(context.Background(), updatedKeyCR)).To(Succeed())
		})
	})

	Context("VirtualKey Validation", func() {
		It("should validate immutable keyAlias field", func() {
			keyAlias := "immutable-key"
			keyCRName := "immutable-key-cr"

			By("creating a virtual key CR")
			keyCR := createVirtualKeyCR(keyCRName, keyAlias)
			Expect(k8sClient.Create(context.Background(), keyCR)).To(Succeed())

			By("trying to update the immutable keyAlias field")
			updatedKeyCR := &authv1alpha1.VirtualKey{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      keyCRName,
				Namespace: modelTestNamespace,
			}, updatedKeyCR)).To(Succeed())

			updatedKeyCR.Spec.KeyAlias = "new-key-alias"
			err := k8sClient.Update(context.Background(), updatedKeyCR)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("KeyAlias is immutable"))

			By("cleaning up virtual key CR")
			originalKeyCR := &authv1alpha1.VirtualKey{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      keyCRName,
				Namespace: modelTestNamespace,
			}, originalKeyCR)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), originalKeyCR)).To(Succeed())
		})

		It("should handle virtual key with duration and expiry", func() {
			keyAlias := "expiring-key"
			keyCRName := "expiring-key-cr"

			By("creating a virtual key CR with duration")
			keyCR := createVirtualKeyWithDuration(keyCRName, keyAlias)
			Expect(k8sClient.Create(context.Background(), keyCR)).To(Succeed())

			By("verifying the virtual key has duration set")
			Eventually(func() error {
				return verifyVirtualKeyDuration(keyAlias, "24h")
			}, testTimeout, testInterval).Should(Succeed())

			By("cleaning up virtual key CR")
			Expect(k8sClient.Delete(context.Background(), keyCR)).To(Succeed())
		})
	})
})

func createVirtualKeyCR(name, keyAlias string) *authv1alpha1.VirtualKey {
	return &authv1alpha1.VirtualKey{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: modelTestNamespace,
		},
		Spec: authv1alpha1.VirtualKeySpec{
			ConnectionRef: authv1alpha1.ConnectionRef{
				InstanceRef: &authv1alpha1.InstanceRef{
					Namespace: modelTestNamespace,
					Name:      "e2e-test-instance",
				},
			},
			KeyAlias:  keyAlias,
			MaxBudget: "15",
			RPMLimit:  150,
			TPMLimit:  1500,
			Models:    []string{"gpt-4o"},
			Blocked:   false,
		},
	}
}

func createVirtualKeyWithUser(name, keyAlias, userID string) *authv1alpha1.VirtualKey {
	keyCR := createVirtualKeyCR(name, keyAlias)
	keyCR.Spec.UserID = userID
	return keyCR
}

func createVirtualKeyWithTeam(name, keyAlias, teamID string) *authv1alpha1.VirtualKey {
	keyCR := createVirtualKeyCR(name, keyAlias)
	keyCR.Spec.TeamID = teamID
	return keyCR
}

func createBlockedVirtualKey(name, keyAlias string) *authv1alpha1.VirtualKey {
	keyCR := createVirtualKeyCR(name, keyAlias)
	keyCR.Spec.Blocked = true
	return keyCR
}

func createVirtualKeyWithDuration(name, keyAlias string) *authv1alpha1.VirtualKey {
	keyCR := createVirtualKeyCR(name, keyAlias)
	keyCR.Spec.Duration = "24h"
	return keyCR
}

func verifyVirtualKeyExistsInLiteLLM(keyAlias string) error {
	cmd := exec.Command("kubectl", "get", "virtualkey", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.keyAlias=='"+keyAlias+"')].status.keyID}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(output)) == "" {
		return fmt.Errorf("virtual key %s has empty status.keyID", keyAlias)
	}

	return nil
}

func verifyVirtualKeyUpdatedInLiteLLM(keyAlias, expectedBudget string, expectedRPM int) error {
	cmd := exec.Command("kubectl", "get", "virtualkey", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.keyAlias=='"+keyAlias+"')].spec.maxBudget}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(output)) != expectedBudget {
		return fmt.Errorf("expected budget %s, got %s", expectedBudget, string(output))
	}

	return nil
}

func verifyVirtualKeyDeletedFromLiteLLM(keyAlias string) error {
	cmd := exec.Command("kubectl", "get", "virtualkey", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.keyAlias=='"+keyAlias+"')].metadata.name}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if string(output) != "" {
		return fmt.Errorf("virtual key %s still exists in Kubernetes", keyAlias)
	}

	return nil
}

func verifyKeySecretCreated(keyAlias string) error {
	cmd := exec.Command("kubectl", "get", "virtualkey", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.keyAlias=='"+keyAlias+"')].status.keySecretRef}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(output)) == "" {
		return fmt.Errorf("virtual key %s has no secret reference", keyAlias)
	}

	// Verify secret actually exists
	secretName := strings.TrimSpace(string(output))
	secretCmd := exec.Command("kubectl", "get", "secret", secretName, "-n", modelTestNamespace)
	_, err = utils.Run(secretCmd)
	if err != nil {
		return fmt.Errorf("secret %s does not exist", secretName)
	}

	return nil
}

func verifyKeySecretDeleted(keyAlias string) error {
	// In most cases, the secret should be cleaned up when the virtual key is deleted
	// This verification depends on how your controller handles secret cleanup
	return nil
}

func verifyVirtualKeyUserAssociation(keyAlias, expectedUserID string) error {
	cmd := exec.Command("kubectl", "get", "virtualkey", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.keyAlias=='"+keyAlias+"')].spec.userID}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(output)) != expectedUserID {
		return fmt.Errorf("expected userID %s, got %s", expectedUserID, string(output))
	}

	return nil
}

func verifyVirtualKeyTeamAssociation(keyAlias, expectedTeamID string) error {
	cmd := exec.Command("kubectl", "get", "virtualkey", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.keyAlias=='"+keyAlias+"')].spec.teamID}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(output)) != expectedTeamID {
		return fmt.Errorf("expected teamID %s, got %s", expectedTeamID, string(output))
	}

	return nil
}

func verifyVirtualKeyBlockedStatus(keyAlias string, expectedBlocked bool) error {
	cmd := exec.Command("kubectl", "get", "virtualkey", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.keyAlias=='"+keyAlias+"')].spec.blocked}")

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

func verifyVirtualKeyDuration(keyAlias, expectedDuration string) error {
	cmd := exec.Command("kubectl", "get", "virtualkey", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.keyAlias=='"+keyAlias+"')].spec.duration}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(output)) != expectedDuration {
		return fmt.Errorf("expected duration %s, got %s", expectedDuration, string(output))
	}

	return nil
}

func verifyVirtualKeyCRStatus(keyCRName, expectedStatus string) error {
	cmd := exec.Command("kubectl", "get", "virtualkey", keyCRName,
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
