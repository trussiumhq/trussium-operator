# ADR 0004 — Use Owner References for Managed Resources

- Status: Accepted
- Date: August 2026

## Context

Each `TrussiumRuntime` requires several namespaced Kubernetes resources:

- ConfigMap
- ServiceAccount
- Service
- Deployment

The operator must establish ownership, correct drift, recreate deleted
resources, and safely remove managed resources when the custom resource is
deleted.

The current milestone does not create external infrastructure or resources
outside Kubernetes garbage collection.

## Decision

Every resource managed for a `TrussiumRuntime` will have a Kubernetes
controller owner reference pointing to that custom resource.

The controller will use deterministic names and create-or-update
reconciliation.

The controller will watch:

- `TrussiumRuntime`
- Owned ConfigMaps
- Owned ServiceAccounts
- Owned Services
- Owned Deployments

No finalizer will be introduced during core reconciliation.

## Naming Decision

Managed resources use the owning `TrussiumRuntime` name and namespace.

For example:

    TrussiumRuntime/private-ai
    ConfigMap/private-ai
    ServiceAccount/private-ai
    Service/private-ai
    Deployment/private-ai

This keeps discovery simple and avoids additional name-generation state.

## Drift Decision

The operator owns the fields required by the public `TrussiumRuntime`
contract.

During reconciliation, those fields are reset to their desired values.

Unsupported direct modifications to generated resources may be overwritten.

New user customization must be represented through explicit custom-resource
fields rather than unmanaged mutations.

## Secret Decision

The operator projects provider credentials through Kubernetes Secret key
references.

It does not read Secret values during core reconciliation.

Secret read permissions are therefore intentionally excluded from controller
RBAC.

## Deletion Decision

Kubernetes garbage collection removes namespaced managed resources when the
owning `TrussiumRuntime` is deleted.

A finalizer is unnecessary because no external cleanup is required.

A future capability may introduce a finalizer only when Kubernetes owner
references cannot safely complete the required cleanup.

## Consequences

### Positive

- Kubernetes-native lifecycle management
- Automatic garbage collection
- Clear ownership
- Reconciliation after secondary-resource changes
- Deterministic resource discovery
- Straightforward drift correction
- No custom deletion state machine

### Negative

- Direct edits to managed fields are overwritten.
- Resource names cannot vary independently from the custom resource.
- Ownership conflicts must surface as reconciliation errors.
- Future external-resource management may require a finalizer.

## Follow-Up

The next milestone will add:

- Observed generation
- Ready and available replica counts
- Runtime endpoint
- Kubernetes conditions
- Reconciliation Events
- Stable condition reasons
