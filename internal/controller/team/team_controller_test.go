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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/controller/base"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
)

type mockLitellmTeamClient struct {
	teams              map[string]*litellm.TeamResponse
	createError        error
	updateError        error
	deleteError        error
	getTeamIDError     error
	updateNeeded       bool
	createTeamFunc     func(ctx context.Context, req *litellm.TeamRequest) (litellm.TeamResponse, error)
	updateTeamFunc     func(ctx context.Context, req *litellm.TeamRequest) (litellm.TeamResponse, error)
	deleteTeamFunc     func(ctx context.Context, teamID string) error
	getTeamFunc        func(ctx context.Context, teamID string) (litellm.TeamResponse, error)
	getTeamIDFunc      func(ctx context.Context, teamAlias string) (string, error)
	isUpdateNeededFunc func(ctx context.Context, observed *litellm.TeamResponse, desired *litellm.TeamRequest) bool
}

func (m *mockLitellmTeamClient) CreateTeam(ctx context.Context, req *litellm.TeamRequest) (litellm.TeamResponse, error) {
	if m.createError != nil {
		return litellm.TeamResponse{}, m.createError
	}
	if m.createTeamFunc != nil {
		return m.createTeamFunc(ctx, req)
	}

	// Default implementation
	team := &litellm.TeamResponse{
		TeamID:         "team-" + req.TeamAlias,
		TeamAlias:      req.TeamAlias,
		OrganizationID: req.OrganizationID,
		CreatedAt:      "2024-03-20T10:00:00Z",
		UpdatedAt:      "2024-03-20T10:00:00Z",
		MembersWithRole: []litellm.TeamMemberWithRole{
			{
				UserID:    "user1",
				UserEmail: "user1@example.com",
				Role:      "admin",
			},
		},
	}
	m.teams[team.TeamID] = team
	return *team, nil
}

func (m *mockLitellmTeamClient) DeleteTeam(ctx context.Context, teamID string) error {
	if m.deleteError != nil {
		return m.deleteError
	}
	if m.deleteTeamFunc != nil {
		return m.deleteTeamFunc(ctx, teamID)
	}

	delete(m.teams, teamID)
	return nil
}

func (m *mockLitellmTeamClient) GetTeam(ctx context.Context, teamID string) (litellm.TeamResponse, error) {
	if m.getTeamFunc != nil {
		return m.getTeamFunc(ctx, teamID)
	}

	team, exists := m.teams[teamID]
	if !exists {
		return litellm.TeamResponse{}, errors.New("team not found")
	}
	return *team, nil
}

func (m *mockLitellmTeamClient) GetTeamID(ctx context.Context, teamAlias string) (string, error) {
	if m.getTeamIDError != nil {
		return "", m.getTeamIDError
	}
	if m.getTeamIDFunc != nil {
		return m.getTeamIDFunc(ctx, teamAlias)
	}

	// Search for team by alias
	for _, team := range m.teams {
		if team.TeamAlias == teamAlias {
			return team.TeamID, nil
		}
	}
	return "", nil
}

func (m *mockLitellmTeamClient) IsTeamUpdateNeeded(ctx context.Context, observed *litellm.TeamResponse, desired *litellm.TeamRequest) bool {
	if m.isUpdateNeededFunc != nil {
		return m.isUpdateNeededFunc(ctx, observed, desired)
	}
	return m.updateNeeded
}

func (m *mockLitellmTeamClient) UpdateTeam(ctx context.Context, req *litellm.TeamRequest) (litellm.TeamResponse, error) {
	if m.updateError != nil {
		return litellm.TeamResponse{}, m.updateError
	}
	if m.updateTeamFunc != nil {
		return m.updateTeamFunc(ctx, req)
	}

	// Update existing team
	team, exists := m.teams[req.TeamID]
	if !exists {
		return litellm.TeamResponse{}, errors.New("team not found")
	}

	// Update fields
	team.TeamAlias = req.TeamAlias
	team.OrganizationID = req.OrganizationID
	team.UpdatedAt = "2024-03-20T12:00:00Z"

	return *team, nil
}

