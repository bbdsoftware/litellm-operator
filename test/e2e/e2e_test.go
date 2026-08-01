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
	"strconv"
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

const (
	e2eTestNamespace  = "e2e-test"
	namespace         = "litellm-operator-system"
	testTimeout       = 1 * time.Minute
	testInterval      = 5 * time.Second
	e2eTestInstance   = "e2e-test-instance"
	e2eTestModelGPT4o = "gpt-4o"
)

// Common string constants used when comparing condition statuses in kubectl jsonpath output
const (
	condStatusTrue  = "True"
	condStatusFalse = "False"
	statusReady     = "Ready"
	statusError     = "Error"
)

var k8sClient client.Client

var _ = BeforeSuite(func() {
	By("creating manager namespace")
	cmd := exec.Command("kubectl", "create", "ns", namespace)
	_, _ = utils.Run(cmd)

	var err error

	// projectimage stores the name of the image used in the example
	var projectimage = "litellm-operator:dev"

	By("building the manager(Operator) image")
	cmd = exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectimage))
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("installing CRDs")
	cmd = exec.Command("make", "install")
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("deploying the controller-manager")
	cmd = exec.Command("make", "deploy-controller-dev", fmt.Sprintf("IMG=%s", projectimage))
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("validating that the controller-manager pod is running as expected")
	cmd = exec.Command("kubectl", "wait", "--for=condition=Ready", "pod", "-l", "control-plane=controller-manager", "-n", namespace, "--timeout=300s")
	_, err = utils.Run(cmd)
	if err != nil {
		// Describe the pod to get more information about the error
		cmd = exec.Command("kubectl", "describe", "pod", "-l", "control-plane=controller-manager", "-n", namespace)
		output, _ := utils.Run(cmd)
		fmt.Println("controller-manager: ", string(output))
		Expect(err).NotTo(HaveOccurred())
	}

	// Setup LiteLLM instance for all e2e tests
	By("setting up LiteLLM instance for e2e tests")
	setupLiteLLMInstanceForE2E()
})

func setupLiteLLMInstanceForE2E() {
	// Initialize k8sClient for LiteLLM setup
	cfg := config.GetConfigOrDie()
	var err error
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("creating test namespace")
	cmd := exec.Command("kubectl", "create", "namespace", e2eTestNamespace)
	_, _ = utils.Run(cmd)

	By("Starting Postgres instance")
	createPostgresInstance()

	By("Creating Postgres Secret")
	createPostgresSecret()

	By("creating model secret")
	path := mustSamplePath("test-model-secret.yaml")
	cmd = exec.Command("kubectl", "apply", "-f", path)
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("creating LiteLLM instance")
	createLiteLLMInstance()

	By("waiting for LiteLLM instance to be ready")
	EventuallyWithOffset(1, func() error {
		return waitForLiteLLMInstanceReady()
	}, 3*time.Minute, testInterval).Should(Succeed())
}

var _ = AfterSuite(func() {
	By("cleaning up LiteLLM test namespace")
	// Ensure we wait a moment to allow any final operations to complete
	time.Sleep(2 * time.Second)
	// Deleting the namespace will hang if we don't delete the resources first
	// this can happen if a test fails, which prevents the cleanup from happening
	cmd := exec.Command("kubectl", "delete", "teammemberassociation", "-n", e2eTestNamespace, "--all")
	_, _ = utils.Run(cmd)
	cmd = exec.Command("kubectl", "delete", "team", "-n", e2eTestNamespace, "--all")
	_, _ = utils.Run(cmd)
	cmd = exec.Command("kubectl", "delete", "user", "-n", e2eTestNamespace, "--all")
	_, _ = utils.Run(cmd)
	cmd = exec.Command("kubectl", "delete", "virtualkey", "-n", e2eTestNamespace, "--all")
	_, _ = utils.Run(cmd)
	cmd = exec.Command("kubectl", "delete", "model", "-n", e2eTestNamespace, "--all")
	_, _ = utils.Run(cmd)
	cmd = exec.Command("kubectl", "delete", "namespace", e2eTestNamespace)
	_, _ = utils.Run(cmd)

	By("removing manager namespace")
	cmd = exec.Command("kubectl", "delete", "ns", namespace)
	_, _ = utils.Run(cmd)
})

var _ = Describe("controller", Ordered, func() {
	// Additional e2e test files are included automatically via init() functions
	// in user_e2e_test.go, team_e2e_test.go, virtualkey_e2e_test.go, and integration_e2e_test.go
})

// ============================================================================
// Setup Helpers
// ============================================================================

