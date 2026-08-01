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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	litellmv1alpha1 "github.com/bbdsoftware/litellm-operator/api/litellm/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/test/utils"
)

func init() {
	// Add the scheme
	err := litellmv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		panic(err)
	}
}

var _ = Describe("Model E2E Tests", Ordered, func() {
	BeforeAll(func() {
		// Initialize k8sClient if not already initialized
		if k8sClient == nil {
			cfg := config.GetConfigOrDie()
			var err error
			k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
			Expect(err).NotTo(HaveOccurred())
		}

		// Verify LiteLLM instance is ready (it should already be set up in BeforeSuite)
		By("verifying LiteLLM instance is ready")
		Eventually(func() error {
			return waitForLiteLLMInstanceReady()
		}, 30*time.Second, 2*time.Second).Should(Succeed(), "LiteLLM instance must be ready before tests")
	})

	// BeforeEach ensures the LiteLLM instance is still ready before each test
	BeforeEach(func() {
		By("verifying LiteLLM instance is still ready before test execution")
		Eventually(func() error {
			return waitForLiteLLMInstanceReady()
		}, 30*time.Second, 2*time.Second).Should(Succeed(), "LiteLLM instance must be ready before each test")
	})

	Context("Model Lifecycle", func() {
		// TODO: model update is missing a lot of functionality, and complicated by litellm returning some values
		// in scientific notation, the whole model update needs to be reworked to handle this.
		//
		// It("should create, update, and delete a model successfully", func() {
		// 	modelName := "e2e-test-model"
		// 	modelCRName := "e2e-test-model-cr"

		// 	By("creating a model CR")
		// 	modelCR := createModelCR(modelCRName, modelName)
		// 	Expect(k8sClient.Create(context.Background(), modelCR)).To(Succeed())

		// 	By("verifying the model was created in LiteLLM")
		// 	Eventually(func() error {
		// 		return verifyModelExistsInLiteLLM(modelCRName)
		// 	}, testTimeout, testInterval).Should(Succeed())

		// 	By("updating the model CR")
		// 	updatedModelCR := &litellmv1alpha1.Model{}
		// 	Expect(k8sClient.Get(context.Background(), types.NamespacedName{
		// 		Name:      modelCRName,
		// 		Namespace: modelTestNamespace,
		// 	}, updatedModelCR)).To(Succeed())

		// 	newInputCostPerToken := "0.00004"
		// 	newOutputCostPerToken := "0.00008"
		// 	// Update the model parameters
		// 	updatedModelCR.Spec.LiteLLMParams.InputCostPerToken = stringPtr(newInputCostPerToken)
		// 	updatedModelCR.Spec.LiteLLMParams.OutputCostPerToken = stringPtr(newOutputCostPerToken)
		// 	Expect(k8sClient.Update(context.Background(), updatedModelCR)).To(Succeed())

		// 	By("verifying the model was updated in LiteLLM")
		// 	Eventually(func() error {
		// 		return verifyModelUpdatedInLiteLLM(modelCRName, newInputCostPerToken, newOutputCostPerToken)
		// 	}, testTimeout, testInterval).Should(Succeed())

		// 	By("deleting the model CR")
		// 	Expect(k8sClient.Delete(context.Background(), updatedModelCR)).To(Succeed())

		// 	By("verifying the model was deleted from LiteLLM")
		// 	Eventually(func() error {
		// 		return verifyModelDeletedFromLiteLLM(modelCRName)
		// 	}, testTimeout, testInterval).Should(Succeed())
		// })

		It("should handle model creation with invalid parameters", func() {
			modelName := "invalid-test-model"
			modelCRName := "invalid-test-model-cr"

			By("creating a model CR with invalid parameters")
			invalidModelCR := createInvalidModelCR(modelCRName, modelName)
			Expect(k8sClient.Create(context.Background(), invalidModelCR)).To(Succeed())

			By("verifying the model CR shows error status due to no model specified")
			eventuallyVerify(func() error {
				return verifyModelCRStatusError(modelCRName, statusError, "LiteLLMParams.Model is not set")
			})

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
			eventuallyVerify(func() error {
				if err := verifyModelExistsInLiteLLM(model1CRName); err != nil {
					return err
				}
				return verifyModelExistsInLiteLLM(model2CRName)
			})

			By("verifying both model CRs have ready status")
			eventuallyVerify(func() error {
				if err := verifyModelCRStatus(model1CRName, statusReady); err != nil {
					return err
				}
				return verifyModelCRStatus(model2CRName, statusReady)
			})

			By("cleaning up both model CRs")
			Expect(k8sClient.Delete(context.Background(), model1CR)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), model2CR)).To(Succeed())

			By("verifying both models were deleted from LiteLLM")
			eventuallyVerify(func() error {
				if err := verifyModelDeletedFromLiteLLM(model1Name); err != nil {
					return err
				}
				return verifyModelDeletedFromLiteLLM(model2Name)
			})
		})
	})

	Context("Model Validation", func() {

		It("should validate required fields are present in the secret based on model provider", func() {
			By("creating a model secret with missing fields for the provider")
			genericSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-openai-secret",
					Namespace: e2eTestNamespace,
				},
				Data: map[string][]byte{
					// required fields are apiKey and apiBase, missing should be apiBase
					"apiKey": []byte("test-api-key"),
				},
			}
			Expect(k8sClient.Create(context.Background(), genericSecret)).To(Succeed())

			By("creating a model CR without required fields")
			invalidModelCr := createModelCR("invalidmodelcr", "invalid-secret")
			invalidModelCr.Spec.ModelSecretRef = litellmv1alpha1.SecretRef{
				SecretName: "invalid-openai-secret",
				Namespace:  e2eTestNamespace,
			}
			Expect(k8sClient.Create(context.Background(), invalidModelCr)).To(Succeed())

			By("verifying the model CR shows error status")
			eventuallyVerify(func() error {
				return verifyModelCRStatusError(invalidModelCr.Name, statusError, "required field 'apiBase' is missing for azure provider")
			})

			By("cleaning up invalid model CR")
			Expect(k8sClient.Delete(context.Background(), invalidModelCr)).To(Succeed())
		})
	})

})

