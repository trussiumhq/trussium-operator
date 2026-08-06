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
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

func TestBuildRuntimeStatusReady(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	runtimeResource.Generation = 7

	deployment := buildDeployment(runtimeResource)
	deployment.Generation = 4
	deployment.Status.ObservedGeneration = 4
	deployment.Status.ReadyReplicas = 2
	deployment.Status.AvailableReplicas = 2

	service := buildService(runtimeResource)

	status := buildRuntimeStatus(
		runtimeResource,
		runtimeObservation{
			Deployment: deployment,
			Service:    service,
		},
		validReferenceValidation(),
	)

	if status.ObservedGeneration != 7 {
		t.Fatalf(
			"expected observed generation 7, received %d",
			status.ObservedGeneration,
		)
	}

	if status.ReadyReplicas != 2 {
		t.Fatalf(
			"expected two ready replicas, received %d",
			status.ReadyReplicas,
		)
	}

	if status.AvailableReplicas != 2 {
		t.Fatalf(
			"expected two available replicas, received %d",
			status.AvailableReplicas,
		)
	}

	if status.CurrentImage != testRuntimeImage {
		t.Fatalf(
			"expected image %q, received %q",
			testRuntimeImage,
			status.CurrentImage,
		)
	}

	expectedEndpoint :=
		testRuntimeEndpoint
	if status.Endpoint != expectedEndpoint {
		t.Fatalf(
			"expected endpoint %q, received %q",
			expectedEndpoint,
			status.Endpoint,
		)
	}

	assertRuntimeCondition(
		t,
		status,
		conditionTypeConfigurationValid,
		metav1.ConditionTrue,
		reasonReferencesResolved,
	)
	assertRuntimeCondition(
		t,
		status,
		conditionTypeProgressing,
		metav1.ConditionFalse,
		reasonReconciliationComplete,
	)
	assertRuntimeCondition(
		t,
		status,
		conditionTypeAvailable,
		metav1.ConditionTrue,
		reasonRuntimeAvailable,
	)
	assertRuntimeCondition(
		t,
		status,
		conditionTypeReady,
		metav1.ConditionTrue,
		reasonRuntimeReady,
	)
	assertRuntimeCondition(
		t,
		status,
		conditionTypeDegraded,
		metav1.ConditionFalse,
		reasonAsExpected,
	)
}

func TestBuildRuntimeStatusProgressing(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	runtimeResource.Generation = 3

	deployment := buildDeployment(runtimeResource)
	deployment.Generation = 2
	deployment.Status.ObservedGeneration = 1
	deployment.Status.ReadyReplicas = 1
	deployment.Status.AvailableReplicas = 1

	status := buildRuntimeStatus(
		runtimeResource,
		runtimeObservation{
			Deployment: deployment,
			Service:    buildService(runtimeResource),
		},
		validReferenceValidation(),
	)

	assertRuntimeCondition(
		t,
		status,
		conditionTypeProgressing,
		metav1.ConditionTrue,
		reasonDeploymentGenerationPending,
	)
	assertRuntimeCondition(
		t,
		status,
		conditionTypeAvailable,
		metav1.ConditionTrue,
		reasonRuntimeAvailable,
	)
	assertRuntimeCondition(
		t,
		status,
		conditionTypeReady,
		metav1.ConditionFalse,
		reasonRuntimeNotReady,
	)
	assertRuntimeCondition(
		t,
		status,
		conditionTypeDegraded,
		metav1.ConditionFalse,
		reasonAsExpected,
	)
}

func TestBuildRuntimeStatusProgressDeadlineExceeded(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	runtimeResource.Generation = 4

	deployment := buildDeployment(runtimeResource)
	deployment.Generation = 2
	deployment.Status.ObservedGeneration = 2
	deployment.Status.Conditions = []appsv1.DeploymentCondition{
		{
			Type:    appsv1.DeploymentProgressing,
			Status:  corev1.ConditionFalse,
			Reason:  reasonProgressDeadlineExceeded,
			Message: "Deployment exceeded its progress deadline.",
		},
	}

	status := buildRuntimeStatus(
		runtimeResource,
		runtimeObservation{
			Deployment: deployment,
			Service:    buildService(runtimeResource),
		},
		validReferenceValidation(),
	)

	assertRuntimeCondition(
		t,
		status,
		conditionTypeProgressing,
		metav1.ConditionFalse,
		reasonDeploymentFailed,
	)
	assertRuntimeCondition(
		t,
		status,
		conditionTypeReady,
		metav1.ConditionFalse,
		reasonDeploymentFailed,
	)
	assertRuntimeCondition(
		t,
		status,
		conditionTypeDegraded,
		metav1.ConditionTrue,
		reasonProgressDeadlineExceeded,
	)
}