func waitForLiteLLMInstanceReady() error {
	litellmInstance := &litellmv1alpha1.LiteLLMInstance{}
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      e2eTestInstance,
		Namespace: e2eTestNamespace,
	}, litellmInstance)
	if err != nil {
		return err
	}

	return verifyReady(litellmInstance.Status.Conditions, statusReady)
}

func waitForPostgresInitComplete() error {
	cmd := exec.Command("kubectl", "wait", "--for=condition=Complete", "job/litellm-postgres-1-initdb", "-n", e2eTestNamespace, "--timeout=300s")
	_, err := utils.Run(cmd)
	if err != nil {
		return err
	}
	return nil
}

func waitForPostgresPodReady() error {
	cmd := exec.Command("kubectl", "wait", "--for=condition=Ready", "pod", "-l", "cnpg.io/instanceName=litellm-postgres-1", "-n", e2eTestNamespace, "--timeout=300s")
	_, err := utils.Run(cmd)
	if err != nil {
		return err
	}
	return nil
}

// ============================================================================
// Utility Helpers
// ============================================================================

// eventuallyVerify wraps the common Eventually pattern for verification functions
func eventuallyVerify(verifyFn func() error) {
	Eventually(verifyFn, testTimeout, testInterval).Should(Succeed())
}

// getSecret retrieves a secret by name
func getSecret(secretName string) error {
	secretCR := &corev1.Secret{}
	err := k8sClient.Get(context.Background(), types.NamespacedName{
		Name:      secretName,
		Namespace: e2eTestNamespace,
	}, secretCR)
	if err != nil {
		// Don't wrap NotFound errors so they can be checked with errors.IsNotFound
		if errors.IsNotFound(err) {
			return err
		}
		return fmt.Errorf("failed to get secret %s: %w", secretName, err)
	}
	return nil
}

// verifySecretDeleted verifies that a secret has been deleted from Kubernetes
func verifySecretDeleted(secretName string) error {
	err := getSecret(secretName)

	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return fmt.Errorf("secret %s still exists in Kubernetes", secretName)
}

// compareBudgetStrings compares two budget strings as floats to account for decimal formatting differences
func compareBudgetStrings(expected, actual string) error {
	expectedFloat, err := strconv.ParseFloat(expected, 64)
	if err != nil {
		return fmt.Errorf("failed to parse expected budget %s as float: %w", expected, err)
	}
	actualFloat, err := strconv.ParseFloat(actual, 64)
	if err != nil {
		return fmt.Errorf("failed to parse actual budget %s as float: %w", actual, err)
	}
	if actualFloat != expectedFloat {
		return fmt.Errorf("expected budget %f, got %f", expectedFloat, actualFloat)
	}
	return nil
}

// parseDurationAndCalculateExpiry parses a duration string and calculates the expected expiry time
func parseDurationAndCalculateExpiry(duration string) (time.Time, error) {
	expectedDurationParsed, err := time.ParseDuration(duration)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse expected duration %s: %w", duration, err)
	}
	// calculate the expected expiry time (now + expected duration)
	return time.Now().Add(expectedDurationParsed), nil
}

// findReadyCondition finds the Ready condition in the conditions slice
func findReadyCondition(conditions []metav1.Condition) (*metav1.Condition, error) {
	for i := range conditions {
		if conditions[i].Type == "Ready" {
			return &conditions[i], nil
		}
	}
	return nil, fmt.Errorf("Ready condition not found")
}

// verifyConditionStatus verifies that the condition has the expected status
func verifyConditionStatus(condition metav1.Condition, expectedStatus string) error {
	got := string(condition.Status)
	var expectedConditionStatus string
	switch expectedStatus {
	case statusReady:
		expectedConditionStatus = condStatusTrue
	case statusError:
		expectedConditionStatus = condStatusFalse
	default:
		expectedConditionStatus = expectedStatus
	}

	if got != expectedConditionStatus {
		return fmt.Errorf("expected status %s (condition status %s), got %s", expectedStatus, expectedConditionStatus, got)
	}
	return nil
}

// verifyReady verifies that the Ready condition has the expected status
func verifyReady(conditions []metav1.Condition, expectedStatus string) error {
	readyCondition, err := findReadyCondition(conditions)
	if err != nil {
		return err
	}

	return verifyConditionStatus(*readyCondition, expectedStatus)
}

// verifyReadyError verifies that the Ready condition has the expected status and the error message is as expected
func verifyReadyError(conditions []metav1.Condition, expectedStatus string, errorMsg string) error {
	condition, err := findReadyCondition(conditions)
	if err != nil {
		return err
	}

	err = verifyConditionStatus(*condition, expectedStatus)
	if err != nil {
		return err
	}

	if condition.Message != errorMsg {
		return fmt.Errorf("expected error message '%s', got '%s'", errorMsg, condition.Message)
	}

	return nil
}
