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
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/bbdsoftware/litellm-operator/test/utils"
)

const namespace = "litellm-operator-system"

var _ = BeforeSuite(func() {
	By("creating manager namespace")
	cmd := exec.Command("kubectl", "create", "ns", namespace)
	_, _ = utils.Run(cmd)

	var err error

	// projectimage stores the name of the image used in the example
	var projectimage = "example.com/litellm-operator:v0.0.1"

	By("building the manager(Operator) image")
	cmd = exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectimage))
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("loading the the manager(Operator) image on Kind")
	err = utils.LoadImageToKindClusterWithName(projectimage)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("installing CRDs")
	cmd = exec.Command("make", "install")
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("deploying the controller-manager")
	cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectimage))
	_, err = utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("validating that the controller-manager pod is running as expected")
	cmd = exec.Command("kubectl", "wait", "--for=condition=Ready", "pod", "-l", "control-plane=controller-manager", "-n", namespace, "--timeout=300s")
	_, err = utils.Run(cmd)
	if err != nil {
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
	cmd := exec.Command("kubectl", "create", "namespace", modelTestNamespace)
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
	}, testTimeout, testInterval).Should(Succeed())
}

var _ = AfterSuite(func() {
	By("cleaning up LiteLLM test namespace")
	// Ensure we wait a moment to allow any final operations to complete
	time.Sleep(2 * time.Second)
	// Deleting the namespace will hang if we don't delete the resources first
	// this can happen if a test fails, which prevents the cleanup from happening
	cmd := exec.Command("kubectl", "delete", "teammemberassociation", "-n", modelTestNamespace, "--all")
	_, _ = utils.Run(cmd)
	cmd = exec.Command("kubectl", "delete", "team", "-n", modelTestNamespace, "--all")
	_, _ = utils.Run(cmd)
	cmd = exec.Command("kubectl", "delete", "user", "-n", modelTestNamespace, "--all")
	_, _ = utils.Run(cmd)
	cmd = exec.Command("kubectl", "delete", "virtualkey", "-n", modelTestNamespace, "--all")
	_, _ = utils.Run(cmd)
	cmd = exec.Command("kubectl", "delete", "namespace", modelTestNamespace)
	_, _ = utils.Run(cmd)

	By("removing manager namespace")
	cmd = exec.Command("kubectl", "delete", "ns", namespace)
	_, _ = utils.Run(cmd)
})

var _ = Describe("controller", Ordered, func() {
	// Additional e2e test files are included automatically via init() functions
	// in user_e2e_test.go, team_e2e_test.go, virtualkey_e2e_test.go, and integration_e2e_test.go
})