func TestBuildRuntimeStatusScaledToZero(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	runtimeResource.Generation = 5
	runtimeResource.Spec.Replicas = ptr.To(int32(0))

	deployment := buildDeployment(runtimeResource)
	deployment.Generation = 3
	deployment.Status.ObservedGeneration = 3

	status := buildRuntimeStatus(
		runtimeResource,
		runtimeObservation{
			Deployment: deployment,
			Service:    buildService(runtimeResource),
		},
		validReferenceValidation(),
	)

	assertRuntimeCondition(
		t,
		status,
		conditionTypeProgressing,
		metav1.ConditionFalse,
		reasonScaledToZero,
	)
	assertRuntimeCondition(
		t,
		status,
		conditionTypeAvailable,
		metav1.ConditionFalse,
		reasonScaledToZero,
	)
	assertRuntimeCondition(
		t,
		status,
		conditionTypeReady,
		metav1.ConditionTrue,
		reasonScaledToZero,
	)
	assertRuntimeCondition(
		t,
		status,
		conditionTypeDegraded,
		metav1.ConditionFalse,
		reasonAsExpected,
	)
}

func TestBuildRuntimeStatusInvalidConfiguration(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	runtimeResource.Generation = 2

	status := buildRuntimeStatus(
		runtimeResource,
		runtimeObservation{
			Service: buildService(runtimeResource),
		},
		referenceValidationResult{
			Valid:   false,
			Reason:  reasonSecretNotFound,
			Message: `Referenced Secret testProviderCredentialSecret was not found.`,
		},
	)

	assertRuntimeCondition(
		t,
		status,
		conditionTypeConfigurationValid,
		metav1.ConditionFalse,
		reasonSecretNotFound,
	)
	assertRuntimeCondition(
		t,
		status,
		conditionTypeProgressing,
		metav1.ConditionFalse,
		reasonConfigurationInvalid,
	)
	assertRuntimeCondition(
		t,
		status,
		conditionTypeReady,
		metav1.ConditionFalse,
		reasonConfigurationInvalid,
	)
	assertRuntimeCondition(
		t,
		status,
		conditionTypeDegraded,
		metav1.ConditionTrue,
		reasonConfigurationInvalid,
	)
}

func TestConditionTransitionTimeIsPreserved(t *testing.T) {
	t.Parallel()

	transitionTime := metav1.NewTime(
		time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC),
	)

	runtimeResource := newTestRuntime()
	runtimeResource.Generation = 2
	runtimeResource.Status.Conditions = []metav1.Condition{
		{
			Type:               conditionTypeReady,
			Status:             metav1.ConditionFalse,
			ObservedGeneration: 1,
			Reason:             reasonRuntimeNotReady,
			Message:            "The runtime is not ready.",
			LastTransitionTime: transitionTime,
		},
	}

	deployment := buildDeployment(runtimeResource)
	deployment.Generation = 2
	deployment.Status.ObservedGeneration = 1

	status := buildRuntimeStatus(
		runtimeResource,
		runtimeObservation{
			Deployment: deployment,
			Service:    buildService(runtimeResource),
		},
		validReferenceValidation(),
	)

	readyCondition := runtimeCondition(
		status,
		conditionTypeReady,
	)
	if readyCondition == nil {
		t.Fatal("Ready condition was not created")
	}

	if !readyCondition.LastTransitionTime.Equal(&transitionTime) {
		t.Fatalf(
			"expected transition time %s, received %s",
			transitionTime,
			readyCondition.LastTransitionTime,
		)
	}
}

