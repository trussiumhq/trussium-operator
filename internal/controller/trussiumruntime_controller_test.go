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
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/events"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

func TestReconcileCreatesOwnedRuntimeResources(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme(t)
	runtimeResource := newTestRuntime()
	runtimeResource.UID = types.UID("runtime-uid")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(
			&runtimev1alpha1.TrussiumRuntime{},
		).
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
		WithStatusSubresource(
			&runtimev1alpha1.TrussiumRuntime{},
		).
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
		WithStatusSubresource(
			&runtimev1alpha1.TrussiumRuntime{},
		).
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
		WithStatusSubresource(
			&runtimev1alpha1.TrussiumRuntime{},
		).
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
		t.Fatalf("register core/v1 Kubernetes API: %v", err)
	}

	if err := appsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("register apps/v1 Kubernetes API: %v", err)
	}

	if err := policyv1.AddToScheme(scheme); err != nil {
		t.Fatalf("register policy/v1 Kubernetes API: %v", err)
	}

	if err := runtimev1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf(
			"register runtime.trussium.io/v1alpha1 API: %v",
			err,
		)
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

func TestReconcileUpdatesRuntimeStatus(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme(t)
	runtimeResource := newTestRuntime()
	runtimeResource.UID = types.UID("runtime-status-uid")
	runtimeResource.Generation = 3

	recorder := events.NewFakeRecorder(20)

	kubernetesClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(
			&runtimev1alpha1.TrussiumRuntime{},
		).
		WithObjects(runtimeResource).
		Build()

	reconciler := TrussiumRuntimeReconciler{
		Client:   kubernetesClient,
		Scheme:   scheme,
		Recorder: recorder,
	}

	request := ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(runtimeResource),
	}

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("reconcile TrussiumRuntime: %v", err)
	}

	var storedRuntime runtimev1alpha1.TrussiumRuntime
	if err := kubernetesClient.Get(
		ctx,
		request.NamespacedName,
		&storedRuntime,
	); err != nil {
		t.Fatalf("get reconciled TrussiumRuntime: %v", err)
	}

	if storedRuntime.Status.ObservedGeneration != 3 {
		t.Fatalf(
			"expected observed generation 3, received %d",
			storedRuntime.Status.ObservedGeneration,
		)
	}

	if storedRuntime.Status.CurrentImage != testRuntimeImage {
		t.Fatalf(
			"expected current image %q, received %q",
			testRuntimeImage,
			storedRuntime.Status.CurrentImage,
		)
	}

	expectedEndpoint :=
		testRuntimeEndpoint

	if storedRuntime.Status.Endpoint != expectedEndpoint {
		t.Fatalf(
			"expected endpoint %q, received %q",
			expectedEndpoint,
			storedRuntime.Status.Endpoint,
		)
	}

	assertRuntimeCondition(
		t,
		storedRuntime.Status,
		conditionTypeConfigurationValid,
		metav1.ConditionTrue,
		reasonReferencesResolved,
	)

	assertRuntimeCondition(
		t,
		storedRuntime.Status,
		conditionTypeProgressing,
		metav1.ConditionTrue,
		reasonDeploymentProgressing,
	)
}

