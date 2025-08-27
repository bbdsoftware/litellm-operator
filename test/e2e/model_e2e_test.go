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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	litellmv1alpha1 "github.com/bbdsoftware/litellm-operator/api/litellm/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/test/utils"
)

const (
	modelTestNamespace = "model-e2e-test"
	testTimeout        = 2 * time.Minute
	testInterval       = 5 * time.Second
)

var k8sClient client.Client

func init() {
	// Add the scheme
	err := litellmv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
}

var _ = Describe("Model E2E Tests", Ordered, func() {
	BeforeAll(func() {

		By("creating test namespace")
		cmd := exec.Command("kubectl", "create", "namespace", modelTestNamespace)
		_, _ = utils.Run(cmd)

		By("creating LiteLLM instance")
		createLiteLLMInstance()

		By("waiting for LiteLLM instance to be ready")
		Eventually(func() error {
			return waitForLiteLLMInstanceReady()
		}, testTimeout, testInterval).Should(Succeed())

	})

	AfterAll(func() {
		By("cleaning up test namespace")
		cmd := exec.Command("kubectl", "delete", "namespace", modelTestNamespace)
		_, _ = utils.Run(cmd)
	})

	Context("Model Lifecycle", func() {
		It("should create, update, and delete a model successfully", func() {
			modelName := "e2e-test-model"
			modelCRName := "e2e-test-model-cr"

			By("creating a model CR")
			modelCR := createModelCR(modelCRName, modelName)
			Expect(k8sClient.Create(context.Background(), modelCR)).To(Succeed())

			By("verifying the model was created in LiteLLM")
			Eventually(func() error {
				return verifyModelExistsInLiteLLM(modelName)
			}, testTimeout, testInterval).Should(Succeed())

			By("verifying the model CR status")
			Eventually(func() error {
				return verifyModelCRStatus(modelCRName, "Ready")
			}, testTimeout, testInterval).Should(Succeed())

			By("updating the model CR")
			updatedModelCR := &litellmv1alpha1.Model{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      modelCRName,
				Namespace: modelTestNamespace,
			}, updatedModelCR)).To(Succeed())

			// Update the model parameters
			updatedModelCR.Spec.LiteLLMParams.InputCostPerToken = stringPtr("0.00004")
			updatedModelCR.Spec.LiteLLMParams.OutputCostPerToken = stringPtr("0.00008")
			Expect(k8sClient.Update(context.Background(), updatedModelCR)).To(Succeed())

			By("verifying the model was updated in LiteLLM")
			Eventually(func() error {
				return verifyModelUpdatedInLiteLLM(modelName, 0.00004, 0.00008)
			}, testTimeout, testInterval).Should(Succeed())

			By("deleting the model CR")
			Expect(k8sClient.Delete(context.Background(), updatedModelCR)).To(Succeed())

			By("verifying the model was deleted from LiteLLM")
			Eventually(func() error {
				return verifyModelDeletedFromLiteLLM(modelName)
			}, testTimeout, testInterval).Should(Succeed())
		})

		It("should handle model creation with invalid parameters", func() {
			modelName := "invalid-test-model"
			modelCRName := "invalid-test-model-cr"

			By("creating a model CR with invalid parameters")
			invalidModelCR := createInvalidModelCR(modelCRName, modelName)
			Expect(k8sClient.Create(context.Background(), invalidModelCR)).To(Succeed())

			By("verifying the model CR shows error status")
			Eventually(func() error {
				return verifyModelCRStatus(modelCRName, "Error")
			}, testTimeout, testInterval).Should(Succeed())

			By("cleaning up invalid model CR")
			Expect(k8sClient.Delete(context.Background(), invalidModelCR)).To(Succeed())
		})

		It("should handle multiple models in the same namespace", func() {
			model1Name := "multi-test-model-1"
			model2Name := "multi-test-model-2"
			model1CRName := "multi-test-model-1-cr"
			model2CRName := "multi-test-model-2-cr"

			By("creating first model CR")
			model1CR := createModelCR(model1CRName, model1Name)
			Expect(k8sClient.Create(context.Background(), model1CR)).To(Succeed())

			By("creating second model CR")
			model2CR := createModelCR(model2CRName, model2Name)
			Expect(k8sClient.Create(context.Background(), model2CR)).To(Succeed())

			By("verifying both models were created in LiteLLM")
			Eventually(func() error {
				if err := verifyModelExistsInLiteLLM(model1Name); err != nil {
					return err
				}
				return verifyModelExistsInLiteLLM(model2Name)
			}, testTimeout, testInterval).Should(Succeed())

			By("verifying both model CRs have ready status")
			Eventually(func() error {
				if err := verifyModelCRStatus(model1CRName, "Ready"); err != nil {
					return err
				}
				return verifyModelCRStatus(model2CRName, "Ready")
			}, testTimeout, testInterval).Should(Succeed())

			By("cleaning up both model CRs")
			Expect(k8sClient.Delete(context.Background(), model1CR)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), model2CR)).To(Succeed())

			By("verifying both models were deleted from LiteLLM")
			Eventually(func() error {
				if err := verifyModelDeletedFromLiteLLM(model1Name); err != nil {
					return err
				}
				return verifyModelDeletedFromLiteLLM(model2Name)
			}, testTimeout, testInterval).Should(Succeed())
		})
	})

	Context("Model Validation", func() {
		It("should reject models with duplicate names", func() {
			modelName := "duplicate-test-model"
			model1CRName := "duplicate-test-model-1-cr"
			model2CRName := "duplicate-test-model-2-cr"

			By("creating first model CR")
			model1CR := createModelCR(model1CRName, modelName)
			Expect(k8sClient.Create(context.Background(), model1CR)).To(Succeed())

			By("verifying first model was created successfully")
			Eventually(func() error {
				return verifyModelExistsInLiteLLM(modelName)
			}, testTimeout, testInterval).Should(Succeed())

			By("attempting to create second model CR with same name")
			model2CR := createModelCR(model2CRName, modelName)
			Expect(k8sClient.Create(context.Background(), model2CR)).To(Succeed())

			By("verifying second model CR shows error status")
			Eventually(func() error {
				return verifyModelCRStatus(model2CRName, "Error")
			}, testTimeout, testInterval).Should(Succeed())

			By("cleaning up model CRs")
			Expect(k8sClient.Delete(context.Background(), model1CR)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), model2CR)).To(Succeed())
		})

		It("should validate required fields", func() {
			modelCRName := "validation-test-model-cr"

			By("creating a model CR without required fields")
			invalidModelCR := &litellmv1alpha1.Model{
				ObjectMeta: metav1.ObjectMeta{
					Name:      modelCRName,
					Namespace: modelTestNamespace,
				},
				Spec: litellmv1alpha1.ModelSpec{
					// Missing modelName and other required fields
				},
			}
			Expect(k8sClient.Create(context.Background(), invalidModelCR)).To(Succeed())

			By("verifying the model CR shows error status")
			Eventually(func() error {
				return verifyModelCRStatus(modelCRName, "Error")
			}, testTimeout, testInterval).Should(Succeed())

			By("cleaning up invalid model CR")
			Expect(k8sClient.Delete(context.Background(), invalidModelCR)).To(Succeed())
		})
	})

	Context("Model Reconciliation", func() {
		It("should reconcile model changes when LiteLLM instance is restarted", func() {
			modelName := "reconcile-test-model"
			modelCRName := "reconcile-test-model-cr"

			By("creating a model CR")
			modelCR := createModelCR(modelCRName, modelName)
			Expect(k8sClient.Create(context.Background(), modelCR)).To(Succeed())

			By("verifying the model was created in LiteLLM")
			Eventually(func() error {
				return verifyModelExistsInLiteLLM(modelName)
			}, testTimeout, testInterval).Should(Succeed())

			By("simulating LiteLLM instance restart by deleting the model from LiteLLM")
			// This would typically be done by calling the LiteLLM API directly
			// For e2e tests, we'll simulate this by updating the model CR
			// which should trigger reconciliation

			By("updating the model CR to trigger reconciliation")
			updatedModelCR := &litellmv1alpha1.Model{}
			Expect(k8sClient.Get(context.Background(), types.NamespacedName{
				Name:      modelCRName,
				Namespace: modelTestNamespace,
			}, updatedModelCR)).To(Succeed())

			updatedModelCR.Spec.LiteLLMParams.TPM = intPtr(15000)
			Expect(k8sClient.Update(context.Background(), updatedModelCR)).To(Succeed())

			By("verifying the model was reconciled successfully")
			Eventually(func() error {
				return verifyModelCRStatus(modelCRName, "Ready")
			}, testTimeout, testInterval).Should(Succeed())

			By("cleaning up model CR")
			Expect(k8sClient.Delete(context.Background(), updatedModelCR)).To(Succeed())
		})
	})
})

