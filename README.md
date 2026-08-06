# Trussium Operator

The Trussium Operator is the Kubernetes-native lifecycle manager for
[Trussium](https://github.com/trussium/trussium) runtime instances.

It provides a declarative Kubernetes API for deploying, configuring, upgrading,
and observing released Trussium runtime containers.

> **Project status:** Pre-alpha. The repository currently contains the
> engineering foundation. The first `TrussiumRuntime` API has not yet been
> released.

## Architecture

The operator will watch `TrussiumRuntime` custom resources and reconcile the
Kubernetes resources required to operate each runtime instance.

```text
TrussiumRuntime custom resource
             │
             ▼
      Trussium Operator
             │
             ├── ServiceAccount
             ├── ConfigMap
             ├── Deployment
             ├── Service
             ├── PodDisruptionBudget
             └── Status conditions
                         │
                         ▼
       ghcr.io/trussium/trussium:<version>
```

The operator does not contain or compile the Trussium Python runtime. It
consumes released runtime container images and manages their Kubernetes
deployment lifecycle.

## Repository responsibilities

This repository owns:

- Kubernetes custom resource definitions
- Kubernetes controllers
- Runtime deployment reconciliation
- Runtime configuration projection
- Kubernetes status reporting
- Operator packaging and installation
- Runtime and operator compatibility documentation

The public [`trussium`](https://github.com/trussium/trussium) repository owns:

- Runtime APIs
- Provider adapters
- AI execution behaviour
- Runtime configuration semantics
- Runtime container images
- Runtime package and container releases

## Technology

- Go
- Kubebuilder
- controller-runtime
- controller-tools
- Kustomize
- envtest
- Kind
- Helm in a later milestone

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for local setup and validation.

The standard validation workflow is:

```bash
make generate
make manifests
make fmt
make vet
make test
make lint
```

## Roadmap

The public operator roadmap is maintained in [ROADMAP.md](ROADMAP.md).

The next planned milestone is the initial
`runtime.trussium.io/v1alpha1` `TrussiumRuntime` API.

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a change.

## Security

Do not report suspected vulnerabilities through public issues. Follow
[SECURITY.md](SECURITY.md).

## Licence

Trussium Operator is licensed under the Apache License 2.0.
