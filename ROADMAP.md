# Trussium Operator Roadmap

_Last updated: August 2026_

This roadmap describes the direction of the public Trussium Kubernetes
Operator.

Implementation tasks, defects, and narrowly scoped enhancements are tracked
through GitHub Issues.

---

## Current Focus

The production Kubernetes deployment contract for `TrussiumRuntime` workloads
has been implemented.

The operator now manages production-oriented runtime Deployments with:

- Startup, liveness, and readiness probes
- Numeric non-root execution
- `RuntimeDefault` seccomp
- Read-only root filesystem
- Disabled privilege escalation
- Dropped Linux capabilities
- Graceful shutdown-aware termination timing
- Zero-unavailable rolling updates
- Limited Deployment revision history
- Hostname topology spreading
- Managed PodDisruptionBudgets
- Additional Pod labels and annotations
- Node selectors
- Tolerations
- Affinity configuration

The operator continues to preserve its existing security boundaries:

- No direct Pod permissions
- No Secret mutation permissions
- No arbitrary PodSpec exposure
- No privileged runtime configuration
- No ServiceAccount token mounting for runtime workloads

The next priority is the runtime upgrade lifecycle and compatibility contract.

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

**Status:** ✅ Completed

Define the first Kubernetes API for declaring a Trussium runtime instance.

### Delivered

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

Provider credentials are never embedded directly in custom resources.

---

## Milestone O3 — Core Runtime Reconciliation

**Status:** ✅ Completed

Reconcile a functional Trussium runtime from a `TrussiumRuntime` resource.

### Delivered

- `TrussiumRuntime` controller scaffold
- Controller registration with the manager
- Watches for `TrussiumRuntime`
- Watches for owned ConfigMaps
- Watches for owned ServiceAccounts
- Watches for owned Services
- Watches for owned Deployments
- ConfigMap reconciliation
- ServiceAccount reconciliation
- Service reconciliation
- Deployment reconciliation
- Stable Kubernetes labels
- Stable selectors
- Deterministic resource names
- Controller owner references
- Kubernetes garbage-collection ownership
- Create-or-update reconciliation
- Drift correction
- Deleted-resource recreation
- Idempotent repeated reconciliation
- Runtime image tag rendering
- Runtime image digest rendering
- Image-pull Secret projection
- Runtime environment projection
- Provider configuration projection
- Provider credential Secret projection
- Service configuration projection
- Replica projection
- Resource request and limit projection
- Dedicated ServiceAccount
- Disabled ServiceAccount token mounting
- Disabled Kubernetes Service links
- Least-privilege generated RBAC
- No Secret read permissions
- Unit-tested resource builders
- Fake-client reconciliation tests
- Managed-resource ownership tests
- Drift-correction tests
- Deleted-resource recreation tests
- Idempotency tests
- Core reconciliation documentation
- Managed-resource ownership architecture decision record

Status reporting and Kubernetes Events were intentionally deferred to the next
milestone and subsequently implemented in Milestone O4.

---

## Milestone O4 — Status and Kubernetes Events

**Status:** ✅ Completed

Expose runtime state through Kubernetes-native status and Events.

### Delivered

- Status-subresource updates
- Semantic no-op status detection
- Condition transition-time preservation
- `observedGeneration`
- Ready replica projection
- Available replica projection
- Current runtime image projection
- Internal runtime Service endpoint
- `ConfigurationValid` condition
- `Progressing` condition
- `Available` condition
- `Ready` condition
- `Degraded` condition
- Stable condition reasons
- Condition observed generations
- Deployment generation observation
- Deployment progress reporting
- Progress deadline failure reporting
- Deployment replica failure reporting
- Scale-to-zero status semantics
- Provider credential Secret existence validation
- Image-pull Secret existence validation
- Namespace-scoped Secret validation
- Secret-reference watch
- Deployment blocking for invalid references
- Secret-reference recovery
- Status-only primary-update filtering
- Modern `events.k8s.io/v1` recorder
- `RuntimeProgressing` Event
- `RuntimeReady` Event
- `RuntimeRecovered` Event
- `RuntimeScaledToZero` Event
- `ConfigurationInvalid` Event
- `ReconciliationFailed` Event
- `RuntimeDegraded` Event
- Duplicate transition-Event prevention
- Status RBAC
- Read-only Secret RBAC
- Modern Events RBAC
- No Pod permissions
- Status construction tests
- Secret-reference validation tests
- Transition-time tests
- Status-subresource tests
- No-op status-write tests
- Transition Event tests
- Secret recovery tests
- Secret mapping tests
- Status and Events documentation
- Status and transition-Event architecture decision record

The controller never reads, logs, copies, or exposes Secret values.

---

## Milestone O5 — Production Deployment Contract

**Status:** ✅ Completed

Reach parity with the production Kubernetes workload contract maintained by the
public Trussium runtime repository while preserving a constrained and secure
operator API.

### Delivered

