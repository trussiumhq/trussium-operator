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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

func TestRuntimeScalesToZero(
	t *testing.T,
) {
	fixture := establishSuccessfulRuntime(t)

	initialImage :=
		testRuntimeImageRepository + ":" +
			testRuntimeImageTag

	previousRuntimeGeneration :=
		fixture.Runtime.Generation

	updateRuntimeReplicas(
		t,
		fixture.Key,
		0,
	)

	deployment := waitForDeploymentReplicas(
		t,
		fixture.Key,
		0,
	)

	if deployment.Spec.Replicas == nil {
		t.Fatal(
			"Deployment replicas must be set after scale to zero",
		)
	}

	if *deployment.Spec.Replicas != 0 {
		t.Fatalf(
			"Deployment replicas = %d, expected 0",
			*deployment.Spec.Replicas,
		)
	}

	markDeploymentScaledToZero(
		t,
		fixture.Key,
	)

	scaledRuntime := waitForRuntimeStatus(
		t,
		fixture.Key,
		func(
			runtimeResource *runtimev1alpha1.TrussiumRuntime,
		) bool {
			return runtimeResource.Generation >
				previousRuntimeGeneration &&
				runtimeResource.Status.ObservedGeneration ==
					runtimeResource.Generation &&
				runtimeResource.Status.ReadyReplicas == 0 &&
				runtimeResource.Status.AvailableReplicas == 0
		},
	)

	if scaledRuntime.Status.DesiredImage !=
		initialImage {
		t.Fatalf(
			"desiredImage = %q after scale to zero, expected %q",
			scaledRuntime.Status.DesiredImage,
			initialImage,
		)
	}

	if scaledRuntime.Status.CurrentImage !=
		initialImage {
		t.Fatalf(
			"currentImage = %q after scale to zero, expected %q",
			scaledRuntime.Status.CurrentImage,
			initialImage,
		)
	}

	if scaledRuntime.Status.LastSuccessfulImage !=
		initialImage {
		t.Fatalf(
			"lastSuccessfulImage = %q after scale to zero, expected %q",
			scaledRuntime.Status.LastSuccessfulImage,
			initialImage,
		)
	}

	progressing := runtimeCondition(
		scaledRuntime,
		"Progressing",
	)
	if progressing == nil {
		t.Fatal(
			"Progressing condition was not persisted after scale to zero",
		)
	}

	if progressing.Status != metav1.ConditionFalse {
		t.Fatalf(
			"Progressing status = %q after scale to zero, expected False",
			progressing.Status,
		)
	}

	if progressing.Reason != scaledToZeroReason {
		t.Fatalf(
			"Progressing reason = %q after scale to zero, expected %q",
			progressing.Reason,
			scaledToZeroReason,
		)
	}

	ready := runtimeCondition(
		scaledRuntime,
		"Ready",
	)
	if ready == nil {
		t.Fatal(
			"Ready condition was not persisted after scale to zero",
		)
	}

	if ready.Status != metav1.ConditionTrue {
		t.Fatalf(
			"Ready status = %q after scale to zero, expected True",
			ready.Status,
		)
	}

	if ready.Reason != scaledToZeroReason {
		t.Fatalf(
			"Ready reason = %q after scale to zero, expected %q",
			ready.Reason,
			scaledToZeroReason,
		)
	}

	upgrading := runtimeCondition(
		scaledRuntime,
		upgradingConditionType,
	)
	if upgrading == nil {
		t.Fatal(
			"Upgrading condition was not persisted after scale to zero",
		)
	}

	if upgrading.Status != metav1.ConditionFalse {
		t.Fatalf(
			"Upgrading status = %q after scale to zero, expected False",
			upgrading.Status,
		)
	}

	if upgrading.Reason != noUpgradeReason {
		t.Fatalf(
			"Upgrading reason = %q after scale to zero, expected %q",
			upgrading.Reason,
			noUpgradeReason,
		)
	}
}

func updateRuntimeReplicas(
	t *testing.T,
	key client.ObjectKey,
	replicas int32,
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

			runtimeResource.Spec.Replicas = &replicas

			return testClient.Update(
				context.Background(),
				&runtimeResource,
			)
		},
	)
	if err != nil {
		t.Fatalf(
			"update TrussiumRuntime replicas to %d: %v",
			replicas,
			err,
		)
	}
}

func waitForDeploymentReplicas(
	t *testing.T,
	key client.ObjectKey,
	expectedReplicas int32,
) *appsv1.Deployment {
	t.Helper()

	var deployment appsv1.Deployment

	eventually(
		t,
		"Deployment replica count is reconciled",
		func() (bool, error) {
			if err := testClient.Get(
				context.Background(),
				key,
				&deployment,
			); err != nil {
				return false, err
			}

			return deployment.Spec.Replicas != nil &&
					*deployment.Spec.Replicas ==
						expectedReplicas,
				nil
		},
	)

	return deployment.DeepCopy()
}

func markDeploymentScaledToZero(
	t *testing.T,
	key client.ObjectKey,
) {
	t.Helper()

	eventually(
		t,
		"Deployment status reaches scaled-to-zero state",
		func() (bool, error) {
			var deployment appsv1.Deployment

			if err := testClient.Get(
				context.Background(),
				key,
				&deployment,
			); err != nil {
				return false, err
			}

			if deployment.Spec.Replicas == nil ||
				*deployment.Spec.Replicas != 0 {
				return false, nil
			}

			deployment.Status.ObservedGeneration =
				deployment.Generation
			deployment.Status.Replicas = 0
			deployment.Status.UpdatedReplicas = 0
			deployment.Status.ReadyReplicas = 0
			deployment.Status.AvailableReplicas = 0
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

			return true, nil
		},
	)
}
