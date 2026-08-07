# ADR 0006 — Enforce a Production Runtime Workload Contract

- Status: Accepted
- Date: August 2026

## Context

The operator initially created the Kubernetes resources required to run
Trussium, but the Deployment remained intentionally minimal.

A production runtime workload requires consistent health checks, security
controls, graceful shutdown behaviour, safe rolling updates, disruption
protection, and predictable scheduling behaviour.

Exposing the complete Kubernetes PodSpec through the custom resource would
provide flexibility but would also weaken the operator's ability to guarantee a
supported and secure runtime environment.

The public Trussium runtime repository already defines a maintained Kubernetes
deployment contract.

## Decision

The operator will enforce an opinionated production workload contract rather
than expose arbitrary Pod and container configuration.

The managed Deployment will include:

- Startup probe
- Liveness probe
- Readiness probe
- Numeric non-root runtime identity
- `RuntimeDefault` seccomp
- Disabled privilege escalation
- Read-only root filesystem
- Dropped Linux capabilities
- Disabled ServiceAccount token automount
- Disabled Service links
- Graceful Kubernetes termination timing
- Zero-unavailable rolling updates
- Limited Deployment revision history
- Hostname topology spreading

The operator will additionally own a `policy/v1` PodDisruptionBudget.

## Runtime Identity

The managed runtime executes as:

    UID: 10001
    GID: 10001

These values match the maintained runtime container contract.

Users cannot override the runtime identity in `v1alpha1`.

## Health Contract

The operator uses:

    /health/live

for startup and liveness checks and:

    /health/ready

for readiness.

Probe timing is operator-managed and not configurable through the custom
resource.

## Graceful Termination

Kubernetes termination grace is derived from the runtime shutdown drain
configuration.

The controller adds a six-second margin to the runtime drain timeout.

The default therefore becomes:

    30 + 6 = 36 seconds

## Rollout Strategy

Deployments use:

    maxUnavailable: 0
    maxSurge: 1

This prioritizes availability during ordinary rolling updates.

## PodDisruptionBudget

Every runtime receives:

    maxUnavailable: 1

The PodDisruptionBudget is owned by the custom resource and reconciled like the
other managed resources.

## Safe Customization

The API exposes a constrained set of Pod-level customization:

- Additional labels
- Additional annotations
- Node selector
- Tolerations
- Affinity

Operator-owned labels remain authoritative.

The API does not embed an unrestricted PodSpec.

## Consequences

### Positive

- Consistent production posture
- Secure defaults
- Predictable runtime health behaviour
- Safer rollouts
- Graceful request draining
- Kubernetes disruption protection
- Controlled scheduling flexibility
- Stable operator support boundary

### Negative

- Advanced Kubernetes workload customization is intentionally limited.
- Probe configuration cannot be tuned per runtime.
- Security identity cannot be overridden.
- PodDisruptionBudget policy cannot yet be customized.

## Alternatives Considered

### Expose the Entire PodSpec

Rejected because it would:

- Make the API tightly coupled to Kubernetes PodSpec evolution
- Allow users to bypass operator security controls
- Make compatibility difficult to reason about
- Increase reconciliation complexity
- Create an effectively unbounded support surface

### Keep the Deployment Minimal

Rejected because the operator would not provide production-grade behaviour
despite managing runtime lifecycle.

## Follow-Up

The next milestone will define explicit runtime upgrade lifecycle behaviour and
compatibility handling.
