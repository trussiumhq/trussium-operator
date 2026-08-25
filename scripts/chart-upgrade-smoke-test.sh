#!/usr/bin/env bash
set -euo pipefail

namespace="trussium-operator-upgrade-test"
release="trussium-operator"
previous_version="${PREVIOUS_CHART_VERSION:-0.3.1}"
previous_chart="/tmp/trussium-operator-${previous_version}.tgz"

cleanup() {
  helm uninstall "$release" --namespace "$namespace" --ignore-not-found
  kubectl delete namespace "$namespace" --ignore-not-found --wait=false
}
trap cleanup EXIT

helm pull "oci://ghcr.io/trussiumhq/charts/trussium-operator" --version "$previous_version" --destination /tmp
helm install "$release" "$previous_chart" --namespace "$namespace" --create-namespace --wait --timeout=3m

kubectl apply -f - <<'YAML'
apiVersion: runtime.trussium.io/v1alpha1
kind: TrussiumRuntime
metadata:
  name: upgrade-smoke
  namespace: trussium-operator-upgrade-test
spec:
  image:
    repository: ghcr.io/trussiumhq/trussium
    tag: v0.24.0
  provider:
    type: ollama
    model: llama3.2
YAML

kubectl get trussiumruntime upgrade-smoke --namespace "$namespace"
helm upgrade "$release" charts/trussium-operator --namespace "$namespace" --wait --timeout=3m
kubectl get trussiumruntime upgrade-smoke --namespace "$namespace"
kubectl rollout status deployment/"$release-trussium-operator" --namespace "$namespace" --timeout=2m
