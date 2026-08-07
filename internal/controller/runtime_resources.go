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
	"maps"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	runtimev1alpha1 "github.com/trussiumhq/trussium-operator/api/v1alpha1"
	policyv1 "k8s.io/api/policy/v1"
)

const (
	runtimeContainerName = "trussium"
	runtimeHTTPPortName  = "http"
	runtimeHTTPPort      = int32(9000)

	runtimeUserID                  = int64(10001)
	runtimeGroupID                 = int64(10001)
	defaultShutdownDrainSeconds    = int32(30)
	terminationGraceMarginSeconds  = int64(6)
	defaultTerminationGraceSeconds = int64(36)
	deploymentRevisionHistoryLimit = int32(3)
	topologyHostnameKey            = "kubernetes.io/hostname"
	healthLivePath                 = "/health/live"
	healthReadyPath                = "/health/ready"

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

func terminationGracePeriodSeconds(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) int64 {
	if runtimeResource.Spec.Runtime == nil ||
		runtimeResource.Spec.Runtime.ShutdownDrainTimeoutSeconds == nil {
		return defaultTerminationGraceSeconds
	}

	return int64(
		*runtimeResource.Spec.Runtime.ShutdownDrainTimeoutSeconds,
	) + terminationGraceMarginSeconds
}

func mergedPodLabels(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) map[string]string {
	labels := runtimeLabels(runtimeResource)

	if runtimeResource.Spec.PodMetadata == nil {
		return labels
	}

	for key, value := range runtimeResource.Spec.PodMetadata.Labels {
		if _, reserved := labels[key]; reserved {
			continue
		}

		labels[key] = value
	}

	return labels
}

func podAnnotations(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) map[string]string {
	if runtimeResource.Spec.PodMetadata == nil ||
		len(runtimeResource.Spec.PodMetadata.Annotations) == 0 {
		return nil
	}

	annotations := make(
		map[string]string,
		len(runtimeResource.Spec.PodMetadata.Annotations),
	)

	maps.Copy(
		annotations,
		runtimeResource.Spec.PodMetadata.Annotations,
	)

	return annotations
}

func runtimeNodeSelector(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) map[string]string {
	if runtimeResource.Spec.Scheduling == nil {
		return nil
	}

	return runtimeResource.Spec.Scheduling.NodeSelector
}

func runtimeTolerations(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) []corev1.Toleration {
	if runtimeResource.Spec.Scheduling == nil {
		return nil
	}

	return runtimeResource.Spec.Scheduling.Tolerations
}

func runtimeAffinity(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) *corev1.Affinity {
	if runtimeResource.Spec.Scheduling == nil ||
		runtimeResource.Spec.Scheduling.Affinity == nil {
		return nil
	}

	return runtimeResource.Spec.Scheduling.Affinity.DeepCopy()
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
	podLabels := mergedPodLabels(runtimeResource)

	maxUnavailable := intstr.FromInt32(0)
	maxSurge := intstr.FromInt32(1)

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
			RevisionHistoryLimit: ptr.To(
				deploymentRevisionHistoryLimit,
			),
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxUnavailable: &maxUnavailable,
					MaxSurge:       &maxSurge,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: podAnnotations(runtimeResource),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: runtimeResource.Name,

					AutomountServiceAccountToken: ptr.To(false),
					EnableServiceLinks:           ptr.To(false),

					TerminationGracePeriodSeconds: ptr.To(
						terminationGracePeriodSeconds(
							runtimeResource,
						),
					),

					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptr.To(true),
						RunAsUser:    ptr.To(runtimeUserID),
						RunAsGroup:   ptr.To(runtimeGroupID),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},

					ImagePullSecrets: imagePullSecrets(
						runtimeResource,
					),

					NodeSelector: runtimeNodeSelector(
						runtimeResource,
					),
					Tolerations: runtimeTolerations(
						runtimeResource,
					),
					Affinity: runtimeAffinity(
						runtimeResource,
					),

					TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
						{
							MaxSkew:           1,
							TopologyKey:       topologyHostnameKey,
							WhenUnsatisfiable: corev1.ScheduleAnyway,
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: selectorLabels,
							},
						},
					},

					Containers: []corev1.Container{
						{
							Name:            runtimeContainerName,
							Image:           runtimeImage(runtimeResource.Spec.Image),
							ImagePullPolicy: runtimeResource.Spec.Image.PullPolicy,

							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: ptr.To(false),
								ReadOnlyRootFilesystem:   ptr.To(true),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},

							Ports: []corev1.ContainerPort{
								{
									Name:          runtimeHTTPPortName,
									ContainerPort: runtimeHTTPPort,
									Protocol:      corev1.ProtocolTCP,
								},
							},

							StartupProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: healthLivePath,
										Port: intstr.FromString(
											runtimeHTTPPortName,
										),
									},
								},
								PeriodSeconds:    2,
								TimeoutSeconds:   1,
								FailureThreshold: 30,
							},

							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: healthLivePath,
										Port: intstr.FromString(
											runtimeHTTPPortName,
										),
									},
								},
								PeriodSeconds:    10,
								TimeoutSeconds:   2,
								FailureThreshold: 3,
							},

							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: healthReadyPath,
										Port: intstr.FromString(
											runtimeHTTPPortName,
										),
									},
								},
								PeriodSeconds:    5,
								TimeoutSeconds:   2,
								FailureThreshold: 3,
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

							Env: providerCredentialEnv(
								runtimeResource,
							),

							Resources: *runtimeResource.Spec.Resources.DeepCopy(),
						},
					},
				},
			},
		},
	}
}

func buildPodDisruptionBudget(
	runtimeResource *runtimev1alpha1.TrussiumRuntime,
) *policyv1.PodDisruptionBudget {
	maxUnavailable := intstr.FromInt32(1)

	return &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runtimeResource.Name,
			Namespace: runtimeResource.Namespace,
			Labels:    runtimeLabels(runtimeResource),
		},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MaxUnavailable: &maxUnavailable,
			Selector: &metav1.LabelSelector{
				MatchLabels: runtimeSelectorLabels(runtimeResource),
			},
		},
	}
}
