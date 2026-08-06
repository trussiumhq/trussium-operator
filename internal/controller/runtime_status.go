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
	"context"
	"fmt"
	"slices"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

const (
	conditionTypeConfigurationValid = "ConfigurationValid"
	conditionTypeProgressing        = "Progressing"
	conditionTypeAvailable          = "Available"
	conditionTypeReady              = "Ready"
	conditionTypeDegraded           = "Degraded"

	reasonReferencesResolved          = "ReferencesResolved"
	reasonSecretNotFound              = "SecretNotFound"
	reasonReferenceCheckFailed        = "ReferenceCheckFailed"
	reasonDeploymentProgressing       = "DeploymentProgressing"
	reasonDeploymentGenerationPending = "DeploymentGenerationPending"
	reasonReconciliationComplete      = "ReconciliationComplete"
	reasonConfigurationInvalid        = "ConfigurationInvalid"
	reasonDeploymentFailed            = "DeploymentFailed"
	reasonScaledToZero                = "ScaledToZero"
	reasonRuntimeAvailable            = "RuntimeAvailable"
	reasonRuntimeUnavailable          = "RuntimeUnavailable"
	reasonRuntimeReady                = "RuntimeReady"
	reasonRuntimeNotReady             = "RuntimeNotReady"
	reasonAsExpected                  = "AsExpected"
	reasonProgressDeadlineExceeded    = "ProgressDeadlineExceeded"
	reasonReplicaFailure              = "ReplicaFailure"
	reasonReconciliationFailed        = "ReconciliationFailed"
)

type referenceValidationResult struct {
	Valid   bool
	Reason  string
	Message string
	Err     error
}

type runtimeObservation struct {
	Deployment *appsv1.Deployment
	Service    *corev1.Service
}

func validReferenceValidation() referenceValidationResult {
	return referenceValidationResult{
		Valid:   true,
		Reason:  reasonReferencesResolved,
		Message: "All referenced Secrets are available.",
	}
}

// validateSecretReferences confirms that all provider and image-pull Secret
// references exist in the TrussiumRuntime namespace.
//
// Secret values are never read or copied.
func (r *TrussiumRuntimeReconciler) validateSecretReferences(
	ctx context.Context,
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) referenceValidationResult {
	for _, secretName := range referencedSecretNames(runtimeResource) {
		var secret corev1.Secret

		err := r.Get(
			ctx,
			client.ObjectKey{
				Name:      secretName,
				Namespace: runtimeResource.Namespace,
			},
			&secret,
		)
		if apierrors.IsNotFound(err) {
			return referenceValidationResult{
				Valid:  false,
				Reason: reasonSecretNotFound,
				Message: fmt.Sprintf(
					"Referenced Secret %q was not found.",
					secretName,
				),
			}
		}

		if err != nil {
			return referenceValidationResult{
				Valid:   false,
				Reason:  reasonReferenceCheckFailed,
				Message: "Unable to verify referenced Secrets.",
				Err: fmt.Errorf(
					"get referenced Secret %s/%s: %w",
					runtimeResource.Namespace,
					secretName,
					err,
				),
			}
		}
	}

	return validReferenceValidation()
}

func referencedSecretNames(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) []string {
	uniqueNames := make(map[string]struct{})

	credentialsReference :=
		runtimeResource.Spec.Provider.CredentialsSecretRef
	if credentialsReference != nil {
		uniqueNames[credentialsReference.Name] = struct{}{}
	}

	for _, reference := range runtimeResource.Spec.ImagePullSecrets {
		uniqueNames[reference.Name] = struct{}{}
	}

	names := make([]string, 0, len(uniqueNames))
	for name := range uniqueNames {
		names = append(names, name)
	}

	slices.Sort(names)

	return names
}

func runtimeReferencesSecret(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	secretName string,
) bool {
	return slices.Contains(
		referencedSecretNames(runtimeResource),
		secretName,
	)
}

// loadRuntimeObservation reads the currently managed Deployment and Service.
//
// Missing resources are represented as nil because reconciliation may be
// observing an object before all managed resources have been created.
func (r *TrussiumRuntimeReconciler) loadRuntimeObservation(
	ctx context.Context,
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) (runtimeObservation, error) {
	key := client.ObjectKeyFromObject(runtimeResource)
	observation := runtimeObservation{}

	var deployment appsv1.Deployment
	if err := r.Get(ctx, key, &deployment); err != nil {
		if !apierrors.IsNotFound(err) {
			return runtimeObservation{}, fmt.Errorf(
				"get managed Deployment %s/%s: %w",
				key.Namespace,
				key.Name,
				err,
			)
		}
	} else {
		observation.Deployment = &deployment
	}

	var service corev1.Service
	if err := r.Get(ctx, key, &service); err != nil {
		if !apierrors.IsNotFound(err) {
			return runtimeObservation{}, fmt.Errorf(
				"get managed Service %s/%s: %w",
				key.Namespace,
				key.Name,
				err,
			)
		}
	} else {
		observation.Service = &service
	}

	return observation, nil
}

