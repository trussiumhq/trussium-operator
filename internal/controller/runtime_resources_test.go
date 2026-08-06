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
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	runtimev1alpha1 "github.com/trussium/trussium-operator/api/v1alpha1"
)

func TestRuntimeImageWithTag(t *testing.T) {
	t.Parallel()

	tag := testRuntimeTag

	image := runtimeImage(runtimev1alpha1.RuntimeImageSpec{
		Repository: testRuntimeRepository,
		Tag:        &tag,
	})

	expected := testRuntimeImage
	if image != expected {
		t.Fatalf("expected image %q, received %q", expected, image)
	}
}

func TestRuntimeImageWithDigest(t *testing.T) {
	t.Parallel()

	digest := "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	image := runtimeImage(runtimev1alpha1.RuntimeImageSpec{
		Repository: testRuntimeRepository,
		Digest:     &digest,
	})

	expected := testRuntimeRepository + "@" + digest
	if image != expected {
		t.Fatalf("expected image %q, received %q", expected, image)
	}
}

func TestRuntimeConfigData(t *testing.T) {
	t.Parallel()

	baseURL := "http://ollama.ollama.svc.cluster.local:11434/v1"
	providerTimeout := int32(75)
	streamTimeout := int32(40)
	shutdownTimeout := int32(50)

	runtimeResource := newTestRuntime()
	runtimeResource.Spec.Provider.BaseURL = &baseURL
	runtimeResource.Spec.Runtime = &runtimev1alpha1.RuntimeSettingsSpec{
		ProviderRequestTimeoutSeconds: &providerTimeout,
		StreamIdleTimeoutSeconds:      &streamTimeout,
		ShutdownDrainTimeoutSeconds:   &shutdownTimeout,
	}

	actual := runtimeConfigData(runtimeResource)

	expected := map[string]string{
		envEnvironment:            "production",
		envRuntimeHost:            "0.0.0.0",
		envRuntimePort:            "9000",
		envProviderName:           "ollama",
		envProviderBaseURL:        baseURL,
		envProviderRequestSeconds: "75",
		envProviderStreamIdle:     "40",
		envRuntimeShutdownSeconds: "50",
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf(
			"unexpected ConfigMap data\nexpected: %#v\nactual: %#v",
			expected,
			actual,
		)
	}
}

func TestRuntimeConfigDataOmitsOptionalSettings(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	runtimeResource.Spec.Provider.BaseURL = nil
	runtimeResource.Spec.Runtime = nil

	actual := runtimeConfigData(runtimeResource)

	if _, exists := actual[envProviderBaseURL]; exists {
		t.Fatal("provider base URL should be omitted")
	}

	if _, exists := actual[envProviderRequestSeconds]; exists {
		t.Fatal("provider timeout should be omitted")
	}

	if _, exists := actual[envProviderStreamIdle]; exists {
		t.Fatal("stream-idle timeout should be omitted")
	}

	if _, exists := actual[envRuntimeShutdownSeconds]; exists {
		t.Fatal("shutdown timeout should be omitted")
	}
}

func TestBuildServiceAccount(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	runtimeResource.Spec.ImagePullSecrets = []runtimev1alpha1.NamedReference{
		{Name: testImagePullSecret},
	}

	serviceAccount := buildServiceAccount(runtimeResource)

	if serviceAccount.Name != runtimeResource.Name {
		t.Fatalf(
			"expected ServiceAccount name %q, received %q",
			runtimeResource.Name,
			serviceAccount.Name,
		)
	}

	if serviceAccount.AutomountServiceAccountToken == nil ||
		*serviceAccount.AutomountServiceAccountToken {
		t.Fatal("expected ServiceAccount token automount to be disabled")
	}

	expectedSecrets := []corev1.LocalObjectReference{
		{Name: testImagePullSecret},
	}

	if !reflect.DeepEqual(
		expectedSecrets,
		serviceAccount.ImagePullSecrets,
	) {
		t.Fatalf(
			"unexpected image-pull Secrets: %#v",
			serviceAccount.ImagePullSecrets,
		)
	}
}

func TestBuildService(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	runtimeResource.Spec.Service.Type = corev1.ServiceTypeLoadBalancer
	runtimeResource.Spec.Service.Port = 9443

	service := buildService(runtimeResource)

	if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
		t.Fatalf(
			"expected LoadBalancer Service, received %q",
			service.Spec.Type,
		)
	}

	if len(service.Spec.Ports) != 1 {
		t.Fatalf(
			"expected one Service port, received %d",
			len(service.Spec.Ports),
		)
	}

	port := service.Spec.Ports[0]
	if port.Port != 9443 {
		t.Fatalf("expected Service port 9443, received %d", port.Port)
	}

	if port.TargetPort.StrVal != runtimeHTTPPortName {
		t.Fatalf(
			"expected target port %q, received %q",
			runtimeHTTPPortName,
			port.TargetPort.StrVal,
		)
	}
}

