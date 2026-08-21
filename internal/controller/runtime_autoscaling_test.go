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
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

func TestBuildHorizontalPodAutoscaler(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	runtimeResource.Spec.Autoscaling = &runtimev1alpha1.RuntimeAutoscalingSpec{
		Enabled:                        true,
		MinReplicas:                    2,
		MaxReplicas:                    8,
		TargetCPUUtilizationPercentage: 65,
	}

	hpa := buildHorizontalPodAutoscaler(runtimeResource)
	if hpa.Spec.MinReplicas == nil || *hpa.Spec.MinReplicas != 2 || hpa.Spec.MaxReplicas != 8 {
		t.Fatalf("unexpected HPA replica bounds: %#v", hpa.Spec)
	}
	if hpa.Spec.ScaleTargetRef.APIVersion != "apps/v1" || hpa.Spec.ScaleTargetRef.Kind != "Deployment" || hpa.Spec.ScaleTargetRef.Name != runtimeResource.Name {
		t.Fatalf("unexpected HPA target: %#v", hpa.Spec.ScaleTargetRef)
	}
	if len(hpa.Spec.Metrics) != 1 || hpa.Spec.Metrics[0].Resource == nil || hpa.Spec.Metrics[0].Resource.Target.AverageUtilization == nil || *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization != 65 {
		t.Fatalf("unexpected HPA metrics: %#v", hpa.Spec.Metrics)
	}
}

func TestReconcilePreservesHPAReplicaDecision(t *testing.T) {
	ctx := context.Background()
	scheme := newTestScheme(t)
	runtimeResource := newTestRuntime()
	runtimeResource.UID = types.UID("runtime-hpa-owner-uid")
	runtimeResource.Spec.Autoscaling = &runtimev1alpha1.RuntimeAutoscalingSpec{Enabled: true, MinReplicas: 2, MaxReplicas: 8, TargetCPUUtilizationPercentage: 65}

	kubernetesClient := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&runtimev1alpha1.TrussiumRuntime{}).WithObjects(runtimeResource).Build()
	reconciler := TrussiumRuntimeReconciler{Client: kubernetesClient, Scheme: scheme}
	request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(runtimeResource)}
	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("initial reconciliation: %v", err)
	}

	var hpa autoscalingv2.HorizontalPodAutoscaler
	if err := kubernetesClient.Get(ctx, request.NamespacedName, &hpa); err != nil {
		t.Fatalf("get managed HPA: %v", err)
	}
	if len(hpa.OwnerReferences) != 1 || hpa.OwnerReferences[0].UID != runtimeResource.UID {
		t.Fatalf("unexpected HPA owner references: %#v", hpa.OwnerReferences)
	}

	var deployment appsv1.Deployment
	if err := kubernetesClient.Get(ctx, request.NamespacedName, &deployment); err != nil {
		t.Fatalf("get managed Deployment: %v", err)
	}
	decision := int32(5)
	deployment.Spec.Replicas = &decision
	if err := kubernetesClient.Update(ctx, &deployment); err != nil {
		t.Fatalf("apply HPA replica decision: %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("reconcile HPA-managed Deployment: %v", err)
	}
	if err := kubernetesClient.Get(ctx, request.NamespacedName, &deployment); err != nil {
		t.Fatalf("get reconciled Deployment: %v", err)
	}
	if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas != decision {
		t.Fatalf("expected HPA replica decision %d to be preserved, received %#v", decision, deployment.Spec.Replicas)
	}

	var storedRuntime runtimev1alpha1.TrussiumRuntime
	if err := kubernetesClient.Get(ctx, request.NamespacedName, &storedRuntime); err != nil {
		t.Fatalf("get managed TrussiumRuntime: %v", err)
	}
	storedRuntime.Spec.Autoscaling.Enabled = false
	if err := kubernetesClient.Update(ctx, &storedRuntime); err != nil {
		t.Fatalf("disable autoscaling: %v", err)
	}
	if _, err := reconciler.Reconcile(ctx, request); err != nil {
		t.Fatalf("reconcile disabled autoscaling: %v", err)
	}
	if err := kubernetesClient.Get(ctx, request.NamespacedName, &hpa); err == nil {
		t.Fatal("expected HPA deletion after autoscaling is disabled")
	}
	if err := kubernetesClient.Get(ctx, request.NamespacedName, &deployment); err != nil {
		t.Fatalf("get Deployment after autoscaling is disabled: %v", err)
	}
	if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas != 2 {
		t.Fatalf("expected spec replica count restoration, received %#v", deployment.Spec.Replicas)
	}
}
