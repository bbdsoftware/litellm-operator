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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client "sigs.k8s.io/controller-runtime/pkg/client"

	litellmv1alpha1 "github.com/bbdsoftware/litellm-operator/api/litellm/v1alpha1"
	util "github.com/bbdsoftware/litellm-operator/internal/util"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
			existingSecret := &corev1.Secret{}
			err := k8sClient.Get(ctx, types.NamespacedName{Name: databaseSecret.Name, Namespace: databaseSecret.Namespace}, existingSecret)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, databaseSecret)).To(Succeed())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

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
			existingSecret = &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: redisSecret.Name, Namespace: redisSecret.Namespace}, existingSecret)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, redisSecret)).To(Succeed())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			By("creating the test openAI model secret")
			openAIModelSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-openai-model-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"ApiKey":  []byte("test-api-key"),
					"ApiBase": []byte("test-api-base-url"),
				},
			}
			existingSecret = &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: openAIModelSecret.Name, Namespace: openAIModelSecret.Namespace}, existingSecret)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, openAIModelSecret)).To(Succeed())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			By("creating the test bedrock model secret")
			bedrockModelSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-bedrock-model-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"AwsAccessKeyId":     []byte("test-access-key-id"),
					"AwsSecretAccessKey": []byte("test-secret-access-key"),
				},
			}
			existingSecret = &corev1.Secret{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: bedrockModelSecret.Name, Namespace: bedrockModelSecret.Namespace}, existingSecret)
			if err != nil && errors.IsNotFound(err) {
				Expect(k8sClient.Create(ctx, bedrockModelSecret)).To(Succeed())
			} else {
				Expect(err).NotTo(HaveOccurred())
			}

			By("creating the custom resource for the Kind LiteLLMInstance")
			err = k8sClient.Get(ctx, typeNamespacedName, litellminstance)
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
						Models: []litellmv1alpha1.InitModelInstance{
							{
								ModelName:  "amazon.titan-embed-text-v1",
								Identifier: "aws/bedrock-3.0",
								ModelCredentials: litellmv1alpha1.ModelCredentialSecretRef{
									NameRef: "test-bedrock-model-secret",
									Keys: litellmv1alpha1.ModelCredentialSecretKeys{
										AwsAccessKeyID:     "AwsAccessKeyId",
										AwsSecretAccessKey: "AwsSecretAccessKey",
									},
								},
								RequiresAuth: true,
								LiteLLMParams: litellmv1alpha1.LiteLLMParams{
									AwsRegionName:     util.StringPtrOrNil("us-east-1"),
									Model:             util.StringPtrOrNil("amazon.titan-embed-text-v1"),
									MaxBudget:         util.StringPtrOrNil("1000.988"),
									UseLiteLLMProxy:   util.BoolPtr(true),
									InputCostPerToken: util.StringPtrOrNil("0.0001"),
								},
							}, {
								ModelName:    "gpt-3.5-turbo",
								RequiresAuth: true,
								Identifier:   "gpt-3.5",
								ModelCredentials: litellmv1alpha1.ModelCredentialSecretRef{
									NameRef: "test-openai-model-secret",
									Keys: litellmv1alpha1.ModelCredentialSecretKeys{
										ApiKey:  "ApiKey",
										ApiBase: "ApiBase",
									},
								},
								LiteLLMParams: litellmv1alpha1.LiteLLMParams{
									Model:             util.StringPtrOrNil("gpt-3.5-turbo"),
									MaxBudget:         util.StringPtrOrNil("98.0"),
									Organization:      util.StringPtrOrNil("test-org"),
									UseLiteLLMProxy:   util.BoolPtr(true),
									InputCostPerToken: util.StringPtrOrNil("0.000000089"),
								},
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

			if err != nil && errors.IsNotFound(err) {
				return // Resource does not exist, nothing to clean up
			}
			Expect(err).NotTo(HaveOccurred(), "Failed to get the resource instance")

			By("Cleanup the specific resource instance ConfigMap")
			configMap := &corev1.ConfigMap{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: resourceName + "-config", Namespace: "default"}, configMap)
			if err == nil {
				Expect(k8sClient.Delete(ctx, configMap)).To(Succeed())
			} else if !errors.IsNotFound(err) {
				Fail("Failed to get the configmap for the resource: " + err.Error())
			}

			By("Cleanup the specific resource instance LiteLLMInstance")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			By("Cleanup the test secrets")
			secrets := &corev1.SecretList{}
			err = k8sClient.List(ctx, secrets, &client.ListOptions{Namespace: "default"})
			Expect(err).NotTo(HaveOccurred(), "Failed to list secrets in the default namespace")
			for _, secret := range secrets.Items {
				Expect(k8sClient.Delete(ctx, &secret)).To(Succeed())
			}
		})

		It("should successfully reconcile the resource", func() {

			By("Reconciling the created resource")
			controllerReconciler := NewLiteLLMInstanceReconciler(k8sClient, k8sClient.Scheme())

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.

			//verify the configmap exists with its values
			configMap := &corev1.ConfigMap{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: resourceName + "-config", Namespace: "default"}, configMap)
			Expect(err).NotTo(HaveOccurred())
			Expect(configMap.Data).To(HaveKey("proxy_server_config.yaml"))
			yamlData := configMap.Data["proxy_server_config.yaml"]
			Expect(yamlData).To(ContainSubstring("router_settings"))
			Expect(yamlData).To(ContainSubstring("general_settings"))
			Expect(yamlData).To(ContainSubstring("model_list:"))
			// model names created from LiteLLMInstance models have a source tag appended
			Expect(yamlData).To(ContainSubstring("model_name: amazon.titan-embed-text-v1-[inst]"))
			Expect(yamlData).To(ContainSubstring("api_key: os.environ/gpt-3-5-apikey"))
			Expect(yamlData).To(ContainSubstring("aws_access_key_id: os.environ/aws-bedrock-3-0-awsaccesskeyid"))
			Expect(yamlData).To(ContainSubstring("model: gpt-3.5-turbo"))
		})
	})

})

