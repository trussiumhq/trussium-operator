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
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

func TestManagerReconcilesTrussiumRuntime(
	t *testing.T,
) {
	namespace := createTestNamespace(t)

	runtimeResource := createTestRuntime(
		t,
		namespace.Name,
	)

	key := client.ObjectKey{
		Name:      runtimeResource.Name,
		Namespace: runtimeResource.Namespace,
	}

	configMap := waitForObject(
		t,
		key,
		&corev1.ConfigMap{},
	)

	serviceAccount := waitForObject(
		t,
		key,
		&corev1.ServiceAccount{},
	)

	service := waitForObject(
		t,
		key,
		&corev1.Service{},
	)

	deployment := waitForObject(
		t,
		key,
		&appsv1.Deployment{},
	)

	podDisruptionBudget := waitForObject(
		t,
		key,
		&policyv1.PodDisruptionBudget{},
	)

	t.Run(
		"creates the managed Kubernetes resources",
		func(t *testing.T) {
			assertManagedResources(
				t,
				configMap,
				serviceAccount,
				service,
				deployment,
				podDisruptionBudget,
			)
		},
	)

	t.Run(
		"sets controller ownership on managed resources",
		func(t *testing.T) {
			assertManagedResourceOwnership(
				t,
				runtimeResource,
				configMap,
				serviceAccount,
				service,
				deployment,
				podDisruptionBudget,
			)
		},
	)

	t.Run(
		"persists observed state through the status subresource",
		func(t *testing.T) {
			assertRuntimeObservedStatus(
				t,
				key,
				namespace.Name,
			)
		},
	)
}

func assertManagedResources(
	t *testing.T,
	configMap *corev1.ConfigMap,
	serviceAccount *corev1.ServiceAccount,
	service *corev1.Service,
	deployment *appsv1.Deployment,
	podDisruptionBudget *policyv1.PodDisruptionBudget,
) {
	t.Helper()

	assertManagedConfigMap(
		t,
		configMap,
	)

	assertManagedServiceAccount(
		t,
		serviceAccount,
	)

	assertManagedService(
		t,
		service,
	)

	assertManagedDeployment(
		t,
		deployment,
	)

	assertManagedPodDisruptionBudget(
		t,
		podDisruptionBudget,
	)
}

func assertManagedConfigMap(
	t *testing.T,
	configMap *corev1.ConfigMap,
) {
	t.Helper()

	if got := configMap.Data["TRUSSIUM_ENVIRONMENT"]; got != "production" {
		t.Fatalf(
			"ConfigMap TRUSSIUM_ENVIRONMENT = %q, expected production",
			got,
		)
	}

	if got := configMap.Data["TRUSSIUM_RUNTIME__PORT"]; got != "9000" {
		t.Fatalf(
			"ConfigMap TRUSSIUM_RUNTIME__PORT = %q, expected 9000",
			got,
		)
	}

	if got := configMap.Data["TRUSSIUM_PROVIDER__NAME"]; got != "ollama" {
		t.Fatalf(
			"ConfigMap TRUSSIUM_PROVIDER__NAME = %q, expected ollama",
			got,
		)
	}
}

func assertManagedServiceAccount(
	t *testing.T,
	serviceAccount *corev1.ServiceAccount,
) {
	t.Helper()

	if serviceAccount.AutomountServiceAccountToken == nil {
		t.Fatal(
			"ServiceAccount automountServiceAccountToken must be set",
		)
	}

	if *serviceAccount.AutomountServiceAccountToken {
		t.Fatal(
			"ServiceAccount must disable automountServiceAccountToken",
		)
	}
}

func assertManagedService(
	t *testing.T,
	service *corev1.Service,
) {
	t.Helper()

	if service.Spec.Type != corev1.ServiceTypeClusterIP {
		t.Fatalf(
			"Service type = %q, expected ClusterIP",
			service.Spec.Type,
		)
	}

	if len(service.Spec.Ports) != 1 {
		t.Fatalf(
			"Service has %d ports, expected 1",
			len(service.Spec.Ports),
		)
	}

	if service.Spec.Ports[0].Port != 9000 {
		t.Fatalf(
			"Service port = %d, expected 9000",
			service.Spec.Ports[0].Port,
		)
	}
}

func assertManagedDeployment(
	t *testing.T,
	deployment *appsv1.Deployment,
) {
	t.Helper()

	if deployment.Spec.Replicas == nil {
		t.Fatal(
			"Deployment replicas must be set",
		)
	}

	if *deployment.Spec.Replicas != 1 {
		t.Fatalf(
			"Deployment replicas = %d, expected 1",
			*deployment.Spec.Replicas,
		)
	}

	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf(
			"Deployment has %d containers, expected 1",
			len(deployment.Spec.Template.Spec.Containers),
		)
	}

	container := deployment.Spec.Template.Spec.Containers[0]

	if container.Name != "trussium" {
		t.Fatalf(
			"runtime container name = %q, expected trussium",
			container.Name,
		)
	}

	expectedImage :=
		testRuntimeImageRepository + ":" + testRuntimeImageTag

	if container.Image != expectedImage {
		t.Fatalf(
			"runtime container image = %q, expected %q",
			container.Image,
			expectedImage,
		)
	}
}

