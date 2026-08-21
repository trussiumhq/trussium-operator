/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package integration

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

func TestOptInAutoscalingLifecycle(t *testing.T) {
	namespace := createTestNamespace(t)
	runtimeResource := createTestRuntime(t, namespace.Name)
	key := client.ObjectKeyFromObject(runtimeResource)

	updateRuntimeAutoscaling(t, key, true)
	hpa := waitForObject(t, key, &autoscalingv2.HorizontalPodAutoscaler{})
	assertControllerOwner(t, hpa, runtimeResource)
	if hpa.Spec.MinReplicas == nil || *hpa.Spec.MinReplicas != 2 || hpa.Spec.MaxReplicas != 8 {
		t.Fatalf("unexpected HPA bounds: %#v", hpa.Spec)
	}

	var deployment appsv1.Deployment
	if err := testClient.Get(context.Background(), key, &deployment); err != nil {
		t.Fatalf("get managed Deployment: %v", err)
	}
	scaled := int32(5)
	deployment.Spec.Replicas = &scaled
	if err := testClient.Update(context.Background(), &deployment); err != nil {
		t.Fatalf("set HPA scale decision: %v", err)
	}
	eventually(t, "HPA scale decision is preserved", func() (bool, error) {
		var current appsv1.Deployment
		if err := testClient.Get(context.Background(), key, &current); err != nil {
			return false, err
		}
		return current.Spec.Replicas != nil && *current.Spec.Replicas == scaled, nil
	})

	hpa.Spec.MaxReplicas = 3
	if err := testClient.Update(context.Background(), hpa); err != nil {
		t.Fatalf("introduce HPA drift: %v", err)
	}
	eventually(t, "HPA drift is corrected", func() (bool, error) {
		var current autoscalingv2.HorizontalPodAutoscaler
		if err := testClient.Get(context.Background(), key, &current); err != nil {
			return false, err
		}
		return current.Spec.MaxReplicas == 8, nil
	})

	oldUID := hpa.UID
	deleteManagedObject(t, hpa)
	waitForRecreatedObject(t, key, &autoscalingv2.HorizontalPodAutoscaler{}, oldUID)

	updateRuntimeAutoscaling(t, key, false)
	eventually(t, "HPA is removed", func() (bool, error) {
		var current autoscalingv2.HorizontalPodAutoscaler
		err := testClient.Get(context.Background(), key, &current)
		return apierrors.IsNotFound(err), nil
	})
}

func updateRuntimeAutoscaling(t *testing.T, key client.ObjectKey, enabled bool) {
	t.Helper()
	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var current runtimev1alpha1.TrussiumRuntime
		if err := testClient.Get(context.Background(), key, &current); err != nil {
			return err
		}
		current.Spec.Autoscaling = &runtimev1alpha1.RuntimeAutoscalingSpec{Enabled: enabled, MinReplicas: 2, MaxReplicas: 8, TargetCPUUtilizationPercentage: 65}
		return testClient.Update(context.Background(), &current)
	})
	if err != nil {
		t.Fatalf("update autoscaling: %v", err)
	}
}
