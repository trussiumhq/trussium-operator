#!/usr/bin/env bash
set -euo pipefail

namespace="trussium-operator-helm-test"
release="trussium-operator"
image_repository="${TRUSSIUM_OPERATOR_IMAGE_REPOSITORY:-ghcr.io/trussiumhq/trussium-operator}"
image_tag="${TRUSSIUM_OPERATOR_IMAGE_TAG:-latest}"

cleanup() {
  helm uninstall "$release" --namespace "$namespace" --ignore-not-found
  kubectl delete namespace "$namespace" --ignore-not-found --wait=false
}

cleanup_on_success() {
  status=$?
  if [ "$status" -eq 0 ]; then
    cleanup
  fi
  exit "$status"
}
trap cleanup_on_success EXIT

helm install "$release" charts/trussium-operator \
  --namespace "$namespace" \
  --create-namespace \
  --set "image.repository=$image_repository" \
  --set "image.tag=$image_tag"

kubectl rollout status deployment/"$release-trussium-operator" \
  --namespace "$namespace" \
  --timeout=2m