func buildRuntimeStatus(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	observation runtimeObservation,
	validation referenceValidationResult,
) runtimev1alpha1.TrussiumRuntimeStatus {
	status := copyRuntimeStatus(runtimeResource.Status)

	status.ObservedGeneration = runtimeResource.Generation
	status.ReadyReplicas = 0
	status.AvailableReplicas = 0
	status.CurrentImage = deploymentCurrentImage(
		observation.Deployment,
	)
	status.Endpoint = runtimeServiceEndpoint(observation.Service)

	if observation.Deployment != nil {
		status.ReadyReplicas =
			observation.Deployment.Status.ReadyReplicas
		status.AvailableReplicas =
			observation.Deployment.Status.AvailableReplicas
	}

	if !validation.Valid {
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeConfigurationValid,
			metav1.ConditionFalse,
			validation.Reason,
			validation.Message,
		)
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeProgressing,
			metav1.ConditionFalse,
			reasonConfigurationInvalid,
			"Runtime reconciliation is blocked by invalid configuration references.",
		)
		setAvailabilityCondition(
			&status,
			runtimeResource.Generation,
			desiredReplicas(runtimeResource),
		)
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeReady,
			metav1.ConditionFalse,
			reasonConfigurationInvalid,
			"The runtime is not ready because configuration references are invalid.",
		)
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeDegraded,
			metav1.ConditionTrue,
			reasonConfigurationInvalid,
			validation.Message,
		)

		return status
	}

	setRuntimeCondition(
		&status,
		runtimeResource.Generation,
		conditionTypeConfigurationValid,
		metav1.ConditionTrue,
		reasonReferencesResolved,
		"All referenced Secrets are available.",
	)

	deployment := observation.Deployment
	desiredReplicaCount := desiredReplicas(runtimeResource)

	if deployment == nil {
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeProgressing,
			metav1.ConditionTrue,
			reasonDeploymentProgressing,
			"The runtime Deployment is being created.",
		)
		setAvailabilityCondition(
			&status,
			runtimeResource.Generation,
			desiredReplicaCount,
		)
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeReady,
			metav1.ConditionFalse,
			reasonRuntimeNotReady,
			"The runtime Deployment is not yet available.",
		)
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeDegraded,
			metav1.ConditionFalse,
			reasonAsExpected,
			"No degraded runtime condition has been observed.",
		)

		return status
	}

	deploymentObserved :=
		deployment.Status.ObservedGeneration >= deployment.Generation

	if desiredReplicaCount == 0 {
		if deploymentObserved &&
			status.ReadyReplicas == 0 &&
			status.AvailableReplicas == 0 {
			setRuntimeCondition(
				&status,
				runtimeResource.Generation,
				conditionTypeProgressing,
				metav1.ConditionFalse,
				reasonScaledToZero,
				"The runtime has reached the requested scaled-to-zero state.",
			)
			setRuntimeCondition(
				&status,
				runtimeResource.Generation,
				conditionTypeAvailable,
				metav1.ConditionFalse,
				reasonScaledToZero,
				"No runtime replicas are requested.",
			)
			setRuntimeCondition(
				&status,
				runtimeResource.Generation,
				conditionTypeReady,
				metav1.ConditionTrue,
				reasonScaledToZero,
				"The runtime has reached the requested scaled-to-zero state.",
			)
			setRuntimeCondition(
				&status,
				runtimeResource.Generation,
				conditionTypeDegraded,
				metav1.ConditionFalse,
				reasonAsExpected,
				"No degraded runtime condition has been observed.",
			)

			return status
		}

		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeProgressing,
			metav1.ConditionTrue,
			reasonDeploymentGenerationPending,
			"The Deployment is converging toward the scaled-to-zero state.",
		)
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeAvailable,
			metav1.ConditionFalse,
			reasonRuntimeUnavailable,
			"The runtime is not currently available.",
		)
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeReady,
			metav1.ConditionFalse,
			reasonRuntimeNotReady,
			"The runtime has not yet reached the scaled-to-zero state.",
		)
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeDegraded,
			metav1.ConditionFalse,
			reasonAsExpected,
			"No degraded runtime condition has been observed.",
		)

		return status
	}

	failureReason, failureMessage, deploymentFailed :=
		deploymentFailure(deployment)

	if deploymentFailed {
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeProgressing,
			metav1.ConditionFalse,
			reasonDeploymentFailed,
			failureMessage,
		)
		setAvailabilityCondition(
			&status,
			runtimeResource.Generation,
			desiredReplicaCount,
		)
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeReady,
			metav1.ConditionFalse,
			reasonDeploymentFailed,
			failureMessage,
		)
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeDegraded,
			metav1.ConditionTrue,
			failureReason,
			failureMessage,
		)

		return status
	}

	runtimeReady :=
		deploymentObserved &&
			status.ReadyReplicas == desiredReplicaCount &&
			status.AvailableReplicas == desiredReplicaCount

	if runtimeReady {
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeProgressing,
			metav1.ConditionFalse,
			reasonReconciliationComplete,
			"The runtime Deployment has reached the desired state.",
		)
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeAvailable,
			metav1.ConditionTrue,
			reasonRuntimeAvailable,
			fmt.Sprintf(
				"%d runtime replica(s) are available.",
				status.AvailableReplicas,
			),
		)
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeReady,
			metav1.ConditionTrue,
			reasonRuntimeReady,
			"All requested runtime replicas are ready and available.",
		)
		setRuntimeCondition(
			&status,
			runtimeResource.Generation,
			conditionTypeDegraded,
			metav1.ConditionFalse,
			reasonAsExpected,
			"No degraded runtime condition has been observed.",
		)

		return status
	}

	progressReason := reasonDeploymentProgressing
	progressMessage := fmt.Sprintf(
		"The runtime is progressing toward %d ready replica(s).",
		desiredReplicaCount,
	)

	if !deploymentObserved {
		progressReason = reasonDeploymentGenerationPending
		progressMessage =
			"The Deployment controller has not yet observed the current Deployment generation."
	}

	setRuntimeCondition(
		&status,
		runtimeResource.Generation,
		conditionTypeProgressing,
		metav1.ConditionTrue,
		progressReason,
		progressMessage,
	)
	setAvailabilityCondition(
		&status,
		runtimeResource.Generation,
		desiredReplicaCount,
	)
	setRuntimeCondition(
		&status,
		runtimeResource.Generation,
		conditionTypeReady,
		metav1.ConditionFalse,
		reasonRuntimeNotReady,
		fmt.Sprintf(
			"%d of %d requested runtime replica(s) are ready.",
			status.ReadyReplicas,
			desiredReplicaCount,
		),
	)
	setRuntimeCondition(
		&status,
		runtimeResource.Generation,
		conditionTypeDegraded,
		metav1.ConditionFalse,
		reasonAsExpected,
		"No degraded runtime condition has been observed.",
	)

	return status
}