func TestReconcileBlocksDeploymentForMissingProviderSecret(
	t *testing.T,
) {
	ctx := context.Background()
	scheme := newTestScheme(t)
	runtimeResource := newTestRuntime()
	runtimeResource.UID = types.UID("missing-secret-uid")
	runtimeResource.Generation = 2
	runtimeResource.Spec.Provider.CredentialsSecretRef =
		&runtimev1alpha1.SecretKeyReference{
			Name: testProviderCredentialSecret,
			Key:  testProviderCredentialKey,
		}

	recorder := events.NewFakeRecorder(20)

	kubernetesClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(
			&runtimev1alpha1.TrussiumRuntime{},
		).
		WithObjects(runtimeResource).
		Build()

	reconciler := TrussiumRuntimeReconciler{
		Client:   kubernetesClient,
		Scheme:   scheme,
		Recorder: recorder,
	}

	request := ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(runtimeResource),
	}

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf(
			"missing referenced Secret must be reported in status: %v",
			err,
		)
	}

	var deployment appsv1.Deployment
	err := kubernetesClient.Get(
		ctx,
		request.NamespacedName,
		&deployment,
	)
	if !apierrors.IsNotFound(err) {
		t.Fatalf(
			"expected Deployment not to be created, received %v",
			err,
		)
	}

	var storedRuntime runtimev1alpha1.TrussiumRuntime
	if err := kubernetesClient.Get(
		ctx,
		request.NamespacedName,
		&storedRuntime,
	); err != nil {
		t.Fatalf("get updated TrussiumRuntime: %v", err)
	}

	assertRuntimeCondition(
		t,
		storedRuntime.Status,
		conditionTypeConfigurationValid,
		metav1.ConditionFalse,
		reasonSecretNotFound,
	)

	assertRuntimeCondition(
		t,
		storedRuntime.Status,
		conditionTypeDegraded,
		metav1.ConditionTrue,
		reasonConfigurationInvalid,
	)

	event := receiveRecordedEvent(t, recorder)
	if !strings.Contains(
		event,
		eventReasonConfigurationInvalid,
	) {
		t.Fatalf(
			"expected ConfigurationInvalid Event, received %q",
			event,
		)
	}
}

func TestReconcileRecoversAfterReferencedSecretIsCreated(
	t *testing.T,
) {
	ctx := context.Background()
	scheme := newTestScheme(t)
	runtimeResource := newTestRuntime()
	runtimeResource.UID = types.UID("secret-recovery-uid")
	runtimeResource.Generation = 2
	runtimeResource.Spec.Provider.CredentialsSecretRef =
		&runtimev1alpha1.SecretKeyReference{
			Name: testProviderCredentialSecret,
			Key:  testProviderCredentialKey,
		}

	recorder := events.NewFakeRecorder(30)

	kubernetesClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(
			&runtimev1alpha1.TrussiumRuntime{},
		).
		WithObjects(runtimeResource).
		Build()

	reconciler := TrussiumRuntimeReconciler{
		Client:   kubernetesClient,
		Scheme:   scheme,
		Recorder: recorder,
	}

	request := ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(runtimeResource),
	}

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("initial invalid reconciliation: %v", err)
	}

	drainRecordedEvents(recorder)

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testProviderCredentialSecret,
			Namespace: runtimeResource.Namespace,
		},
	}

	if err := kubernetesClient.Create(ctx, secret); err != nil {
		t.Fatalf("create referenced Secret: %v", err)
	}

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("reconcile after Secret creation: %v", err)
	}

	var deployment appsv1.Deployment
	if err := kubernetesClient.Get(
		ctx,
		request.NamespacedName,
		&deployment,
	); err != nil {
		t.Fatalf(
			"expected Deployment after Secret recovery: %v",
			err,
		)
	}

	var storedRuntime runtimev1alpha1.TrussiumRuntime
	if err := kubernetesClient.Get(
		ctx,
		request.NamespacedName,
		&storedRuntime,
	); err != nil {
		t.Fatalf("get recovered TrussiumRuntime: %v", err)
	}

	assertRuntimeCondition(
		t,
		storedRuntime.Status,
		conditionTypeConfigurationValid,
		metav1.ConditionTrue,
		reasonReferencesResolved,
	)

	foundRecoveryEvent := false
	for _, event := range drainRecordedEvents(recorder) {
		if strings.Contains(
			event,
			eventReasonRuntimeRecovered,
		) {
			foundRecoveryEvent = true
			break
		}
	}

	if !foundRecoveryEvent {
		t.Fatal("expected RuntimeRecovered Event")
	}
}

func TestReconcileDoesNotEmitDuplicateTransitionEvents(
	t *testing.T,
) {
	ctx := context.Background()
	scheme := newTestScheme(t)
	runtimeResource := newTestRuntime()
	runtimeResource.UID = types.UID("event-idempotency-uid")
	runtimeResource.Generation = 1

	recorder := events.NewFakeRecorder(30)

	kubernetesClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(
			&runtimev1alpha1.TrussiumRuntime{},
		).
		WithObjects(runtimeResource).
		Build()

	reconciler := TrussiumRuntimeReconciler{
		Client:   kubernetesClient,
		Scheme:   scheme,
		Recorder: recorder,
	}

	request := ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(runtimeResource),
	}

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("first reconciliation: %v", err)
	}

	drainRecordedEvents(recorder)

	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("second reconciliation: %v", err)
	}

	eventsAfterSecondReconciliation :=
		drainRecordedEvents(recorder)

	if len(eventsAfterSecondReconciliation) != 0 {
		t.Fatalf(
			"expected no duplicate transition Events, received %#v",
			eventsAfterSecondReconciliation,
		)
	}
}

