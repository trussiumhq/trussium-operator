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
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

const (
	runtimeNameLabel      = "app.kubernetes.io/name"
	runtimeInstanceLabel  = "app.kubernetes.io/instance"
	runtimeManagedByLabel = "app.kubernetes.io/managed-by"

	expectedRuntimeName      = "trussium"
	expectedRuntimeManagedBy = "trussium-operator"

	driftedLabelKey   = "drifted"
	driftedLabelValue = "true"
)

func TestManagedResourceDriftCorrection(
	t *testing.T,
) {
	namespace := createTestNamespace(t)
	runtimeResource := createTestRuntime(
		t,
		namespace.Name,
	)

	key := client.ObjectKeyFromObject(runtimeResource)
	initialGeneration := runtimeResource.Generation

	service := waitForObject(
		t,
		key,
		&corev1.Service{},
	)

	waitForObject(
		t,
		key,
		&appsv1.Deployment{},
	)

	waitForObject(
		t,
		key,
		&policyv1.PodDisruptionBudget{},
	)

	t.Run(
		"corrects Service drift",
		func(t *testing.T) {
			driftService(
				t,
				service,
			)

			assertServiceRestored(
				t,
				key,
				runtimeResource.Name,
			)
		},
	)

	t.Run(
		"corrects Deployment drift",
		func(t *testing.T) {
			current := waitForObject(
				t,
				key,
				&appsv1.Deployment{},
			)

			driftDeployment(
				t,
				current,
			)

			assertDeploymentRestored(
				t,
				key,
			)
		},
	)

	t.Run(
		"corrects PodDisruptionBudget drift",
		func(t *testing.T) {
			current := waitForObject(
				t,
				key,
				&policyv1.PodDisruptionBudget{},
			)

			driftPodDisruptionBudget(
				t,
				current,
			)

			assertPodDisruptionBudgetRestored(
				t,
				key,
				runtimeResource.Name,
			)
		},
	)

	assertRuntimeGenerationPreserved(
		t,
		key,
		initialGeneration,
	)
}

func TestManagedResourceRecreation(
	t *testing.T,
) {
	namespace := createTestNamespace(t)
	runtimeResource := createTestRuntime(
		t,
		namespace.Name,
	)

	key := client.ObjectKeyFromObject(runtimeResource)
	initialGeneration := runtimeResource.Generation

	t.Run(
		"recreates deleted Service",
		func(t *testing.T) {
			service := waitForObject(
				t,
				key,
				&corev1.Service{},
			)

			oldUID := service.UID

			deleteManagedObject(
				t,
				service,
			)

			recreated := waitForRecreatedObject(
				t,
				key,
				&corev1.Service{},
				oldUID,
			)

			assertControllerOwner(
				t,
				recreated,
				runtimeResource,
			)

			assertServiceDesiredState(
				t,
				recreated,
				runtimeResource.Name,
			)
		},
	)

	t.Run(
		"recreates deleted Deployment",
		func(t *testing.T) {
			deployment := waitForObject(
				t,
				key,
				&appsv1.Deployment{},
			)

			oldUID := deployment.UID

			deleteManagedObject(
				t,
				deployment,
			)

			recreated := waitForRecreatedObject(
				t,
				key,
				&appsv1.Deployment{},
				oldUID,
			)

			assertControllerOwner(
				t,
				recreated,
				runtimeResource,
			)

			assertDeploymentDesiredState(
				t,
				recreated,
			)
		},
	)

	t.Run(
		"recreates deleted PodDisruptionBudget",
		func(t *testing.T) {
			podDisruptionBudget := waitForObject(
				t,
				key,
				&policyv1.PodDisruptionBudget{},
			)

			oldUID := podDisruptionBudget.UID

			deleteManagedObject(
				t,
				podDisruptionBudget,
			)

			recreated := waitForRecreatedObject(
				t,
				key,
				&policyv1.PodDisruptionBudget{},
				oldUID,
			)

			assertControllerOwner(
				t,
				recreated,
				runtimeResource,
			)

			assertPodDisruptionBudgetDesiredState(
				t,
				recreated,
				runtimeResource.Name,
			)
		},
	)

	assertRuntimeGenerationPreserved(
		t,
		key,
		initialGeneration,
	)
}