func mustSamplePath(relParts ...string) string {
	// determine directory of this source file
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		Fail("unable to determine caller file path")
	}
	baseDir := filepath.Dir(thisFile)

	// compose path: ../samples/<file> relative to this test file
	parts := append([]string{baseDir, "..", "samples"}, relParts...)
	p := filepath.Clean(filepath.Join(parts...))

	if _, err := os.Stat(p); os.IsNotExist(err) {
		Fail(fmt.Sprintf("sample file not found: %s (cwd=%s)", p, mustGetwd()))
	}
	return p
}

func mustGetwd() string {
	wd, _ := os.Getwd()
	return wd
}

func createPostgresSecret() {
	// create postgres secret from yaml
	path := mustSamplePath("postgres-secret.yaml")
	cmd := exec.Command("kubectl", "apply", "-f", path)
	_, err := utils.Run(cmd)
	if err != nil {
		Expect(err).NotTo(HaveOccurred())
	}
}

func createPostgresInstance() {
	//install Postges operator
	cmd := exec.Command("kubectl", "apply", "-f", "https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/release-1.21/releases/cnpg-1.21.0.yaml")
	_, err := utils.Run(cmd)
	if err != nil {
		Expect(err).NotTo(HaveOccurred())
	}
	// Wait for the Postgres operator to be ready
	cmd = exec.Command("kubectl", "wait", "--for=condition=Ready", "pod", "-l", "app.kubernetes.io/name=cloudnative-pg", "-n", "cnpg-system", "--timeout=300s")
	_, err = utils.Run(cmd)
	if err != nil {
		Expect(err).NotTo(HaveOccurred())
	}
	// Deploy PostgreSQL
	path := mustSamplePath("test-postgres.yaml")
	cmd = exec.Command("kubectl", "apply", "-f", path)
	_, err = utils.Run(cmd)
	if err != nil {
		Expect(err).NotTo(HaveOccurred())
	}

	// Wait for the postgres init job to complete
	EventuallyWithOffset(1, func() error {
		return waitForPostgresInitComplete()
	}, testTimeout, testInterval).Should(Succeed())

	// Wait for the cluster to be ready
	cmd = exec.Command("kubectl", "wait", "--for=condition=Ready", "cluster/litellm-postgres", "-n", e2eTestNamespace, "--timeout=300s")
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())

	// Wait for pod to be ready
	EventuallyWithOffset(1, func() error {
		return waitForPostgresPodReady()
	}, testTimeout, testInterval).Should(Succeed())
}

// ============================================================================
// Creation Helpers
// ============================================================================

