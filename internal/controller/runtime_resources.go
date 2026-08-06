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
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
)

const (
	runtimeContainerName = "trussium"
	runtimeHTTPPortName  = "http"
	runtimeHTTPPort      = int32(9000)

	envEnvironment             = "TRUSSIUM_ENVIRONMENT"
	envRuntimeHost             = "TRUSSIUM_RUNTIME__HOST"
	envRuntimePort             = "TRUSSIUM_RUNTIME__PORT"
	envRuntimeShutdownSeconds  = "TRUSSIUM_RUNTIME__GRACEFUL_SHUTDOWN_SECONDS"
	envProviderName            = "TRUSSIUM_PROVIDER__NAME"
	envProviderBaseURL         = "TRUSSIUM_PROVIDER__BASE_URL"
	envProviderAPIKey          = "TRUSSIUM_PROVIDER__API_KEY"
	envProviderRequestSeconds  = "TRUSSIUM_TIMEOUTS__PROVIDER_REQUEST_SECONDS"
	envProviderStreamIdle      = "TRUSSIUM_TIMEOUTS__STREAM_IDLE_SECONDS"
	defaultRuntimeEnvironment  = "production"
	defaultRuntimeHost         = "0.0.0.0"
	defaultRuntimeReplicaCount = int32(1)
	defaultServicePort         = int32(9000)
)

// runtimeLabels returns stable labels shared by resources managed for one
// TrussiumRuntime.
func runtimeLabels(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       runtimeContainerName,
		"app.kubernetes.io/instance":   runtimeResource.Name,
		"app.kubernetes.io/component":  "runtime",
		"app.kubernetes.io/managed-by": "trussium-operator",
	}
}

// runtimeSelectorLabels returns the immutable label set used by Services and
// Deployments to select a runtime instance.
func runtimeSelectorLabels(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":     runtimeContainerName,
		"app.kubernetes.io/instance": runtimeResource.Name,
	}
}

// runtimeImage returns the complete container image selected by the custom
// resource.
func runtimeImage(
	imageSpec runtimev1alpha1.RuntimeImageSpec,
) string {
	if imageSpec.Digest != nil {
		return imageSpec.Repository + "@" + *imageSpec.Digest
	}

	if imageSpec.Tag != nil {
		return imageSpec.Repository + ":" + *imageSpec.Tag
	}

	return imageSpec.Repository
}

// desiredReplicas returns the schema default when a non-defaulted object is
// supplied directly to a unit test or fake client.
func desiredReplicas(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) int32 {
	if runtimeResource.Spec.Replicas == nil {
		return defaultRuntimeReplicaCount
	}

	return *runtimeResource.Spec.Replicas
}

// desiredServiceType returns the schema default when a non-defaulted object is
// supplied directly to a unit test or fake client.
func desiredServiceType(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) corev1.ServiceType {
	if runtimeResource.Spec.Service.Type == "" {
		return corev1.ServiceTypeClusterIP
	}

	return runtimeResource.Spec.Service.Type
}

// desiredServicePort returns the schema default when a non-defaulted object is
// supplied directly to a unit test or fake client.
func desiredServicePort(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) int32 {
	if runtimeResource.Spec.Service.Port == 0 {
		return defaultServicePort
	}

	return runtimeResource.Spec.Service.Port
}

// imagePullSecrets converts API references to Kubernetes local references.
func imagePullSecrets(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) []corev1.LocalObjectReference {
	if len(runtimeResource.Spec.ImagePullSecrets) == 0 {
		return nil
	}

	references := make(
		[]corev1.LocalObjectReference,
		0,
		len(runtimeResource.Spec.ImagePullSecrets),
	)

	for _, reference := range runtimeResource.Spec.ImagePullSecrets {
		references = append(
			references,
			corev1.LocalObjectReference{Name: reference.Name},
		)
	}

	return references
}