func assertManagedPodDisruptionBudget(
	t *testing.T,
	podDisruptionBudget *policyv1.PodDisruptionBudget,
) {
	t.Helper()

	if podDisruptionBudget.Spec.Selector == nil {
		t.Fatal(
			"PodDisruptionBudget must define a selector",
		)
	}
}

func assertManagedResourceOwnership(
	t *testing.T,
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	configMap *corev1.ConfigMap,
	serviceAccount *corev1.ServiceAccount,
	service *corev1.Service,
	deployment *appsv1.Deployment,
	podDisruptionBudget *policyv1.PodDisruptionBudget,
) {
	t.Helper()

	assertControllerOwner(
		t,
		configMap,
		runtimeResource,
	)

	assertControllerOwner(
		t,
		serviceAccount,
		runtimeResource,
	)

	assertControllerOwner(
		t,
		service,
		runtimeResource,
	)

	assertControllerOwner(
		t,
		deployment,
		runtimeResource,
	)

	assertControllerOwner(
		t,
		podDisruptionBudget,
		runtimeResource,
	)
}

func assertRuntimeObservedStatus(
	t *testing.T,
	key client.ObjectKey,
	namespace string,
) {
	t.Helper()

	expectedImage :=
		testRuntimeImageRepository + ":" + testRuntimeImageTag

	expectedEndpoint :=
		"http://runtime." +
			namespace +
			".svc.cluster.local:9000"

	current := waitForRuntimeStatus(
		t,
		key,
		func(
			runtimeStatus *runtimev1alpha1.TrussiumRuntime,
		) bool {
			return runtimeStatus.Status.ObservedGeneration ==
				runtimeStatus.Generation &&
				runtimeStatus.Status.DesiredImage ==
					expectedImage &&
				runtimeStatus.Status.CurrentImage ==
					expectedImage &&
				runtimeStatus.Status.Endpoint ==
					expectedEndpoint &&
				runtimeCondition(
					runtimeStatus,
					"ConfigurationValid",
				) != nil
		},
	)

	assertRuntimeStatusValues(
		t,
		current,
		expectedImage,
		expectedEndpoint,
	)

	assertConfigurationValidCondition(
		t,
		current,
	)

	assertProgressingCondition(
		t,
		current,
	)

	assertReadyCondition(
		t,
		current,
	)
}

func assertRuntimeStatusValues(
	t *testing.T,
	current *runtimev1alpha1.TrussiumRuntime,
	expectedImage string,
	expectedEndpoint string,
) {
	t.Helper()

	if current.Status.ObservedGeneration != current.Generation {
		t.Fatalf(
			"status observedGeneration = %d, expected generation %d",
			current.Status.ObservedGeneration,
			current.Generation,
		)
	}

	if current.Status.DesiredImage != expectedImage {
		t.Fatalf(
			"status desiredImage = %q, expected %q",
			current.Status.DesiredImage,
			expectedImage,
		)
	}

	if current.Status.CurrentImage != expectedImage {
		t.Fatalf(
			"status currentImage = %q, expected %q",
			current.Status.CurrentImage,
			expectedImage,
		)
	}

	if current.Status.Endpoint != expectedEndpoint {
		t.Fatalf(
			"status endpoint = %q, expected %q",
			current.Status.Endpoint,
			expectedEndpoint,
		)
	}
}

func assertConfigurationValidCondition(
	t *testing.T,
	current *runtimev1alpha1.TrussiumRuntime,
) {
	t.Helper()

	condition := runtimeCondition(
		current,
		"ConfigurationValid",
	)

	if condition == nil {
		t.Fatal(
			"ConfigurationValid condition was not persisted",
		)
	}

	if condition.Status != metav1.ConditionTrue {
		t.Fatalf(
			"ConfigurationValid status = %q, expected True",
			condition.Status,
		)
	}

	if condition.Reason != "ReferencesResolved" {
		t.Fatalf(
			"ConfigurationValid reason = %q, expected ReferencesResolved",
			condition.Reason,
		)
	}

	if condition.ObservedGeneration != current.Generation {
		t.Fatalf(
			"ConfigurationValid observedGeneration = %d, expected %d",
			condition.ObservedGeneration,
			current.Generation,
		)
	}
}

func assertProgressingCondition(
	t *testing.T,
	current *runtimev1alpha1.TrussiumRuntime,
) {
	t.Helper()

	condition := runtimeCondition(
		current,
		"Progressing",
	)

	if condition == nil {
		t.Fatal(
			"Progressing condition was not persisted",
		)
	}

	if condition.Status != metav1.ConditionTrue {
		t.Fatalf(
			"Progressing status = %q, expected True",
			condition.Status,
		)
	}
}

func assertReadyCondition(
	t *testing.T,
	current *runtimev1alpha1.TrussiumRuntime,
) {
	t.Helper()

	condition := runtimeCondition(
		current,
		"Ready",
	)

	if condition == nil {
		t.Fatal(
			"Ready condition was not persisted",
		)
	}

	if condition.Status != metav1.ConditionFalse {
		t.Fatalf(
			"Ready status = %q, expected False",
			condition.Status,
		)
	}
}
