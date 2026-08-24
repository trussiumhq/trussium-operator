# Trussium Operator Architecture

## Purpose

The Trussium Operator provides a declarative Kubernetes API for deploying and
managing released Trussium runtime instances.

It translates Trussium-specific desired state into Kubernetes resources and
continuously reconciles those resources until the observed cluster state
matches the declaration.

## System Context

```text
Developer or platform engineer
             │
             │ kubectl or GitOps
             ▼
      TrussiumRuntime CR
             │
             ▼
      Trussium Operator
             │
             ├── ConfigMap
             ├── ServiceAccount
             ├── Deployment
             ├── Service
             ├── PodDisruptionBudget
             ├── Optional NetworkPolicy
             └── Status and Events
                         │
                         ▼
        Trussium runtime container
                         │
                         ▼
        OpenAI, Ollama, and compatible providers
```

## Repository Boundary

### `trussium`

The public `trussium` repository owns the runtime data plane:

- Provider-neutral AI APIs
- Provider adapters
- Execution behaviour
- Streaming behaviour
- Runtime configuration semantics
- Runtime observability
- Runtime container images
- Runtime package releases

### `trussium-operator`

This repository owns Kubernetes lifecycle management:

- Kubernetes APIs and CRDs
- Kubernetes controllers
- Desired resource construction
- Resource ownership
- Runtime deployment configuration
- Rollout status
- Kubernetes Events
- Operator packaging

The operator depends on the runtime's published deployment contract, not on its
Python implementation.

## Image Relationship

The runtime repository publishes immutable container versions:

```text
ghcr.io/trussiumhq/trussium:vX.Y.Z
```

A future `TrussiumRuntime` resource will select one of those versions.

The operator will create a Kubernetes Deployment containing the selected image.
Kubernetes nodes, rather than the operator process, pull that image through the
cluster container runtime.

## Reconciliation Model

A reconciliation iteration will eventually:

1. Read the `TrussiumRuntime` resource.
2. Validate referenced Kubernetes objects.
3. Construct the desired managed resources.
4. Create or update those resources.
5. Inspect Deployment rollout state.
6. Update `TrussiumRuntime.status`.
7. Emit Kubernetes Events for meaningful transitions.
8. Return without unnecessary polling.

Managed resources must:

- Be deterministic.
- Be idempotently created or updated.
- Carry stable labels.
- Carry a controller owner reference.
- Avoid embedding secrets.
- Be reconstructable from the custom resource and referenced objects.

## Planned API

The first API will be:

```text
Group:   runtime.trussium.io
Version: v1alpha1
Kind:    TrussiumRuntime
Scope:   Namespaced
```

The API will expose Trussium-specific intent rather than embedding an arbitrary
Kubernetes Pod or Deployment specification.

## Security Model

Provider credentials will never be accepted as plaintext custom-resource
fields.

The resource will reference Kubernetes Secrets by name. The operator will
project those references into runtime pods without logging credential values.

The operator manager will run:

- As a non-root user
- With minimum required Kubernetes permissions
- With configurable leader election, enabled by the Helm chart by default
- With health and readiness endpoints
- With structured controller-runtime logging

## Availability Model

Runtime workloads support:

- Multiple replicas
- Startup, liveness, and readiness probes
- PodDisruptionBudget
- Topology spreading
- Rolling updates
- Graceful shutdown
- Resource requests and limits

The Helm chart enables leader election by default. Keep it enabled when the
operator is deployed with multiple replicas.

## Deletion Model

Kubernetes owner references and garbage collection will be preferred.

A finalizer will only be added when the operator must clean up external state or
resources that Kubernetes garbage collection cannot safely handle.

The first runtime reconciliation should not require a finalizer.