func buildReconciliationFailureStatus(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	observation runtimeObservation,
	reconciliationError error,
) runtimev1alpha1.TrussiumRuntimeStatus {
	status := buildRuntimeStatus(
		runtimeResource,
		observation,
		validReferenceValidation(),
	)

	message := fmt.Sprintf(
		"Runtime reconciliation failed: %v",
		reconciliationError,
	)

	setRuntimeCondition(
		&status,
		runtimeResource.Generation,
		conditionTypeProgressing,
		metav1.ConditionFalse,
		reasonReconciliationFailed,
		message,
	)
	setRuntimeCondition(
		&status,
		runtimeResource.Generation,
		conditionTypeReady,
		metav1.ConditionFalse,
		reasonReconciliationFailed,
		message,
	)
	setRuntimeCondition(
		&status,
		runtimeResource.Generation,
		conditionTypeDegraded,
		metav1.ConditionTrue,
		reasonReconciliationFailed,
		message,
	)

	return status
}

func setAvailabilityCondition(
	status *runtimev1alpha1.TrussiumRuntimeStatus,
	observedGeneration int64,
	desiredReplicaCount int32,
) {
	if desiredReplicaCount == 0 {
		setRuntimeCondition(
			status,
			observedGeneration,
			conditionTypeAvailable,
			metav1.ConditionFalse,
			reasonScaledToZero,
			"No runtime replicas are requested.",
		)

		return
	}

	if status.AvailableReplicas > 0 {
		setRuntimeCondition(
			status,
			observedGeneration,
			conditionTypeAvailable,
			metav1.ConditionTrue,
			reasonRuntimeAvailable,
			fmt.Sprintf(
				"%d runtime replica(s) are available.",
				status.AvailableReplicas,
			),
		)

		return
	}

	setRuntimeCondition(
		status,
		observedGeneration,
		conditionTypeAvailable,
		metav1.ConditionFalse,
		reasonRuntimeUnavailable,
		"No runtime replicas are currently available.",
	)
}

func setRuntimeCondition(
	status *runtimev1alpha1.TrussiumRuntimeStatus,
	observedGeneration int64,
	conditionType string,
	conditionStatus metav1.ConditionStatus,
	reason string,
	message string,
) {
	meta.SetStatusCondition(
		&status.Conditions,
		metav1.Condition{
			Type:               conditionType,
			Status:             conditionStatus,
			ObservedGeneration: observedGeneration,
			Reason:             reason,
			Message:            message,
		},
	)
}

