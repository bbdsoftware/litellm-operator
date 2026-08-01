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
	"time"

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

var _ = Describe("VirtualKey E2E Tests", Ordered, func() {
	BeforeAll(func() {
		// Initialize k8sClient (reinitialize to ensure auth scheme is registered)
		cfg := config.GetConfigOrDie()
		var err error
		k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
		Expect(err).NotTo(HaveOccurred())
	})

	Context("VirtualKey Lifecycle", func() {
		It("should create, update, and delete a virtual key successfully", func() {
			keyAlias := "e2e-test-key"
			keyCRName := "e2e-test-key-cr"

			By("creating a virtual key CR")
			keyCR := createVirtualKeyCR(keyCRName, keyAlias)
			Expect(k8sClient.Create(context.Background(), keyCR)).To(Succeed())

			By("verifying the virtual key was created in LiteLLM")
			eventuallyVerify(func() error {
				return verifyVirtualKeyExistsInLiteLLM(keyCRName)
			})

			By("verifying virtual key CR has ready status")
			eventuallyVerify(func() error {
				return verifyVirtualKeyCRStatus(keyCRName, "Ready")
			})

			By("verifying key secret was created")
			eventuallyVerify(func() error {
				return verifyKeySecretCreated(keyCRName)
			})

			By("updating the virtual key CR")
			updatedKeyCR, err := getVirtualKeyCR(keyCRName)
			Expect(err).NotTo(HaveOccurred())

			// grab the secret reference for cleanup check later
			secretRef := updatedKeyCR.Status.KeySecretRef

			// Update key properties
			newBudget := "30"
			newRPM := 300
			updatedKeyCR.Spec.MaxBudget = newBudget
			updatedKeyCR.Spec.RPMLimit = newRPM
			Expect(k8sClient.Update(context.Background(), updatedKeyCR)).To(Succeed())

			By("verifying the virtual key was updated in LiteLLM")
			eventuallyVerify(func() error {
				return verifyVirtualKeyUpdatedInLiteLLM(keyCRName, newBudget, newRPM)
			})

			By("deleting the virtual key CR")
			Expect(k8sClient.Delete(context.Background(), updatedKeyCR)).To(Succeed())

			By("verifying the virtual key was deleted from LiteLLM")
			eventuallyVerify(func() error {
				return verifyVirtualKeyDeletedFromLiteLLM(keyCRName)
			})

			By("verifying key secret was deleted")
			eventuallyVerify(func() error {
				return verifySecretDeleted(secretRef)
			})
		})

		It("should handle virtual key with user association", func() {
			keyAlias := "user-associated-key"
			keyCRName := "user-associated-key-cr"
			userID := "test-user-123"

			By("creating a virtual key CR with user association")
			keyCR := createVirtualKeyWithUser(keyCRName, keyAlias, userID)
			Expect(k8sClient.Create(context.Background(), keyCR)).To(Succeed())

			By("verifying the virtual key has user association")
			eventuallyVerify(func() error {
				return verifyVirtualKeyUserAssociation(keyCRName, userID)
			})

			By("cleaning up virtual key CR")
			Expect(deleteVirtualKeyCR(keyCRName)).To(Succeed())
		})

		It("should handle virtual key with team association", func() {
			keyAlias := "team-associated-key"
			keyCRName := "team-associated-key-cr"
			teamID := "test-team-456"

			By("creating a virtual key CR with team association")
			keyCR := createVirtualKeyWithTeam(keyCRName, keyAlias, teamID)
			Expect(k8sClient.Create(context.Background(), keyCR)).To(Succeed())

			By("verifying the virtual key has team association")
			eventuallyVerify(func() error {
				return verifyVirtualKeyTeamAssociation(keyCRName, teamID)
			})

			By("cleaning up virtual key CR")
			Expect(deleteVirtualKeyCR(keyCRName)).To(Succeed())
		})

		It("should handle blocking and unblocking a virtual key", func() {
			keyAlias := "blocked-key"
			keyCRName := "blocked-key-cr"

			By("creating a virtual key CR")
			keyCR := createVirtualKeyCR(keyCRName, keyAlias)
			Expect(k8sClient.Create(context.Background(), keyCR)).To(Succeed())

			By("verifying virtual key CR has ready status")
			eventuallyVerify(func() error {
				return verifyVirtualKeyCRStatus(keyCRName, "Ready")
			})

			By("blocking the virtual key")
			updatedKeyCR, err := getVirtualKeyCR(keyCRName)
			Expect(err).NotTo(HaveOccurred())
			updatedKeyCR.Spec.Blocked = true
			Expect(k8sClient.Update(context.Background(), updatedKeyCR)).To(Succeed())

			By("verifying the virtual key is blocked")
			eventuallyVerify(func() error {
				return verifyVirtualKeyBlockedStatus(keyCRName, true)
			})

			By("unblocking the virtual key")
			updatedKeyCR, err = getVirtualKeyCR(keyCRName)
			Expect(err).NotTo(HaveOccurred())
			updatedKeyCR.Spec.Blocked = false
			Expect(k8sClient.Update(context.Background(), updatedKeyCR)).To(Succeed())

			By("verifying the virtual key is unblocked")
			eventuallyVerify(func() error {
				return verifyVirtualKeyBlockedStatus(keyCRName, false)
			})

			By("cleaning up virtual key CR")
			Expect(deleteVirtualKeyCR(keyCRName)).To(Succeed())
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
			updatedKeyCR, err := getVirtualKeyCR(keyCRName)
			Expect(err).NotTo(HaveOccurred())

			updatedKeyCR.Spec.KeyAlias = "new-key-alias"
			err = k8sClient.Update(context.Background(), updatedKeyCR)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("KeyAlias is immutable"))

			By("cleaning up virtual key CR")
			Expect(deleteVirtualKeyCR(keyCRName)).To(Succeed())
		})

		It("should handle virtual key with duration and expiry", func() {
			keyAlias := "expiring-key"
			keyCRName := "expiring-key-cr"
			duration := "24h"

			By("creating a virtual key CR with duration")
			keyCR := createVirtualKeyWithDuration(keyCRName, keyAlias, duration)
			Expect(k8sClient.Create(context.Background(), keyCR)).To(Succeed())

			expectedExpires, err := parseDurationAndCalculateExpiry(duration)
			Expect(err).NotTo(HaveOccurred())

			By("verifying the virtual key has expiry with the expected duration")
			eventuallyVerify(func() error {
				return verifyVirtualKeyExpiry(keyCRName, duration, expectedExpires)
			})

			By("cleaning up virtual key CR")
			Expect(deleteVirtualKeyCR(keyCRName)).To(Succeed())
		})
	})
})