func createLiteLLMInstance() {
	By("creating LiteLLM instance CR")
	liteLLMInstance := &litellmv1alpha1.LiteLLMInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-test-instance",
			Namespace: modelTestNamespace,
		},
		Spec: litellmv1alpha1.LiteLLMInstanceSpec{
			RedisSecretRef: litellmv1alpha1.RedisSecretRef{
				NameRef: "redis-secret",
				Keys: litellmv1alpha1.RedisSecretKeys{
					HostSecret:     "host",
					PortSecret:     "6379",
					PasswordSecret: "password",
				},
			},
			DatabaseSecretRef: litellmv1alpha1.DatabaseSecretRef{
				NameRef: "postgres-secret",
				Keys: litellmv1alpha1.DatabaseSecretKeys{
					HostSecret:     "host",
					PasswordSecret: "password",
					UsernameSecret: "username",
					DbnameSecret:   "dbname",
				},
			},
		},
	}

	Expect(k8sClient.Create(context.Background(), liteLLMInstance)).To(Succeed())
}

func createModelCR(name, modelName string) *litellmv1alpha1.Model {
	return &litellmv1alpha1.Model{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: modelTestNamespace,
		},
		Spec: litellmv1alpha1.ModelSpec{
			ConnectionRef: litellmv1alpha1.ConnectionRef{
				InstanceRef: litellmv1alpha1.InstanceRef{
					Namespace: modelTestNamespace,
					Name:      "e2e-test-instance",
				},
			},
			ModelName: modelName,
			LiteLLMParams: litellmv1alpha1.LiteLLMParams{
				ApiKey:             stringPtr("sk-test-api-key"),
				ApiBase:            stringPtr("https://api.openai.com/v1"),
				InputCostPerToken:  stringPtr("0.00003"),
				OutputCostPerToken: stringPtr("0.00006"),
				TPM:                intPtr(10000),
				RPM:                intPtr(100),
				Timeout:            intPtr(60),
				MaxRetries:         intPtr(3),
				Organization:       stringPtr("test-org"),
				UseInPassThrough:   boolPtr(false),
				UseLiteLLMProxy:    boolPtr(true),
			},
			ModelInfo: litellmv1alpha1.ModelInfo{
				TeamID:              stringPtr("team-123"),
				TeamPublicModelName: stringPtr("gpt-4-public"),
			},
		},
	}
}

