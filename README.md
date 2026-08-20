# Trussium Operator

The Trussium Operator is the Kubernetes-native lifecycle manager for
[Trussium](https://github.com/trussium/trussium) runtime instances.

It provides a declarative Kubernetes API for deploying, configuring, upgrading,
and observing released Trussium runtime containers.

> **Project status:** Production workload hardening is implemented, including health probes,
> non-root execution, secure container defaults, graceful termination,
> zero-unavailable rolling updates, topology spreading, PodDisruptionBudget
> management, and constrained scheduling customization.

## Architecture

    TrussiumRuntime custom resource
                 │
                 ▼
          Trussium Operator
                 │
                 ├── validates referenced Secrets
                 ├── reconciles ConfigMap
                 ├── reconciles ServiceAccount
                 ├── reconciles Service
                 ├── reconciles Deployment
                 ├── updates runtime status
                 └── emits Kubernetes Events
                             │
                             ▼
           ghcr.io/trussiumhq/trussium:<version>

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
        repository: ghcr.io/trussiumhq/trussium
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
- PodDisruptionBudget

The controller uses stable labels, controller owner references, deterministic
resource names, and create-or-update reconciliation.

It recreates deleted managed resources and corrects configuration drift.

See [docs/RECONCILIATION.md](docs/RECONCILIATION.md) for the reconciliation
contract.

## Runtime Status

The operator reports:

- Observed generation
- Ready replicas
- Available replicas
- Current runtime image
- Internal Service endpoint
- Configuration validity
- Deployment progress
- Runtime availability
- Runtime readiness
- Degraded state

Referenced provider and image-pull Secrets are checked for existence without
reading their values.

The operator emits transition-based `events.k8s.io/v1` Events for readiness,
progress, recovery, configuration failures, degraded state, and reconciliation
failures.

See [docs/STATUS_AND_EVENTS.md](docs/STATUS_AND_EVENTS.md).

## Repository Responsibilities

This repository owns:

- Kubernetes custom resource definitions
- Kubernetes controllers
- Runtime deployment reconciliation
- Runtime configuration projection
- Kubernetes status reporting
- Kubernetes lifecycle Events
- Operator packaging and installation
- Runtime and operator compatibility documentation

The public [`trussium`](https://github.com/trussium/trussium) repository owns:

- Runtime APIs
- Provider adapters
- AI execution behaviour
- Runtime configuration semantics
- Runtime container images
- Runtime package and container releases

## Production Workload Contract

Managed runtime Pods include:

- Startup, liveness, and readiness probes
- Numeric non-root execution
- RuntimeDefault seccomp
- Read-only root filesystem
- Disabled privilege escalation
- Dropped Linux capabilities
- Graceful Kubernetes termination
- Zero-unavailable rolling updates
- Topology spreading
- PodDisruptionBudget protection

Supported Pod customization includes:

- Additional metadata
- Node selectors
- Tolerations
- Affinity

See [docs/PRODUCTION_RUNTIME.md](docs/PRODUCTION_RUNTIME.md).

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
    make test-e2e

## Runtime Upgrades

The operator observes runtime image transitions through Kubernetes Deployment
status.

Upgrade status reports:

- Desired runtime image
- Deployment-configured runtime image
- Last successfully rolled-out image
- Upgrade progress
- Upgrade completion
- Upgrade failure

Runtime configuration changes also trigger Deployment revisions through a
deterministic Pod-template checksum.

The operator does not automatically roll back failed upgrades and does not
require Pod or ReplicaSet permissions.

See:

- [docs/UPGRADES.md](docs/UPGRADES.md)
- [docs/COMPATIBILITY.md](docs/COMPATIBILITY.md)

## Roadmap

The public operator roadmap is maintained in [ROADMAP.md](ROADMAP.md).

The next milestone hardens the managed Trussium runtime workload for
production Kubernetes operation.

## Contributing

Read [CONTRIBUTING.md](CONTRIBUTING.md) before opening a change.

## Security

Do not report suspected vulnerabilities through public issues. Follow
[SECURITY.md](SECURITY.md).

## Licence

Trussium Operator is licensed under the Apache License 2.0.