// ============================================================================
// Creation Helpers
// ============================================================================

func createVirtualKeyCR(name, keyAlias string) *authv1alpha1.VirtualKey {
	return &authv1alpha1.VirtualKey{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e2eTestNamespace,
		},
		Spec: authv1alpha1.VirtualKeySpec{
			ConnectionRef: authv1alpha1.ConnectionRef{
				InstanceRef: &authv1alpha1.InstanceRef{
					Namespace: e2eTestNamespace,
					Name:      e2eTestInstance,
				},
			},
			KeyAlias:  keyAlias,
			MaxBudget: "15",
			RPMLimit:  150,
			TPMLimit:  1500,
			Models:    []string{e2eTestModelGPT4o},
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

func createVirtualKeyWithDuration(name, keyAlias, duration string) *authv1alpha1.VirtualKey {
	keyCR := createVirtualKeyCR(name, keyAlias)
	keyCR.Spec.Duration = duration
	return keyCR
}

// ============================================================================
// Retrieval Helpers
// ============================================================================

func getVirtualKeyCR(keyCRName string) (*authv1alpha1.VirtualKey, error) {
	virtualKeyCR := &authv1alpha1.VirtualKey{}
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      keyCRName,
		Namespace: e2eTestNamespace,
	}, virtualKeyCR)
	if err != nil {
		return nil, fmt.Errorf("failed to get virtual key %s: %w", keyCRName, err)
	}
	return virtualKeyCR, nil
}

// ============================================================================
// Verification Helpers - LiteLLM
// ============================================================================

func verifyVirtualKeyExistsInLiteLLM(keyCRName string) error {
	virtualKeyCR, err := getVirtualKeyCR(keyCRName)
	if err != nil {
		return err
	}

	if virtualKeyCR.Status.KeyID == "" {
		return fmt.Errorf("virtual key %s has empty status.keyID", keyCRName)
	}

	return nil
}

func verifyVirtualKeyUpdatedInLiteLLM(keyCRName, expectedBudget string, expectedRPM int) error {
	virtualKeyCR, err := getVirtualKeyCR(keyCRName)
	if err != nil {
		return err
	}

	// Verify budget
	if err := compareBudgetStrings(expectedBudget, virtualKeyCR.Status.MaxBudget); err != nil {
		return err
	}

	// Verify RPM limit
	if virtualKeyCR.Status.RPMLimit != expectedRPM {
		return fmt.Errorf("expected RPM limit %d, got %d", expectedRPM, virtualKeyCR.Status.RPMLimit)
	}

	return nil
}

