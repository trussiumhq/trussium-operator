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

package v1alpha1

import (
	"encoding/json"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestProductionPodConfigurationJSONRoundTrip(
	t *testing.T,
) {
	t.Parallel()

	original := TrussiumRuntime{
		Spec: TrussiumRuntimeSpec{
			PodMetadata: &PodMetadataSpec{
				Labels: map[string]string{
					"team": "ai-platform",
				},
				Annotations: map[string]string{
					"example.com/owner": "inference",
				},
			},
			Scheduling: &SchedulingSpec{
				NodeSelector: map[string]string{
					"accelerator": "cpu",
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
					NodeAffinity: &corev1.NodeAffinity{},
				},
			},
		},
	}

	payload, err := json.Marshal(original)
	if err != nil {
		t.Fatalf(
			"marshal TrussiumRuntime: %v",
			err,
		)
	}

	var decoded TrussiumRuntime

	if err := json.Unmarshal(
		payload,
		&decoded,
	); err != nil {
		t.Fatalf(
			"unmarshal TrussiumRuntime: %v",
			err,
		)
	}

	if !reflect.DeepEqual(
		original.Spec.PodMetadata,
		decoded.Spec.PodMetadata,
	) {
		t.Fatalf(
			"Pod metadata changed during JSON round trip: expected %#v, received %#v",
			original.Spec.PodMetadata,
			decoded.Spec.PodMetadata,
		)
	}

	if !reflect.DeepEqual(
		original.Spec.Scheduling,
		decoded.Spec.Scheduling,
	) {
		t.Fatalf(
			"scheduling changed during JSON round trip: expected %#v, received %#v",
			original.Spec.Scheduling,
			decoded.Spec.Scheduling,
		)
	}
}
