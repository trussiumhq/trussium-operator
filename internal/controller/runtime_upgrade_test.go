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
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

func TestRuntimeConfigChecksumIsStable(t *testing.T) {
	t.Parallel()

	first := map[string]string{
		envProviderName:          testProviderName,
		"TRUSSIUM_RUNTIME__PORT": "9000",
		"TRUSSIUM_ENVIRONMENT":   defaultRuntimeEnvironment,
	}

	second := map[string]string{
		"TRUSSIUM_ENVIRONMENT":   defaultRuntimeEnvironment,
		"TRUSSIUM_RUNTIME__PORT": "9000",
		envProviderName:          testProviderName,
	}

	firstChecksum := runtimeConfigChecksum(first)
	secondChecksum := runtimeConfigChecksum(second)

	if firstChecksum != secondChecksum {
		t.Fatalf(
			"expected stable checksum, received %q and %q",
			firstChecksum,
			secondChecksum,
		)
	}
}

func TestRuntimeConfigChecksumChangesWithConfiguration(
	t *testing.T,
) {
	t.Parallel()

	first := map[string]string{
		envProviderName: testProviderName,
	}

	second := map[string]string{
		envProviderName: "openai",
	}

	if runtimeConfigChecksum(first) ==
		runtimeConfigChecksum(second) {
		t.Fatal(
			"expected configuration change to alter checksum",
		)
	}
}

func TestBuildDeploymentProjectsConfigurationChecksum(
	t *testing.T,
) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	deployment := buildDeployment(runtimeResource)

	actual :=
		deployment.Spec.Template.Annotations[runtimeConfigChecksumAnnotation]

	expected := runtimeConfigChecksum(
		runtimeConfigData(runtimeResource),
	)

	if actual != expected {
		t.Fatalf(
			"expected configuration checksum %q, received %q",
			expected,
			actual,
		)
	}

	if actual == "" {
		t.Fatal(
			"expected configuration checksum annotation",
		)
	}
}

func TestConfigurationChangeChangesPodTemplateChecksum(
	t *testing.T,
) {
	t.Parallel()

	firstRuntime := newTestRuntime()
	secondRuntime := newTestRuntime()

	secondRuntime.Spec.Provider.Type =
		runtimev1alpha1.ProviderTypeOpenAI

	firstDeployment := buildDeployment(firstRuntime)
	secondDeployment := buildDeployment(secondRuntime)

	firstChecksum :=
		firstDeployment.Spec.Template.Annotations[runtimeConfigChecksumAnnotation]

	secondChecksum :=
		secondDeployment.Spec.Template.Annotations[runtimeConfigChecksumAnnotation]

	if firstChecksum == secondChecksum {
		t.Fatalf(
			"expected runtime configuration change to alter Pod template checksum, both were %q",
			firstChecksum,
		)
	}
}

func TestUserCannotOverrideConfigurationChecksum(
	t *testing.T,
) {
	t.Parallel()

	runtimeResource := newTestRuntime()

	runtimeResource.Spec.PodMetadata =
		&runtimev1alpha1.PodMetadataSpec{
			Annotations: map[string]string{
				runtimeConfigChecksumAnnotation: "user-controlled-value",
			},
		}

	deployment := buildDeployment(runtimeResource)

	actual := deployment.Spec.Template.Annotations[runtimeConfigChecksumAnnotation]

	expected := runtimeConfigChecksum(
		runtimeConfigData(runtimeResource),
	)

	if actual != expected {
		t.Fatalf(
			"expected operator checksum %q, received %q",
			expected,
			actual,
		)
	}

	if actual == "user-controlled-value" {
		t.Fatal(
			"configuration checksum annotation must remain operator-owned",
		)
	}
}

func TestBuildDeploymentSetsProgressDeadline(t *testing.T) {
	t.Parallel()

	deployment := buildDeployment(newTestRuntime())

	if deployment.Spec.ProgressDeadlineSeconds == nil {
		t.Fatal(
			"expected Deployment progress deadline",
		)
	}

	if *deployment.Spec.ProgressDeadlineSeconds !=
		deploymentProgressDeadlineSeconds {
		t.Fatalf(
			"expected progress deadline %d, received %d",
			deploymentProgressDeadlineSeconds,
			*deployment.Spec.ProgressDeadlineSeconds,
		)
	}
}