func createLiteLLMInstance() {
	By("creating LiteLLM instance CR")
	liteLLMInstance := &litellmv1alpha1.LiteLLMInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      e2eTestInstance,
			Namespace: e2eTestNamespace,
		},
		Spec: litellmv1alpha1.LiteLLMInstanceSpec{
			Image: "ghcr.io/berriai/litellm-database:main-v1.74.9.rc.1",
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
			Namespace: e2eTestNamespace,
		},
		Spec: litellmv1alpha1.ModelSpec{
			ConnectionRef: litellmv1alpha1.ConnectionRef{
				InstanceRef: litellmv1alpha1.InstanceRef{
					Namespace: e2eTestNamespace,
					Name:      e2eTestInstance,
				},
			},
			ModelName: modelName,
			ModelSecretRef: litellmv1alpha1.SecretRef{
				Namespace:  e2eTestNamespace,
				SecretName: "test-model-secret",
			},
			LiteLLMParams: litellmv1alpha1.LiteLLMParams{
				ApiKey:             stringPtr("sk-test-api-key"),
				ApiBase:            stringPtr("https://api.openai.com/v1"),
				InputCostPerToken:  stringPtr("0.00003"),
				OutputCostPerToken: stringPtr("0.00006"),
				TPM:                intPtr(10000),
				RPM:                intPtr(100),
				Timeout:            intPtr(60),
				Model:              stringPtr("azure/gpt-4"),
				MaxRetries:         intPtr(3),
				Organization:       stringPtr("test-org"),
				UseInPassThrough:   boolPtr(false),
				UseLiteLLMProxy:    boolPtr(true),
			},
		},
	}
}

func createInvalidModelCR(name, modelName string) *litellmv1alpha1.Model {
	return &litellmv1alpha1.Model{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: e2eTestNamespace,
		},
		Spec: litellmv1alpha1.ModelSpec{
			ConnectionRef: litellmv1alpha1.ConnectionRef{
				InstanceRef: litellmv1alpha1.InstanceRef{
					Namespace: e2eTestNamespace,
					Name:      e2eTestInstance,
				},
			},
			ModelName: modelName,
			ModelSecretRef: litellmv1alpha1.SecretRef{
				Namespace:  e2eTestNamespace,
				SecretName: "test-model-secret",
			},
			LiteLLMParams: litellmv1alpha1.LiteLLMParams{
				// Missing model
				ApiBase:            stringPtr("https://api.openai.com/v1"),
				InputCostPerToken:  stringPtr("-0.00003"), // Invalid negative cost
				OutputCostPerToken: stringPtr("0.00006"),
			},
		},
	}
}

// ============================================================================
// Retrieval Helpers
// ============================================================================

func getModelCR(modelCRName string) (*litellmv1alpha1.Model, error) {
	modelCR := &litellmv1alpha1.Model{}
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      modelCRName,
		Namespace: e2eTestNamespace,
	}, modelCR)
	if err != nil {
		return nil, fmt.Errorf("failed to get model %s: %w", modelCRName, err)
	}
	return modelCR, nil
}

// ============================================================================
// Verification Helpers - LiteLLM
// ============================================================================

func verifyModelCRStatusError(modelCRName, expectedStatus string, errorMsg string) error {
	modelCR, err := getModelCR(modelCRName)
	if err != nil {
		return err
	}

	return verifyReadyError(modelCR.GetConditions(), expectedStatus, errorMsg)
}

func verifyModelExistsInLiteLLM(modelCRName string) error {
	modelCR, err := getModelCR(modelCRName)
	if err != nil {
		return err
	}

	if modelCR.Status.ModelId == nil || *modelCR.Status.ModelId == "" {
		return fmt.Errorf("model %s has empty status.modelId", modelCRName)
	}

	return nil
}

func verifyModelDeletedFromLiteLLM(modelCRName string) error {
	modelCR := &litellmv1alpha1.Model{}
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      modelCRName,
		Namespace: e2eTestNamespace,
	}, modelCR)

	if err != nil {
		// If the CR is not found, that's acceptable - it means it was deleted
		if errors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("failed to get model %s: %w", modelCRName, err)
	}

	if modelCR.GetDeletionTimestamp().IsZero() {
		return fmt.Errorf("model %s is not marked for deletion", modelCRName)
	}

	return fmt.Errorf("model %s is not deleted", modelCRName)
}

// ============================================================================
// Verification Helpers - Status
// ============================================================================

func verifyModelCRStatus(modelCRName, expectedStatus string) error {
	modelCR, err := getModelCR(modelCRName)
	if err != nil {
		return err
	}

	return verifyReady(modelCR.GetConditions(), expectedStatus)
}

// ============================================================================
// Pointer Helpers - for creating pointers to primitive types
// ============================================================================
func stringPtr(v string) *string { return &v }
func intPtr(v int) *int          { return &v }
func boolPtr(v bool) *bool       { return &v }
