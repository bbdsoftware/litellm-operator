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

var _ = Describe("User E2E Tests", Ordered, func() {
	BeforeAll(func() {
		// Initialize k8sClient (reinitialize to ensure auth scheme is registered)
		cfg := config.GetConfigOrDie()
		var err error
		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred())
	})

	Context("User Lifecycle", func() {
		It("should create, update, and delete a user successfully", func() {
			userEmail := "e2e-test@example.com"
			userCRName := "e2e-test-user"

			By("creating a user CR")
			userCR := createUserCR(userCRName, userEmail)
			Expect(k8sClient.Create(context.Background(), userCR)).To(Succeed())

			By("verifying the user was created in LiteLLM")
			eventuallyVerify(func() error {
				return verifyUserExistsInLiteLLM(userCRName)
			})

			By("verifying user CR has ready status")
			eventuallyVerify(func() error {
				return verifyUserCRStatus(userCRName, statusReady)
			})

			By("updating the user CR")
			updatedUserCR, err := getUserCR(userCRName)
			Expect(err).NotTo(HaveOccurred())

			// Update user properties
			newBudget := "20"
			newRPM := 200
			updatedUserCR.Spec.MaxBudget = newBudget
			updatedUserCR.Spec.RPMLimit = newRPM
			Expect(k8sClient.Update(context.Background(), updatedUserCR)).To(Succeed())

			By("verifying the user was updated in LiteLLM")
			eventuallyVerify(func() error {
				return verifyUserUpdatedInLiteLLM(userCRName, newBudget, newRPM)
			})

			By("deleting the user CR")
			Expect(k8sClient.Delete(context.Background(), updatedUserCR)).To(Succeed())

			By("verifying the user was deleted from LiteLLM")
			eventuallyVerify(func() error {
				return verifyUserDeletedFromLiteLLM(userCRName)
			})
		})

		It("should handle user with auto-create key", func() {
			userEmail := "auto-key@example.com"
			userCRName := "auto-key-user"

			By("creating a user CR with auto-create key enabled")
			userCR := createUserWithAutoKey(userCRName, userEmail)
			Expect(k8sClient.Create(context.Background(), userCR)).To(Succeed())

			By("verifying the user was created with a key")
			eventuallyVerify(func() error {
				return verifyUserKeyCreated(userCRName)
			})

			By("cleaning up user CR")
			Expect(deleteUserCR(userCRName)).To(Succeed())
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
			updatedUserCR, err := getUserCR(userCRName)
			Expect(err).NotTo(HaveOccurred())

			updatedUserCR.Spec.UserEmail = "new-email@example.com"
			err = k8sClient.Update(context.Background(), updatedUserCR)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("UserEmail is immutable"))

			By("cleaning up user CR")
			Expect(deleteUserCR(userCRName)).To(Succeed())
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
			// no need to clean up user CR as it was not created
		})
	})
})

func createUserCR(name, email string) *authv1alpha1.User {
	return &authv1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e2eTestNamespace,
		},
		Spec: authv1alpha1.UserSpec{
			ConnectionRef: authv1alpha1.ConnectionRef{
				InstanceRef: &authv1alpha1.InstanceRef{
					Namespace: e2eTestNamespace,
					Name:      e2eTestInstance,
				},
			},
			UserEmail:     email,
			UserAlias:     name,
			UserRole:      "internal_user",
			MaxBudget:     "10",
			RPMLimit:      100,
			TPMLimit:      1000,
			Models:        []string{e2eTestModelGPT4o},
			AutoCreateKey: false,
		},
	}
}

func createUserWithAutoKey(name, email string) *authv1alpha1.User {
	userCR := createUserCR(name, email)
	userCR.Spec.AutoCreateKey = true
	userCR.Spec.KeyAlias = name + "-key"
	return userCR
}

func verifyUserUpdatedInLiteLLM(userCRName, expectedBudget string, expectedRPM int) error {
	userCR, err := getUserCR(userCRName)
	if err != nil {
		return err
	}

	// Verify budget
	if err := compareBudgetStrings(expectedBudget, userCR.Status.MaxBudget); err != nil {
		return err
	}

	// Verify RPM limit
	if userCR.Status.RPMLimit != expectedRPM {
		return fmt.Errorf("expected RPM limit %d, got %d", expectedRPM, userCR.Status.RPMLimit)
	}

	return nil
}

// ============================================================================
// Retrieval Helpers
// ============================================================================

func getUserCR(userCRName string) (*authv1alpha1.User, error) {
	userCR := &authv1alpha1.User{}
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      userCRName,
		Namespace: e2eTestNamespace,
	}, userCR)
	if err != nil {
		return nil, fmt.Errorf("failed to get user %s: %w", userCRName, err)
	}
	return userCR, nil
}

// ============================================================================
// Verification Helpers - LiteLLM
// ============================================================================

func verifyUserExistsInLiteLLM(userCRName string) error {
	userCR, err := getUserCR(userCRName)
	if err != nil {
		return err
	}

	if userCR.Status.UserID == "" {
		return fmt.Errorf("user %s has empty status.userID", userCRName)
	}

	return nil
}

func verifyUserDeletedFromLiteLLM(userCRName string) error {
	userCR := &authv1alpha1.User{}
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      userCRName,
		Namespace: e2eTestNamespace,
	}, userCR)

	if err != nil {
		// If the CR is not found, that's acceptable - it means it was deleted
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get user %s: %w", userCRName, err)
	}

	if userCR.GetDeletionTimestamp().IsZero() {
		return fmt.Errorf("user %s is not marked for deletion", userCRName)
	}

	return fmt.Errorf("user %s is not deleted", userCRName)
}

func verifyUserKeyCreated(userCRName string) error {
	userCR, err := getUserCR(userCRName)
	if err != nil {
		return err
	}

	if userCR.Status.KeySecretRef == "" {
		return fmt.Errorf("user %s has no key secret reference", userCRName)
	}

	return nil
}

// ============================================================================
// Verification Helpers - Status
// ============================================================================

func verifyUserCRStatus(userCRName, expectedStatus string) error {
	userCR, err := getUserCR(userCRName)
	if err != nil {
		return err
	}

	return verifyReady(userCR.GetConditions(), expectedStatus)
}

// ============================================================================
// Utility Helpers
// ============================================================================

// deleteUserCR deletes a user CR by name
func deleteUserCR(userCRName string) error {
	userCR, err := getUserCR(userCRName)
	if err != nil {
		return err
	}
	return k8sClient.Delete(context.Background(), userCR)
}
