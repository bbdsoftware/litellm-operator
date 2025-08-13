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

package litellm

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	litellmv1alpha1 "github.com/bbdsoftware/litellm-operator/api/litellm/v1alpha1"
)

var _ = Describe("LiteLLMInstance Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		litellminstance := &litellmv1alpha1.LiteLLMInstance{}

		BeforeEach(func() {
			By("creating the test database secret")
			databaseSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-database-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"host":     []byte("localhost"),
					"password": []byte("test-password"),
					"username": []byte("test-user"),
					"dbname":   []byte("test-db"),
				},
			}
			Expect(k8sClient.Create(ctx, databaseSecret)).To(Succeed())

			By("creating the test redis secret")
			redisSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-redis-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"host":     []byte("localhost"),
					"password": []byte("test-password"),
				},
			}
			Expect(k8sClient.Create(ctx, redisSecret)).To(Succeed())

			By("creating the custom resource for the Kind LiteLLMInstance")
			err := k8sClient.Get(ctx, typeNamespacedName, litellminstance)
			if err != nil && errors.IsNotFound(err) {
				resource := &litellmv1alpha1.LiteLLMInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: litellmv1alpha1.LiteLLMInstanceSpec{
						Image:     "ghcr.io/berriai/litellm-database:main-v1.74.9.rc.1",
						MasterKey: "test-master-key",
						DatabaseSecretRef: litellmv1alpha1.DatabaseSecretRef{
							NameRef: "test-database-secret",
							Keys: litellmv1alpha1.DatabaseSecretKeys{
								HostSecret:     "host",
								PasswordSecret: "password",
								UsernameSecret: "username",
								DbnameSecret:   "dbname",
							},
						},
						RedisSecretRef: litellmv1alpha1.RedisSecretRef{
							NameRef: "test-redis-secret",
							Keys: litellmv1alpha1.RedisSecretKeys{
								HostSecret:     "host",
								PortSecret:     "port",
								PasswordSecret: "password",
							},
						},
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &litellmv1alpha1.LiteLLMInstance{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance LiteLLMInstance")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			By("Cleanup the test secrets")
			databaseSecret := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-database-secret", Namespace: "default"}, databaseSecret)
			if err == nil {
				Expect(k8sClient.Delete(ctx, databaseSecret)).To(Succeed())
			}

			redisSecret := &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-redis-secret", Namespace: "default"}, redisSecret)
			if err == nil {
				Expect(k8sClient.Delete(ctx, redisSecret)).To(Succeed())
			}
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &LiteLLMInstanceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})
})