// Helper function to create test team
func createTestTeam() *authv1alpha1.Team {
	const testTeamName = "test-team"
	return &authv1alpha1.Team{
		ObjectMeta: metav1.ObjectMeta{
			Name:       testTeamName,
			Namespace:  "default",
			Generation: 1,
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
}

// Helper function to assert condition status
func assertCondition(conditions []metav1.Condition, condType string, status metav1.ConditionStatus, reason string) {
	var found *metav1.Condition
	for _, cond := range conditions {
		if cond.Type == condType {
			found = &cond
			break
		}
	}

	// Debug: print all conditions if assertion fails
	if found == nil || found.Status != status || found.Reason != reason {
		GinkgoWriter.Printf("Available conditions:\n")
		for _, cond := range conditions {
			GinkgoWriter.Printf("  Type: %s, Status: %s, Reason: %s, Message: %s\n",
				cond.Type, cond.Status, cond.Reason, cond.Message)
		}
	}

	Expect(found).NotTo(BeNil(), "condition %s should exist", condType)
	Expect(found.Status).To(Equal(status), "condition %s status", condType)
	Expect(found.Reason).To(Equal(reason), "condition %s reason", condType)
}

var _ = Describe("Team Controller", func() {
	const (
		resourceName = "test-team"
		namespace    = "default"
		secretName   = "test-secret"
	)

	var (
		ctx                context.Context
		typeNamespacedName types.NamespacedName
		reconciler         *TeamReconciler
		mockClient         *mockLitellmTeamClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		typeNamespacedName = types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}

		By("creating the test connection secret")
		connectionSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"masterkey": []byte("test-master-key"),
				"url":       []byte("http://test-url"),
			},
		}
		Expect(k8sClient.Create(ctx, connectionSecret)).To(Succeed())

		By("setting up the reconciler with mock client")
		mockClient = &mockLitellmTeamClient{
			teams: make(map[string]*litellm.TeamResponse),
		}
		reconciler = NewTeamReconciler(k8sClient, k8sClient.Scheme())
		reconciler.LitellmClient = mockClient
	})

	AfterEach(func() {
		By("cleaning up the test team")
		team := &authv1alpha1.Team{}
		err := k8sClient.Get(ctx, typeNamespacedName, team)
		if err == nil {
			// Remove finalizers to allow garbage collection
			team.Finalizers = nil
			_ = k8sClient.Update(ctx, team)
			_ = k8sClient.Delete(ctx, team)

			// Wait until it's gone
			Eventually(func() bool {
				t := &authv1alpha1.Team{}
				e := k8sClient.Get(ctx, typeNamespacedName, t)
				return apierrors.IsNotFound(e)
			}, time.Second*5, time.Millisecond*200).Should(BeTrue())
		} else if !apierrors.IsNotFound(err) {
			Fail("unexpected get error: " + err.Error())
		}

		By("cleaning up the test connection secret")
		connectionSecret := &corev1.Secret{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, connectionSecret)
		if err == nil {
			Expect(k8sClient.Delete(ctx, connectionSecret)).To(Succeed())
		}
	})

	Context("Team reconciliation behaviour", func() {
		It("creates team via LiteLLM and correctly updates status", func() {
			By("creating the team CR without status")
			team := createTestTeam()
			Expect(k8sClient.Create(ctx, team)).To(Succeed())

			By("reconciling the created resource")
			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the team status was updated")
			updatedTeam := &authv1alpha1.Team{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTeam)
			Expect(err).NotTo(HaveOccurred())

			Expect(updatedTeam.Status.TeamID).To(Equal("team-test-alias"))
			Expect(updatedTeam.Status.TeamAlias).To(Equal("test-alias"))
			Expect(updatedTeam.Status.OrganizationID).To(Equal("test-org"))
			Expect(updatedTeam.Status.ObservedGeneration).To(Equal(int64(1)))

			By("verifying the conditions were set correctly")
			assertCondition(updatedTeam.Status.Conditions, base.CondReady, metav1.ConditionTrue, base.ReasonReady)
			assertCondition(updatedTeam.Status.Conditions, base.CondProgressing, metav1.ConditionFalse, base.ReasonReady)
		})

		It("handles team alias conflicts correctly", func() {
			By("setting up mock to return existing team with different ID")
			existingTeam := &litellm.TeamResponse{
				TeamID:    "different-team-id",
				TeamAlias: "test-alias",
			}
			mockClient.teams["different-team-id"] = existingTeam

			By("creating the team CR")
			team := createTestTeam()
			Expect(k8sClient.Create(ctx, team)).To(Succeed())

			By("updating team status to simulate existing team")
			// Get the created team and update its status
			createdTeam := &authv1alpha1.Team{}
			err := k8sClient.Get(ctx, typeNamespacedName, createdTeam)
			Expect(err).NotTo(HaveOccurred())
			createdTeam.Status.TeamID = "my-team-id" // Different from existing
			Expect(k8sClient.Status().Update(ctx, createdTeam)).To(Succeed())

			By("reconciling should detect conflict")
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying error condition was set")
			updatedTeam := &authv1alpha1.Team{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTeam)
			Expect(err).NotTo(HaveOccurred())

			assertCondition(updatedTeam.Status.Conditions, base.CondDegraded, metav1.ConditionTrue, base.ReasonConfigError)
		})

		It("correctly handles drift detection and repair", func() {
			By("setting up mock with existing team")
			existingTeam := &litellm.TeamResponse{
				TeamID:         "existing-team-id",
				TeamAlias:      "test-alias",
				OrganizationID: "old-org",
			}
			mockClient.teams["existing-team-id"] = existingTeam
			mockClient.updateNeeded = true

			By("creating the team CR")
			team := createTestTeam()
			Expect(k8sClient.Create(ctx, team)).To(Succeed())

			By("updating team status to simulate existing team")
			createdTeam := &authv1alpha1.Team{}
			err := k8sClient.Get(ctx, typeNamespacedName, createdTeam)
			Expect(err).NotTo(HaveOccurred())
			createdTeam.Status.TeamID = "existing-team-id"
			Expect(k8sClient.Status().Update(ctx, createdTeam)).To(Succeed())

			By("reconciling should detect and repair drift")
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the team was updated with correct values")
			updatedTeam := &authv1alpha1.Team{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTeam)
			Expect(err).NotTo(HaveOccurred())

			Expect(updatedTeam.Status.OrganizationID).To(Equal("test-org")) // Should be updated
			assertCondition(updatedTeam.Status.Conditions, base.CondReady, metav1.ConditionTrue, base.ReasonReady)
		})
	})

	Context("Error handling behaviour", func() {
		It("handles LiteLLM connection errors gracefully", func() {
			By("setting up mock to return connection error")
			mockClient.createError = errors.New("connection refused")

			By("creating the team CR")
			team := createTestTeam()
			Expect(k8sClient.Create(ctx, team)).To(Succeed())

			By("reconciling should handle error gracefully")
			result, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))

			By("verifying degraded condition was set")
			updatedTeam := &authv1alpha1.Team{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTeam)
			Expect(err).NotTo(HaveOccurred())

			assertCondition(updatedTeam.Status.Conditions, base.CondDegraded, metav1.ConditionTrue, base.ReasonLitellmError)
		})
	})

	Context("Finalizer lifecycle behaviour", func() {
		It("properly handles team deletion", func() {
			By("creating and reconciling team first")
			team := createTestTeam()
			Expect(k8sClient.Create(ctx, team)).To(Succeed())

			_, err := reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("deleting the team")
			updatedTeam := &authv1alpha1.Team{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedTeam)
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Delete(ctx, updatedTeam)).To(Succeed())

			By("reconciling deletion should remove external resource")
			_, err = reconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("verifying external team was deleted")
			_, exists := mockClient.teams["team-test-alias"]
			Expect(exists).To(BeFalse())
		})
	})
})
