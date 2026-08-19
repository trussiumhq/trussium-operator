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
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

func TestRuntimeUpgradeStartedEvent(t *testing.T) {
	t.Parallel()

	recorder := events.NewFakeRecorder(10)

	reconciler := TrussiumRuntimeReconciler{
		Recorder: recorder,
	}

	runtimeResource := newTestRuntime()

	previousStatus := runtimev1alpha1.TrussiumRuntimeStatus{
		DesiredImage:        testRuntimeImage,
		CurrentImage:        testRuntimeImage,
		LastSuccessfulImage: testRuntimeImage,
	}

	setRuntimeCondition(
		&previousStatus,
		1,
		conditionTypeUpgrading,
		metav1.ConditionFalse,
		reasonNoUpgrade,
		"No runtime image upgrade is in progress.",
	)

	currentStatus := copyRuntimeStatus(previousStatus)
	currentStatus.DesiredImage =
		testUpgradeRuntimeImage
	currentStatus.CurrentImage =
		testUpgradeRuntimeImage

	setRuntimeCondition(
		&currentStatus,
		2,
		conditionTypeUpgrading,
		metav1.ConditionTrue,
		reasonUpgradeInProgress,
		"Runtime image upgrade is in progress.",
	)

	reconciler.recordStatusTransitionEvents(
		runtimeResource,
		previousStatus,
		currentStatus,
	)

	recordedEvents := drainRecordedEvents(recorder)

	assertUpgradeEvent(
		t,
		recordedEvents,
		eventReasonRuntimeUpgradeStarted,
		testRuntimeImage,
		testUpgradeRuntimeImage,
	)
}

func TestRuntimeUpgradeCompletedEvent(t *testing.T) {
	t.Parallel()

	recorder := events.NewFakeRecorder(10)

	reconciler := TrussiumRuntimeReconciler{
		Recorder: recorder,
	}

	runtimeResource := newTestRuntime()

	previousStatus := runtimev1alpha1.TrussiumRuntimeStatus{
		DesiredImage:        testUpgradeRuntimeImage,
		CurrentImage:        testUpgradeRuntimeImage,
		LastSuccessfulImage: testRuntimeImage,
	}

	setRuntimeCondition(
		&previousStatus,
		2,
		conditionTypeUpgrading,
		metav1.ConditionTrue,
		reasonUpgradeInProgress,
		"Runtime image upgrade is in progress.",
	)

	currentStatus := copyRuntimeStatus(previousStatus)
	currentStatus.LastSuccessfulImage =
		testUpgradeRuntimeImage

	setRuntimeCondition(
		&currentStatus,
		2,
		conditionTypeUpgrading,
		metav1.ConditionFalse,
		reasonUpgradeComplete,
		"Runtime image upgrade completed.",
	)

	reconciler.recordStatusTransitionEvents(
		runtimeResource,
		previousStatus,
		currentStatus,
	)

	recordedEvents := drainRecordedEvents(recorder)

	assertUpgradeEvent(
		t,
		recordedEvents,
		eventReasonRuntimeUpgradeCompleted,
		testRuntimeImage,
		testUpgradeRuntimeImage,
	)
}

func TestRuntimeUpgradeFailedEvent(t *testing.T) {
	t.Parallel()

	recorder := events.NewFakeRecorder(10)

	reconciler := TrussiumRuntimeReconciler{
		Recorder: recorder,
	}

	runtimeResource := newTestRuntime()

	previousStatus := runtimev1alpha1.TrussiumRuntimeStatus{
		DesiredImage:        testUpgradeRuntimeImage,
		CurrentImage:        testUpgradeRuntimeImage,
		LastSuccessfulImage: testRuntimeImage,
	}

	setRuntimeCondition(
		&previousStatus,
		2,
		conditionTypeUpgrading,
		metav1.ConditionTrue,
		reasonUpgradeInProgress,
		"Runtime image upgrade is in progress.",
	)

	currentStatus := copyRuntimeStatus(previousStatus)

	setRuntimeCondition(
		&currentStatus,
		2,
		conditionTypeUpgrading,
		metav1.ConditionFalse,
		reasonUpgradeFailed,
		"Runtime image upgrade failed.",
	)

	reconciler.recordStatusTransitionEvents(
		runtimeResource,
		previousStatus,
		currentStatus,
	)

	recordedEvents := drainRecordedEvents(recorder)

	assertUpgradeEvent(
		t,
		recordedEvents,
		eventReasonRuntimeUpgradeFailed,
		testRuntimeImage,
		testUpgradeRuntimeImage,
	)
}

