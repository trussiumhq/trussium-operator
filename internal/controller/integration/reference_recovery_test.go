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

package integration

import (
	"context"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

const (
	testProviderSecretName    = "provider-credentials"
	testProviderSecretKey     = "api-key"
	testImagePullSecretName   = "registry-credentials"
	configurationValidType    = "ConfigurationValid"
	secretNotFoundReason      = "SecretNotFound"
	referencesResolvedReason  = "ReferencesResolved"
	providerAPIKeyEnvironment = "TRUSSIUM_PROVIDER__API_KEY"
)

func TestProviderSecretReferenceRecovery(
	t *testing.T,
) {
	namespace := createTestNamespace(t)

	runtimeResource := createRuntimeWithSecretReferences(
		t,
		namespace.Name,
		&runtimev1alpha1.SecretKeyReference{
			Name: testProviderSecretName,
			Key:  testProviderSecretKey,
		},
		nil,
	)

	key := client.ObjectKeyFromObject(runtimeResource)
	initialGeneration := runtimeResource.Generation

	invalidRuntime := waitForConfigurationValidConditionState(
		t,
		key,
		metav1.ConditionFalse,
		secretNotFoundReason,
	)

	assertMissingSecretMessage(
		t,
		invalidRuntime,
		testProviderSecretName,
	)

	assertManagedDeploymentAbsent(
		t,
		key,
	)

	createTestSecret(
		t,
		namespace.Name,
		testProviderSecretName,
		corev1.SecretTypeOpaque,
		map[string][]byte{
			testProviderSecretKey: []byte("integration-test-api-key"),
		},
	)

	recoveredRuntime := waitForConfigurationValidConditionState(
		t,
		key,
		metav1.ConditionTrue,
		referencesResolvedReason,
	)

	assertRuntimeGenerationUnchanged(
		t,
		recoveredRuntime,
		initialGeneration,
	)

	deployment := waitForObject(
		t,
		key,
		&appsv1.Deployment{},
	)

	assertProviderSecretProjection(
		t,
		deployment,
		testProviderSecretName,
		testProviderSecretKey,
	)
}

func TestImagePullSecretReferenceRecovery(
	t *testing.T,
) {
	namespace := createTestNamespace(t)

	runtimeResource := createRuntimeWithSecretReferences(
		t,
		namespace.Name,
		nil,
		[]runtimev1alpha1.NamedReference{
			{
				Name: testImagePullSecretName,
			},
		},
	)

	key := client.ObjectKeyFromObject(runtimeResource)
	initialGeneration := runtimeResource.Generation

	invalidRuntime := waitForConfigurationValidConditionState(
		t,
		key,
		metav1.ConditionFalse,
		secretNotFoundReason,
	)

	assertMissingSecretMessage(
		t,
		invalidRuntime,
		testImagePullSecretName,
	)

	assertManagedDeploymentAbsent(
		t,
		key,
	)

	createTestSecret(
		t,
		namespace.Name,
		testImagePullSecretName,
		corev1.SecretTypeDockerConfigJson,
		map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(`{"auths":{}}`),
		},
	)

	recoveredRuntime := waitForConfigurationValidConditionState(
		t,
		key,
		metav1.ConditionTrue,
		referencesResolvedReason,
	)

	assertRuntimeGenerationUnchanged(
		t,
		recoveredRuntime,
		initialGeneration,
	)

	serviceAccount := waitForObject(
		t,
		key,
		&corev1.ServiceAccount{},
	)

	deployment := waitForObject(
		t,
		key,
		&appsv1.Deployment{},
	)

	assertImagePullSecretProjection(
		t,
		serviceAccount,
		deployment,
		testImagePullSecretName,
	)
}

func createRuntimeWithSecretReferences(
	t *testing.T,
	namespace string,
	credentialsSecretRef *runtimev1alpha1.SecretKeyReference,
	imagePullSecrets []runtimev1alpha1.NamedReference,
) *runtimev1alpha1.TrussiumRuntime {
	t.Helper()

	replicas := int32(1)
	tag := testRuntimeImageTag

	runtimeResource := &runtimev1alpha1.TrussiumRuntime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "runtime",
			Namespace: namespace,
		},
		Spec: runtimev1alpha1.TrussiumRuntimeSpec{
			Image: runtimev1alpha1.RuntimeImageSpec{
				Repository: testRuntimeImageRepository,
				Tag:        &tag,
				PullPolicy: corev1.PullIfNotPresent,
			},
			Replicas:         &replicas,
			ImagePullSecrets: imagePullSecrets,
			Provider: runtimev1alpha1.ProviderSpec{
				Type:                 runtimev1alpha1.ProviderTypeOpenAI,
				Model:                "gpt-4.1-mini",
				CredentialsSecretRef: credentialsSecretRef,
			},
			Service: runtimev1alpha1.RuntimeServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Port: 9000,
			},
		},
	}

	if err := testClient.Create(
		context.Background(),
		runtimeResource,
	); err != nil {
		t.Fatalf(
			"create TrussiumRuntime with Secret references: %v",
			err,
		)
	}

	return runtimeResource
}

