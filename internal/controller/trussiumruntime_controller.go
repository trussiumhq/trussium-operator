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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	runtimev1alpha1 "github.com/trussium/trussium-operator/api/v1alpha1"
)

// TrussiumRuntimeReconciler reconciles a TrussiumRuntime resource.
type TrussiumRuntimeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=runtime.trussium.io,resources=trussiumruntimes,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps;serviceaccounts;services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile ensures that the Kubernetes resources owned by a TrussiumRuntime
// match the desired custom-resource specification.
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

	if err := r.reconcileConfigMap(ctx, &runtimeResource); err != nil {
		return ctrl.Result{}, fmt.Errorf(
			"reconcile ConfigMap %s/%s: %w",
			runtimeResource.Namespace,
			runtimeResource.Name,
			err,
		)
	}

	if err := r.reconcileServiceAccount(ctx, &runtimeResource); err != nil {
		return ctrl.Result{}, fmt.Errorf(
			"reconcile ServiceAccount %s/%s: %w",
			runtimeResource.Namespace,
			runtimeResource.Name,
			err,
		)
	}

	if err := r.reconcileService(ctx, &runtimeResource); err != nil {
		return ctrl.Result{}, fmt.Errorf(
			"reconcile Service %s/%s: %w",
			runtimeResource.Namespace,
			runtimeResource.Name,
			err,
		)
	}

	if err := r.reconcileDeployment(ctx, &runtimeResource); err != nil {
		return ctrl.Result{}, fmt.Errorf(
			"reconcile Deployment %s/%s: %w",
			runtimeResource.Namespace,
			runtimeResource.Name,
			err,
		)
	}

	logger.Info(
		"reconciled TrussiumRuntime resources",
		"name",
		runtimeResource.Name,
		"namespace",
		runtimeResource.Namespace,
	)

	return ctrl.Result{}, nil
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

// SetupWithManager configures watches for TrussiumRuntime and its owned
// Kubernetes resources.
func (r *TrussiumRuntimeReconciler) SetupWithManager(
	mgr ctrl.Manager,
) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&runtimev1alpha1.TrussiumRuntime{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&appsv1.Deployment{}).
		Named("trussiumruntime").
		Complete(r)
}
