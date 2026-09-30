#!/usr/bin/env bash
set -euo pipefail

namespace="${UPGRADE_TEST_NAMESPACE:-trussium-operator-upgrade-test}"
release="${UPGRADE_TEST_RELEASE:-trussium-operator}"
previous_version="${PREVIOUS_CHART_VERSION:-0.3.1}"
runtime_tag="${TRUSSIUM_RUNTIME_TAG:-1.0.0}"
runtime_rollback_tag="${TRUSSIUM_RUNTIME_ROLLBACK_TAG:-}"
operator_image_tag="${TRUSSIUM_OPERATOR_IMAGE_TAG:-0.14.0}"
verify_runtime_image_status="${VERIFY_RUNTIME_IMAGE_STATUS:-false}"
runtime_repository="ghcr.io/trussiumhq/trussium"
runtime_name="upgrade-smoke"
previous_chart="/tmp/trussium-operator-${previous_version}.tgz"

wait_for_runtime_image() {
  local tag="$1"
  local expected_image="${runtime_repository}:${tag}"
  local configured_image

  if [[ "$verify_runtime_image_status" == "true" ]]; then
    echo "Waiting for ${runtime_name} to report successful image ${expected_image}"
    kubectl wait \
      --namespace "$namespace" \
      --for=condition=Available \
      --timeout=3m \
      "deployment/${runtime_name}"
    kubectl wait \
      --namespace "$namespace" \
      --for="jsonpath={.status.lastSuccessfulImage}=${expected_image}" \
      --timeout=2m \
      "trussiumruntime/${runtime_name}"
    kubectl wait \
      --namespace "$namespace" \
      --for='condition=Ready' \
      --timeout=2m \
      "trussiumruntime/${runtime_name}"
    kubectl rollout status \
      "deployment/${runtime_name}" \
      --namespace "$namespace" \
    --timeout=2m

    configured_image="$(kubectl get deployment "$runtime_name" \
      --namespace "$namespace" \
      -o jsonpath='{.spec.template.spec.containers[0].image}')"
    if [[ "$configured_image" != "$expected_image" ]]; then
      echo "Expected runtime Deployment image ${expected_image}, got ${configured_image}" >&2
      exit 1
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

kubectl get trussiumruntime "$runtime_name" --namespace "$namespace"

helm upgrade "$release" charts/trussium-operator \
  --namespace "$namespace" \
  --set "image.tag=$operator_image_tag" \
  --timeout=3m

service_account="system:serviceaccount:${namespace}:${release}-trussium-operator"
for resource in horizontalpodautoscalers.autoscaling networkpolicies.networking.k8s.io; do
  for verb in get list watch; do
    if [[ "$(kubectl auth can-i "$verb" "$resource" \
      --all-namespaces --as="$service_account")" != "yes" ]]; then
      echo "Upgraded Operator service account cannot ${verb} ${resource} cluster-wide" >&2
      return 1
    fi
  done
done

# The old controller can have informers stuck after starting without these
# permissions. Restart only after confirming the upgraded ClusterRole grants
# them, then wait for a clean cache sync and rollout.
kubectl rollout restart "deployment/${release}-trussium-operator" \
  --namespace "$namespace"
kubectl rollout status "deployment/${release}-trussium-operator" \
  --namespace "$namespace" --timeout=3m

wait_for_runtime_image "$initial_runtime_tag"

if [[ -n "$runtime_rollback_tag" ]]; then
  set_runtime_image "$runtime_tag"
fi

kubectl get trussiumruntime "$runtime_name" --namespace "$namespace"
wait_for_runtime_image "$runtime_tag"
kubectl rollout status deployment/"$release-trussium-operator" --namespace "$namespace" --timeout=2m

if [[ -n "$runtime_rollback_tag" ]]; then
  set_runtime_image "$runtime_rollback_tag"
else
  # Historical release-matrix entries still exercise Helm chart rollback.
  helm rollback "$release" 1 --namespace "$namespace" --wait --timeout=3m
  kubectl get trussiumruntime "$runtime_name" --namespace "$namespace"
  wait_for_runtime_image "$runtime_tag"
fi
