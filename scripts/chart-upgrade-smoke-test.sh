#!/usr/bin/env bash
set -euo pipefail

namespace="${UPGRADE_TEST_NAMESPACE:-trussium-operator-upgrade-test}"
release="${UPGRADE_TEST_RELEASE:-trussium-operator}"
previous_version="${PREVIOUS_CHART_VERSION:-0.3.1}"
runtime_tag="${TRUSSIUM_RUNTIME_TAG:-1.0.0}"
runtime_rollback_tag="${TRUSSIUM_RUNTIME_ROLLBACK_TAG:-}"
current_operator_chart_version="${CURRENT_OPERATOR_CHART_VERSION:-}"
verify_runtime_image_status="${VERIFY_RUNTIME_IMAGE_STATUS:-false}"
runtime_repository="ghcr.io/trussiumhq/trussium"
runtime_name="upgrade-smoke"
previous_chart="/tmp/trussium-operator-${previous_version}.tgz"
current_operator_chart="/tmp/trussium-operator-${current_operator_chart_version}.tgz"

wait_for_runtime_image() {
  local tag="$1"
  local expected_image="${runtime_repository}:${tag}"
  local configured_image

  if [[ "$verify_runtime_image_status" == "true" ]]; then
    echo "Waiting for ${runtime_name} to report successful image ${expected_image}"
    kubectl wait \
      --namespace "$namespace" \
      --for="jsonpath={.status.lastSuccessfulImage}=${expected_image}" \
      --timeout=5m \
      "trussiumruntime/${runtime_name}"
    kubectl wait \
      --namespace "$namespace" \
      --for='condition=Ready' \
      --timeout=5m \
      "trussiumruntime/${runtime_name}"
    kubectl rollout status \
      "deployment/${runtime_name}" \
      --namespace "$namespace" \
      --timeout=5m

    configured_image="$(kubectl get deployment "$runtime_name" \
      --namespace "$namespace" \
      -o jsonpath='{.spec.template.spec.containers[0].image}')"
    if [[ "$configured_image" != "$expected_image" ]]; then
      echo "Expected runtime Deployment image ${expected_image}, got ${configured_image}" >&2
      return 1
    fi

    kubectl get "trussiumruntime/${runtime_name}" \
      --namespace "$namespace" \
      --output yaml
  else
    # Older matrix entries predate rollout status fields; retain their
    # resource-presence check while validating current status in the latest row.
    kubectl get "trussiumruntime/${runtime_name}" --namespace "$namespace"
  fi
}

set_runtime_image() {
  local tag="$1"

  kubectl patch "trussiumruntime/${runtime_name}" \
    --namespace "$namespace" \
    --type=merge \
    --patch "{\"spec\":{\"image\":{\"tag\":\"${tag}\"}}}"
  wait_for_runtime_image "$tag"
}

cleanup() {
  local exit_code=$?
  trap - EXIT

  if [[ "$exit_code" -eq 0 ]]; then
    helm uninstall "$release" --namespace "$namespace" --ignore-not-found
    kubectl delete namespace "$namespace" --ignore-not-found --wait=false
  else
    echo "Preserving namespace ${namespace} for CI failure diagnostics" >&2
  fi

  exit "$exit_code"
}
trap cleanup EXIT

helm pull "oci://ghcr.io/trussiumhq/charts/trussium-operator" --version "$previous_version" --destination /tmp
helm install "$release" "$previous_chart" --namespace "$namespace" --create-namespace --wait --timeout=3m

initial_runtime_tag="$runtime_tag"
if [[ -n "$runtime_rollback_tag" ]]; then
  initial_runtime_tag="$runtime_rollback_tag"
fi

kubectl apply -f - <<YAML
apiVersion: runtime.trussium.io/v1alpha1
kind: TrussiumRuntime
metadata:
  name: ${runtime_name}
  namespace: ${namespace}
spec:
  image:
    repository: ${runtime_repository}
    tag: ${initial_runtime_tag}
  provider:
    type: ollama
    model: llama3.2
YAML

wait_for_runtime_image "$initial_runtime_tag"
kubectl get trussiumruntime "$runtime_name" --namespace "$namespace"

if [[ -n "$runtime_rollback_tag" ]]; then
  set_runtime_image "$runtime_tag"
fi

# The upgrade matrix runs before the candidate image is published. Use the
# current published operator chart when explicitly requested; otherwise use
# the checked-out chart and its workflow-provided image override.
if [[ -n "$current_operator_chart_version" ]]; then
  helm pull "oci://ghcr.io/trussiumhq/charts/trussium-operator" \
    --version "$current_operator_chart_version" \
    --destination /tmp
  helm upgrade "$release" "$current_operator_chart" \
    --namespace "$namespace" --wait --timeout=3m
else
  helm upgrade "$release" charts/trussium-operator \
    --namespace "$namespace" --set image.tag=0.14.0 --wait --timeout=3m
fi

kubectl get trussiumruntime "$runtime_name" --namespace "$namespace"
wait_for_runtime_image "$runtime_tag"
kubectl rollout status deployment/"$release-trussium-operator" --namespace "$namespace" --timeout=2m

helm rollback "$release" 1 --namespace "$namespace" --wait --timeout=3m
kubectl get trussiumruntime "$runtime_name" --namespace "$namespace"
wait_for_runtime_image "$runtime_tag"
kubectl rollout status deployment/"$release-trussium-operator" --namespace "$namespace" --timeout=2m

if [[ -n "$runtime_rollback_tag" ]]; then
  set_runtime_image "$runtime_rollback_tag"
fi
