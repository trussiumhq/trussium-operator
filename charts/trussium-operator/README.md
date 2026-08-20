# Trussium Operator Helm Chart

This chart installs the Trussium Operator, including its `TrussiumRuntime` CRD,
controller, RBAC, leader-election permissions, and metrics service.

It does not install a Trussium runtime workload. Install the runtime chart from
[`trussiumhq/trussium-helm`](https://github.com/trussiumhq/trussium-helm), then
create `TrussiumRuntime` resources for operator-managed runtime instances.

## Install

```bash
helm install trussium-operator ./charts/trussium-operator \
  --namespace trussium-operator-system \
  --create-namespace
```

OCI distribution is added with the chart release automation.

## Configuration

The controller image defaults to the chart `appVersion`. Override it with
`image.repository` and `image.tag`; configure standard Kubernetes placement,
resource, pull-secret, and service-account settings through `values.yaml`.

Validate local rendering with:

```bash
make helm-lint
```
