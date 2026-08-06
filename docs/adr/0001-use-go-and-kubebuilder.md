# ADR 0001 — Use Go and Kubebuilder

- Status: Accepted
- Date: August 2026

## Context

The Trussium Operator must define Kubernetes APIs, watch custom resources,
reconcile owned resources, report status, and integrate with Kubernetes-native
testing and packaging workflows.

## Decision

The operator will be implemented in Go using:

- Kubebuilder
- controller-runtime
- controller-tools
- Kustomize
- envtest
- Kind

The repository will use the Kubebuilder `go/v4` project layout.

## Rationale

Go and controller-runtime provide direct access to Kubernetes API machinery and
the established reconciliation model used by Kubernetes controllers.

Kubebuilder provides consistent scaffolding for:

- API types
- Controllers
- RBAC generation
- CRD generation
- Manager deployment
- Health endpoints
- Testing
- Tool-version management

## Consequences

### Positive

- Strong alignment with Kubernetes conventions
- Typed API definitions
- Mature controller testing
- Generated RBAC and CRD assets
- Straightforward access to Kubernetes libraries

### Negative

- The operator uses a different language from the Python Trussium runtime.
- Contributors must understand Kubernetes API conventions and generated code.
- Kubernetes dependency upgrades require compatibility management.

## Follow-Up

The operator and runtime communicate through container, configuration, health,
and Kubernetes contracts rather than sharing implementation code.