- Runtime startup probe
- Runtime liveness probe
- Runtime readiness probe
- Named `http` probe port
- Numeric runtime UID `10001`
- Numeric runtime GID `10001`
- Non-root execution enforcement
- `RuntimeDefault` seccomp profile
- Read-only root filesystem
- Disabled privilege escalation
- Dropped Linux capabilities
- Non-privileged runtime container
- Disabled ServiceAccount token mounting
- Disabled Kubernetes Service links
- Graceful shutdown-aware Kubernetes termination
- Default 36-second termination grace period
- Configured drain timeout plus six-second Kubernetes safety margin
- RollingUpdate Deployment strategy
- Zero `maxUnavailable`
- One-replica `maxSurge`
- Deployment revision-history limit
- Hostname topology spreading
- Stable topology selector labels
- Managed `policy/v1` PodDisruptionBudget
- `maxUnavailable: 1` disruption policy
- PodDisruptionBudget controller ownership
- PodDisruptionBudget drift correction
- Deleted PodDisruptionBudget recreation
- PodDisruptionBudget owned-resource watch
- PodDisruptionBudget RBAC
- Additional runtime Pod labels
- Additional runtime Pod annotations
- Reserved operator-label protection
- Node selector support
- Toleration support
- Affinity support
- Additive `v1alpha1` API extension
- Kubernetes-native scheduling structures
- Production workload builder tests
- Probe contract tests
- Security-context tests
- Graceful-termination tests
- Rolling-update tests
- Topology-spread tests
- Pod metadata projection tests
- Scheduling projection tests
- PodDisruptionBudget builder tests
- PodDisruptionBudget reconciliation tests
- PodDisruptionBudget drift tests
- PodDisruptionBudget recreation tests
- API JSON round-trip tests
- Production workload documentation
- Production workload architecture decision record
- Canonical runtime image references updated to `ghcr.io/trussiumhq/trussium`

The operator does not expose an unrestricted Kubernetes `PodSpec`. Production
security and health invariants remain operator-controlled.

---

## Milestone O6 — Upgrade Lifecycle

**Status:** 🗓 Planned

Provide predictable runtime configuration and image upgrades with explicit
observability and compatibility boundaries.

### Deliverables

- Explicit desired runtime version reporting
- Explicit current runtime version reporting
- Runtime image transition observation
- Upgrade-in-progress condition
- Upgrade-complete condition
- Upgrade-failed condition
- Stable upgrade condition reasons
- Upgrade lifecycle Events
- Deployment rollout monitoring
- Failed rollout detection
- Upgrade completion detection
- Configuration checksum annotations
- Configuration-triggered rollouts
- Safe image transition behaviour
- Manual drift recovery
- Upgrade documentation
- Rollback documentation
- Operator/runtime compatibility documentation
- Runtime compatibility matrix

The upgrade lifecycle will build on Deployment status rather than introducing
direct Pod reads.

---

## Milestone O7 — Controller and End-to-End Testing

**Status:** 🗓 Planned

Validate the controller against Kubernetes API machinery and real clusters.

### Deliverables

- envtest integration suite
- CRD installation tests
- Controller-manager integration tests
- Reconciliation tests
- Managed-resource ownership tests
- Update tests
- Status tests
- Deletion tests
- Missing-reference tests
- Secret-reference recovery tests
- PodDisruptionBudget integration tests
- Upgrade lifecycle tests
- Kind-cluster tests
- Operator installation test
- Runtime Deployment readiness test
- Runtime health endpoint validation
- Runtime configuration rollout test
- Runtime image upgrade test
- Owned-resource garbage-collection test

---

## Milestone O8 — Operator Packaging and Release

**Status:** 🗓 Planned

Make the operator independently installable and releasable.

### Deliverables

- Production operator container
- Numeric non-root operator identity
- Hardened operator container security context
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

Add operational features after the core API, reconciliation, upgrade, and
packaging contracts are stable.

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
ghcr.io/trussiumhq/trussium:<version>
```

It does not import, compile, or duplicate runtime source code.

The operator manages Kubernetes lifecycle while the runtime repository remains
authoritative for:

- Runtime process behaviour
- Runtime APIs
- Provider integrations
- Health endpoint semantics
- Runtime configuration semantics
- Runtime container releases

Compatibility will be tracked by operator and runtime release:

| Operator version | Supported Trussium versions |
|---|---|
| Pre-release | To be established during the upgrade and end-to-end milestones |

---

## Immediate Priority

The next priority is Milestone O6:

1. Define the runtime upgrade lifecycle and compatibility contract.

The next milestone will focus on:

- Explicit desired and current runtime version reporting
- Upgrade-state conditions
- Upgrade lifecycle Events
- Deployment rollout monitoring
- Rollout failure handling
- Upgrade completion detection
- Configuration-triggered rollouts
- Safe runtime image transitions
- Operator/runtime compatibility boundaries
- Upgrade documentation
- Rollback documentation
- Compatibility matrix foundations
