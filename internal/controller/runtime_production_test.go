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
	"encoding/json"
	"slices"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

const (
	testAdditionalPodLabelKey      = "team"
	testAdditionalPodLabelValue    = "ai-platform"
	testPodAnnotationKey           = "example.com/owner"
	testPodAnnotationValue         = "inference"
	testNodeSelectorKey            = "accelerator"
	testNodeSelectorValue          = "cpu"
	testReservedLabelOverrideValue = "must-not-win"
)

func TestBuildDeploymentProductionSecurityContext(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	deployment := buildDeployment(runtimeResource)

	podSpec := deployment.Spec.Template.Spec

	if podSpec.SecurityContext == nil {
		t.Fatal("expected Pod security context")
	}

	if podSpec.SecurityContext.RunAsNonRoot == nil ||
		!*podSpec.SecurityContext.RunAsNonRoot {
		t.Fatal("expected runAsNonRoot=true")
	}

	if podSpec.SecurityContext.RunAsUser == nil ||
		*podSpec.SecurityContext.RunAsUser != runtimeUserID {
		t.Fatalf(
			"expected runAsUser=%d",
			runtimeUserID,
		)
	}

	if podSpec.SecurityContext.RunAsGroup == nil ||
		*podSpec.SecurityContext.RunAsGroup != runtimeGroupID {
		t.Fatalf(
			"expected runAsGroup=%d",
			runtimeGroupID,
		)
	}

	if podSpec.SecurityContext.SeccompProfile == nil {
		t.Fatal("expected Pod seccomp profile")
	}

	if podSpec.SecurityContext.SeccompProfile.Type !=
		corev1.SeccompProfileTypeRuntimeDefault {
		t.Fatalf(
			"expected RuntimeDefault seccomp profile, received %q",
			podSpec.SecurityContext.SeccompProfile.Type,
		)
	}

	if len(podSpec.Containers) != 1 {
		t.Fatalf(
			"expected one runtime container, received %d",
			len(podSpec.Containers),
		)
	}

	container := podSpec.Containers[0]

	if container.SecurityContext == nil {
		t.Fatal("expected container security context")
	}

	if container.SecurityContext.AllowPrivilegeEscalation == nil ||
		*container.SecurityContext.AllowPrivilegeEscalation {
		t.Fatal(
			"expected allowPrivilegeEscalation=false",
		)
	}

	if container.SecurityContext.ReadOnlyRootFilesystem == nil ||
		!*container.SecurityContext.ReadOnlyRootFilesystem {
		t.Fatal(
			"expected readOnlyRootFilesystem=true",
		)
	}

	if container.SecurityContext.Privileged != nil &&
		*container.SecurityContext.Privileged {
		t.Fatal("runtime container must not be privileged")
	}

	if container.SecurityContext.Capabilities == nil {
		t.Fatal("expected container capabilities configuration")
	}

	if !slices.Contains(
		container.SecurityContext.Capabilities.Drop,
		corev1.Capability("ALL"),
	) {
		t.Fatalf(
			"expected all Linux capabilities to be dropped, received %#v",
			container.SecurityContext.Capabilities.Drop,
		)
	}

	if podSpec.AutomountServiceAccountToken == nil ||
		*podSpec.AutomountServiceAccountToken {
		t.Fatal(
			"expected service-account token automount to remain disabled",
		)
	}

	if podSpec.EnableServiceLinks == nil ||
		*podSpec.EnableServiceLinks {
		t.Fatal(
			"expected Kubernetes service links to remain disabled",
		)
	}
}

func TestBuildDeploymentHealthProbes(t *testing.T) {
	t.Parallel()

	deployment := buildDeployment(newTestRuntime())
	container := deployment.Spec.Template.Spec.Containers[0]

	assertHTTPProbe(
		t,
		"startup",
		container.StartupProbe,
		healthLivePath,
		2,
		1,
		30,
	)

	assertHTTPProbe(
		t,
		"liveness",
		container.LivenessProbe,
		healthLivePath,
		10,
		2,
		3,
	)

	assertHTTPProbe(
		t,
		"readiness",
		container.ReadinessProbe,
		healthReadyPath,
		5,
		2,
		3,
	)
}

func TestBuildDeploymentDefaultTerminationGracePeriod(t *testing.T) {
	t.Parallel()

	deployment := buildDeployment(newTestRuntime())

	actual :=
		deployment.Spec.Template.Spec.TerminationGracePeriodSeconds

	if actual == nil {
		t.Fatal("expected termination grace period")
	}

	if *actual != defaultTerminationGraceSeconds {
		t.Fatalf(
			"expected default termination grace period %d, received %d",
			defaultTerminationGraceSeconds,
			*actual,
		)
	}
}

