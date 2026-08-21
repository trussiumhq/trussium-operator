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

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

func TestOptInNetworkPolicyLifecycle(t *testing.T) {
	namespace := createTestNamespace(t)
	runtimeResource := createTestRuntime(t, namespace.Name)
	key := client.ObjectKeyFromObject(runtimeResource)

	assertNetworkPolicyAbsent(t, key)

	updateRuntimeNetworkPolicy(t, key, true)

	policy := waitForObject(t, key, &networkingv1.NetworkPolicy{})
	assertControllerOwner(t, policy, runtimeResource)
	assertNetworkPolicyDesiredState(t, policy, runtimeResource.Name)

	drifted := policy.DeepCopy()
	drifted.Labels = map[string]string{driftedLabelKey: driftedLabelValue}
	drifted.Spec.Ingress = nil
	if err := testClient.Update(context.Background(), drifted); err != nil {
		t.Fatalf("introduce NetworkPolicy drift: %v", err)
	}

	eventually(
		t,
		"NetworkPolicy desired state restored",
		func() (bool, error) {
			var current networkingv1.NetworkPolicy
			if err := testClient.Get(context.Background(), key, &current); err != nil {
				return false, err
			}

			return networkPolicyMatchesDesiredState(&current, runtimeResource.Name), nil
		},
	)

	restored := waitForObject(t, key, &networkingv1.NetworkPolicy{})
	oldUID := restored.UID
	deleteManagedObject(t, restored)
	recreated := waitForRecreatedObject(t, key, &networkingv1.NetworkPolicy{}, oldUID)
	assertControllerOwner(t, recreated, runtimeResource)
	assertNetworkPolicyDesiredState(t, recreated, runtimeResource.Name)

	updateRuntimeNetworkPolicy(t, key, false)
	assertNetworkPolicyAbsent(t, key)
}

func TestNetworkPolicyEnabledRequiresIngress(t *testing.T) {
	namespace := createTestNamespace(t)
	runtimeResource := createTestRuntime(t, namespace.Name)
	key := client.ObjectKeyFromObject(runtimeResource)

	var current runtimev1alpha1.TrussiumRuntime
	if err := testClient.Get(context.Background(), key, &current); err != nil {
		t.Fatalf("get managed TrussiumRuntime: %v", err)
	}

	current.Spec.NetworkPolicy = &runtimev1alpha1.RuntimeNetworkPolicySpec{
		Enabled: true,
	}

	err := testClient.Update(context.Background(), &current)
	if !apierrors.IsInvalid(err) {
		t.Fatalf("expected API validation error for missing ingress rules, received: %v", err)
	}
}

func updateRuntimeNetworkPolicy(
	t *testing.T,
	key client.ObjectKey,
	enabled bool,
) {
	t.Helper()

	err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var runtimeResource runtimev1alpha1.TrussiumRuntime
		if err := testClient.Get(context.Background(), key, &runtimeResource); err != nil {
			return err
		}

		runtimeResource.Spec.NetworkPolicy = &runtimev1alpha1.RuntimeNetworkPolicySpec{
			Enabled: enabled,
			Ingress: []runtimev1alpha1.RuntimeNetworkPolicyIngressRule{{
				NamespaceSelector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"kubernetes.io/metadata.name": "clients",
					},
				},
			}},
		}

		return testClient.Update(context.Background(), &runtimeResource)
	})
	if err != nil {
		t.Fatalf("update TrussiumRuntime NetworkPolicy: %v", err)
	}
}

func assertNetworkPolicyAbsent(t *testing.T, key client.ObjectKey) {
	t.Helper()

	eventually(t, "NetworkPolicy is absent", func() (bool, error) {
		var policy networkingv1.NetworkPolicy
		err := testClient.Get(context.Background(), key, &policy)

		return apierrors.IsNotFound(err), nil
	})
}

func assertNetworkPolicyDesiredState(
	t *testing.T,
	policy *networkingv1.NetworkPolicy,
	runtimeName string,
) {
	t.Helper()

	if !networkPolicyMatchesDesiredState(policy, runtimeName) {
		t.Fatalf("NetworkPolicy %s/%s does not match desired state: %#v", policy.Namespace, policy.Name, policy.Spec)
	}
}

func networkPolicyMatchesDesiredState(
	policy *networkingv1.NetworkPolicy,
	runtimeName string,
) bool {
	if policy.Labels[runtimeNameLabel] != expectedRuntimeName ||
		policy.Labels[runtimeManagedByLabel] != expectedRuntimeManagedBy ||
		policy.Spec.PodSelector.MatchLabels[runtimeNameLabel] != expectedRuntimeName ||
		policy.Spec.PodSelector.MatchLabels[runtimeInstanceLabel] != runtimeName ||
		len(policy.Spec.PolicyTypes) != 1 ||
		policy.Spec.PolicyTypes[0] != networkingv1.PolicyTypeIngress ||
		len(policy.Spec.Ingress) != 1 {
		return false
	}

	rule := policy.Spec.Ingress[0]
	if len(rule.From) != 1 || len(rule.Ports) != 1 ||
		rule.From[0].NamespaceSelector == nil ||
		rule.From[0].NamespaceSelector.MatchLabels["kubernetes.io/metadata.name"] != "clients" {
		return false
	}

	port := rule.Ports[0]
	return port.Protocol != nil && *port.Protocol == corev1.ProtocolTCP &&
		port.Port != nil && *port.Port == intstr.FromInt32(9000)
}
