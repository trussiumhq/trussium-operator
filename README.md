# Trussium Operator

The Trussium Operator is the Kubernetes-native lifecycle manager for
[Trussium](https://github.com/trussium/trussium) runtime instances.

It provides a declarative Kubernetes API for deploying, configuring, upgrading,
and observing released Trussium runtime containers.

> **Project status:** Pre-alpha. The initial `TrussiumRuntime v1alpha1` API and
> core ConfigMap, ServiceAccount, Service, and Deployment reconciliation are
> implemented. Runtime status and Kubernetes Events are the next milestone.

## Architecture

    TrussiumRuntime custom resource
                 │
                 ▼
          Trussium Operator
                 │
                 ├── ConfigMap
                 ├── ServiceAccount
                 ├── Service
                 └── Deployment
                             │
                             ▼
           ghcr.io/trussium/trussium:<version>

The operator does not contain or compile the Trussium Python runtime. It
consumes released runtime container images and manages their Kubernetes
deployment lifecycle.

## Custom Resource

The initial API is:

    Group:   runtime.trussium.io
    Version: v1alpha1
    Kind:    TrussiumRuntime
    Scope:   Namespaced

Example:

    apiVersion: runtime.trussium.io/v1alpha1
    kind: TrussiumRuntime
    metadata:
      name: private-ai
      namespace: trussium
    spec:
      image:
        repository: ghcr.io/trussium/trussium
        tag: v0.23.0

      provider:
        type: ollama
        model: llama3.2
        baseURL: http://ollama.ollama.svc.cluster.local:11434/v1

See [docs/CUSTOM_RESOURCE.md](docs/CUSTOM_RESOURCE.md) for the complete API
contract.

## Core Reconciliation

For every `TrussiumRuntime`, the operator manages:

- ConfigMap
- ServiceAccount
- Service
- Deployment

The controller uses stable labels, controller owner references, deterministic
resource names, and create-or-update reconciliation.

It recreates deleted managed resources and corrects configuration drift.

See [docs/RECONCILIATION.md](docs/RECONCILIATION.md) for the complete
reconciliation contract.

## Repository Responsibilities

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

Run:

    make generate
    make manifests
    make fmt
    make vet
    make test
    make lint

## Roadmap

The public operator roadmap is maintained in [ROADMAP.md](ROADMAP.md).

The next milestone adds Kubernetes-native runtime status and reconciliation
Events.

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a change.

## Security

Do not report suspected vulnerabilities through public issues. Follow
[SECURITY.md](SECURITY.md).

## Licence

Trussium Operator is licensed under the Apache License 2.0.
