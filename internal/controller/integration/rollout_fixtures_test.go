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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

const (
	testRuntimeImageTagV2 = "0.24.0"

	runtimeConfigChecksumAnnotation = "runtime.trussium.io/config-checksum"

	upgradingConditionType = "Upgrading"

	noUpgradeReason         = "NoUpgrade"
	upgradeInProgressReason = "UpgradeInProgress"
	upgradeCompleteReason   = "UpgradeComplete"
	upgradeFailedReason     = "UpgradeFailed"
	scaledToZeroReason      = "ScaledToZero"

	progressDeadlineExceededReason = "ProgressDeadlineExceeded"

	providerBaseURLEnvironment = "TRUSSIUM_PROVIDER__BASE_URL"
)

type runtimeRolloutFixture struct {
	Runtime    *runtimev1alpha1.TrussiumRuntime
	Key        client.ObjectKey
	Deployment *appsv1.Deployment
}

func establishSuccessfulRuntime(
	t *testing.T,
) runtimeRolloutFixture {
	t.Helper()

	namespace := createTestNamespace(t)

	runtimeResource := createTestRuntime(
		t,
		namespace.Name,
	)

	key := client.ObjectKeyFromObject(runtimeResource)

	waitForObject(
		t,
		key,
		&appsv1.Deployment{},
	)

	deployment := markDeploymentRolloutComplete(
		t,
		key,
	)

	initialImage :=
		testRuntimeImageRepository + ":" + testRuntimeImageTag

	current := waitForRuntimeStatus(
		t,
		key,
		func(
			resource *runtimev1alpha1.TrussiumRuntime,
		) bool {
			condition := runtimeCondition(
				resource,
				upgradingConditionType,
			)

			return resource.Status.LastSuccessfulImage ==
				initialImage &&
				condition != nil &&
				condition.Status == metav1.ConditionFalse &&
				condition.Reason == noUpgradeReason
		},
	)

	return runtimeRolloutFixture{
		Runtime:    current,
		Key:        key,
		Deployment: deployment,
	}
}

func markDeploymentRolloutComplete(
	t *testing.T,
	key client.ObjectKey,
) *appsv1.Deployment {
	t.Helper()

	var completedDeployment appsv1.Deployment

	eventually(
		t,
		"Deployment rollout status becomes complete",
		func() (bool, error) {
			var deployment appsv1.Deployment

			if err := testClient.Get(
				context.Background(),
				key,
				&deployment,
			); err != nil {
				return false, err
			}

			desiredReplicas := int32(1)
			if deployment.Spec.Replicas != nil {
				desiredReplicas = *deployment.Spec.Replicas
			}

			deployment.Status.ObservedGeneration =
				deployment.Generation
			deployment.Status.Replicas =
				desiredReplicas
			deployment.Status.UpdatedReplicas =
				desiredReplicas
			deployment.Status.ReadyReplicas =
				desiredReplicas
			deployment.Status.AvailableReplicas =
				desiredReplicas
			deployment.Status.UnavailableReplicas = 0

			if err := testClient.Status().Update(
				context.Background(),
				&deployment,
			); err != nil {
				if apierrors.IsConflict(err) {
					return false, nil
				}

				return false, err
			}

			completedDeployment = *deployment.DeepCopy()

			return true, nil
		},
	)

	return completedDeployment.DeepCopy()
}

func markDeploymentRolloutFailed(
	t *testing.T,
	key client.ObjectKey,
) {
	t.Helper()

	eventually(
		t,
		"Deployment rollout status becomes failed",
		func() (bool, error) {
			var deployment appsv1.Deployment

			if err := testClient.Get(
				context.Background(),
				key,
				&deployment,
			); err != nil {
				return false, err
			}

			deployment.Status.ObservedGeneration =
				deployment.Generation
			deployment.Status.Conditions =
				[]appsv1.DeploymentCondition{
					{
						Type:               appsv1.DeploymentProgressing,
						Status:             corev1.ConditionFalse,
						Reason:             progressDeadlineExceededReason,
						Message:            "integration test simulated progress deadline failure",
						LastUpdateTime:     metav1.Now(),
						LastTransitionTime: metav1.Now(),
					},
				}

			if err := testClient.Status().Update(
				context.Background(),
				&deployment,
			); err != nil {
				if apierrors.IsConflict(err) {
					return false, nil
				}

				return false, err
			}

			return true, nil
		},
	)
}

