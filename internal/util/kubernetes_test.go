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

package util

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace     = "litellm"
	testServiceName   = "litellm-proxy-service"
	testAppLabel      = "app"
	testAppLabelValue = "litellm"
	negAnnotation     = "cloud.google.com/neg"
	foreignAnnotation = "example.com/managed-by"
)

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to build scheme: %v", err)
	}
	return scheme
}

// testOwner returns an owner object SetControllerReference can be called against.
func testOwner() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "litellm-owner",
			Namespace: testNamespace,
			UID:       types.UID("11111111-1111-1111-1111-111111111111"),
		},
	}
}

// testService mirrors how the controller builds its Service: labels, no annotations.
func testService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testServiceName,
			Namespace: testNamespace,
			Labels:    map[string]string{testAppLabel: testAppLabelValue},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{testAppLabel: testAppLabelValue},
			Ports: []corev1.ServicePort{
				{Name: "http", Protocol: corev1.ProtocolTCP, Port: 80, TargetPort: FromInt(4000)},
			},
			Type: corev1.ServiceTypeClusterIP,
		},
	}
}

// TestCreateOrUpdateWithRetryPreservesForeignAnnotations asserts that an update
// keeps annotations written by other controllers. The operator builds its child
// objects from scratch on every reconcile and sets no annotations on them, so
// without merging, every update strips whatever another controller has added.
func TestCreateOrUpdateWithRetryPreservesForeignAnnotations(t *testing.T) {
	scheme := testScheme(t)
	owner := testOwner()

	existing := testService()
	existing.Annotations = map[string]string{
		negAnnotation:     `{"exposed_ports":{"80":{}}}`,
		foreignAnnotation: "another-controller",
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()

	// Change a field the operator genuinely owns, so an update is required.
	desired := testService()
	desired.Spec.Ports[0].Port = 8080

	if _, err := CreateOrUpdateWithRetry(context.Background(), c, scheme, desired, owner); err != nil {
		t.Fatalf("CreateOrUpdateWithRetry returned error: %v", err)
	}

	got := &corev1.Service{}
	key := client.ObjectKey{Name: testServiceName, Namespace: testNamespace}
	if err := c.Get(context.Background(), key, got); err != nil {
		t.Fatalf("failed to read back Service: %v", err)
	}

	if got.Spec.Ports[0].Port != 8080 {
		t.Errorf("operator-owned change was not applied: port = %d, want 8080", got.Spec.Ports[0].Port)
	}
	if got.Annotations[negAnnotation] == "" {
		t.Errorf("annotation %q was stripped by the update; annotations owned by other controllers must survive", negAnnotation)
	}
	if got.Annotations[foreignAnnotation] != "another-controller" {
		t.Errorf("annotation %s = %q, want %q", foreignAnnotation, got.Annotations[foreignAnnotation], "another-controller")
	}
}

// TestCreateOrUpdateWithRetryOperatorAnnotationsWin asserts the operator stays
// authoritative over annotations it declares itself.
func TestCreateOrUpdateWithRetryOperatorAnnotationsWin(t *testing.T) {
	scheme := testScheme(t)
	owner := testOwner()

	const ownedKey = "litellm.bbd.co.za/config-hash"

	existing := testService()
	existing.Annotations = map[string]string{
		ownedKey:      "stale",
		negAnnotation: `{"exposed_ports":{"80":{}}}`,
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()

	desired := testService()
	desired.Annotations = map[string]string{ownedKey: "current"}
	desired.Spec.Ports[0].Port = 8080

	if _, err := CreateOrUpdateWithRetry(context.Background(), c, scheme, desired, owner); err != nil {
		t.Fatalf("CreateOrUpdateWithRetry returned error: %v", err)
	}

	got := &corev1.Service{}
	key := client.ObjectKey{Name: testServiceName, Namespace: testNamespace}
	if err := c.Get(context.Background(), key, got); err != nil {
		t.Fatalf("failed to read back Service: %v", err)
	}

	if got.Annotations[ownedKey] != "current" {
		t.Errorf("annotation %s = %q, want %q", ownedKey, got.Annotations[ownedKey], "current")
	}
	if got.Annotations[negAnnotation] == "" {
		t.Errorf("annotation %q was stripped by the update", negAnnotation)
	}
}
