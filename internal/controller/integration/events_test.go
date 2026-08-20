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

	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

const (
	runtimeUpgradeStartedEvent   = "RuntimeUpgradeStarted"
	runtimeUpgradeCompletedEvent = "RuntimeUpgradeCompleted"
	runtimeUpgradeFailedEvent    = "RuntimeUpgradeFailed"
	runtimeScaledToZeroEvent     = "RuntimeScaledToZero"
)

func TestUpgradeLifecyclePersistsKubernetesEvents(
	t *testing.T,
) {
	t.Run(
		"successful upgrade persists started and completed events",
		func(t *testing.T) {
			fixture :=
				establishSuccessfulRuntime(t)

			previousGeneration :=
				fixture.Deployment.Generation

			updateRuntimeImageTag(
				t,
				fixture.Key,
			)

			waitForDeploymentImage(
				t,
				fixture.Key,
				previousGeneration,
			)

			waitForUpgradeConditionState(
				t,
				fixture.Key,
				metav1.ConditionTrue,
				upgradeInProgressReason,
			)

			waitForRuntimeEventReason(
				t,
				fixture.Key,
				fixture.Runtime.UID,
				runtimeUpgradeStartedEvent,
			)

			markDeploymentRolloutComplete(
				t,
				fixture.Key,
			)

			waitForUpgradeConditionState(
				t,
				fixture.Key,
				metav1.ConditionFalse,
				upgradeCompleteReason,
			)

			waitForRuntimeEventReason(
				t,
				fixture.Key,
				fixture.Runtime.UID,
				runtimeUpgradeCompletedEvent,
			)
		},
	)

	t.Run(
		"failed upgrade persists failure event",
		func(t *testing.T) {
			fixture :=
				establishSuccessfulRuntime(t)

			previousGeneration :=
				fixture.Deployment.Generation

			updateRuntimeImageTag(
				t,
				fixture.Key,
			)

			waitForDeploymentImage(
				t,
				fixture.Key,
				previousGeneration,
			)

			waitForUpgradeConditionState(
				t,
				fixture.Key,
				metav1.ConditionTrue,
				upgradeInProgressReason,
			)

			waitForRuntimeEventReason(
				t,
				fixture.Key,
				fixture.Runtime.UID,
				runtimeUpgradeStartedEvent,
			)

			markDeploymentRolloutFailed(
				t,
				fixture.Key,
			)

			waitForUpgradeConditionState(
				t,
				fixture.Key,
				metav1.ConditionFalse,
				upgradeFailedReason,
			)

			waitForRuntimeEventReason(
				t,
				fixture.Key,
				fixture.Runtime.UID,
				runtimeUpgradeFailedEvent,
			)
		},
	)
}

func TestScaleToZeroPersistsKubernetesEvent(
	t *testing.T,
) {
	fixture := establishSuccessfulRuntime(t)

	updateRuntimeReplicas(
		t,
		fixture.Key,
		0,
	)

	waitForDeploymentReplicas(
		t,
		fixture.Key,
		0,
	)

	markDeploymentScaledToZero(
		t,
		fixture.Key,
	)

	waitForRuntimeStatus(
		t,
		fixture.Key,
		func(
			runtimeResource *runtimev1alpha1.TrussiumRuntime,
		) bool {
			progressing := runtimeCondition(
				runtimeResource,
				"Progressing",
			)

			return runtimeResource.Status.ObservedGeneration ==
				runtimeResource.Generation &&
				runtimeResource.Status.ReadyReplicas == 0 &&
				runtimeResource.Status.AvailableReplicas == 0 &&
				progressing != nil &&
				progressing.Status == metav1.ConditionFalse &&
				progressing.Reason == scaledToZeroReason
		},
	)

	waitForRuntimeEventReason(
		t,
		fixture.Key,
		fixture.Runtime.UID,
		runtimeScaledToZeroEvent,
	)
}

func waitForRuntimeEventReason(
	t *testing.T,
	key client.ObjectKey,
	runtimeUID types.UID,
	reason string,
) {
	t.Helper()

	eventually(
		t,
		"Kubernetes Event "+reason,
		func() (bool, error) {
			var events eventsv1.EventList

			if err := testClient.List(
				context.Background(),
				&events,
				client.InNamespace(key.Namespace),
			); err != nil {
				return false, err
			}

			for index := range events.Items {
				event := &events.Items[index]

				if event.Reason != reason {
					continue
				}

				if event.Regarding.Name != key.Name {
					continue
				}

				if event.Regarding.Namespace !=
					key.Namespace {
					continue
				}

				if event.Regarding.UID != runtimeUID {
					continue
				}

				if event.Regarding.Kind !=
					"TrussiumRuntime" {
					continue
				}

				return true, nil
			}

			return false, nil
		},
	)
}
