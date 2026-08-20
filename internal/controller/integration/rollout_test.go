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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

func TestConfigurationRolloutAndSuccessfulImageUpgrade(
	t *testing.T,
) {
	fixture := establishSuccessfulRuntime(t)

	initialImage :=
		testRuntimeImageRepository + ":" +
			testRuntimeImageTag

	initialChecksum := deploymentConfigChecksum(
		t,
		fixture.Deployment,
	)

	initialDeploymentGeneration :=
		fixture.Deployment.Generation

	t.Run(
		"configuration change rolls the Deployment without an image upgrade",
		func(t *testing.T) {
			baseURL :=
				"http://ollama.integration.invalid:11434"

			updatedRuntime :=
				updateRuntimeProviderBaseURL(
					t,
					fixture.Key,
					baseURL,
				)

			if updatedRuntime.Generation <=
				fixture.Runtime.Generation {
				t.Fatalf(
					"TrussiumRuntime generation = %d after configuration update, expected greater than %d",
					updatedRuntime.Generation,
					fixture.Runtime.Generation,
				)
			}

			waitForConfigMapValue(
				t,
				fixture.Key,
				providerBaseURLEnvironment,
				baseURL,
			)

			rolledDeployment :=
				waitForDeploymentChecksumChange(
					t,
					fixture.Key,
					initialChecksum,
					initialDeploymentGeneration,
				)

			if runtimeContainerImage(
				t,
				rolledDeployment,
			) != initialImage {
				t.Fatalf(
					"configuration rollout changed runtime image to %q, expected %q",
					runtimeContainerImage(
						t,
						rolledDeployment,
					),
					initialImage,
				)
			}

			current :=
				waitForUpgradeConditionState(
					t,
					fixture.Key,
					metav1.ConditionFalse,
					noUpgradeReason,
				)

			if current.Status.LastSuccessfulImage !=
				initialImage {
				t.Fatalf(
					"lastSuccessfulImage = %q after configuration rollout, expected %q",
					current.Status.LastSuccessfulImage,
					initialImage,
				)
			}

			completedDeployment :=
				markDeploymentRolloutComplete(
					t,
					fixture.Key,
				)

			waitForRuntimeStatus(
				t,
				fixture.Key,
				func(
					runtimeResource *runtimev1alpha1.TrussiumRuntime,
				) bool {
					return runtimeResource.Status.
						LastSuccessfulImage ==
						initialImage &&
						runtimeResource.Status.
							ReadyReplicas == 1 &&
						runtimeResource.Status.
							AvailableReplicas == 1
				},
			)

			fixture.Deployment =
				completedDeployment
			fixture.Runtime =
				updatedRuntime
		},
	)

	t.Run(
		"image change completes an upgrade",
		func(t *testing.T) {
			previousGeneration :=
				fixture.Deployment.Generation

			updateRuntimeImageTag(
				t,
				fixture.Key,
			)

			expectedImage :=
				testRuntimeImageRepository + ":" +
					testRuntimeImageTagV2

			upgradingDeployment :=
				waitForDeploymentImage(
					t,
					fixture.Key,
					previousGeneration,
				)

			inProgress :=
				waitForUpgradeConditionState(
					t,
					fixture.Key,
					metav1.ConditionTrue,
					upgradeInProgressReason,
				)

			if inProgress.Status.LastSuccessfulImage !=
				initialImage {
				t.Fatalf(
					"lastSuccessfulImage = %q during upgrade, expected %q",
					inProgress.Status.LastSuccessfulImage,
					initialImage,
				)
			}

			if inProgress.Status.DesiredImage !=
				expectedImage {
				t.Fatalf(
					"desiredImage = %q during upgrade, expected %q",
					inProgress.Status.DesiredImage,
					expectedImage,
				)
			}

			if inProgress.Status.CurrentImage !=
				expectedImage {
				t.Fatalf(
					"currentImage = %q during upgrade, expected %q",
					inProgress.Status.CurrentImage,
					expectedImage,
				)
			}

			markDeploymentRolloutComplete(
				t,
				fixture.Key,
			)

			completed :=
				waitForUpgradeConditionState(
					t,
					fixture.Key,
					metav1.ConditionFalse,
					upgradeCompleteReason,
				)

			if completed.Status.LastSuccessfulImage !=
				expectedImage {
				t.Fatalf(
					"lastSuccessfulImage = %q after successful upgrade, expected %q",
					completed.Status.LastSuccessfulImage,
					expectedImage,
				)
			}

			if completed.Status.DesiredImage !=
				expectedImage {
				t.Fatalf(
					"desiredImage = %q after successful upgrade, expected %q",
					completed.Status.DesiredImage,
					expectedImage,
				)
			}

			if completed.Status.CurrentImage !=
				expectedImage {
				t.Fatalf(
					"currentImage = %q after successful upgrade, expected %q",
					completed.Status.CurrentImage,
					expectedImage,
				)
			}

			if runtimeContainerImage(
				t,
				upgradingDeployment,
			) != expectedImage {
				t.Fatalf(
					"Deployment image = %q, expected %q",
					runtimeContainerImage(
						t,
						upgradingDeployment,
					),
					expectedImage,
				)
			}
		},
	)
}

func TestFailedImageUpgradeDoesNotRollback(
	t *testing.T,
) {
	fixture := establishSuccessfulRuntime(t)

	initialImage :=
		testRuntimeImageRepository + ":" +
			testRuntimeImageTag

	expectedImage :=
		testRuntimeImageRepository + ":" +
			testRuntimeImageTagV2

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

	markDeploymentRolloutFailed(
		t,
		fixture.Key,
	)

	failed :=
		waitForUpgradeConditionState(
			t,
			fixture.Key,
			metav1.ConditionFalse,
			upgradeFailedReason,
		)

	if failed.Status.LastSuccessfulImage !=
		initialImage {
		t.Fatalf(
			"lastSuccessfulImage = %q after failed upgrade, expected %q",
			failed.Status.LastSuccessfulImage,
			initialImage,
		)
	}

	if failed.Status.DesiredImage != expectedImage {
		t.Fatalf(
			"desiredImage = %q after failed upgrade, expected %q",
			failed.Status.DesiredImage,
			expectedImage,
		)
	}

	if failed.Status.CurrentImage != expectedImage {
		t.Fatalf(
			"currentImage = %q after failed upgrade, expected %q",
			failed.Status.CurrentImage,
			expectedImage,
		)
	}

	degraded := runtimeCondition(
		failed,
		"Degraded",
	)

	if degraded == nil {
		t.Fatal(
			"Degraded condition was not persisted after failed upgrade",
		)
	}

	if degraded.Status != metav1.ConditionTrue {
		t.Fatalf(
			"Degraded status = %q after failed upgrade, expected True",
			degraded.Status,
		)
	}

	if degraded.Reason !=
		progressDeadlineExceededReason {
		t.Fatalf(
			"Degraded reason = %q after failed upgrade, expected %q",
			degraded.Reason,
			progressDeadlineExceededReason,
		)
	}

	var deployment appsv1.Deployment

	if err := testClient.Get(
		context.Background(),
		fixture.Key,
		&deployment,
	); err != nil {
		t.Fatalf(
			"get Deployment after failed upgrade: %v",
			err,
		)
	}

	if runtimeContainerImage(
		t,
		&deployment,
	) != expectedImage {
		t.Fatalf(
			"Deployment image was rolled back to %q, expected failed desired image %q to remain",
			runtimeContainerImage(
				t,
				&deployment,
			),
			expectedImage,
		)
	}
}