func runtimeCondition(
	status runtimev1alpha1.TrussiumRuntimeStatus,
	conditionType string,
) *metav1.Condition {
	return meta.FindStatusCondition(
		status.Conditions,
		conditionType,
	)
}

func conditionChanged(
	previousStatus runtimev1alpha1.TrussiumRuntimeStatus,
	currentStatus runtimev1alpha1.TrussiumRuntimeStatus,
	conditionType string,
) bool {
	previousCondition := runtimeCondition(
		previousStatus,
		conditionType,
	)
	currentCondition := runtimeCondition(
		currentStatus,
		conditionType,
	)

	if currentCondition == nil {
		return false
	}

	if previousCondition == nil {
		return true
	}

	return previousCondition.Status != currentCondition.Status ||
		previousCondition.Reason != currentCondition.Reason
}

func deploymentFailure(
	deployment *appsv1.Deployment,
) (string, string, bool) {
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentProgressing &&
			condition.Status == corev1.ConditionFalse &&
			condition.Reason == reasonProgressDeadlineExceeded {
			message := condition.Message
			if message == "" {
				message =
					"The runtime Deployment exceeded its progress deadline."
			}

			return reasonProgressDeadlineExceeded, message, true
		}
	}

	for _, condition := range deployment.Status.Conditions {
		if condition.Type == appsv1.DeploymentReplicaFailure &&
			condition.Status == corev1.ConditionTrue {
			message := condition.Message
			if message == "" {
				message =
					"The runtime Deployment reported a replica failure."
			}

			return reasonReplicaFailure, message, true
		}
	}

	return "", "", false
}

func deploymentCurrentImage(
	deployment *appsv1.Deployment,
) string {
	if deployment == nil {
		return ""
	}

	for _, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == runtimeContainerName {
			return container.Image
		}
	}

	if len(deployment.Spec.Template.Spec.Containers) == 0 {
		return ""
	}

	return deployment.Spec.Template.Spec.Containers[0].Image
}

func runtimeServiceEndpoint(
	service *corev1.Service,
) string {
	if service == nil || len(service.Spec.Ports) == 0 {
		return ""
	}

	return fmt.Sprintf(
		"http://%s.%s.svc.cluster.local:%d",
		service.Name,
		service.Namespace,
		service.Spec.Ports[0].Port,
	)
}

func copyRuntimeStatus(
	status runtimev1alpha1.TrussiumRuntimeStatus,
) runtimev1alpha1.TrussiumRuntimeStatus {
	copiedStatus := status
	copiedStatus.Conditions = append(
		[]metav1.Condition(nil),
		status.Conditions...,
	)

	return copiedStatus
}

// preserveRuntimeConditionTransitionTimes ensures LastTransitionTime changes
// only when the condition Status changes.
//
// Kubernetes API storage may normalize timestamp precision. Reusing the stored
// timestamp also prevents unnecessary status writes caused only by timestamp
// representation differences.
func preserveRuntimeConditionTransitionTimes(
	currentStatus runtimev1alpha1.TrussiumRuntimeStatus,
	desiredStatus runtimev1alpha1.TrussiumRuntimeStatus,
) runtimev1alpha1.TrussiumRuntimeStatus {
	normalizedStatus := copyRuntimeStatus(desiredStatus)

	for index := range normalizedStatus.Conditions {
		desiredCondition := &normalizedStatus.Conditions[index]

		currentCondition := runtimeCondition(
			currentStatus,
			desiredCondition.Type,
		)
		if currentCondition == nil {
			continue
		}

		if currentCondition.Status != desiredCondition.Status {
			continue
		}

		desiredCondition.LastTransitionTime =
			currentCondition.LastTransitionTime
	}

	return normalizedStatus
}

// updateRuntimeStatus writes through the status subresource only when the
// desired status differs from the stored status.
func (r *TrussiumRuntimeReconciler) updateRuntimeStatus(
	ctx context.Context,
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	desiredStatus runtimev1alpha1.TrussiumRuntimeStatus,
) (bool, error) {
	normalizedStatus := preserveRuntimeConditionTransitionTimes(
		runtimeResource.Status,
		desiredStatus,
	)

	if equality.Semantic.DeepEqual(
		runtimeResource.Status,
		normalizedStatus,
	) {
		return false, nil
	}

	runtimeResource.Status = normalizedStatus

	if err := r.Status().Update(ctx, runtimeResource); err != nil {
		return false, fmt.Errorf(
			"update TrussiumRuntime status %s/%s: %w",
			runtimeResource.Namespace,
			runtimeResource.Name,
			err,
		)
	}

	return true, nil
}