func driftService(
	t *testing.T,
	service *corev1.Service,
) {
	t.Helper()

	current := service.DeepCopy()

	current.Labels = map[string]string{
		driftedLabelKey: driftedLabelValue,
	}

	current.Spec.Selector = map[string]string{
		driftedLabelKey: driftedLabelValue,
	}

	if len(current.Spec.Ports) == 0 {
		t.Fatal(
			"Service has no ports to drift",
		)
	}

	current.Spec.Ports[0].Port = 9999

	if err := testClient.Update(
		context.Background(),
		current,
	); err != nil {
		t.Fatalf(
			"introduce Service drift: %v",
			err,
		)
	}
}

func driftDeployment(
	t *testing.T,
	deployment *appsv1.Deployment,
) {
	t.Helper()

	current := deployment.DeepCopy()

	replicas := int32(7)
	current.Spec.Replicas = &replicas

	current.Labels = map[string]string{
		driftedLabelKey: driftedLabelValue,
	}

	if len(current.Spec.Template.Spec.Containers) == 0 {
		t.Fatal(
			"Deployment has no containers to drift",
		)
	}

	current.Spec.Template.Spec.Containers[0].Image =
		"invalid.example/drifted:latest"

	if err := testClient.Update(
		context.Background(),
		current,
	); err != nil {
		t.Fatalf(
			"introduce Deployment drift: %v",
			err,
		)
	}
}

func driftPodDisruptionBudget(
	t *testing.T,
	podDisruptionBudget *policyv1.PodDisruptionBudget,
) {
	t.Helper()

	current := podDisruptionBudget.DeepCopy()

	maxUnavailable := intstr.FromInt32(0)
	current.Spec.MaxUnavailable = &maxUnavailable

	current.Labels = map[string]string{
		driftedLabelKey: driftedLabelValue,
	}

	current.Spec.Selector.MatchLabels = map[string]string{
		driftedLabelKey: driftedLabelValue,
	}

	if err := testClient.Update(
		context.Background(),
		current,
	); err != nil {
		t.Fatalf(
			"introduce PodDisruptionBudget drift: %v",
			err,
		)
	}
}

func assertServiceRestored(
	t *testing.T,
	key client.ObjectKey,
	runtimeName string,
) {
	t.Helper()

	eventually(
		t,
		"Service desired state restored",
		func() (bool, error) {
			var service corev1.Service

			if err := testClient.Get(
				context.Background(),
				key,
				&service,
			); err != nil {
				return false, err
			}

			return serviceMatchesDesiredState(
				&service,
				runtimeName,
			), nil
		},
	)
}

func assertDeploymentRestored(
	t *testing.T,
	key client.ObjectKey,
) {
	t.Helper()

	eventually(
		t,
		"Deployment desired state restored",
		func() (bool, error) {
			var deployment appsv1.Deployment

			if err := testClient.Get(
				context.Background(),
				key,
				&deployment,
			); err != nil {
				return false, err
			}

			return deploymentMatchesDesiredState(
				&deployment,
			), nil
		},
	)
}

func assertPodDisruptionBudgetRestored(
	t *testing.T,
	key client.ObjectKey,
	runtimeName string,
) {
	t.Helper()

	eventually(
		t,
		"PodDisruptionBudget desired state restored",
		func() (bool, error) {
			var podDisruptionBudget policyv1.PodDisruptionBudget

			if err := testClient.Get(
				context.Background(),
				key,
				&podDisruptionBudget,
			); err != nil {
				return false, err
			}

			return podDisruptionBudgetMatchesDesiredState(
				&podDisruptionBudget,
				runtimeName,
			), nil
		},
	)
}

func serviceMatchesDesiredState(
	service *corev1.Service,
	runtimeName string,
) bool {
	if service.Labels[runtimeNameLabel] != expectedRuntimeName {
		return false
	}

	if service.Labels[runtimeInstanceLabel] != runtimeName {
		return false
	}

	if service.Labels[runtimeManagedByLabel] !=
		expectedRuntimeManagedBy {
		return false
	}

	if service.Spec.Type != corev1.ServiceTypeClusterIP {
		return false
	}

	if service.Spec.Selector[runtimeNameLabel] !=
		expectedRuntimeName {
		return false
	}

	if service.Spec.Selector[runtimeInstanceLabel] != runtimeName {
		return false
	}

	if len(service.Spec.Ports) != 1 {
		return false
	}

	if service.Spec.Ports[0].Port != 9000 {
		return false
	}

	if service.Spec.Ports[0].TargetPort !=
		intstr.FromString("http") {
		return false
	}

	return true
}

