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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProviderType identifies a provider implementation supported by Trussium.
type ProviderType string

const (
	// ProviderTypeOpenAI selects the OpenAI provider adapter.
	ProviderTypeOpenAI ProviderType = "openai"

	// ProviderTypeOllama selects the Ollama provider adapter.
	ProviderTypeOllama ProviderType = "ollama"
)

// NamedReference identifies a namespaced Kubernetes object by name.
type NamedReference struct {
	// Name is the name of the referenced object in the TrussiumRuntime namespace.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9.]*[a-z0-9])?$`
	Name string `json:"name"`
}

// SecretKeyReference identifies a key in a namespaced Kubernetes Secret.
type SecretKeyReference struct {
	// Name is the Secret name in the TrussiumRuntime namespace.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9.]*[a-z0-9])?$`
	Name string `json:"name"`

	// Key is the key containing the credential value.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9._-]+$`
	Key string `json:"key"`
}

// +kubebuilder:validation:XValidation:rule="has(self.tag) != has(self.digest)",message="exactly one of tag or digest must be specified"

// RuntimeImageSpec defines the Trussium runtime container image.
type RuntimeImageSpec struct {
	// Repository is the container image repository without a tag or digest.
	//
	// Example: ghcr.io/trussium/trussium
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9][A-Za-z0-9._:/-]*$`
	Repository string `json:"repository"`

	// Tag is an immutable or versioned image tag.
	//
	// Exactly one of tag or digest must be supplied.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=128
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9_][A-Za-z0-9._-]{0,127}$`
	Tag *string `json:"tag,omitempty"`

	// Digest is a SHA-256 OCI image digest.
	//
	// Exactly one of tag or digest must be supplied.
	// +optional
	// +kubebuilder:validation:Pattern=`^sha256:[a-f0-9]{64}$`
	Digest *string `json:"digest,omitempty"`

	// PullPolicy controls when Kubernetes pulls the runtime image.
	// +optional
	// +kubebuilder:default=IfNotPresent
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`
}

// ProviderSpec defines the AI provider used by the runtime.
type ProviderSpec struct {
	// Type selects the provider adapter.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=openai;ollama
	Type ProviderType `json:"type"`

	// Model identifies the model requested from the provider.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	Model string `json:"model"`

	// BaseURL overrides the provider API endpoint.
	//
	// It is commonly used for Ollama and OpenAI-compatible private endpoints.
	// +optional
	// +kubebuilder:validation:Format=uri
	// +kubebuilder:validation:MaxLength=2048
	BaseURL *string `json:"baseURL,omitempty"`

	// CredentialsSecretRef references the Secret key containing provider
	// credentials.
	//
	// Credential values must never be embedded in the custom resource.
	// +optional
	CredentialsSecretRef *SecretKeyReference `json:"credentialsSecretRef,omitempty"`
}

// RuntimeSettingsSpec defines optional Trussium runtime overrides.
//
// Omitted fields use the defaults owned by the selected Trussium runtime
// release.
type RuntimeSettingsSpec struct {
	// ProviderRequestTimeoutSeconds limits a complete provider request.
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3600
	ProviderRequestTimeoutSeconds *int32 `json:"providerRequestTimeoutSeconds,omitempty"`

	// StreamIdleTimeoutSeconds limits the wait between provider stream events.
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3600
	StreamIdleTimeoutSeconds *int32 `json:"streamIdleTimeoutSeconds,omitempty"`

	// ShutdownDrainTimeoutSeconds limits active-workload draining during
	// graceful shutdown.
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3600
	ShutdownDrainTimeoutSeconds *int32 `json:"shutdownDrainTimeoutSeconds,omitempty"`
}

// RuntimeServiceSpec defines the Kubernetes Service exposed for a runtime.
type RuntimeServiceSpec struct {
	// Type is the Kubernetes Service type.
	// +optional
	// +kubebuilder:default=ClusterIP
	// +kubebuilder:validation:Enum=ClusterIP;NodePort;LoadBalancer
	Type corev1.ServiceType `json:"type,omitempty"`

	// Port is the Service port used to expose the Trussium HTTP API.
	// +optional
	// +kubebuilder:default=9000
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port,omitempty"`
}

// TrussiumRuntimeSpec defines the desired state of a Trussium runtime.
type TrussiumRuntimeSpec struct {
	// Image selects the released Trussium runtime container image.
	// +kubebuilder:validation:Required
	Image RuntimeImageSpec `json:"image"`

	// Replicas is the desired number of runtime pods.
	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=1000
	Replicas *int32 `json:"replicas,omitempty"`

	// ImagePullSecrets references Secrets used to pull runtime images.
	//
	// References are resolved in the TrussiumRuntime namespace.
	// +optional
	// +kubebuilder:validation:MaxItems=20
	// +listType=map
	// +listMapKey=name
	ImagePullSecrets []NamedReference `json:"imagePullSecrets,omitempty"`

	// Provider selects the AI provider and model.
	// +kubebuilder:validation:Required
	Provider ProviderSpec `json:"provider"`

	// Runtime contains optional runtime configuration overrides.
	// +optional
	Runtime *RuntimeSettingsSpec `json:"runtime,omitempty"`

	// Service defines the Kubernetes Service exposed for the runtime.
	// +optional
	// +kubebuilder:default={}
	Service RuntimeServiceSpec `json:"service,omitempty"`

	// Resources defines runtime-container compute resource requirements.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// TrussiumRuntimeStatus defines the observed state of a Trussium runtime.
type TrussiumRuntimeStatus struct {
	// ObservedGeneration is the most recent resource generation processed by
	// the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// ReadyReplicas is the number of runtime replicas currently ready.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// AvailableReplicas is the number of runtime replicas currently available.
	// +optional
	AvailableReplicas int32 `json:"availableReplicas,omitempty"`

	// CurrentImage is the complete runtime image observed by the controller.
	// +optional
	CurrentImage string `json:"currentImage,omitempty"`

	// Endpoint is the Kubernetes Service endpoint for the runtime.
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Conditions describe the current runtime state.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=truntime
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyReplicas"
// +kubebuilder:printcolumn:name="Desired",type="integer",JSONPath=".spec.replicas"
// +kubebuilder:printcolumn:name="Provider",type="string",JSONPath=".spec.provider.type"
// +kubebuilder:printcolumn:name="Model",type="string",JSONPath=".spec.provider.model"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.image.tag"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// TrussiumRuntime declares a Trussium runtime instance.
type TrussiumRuntime struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired runtime configuration.
	// +kubebuilder:validation:Required
	Spec TrussiumRuntimeSpec `json:"spec"`

	// Status reports the observed runtime state.
	// +optional
	Status TrussiumRuntimeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TrussiumRuntimeList contains a list of TrussiumRuntime resources.
type TrussiumRuntimeList struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ListMeta `json:"metadata,omitempty"`

	Items []TrussiumRuntime `json:"items"`
}
