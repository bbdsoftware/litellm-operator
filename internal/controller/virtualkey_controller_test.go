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

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	authv1alpha1 "github.com/bbdsoftware/litellm-operator/api/v1alpha1"
	"github.com/bbdsoftware/litellm-operator/internal/litellm"
)

type FakeLitellmVirtualKeyClient struct{}

func (l *FakeLitellmVirtualKeyClient) GenerateVirtualKey(ctx context.Context, req *litellm.VirtualKeyRequest) (litellm.VirtualKeyResponse, error) {
	return litellm.VirtualKeyResponse{
		KeyAlias:  "test-virtual-key-alias",
		KeyName:   "test-virtual-key-name",
		UserID:    "test-user-id",
		Expires:   "2024-03-20T10:00:00Z",
		Key:       "test-secret-key",
		TokenID:   "test-token-id",
		MaxBudget: 100.0,
	}, nil
}

func (l *FakeLitellmVirtualKeyClient) DeleteVirtualKey(ctx context.Context, keyAlias string) error {
	return nil
}

func (l *FakeLitellmVirtualKeyClient) CheckVirtualKeyExists(ctx context.Context, virtualKeyID string) (bool, error) {
	return true, nil
}

var _ = Describe("VirtualKey Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default", // TODO(user):Modify as needed
		}
		virtualkey := &authv1alpha1.VirtualKey{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind VirtualKey")
			err := k8sClient.Get(ctx, typeNamespacedName, virtualkey)
			if err != nil && errors.IsNotFound(err) {
				resource := &authv1alpha1.VirtualKey{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					// TODO(user): Specify other spec details if needed.
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// TODO(user): Cleanup logic after each test, like removing the resource instance.
			resource := &authv1alpha1.VirtualKey{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance VirtualKey")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &VirtualKeyReconciler{
				Client:            k8sClient,
				Scheme:            k8sClient.Scheme(),
				LitellmVirtualKey: &FakeLitellmVirtualKeyClient{},
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