func createTestSecret(
	t *testing.T,
	namespace string,
	name string,
	secretType corev1.SecretType,
	data map[string][]byte,
) {
	t.Helper()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: secretType,
		Data: data,
	}

	if err := testClient.Create(
		context.Background(),
		secret,
	); err != nil {
		t.Fatalf(
			"create Secret %s/%s: %v",
			namespace,
			name,
			err,
		)
	}
}

func waitForConfigurationValidConditionState(
	t *testing.T,
	key client.ObjectKey,
	status metav1.ConditionStatus,
	reason string,
) *runtimev1alpha1.TrussiumRuntime {
	t.Helper()

	return waitForRuntimeStatus(
		t,
		key,
		func(
			runtimeResource *runtimev1alpha1.TrussiumRuntime,
		) bool {
			condition := runtimeCondition(
				runtimeResource,
				configurationValidType,
			)

			return condition != nil &&
				condition.Status == status &&
				condition.Reason == reason
		},
	)
}

func assertMissingSecretMessage(
	t *testing.T,
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	secretName string,
) {
	t.Helper()

	condition := runtimeCondition(
		runtimeResource,
		configurationValidType,
	)
	if condition == nil {
		t.Fatal(
			"ConfigurationValid condition was not persisted",
		)
	}

	if !strings.Contains(
		condition.Message,
		secretName,
	) {
		t.Fatalf(
			"ConfigurationValid message %q does not reference missing Secret %q",
			condition.Message,
			secretName,
		)
	}
}

func assertManagedDeploymentAbsent(
	t *testing.T,
	key client.ObjectKey,
) {
	t.Helper()

	var deployment appsv1.Deployment

	err := testClient.Get(
		context.Background(),
		key,
		&deployment,
	)

	if err == nil {
		t.Fatalf(
			"Deployment %s unexpectedly exists while Secret reference is invalid",
			key.String(),
		)
	}

	if !apierrors.IsNotFound(err) {
		t.Fatalf(
			"get Deployment %s while checking absence: %v",
			key.String(),
			err,
		)
	}
}

func assertRuntimeGenerationUnchanged(
	t *testing.T,
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	expectedGeneration int64,
) {
	t.Helper()

	if runtimeResource.Generation != expectedGeneration {
		t.Fatalf(
			"TrussiumRuntime generation changed from %d to %d during Secret recovery",
			expectedGeneration,
			runtimeResource.Generation,
		)
	}

	if runtimeResource.Status.ObservedGeneration != expectedGeneration {
		t.Fatalf(
			"status observedGeneration = %d, expected %d",
			runtimeResource.Status.ObservedGeneration,
			expectedGeneration,
		)
	}
}

func assertProviderSecretProjection(
	t *testing.T,
	deployment *appsv1.Deployment,
	secretName string,
	secretKey string,
) {
	t.Helper()

	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf(
			"Deployment has %d containers, expected 1",
			len(deployment.Spec.Template.Spec.Containers),
		)
	}

	container := deployment.Spec.Template.Spec.Containers[0]

	for _, environmentVariable := range container.Env {
		if environmentVariable.Name != providerAPIKeyEnvironment {
			continue
		}

		if environmentVariable.ValueFrom == nil {
			t.Fatalf(
				"%s must use valueFrom",
				providerAPIKeyEnvironment,
			)
		}

		secretReference := environmentVariable.ValueFrom.SecretKeyRef
		if secretReference == nil {
			t.Fatalf(
				"%s must use secretKeyRef",
				providerAPIKeyEnvironment,
			)
		}

		if secretReference.Name != secretName {
			t.Fatalf(
				"%s Secret name = %q, expected %q",
				providerAPIKeyEnvironment,
				secretReference.Name,
				secretName,
			)
		}

		if secretReference.Key != secretKey {
			t.Fatalf(
				"%s Secret key = %q, expected %q",
				providerAPIKeyEnvironment,
				secretReference.Key,
				secretKey,
			)
		}

		return
	}

	t.Fatalf(
		"Deployment does not project provider credential environment variable %q",
		providerAPIKeyEnvironment,
	)
}

func assertImagePullSecretProjection(
	t *testing.T,
	serviceAccount *corev1.ServiceAccount,
	deployment *appsv1.Deployment,
	secretName string,
) {
	t.Helper()

	if !containsLocalObjectReference(
		serviceAccount.ImagePullSecrets,
		secretName,
	) {
		t.Fatalf(
			"ServiceAccount imagePullSecrets does not contain %q",
			secretName,
		)
	}

	if !containsLocalObjectReference(
		deployment.Spec.Template.Spec.ImagePullSecrets,
		secretName,
	) {
		t.Fatalf(
			"Deployment imagePullSecrets does not contain %q",
			secretName,
		)
	}
}

func containsLocalObjectReference(
	references []corev1.LocalObjectReference,
	name string,
) bool {
	for _, reference := range references {
		if reference.Name == name {
			return true
		}
	}

	return false
}