func TestBuildRuntimeStatusReportsDesiredImage(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()

	status := buildRuntimeStatus(
		runtimeResource,
		runtimeObservation{},
		validReferenceValidation(),
	)

	if status.DesiredImage != testRuntimeImage {
		t.Fatalf(
			"expected desired image %q, received %q",
			testRuntimeImage,
			status.DesiredImage,
		)
	}

	if status.LastSuccessfulImage != "" {
		t.Fatalf(
			"expected no successful image before rollout completion, received %q",
			status.LastSuccessfulImage,
		)
	}
}

func completedDeployment(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) *appsv1.Deployment {
	deployment := buildDeployment(runtimeResource)
	deployment.Generation = 3

	desired := desiredReplicas(runtimeResource)

	deployment.Status.ObservedGeneration = 3
	deployment.Status.Replicas = desired
	deployment.Status.UpdatedReplicas = desired
	deployment.Status.ReadyReplicas = desired
	deployment.Status.AvailableReplicas = desired
	deployment.Status.UnavailableReplicas = 0

	deployment.Status.Conditions =
		[]appsv1.DeploymentCondition{
			{
				Type:   appsv1.DeploymentProgressing,
				Status: corev1.ConditionTrue,
				Reason: "NewReplicaSetAvailable",
			},
		}

	return deployment
}

func setUpgradeImage(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) {
	runtimeResource.Spec.Image.Tag =
		ptr.To(testUpgradeRuntimeTag)

	runtimeResource.Spec.Image.Digest = nil
}

func TestInitialRolloutEstablishesSuccessfulImage(
	t *testing.T,
) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	deployment := completedDeployment(runtimeResource)

	status := buildRuntimeStatus(
		runtimeResource,
		runtimeObservation{
			Deployment: deployment,
		},
		validReferenceValidation(),
	)

	if status.LastSuccessfulImage != testRuntimeImage {
		t.Fatalf(
			"expected initial successful image %q, received %q",
			testRuntimeImage,
			status.LastSuccessfulImage,
		)
	}

	assertRuntimeCondition(
		t,
		status,
		conditionTypeUpgrading,
		metav1.ConditionFalse,
		reasonNoUpgrade,
	)
}

func TestRuntimeImageUpgradeProgressing(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()

	runtimeResource.Status.LastSuccessfulImage =
		testRuntimeImage

	setUpgradeImage(runtimeResource)

	deployment := buildDeployment(runtimeResource)
	deployment.Generation = 4
	deployment.Status.ObservedGeneration = 3

	status := buildRuntimeStatus(
		runtimeResource,
		runtimeObservation{
			Deployment: deployment,
		},
		validReferenceValidation(),
	)

	if status.DesiredImage != testUpgradeRuntimeImage {
		t.Fatalf(
			"expected desired image %q, received %q",
			testUpgradeRuntimeImage,
			status.DesiredImage,
		)
	}

	if status.LastSuccessfulImage != testRuntimeImage {
		t.Fatalf(
			"expected successful image to remain %q, received %q",
			testRuntimeImage,
			status.LastSuccessfulImage,
		)
	}

	assertRuntimeCondition(
		t,
		status,
		conditionTypeUpgrading,
		metav1.ConditionTrue,
		reasonUpgradeInProgress,
	)
}

func TestRuntimeImageUpgradeCompletes(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()

	runtimeResource.Status.LastSuccessfulImage =
		testRuntimeImage

	setUpgradeImage(runtimeResource)

	deployment := completedDeployment(runtimeResource)

	status := buildRuntimeStatus(
		runtimeResource,
		runtimeObservation{
			Deployment: deployment,
		},
		validReferenceValidation(),
	)

	if status.LastSuccessfulImage !=
		testUpgradeRuntimeImage {
		t.Fatalf(
			"expected successful image %q, received %q",
			testUpgradeRuntimeImage,
			status.LastSuccessfulImage,
		)
	}

	assertRuntimeCondition(
		t,
		status,
		conditionTypeUpgrading,
		metav1.ConditionFalse,
		reasonUpgradeComplete,
	)
}