// Additional unit tests for helper functions and rendering logic
var _ = Describe("LiteLLMInstance helpers and rendering", func() {
	ctx := context.Background()

	It("sanitizeKey should produce valid secret names", func() {
		Expect(sanitizeKey("AwsAccessKeyId")).To(Equal("awsaccesskeyid"))
		Expect(sanitizeKey("aws/bedrock-3.0")).To(Equal("aws-bedrock-3-0"))
		// long or weird characters should be normalized
		Expect(sanitizeKey("__Hello.World!!@@")).To(Equal("hello-world"))
	})

	It("buildSecretData preserves existing master key when none provided", func() {
		existing := &corev1.Secret{Data: map[string][]byte{"masterkey": []byte("existing-key")}}
		data := buildSecretData("", existing)
		Expect(string(data["masterkey"])).To(Equal("existing-key"))
	})

	It("buildSecretData uses provided master key when present", func() {
		data := buildSecretData("provided-key", nil)
		Expect(string(data["masterkey"])).To(Equal("provided-key"))
	})

	It("validateModelListForDuplicates detects duplicate identifiers", func() {
		models := []litellmv1alpha1.InitModelInstance{
			{Identifier: "a"},
			{Identifier: "b"},
			{Identifier: "a"},
		}
		ok, err := validateModelListForDuplicates(models)
		Expect(ok).To(BeFalse())
		Expect(err).To(HaveOccurred())
	})

	It("renderProxyConfig returns error when model missing required model name", func() {
		llm := &litellmv1alpha1.LiteLLMInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "x", Namespace: "default"},
			Spec: litellmv1alpha1.LiteLLMInstanceSpec{
				Models: []litellmv1alpha1.InitModelInstance{{LiteLLMParams: litellmv1alpha1.LiteLLMParams{Model: util.StringPtrOrNil("")}}},
			},
		}
		_, err := renderProxyConfig(llm, ctx, k8sClient, k8sClient.Scheme())
		Expect(err).To(HaveOccurred())
	})

	It("renderProxyConfig includes router settings from redis secret ref and maps model secret keys to os.environ/<name>", func() {
		// create a fake model credentials secret and ensure createAdditionalModelSecrets will make secrets
		modelCred := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "model-creds", Namespace: "default"},
			Data:       map[string][]byte{"ApiKey": []byte("k1")},
		}
		Expect(k8sClient.Create(ctx, modelCred)).To(Succeed())

		llm := &litellmv1alpha1.LiteLLMInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "x", Namespace: "default"},
			Spec: litellmv1alpha1.LiteLLMInstanceSpec{
				RedisSecretRef: litellmv1alpha1.RedisSecretRef{
					NameRef: "test-redis-secret",
					Keys:    litellmv1alpha1.RedisSecretKeys{HostSecret: "host", PortSecret: "port", PasswordSecret: "password"},
				},
				Models: []litellmv1alpha1.InitModelInstance{
					{
						ModelName:        "gpt-3.5",
						Identifier:       "gpt-3.5",
						RequiresAuth:     true,
						ModelCredentials: litellmv1alpha1.ModelCredentialSecretRef{NameRef: "model-creds", Keys: litellmv1alpha1.ModelCredentialSecretKeys{ApiKey: "ApiKey"}},
						LiteLLMParams:    litellmv1alpha1.LiteLLMParams{Model: util.StringPtrOrNil("gpt-3.5-turbo")},
					},
				},
			},
		}

		// ensure the owner LiteLLMInstance exists so created secrets can set ownerReferences
		existingLLM := &litellmv1alpha1.LiteLLMInstance{}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: llm.Name, Namespace: llm.Namespace}, existingLLM)
		if err != nil && errors.IsNotFound(err) {
			Expect(k8sClient.Create(ctx, llm)).To(Succeed())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}

		// create the redis secret referenced by the llm so router_settings populate (only if absent)
		redisSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "test-redis-secret", Namespace: "default"},
			Data:       map[string][]byte{"host": []byte("host"), "port": []byte("port"), "password": []byte("password")},
		}
		existingSecret := &corev1.Secret{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: redisSecret.Name, Namespace: redisSecret.Namespace}, existingSecret)
		if err != nil && errors.IsNotFound(err) {
			Expect(k8sClient.Create(ctx, redisSecret)).To(Succeed())
		} else {
			Expect(err).NotTo(HaveOccurred())
		}

		yamlStr, err := renderProxyConfig(llm, ctx, k8sClient, k8sClient.Scheme())
		Expect(err).NotTo(HaveOccurred())
		Expect(yamlStr).To(ContainSubstring("router_settings"))
		Expect(yamlStr).To(ContainSubstring("host: host"))
		Expect(yamlStr).To(ContainSubstring("model_name: gpt-3.5-[inst]"))
		// created secret names are sanitized; ensure the os.environ/ prefix is present
		Expect(yamlStr).To(ContainSubstring("os.environ/"))
	})

})
