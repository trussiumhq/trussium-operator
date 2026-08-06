/*
Copyright 2026.

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
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	runtimev1alpha1 "github.com/trussium/trussium-operator/api/v1alpha1"
)

func TestReconcileCreatesOwnedRuntimeResources(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme(t)
	runtimeResource := newTestRuntime()
	runtimeResource.UID = types.UID("runtime-uid")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(runtimeResource).
		Build()

	reconciler := TrussiumRuntimeReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      runtimeResource.Name,
			Namespace: runtimeResource.Namespace,
		},
	}

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("reconcile TrussiumRuntime: %v", err)
	}

	assertManagedResourceExists(
		t,
		ctx,
		fakeClient,
		runtimeResource,
		&corev1.ConfigMap{},
	)

	assertManagedResourceExists(
		t,
		ctx,
		fakeClient,
		runtimeResource,
		&corev1.ServiceAccount{},
	)

	assertManagedResourceExists(
		t,
		ctx,
		fakeClient,
		runtimeResource,
		&corev1.Service{},
	)

	assertManagedResourceExists(
		t,
		ctx,
		fakeClient,
		runtimeResource,
		&appsv1.Deployment{},
	)
}

func TestReconcileCorrectsDrift(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme(t)
	runtimeResource := newTestRuntime()
	runtimeResource.UID = types.UID("runtime-uid")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(runtimeResource).
		Build()

	reconciler := TrussiumRuntimeReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	request := ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(runtimeResource),
	}

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("initial reconciliation: %v", err)
	}

	var deployment appsv1.Deployment
	if err := fakeClient.Get(
		ctx,
		request.NamespacedName,
		&deployment,
	); err != nil {
		t.Fatalf("get managed Deployment: %v", err)
	}

	deployment.Spec.Replicas = nil
	deployment.Spec.Template.Spec.Containers[0].Image = "invalid/image:drift"

	if err := fakeClient.Update(ctx, &deployment); err != nil {
		t.Fatalf("introduce Deployment drift: %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("reconcile drifted Deployment: %v", err)
	}

	if err := fakeClient.Get(
		ctx,
		request.NamespacedName,
		&deployment,
	); err != nil {
		t.Fatalf("get corrected Deployment: %v", err)
	}

	if deployment.Spec.Replicas == nil ||
		*deployment.Spec.Replicas != 2 {
		t.Fatalf(
			"expected two replicas after drift correction, received %#v",
			deployment.Spec.Replicas,
		)
	}

	expectedImage := testRuntimeImage
	actualImage := deployment.Spec.Template.Spec.Containers[0].Image

	if actualImage != expectedImage {
		t.Fatalf(
			"expected image %q after drift correction, received %q",
			expectedImage,
			actualImage,
		)
	}
}

func TestReconcileRecreatesDeletedManagedResource(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme(t)
	runtimeResource := newTestRuntime()
	runtimeResource.UID = types.UID("runtime-uid")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(runtimeResource).
		Build()

	reconciler := TrussiumRuntimeReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	request := ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(runtimeResource),
	}

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("initial reconciliation: %v", err)
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runtimeResource.Name,
			Namespace: runtimeResource.Namespace,
		},
	}

	if err := fakeClient.Delete(ctx, service); err != nil {
		t.Fatalf("delete managed Service: %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("reconcile after Service deletion: %v", err)
	}

	var recreated corev1.Service
	if err := fakeClient.Get(
		ctx,
		request.NamespacedName,
		&recreated,
	); err != nil {
		t.Fatalf("get recreated Service: %v", err)
	}

	if !metav1.IsControlledBy(&recreated, runtimeResource) {
		t.Fatal("recreated Service is missing controller owner reference")
	}
}

func TestReconcileIsIdempotent(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme(t)
	runtimeResource := newTestRuntime()
	runtimeResource.UID = types.UID("runtime-uid")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(runtimeResource).
		Build()

	reconciler := TrussiumRuntimeReconciler{
		Client: fakeClient,
		Scheme: scheme,
	}

	request := ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(runtimeResource),
	}

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("first reconciliation: %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("second reconciliation: %v", err)
	}
}

func TestReconcileIgnoresMissingRuntime(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme(t)

	reconciler := TrussiumRuntimeReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			Build(),
		Scheme: scheme,
	}

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "missing",
			Namespace: testRuntimeNamespace,
		},
	}

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("expected missing runtime to be ignored: %v", err)
	}
}

func newTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()

	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("register core Kubernetes API: %v", err)
	}

	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("register apps Kubernetes API: %v", err)
	}

	if err := runtimev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("register TrussiumRuntime API: %v", err)
	}

	return scheme
}

func assertManagedResourceExists(
	t *testing.T,
	ctx context.Context,
	kubernetesClient client.Client,
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	object client.Object,
) {
	t.Helper()

	key := client.ObjectKeyFromObject(runtimeResource)

	if err := kubernetesClient.Get(ctx, key, object); err != nil {
		t.Fatalf("get managed %T: %v", object, err)
	}

	if !metav1.IsControlledBy(object, runtimeResource) {
		t.Fatalf(
			"managed %T is missing controller owner reference",
			object,
		)
	}

	if object.GetLabels()["app.kubernetes.io/managed-by"] !=
		"trussium-operator" {
		t.Fatalf(
			"managed %T is missing operator label",
			object,
		)
	}
}