func TestValidateSecretReferences(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	scheme := newTestScheme(t)
	runtimeResource := newTestRuntime()

	runtimeResource.Spec.Provider.CredentialsSecretRef =
		&runtimev1alpha1.SecretKeyReference{
			Name: testProviderCredentialSecret,
			Key:  testProviderCredentialKey,
		}
	runtimeResource.Spec.ImagePullSecrets =
		[]runtimev1alpha1.NamedReference{
			{Name: testImagePullSecret},
		}

	kubernetesClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(
			&runtimev1alpha1.TrussiumRuntime{},
		).
		WithObjects(
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testProviderCredentialSecret,
					Namespace: runtimeResource.Namespace,
				},
			},
			&corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      testImagePullSecret,
					Namespace: runtimeResource.Namespace,
				},
			},
		).
		Build()

	reconciler := TrussiumRuntimeReconciler{
		Client: kubernetesClient,
		Scheme: scheme,
	}

	result := reconciler.validateSecretReferences(
		ctx,
		runtimeResource,
	)

	if !result.Valid {
		t.Fatalf(
			"expected valid Secret references, received %#v",
			result,
		)
	}

	if result.Err != nil {
		t.Fatalf(
			"expected no validation error, received %v",
			result.Err,
		)
	}
}

func TestValidateSecretReferencesReportsMissingSecret(
	t *testing.T,
) {
	t.Parallel()

	ctx := context.Background()
	scheme := newTestScheme(t)
	runtimeResource := newTestRuntime()

	runtimeResource.Spec.Provider.CredentialsSecretRef =
		&runtimev1alpha1.SecretKeyReference{
			Name: "missing-provider-credentials",
			Key:  testProviderCredentialKey,
		}

	reconciler := TrussiumRuntimeReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			Build(),
		Scheme: scheme,
	}

	result := reconciler.validateSecretReferences(
		ctx,
		runtimeResource,
	)

	if result.Valid {
		t.Fatal("expected missing Secret validation to fail")
	}

	if result.Reason != reasonSecretNotFound {
		t.Fatalf(
			"expected reason %q, received %q",
			reasonSecretNotFound,
			result.Reason,
		)
	}

	if result.Err != nil {
		t.Fatalf(
			"expected missing Secret to be a status result, received %v",
			result.Err,
		)
	}
}

func TestRuntimeReferencesSecret(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	runtimeResource.Spec.Provider.CredentialsSecretRef =
		&runtimev1alpha1.SecretKeyReference{
			Name: testProviderCredentialSecret,
			Key:  testProviderCredentialKey,
		}
	runtimeResource.Spec.ImagePullSecrets =
		[]runtimev1alpha1.NamedReference{
			{Name: testImagePullSecret},
		}

	if !runtimeReferencesSecret(
		runtimeResource,
		testProviderCredentialSecret,
	) {
		t.Fatal("expected provider Secret to be referenced")
	}

	if !runtimeReferencesSecret(
		runtimeResource,
		testImagePullSecret,
	) {
		t.Fatal("expected image-pull Secret to be referenced")
	}

	if runtimeReferencesSecret(
		runtimeResource,
		"unrelated-secret",
	) {
		t.Fatal("unrelated Secret must not match")
	}
}

func assertRuntimeCondition(
	t *testing.T,
	status runtimev1alpha1.TrussiumRuntimeStatus,
	conditionType string,
	expectedStatus metav1.ConditionStatus,
	expectedReason string,
) {
	t.Helper()

	condition := runtimeCondition(status, conditionType)
	if condition == nil {
		t.Fatalf(
			"condition %q was not found in %#v",
			conditionType,
			status.Conditions,
		)
	}

	if condition.Status != expectedStatus {
		t.Fatalf(
			"expected condition %q status %q, received %q",
			conditionType,
			expectedStatus,
			condition.Status,
		)
	}

	if condition.Reason != expectedReason {
		t.Fatalf(
			"expected condition %q reason %q, received %q",
			conditionType,
			expectedReason,
			condition.Reason,
		)
	}
}

func TestConditionChanged(t *testing.T) {
	t.Parallel()

	previousStatus := runtimev1alpha1.TrussiumRuntimeStatus{}
	setRuntimeCondition(
		&previousStatus,
		1,
		conditionTypeReady,
		metav1.ConditionFalse,
		reasonRuntimeNotReady,
		"The runtime is not ready.",
	)

	unchangedStatus := copyRuntimeStatus(previousStatus)
	setRuntimeCondition(
		&unchangedStatus,
		2,
		conditionTypeReady,
		metav1.ConditionFalse,
		reasonRuntimeNotReady,
		"The runtime is still not ready.",
	)

	if conditionChanged(
		previousStatus,
		unchangedStatus,
		conditionTypeReady,
	) {
		t.Fatal(
			"condition must not be considered changed when status and reason are unchanged",
		)
	}

	changedStatus := copyRuntimeStatus(previousStatus)
	setRuntimeCondition(
		&changedStatus,
		2,
		conditionTypeReady,
		metav1.ConditionTrue,
		reasonRuntimeReady,
		testRuntimeReadyMessage,
	)

	if !conditionChanged(
		previousStatus,
		changedStatus,
		conditionTypeReady,
	) {
		t.Fatal(
			"condition must be considered changed when status changes",
		)
	}
}

