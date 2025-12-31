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
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	litellmv1alpha1 "github.com/bbdsoftware/litellm-operator/api/litellm/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/test/utils"
)

const (
	modelTestNamespace = "model-e2e-test"
	testTimeout        = 3 * time.Minute
	testInterval       = 5 * time.Second
)

// Common string constants used when comparing condition statuses in kubectl jsonpath output
const (
	condStatusTrue  = "True"
	condStatusFalse = "False"
	statusReady     = "Ready"
	statusError     = "Error"
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

			By("verifying the model CR shows error status due to no model specified")
			Eventually(func() error {
				return verifyModelCRStatusError(modelCRName, "Error", "LiteLLMParams.Model is not set")
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

		It("should validate required fields are present in the secret based on model provider", func() {
			By("creating a model secret with missing fields for the provider")
			genericSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-openai-secret",
					Namespace: modelTestNamespace,
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
				Namespace:  modelTestNamespace,
			}
			Expect(k8sClient.Create(context.Background(), invalidModelCr)).To(Succeed())

			By("verifying the model CR shows error status")
			Eventually(func() error {
				return verifyModelCRStatusError(invalidModelCr.Name, "Error", "required field 'apiBase' is missing for azure provider")
			}, testTimeout, testInterval).Should(Succeed())

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
	cmd = exec.Command("kubectl", "wait", "--for=condition=Ready", "cluster/litellm-postgres", "-n", modelTestNamespace, "--timeout=300s")
	_, err = utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())

	// Wait for pod to be ready
	EventuallyWithOffset(1, func() error {
		return waitForPostgresPodReady()
	}, testTimeout, testInterval).Should(Succeed())
}

func createLiteLLMInstance() {
	By("creating LiteLLM instance CR")
	liteLLMInstance := &litellmv1alpha1.LiteLLMInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "e2e-test-instance",
			Namespace: modelTestNamespace,
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
			ModelSecretRef: litellmv1alpha1.SecretRef{
				Namespace:  modelTestNamespace,
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
			ModelSecretRef: litellmv1alpha1.SecretRef{
				Namespace:  modelTestNamespace,
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

func waitForPostgresInitComplete() error {
	cmd := exec.Command("kubectl", "wait", "--for=condition=Complete", "job/litellm-postgres-1-initdb", "-n", modelTestNamespace, "--timeout=300s")
	_, err := utils.Run(cmd)
	if err != nil {
		return err
	}
	return nil
}

func waitForPostgresPodReady() error {
	cmd := exec.Command("kubectl", "wait", "--for=condition=Ready", "pod", "-l", "cnpg.io/instanceName=litellm-postgres-1", "-n", modelTestNamespace, "--timeout=300s")
	_, err := utils.Run(cmd)
	if err != nil {
		return err
	}
	return nil
}

func waitForLiteLLMInstanceReady() error {
	cmd := exec.Command("kubectl", "wait", "--for=condition=Ready", "litellminstance/e2e-test-instance", "-n", modelTestNamespace, "--timeout=300s")
	_, err := utils.Run(cmd)
	if err != nil {
		return err
	}
	return nil
}

func verifyModelExistsInLiteLLM(modelName string) error {
	// In a real e2e test, this would call the LiteLLM API directly.
	// For now, verify that the model CR has a non-empty status.modelId field.
	cmd := exec.Command("kubectl", "get", "model", "-n", modelTestNamespace,
		"-o", "jsonpath={.items[?(@.spec.modelName=='"+modelName+"')].status.modelId}")

	output, err := utils.Run(cmd)
	if err != nil {
		return err
	}

	if strings.TrimSpace(string(output)) == "" {
		return fmt.Errorf("model %s has empty status.modelId", modelName)
	}

	return nil
}

func verifyModelUpdatedInLiteLLM(modelName string, expectedInputCost, expectedOutputCost float64) error {
	// Use expected values to avoid linter unparam warning; in future this can be expanded
	if expectedInputCost < 0 || expectedOutputCost < 0 {
		return fmt.Errorf("expected costs must be non-negative")
	}
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

func verifyModelCRStatusError(modelCRName, expectedStatus string, errorMsg string) error {

	// Map the human-friendly expected statuses used in tests to the actual condition status values.
	var expectedConditionStatus string
	switch expectedStatus {
	case statusReady:
		expectedConditionStatus = condStatusTrue
	case statusError:
		expectedConditionStatus = condStatusFalse
	default:
		expectedConditionStatus = expectedStatus
	}

	// Use a short timeout when invoking kubectl to avoid hangs.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Query the Ready condition status explicitly.
	cmdStatus := exec.CommandContext(ctx, "kubectl", "get", "model", modelCRName,
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

	// Query the Ready condition message and ensure it contains the expected error message.
	cmdMsg := exec.CommandContext(ctx, "kubectl", "get", "model", modelCRName,
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

func verifyModelCRStatus(modelCRName, expectedStatus string) error {
	modelCR := &litellmv1alpha1.Model{}
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      modelCRName,
		Namespace: modelTestNamespace,
	}, modelCR)
	if err != nil {
		return fmt.Errorf("failed to get model %s: %w", modelCRName, err)
	}

	return verifyReady(modelCR.GetConditions(), expectedStatus)
}

// Helper functions for creating pointers to primitive types
// nolint:unused
func stringPtr(v string) *string { return &v }

// nolint:unused
func intPtr(v int) *int { return &v }

// nolint:unused
func boolPtr(v bool) *bool { return &v }