func TestInitialDeploymentDoesNotEmitUpgradeEvent(
	t *testing.T,
) {
	t.Parallel()

	recorder := events.NewFakeRecorder(10)

	reconciler := TrussiumRuntimeReconciler{
		Recorder: recorder,
	}

	runtimeResource := newTestRuntime()

	previousStatus :=
		runtimev1alpha1.TrussiumRuntimeStatus{}

	currentStatus :=
		runtimev1alpha1.TrussiumRuntimeStatus{
			DesiredImage:        testRuntimeImage,
			CurrentImage:        testRuntimeImage,
			LastSuccessfulImage: testRuntimeImage,
		}

	setRuntimeCondition(
		&currentStatus,
		1,
		conditionTypeUpgrading,
		metav1.ConditionFalse,
		reasonNoUpgrade,
		"No runtime image upgrade is in progress.",
	)

	reconciler.recordStatusTransitionEvents(
		runtimeResource,
		previousStatus,
		currentStatus,
	)

	recordedEvents := drainRecordedEvents(recorder)

	for _, recordedEvent := range recordedEvents {
		if strings.Contains(
			recordedEvent,
			eventReasonRuntimeUpgradeStarted,
		) ||
			strings.Contains(
				recordedEvent,
				eventReasonRuntimeUpgradeCompleted,
			) ||
			strings.Contains(
				recordedEvent,
				eventReasonRuntimeUpgradeFailed,
			) {
			t.Fatalf(
				"initial deployment must not emit an upgrade Event: %q",
				recordedEvent,
			)
		}
	}
}

func TestUnchangedUpgradeStateDoesNotEmitDuplicateEvent(
	t *testing.T,
) {
	t.Parallel()

	recorder := events.NewFakeRecorder(10)

	reconciler := TrussiumRuntimeReconciler{
		Recorder: recorder,
	}

	runtimeResource := newTestRuntime()

	previousStatus := runtimev1alpha1.TrussiumRuntimeStatus{
		DesiredImage:        testUpgradeRuntimeImage,
		CurrentImage:        testUpgradeRuntimeImage,
		LastSuccessfulImage: testRuntimeImage,
	}

	setRuntimeCondition(
		&previousStatus,
		2,
		conditionTypeUpgrading,
		metav1.ConditionTrue,
		reasonUpgradeInProgress,
		"Runtime image upgrade is in progress.",
	)

	currentStatus := copyRuntimeStatus(previousStatus)

	reconciler.recordStatusTransitionEvents(
		runtimeResource,
		previousStatus,
		currentStatus,
	)

	recordedEvents := drainRecordedEvents(recorder)

	if len(recordedEvents) != 0 {
		t.Fatalf(
			"expected no duplicate Event, received %#v",
			recordedEvents,
		)
	}
}

func TestValidUpgradeEventImages(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		source      string
		destination string
		expected    bool
	}{
		{
			name:        "valid transition",
			source:      testRuntimeImage,
			destination: testUpgradeRuntimeImage,
			expected:    true,
		},
		{
			name:        "missing source",
			source:      "",
			destination: testUpgradeRuntimeImage,
			expected:    false,
		},
		{
			name:        "missing destination",
			source:      testRuntimeImage,
			destination: "",
			expected:    false,
		},
		{
			name:        "same image",
			source:      testRuntimeImage,
			destination: testRuntimeImage,
			expected:    false,
		},
	}

	for _, testCase := range testCases {

		t.Run(
			testCase.name,
			func(t *testing.T) {
				t.Parallel()

				actual := validUpgradeEventImages(
					testCase.source,
					testCase.destination,
				)

				if actual != testCase.expected {
					t.Fatalf(
						"expected %t, received %t",
						testCase.expected,
						actual,
					)
				}
			},
		)
	}
}

func assertUpgradeEvent(
	t *testing.T,
	recordedEvents []string,
	reason string,
	sourceImage string,
	destinationImage string,
) {
	t.Helper()

	for _, recordedEvent := range recordedEvents {
		if !strings.Contains(
			recordedEvent,
			reason,
		) {
			continue
		}

		if !strings.Contains(
			recordedEvent,
			sourceImage,
		) {
			t.Fatalf(
				"Event %q does not contain source image %q",
				recordedEvent,
				sourceImage,
			)
		}

		if !strings.Contains(
			recordedEvent,
			destinationImage,
		) {
			t.Fatalf(
				"Event %q does not contain destination image %q",
				recordedEvent,
				destinationImage,
			)
		}

		return
	}

	t.Fatalf(
		"expected Event reason %q, received %#v",
		reason,
		recordedEvents,
	)
}