func TestTerminationGracePeriodUsesConfiguredDrainTimeout(
	t *testing.T,
) {
	t.Parallel()

	runtimeResource := &runtimev1alpha1.TrussiumRuntime{}

	payload := []byte(`{
		"spec": {
			"runtime": {
				"shutdownDrainTimeoutSeconds": 45
			}
		}
	}`)

	if err := json.Unmarshal(payload, runtimeResource); err != nil {
		t.Fatalf(
			"unmarshal runtime drain configuration: %v",
			err,
		)
	}

	actual := terminationGracePeriodSeconds(runtimeResource)
	expected := int64(51)

	if actual != expected {
		t.Fatalf(
			"expected termination grace period %d, received %d",
			expected,
			actual,
		)
	}
}

func TestBuildDeploymentRollingUpdateStrategy(t *testing.T) {
	t.Parallel()

	deployment := buildDeployment(newTestRuntime())

	if deployment.Spec.RevisionHistoryLimit == nil {
		t.Fatal("expected revision history limit")
	}

	if *deployment.Spec.RevisionHistoryLimit !=
		deploymentRevisionHistoryLimit {
		t.Fatalf(
			"expected revision history limit %d, received %d",
			deploymentRevisionHistoryLimit,
			*deployment.Spec.RevisionHistoryLimit,
		)
	}

	if deployment.Spec.Strategy.Type !=
		"RollingUpdate" {
		t.Fatalf(
			"expected RollingUpdate strategy, received %q",
			deployment.Spec.Strategy.Type,
		)
	}

	rollingUpdate := deployment.Spec.Strategy.RollingUpdate
	if rollingUpdate == nil {
		t.Fatal("expected rolling-update configuration")
	}

	assertIntOrStringValue(
		t,
		"maxUnavailable",
		rollingUpdate.MaxUnavailable,
		0,
	)

	assertIntOrStringValue(
		t,
		"maxSurge",
		rollingUpdate.MaxSurge,
		1,
	)
}

func TestBuildDeploymentTopologySpreadConstraint(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	deployment := buildDeployment(runtimeResource)

	constraints :=
		deployment.Spec.Template.Spec.TopologySpreadConstraints

	if len(constraints) != 1 {
		t.Fatalf(
			"expected one topology-spread constraint, received %d",
			len(constraints),
		)
	}

	constraint := constraints[0]

	if constraint.MaxSkew != 1 {
		t.Fatalf(
			"expected maxSkew=1, received %d",
			constraint.MaxSkew,
		)
	}

	if constraint.TopologyKey != topologyHostnameKey {
		t.Fatalf(
			"expected topology key %q, received %q",
			topologyHostnameKey,
			constraint.TopologyKey,
		)
	}

	if constraint.WhenUnsatisfiable != corev1.ScheduleAnyway {
		t.Fatalf(
			"expected ScheduleAnyway, received %q",
			constraint.WhenUnsatisfiable,
		)
	}

	if constraint.LabelSelector == nil {
		t.Fatal("expected topology label selector")
	}

	expectedLabels := runtimeSelectorLabels(runtimeResource)

	if len(constraint.LabelSelector.MatchLabels) !=
		len(expectedLabels) {
		t.Fatalf(
			"expected topology selector %#v, received %#v",
			expectedLabels,
			constraint.LabelSelector.MatchLabels,
		)
	}

	for key, expectedValue := range expectedLabels {
		if actualValue :=
			constraint.LabelSelector.MatchLabels[key]; actualValue != expectedValue {
			t.Fatalf(
				"expected selector %s=%q, received %q",
				key,
				expectedValue,
				actualValue,
			)
		}
	}
}

func TestBuildDeploymentProjectsPodMetadata(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()

	reservedLabels := runtimeLabels(runtimeResource)
	var reservedKey string
	var reservedValue string

	for key, value := range reservedLabels {
		reservedKey = key
		reservedValue = value
		break
	}

	if reservedKey == "" {
		t.Fatal("expected at least one operator-owned label")
	}

	runtimeResource.Spec.PodMetadata =
		&runtimev1alpha1.PodMetadataSpec{
			Labels: map[string]string{
				testAdditionalPodLabelKey: testAdditionalPodLabelValue,
				reservedKey:               testReservedLabelOverrideValue,
			},
			Annotations: map[string]string{
				testPodAnnotationKey: testPodAnnotationValue,
			},
		}

	deployment := buildDeployment(runtimeResource)
	template := deployment.Spec.Template

	if actual :=
		template.Labels[testAdditionalPodLabelKey]; actual != testAdditionalPodLabelValue {
		t.Fatalf(
			"expected additional Pod label %q, received %q",
			testAdditionalPodLabelValue,
			actual,
		)
	}

	if actual := template.Labels[reservedKey]; actual != reservedValue {
		t.Fatalf(
			"operator-owned label %q was overridden: expected %q, received %q",
			reservedKey,
			reservedValue,
			actual,
		)
	}

	if actual :=
		template.Annotations[testPodAnnotationKey]; actual != testPodAnnotationValue {
		t.Fatalf(
			"expected Pod annotation %q, received %q",
			testPodAnnotationValue,
			actual,
		)
	}
}