func TestBuildDeployment(t *testing.T) {
	t.Parallel()

	runtimeResource := newTestRuntime()
	runtimeResource.Spec.ImagePullSecrets = []runtimev1alpha1.NamedReference{
		{Name: testImagePullSecret},
	}
	runtimeResource.Spec.Provider.CredentialsSecretRef =
		&runtimev1alpha1.SecretKeyReference{
			Name: "ollama-credentials",
			Key:  "api-key",
		}
	runtimeResource.Spec.Resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("250m"),
			corev1.ResourceMemory: resource.MustParse("256Mi"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse("1"),
			corev1.ResourceMemory: resource.MustParse("1Gi"),
		},
	}

	deployment := buildDeployment(runtimeResource)

	if deployment.Spec.Replicas == nil ||
		*deployment.Spec.Replicas != 2 {
		t.Fatalf(
			"expected two replicas, received %#v",
			deployment.Spec.Replicas,
		)
	}

	if deployment.Spec.Template.Spec.ServiceAccountName !=
		runtimeResource.Name {
		t.Fatalf(
			"expected ServiceAccount %q, received %q",
			runtimeResource.Name,
			deployment.Spec.Template.Spec.ServiceAccountName,
		)
	}

	if deployment.Spec.Template.Spec.EnableServiceLinks == nil ||
		*deployment.Spec.Template.Spec.EnableServiceLinks {
		t.Fatal("expected Service links to be disabled")
	}

	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		t.Fatalf(
			"expected one container, received %d",
			len(deployment.Spec.Template.Spec.Containers),
		)
	}

	container := deployment.Spec.Template.Spec.Containers[0]

	expectedImage := testRuntimeImage
	if container.Image != expectedImage {
		t.Fatalf(
			"expected image %q, received %q",
			expectedImage,
			container.Image,
		)
	}

	if len(container.EnvFrom) != 1 ||
		container.EnvFrom[0].ConfigMapRef == nil ||
		container.EnvFrom[0].ConfigMapRef.Name != runtimeResource.Name {
		t.Fatalf(
			"expected ConfigMap reference %q, received %#v",
			runtimeResource.Name,
			container.EnvFrom,
		)
	}

	if len(container.Env) != 1 {
		t.Fatalf(
			"expected one credential environment variable, received %d",
			len(container.Env),
		)
	}

	credential := container.Env[0]
	if credential.Name != envProviderAPIKey {
		t.Fatalf(
			"expected credential environment variable %q, received %q",
			envProviderAPIKey,
			credential.Name,
		)
	}

	if credential.ValueFrom == nil ||
		credential.ValueFrom.SecretKeyRef == nil {
		t.Fatal("expected provider Secret key reference")
	}

	if credential.ValueFrom.SecretKeyRef.Name != "ollama-credentials" ||
		credential.ValueFrom.SecretKeyRef.Key != "api-key" {
		t.Fatalf(
			"unexpected Secret key reference: %#v",
			credential.ValueFrom.SecretKeyRef,
		)
	}

	if !reflect.DeepEqual(
		runtimeResource.Spec.Resources,
		container.Resources,
	) {
		t.Fatalf(
			"unexpected resource requirements: %#v",
			container.Resources,
		)
	}
}

func TestBuildDeploymentWithoutProviderCredentials(t *testing.T) {
	t.Parallel()

	deployment := buildDeployment(newTestRuntime())
	container := deployment.Spec.Template.Spec.Containers[0]

	if len(container.Env) != 0 {
		t.Fatalf(
			"expected no credential environment variables, received %#v",
			container.Env,
		)
	}
}

func newTestRuntime() *runtimev1alpha1.TrussiumRuntime {
	tag := testRuntimeTag

	return &runtimev1alpha1.TrussiumRuntime{
		TypeMeta: metav1.TypeMeta{
			APIVersion: runtimev1alpha1.GroupVersion.String(),
			Kind:       "TrussiumRuntime",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "production",
			Namespace: testRuntimeNamespace,
		},
		Spec: runtimev1alpha1.TrussiumRuntimeSpec{
			Image: runtimev1alpha1.RuntimeImageSpec{
				Repository: testRuntimeRepository,
				Tag:        &tag,
				PullPolicy: corev1.PullIfNotPresent,
			},
			Replicas: ptr.To(int32(2)),
			Provider: runtimev1alpha1.ProviderSpec{
				Type:  runtimev1alpha1.ProviderTypeOllama,
				Model: "llama3.2",
			},
			Service: runtimev1alpha1.RuntimeServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Port: 9000,
			},
		},
	}
}
