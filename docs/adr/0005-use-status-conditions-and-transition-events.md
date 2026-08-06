# ADR 0005 — Use Status Conditions and Transition Events

- Status: Accepted
- Date: August 2026

## Context

Core reconciliation creates the Kubernetes resources required by a
`TrussiumRuntime`, but desired resources alone do not communicate whether the
runtime has reached its requested state.

Users, automation, and GitOps systems need a stable machine-readable status
contract. Cluster operators also need concise human-readable lifecycle
information through `kubectl describe`.

Updating status during every reconciliation or emitting Events during every
loop would create API-server noise and could cause reconciliation loops.

## Decision

The operator will update the existing `TrussiumRuntime.status` subresource and
manage these conditions:

- `ConfigurationValid`
- `Progressing`
- `Available`
- `Ready`
- `Degraded`

Each condition includes:

- Status
- Stable reason
- Human-readable message
- Observed generation
- Last transition time

The controller will emit Kubernetes `events.k8s.io/v1` Events only for
meaningful condition transitions and reconciliation failures.

## Status Source

Deployment readiness will be derived from the managed Deployment:

- `metadata.generation`
- `status.observedGeneration`
- `status.readyReplicas`
- `status.availableReplicas`
- Deployment progress conditions
- Deployment replica-failure conditions

The operator will not inspect Pods directly during this milestone.

## Configuration Validation

The controller will verify that referenced provider and image-pull Secrets
exist in the same namespace as the `TrussiumRuntime`.

It will not read Secret values.

Missing references block Deployment reconciliation and are represented through
status and a Warning Event.

## Event Decision

Normal Events include:

- `RuntimeProgressing`
- `RuntimeReady`
- `RuntimeRecovered`
- `RuntimeScaledToZero`

Warning Events include:

- `ConfigurationInvalid`
- `ReconciliationFailed`
- `RuntimeDegraded`

Events are emitted only when a relevant condition status or reason changes.

## Status Write Decision

Before updating status, the controller will:

1. Preserve stored transition times where condition status is unchanged.
2. Compare stored and desired status semantically.
3. Skip status writes when no difference exists.

Primary custom-resource watches will use generation changes so status-only
updates do not enqueue another reconciliation.

Deployment status changes remain observable through the owned Deployment
watch.

## Service Endpoint Decision

The initial endpoint is the internal Kubernetes Service DNS address:

    http://<name>.<namespace>.svc.cluster.local:<port>

External LoadBalancer address discovery is deferred.

## Scale-to-Zero Decision

For a desired replica count of zero:

- `Ready=True` after the Deployment reaches zero ready and available replicas
- `Available=False`
- `Progressing=False`
- `Degraded=False`

This represents successful convergence rather than unavailability or failure.

## Consequences

### Positive

- Kubernetes-native machine-readable status
- Stable condition reasons for automation
- Human-readable transition Events
- No Pod permissions
- No Secret-value access
- Reduced status-write noise
- No status-update reconciliation loop
- Clear scale-to-zero semantics

### Negative

- Deployment readiness is not equivalent to application-level runtime health.
- External Service addresses are not reported.
- Secret existence does not prove credential correctness.
- Runtime provider connectivity is not validated.
- Events are best-effort and are not a durable audit log.

## Follow-Up

The next milestone will harden the managed runtime workload with:

- Startup probe
- Liveness probe
- Readiness probe
- Container security context
- Pod security context
- Graceful termination
- PodDisruptionBudget
- Topology spreading
- Zero-unavailable rolling updates