func TestMapSecretToRuntimes(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme(t)

	firstRuntime := newTestRuntime()
	firstRuntime.Name = "first"
	firstRuntime.Spec.Provider.CredentialsSecretRef =
		&runtimev1alpha1.SecretKeyReference{
			Name: testProviderCredentialSecret,
			Key:  testProviderCredentialKey,
		}

	secondRuntime := newTestRuntime()
	secondRuntime.Name = "second"
	secondRuntime.Spec.ImagePullSecrets =
		[]runtimev1alpha1.NamedReference{
			{Name: testImagePullSecret},
		}

	unrelatedRuntime := newTestRuntime()
	unrelatedRuntime.Name = "unrelated"

	kubernetesClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(
			&runtimev1alpha1.TrussiumRuntime{},
		).
		WithObjects(
			firstRuntime,
			secondRuntime,
			unrelatedRuntime,
		).
		Build()

	reconciler := TrussiumRuntimeReconciler{
		Client: kubernetesClient,
		Scheme: scheme,
	}

	requests := reconciler.mapSecretToRuntimes(
		ctx,
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      testProviderCredentialSecret,
				Namespace: testRuntimeNamespace,
			},
		},
	)

	if len(requests) != 1 {
		t.Fatalf(
			"expected one request, received %#v",
			requests,
		)
	}

	if requests[0].Name != firstRuntime.Name {
		t.Fatalf(
			"expected runtime %q, received %q",
			firstRuntime.Name,
			requests[0].Name,
		)
	}
}

func receiveRecordedEvent(
	t *testing.T,
	recorder *events.FakeRecorder,
) string {
	t.Helper()

	select {
	case event := <-recorder.Events:
		return event
	default:
		t.Fatal("expected a recorded Kubernetes Event")
		return ""
	}
}

func drainRecordedEvents(
	recorder *events.FakeRecorder,
) []string {
	recordedEvents := make([]string, 0)

	for {
		select {
		case event := <-recorder.Events:
			recordedEvents = append(
				recordedEvents,
				event,
			)
		default:
			return recordedEvents
		}
	}
}

func TestReconcileCreatesOwnedPodDisruptionBudget(
	t *testing.T,
) {
	ctx := context.Background()
	scheme := newTestScheme(t)

	runtimeResource := newTestRuntime()
	runtimeResource.UID =
		types.UID("runtime-pdb-owner-uid")

	kubernetesClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(
			&runtimev1alpha1.TrussiumRuntime{},
		).
		WithObjects(runtimeResource).
		Build()

	reconciler := TrussiumRuntimeReconciler{
		Client: kubernetesClient,
		Scheme: scheme,
	}

	request := ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(runtimeResource),
	}

	if _, err := reconciler.Reconcile(
		ctx,
		request,
	); err != nil {
		t.Fatalf(
			"reconcile TrussiumRuntime: %v",
			err,
		)
	}

	var pdb policyv1.PodDisruptionBudget

	if err := kubernetesClient.Get(
		ctx,
		request.NamespacedName,
		&pdb,
	); err != nil {
		t.Fatalf(
			"get managed PodDisruptionBudget: %v",
			err,
		)
	}

	if pdb.Spec.MaxUnavailable == nil ||
		pdb.Spec.MaxUnavailable.IntVal != 1 {
		t.Fatalf(
			"expected maxUnavailable=1, received %#v",
			pdb.Spec.MaxUnavailable,
		)
	}

	if len(pdb.OwnerReferences) != 1 {
		t.Fatalf(
			"expected one owner reference, received %#v",
			pdb.OwnerReferences,
		)
	}

	owner := pdb.OwnerReferences[0]

	if owner.UID != runtimeResource.UID {
		t.Fatalf(
			"expected owner UID %q, received %q",
			runtimeResource.UID,
			owner.UID,
		)
	}

	if owner.Controller == nil || !*owner.Controller {
		t.Fatal(
			"expected TrussiumRuntime to be the PDB controller owner",
		)
	}
}

