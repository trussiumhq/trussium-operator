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
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

func TestAddToSchemeRegistersTrussiumRuntimeTypes(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	if err := AddToScheme(scheme); err != nil {
		t.Fatalf("register TrussiumRuntime API types: %v", err)
	}

	runtimeGVK := GroupVersion.WithKind("TrussiumRuntime")
	if !scheme.Recognizes(runtimeGVK) {
		t.Fatalf("scheme does not recognize %s", runtimeGVK)
	}

	runtimeObject, err := scheme.New(runtimeGVK)
	if err != nil {
		t.Fatalf("create TrussiumRuntime from scheme: %v", err)
	}

	if _, ok := runtimeObject.(*TrussiumRuntime); !ok {
		t.Fatalf(
			"expected *TrussiumRuntime, received %T",
			runtimeObject,
		)
	}

	listGVK := GroupVersion.WithKind("TrussiumRuntimeList")
	if !scheme.Recognizes(listGVK) {
		t.Fatalf("scheme does not recognize %s", listGVK)
	}

	listObject, err := scheme.New(listGVK)
	if err != nil {
		t.Fatalf("create TrussiumRuntimeList from scheme: %v", err)
	}

	if _, ok := listObject.(*TrussiumRuntimeList); !ok {
		t.Fatalf(
			"expected *TrussiumRuntimeList, received %T",
			listObject,
		)
	}
}

func TestTrussiumRuntimeJSONRoundTrip(t *testing.T) {
	t.Parallel()

	tag := "v0.10.0"
	replicas := int32(2)
	baseURL := "http://ollama.ollama.svc.cluster.local:11434/v1"
	providerTimeout := int32(60)
	streamIdleTimeout := int32(30)
	shutdownTimeout := int32(45)

	original := TrussiumRuntime{
		TypeMeta: metav1.TypeMeta{
			APIVersion: GroupVersion.String(),
			Kind:       "TrussiumRuntime",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "production",
			Namespace: "trussium",
		},
		Spec: TrussiumRuntimeSpec{
			Image: RuntimeImageSpec{
				Repository: "ghcr.io/trussium/trussium",
				Tag:        &tag,
				PullPolicy: corev1.PullIfNotPresent,
			},
			Replicas: &replicas,
			ImagePullSecrets: []NamedReference{
				{Name: "ghcr-credentials"},
			},
			Provider: ProviderSpec{
				Type:    ProviderTypeOllama,
				Model:   "llama3.2",
				BaseURL: &baseURL,
			},
			Runtime: &RuntimeSettingsSpec{
				ProviderRequestTimeoutSeconds: &providerTimeout,
				StreamIdleTimeoutSeconds:      &streamIdleTimeout,
				ShutdownDrainTimeoutSeconds:   &shutdownTimeout,
			},
			Service: RuntimeServiceSpec{
				Type: corev1.ServiceTypeClusterIP,
				Port: 9000,
			},
			Resources: corev1.ResourceRequirements{},
		},
		Status: TrussiumRuntimeStatus{
			ObservedGeneration: 4,
			ReadyReplicas:      2,
			AvailableReplicas:  2,
			CurrentImage:       "ghcr.io/trussium/trussium:v0.10.0",
			Endpoint:           "http://production.trussium.svc.cluster.local:9000",
		},
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal TrussiumRuntime: %v", err)
	}

	var decoded TrussiumRuntime
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal TrussiumRuntime: %v", err)
	}

	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf(
			"round-trip mismatch\noriginal: %#v\ndecoded: %#v",
			original,
			decoded,
		)
	}
}