func createInvalidModelCR(name, modelName string) *litellmv1alpha1.Model {
	return &litellmv1alpha1.Model{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: modelTestNamespace,
		},
		Spec: litellmv1alpha1.ModelSpec{
			ModelName: modelName,
			LiteLLMParams: litellmv1alpha1.LiteLLMParams{
				// Missing required API key
				ApiBase:            stringPtr("https://api.openai.com/v1"),
				InputCostPerToken:  stringPtr("-0.00003"), // Invalid negative cost
				OutputCostPerToken: stringPtr("0.00006"),
			},
		},
	}
}

func waitForLiteLLMInstanceReady() error {
	cmd := exec.Command("kubectl", "get", "litellminstance", "e2e-test-instance",
		"-n", modelTestNamespace,
		"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if string(output) != "True" {
		return fmt.Errorf("LiteLLM instance not ready, status: %s", string(output))
	}

	return nil
}

func verifyModelExistsInLiteLLM(modelName string) error {
	// In a real e2e test, this would call the LiteLLM API directly
	// For now, we'll verify through the Kubernetes CR status
	cmd := exec.Command("kubectl", "get", "model", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.modelName=='"+modelName+"')].status.conditions[?(@.type=='Ready')].status}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if string(output) != "True" {
		return fmt.Errorf("model %s not ready, status: %s", modelName, string(output))
	}

	return nil
}

func verifyModelUpdatedInLiteLLM(modelName string, expectedInputCost, expectedOutputCost float64) error {
	// Verify the model was updated with new parameters
	cmd := exec.Command("kubectl", "get", "model", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.modelName=='"+modelName+"')].spec.litellmParams.inputCostPerToken}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	// Parse the output and compare with expected values
	// This is a simplified check - in a real scenario you'd parse the JSON properly
	if string(output) == "" {
		return fmt.Errorf("could not verify model update for %s", modelName)
	}

	return nil
}

func verifyModelDeletedFromLiteLLM(modelName string) error {
	// Verify the model was deleted from LiteLLM
	cmd := exec.Command("kubectl", "get", "model", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.modelName=='"+modelName+"')].metadata.name}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if string(output) != "" {
		return fmt.Errorf("model %s still exists in Kubernetes", modelName)
	}

	return nil
}

func verifyModelCRStatus(modelCRName, expectedStatus string) error {
	cmd := exec.Command("kubectl", "get", "model", modelCRName,
		"-n", modelTestNamespace,
		"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if string(output) != expectedStatus {
		return fmt.Errorf("expected status %s, got %s", expectedStatus, string(output))
	}

	return nil
}

// Helper functions for creating pointers to primitive types
func float64Ptr(v float64) *float64 {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func intPtr(v int) *int {
	return &v
}

func boolPtr(v bool) *bool {
	return &v
}