func TestReconcileCorrectsPodDisruptionBudgetDrift(
	t *testing.T,
) {
	ctx := context.Background()
	scheme := newTestScheme(t)

	runtimeResource := newTestRuntime()
	runtimeResource.UID =
		types.UID("runtime-pdb-drift-uid")

	kubernetesClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(
			&runtimev1alpha1.TrussiumRuntime{},
		).
		WithObjects(runtimeResource).
		Build()

	reconciler := TrussiumRuntimeReconciler{
		Client: kubernetesClient,
		Scheme: scheme,
	}

	request := ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(runtimeResource),
	}

	if _, err := reconciler.Reconcile(
		ctx,
		request,
	); err != nil {
		t.Fatalf(
			"initial reconciliation: %v",
			err,
		)
	}

	var pdb policyv1.PodDisruptionBudget

	if err := kubernetesClient.Get(
		ctx,
		request.NamespacedName,
		&pdb,
	); err != nil {
		t.Fatalf(
			"get managed PodDisruptionBudget: %v",
			err,
		)
	}

	pdb.Spec.MaxUnavailable =
		ptr.To(intstr.FromInt32(2))

	if err := kubernetesClient.Update(
		ctx,
		&pdb,
	); err != nil {
		t.Fatalf(
			"introduce PodDisruptionBudget drift: %v",
			err,
		)
	}

	if _, err := reconciler.Reconcile(
		ctx,
		request,
	); err != nil {
		t.Fatalf(
			"reconcile PodDisruptionBudget drift: %v",
			err,
		)
	}

	var corrected policyv1.PodDisruptionBudget

	if err := kubernetesClient.Get(
		ctx,
		request.NamespacedName,
		&corrected,
	); err != nil {
		t.Fatalf(
			"get corrected PodDisruptionBudget: %v",
			err,
		)
	}

	if corrected.Spec.MaxUnavailable == nil ||
		corrected.Spec.MaxUnavailable.IntVal != 1 {
		t.Fatalf(
			"expected corrected maxUnavailable=1, received %#v",
			corrected.Spec.MaxUnavailable,
		)
	}
}

func TestReconcileRecreatesDeletedPodDisruptionBudget(
	t *testing.T,
) {
	ctx := context.Background()
	scheme := newTestScheme(t)

	runtimeResource := newTestRuntime()
	runtimeResource.UID =
		types.UID("runtime-pdb-recreate-uid")

	kubernetesClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(
			&runtimev1alpha1.TrussiumRuntime{},
		).
		WithObjects(runtimeResource).
		Build()

	reconciler := TrussiumRuntimeReconciler{
		Client: kubernetesClient,
		Scheme: scheme,
	}

	request := ctrl.Request{
		NamespacedName: client.ObjectKeyFromObject(runtimeResource),
	}

	if _, err := reconciler.Reconcile(
		ctx,
		request,
	); err != nil {
		t.Fatalf(
			"initial reconciliation: %v",
			err,
		)
	}

	var pdb policyv1.PodDisruptionBudget

	if err := kubernetesClient.Get(
		ctx,
		request.NamespacedName,
		&pdb,
	); err != nil {
		t.Fatalf(
			"get managed PodDisruptionBudget: %v",
			err,
		)
	}

	if err := kubernetesClient.Delete(
		ctx,
		&pdb,
	); err != nil {
		t.Fatalf(
			"delete managed PodDisruptionBudget: %v",
			err,
		)
	}

	if _, err := reconciler.Reconcile(
		ctx,
		request,
	); err != nil {
		t.Fatalf(
			"reconcile deleted PodDisruptionBudget: %v",
			err,
		)
	}

	var recreated policyv1.PodDisruptionBudget

	if err := kubernetesClient.Get(
		ctx,
		request.NamespacedName,
		&recreated,
	); err != nil {
		t.Fatalf(
			"expected PodDisruptionBudget recreation: %v",
			err,
		)
	}

	if recreated.Spec.MaxUnavailable == nil ||
		recreated.Spec.MaxUnavailable.IntVal != 1 {
		t.Fatalf(
			"unexpected recreated PDB: %#v",
			recreated.Spec,
		)
	}
}