func TestGeneratedCRDContract(t *testing.T) {
	t.Parallel()

	crd := loadGeneratedCRD(t)

	if crd.Spec.Group != "runtime.trussium.io" {
		t.Fatalf(
			"expected CRD group runtime.trussium.io, received %q",
			crd.Spec.Group,
		)
	}

	if crd.Spec.Scope != apiextensionsv1.NamespaceScoped {
		t.Fatalf(
			"expected namespaced CRD, received %q",
			crd.Spec.Scope,
		)
	}

	if crd.Spec.Names.Kind != "TrussiumRuntime" {
		t.Fatalf(
			"expected kind TrussiumRuntime, received %q",
			crd.Spec.Names.Kind,
		)
	}

	if crd.Spec.Names.Plural != "trussiumruntimes" {
		t.Fatalf(
			"expected plural trussiumruntimes, received %q",
			crd.Spec.Names.Plural,
		)
	}

	if !containsString(crd.Spec.Names.ShortNames, "truntime") {
		t.Fatalf(
			"expected short name truntime, received %#v",
			crd.Spec.Names.ShortNames,
		)
	}

	version := requireCRDVersion(t, crd, "v1alpha1")

	if !version.Served {
		t.Fatal("expected v1alpha1 to be served")
	}

	if !version.Storage {
		t.Fatal("expected v1alpha1 to be the storage version")
	}

	if version.Subresources == nil || version.Subresources.Status == nil {
		t.Fatal("expected status subresource to be enabled")
	}

	rootSchema := version.Schema.OpenAPIV3Schema
	if rootSchema == nil {
		t.Fatal("expected generated OpenAPI schema")
	}

	specSchema := requireSchemaProperty(t, *rootSchema, "spec")
	requireRequiredFields(t, specSchema, "image", "provider")

	replicasSchema := requireSchemaProperty(t, specSchema, "replicas")
	requireJSONDefault(t, replicasSchema, "1")

	if replicasSchema.Minimum == nil || *replicasSchema.Minimum != 0 {
		t.Fatalf(
			"expected replicas minimum 0, received %#v",
			replicasSchema.Minimum,
		)
	}

	imageSchema := requireSchemaProperty(t, specSchema, "image")
	requireRequiredFields(t, imageSchema, "repository")
	requireXValidationRule(
		t,
		imageSchema,
		"has(self.tag) != has(self.digest)",
	)

	pullPolicySchema := requireSchemaProperty(
		t,
		imageSchema,
		"pullPolicy",
	)
	requireJSONDefault(t, pullPolicySchema, `"IfNotPresent"`)
	requireEnumStrings(
		t,
		pullPolicySchema,
		"Always",
		"IfNotPresent",
		"Never",
	)

	providerSchema := requireSchemaProperty(t, specSchema, "provider")
	requireRequiredFields(t, providerSchema, "type", "model")

	providerTypeSchema := requireSchemaProperty(
		t,
		providerSchema,
		"type",
	)
	requireEnumStrings(
		t,
		providerTypeSchema,
		"openai",
		"ollama",
	)

	serviceSchema := requireSchemaProperty(t, specSchema, "service")
	requireJSONDefault(t, serviceSchema, "{}")

	serviceTypeSchema := requireSchemaProperty(
		t,
		serviceSchema,
		"type",
	)
	requireJSONDefault(t, serviceTypeSchema, `"ClusterIP"`)
	requireEnumStrings(
		t,
		serviceTypeSchema,
		"ClusterIP",
		"NodePort",
		"LoadBalancer",
	)

	servicePortSchema := requireSchemaProperty(
		t,
		serviceSchema,
		"port",
	)
	requireJSONDefault(t, servicePortSchema, "9000")

	statusSchema := requireSchemaProperty(t, *rootSchema, "status")
	conditionsSchema := requireSchemaProperty(
		t,
		statusSchema,
		"conditions",
	)

	if conditionsSchema.XListType == nil ||
		*conditionsSchema.XListType != "map" {
		t.Fatalf(
			"expected conditions list type map, received %#v",
			conditionsSchema.XListType,
		)
	}

	if !containsString(conditionsSchema.XListMapKeys, "type") {
		t.Fatalf(
			"expected conditions list map key type, received %#v",
			conditionsSchema.XListMapKeys,
		)
	}

	requiredColumns := []string{
		"Ready",
		"Desired",
		"Provider",
		"Model",
		"Version",
		"Age",
	}

	columnNames := make(
		[]string,
		0,
		len(version.AdditionalPrinterColumns),
	)
	for _, column := range version.AdditionalPrinterColumns {
		columnNames = append(columnNames, column.Name)
	}

	for _, requiredColumn := range requiredColumns {
		if !containsString(columnNames, requiredColumn) {
			t.Fatalf(
				"missing printer column %q in %#v",
				requiredColumn,
				columnNames,
			)
		}
	}
}

