# ADR 0003 — Define the TrussiumRuntime v1alpha1 API

- Status: Accepted
- Date: August 2026

## Context

The Trussium Operator requires a Kubernetes API through which users and GitOps
systems can declare a Trussium runtime deployment.

The API must expose Trussium-specific intent while maintaining a clear boundary
between:

- Runtime configuration
- Kubernetes deployment configuration
- Provider credentials
- Controller-owned observed status

Exposing an unrestricted Pod or Deployment specification would make the API
difficult to validate, support, evolve, and secure.

## Decision

The initial API is:

    Group:   runtime.trussium.io
    Version: v1alpha1
    Kind:    TrussiumRuntime
    Scope:   Namespaced

The desired-state specification includes:

- Runtime image repository
- Exactly one image tag or digest
- Image pull policy
- Image-pull Secret references
- Replica count
- Provider type
- Provider model
- Optional provider base URL
- Provider credential Secret reference
- Optional runtime timeout overrides
- Service type and port
- Resource requests and limits

The status contract includes:

- Observed generation
- Ready replicas
- Available replicas
- Current image
- Runtime endpoint
- Standard Kubernetes conditions

## Credential Decision

Provider credentials are represented only through namespaced Secret name and
key references.

Plaintext credential values are not accepted in custom resources.

## Image Decision

The API separates the image repository from the image tag or digest.

Exactly one tag or digest must be provided.

This supports readable versioned deployments while allowing stronger
digest-pinned deployments.

## Kubernetes Configuration Decision

The API does not embed:

- `corev1.PodSpec`
- `appsv1.DeploymentSpec`
- Arbitrary container definitions
- Arbitrary environment variables
- Arbitrary volumes

Additional Kubernetes configuration will be introduced only where it represents
a stable and supportable Trussium operating requirement.

## Defaulting Decision

The Kubernetes API schema supplies defaults for:

- Replica count
- Image pull policy
- Service type
- Service port

Runtime execution timeout defaults remain owned by the selected runtime
release. The operator API only provides optional overrides.

## Controller Decision

This milestone creates the resource and generated CRD without scaffolding a
controller.

Reconciliation is intentionally delayed until the desired-state contract has
been reviewed and merged.

## Consequences

### Positive

- Clear public API boundary
- Kubernetes-native declarative configuration
- Secret-safe provider configuration
- Generated OpenAPI validation
- GitOps-compatible resource
- Independent API review before controller implementation

### Negative

- The v1alpha1 API may require conversion work when future versions are added.
- Provider types are initially limited to known runtime implementations.
- Advanced Pod customization is intentionally unavailable.
- Runtime and operator release compatibility must be documented separately.

## Follow-Up

The next milestone will implement core reconciliation for:

- ConfigMap
- ServiceAccount
- Service
- Deployment

Status updates and production-hardening resources will follow separately.