func updateRuntimeProviderBaseURL(
	t *testing.T,
	key client.ObjectKey,
	baseURL string,
) *runtimev1alpha1.TrussiumRuntime {
	t.Helper()

	var updatedRuntime runtimev1alpha1.TrussiumRuntime

	err := retry.RetryOnConflict(
		retry.DefaultBackoff,
		func() error {
			var runtimeResource runtimev1alpha1.TrussiumRuntime

			if err := testClient.Get(
				context.Background(),
				key,
				&runtimeResource,
			); err != nil {
				return err
			}

			runtimeResource.Spec.Provider.BaseURL = &baseURL

			if err := testClient.Update(
				context.Background(),
				&runtimeResource,
			); err != nil {
				return err
			}

			updatedRuntime = *runtimeResource.DeepCopy()

			return nil
		},
	)
	if err != nil {
		t.Fatalf(
			"update TrussiumRuntime provider base URL: %v",
			err,
		)
	}

	return updatedRuntime.DeepCopy()
}

func updateRuntimeImageTag(
	t *testing.T,
	key client.ObjectKey,
) {
	t.Helper()

	err := retry.RetryOnConflict(
		retry.DefaultBackoff,
		func() error {
			var runtimeResource runtimev1alpha1.TrussiumRuntime

			if err := testClient.Get(
				context.Background(),
				key,
				&runtimeResource,
			); err != nil {
				return err
			}

			tag := testRuntimeImageTagV2

			runtimeResource.Spec.Image.Tag = &tag
			runtimeResource.Spec.Image.Digest = nil

			return testClient.Update(
				context.Background(),
				&runtimeResource,
			)
		},
	)
	if err != nil {
		t.Fatalf(
			"update TrussiumRuntime image tag: %v",
			err,
		)
	}
}

func deploymentConfigChecksum(
	t *testing.T,
	deployment *appsv1.Deployment,
) string {
	t.Helper()

	checksum := deployment.Spec.Template.Annotations[runtimeConfigChecksumAnnotation]

	if checksum == "" {
		t.Fatalf(
			"Deployment %s/%s is missing %q annotation",
			deployment.Namespace,
			deployment.Name,
			runtimeConfigChecksumAnnotation,
		)
	}

	return checksum
}

func waitForDeploymentChecksumChange(
	t *testing.T,
	key client.ObjectKey,
	previousChecksum string,
	previousGeneration int64,
) *appsv1.Deployment {
	t.Helper()

	var current appsv1.Deployment

	eventually(
		t,
		"Deployment configuration checksum changes",
		func() (bool, error) {
			if err := testClient.Get(
				context.Background(),
				key,
				&current,
			); err != nil {
				return false, err
			}

			checksum :=
				current.Spec.Template.Annotations[runtimeConfigChecksumAnnotation]

			return checksum != "" &&
					checksum != previousChecksum &&
					current.Generation > previousGeneration,
				nil
		},
	)

	return current.DeepCopy()
}

func waitForDeploymentImage(
	t *testing.T,
	key client.ObjectKey,
	previousGeneration int64,
) *appsv1.Deployment {
	t.Helper()

	var current appsv1.Deployment

	expectedImage :=
		testRuntimeImageRepository + ":" +
			testRuntimeImageTagV2

	eventually(
		t,
		"Deployment runtime image changes",
		func() (bool, error) {
			if err := testClient.Get(
				context.Background(),
				key,
				&current,
			); err != nil {
				return false, err
			}

			if len(
				current.Spec.Template.Spec.Containers,
			) != 1 {
				return false, nil
			}

			return current.Spec.Template.Spec.
					Containers[0].Image == expectedImage &&
					current.Generation > previousGeneration,
				nil
		},
	)

	return current.DeepCopy()
}

func waitForUpgradeConditionState(
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
				upgradingConditionType,
			)

			return condition != nil &&
				condition.Status == status &&
				condition.Reason == reason
		},
	)
}

func waitForConfigMapValue(
	t *testing.T,
	key client.ObjectKey,
	name string,
	expectedValue string,
) {
	t.Helper()

	eventually(
		t,
		"ConfigMap configuration value is updated",
		func() (bool, error) {
			var configMap corev1.ConfigMap

			if err := testClient.Get(
				context.Background(),
				key,
				&configMap,
			); err != nil {
				return false, err
			}

			return configMap.Data[name] ==
				expectedValue, nil
		},
	)
}

func runtimeContainerImage(
	t *testing.T,
	deployment *appsv1.Deployment,
) string {
	t.Helper()

	if len(
		deployment.Spec.Template.Spec.Containers,
	) != 1 {
		t.Fatalf(
			"Deployment %s/%s has %d containers, expected 1",
			deployment.Namespace,
			deployment.Name,
			len(
				deployment.Spec.Template.Spec.Containers,
			),
		)
	}

	return deployment.Spec.Template.Spec.
		Containers[0].Image
}
