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
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

const (
	testRuntimeImageRepository = "ghcr.io/trussiumhq/trussium"
	testRuntimeImageTag        = "0.23.0"
	testRuntimeModel           = "llama3.2"
)

func createTestNamespace(
	t *testing.T,
) *corev1.Namespace {
	t.Helper()

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "trussium-integration-",
		},
	}

	if err := testClient.Create(
		context.Background(),
		namespace,
	); err != nil {
		t.Fatalf("create integration namespace: %v", err)
	}

	return namespace
}

func createTestRuntime(
	t *testing.T,
	namespace string,
) *runtimev1alpha1.TrussiumRuntime {
	t.Helper()

	replicas := int32(1)
	tag := testRuntimeImageTag

	runtimeResource := &runtimev1alpha1.TrussiumRuntime{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "runtime",
			Namespace: namespace,
		},
		Spec: runtimev1alpha1.TrussiumRuntimeSpec{
			Image: runtimev1alpha1.RuntimeImageSpec{
				Repository: testRuntimeImageRepository,
				Tag:        &tag,
				PullPolicy: corev1.PullIfNotPresent,
			},
			Replicas: &replicas,
			Provider: runtimev1alpha1.ProviderSpec{
				Type:  runtimev1alpha1.ProviderTypeOllama,
				Model: testRuntimeModel,
			},
			Service: runtimev1alpha1.RuntimeServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Port: 9000,
			},
		},
	}

	if err := testClient.Create(
		context.Background(),
		runtimeResource,
	); err != nil {
		t.Fatalf("create TrussiumRuntime: %v", err)
	}

	return runtimeResource
}

func waitForObject[T client.Object](
	t *testing.T,
	key client.ObjectKey,
	object T,
) T {
	t.Helper()

	eventually(
		t,
		fmt.Sprintf("%T %s", object, key.String()),
		func() (bool, error) {
			err := testClient.Get(
				context.Background(),
				key,
				object,
			)
			if apierrors.IsNotFound(err) {
				return false, nil
			}

			if err != nil {
				return false, err
			}

			return true, nil
		},
	)

	return object
}

func waitForRuntimeStatus(
	t *testing.T,
	key client.ObjectKey,
	condition func(
		*runtimev1alpha1.TrussiumRuntime,
	) bool,
) *runtimev1alpha1.TrussiumRuntime {
	t.Helper()

	var current runtimev1alpha1.TrussiumRuntime

	eventually(
		t,
		fmt.Sprintf(
			"TrussiumRuntime status %s",
			key.String(),
		),
		func() (bool, error) {
			if err := testClient.Get(
				context.Background(),
				key,
				&current,
			); err != nil {
				return false, err
			}

			return condition(&current), nil
		},
	)

	return current.DeepCopy()
}

func eventually(
	t *testing.T,
	description string,
	condition func() (bool, error),
) {
	t.Helper()

	deadline := time.Now().Add(integrationTimeout)

	var lastError error

	for time.Now().Before(deadline) {
		done, err := condition()
		if err != nil {
			lastError = err
		} else if done {
			return
		}

		time.Sleep(integrationPollInterval)
	}

	if lastError != nil {
		t.Fatalf(
			"timed out waiting for %s: %v",
			description,
			lastError,
		)
	}

	t.Fatalf(
		"timed out waiting for %s",
		description,
	)
}

func assertControllerOwner(
	t *testing.T,
	child metav1.Object,
	owner *runtimev1alpha1.TrussiumRuntime,
) {
	t.Helper()

	for _, reference := range child.GetOwnerReferences() {
		if reference.UID != owner.UID {
			continue
		}

		if reference.Controller == nil ||
			!*reference.Controller {
			t.Fatalf(
				"%T %s/%s owner reference for %s is not a controller reference",
				child,
				child.GetNamespace(),
				child.GetName(),
				owner.Name,
			)
		}

		if reference.APIVersion != runtimev1alpha1.GroupVersion.String() {
			t.Fatalf(
				"%T %s/%s has owner apiVersion %q, expected %q",
				child,
				child.GetNamespace(),
				child.GetName(),
				reference.APIVersion,
				runtimev1alpha1.GroupVersion.String(),
			)
		}

		if reference.Kind != "TrussiumRuntime" {
			t.Fatalf(
				"%T %s/%s has owner kind %q, expected TrussiumRuntime",
				child,
				child.GetNamespace(),
				child.GetName(),
				reference.Kind,
			)
		}

		return
	}

	t.Fatalf(
		"%T %s/%s is not controlled by TrussiumRuntime %s/%s",
		child,
		child.GetNamespace(),
		child.GetName(),
		owner.Namespace,
		owner.Name,
	)
}

func runtimeCondition(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
	conditionType string,
) *metav1.Condition {
	return meta.FindStatusCondition(
		runtimeResource.Status.Conditions,
		conditionType,
	)
}
