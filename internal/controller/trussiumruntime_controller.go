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
	"errors"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

// TrussiumRuntimeReconciler reconciles a TrussiumRuntime resource.
type TrussiumRuntimeReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder
}

// +kubebuilder:rbac:groups=runtime.trussium.io,resources=trussiumruntimes,verbs=get;list;watch
// +kubebuilder:rbac:groups=runtime.trussium.io,resources=trussiumruntimes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=configmaps;serviceaccounts;services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch

// Reconcile ensures that the Kubernetes resources and observed status owned by
// a TrussiumRuntime match the custom-resource specification.
func (r *TrussiumRuntimeReconciler) Reconcile(
	ctx context.Context,
	req ctrl.Request,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var runtimeResource runtimev1alpha1.TrussiumRuntime
	if err := r.Get(ctx, req.NamespacedName, &runtimeResource); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !runtimeResource.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	previousStatus := copyRuntimeStatus(runtimeResource.Status)

	validation := r.validateSecretReferences(
		ctx,
		&runtimeResource,
	)

	if validation.Err != nil {
		return r.handleReconciliationFailure(
			ctx,
			&runtimeResource,
			previousStatus,
			validation.Err,
		)
	}

	if !validation.Valid {
		return r.reconcileInvalidConfiguration(
			ctx,
			&runtimeResource,
			previousStatus,
			validation,
		)
	}

	if err := r.reconcileCoreResources(
		ctx,
		&runtimeResource,
	); err != nil {
		return r.handleReconciliationFailure(
			ctx,
			&runtimeResource,
			previousStatus,
			err,
		)
	}

	observation, err := r.loadRuntimeObservation(
		ctx,
		&runtimeResource,
	)
	if err != nil {
		return r.handleReconciliationFailure(
			ctx,
			&runtimeResource,
			previousStatus,
			err,
		)
	}

	desiredStatus := buildRuntimeStatus(
		&runtimeResource,
		observation,
		validation,
	)

	statusUpdated, err := r.updateRuntimeStatus(
		ctx,
		&runtimeResource,
		desiredStatus,
	)
	if err != nil {
		r.recordReconciliationFailure(
			&runtimeResource,
			err,
		)

		return ctrl.Result{}, err
	}

	if statusUpdated {
		r.recordStatusTransitionEvents(
			&runtimeResource,
			previousStatus,
			runtimeResource.Status,
		)
	}

	logger.Info(
		"reconciled TrussiumRuntime resources and status",
		"name",
		runtimeResource.Name,
		"namespace",
		runtimeResource.Namespace,
		"statusUpdated",
		statusUpdated,
	)

	return ctrl.Result{}, nil
}

func (r *TrussiumRuntimeReconciler) reconcileInvalidConfiguration(
	ctx context.Context,
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	previousStatus runtimev1alpha1.TrussiumRuntimeStatus,
	validation referenceValidationResult,
) (ctrl.Result, error) {
	observation, err := r.loadRuntimeObservation(
		ctx,
		runtimeResource,
	)
	if err != nil {
		return r.handleReconciliationFailure(
			ctx,
			runtimeResource,
			previousStatus,
			err,
		)
	}

	desiredStatus := buildRuntimeStatus(
		runtimeResource,
		observation,
		validation,
	)

	statusUpdated, err := r.updateRuntimeStatus(
		ctx,
		runtimeResource,
		desiredStatus,
	)
	if err != nil {
		r.recordReconciliationFailure(
			runtimeResource,
			err,
		)

		return ctrl.Result{}, err
	}

	if statusUpdated {
		r.recordStatusTransitionEvents(
			runtimeResource,
			previousStatus,
			runtimeResource.Status,
		)
	}

	return ctrl.Result{}, nil
}

func (r *TrussiumRuntimeReconciler) reconcileCoreResources(
	ctx context.Context,
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) error {
	if err := r.reconcileConfigMap(ctx, runtimeResource); err != nil {
		return fmt.Errorf(
			"reconcile ConfigMap %s/%s: %w",
			runtimeResource.Namespace,
			runtimeResource.Name,
			err,
		)
	}

	if err := r.reconcileServiceAccount(
		ctx,
		runtimeResource,
	); err != nil {
		return fmt.Errorf(
			"reconcile ServiceAccount %s/%s: %w",
			runtimeResource.Namespace,
			runtimeResource.Name,
			err,
		)
	}

	if err := r.reconcileService(ctx, runtimeResource); err != nil {
		return fmt.Errorf(
			"reconcile Service %s/%s: %w",
			runtimeResource.Namespace,
			runtimeResource.Name,
			err,
		)
	}

	if err := r.reconcileDeployment(
		ctx,
		runtimeResource,
	); err != nil {
		return fmt.Errorf(
			"reconcile Deployment %s/%s: %w",
			runtimeResource.Namespace,
			runtimeResource.Name,
			err,
		)
	}

	return nil
}

