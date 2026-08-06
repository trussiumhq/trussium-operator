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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

const (
	eventReasonRuntimeProgressing   = "RuntimeProgressing"
	eventReasonRuntimeReady         = "RuntimeReady"
	eventReasonRuntimeRecovered     = "RuntimeRecovered"
	eventReasonRuntimeScaledToZero  = "RuntimeScaledToZero"
	eventReasonConfigurationInvalid = "ConfigurationInvalid"
	eventReasonReconciliationFailed = "ReconciliationFailed"
	eventReasonRuntimeDegraded      = "RuntimeDegraded"

	eventActionReconcileRuntime      = "ReconcileRuntime"
	eventActionValidateConfiguration = "ValidateConfiguration"
	eventActionObserveRuntime        = "ObserveRuntime"
)

// recordStatusTransitionEvents emits Events only when a meaningful condition
// status or reason changes.
func (r *TrussiumRuntimeReconciler) recordStatusTransitionEvents(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	previousStatus runtimev1alpha1.TrussiumRuntimeStatus,
	currentStatus runtimev1alpha1.TrussiumRuntimeStatus,
) {
	if r.Recorder == nil {
		return
	}

	r.recordConfigurationTransition(
		runtimeResource,
		previousStatus,
		currentStatus,
	)
	r.recordDegradedTransition(
		runtimeResource,
		previousStatus,
		currentStatus,
	)
	r.recordProgressingTransition(
		runtimeResource,
		previousStatus,
		currentStatus,
	)
	r.recordReadyTransition(
		runtimeResource,
		previousStatus,
		currentStatus,
	)
}

func (r *TrussiumRuntimeReconciler) recordConfigurationTransition(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	previousStatus runtimev1alpha1.TrussiumRuntimeStatus,
	currentStatus runtimev1alpha1.TrussiumRuntimeStatus,
) {
	if !conditionChanged(
		previousStatus,
		currentStatus,
		conditionTypeConfigurationValid,
	) {
		return
	}

	condition := runtimeCondition(
		currentStatus,
		conditionTypeConfigurationValid,
	)
	if condition == nil || condition.Status != metav1.ConditionFalse {
		return
	}

	r.Recorder.Eventf(
		runtimeResource,
		nil,
		corev1.EventTypeWarning,
		eventReasonConfigurationInvalid,
		eventActionValidateConfiguration,
		"%s",
		condition.Message,
	)
}

func (r *TrussiumRuntimeReconciler) recordDegradedTransition(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	previousStatus runtimev1alpha1.TrussiumRuntimeStatus,
	currentStatus runtimev1alpha1.TrussiumRuntimeStatus,
) {
	if !conditionChanged(
		previousStatus,
		currentStatus,
		conditionTypeDegraded,
	) {
		return
	}

	condition := runtimeCondition(
		currentStatus,
		conditionTypeDegraded,
	)
	if condition == nil {
		return
	}

	if condition.Status == metav1.ConditionTrue {
		r.Recorder.Eventf(
			runtimeResource,
			nil,
			corev1.EventTypeWarning,
			eventReasonRuntimeDegraded,
			eventActionObserveRuntime,
			"%s",
			condition.Message,
		)

		return
	}

	previousCondition := runtimeCondition(
		previousStatus,
		conditionTypeDegraded,
	)
	if previousCondition == nil ||
		previousCondition.Status != metav1.ConditionTrue {
		return
	}

	r.Recorder.Eventf(
		runtimeResource,
		nil,
		corev1.EventTypeNormal,
		eventReasonRuntimeRecovered,
		eventActionObserveRuntime,
		"%s",
		condition.Message,
	)
}

func (r *TrussiumRuntimeReconciler) recordProgressingTransition(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	previousStatus runtimev1alpha1.TrussiumRuntimeStatus,
	currentStatus runtimev1alpha1.TrussiumRuntimeStatus,
) {
	if !conditionChanged(
		previousStatus,
		currentStatus,
		conditionTypeProgressing,
	) {
		return
	}

	condition := runtimeCondition(
		currentStatus,
		conditionTypeProgressing,
	)
	if condition == nil || condition.Status != metav1.ConditionTrue {
		return
	}

	r.Recorder.Eventf(
		runtimeResource,
		nil,
		corev1.EventTypeNormal,
		eventReasonRuntimeProgressing,
		eventActionReconcileRuntime,
		"%s",
		condition.Message,
	)
}

func (r *TrussiumRuntimeReconciler) recordReadyTransition(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	previousStatus runtimev1alpha1.TrussiumRuntimeStatus,
	currentStatus runtimev1alpha1.TrussiumRuntimeStatus,
) {
	if !conditionChanged(
		previousStatus,
		currentStatus,
		conditionTypeReady,
	) {
		return
	}

	condition := runtimeCondition(
		currentStatus,
		conditionTypeReady,
	)
	if condition == nil || condition.Status != metav1.ConditionTrue {
		return
	}

	eventReason := eventReasonRuntimeReady
	if condition.Reason == reasonScaledToZero {
		eventReason = eventReasonRuntimeScaledToZero
	}

	r.Recorder.Eventf(
		runtimeResource,
		nil,
		corev1.EventTypeNormal,
		eventReason,
		eventActionObserveRuntime,
		"%s",
		condition.Message,
	)
}

func (r *TrussiumRuntimeReconciler) recordReconciliationFailure(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	reconciliationError error,
) {
	if r.Recorder == nil {
		return
	}

	r.Recorder.Eventf(
		runtimeResource,
		nil,
		corev1.EventTypeWarning,
		eventReasonReconciliationFailed,
		eventActionReconcileRuntime,
		"Runtime reconciliation failed: %v",
		reconciliationError,
	)
}

// Compile-time verification that the Kubernetes recorder satisfies the
// controller's recorder dependency.
var _ events.EventRecorder = events.NewFakeRecorder(1)
