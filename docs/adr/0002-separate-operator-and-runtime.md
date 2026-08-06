# ADR 0002 — Separate the Operator from the Runtime

- Status: Accepted
- Date: August 2026

## Context

The public `trussium` repository contains the Python AI runtime, provider
adapters, HTTP API, and runtime packaging.

The Kubernetes operator has a different language, lifecycle, release cadence,
and responsibility.

## Decision

The operator will live in the separate public repository:

```text
trussium-operator
```

The runtime repository remains:

```text
trussium
```

The exact Go module is:

```text
github.com/trussiumhq/trussium-operator
```

The operator will consume released runtime images:

```text
ghcr.io/trussiumhq/trussium:<version>
```

The operator will not import or duplicate Python runtime source.

## Responsibilities

### Runtime Repository

- AI execution
- Provider adapters
- Runtime API
- Streaming
- Timeouts and cancellation
- Runtime configuration semantics
- Runtime container publication

### Operator Repository

- Kubernetes APIs
- Reconciliation
- Runtime Deployment construction
- Kubernetes configuration projection
- Runtime status
- Operator installation

## Consequences

### Positive

- Independent release cycles
- Clear ownership
- No cross-language source dependency
- Compatibility can be tested against immutable runtime releases
- Kubernetes functionality remains focused

### Negative

- Compatibility must be explicitly documented.
- End-to-end tests must coordinate released artifacts.
- Deployment-contract changes require cross-repository planning.

## Commercial Integration

A future commercial Trussium control plane may coordinate the open-source
operator through stable APIs.

The operator must remain independently functional without that control plane.