func TestLoadRuntimeObservation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	scheme := newTestScheme(t)
	runtimeResource := newTestRuntime()

	deployment := buildDeployment(runtimeResource)
	service := buildService(runtimeResource)

	kubernetesClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(
			&runtimev1alpha1.TrussiumRuntime{},
		).
		WithObjects(
			runtimeResource,
			deployment,
			service,
		).
		Build()

	reconciler := TrussiumRuntimeReconciler{
		Client: kubernetesClient,
		Scheme: scheme,
	}

	observation, err := reconciler.loadRuntimeObservation(
		ctx,
		runtimeResource,
	)
	if err != nil {
		t.Fatalf("load runtime observation: %v", err)
	}

	if observation.Deployment == nil {
		t.Fatal("expected the managed Deployment to be observed")
	}

	if observation.Service == nil {
		t.Fatal("expected the managed Service to be observed")
	}

	actualImage := deploymentCurrentImage(observation.Deployment)
	if actualImage != testRuntimeImage {
		t.Fatalf(
			"expected observed image %q, received %q",
			testRuntimeImage,
			actualImage,
		)
	}

	expectedEndpoint :=
		testRuntimeEndpoint

	actualEndpoint := runtimeServiceEndpoint(observation.Service)
	if actualEndpoint != expectedEndpoint {
		t.Fatalf(
			"expected observed endpoint %q, received %q",
			expectedEndpoint,
			actualEndpoint,
		)
	}
}

func TestLoadRuntimeObservationAllowsMissingManagedResources(
	t *testing.T,
) {
	t.Parallel()

	ctx := context.Background()
	scheme := newTestScheme(t)
	runtimeResource := newTestRuntime()

	reconciler := TrussiumRuntimeReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(runtimeResource).
			Build(),
		Scheme: scheme,
	}

	observation, err := reconciler.loadRuntimeObservation(
		ctx,
		runtimeResource,
	)
	if err != nil {
		t.Fatalf("load missing runtime observation: %v", err)
	}

	if observation.Deployment != nil {
		t.Fatal("expected no managed Deployment observation")
	}

	if observation.Service != nil {
		t.Fatal("expected no managed Service observation")
	}
}

func TestUpdateRuntimeStatus(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	scheme := newTestScheme(t)
	runtimeResource := newTestRuntime()
	runtimeResource.Generation = 6

	kubernetesClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(
			&runtimev1alpha1.TrussiumRuntime{},
		).
		WithStatusSubresource(
			&runtimev1alpha1.TrussiumRuntime{},
		).
		WithObjects(runtimeResource).
		Build()

	reconciler := TrussiumRuntimeReconciler{
		Client: kubernetesClient,
		Scheme: scheme,
	}

	var storedRuntime runtimev1alpha1.TrussiumRuntime
	if err := kubernetesClient.Get(
		ctx,
		client.ObjectKeyFromObject(runtimeResource),
		&storedRuntime,
	); err != nil {
		t.Fatalf("get stored TrussiumRuntime: %v", err)
	}

	desiredStatus := runtimev1alpha1.TrussiumRuntimeStatus{
		ObservedGeneration: 6,
		ReadyReplicas:      2,
		AvailableReplicas:  2,
		CurrentImage:       testRuntimeImage,
		Endpoint:           testRuntimeEndpoint,
	}

	setRuntimeCondition(
		&desiredStatus,
		6,
		conditionTypeReady,
		metav1.ConditionTrue,
		reasonRuntimeReady,
		testRuntimeReadyMessage,
	)

	updated, err := reconciler.updateRuntimeStatus(
		ctx,
		&storedRuntime,
		desiredStatus,
	)
	if err != nil {
		t.Fatalf("update TrussiumRuntime status: %v", err)
	}

	if !updated {
		t.Fatal("expected the status subresource to be updated")
	}

	var updatedRuntime runtimev1alpha1.TrussiumRuntime
	if err := kubernetesClient.Get(
		ctx,
		client.ObjectKeyFromObject(runtimeResource),
		&updatedRuntime,
	); err != nil {
		t.Fatalf("get updated TrussiumRuntime: %v", err)
	}

	if updatedRuntime.Status.ObservedGeneration != 6 {
		t.Fatalf(
			"expected observed generation 6, received %d",
			updatedRuntime.Status.ObservedGeneration,
		)
	}

	if updatedRuntime.Status.CurrentImage != testRuntimeImage {
		t.Fatalf(
			"expected current image %q, received %q",
			testRuntimeImage,
			updatedRuntime.Status.CurrentImage,
		)
	}

	updatedAgain, err := reconciler.updateRuntimeStatus(
		ctx,
		&updatedRuntime,
		desiredStatus,
	)
	if err != nil {
		t.Fatalf("repeat no-op status update: %v", err)
	}

	if updatedAgain {
		t.Fatal("expected unchanged status write to be skipped")
	}
}

