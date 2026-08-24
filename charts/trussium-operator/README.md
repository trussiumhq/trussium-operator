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

Released chart versions are available from the OCI registry:

```bash
helm install trussium-operator oci://ghcr.io/trussiumhq/charts/trussium-operator \
  --version <version> \
  --namespace trussium-operator-system \
  --create-namespace
```

## Configuration

The controller image defaults to the chart `appVersion`. Override it with
`image.repository` and `image.tag`; configure standard Kubernetes placement,
resource, pull-secret, and service-account settings through `values.yaml`.
The controller mounts its ServiceAccount token because it must authenticate to
the Kubernetes API; managed Trussium runtime Pods remain tokenless by default.

Set `watchNamespace` to reconcile `TrussiumRuntime` resources in one namespace.
The default empty value watches all namespaces. Namespace scoping limits the
manager cache and watches; the chart's cluster-scoped RBAC remains unchanged.

## Leader Election

`leaderElection.enabled` defaults to `true`. Keep it enabled before increasing
`replicaCount` above one: it ensures only one controller manager actively
reconciles resources at a time. Set it to `false` only for a deliberate
single-manager deployment.

## Prometheus ServiceMonitor

Set `metrics.serviceMonitor.enabled=true` when the Prometheus Operator CRDs are
installed. The ServiceMonitor scrapes the authenticated HTTPS metrics endpoint.
Its default `insecureSkipVerify=true` supports the controller's self-signed
certificate; set it to `false` only after configuring a trusted certificate.

Validate local rendering with:

```bash
make helm-lint
```