func deploymentMatchesDesiredState(
	deployment *appsv1.Deployment,
) bool {
	if deployment.Labels[runtimeNameLabel] !=
		expectedRuntimeName {
		return false
	}

	if deployment.Labels[runtimeManagedByLabel] !=
		expectedRuntimeManagedBy {
		return false
	}

	if deployment.Spec.Replicas == nil ||
		*deployment.Spec.Replicas != 1 {
		return false
	}

	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		return false
	}

	container := deployment.Spec.Template.Spec.Containers[0]

	expectedImage :=
		testRuntimeImageRepository + ":" + testRuntimeImageTag

	if container.Name != "trussium" {
		return false
	}

	if container.Image != expectedImage {
		return false
	}

	return true
}

func podDisruptionBudgetMatchesDesiredState(
	podDisruptionBudget *policyv1.PodDisruptionBudget,
	runtimeName string,
) bool {
	if podDisruptionBudget.Labels[runtimeNameLabel] !=
		expectedRuntimeName {
		return false
	}

	if podDisruptionBudget.Labels[runtimeManagedByLabel] !=
		expectedRuntimeManagedBy {
		return false
	}

	if podDisruptionBudget.Spec.MaxUnavailable == nil {
		return false
	}

	expectedMaxUnavailable := intstr.FromInt32(1)
	if *podDisruptionBudget.Spec.MaxUnavailable !=
		expectedMaxUnavailable {
		return false
	}

	if podDisruptionBudget.Spec.Selector == nil {
		return false
	}

	if podDisruptionBudget.Spec.Selector.MatchLabels[runtimeNameLabel] !=
		expectedRuntimeName {
		return false
	}

	if podDisruptionBudget.Spec.Selector.MatchLabels[runtimeInstanceLabel] !=
		runtimeName {
		return false
	}

	return true
}

func assertServiceDesiredState(
	t *testing.T,
	service *corev1.Service,
	runtimeName string,
) {
	t.Helper()

	if !serviceMatchesDesiredState(
		service,
		runtimeName,
	) {
		t.Fatalf(
			"recreated Service %s/%s does not match desired state",
			service.Namespace,
			service.Name,
		)
	}
}

func assertDeploymentDesiredState(
	t *testing.T,
	deployment *appsv1.Deployment,
) {
	t.Helper()

	if !deploymentMatchesDesiredState(
		deployment,
	) {
		t.Fatalf(
			"recreated Deployment %s/%s does not match desired state",
			deployment.Namespace,
			deployment.Name,
		)
	}
}

func assertPodDisruptionBudgetDesiredState(
	t *testing.T,
	podDisruptionBudget *policyv1.PodDisruptionBudget,
	runtimeName string,
) {
	t.Helper()

	if !podDisruptionBudgetMatchesDesiredState(
		podDisruptionBudget,
		runtimeName,
	) {
		t.Fatalf(
			"recreated PodDisruptionBudget %s/%s does not match desired state",
			podDisruptionBudget.Namespace,
			podDisruptionBudget.Name,
		)
	}
}

func deleteManagedObject(
	t *testing.T,
	object client.Object,
) {
	t.Helper()

	if err := testClient.Delete(
		context.Background(),
		object,
	); err != nil {
		t.Fatalf(
			"delete managed %T %s/%s: %v",
			object,
			object.GetNamespace(),
			object.GetName(),
			err,
		)
	}
}

func waitForRecreatedObject[T client.Object](
	t *testing.T,
	key client.ObjectKey,
	object T,
	oldUID types.UID,
) T {
	t.Helper()

	eventually(
		t,
		"managed resource recreated",
		func() (bool, error) {
			if err := testClient.Get(
				context.Background(),
				key,
				object,
			); err != nil {
				return false, client.IgnoreNotFound(err)
			}

			return object.GetUID() != "" &&
				object.GetUID() != oldUID, nil
		},
	)

	return object
}

func assertRuntimeGenerationPreserved(
	t *testing.T,
	key client.ObjectKey,
	expectedGeneration int64,
) {
	t.Helper()

	eventually(
		t,
		"TrussiumRuntime generation remains unchanged",
		func() (bool, error) {
			var runtimeResource runtimev1alpha1.TrussiumRuntime

			if err := testClient.Get(
				context.Background(),
				key,
				&runtimeResource,
			); err != nil {
				return false, err
			}

			return runtimeResource.Generation ==
				expectedGeneration, nil
		},
	)
}