func TestPreserveRuntimeConditionTransitionTimes(t *testing.T) {
	t.Parallel()

	storedTransitionTime := metav1.NewTime(
		time.Date(
			2026,
			time.August,
			6,
			18,
			30,
			0,
			0,
			time.UTC,
		),
	)

	desiredTransitionTime := metav1.NewTime(
		time.Date(
			2026,
			time.August,
			6,
			18,
			30,
			0,
			987654321,
			time.UTC,
		),
	)

	currentStatus := runtimev1alpha1.TrussiumRuntimeStatus{
		Conditions: []metav1.Condition{
			{
				Type:               conditionTypeReady,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: 4,
				Reason:             reasonRuntimeReady,
				Message:            testRuntimeReadyMessage,
				LastTransitionTime: storedTransitionTime,
			},
		},
	}

	desiredStatus := runtimev1alpha1.TrussiumRuntimeStatus{
		Conditions: []metav1.Condition{
			{
				Type:               conditionTypeReady,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: 4,
				Reason:             reasonRuntimeReady,
				Message:            testRuntimeReadyMessage,
				LastTransitionTime: desiredTransitionTime,
			},
		},
	}

	normalizedStatus := preserveRuntimeConditionTransitionTimes(
		currentStatus,
		desiredStatus,
	)

	readyCondition := runtimeCondition(
		normalizedStatus,
		conditionTypeReady,
	)
	if readyCondition == nil {
		t.Fatal("expected Ready condition")
	}

	if !readyCondition.LastTransitionTime.Equal(
		&storedTransitionTime,
	) {
		t.Fatalf(
			"expected stored transition time %s, received %s",
			storedTransitionTime,
			readyCondition.LastTransitionTime,
		)
	}
}

func TestPreserveRuntimeConditionTransitionTimesOnStatusChange(
	t *testing.T,
) {
	t.Parallel()

	storedTransitionTime := metav1.NewTime(
		time.Date(
			2026,
			time.August,
			6,
			18,
			30,
			0,
			0,
			time.UTC,
		),
	)

	desiredTransitionTime := metav1.NewTime(
		time.Date(
			2026,
			time.August,
			6,
			18,
			35,
			0,
			0,
			time.UTC,
		),
	)

	currentStatus := runtimev1alpha1.TrussiumRuntimeStatus{
		Conditions: []metav1.Condition{
			{
				Type:               conditionTypeReady,
				Status:             metav1.ConditionFalse,
				ObservedGeneration: 3,
				Reason:             reasonRuntimeNotReady,
				Message:            "The runtime is not ready.",
				LastTransitionTime: storedTransitionTime,
			},
		},
	}

	desiredStatus := runtimev1alpha1.TrussiumRuntimeStatus{
		Conditions: []metav1.Condition{
			{
				Type:               conditionTypeReady,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: 4,
				Reason:             reasonRuntimeReady,
				Message:            testRuntimeReadyMessage,
				LastTransitionTime: desiredTransitionTime,
			},
		},
	}

	normalizedStatus := preserveRuntimeConditionTransitionTimes(
		currentStatus,
		desiredStatus,
	)

	readyCondition := runtimeCondition(
		normalizedStatus,
		conditionTypeReady,
	)
	if readyCondition == nil {
		t.Fatal("expected Ready condition")
	}

	if !readyCondition.LastTransitionTime.Equal(
		&desiredTransitionTime,
	) {
		t.Fatalf(
			"expected new transition time %s, received %s",
			desiredTransitionTime,
			readyCondition.LastTransitionTime,
		)
	}
}
