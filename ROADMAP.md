# Trussium Operator Roadmap

_Last updated: August 2026_

This roadmap describes the direction of the public Trussium Kubernetes
Operator.

Implementation tasks, defects, and narrowly scoped enhancements are tracked
through GitHub Issues.

---

## Current Focus

The repository foundation has been established using Go, Kubebuilder, and
controller-runtime.

The next priority after the foundation is merged will be the first namespaced
Kubernetes API:

```text
Group:   runtime.trussium.io
Version: v1alpha1
Kind:    TrussiumRuntime
Scope:   Namespaced
```

The initial API will describe the desired Trussium runtime image, replicas,
provider configuration references, runtime settings, resource boundaries, and
Service configuration.

No runtime reconciliation will be introduced until the API contract is
reviewed and merged.

---

## Status Definitions

- ✅ **Completed** — Core completion criteria have been delivered.
- 🚧 **In Progress** — Implementation is active.
- 🗓 **Planned** — Accepted but not substantially implemented.
- ⏸ **Deferred** — Intentionally postponed pending prerequisites.

---

## Milestone O1 — Project Foundation

**Status:** ✅ Completed

Establish the repository, engineering workflow, documentation, and
architectural boundaries.

### Delivered

- Public `trussium-operator` repository
- Apache License 2.0
- Go module structure
- Kubebuilder `go/v4` project scaffold
- `trussium.io` API domain
- controller-runtime manager foundation
- Generated Kustomize resources
- Generated RBAC foundation
- Generated metrics resources
- Go formatting and vet workflow
- Unit-test workflow
- golangci-lint workflow
- Generated-artifact verification
- GitHub Actions continuous integration
- Dependabot configuration
- Bug and feature issue forms
- Pull request template
- Contribution guide
- Development guide
- Security policy
- Code of Conduct
- Architecture documentation
- Architecture Decision Records
- Branch-protection guidance
- Conventional Commit workflow


---

## Milestone O2 — TrussiumRuntime API

**Status:** 🗓 Planned

Define the first Kubernetes API for declaring a Trussium runtime instance.

### Deliverables

- `runtime.trussium.io/v1alpha1`
- Namespaced `TrussiumRuntime` resource
- Typed desired-state specification
- Typed observed status
- Status subresource
- OpenAPI validation
- Required-field validation
- Safe defaults
- Kubernetes printer columns
- Sample resources
- API contract documentation
- API serialization tests
- Generated CRD manifests
- Generated deep-copy code

### Initial Specification Areas

- Runtime image repository
- Image tag or digest
- Image pull policy
- Image-pull Secret references
- Replica count
- Provider type
- Provider model
- Provider base URL
- Provider credential Secret reference
- Runtime timeout configuration
- Graceful-shutdown configuration
- Service type and port
- Resource requests and limits

Provider credentials will never be embedded directly in custom resources.

---

## Milestone O3 — Core Runtime Reconciliation

**Status:** 🗓 Planned

Reconcile a functional Trussium runtime from a `TrussiumRuntime` resource.

### Deliverables

- ConfigMap reconciliation
- ServiceAccount reconciliation
- Service reconciliation
- Deployment reconciliation
- Stable Kubernetes labels
- Controller owner references
- Drift correction
- Idempotent reconciliation
- Runtime container image projection
- Runtime environment projection
- Resource request and limit projection
- Unit-tested resource builders

---

## Milestone O4 — Status and Kubernetes Events

**Status:** 🗓 Planned

Expose runtime state through Kubernetes-native status and events.

### Deliverables

- `observedGeneration`
- Desired and current image
- Ready replicas
- Available replicas
- Runtime endpoint
- `Ready` condition
- `Available` condition
- `Progressing` condition
- `Degraded` condition
- `ConfigurationValid` condition
- Reconciliation events
- Missing Secret reporting
- Deployment failure reporting
- Stable condition reasons

---

## Milestone O5 — Production Deployment Contract

**Status:** 🗓 Planned

Reach parity with the production Kubernetes contract validated by the public
Trussium runtime repository.

### Deliverables

- Startup probe
- Liveness probe
- Readiness probe
- Non-root container execution
- Read-only root filesystem support
- Dropped Linux capabilities
- No-new-privileges support
- Graceful termination timing
- Provider Secret references
- Image-pull Secret references
- PodDisruptionBudget
- Topology spreading
- Zero-unavailable rolling updates
- Pod labels and annotations
- Node selector
- Tolerations
- Affinity configuration

---

## Milestone O6 — Upgrade Lifecycle

**Status:** 🗓 Planned

Provide predictable runtime configuration and image upgrades.

### Deliverables

- Immutable image-tag support
- Image-digest support
- Deployment rollout monitoring
- Configuration checksum annotations
- Configuration-triggered rollouts
- Current and desired version reporting
- Failed rollout conditions
- Manual drift recovery
- Upgrade documentation
- Rollback documentation
- Runtime compatibility matrix

---

## Milestone O7 — Controller and End-to-End Testing

**Status:** 🗓 Planned

Validate the controller against Kubernetes API machinery and real clusters.

### Deliverables

- envtest integration suite
- CRD installation tests
- Reconciliation tests
- Managed-resource ownership tests
- Update tests
- Status tests
- Deletion tests
- Missing-reference tests
- Kind-cluster tests
- Operator installation test
- Runtime Deployment readiness test
- Runtime health endpoint validation
- Runtime configuration rollout test
- Owned-resource garbage-collection test

---

## Milestone O8 — Operator Packaging and Release

**Status:** 🗓 Planned

Make the operator independently installable and releasable.

### Deliverables

- Production operator container
- Numeric non-root runtime identity
- Multi-platform AMD64 and ARM64 image
- GitHub Container Registry publication
- Software bill of materials
- Build provenance
- Release assets
- CRD installation bundle
- Operator Helm chart
- Release notes
- Upgrade notes
- Compatibility documentation
- Semantic release process

---

## Milestone O9 — Operational Extensions

**Status:** 🗓 Planned

Add operational features after the core API and reconciliation contracts are
stable.

### Potential Deliverables

- HorizontalPodAutoscaler reconciliation
- NetworkPolicy reconciliation
- Leader election
- Configurable watched namespaces
- Controller reconciliation metrics
- ServiceMonitor integration
- Runtime health aggregation
- Provider configuration validation
- Admission webhook
- API conversion webhook
- Runtime compatibility validation

Webhooks will not be added until schema validation is insufficient for a
required invariant.

---

## Open-Source Boundary

The public operator will include:

- Custom resources
- Core reconciliation
- Runtime status
- Configuration and Secret references
- Production deployment settings
- Standard runtime upgrades
- Drift correction
- Operator metrics
- Installation packaging

Commercial Trussium products may provide:

- Multi-cluster fleet inventory
- Central configuration management
- Upgrade waves
- Maintenance windows
- Cross-cluster policy enforcement
- Enterprise audit retention
- Compliance reporting
- Organization-wide approval workflows
- Hosted control-plane services

The public operator will remain independently usable without a commercial
control plane.

---

## Runtime Dependency Contract

The operator consumes released Trussium runtime images:

```text
ghcr.io/trussium/trussium:<version>
```

It does not import, compile, or duplicate runtime source code.

Compatibility will be tracked by operator and runtime release:

| Operator version | Supported Trussium versions |
|---|---|
| Pre-release | To be established during the first end-to-end release |

---

## Immediate Priority

The next priority after the foundation is merged is:

1. Define the `TrussiumRuntime v1alpha1` API.

The API must be reviewed before Deployment reconciliation begins.
