#!/usr/bin/env bash
set -euo pipefail

namespace="${UPGRADE_TEST_NAMESPACE:-trussium-operator-upgrade-test}"
release="${UPGRADE_TEST_RELEASE:-trussium-operator}"
previous_version="${PREVIOUS_CHART_VERSION:-0.3.1}"
runtime_tag="${TRUSSIUM_RUNTIME_TAG:-0.98.1}"
previous_chart="/tmp/trussium-operator-${previous_version}.tgz"

cleanup() {
  helm uninstall "$release" --namespace "$namespace" --ignore-not-found
  kubectl delete namespace "$namespace" --ignore-not-found --wait=false
}
trap cleanup EXIT

helm pull "oci://ghcr.io/trussiumhq/charts/trussium-operator" --version "$previous_version" --destination /tmp
helm install "$release" "$previous_chart" --namespace "$namespace" --create-namespace --wait --timeout=3m

kubectl apply -f - <<YAML
apiVersion: runtime.trussium.io/v1alpha1
kind: TrussiumRuntime
metadata:
  name: upgrade-smoke
  namespace: ${namespace}
spec:
  image:
    repository: ghcr.io/trussiumhq/trussium
    tag: ${runtime_tag}
  provider:
    type: ollama
    model: llama3.2
YAML

kubectl get trussiumruntime upgrade-smoke --namespace "$namespace"
helm upgrade "$release" charts/trussium-operator --namespace "$namespace" --wait --timeout=3m
kubectl get trussiumruntime upgrade-smoke --namespace "$namespace"
kubectl rollout status deployment/"$release-trussium-operator" --namespace "$namespace" --timeout=2m

helm rollback "$release" 1 --namespace "$namespace" --wait --timeout=3m
kubectl get trussiumruntime upgrade-smoke --namespace "$namespace"
kubectl rollout status deployment/"$release-trussium-operator" --namespace "$namespace" --timeout=2m