func TestBuildDeploymentProjectsScheduling(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()

	runtimeResource.Spec.Scheduling =
		&runtimev1alpha1.SchedulingSpec{
			NodeSelector: map[string]string{
				testNodeSelectorKey: testNodeSelectorValue,
			},
			Tolerations: []corev1.Toleration{
				{
					Key:      "workload",
					Operator: corev1.TolerationOpEqual,
					Value:    "ai",
					Effect:   corev1.TaintEffectNoSchedule,
				},
			},
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: []corev1.PreferredSchedulingTerm{
						{
							Weight: 1,
							Preference: corev1.NodeSelectorTerm{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      testNodeSelectorKey,
										Operator: corev1.NodeSelectorOpIn,
										Values: []string{
											testNodeSelectorValue,
										},
									},
								},
							},
						},
					},
				},
			},
		}

	deployment := buildDeployment(runtimeResource)
	podSpec := deployment.Spec.Template.Spec

	if actual :=
		podSpec.NodeSelector[testNodeSelectorKey]; actual != testNodeSelectorValue {
		t.Fatalf(
			"expected node selector value %q, received %q",
			testNodeSelectorValue,
			actual,
		)
	}

	if len(podSpec.Tolerations) != 1 {
		t.Fatalf(
			"expected one toleration, received %d",
			len(podSpec.Tolerations),
		)
	}

	if podSpec.Tolerations[0].Key != "workload" {
		t.Fatalf(
			"expected workload toleration, received %#v",
			podSpec.Tolerations[0],
		)
	}

	if podSpec.Affinity == nil ||
		podSpec.Affinity.NodeAffinity == nil {
		t.Fatal("expected node affinity projection")
	}
}

func TestBuildPodDisruptionBudget(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	pdb := buildPodDisruptionBudget(runtimeResource)

	if pdb.Name != runtimeResource.Name {
		t.Fatalf(
			"expected PDB name %q, received %q",
			runtimeResource.Name,
			pdb.Name,
		)
	}

	if pdb.Namespace != runtimeResource.Namespace {
		t.Fatalf(
			"expected PDB namespace %q, received %q",
			runtimeResource.Namespace,
			pdb.Namespace,
		)
	}

	if pdb.Spec.MaxUnavailable == nil {
		t.Fatal("expected PDB maxUnavailable")
	}

	if pdb.Spec.MaxUnavailable.Type != intstr.Int ||
		pdb.Spec.MaxUnavailable.IntVal != 1 {
		t.Fatalf(
			"expected PDB maxUnavailable=1, received %#v",
			pdb.Spec.MaxUnavailable,
		)
	}

	if pdb.Spec.Selector == nil {
		t.Fatal("expected PDB selector")
	}

	expectedLabels := runtimeSelectorLabels(runtimeResource)

	for key, expectedValue := range expectedLabels {
		if actualValue := pdb.Spec.Selector.MatchLabels[key]; actualValue != expectedValue {
			t.Fatalf(
				"expected PDB selector %s=%q, received %q",
				key,
				expectedValue,
				actualValue,
			)
		}
	}
}

func assertHTTPProbe(
	t *testing.T,
	name string,
	probe *corev1.Probe,
	expectedPath string,
	expectedPeriod int32,
	expectedTimeout int32,
	expectedFailureThreshold int32,
) {
	t.Helper()

	if probe == nil {
		t.Fatalf("expected %s probe", name)
	}

	if probe.HTTPGet == nil {
		t.Fatalf(
			"expected %s probe to use HTTP GET",
			name,
		)
	}

	if probe.HTTPGet.Path != expectedPath {
		t.Fatalf(
			"expected %s path %q, received %q",
			name,
			expectedPath,
			probe.HTTPGet.Path,
		)
	}

	if probe.HTTPGet.Port.Type != intstr.String ||
		probe.HTTPGet.Port.StrVal != runtimeHTTPPortName {
		t.Fatalf(
			"expected %s probe port %q, received %#v",
			name,
			runtimeHTTPPortName,
			probe.HTTPGet.Port,
		)
	}

	if probe.PeriodSeconds != expectedPeriod {
		t.Fatalf(
			"expected %s period %d, received %d",
			name,
			expectedPeriod,
			probe.PeriodSeconds,
		)
	}

	if probe.TimeoutSeconds != expectedTimeout {
		t.Fatalf(
			"expected %s timeout %d, received %d",
			name,
			expectedTimeout,
			probe.TimeoutSeconds,
		)
	}

	if probe.FailureThreshold != expectedFailureThreshold {
		t.Fatalf(
			"expected %s failure threshold %d, received %d",
			name,
			expectedFailureThreshold,
			probe.FailureThreshold,
		)
	}
}

func assertIntOrStringValue(
	t *testing.T,
	name string,
	value *intstr.IntOrString,
	expected int32,
) {
	t.Helper()

	if value == nil {
		t.Fatalf("expected %s", name)
	}

	if value.Type != intstr.Int ||
		value.IntVal != expected {
		t.Fatalf(
			"expected %s=%d, received %#v",
			name,
			expected,
			value,
		)
	}
}
