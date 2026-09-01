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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	testNamespace      = "litellm"
	testServiceName    = "litellm-proxy-service"
	testSAName         = "litellm-sa"
	testSecretName     = "litellm-masterkey"
	testSecretKey      = "masterkey"
	testRoleName       = "litellm-role"
	testAppLabel       = "app"
	testAppLabelValue  = "litellm"
	testConfigMapsRule = "configmaps"
	testGetVerb        = "get"
)

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to build scheme: %v", err)
	}
	return scheme
}

// testOwner returns an owner object that SetControllerReference can be called against.
func testOwner() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "litellm-owner",
			Namespace: testNamespace,
			UID:       types.UID("11111111-1111-1111-1111-111111111111"),
		},
	}
}

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

// TestNeedsUpdateNoSpuriousUpdate asserts that an unchanged object of a type the
// controller owns does not report that it needs an update. Every type the
// controller registers with Owns() triggers a fresh reconcile when written, so a
// type that always reports "needs update" makes the controller reconcile itself
// in a loop.
func TestNeedsUpdateNoSpuriousUpdate(t *testing.T) {
	tests := []struct {
		name string
		obj  client.Object
	}{
		{
			name: "Service",
			obj:  testService(),
		},
		{
			name: "ServiceAccount",
			obj: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{Name: testSAName, Namespace: testNamespace},
			},
		},
		{
			name: "Secret",
			obj: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: testSecretName, Namespace: testNamespace},
				Data:       map[string][]byte{testSecretKey: []byte("sk-test")},
			},
		},
		{
			name: "Role",
			obj: &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{Name: testRoleName, Namespace: testNamespace},
				Rules: []rbacv1.PolicyRule{
					{APIGroups: []string{""}, Resources: []string{testConfigMapsRule}, Verbs: []string{testGetVerb}},
				},
			},
		},
		{
			name: "RoleBinding",
			obj: &rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{Name: "litellm-rb", Namespace: testNamespace},
				RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: testRoleName},
			},
		},
		{
			name: "ConfigMap",
			obj: &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: "litellm-config", Namespace: testNamespace},
				Data:       map[string]string{"config.yaml": "model_list: []"},
			},
		},
		{
			name: "Deployment",
			obj: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "litellm", Namespace: testNamespace},
				Spec: appsv1.DeploymentSpec{
					Replicas: Int32Ptr(1),
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{Name: "litellm", Image: "litellm:v1"}},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := tt.obj.DeepCopyObject().(client.Object)
			if update, _ := needsUpdate(existing, tt.obj); update {
				t.Errorf("needsUpdate reported an update for an unchanged %s; "+
					"the controller Owns() this type, so an unconditional write re-triggers reconcile",
					tt.name)
			}
		})
	}
}

// TestNeedsUpdateDetectsRealChange guards the other direction: a genuine change
// must still be reported, otherwise the fix above would stop reconciling drift.
func TestNeedsUpdateDetectsRealChange(t *testing.T) {
	tests := []struct {
		name     string
		existing client.Object
		desired  client.Object
	}{
		{
			name:     "Service port changed",
			existing: testService(),
			desired: func() *corev1.Service {
				svc := testService()
				svc.Spec.Ports[0].Port = 8080
				return svc
			}(),
		},
		{
			name:     "Service selector changed",
			existing: testService(),
			desired: func() *corev1.Service {
				svc := testService()
				svc.Spec.Selector = map[string]string{testAppLabel: "other"}
				return svc
			}(),
		},
		{
			name: "Secret data changed",
			existing: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: testSecretName, Namespace: testNamespace},
				Data:       map[string][]byte{testSecretKey: []byte("sk-old")},
			},
			desired: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{Name: testSecretName, Namespace: testNamespace},
				Data:       map[string][]byte{testSecretKey: []byte("sk-new")},
			},
		},
		{
			name: "Role rules changed",
			existing: &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{Name: testRoleName, Namespace: testNamespace},
				Rules: []rbacv1.PolicyRule{
					{APIGroups: []string{""}, Resources: []string{testConfigMapsRule}, Verbs: []string{testGetVerb}},
				},
			},
			desired: &rbacv1.Role{
				ObjectMeta: metav1.ObjectMeta{Name: testRoleName, Namespace: testNamespace},
				Rules: []rbacv1.PolicyRule{
					{APIGroups: []string{""}, Resources: []string{testConfigMapsRule, "secrets"}, Verbs: []string{testGetVerb}},
				},
			},
		},
		{
			name: "ServiceAccount labels changed",
			existing: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name: testSAName, Namespace: testNamespace,
					Labels: map[string]string{testAppLabel: testAppLabelValue},
				},
			},
			desired: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name: testSAName, Namespace: testNamespace,
					Labels: map[string]string{testAppLabel: "litellm-renamed"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if update, _ := needsUpdate(tt.existing, tt.desired); !update {
				t.Errorf("needsUpdate did not report an update for %s", tt.name)
			}
		})
	}
}