func verifyVirtualKeyDeletedFromLiteLLM(keyCRName string) error {
	virtualKeyCR := &authv1alpha1.VirtualKey{}
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      keyCRName,
		Namespace: e2eTestNamespace,
	}, virtualKeyCR)

	if err != nil {
		// If the CR is not found, that's acceptable - it means it was deleted
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get virtual key %s: %w", keyCRName, err)
	}

	if virtualKeyCR.GetDeletionTimestamp().IsZero() {
		return fmt.Errorf("virtual key %s is not marked for deletion", keyCRName)
	}

	return fmt.Errorf("virtual key %s is not deleted", keyCRName)
}

// ============================================================================
// Verification Helpers - Status
// ============================================================================

func verifyVirtualKeyCRStatus(keyCRName, expectedStatus string) error {
	virtualKeyCR, err := getVirtualKeyCR(keyCRName)
	if err != nil {
		return err
	}

	return verifyReady(virtualKeyCR.GetConditions(), expectedStatus)
}

// verifyVirtualKeyStatusString is a generic helper to verify string status fields
func verifyVirtualKeyStatusString(keyCRName, fieldName, expectedValue string) error {
	virtualKeyCR, err := getVirtualKeyCR(keyCRName)
	if err != nil {
		return err
	}

	var actualValue string
	switch fieldName {
	case "UserID":
		actualValue = virtualKeyCR.Status.UserID
	case "TeamID":
		actualValue = virtualKeyCR.Status.TeamID
	default:
		return fmt.Errorf("unknown field name: %s", fieldName)
	}

	if actualValue != expectedValue {
		return fmt.Errorf("expected %s %s, got %s", fieldName, expectedValue, actualValue)
	}

	return nil
}

func verifyVirtualKeyUserAssociation(keyCRName, expectedUserID string) error {
	return verifyVirtualKeyStatusString(keyCRName, "UserID", expectedUserID)
}

func verifyVirtualKeyTeamAssociation(keyCRName, expectedTeamID string) error {
	return verifyVirtualKeyStatusString(keyCRName, "TeamID", expectedTeamID)
}

func verifyVirtualKeyExpiry(keyCRName, duration string, expectedExpires time.Time) error {
	virtualKeyCR, err := getVirtualKeyCR(keyCRName)
	if err != nil {
		return err
	}

	// when a duration is set, the response will contain a datestring in the `expires` field
	if virtualKeyCR.Status.Expires == "" {
		return fmt.Errorf("expected an expiry but got none")
	}

	// parse the expires field and convert it to a time.Time
	expires, err := time.Parse(time.RFC3339, virtualKeyCR.Status.Expires)
	if err != nil {
		return fmt.Errorf("failed to parse expires field: %w", err)
	}

	// compare with tolerance (allow up to 10 seconds difference for processing time and clock skew)
	tolerance := 10 * time.Second
	diff := expires.Sub(expectedExpires)
	if diff < 0 {
		diff = -diff
	}
	if diff > tolerance {
		return fmt.Errorf("expires time %v is not approximately %v ahead of now (expected: %v, got: %v, difference: %v, tolerance: %v)",
			expires, duration, expectedExpires, expires, diff, tolerance)
	}

	return nil
}

func verifyVirtualKeyBlockedStatus(keyCRName string, expectedBlocked bool) error {
	virtualKeyCR, err := getVirtualKeyCR(keyCRName)
	if err != nil {
		return err
	}
	if virtualKeyCR.Status.Blocked != expectedBlocked {
		return fmt.Errorf("expected blocked status %t, got %t", expectedBlocked, virtualKeyCR.Status.Blocked)
	}
	return nil
}

// ============================================================================
// Utility Helpers
// ============================================================================

// verifyKeySecretCreated verifies that a k8s secret has been created for a virtual key
func verifyKeySecretCreated(keyCRName string) error {
	virtualKeyCR, err := getVirtualKeyCR(keyCRName)
	if err != nil {
		return err
	}

	if virtualKeyCR.Status.KeySecretRef == "" {
		return fmt.Errorf("virtual key %s has no secret reference", keyCRName)
	}

	// Verify secret actually exists
	return getSecret(virtualKeyCR.Status.KeySecretRef)
}

// deleteVirtualKeyCR deletes a virtual key CR by name
func deleteVirtualKeyCR(keyCRName string) error {
	keyCR, err := getVirtualKeyCR(keyCRName)
	if err != nil {
		return err
	}
	return k8sClient.Delete(context.Background(), keyCR)
}
