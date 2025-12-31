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

package virtualkey

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/auth/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/controller/base"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
	"github.com/bbdsoftware/litellm-operator/internal/util"
)

func setupIntegrationTestVirtualKeyReconciler(client *litellm.LitellmClient) *VirtualKeyReconciler {
	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).NotTo(HaveOccurred())

	reconciler := NewVirtualKeyReconciler(k8sClient, scheme.Scheme)
	reconciler.LitellmClient = client
	reconciler.litellmResourceNaming = util.NewLitellmResourceNaming(&authv1alpha1.ConnectionRef{})

	err = reconciler.SetupWithManager(k8sManager)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err := k8sManager.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()
	return reconciler
}

var _ = Describe("VirtualKey Controller", Ordered, func() {
	var (
		ctx            context.Context
		fixture        *LitellmDockerFixture
		litellmClient  *litellm.LitellmClient
		reconciler     *VirtualKeyReconciler
		virtualKeyName client.ObjectKey

		exerciseVirtualKeyInSecret = func(ctx context.Context) error {
			GinkgoHelper()
			alias := fmt.Sprintf("%s-alias", virtualKeyName.Name)
			secretName := reconciler.litellmResourceNaming.GenerateSecretName(alias)
			secret := &corev1.Secret{}
			err := k8sClient.Get(ctx, types.NamespacedName{
				Name:      secretName,
				Namespace: virtualKeyName.Namespace,
			}, secret)
			if err != nil {
				return err
			}

			key, ok := secret.Data["key"]
			if !ok {
				return errors.New("secret data does not contain key")
			}

			if !strings.HasPrefix(string(key), "sk-") {
				GinkgoWriter.Printf("key does not have expected prefix: %s\n", key)
			}

			litellmClientForVirtualKey := litellm.NewLitellmClient(fixture.BaseURL, string(key))
			_, err = litellmClientForVirtualKey.GetVirtualKeyInfo(ctx, "")
			return err
		}
	)

	BeforeAll(func() {
		ctx = context.Background()
		By("starting LiteLLM in Docker")
		fixture = NewLitellmDockerFixture()
		err := fixture.Setup(ctx)
		Expect(err).NotTo(HaveOccurred())

		litellmClient = litellm.NewLitellmClient(fixture.BaseURL, fixture.MasterKey)
		virtualKeyName = types.NamespacedName{
			Name:      "test-vk",
			Namespace: "default",
		}
		reconciler = setupIntegrationTestVirtualKeyReconciler(litellmClient)
	})

	AfterAll(func() {
		By("tearing down LiteLLM in Docker")
		fixture.Teardown()
	})

	Context("when creating a new virtual key", Ordered, func() {
		BeforeEach(func() {
			// Setup mock to indicate key doesn't exist yet
			// mockClient.keyExists = false
		})

		It("should successfully create virtual key in LiteLLM", func() {
			By("creating a new virtual key in Kubernetes")
			Expect(k8sClient.Create(ctx, createTestVirtualKey(virtualKeyName.Name, virtualKeyName.Namespace))).To(Succeed())

			By("verifying the status in Kubernetes")
			virtualKey := &authv1alpha1.VirtualKey{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, virtualKeyName, virtualKey)
				Expect(err).NotTo(HaveOccurred())
				GinkgoWriter.Printf("VirtualKey status: %+v\n", virtualKey.Status)
				return virtualKey.Status.KeyAlias != ""
			}, 3*time.Second, time.Second).Should(BeTrue())

			// Verify status was updated
			Expect(virtualKey.Status.KeyAlias).To(Equal(virtualKey.Spec.KeyAlias))
			Expect(virtualKey.Status.ObservedGeneration).To(Equal(virtualKey.Generation))
			assertCondition(virtualKey.Status.Conditions, base.CondReady, base.ReasonReady)

			// Verify finalizer was added
			Expect(virtualKey.Finalizers).To(ContainElement(util.FinalizerName))

			By("verifying the virtual key was created in LiteLLM")
			vk, err := litellmClient.GetVirtualKeyInfo(ctx, virtualKey.Status.Token)
			Expect(err).NotTo(HaveOccurred())
			Expect(vk.KeyAlias).To(Equal(virtualKey.Spec.KeyAlias))

		})

		It("should create a secret for the virtual key", func() {
			// Verify secret was created
			secretName := reconciler.litellmResourceNaming.GenerateSecretName(fmt.Sprintf("%s-alias", virtualKeyName.Name))
			secret := &corev1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      secretName,
					Namespace: virtualKeyName.Namespace,
				}, secret)
				return err == nil
			}, 3*time.Second, time.Second).Should(BeTrue())

			// Verify secret data
			Expect(secret.Data).To(HaveKey("key"))
			Expect(string(secret.Data["key"])).To(HavePrefix("sk-"))

			// Verify owner reference
			Expect(secret.OwnerReferences).To(HaveLen(1))
			Expect(secret.OwnerReferences[0].Kind).To(Equal("VirtualKey"))
			Expect(secret.OwnerReferences[0].Name).To(Equal(virtualKeyName.Name))

			By("verifying key in secret can execute LiteLLM operations")
			err := exerciseVirtualKeyInSecret(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when virtual key already exists", func() {
		It("should sync without update when no changes needed", func() {
			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: virtualKeyName,
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(Equal(60 * time.Second))

			// Verify conditions are set correctly
			updatedVK := &authv1alpha1.VirtualKey{}
			err = reconciler.Get(ctx, virtualKeyName, updatedVK)
			Expect(err).NotTo(HaveOccurred())

			assertCondition(updatedVK.Status.Conditions, base.CondReady, base.ReasonReady)
		})

		It("should update virtual key when drift detected", func() {
			By("updating the max budget in the virtual key")
			virtualKey := &authv1alpha1.VirtualKey{}
			err := k8sClient.Get(ctx, virtualKeyName, virtualKey)
			Expect(err).NotTo(HaveOccurred())

			virtualKey.Spec.MaxBudget = "100"

			err = k8sClient.Update(ctx, virtualKey)
			Expect(err).NotTo(HaveOccurred())

			By("verifying the virtual key was updated in Kubernetes")
			updatedVK := &authv1alpha1.VirtualKey{}
			err = k8sClient.Get(ctx, virtualKeyName, updatedVK)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedVK.Spec.MaxBudget).To(Equal("100"))

			By("verifying the virtual key was updated in LiteLLM")
			Eventually(func() bool {
				vk, err := litellmClient.GetVirtualKeyInfo(ctx, virtualKey.Status.Token)
				if err != nil {
					return false
				}
				return vk.MaxBudget == 100
			}, 3*time.Second, time.Second).Should(BeTrue())

			By("verifying key in secret can execute LiteLLM operations")
			err = exerciseVirtualKeyInSecret(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when handling deletion", func() {
		It("should delete virtual key from LiteLLM and remove finalizer", func() {
			virtualKey := &authv1alpha1.VirtualKey{}
			err := k8sClient.Get(ctx, virtualKeyName, virtualKey)
			Expect(err).NotTo(HaveOccurred())

			err = k8sClient.Delete(ctx, virtualKey)
			Expect(err).NotTo(HaveOccurred())

			By("verifying the virtual key was deleted from LiteLLM")
			Eventually(func() bool {
				_, err := litellmClient.GetVirtualKeyInfo(ctx, virtualKey.Status.Token)
				return errors.Is(err, litellm.ErrNotFound)
			}, 3*time.Second, time.Second).Should(BeTrue())

			By("verifying the finalizer was removed")
			updatedVK := &authv1alpha1.VirtualKey{}
			err = k8sClient.Get(ctx, virtualKeyName, updatedVK)
			Expect(kerrors.IsNotFound(err)).To(BeTrue())
		})
	})

	Context("when resource is not found", func() {
		It("should return without error", func() {
			result, err := reconciler.Reconcile(ctx, ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      "non-existent",
					Namespace: "default",
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(ctrl.Result{}))
		})
	})
})