// TestCreateOrUpdateWithRetryIsIdempotent is the loop reproduction. Calling
// CreateOrUpdateWithRetry repeatedly with the same desired Service must issue
// exactly one Create and no Updates. Because the controller registers
// Owns(&corev1.Service{}), every Update here is a fresh reconcile trigger.
func TestCreateOrUpdateWithRetryIsIdempotent(t *testing.T) {
	scheme := testScheme(t)
	owner := testOwner()

	var updates, creates int
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(ctx context.Context, cl client.WithWatch, obj client.Object, opts ...client.CreateOption) error {
				creates++
				return cl.Create(ctx, obj, opts...)
			},
			Update: func(ctx context.Context, cl client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
				updates++
				return cl.Update(ctx, obj, opts...)
			},
		}).
		Build()

	const reconciles = 5
	for i := 0; i < reconciles; i++ {
		if _, err := CreateOrUpdateWithRetry(context.Background(), c, scheme, testService(), owner); err != nil {
			t.Fatalf("reconcile %d: CreateOrUpdateWithRetry returned error: %v", i, err)
		}
	}

	if creates != 1 {
		t.Errorf("expected 1 Create across %d reconciles, got %d", reconciles, creates)
	}
	if updates != 0 {
		t.Errorf("expected 0 Updates across %d reconciles of an unchanged Service, got %d; "+
			"each Update fires the Owns(&corev1.Service{}) watch and triggers the next reconcile",
			reconciles, updates)
	}
}

// TestNeedsUpdateRestartSignal pins which changes require the LiteLLM proxy to
// be restarted. Only a ConfigMap data change does; metadata drift on the same
// ConfigMap is an update without a restart.
func TestNeedsUpdateRestartSignal(t *testing.T) {
	baseConfigMap := func() *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "litellm-config",
				Namespace: testNamespace,
				Labels:    map[string]string{testAppLabel: testAppLabelValue},
			},
			Data: map[string]string{"config.yaml": "model_list: []"},
		}
	}

	tests := []struct {
		name        string
		desired     client.Object
		wantUpdate  bool
		wantRestart bool
	}{
		{
			name:        "ConfigMap data changed requires a restart",
			desired:     func() *corev1.ConfigMap { cm := baseConfigMap(); cm.Data["config.yaml"] = "model_list: [a]"; return cm }(),
			wantUpdate:  true,
			wantRestart: true,
		},
		{
			name: "ConfigMap label changed does not require a restart",
			desired: func() *corev1.ConfigMap {
				cm := baseConfigMap()
				cm.Labels[testAppLabel] = "litellm-renamed"
				return cm
			}(),
			wantUpdate:  true,
			wantRestart: false,
		},
		{
			name:        "unchanged ConfigMap requires neither",
			desired:     baseConfigMap(),
			wantUpdate:  false,
			wantRestart: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update, restart := needsUpdate(baseConfigMap(), tt.desired)
			if update != tt.wantUpdate {
				t.Errorf("update = %v, want %v", update, tt.wantUpdate)
			}
			if restart != tt.wantRestart {
				t.Errorf("restart = %v, want %v", restart, tt.wantRestart)
			}
		})
	}
}