// runtimeConfigData builds the environment configuration projected through the
// managed ConfigMap.
func runtimeConfigData(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) map[string]string {
	data := map[string]string{
		envEnvironment:  defaultRuntimeEnvironment,
		envRuntimeHost:  defaultRuntimeHost,
		envRuntimePort:  strconv.Itoa(int(runtimeHTTPPort)),
		envProviderName: string(runtimeResource.Spec.Provider.Type),
	}

	if runtimeResource.Spec.Provider.BaseURL != nil {
		data[envProviderBaseURL] = *runtimeResource.Spec.Provider.BaseURL
	}

	if runtimeResource.Spec.Runtime == nil {
		return data
	}

	if timeout := runtimeResource.Spec.Runtime.ProviderRequestTimeoutSeconds; timeout != nil {
		data[envProviderRequestSeconds] = strconv.Itoa(int(*timeout))
	}

	if timeout := runtimeResource.Spec.Runtime.StreamIdleTimeoutSeconds; timeout != nil {
		data[envProviderStreamIdle] = strconv.Itoa(int(*timeout))
	}

	if timeout := runtimeResource.Spec.Runtime.ShutdownDrainTimeoutSeconds; timeout != nil {
		data[envRuntimeShutdownSeconds] = strconv.Itoa(int(*timeout))
	}

	return data
}

// buildConfigMap constructs the desired runtime ConfigMap.
func buildConfigMap(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runtimeResource.Name,
			Namespace: runtimeResource.Namespace,
			Labels:    runtimeLabels(runtimeResource),
		},
		Data: runtimeConfigData(runtimeResource),
	}
}

// buildServiceAccount constructs the dedicated runtime ServiceAccount.
func buildServiceAccount(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runtimeResource.Name,
			Namespace: runtimeResource.Namespace,
			Labels:    runtimeLabels(runtimeResource),
		},
		AutomountServiceAccountToken: ptr.To(false),
		ImagePullSecrets:             imagePullSecrets(runtimeResource),
	}
}

// buildService constructs the runtime Service.
func buildService(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runtimeResource.Name,
			Namespace: runtimeResource.Namespace,
			Labels:    runtimeLabels(runtimeResource),
		},
		Spec: corev1.ServiceSpec{
			Type:     desiredServiceType(runtimeResource),
			Selector: runtimeSelectorLabels(runtimeResource),
			Ports: []corev1.ServicePort{
				{
					Name:       runtimeHTTPPortName,
					Protocol:   corev1.ProtocolTCP,
					Port:       desiredServicePort(runtimeResource),
					TargetPort: intstr.FromString(runtimeHTTPPortName),
				},
			},
		},
	}
}

// providerCredentialEnv returns the provider credential environment variable
// when the custom resource references a Secret key.
func providerCredentialEnv(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) []corev1.EnvVar {
	secretReference := runtimeResource.Spec.Provider.CredentialsSecretRef
	if secretReference == nil {
		return nil
	}

	return []corev1.EnvVar{
		{
			Name: envProviderAPIKey,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secretReference.Name,
					},
					Key: secretReference.Key,
				},
			},
		},
	}
}

// buildDeployment constructs the desired runtime Deployment.
func buildDeployment(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) *appsv1.Deployment {
	labels := runtimeLabels(runtimeResource)
	selectorLabels := runtimeSelectorLabels(runtimeResource)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runtimeResource.Name,
			Namespace: runtimeResource.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: ptr.To(desiredReplicas(runtimeResource)),
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:           runtimeResource.Name,
					AutomountServiceAccountToken: ptr.To(false),
					EnableServiceLinks:           ptr.To(false),
					ImagePullSecrets:             imagePullSecrets(runtimeResource),
					Containers: []corev1.Container{
						{
							Name:            runtimeContainerName,
							Image:           runtimeImage(runtimeResource.Spec.Image),
							ImagePullPolicy: runtimeResource.Spec.Image.PullPolicy,
							Ports: []corev1.ContainerPort{
								{
									Name:          runtimeHTTPPortName,
									ContainerPort: runtimeHTTPPort,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: runtimeResource.Name,
										},
									},
								},
							},
							Env:       providerCredentialEnv(runtimeResource),
							Resources: *runtimeResource.Spec.Resources.DeepCopy(),
						},
					},
				},
			},
		},
	}
}