func loadGeneratedCRD(
	t *testing.T,
) apiextensionsv1.CustomResourceDefinition {
	t.Helper()

	path := filepath.Join(
		"..",
		"..",
		"config",
		"crd",
		"bases",
		"runtime.trussium.io_trussiumruntimes.yaml",
	)

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated CRD %s: %v", path, err)
	}

	var crd apiextensionsv1.CustomResourceDefinition
	if err := yaml.Unmarshal(content, &crd); err != nil {
		t.Fatalf("decode generated CRD %s: %v", path, err)
	}

	return crd
}

func requireCRDVersion(
	t *testing.T,
	crd apiextensionsv1.CustomResourceDefinition,
	versionName string,
) apiextensionsv1.CustomResourceDefinitionVersion {
	t.Helper()

	for _, version := range crd.Spec.Versions {
		if version.Name == versionName {
			return version
		}
	}

	t.Fatalf(
		"CRD version %q not found in %#v",
		versionName,
		crd.Spec.Versions,
	)

	return apiextensionsv1.CustomResourceDefinitionVersion{}
}

func requireSchemaProperty(
	t *testing.T,
	schema apiextensionsv1.JSONSchemaProps,
	propertyName string,
) apiextensionsv1.JSONSchemaProps {
	t.Helper()

	property, ok := schema.Properties[propertyName]
	if !ok {
		t.Fatalf(
			"schema property %q not found in %#v",
			propertyName,
			schema.Properties,
		)
	}

	return property
}

func requireRequiredFields(
	t *testing.T,
	schema apiextensionsv1.JSONSchemaProps,
	requiredFields ...string,
) {
	t.Helper()

	for _, requiredField := range requiredFields {
		if !containsString(schema.Required, requiredField) {
			t.Fatalf(
				"required field %q not found in %#v",
				requiredField,
				schema.Required,
			)
		}
	}
}

func requireJSONDefault(
	t *testing.T,
	schema apiextensionsv1.JSONSchemaProps,
	expectedJSON string,
) {
	t.Helper()

	if schema.Default == nil {
		t.Fatalf("expected default %s, received no default", expectedJSON)
	}

	var expected any
	if err := json.Unmarshal([]byte(expectedJSON), &expected); err != nil {
		t.Fatalf("decode expected default %s: %v", expectedJSON, err)
	}

	var actual any
	if err := json.Unmarshal(schema.Default.Raw, &actual); err != nil {
		t.Fatalf(
			"decode generated default %s: %v",
			string(schema.Default.Raw),
			err,
		)
	}

	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf(
			"expected default %#v, received %#v",
			expected,
			actual,
		)
	}
}

func requireEnumStrings(
	t *testing.T,
	schema apiextensionsv1.JSONSchemaProps,
	expectedValues ...string,
) {
	t.Helper()

	actualValues := make([]string, 0, len(schema.Enum))

	for _, enumValue := range schema.Enum {
		var decoded string
		if err := json.Unmarshal(enumValue.Raw, &decoded); err != nil {
			t.Fatalf(
				"decode enum value %s: %v",
				string(enumValue.Raw),
				err,
			)
		}

		actualValues = append(actualValues, decoded)
	}

	for _, expectedValue := range expectedValues {
		if !containsString(actualValues, expectedValue) {
			t.Fatalf(
				"expected enum value %q in %#v",
				expectedValue,
				actualValues,
			)
		}
	}
}

func requireXValidationRule(
	t *testing.T,
	schema apiextensionsv1.JSONSchemaProps,
	expectedRule string,
) {
	t.Helper()

	for _, validationRule := range schema.XValidations {
		if validationRule.Rule == expectedRule {
			return
		}
	}

	t.Fatalf(
		"expected CEL validation rule %q in %#v",
		expectedRule,
		schema.XValidations,
	)
}

func containsString(values []string, expected string) bool {
	return slices.Contains(values, expected)
}