func (r *TrussiumRuntimeReconciler) handleReconciliationFailure(
	ctx context.Context,
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	previousStatus runtimev1alpha1.TrussiumRuntimeStatus,
	reconciliationError error,
) (ctrl.Result, error) {
	observation, observationError := r.loadRuntimeObservation(
		ctx,
		runtimeResource,
	)
	if observationError != nil {
		reconciliationError = errors.Join(
			reconciliationError,
			observationError,
		)
	}

	desiredStatus := buildReconciliationFailureStatus(
		runtimeResource,
		observation,
		reconciliationError,
	)

	statusUpdated, statusError := r.updateRuntimeStatus(
		ctx,
		runtimeResource,
		desiredStatus,
	)
	if statusError != nil {
		reconciliationError = errors.Join(
			reconciliationError,
			statusError,
		)
	}

	if statusUpdated {
		r.recordStatusTransitionEvents(
			runtimeResource,
			previousStatus,
			runtimeResource.Status,
		)
	}

	r.recordReconciliationFailure(
		runtimeResource,
		reconciliationError,
	)

	return ctrl.Result{}, reconciliationError
}

func (r *TrussiumRuntimeReconciler) reconcileConfigMap(
	ctx context.Context,
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) error {
	desired := buildConfigMap(runtimeResource)

	current := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      desired.Name,
			Namespace: desired.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(
		ctx,
		r.Client,
		current,
		func() error {
			current.Labels = desired.Labels
			current.Data = desired.Data

			return controllerutil.SetControllerReference(
				runtimeResource,
				current,
				r.Scheme,
			)
		},
	)

	return err
}

func (r *TrussiumRuntimeReconciler) reconcileServiceAccount(
	ctx context.Context,
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) error {
	desired := buildServiceAccount(runtimeResource)

	current := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      desired.Name,
			Namespace: desired.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(
		ctx,
		r.Client,
		current,
		func() error {
			current.Labels = desired.Labels
			current.AutomountServiceAccountToken =
				desired.AutomountServiceAccountToken
			current.ImagePullSecrets = desired.ImagePullSecrets

			return controllerutil.SetControllerReference(
				runtimeResource,
				current,
				r.Scheme,
			)
		},
	)

	return err
}

func (r *TrussiumRuntimeReconciler) reconcileService(
	ctx context.Context,
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) error {
	desired := buildService(runtimeResource)

	current := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      desired.Name,
			Namespace: desired.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(
		ctx,
		r.Client,
		current,
		func() error {
			current.Labels = desired.Labels
			current.Spec.Type = desired.Spec.Type
			current.Spec.Selector = desired.Spec.Selector
			current.Spec.Ports = desired.Spec.Ports

			return controllerutil.SetControllerReference(
				runtimeResource,
				current,
				r.Scheme,
			)
		},
	)

	return err
}

func (r *TrussiumRuntimeReconciler) reconcileDeployment(
	ctx context.Context,
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) error {
	desired := buildDeployment(runtimeResource)

	current := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      desired.Name,
			Namespace: desired.Namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(
		ctx,
		r.Client,
		current,
		func() error {
			current.Labels = desired.Labels
			current.Spec = desired.Spec

			return controllerutil.SetControllerReference(
				runtimeResource,
				current,
				r.Scheme,
			)
		},
	)

	return err
}

func (r *TrussiumRuntimeReconciler) mapSecretToRuntimes(
	ctx context.Context,
	object client.Object,
) []reconcile.Request {
	var runtimeList runtimev1alpha1.TrussiumRuntimeList

	if err := r.List(
		ctx,
		&runtimeList,
		client.InNamespace(object.GetNamespace()),
	); err != nil {
		log.FromContext(ctx).Error(
			err,
			"unable to list TrussiumRuntime resources for Secret",
			"secret",
			object.GetName(),
			"namespace",
			object.GetNamespace(),
		)

		return nil
	}

	requests := make([]reconcile.Request, 0)

	for index := range runtimeList.Items {
		runtimeResource := &runtimeList.Items[index]

		if !runtimeReferencesSecret(
			runtimeResource,
			object.GetName(),
		) {
			continue
		}

		requests = append(
			requests,
			reconcile.Request{
				NamespacedName: client.ObjectKeyFromObject(
					runtimeResource,
				),
			},
		)
	}

	return requests
}

// SetupWithManager configures watches for TrussiumRuntime, owned Kubernetes
// resources, and referenced Secrets.
func (r *TrussiumRuntimeReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&runtimev1alpha1.TrussiumRuntime{},
			builder.WithPredicates(
				predicate.GenerationChangedPredicate{},
			),
		).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(
				r.mapSecretToRuntimes,
			),
		).
		Named("trussiumruntime").
		Complete(r)
}