func TestRuntimeImageUpgradeFailurePreservesSuccessfulImage(
	t *testing.T,
) {
	t.Parallel()

	runtimeResource := newTestRuntime()

	runtimeResource.Status.LastSuccessfulImage =
		testRuntimeImage

	setUpgradeImage(runtimeResource)

	deployment := buildDeployment(runtimeResource)
	deployment.Generation = 4
	deployment.Status.ObservedGeneration = 4

	deployment.Status.Conditions =
		[]appsv1.DeploymentCondition{
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
		},
		validReferenceValidation(),
	)

	if status.LastSuccessfulImage != testRuntimeImage {
		t.Fatalf(
			"expected failed upgrade to preserve %q, received %q",
			testRuntimeImage,
			status.LastSuccessfulImage,
		)
	}

	assertRuntimeCondition(
		t,
		status,
		conditionTypeUpgrading,
		metav1.ConditionFalse,
		reasonUpgradeFailed,
	)

	assertRuntimeCondition(
		t,
		status,
		conditionTypeDegraded,
		metav1.ConditionTrue,
		reasonProgressDeadlineExceeded,
	)
}

func TestConfigurationRolloutIsNotRuntimeImageUpgrade(
	t *testing.T,
) {
	t.Parallel()

	runtimeResource := newTestRuntime()

	runtimeResource.Status.LastSuccessfulImage =
		testRuntimeImage

	deployment := buildDeployment(runtimeResource)
	deployment.Generation = 5
	deployment.Status.ObservedGeneration = 4

	status := buildRuntimeStatus(
		runtimeResource,
		runtimeObservation{
			Deployment: deployment,
		},
		validReferenceValidation(),
	)

	if status.LastSuccessfulImage != testRuntimeImage {
		t.Fatalf(
			"expected successful image %q, received %q",
			testRuntimeImage,
			status.LastSuccessfulImage,
		)
	}

	assertRuntimeCondition(
		t,
		status,
		conditionTypeUpgrading,
		metav1.ConditionFalse,
		reasonNoUpgrade,
	)
}

func TestScaleToZeroImageUpgradeCompletes(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()

	runtimeResource.Status.LastSuccessfulImage =
		testRuntimeImage

	runtimeResource.Spec.Replicas = ptr.To(int32(0))

	setUpgradeImage(runtimeResource)

	deployment := buildDeployment(runtimeResource)
	deployment.Generation = 6
	deployment.Status.ObservedGeneration = 6

	status := buildRuntimeStatus(
		runtimeResource,
		runtimeObservation{
			Deployment: deployment,
		},
		validReferenceValidation(),
	)

	if status.LastSuccessfulImage !=
		testUpgradeRuntimeImage {
		t.Fatalf(
			"expected scaled-to-zero successful image %q, received %q",
			testUpgradeRuntimeImage,
			status.LastSuccessfulImage,
		)
	}

	assertRuntimeCondition(
		t,
		status,
		conditionTypeUpgrading,
		metav1.ConditionFalse,
		reasonUpgradeComplete,
	)
}

func TestCompletedUpgradeStateIsPreserved(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	setUpgradeImage(runtimeResource)

	runtimeResource.Status.DesiredImage =
		testUpgradeRuntimeImage
	runtimeResource.Status.CurrentImage =
		testUpgradeRuntimeImage
	runtimeResource.Status.LastSuccessfulImage =
		testUpgradeRuntimeImage

	setRuntimeCondition(
		&runtimeResource.Status,
		4,
		conditionTypeUpgrading,
		metav1.ConditionFalse,
		reasonUpgradeComplete,
		"Runtime image upgrade completed.",
	)

	deployment := completedDeployment(
		runtimeResource,
	)

	status := buildRuntimeStatus(
		runtimeResource,
		runtimeObservation{
			Deployment: deployment,
		},
		validReferenceValidation(),
	)

	assertRuntimeCondition(
		t,
		status,
		conditionTypeUpgrading,
		metav1.ConditionFalse,
		reasonUpgradeComplete,
	)
}
