#!/usr/bin/env bash
set -euo pipefail

namespace="trussium-operator-helm-test"
release="trussium-operator"

cleanup() {
  helm uninstall "$release" --namespace "$namespace" --ignore-not-found
  kubectl delete namespace "$namespace" --ignore-not-found --wait=false
}
trap cleanup EXIT

helm install "$release" charts/trussium-operator \
  --namespace "$namespace" \
  --create-namespace

kubectl rollout status deployment/"$release-trussium-operator" \
  --namespace "$namespace" \
  --timeout=2m
