# ADR 0007 — Observe Runtime Upgrades Through Deployment Status

- Status: Accepted
- Date: August 2026

## Context

The Trussium Operator reconciles runtime image changes through a Kubernetes
Deployment.

Before this decision, image changes were applied correctly but were not modeled
as an explicit runtime upgrade lifecycle.

Users need to distinguish:

- Initial runtime deployment
- Upgrade progress
- Upgrade completion
- Upgrade failure
- Last successfully deployed runtime image

The operator also needs to trigger Deployment rollouts when projected ConfigMap
configuration changes.

Several implementation strategies are possible:

- Read Pods directly
- Read ReplicaSets directly
- Poll runtime health endpoints
- Observe the managed Deployment
- Implement custom rollout machinery

## Decision

The operator will derive runtime upgrade state exclusively from the managed
Deployment and the `TrussiumRuntime` desired image.

No Pod or ReplicaSet reads are required.

## Image Status

The status contract contains:

- `desiredImage`
- `currentImage`
- `lastSuccessfulImage`

`desiredImage` represents the image requested by the current custom-resource
specification.

`currentImage` preserves its existing meaning as the image configured on the
managed Deployment.

`lastSuccessfulImage` records the image whose Deployment rollout most recently
completed successfully.

## Initial Deployment

Initial runtime provisioning is not an upgrade.

The first successful Deployment rollout establishes
`lastSuccessfulImage`.

## Upgrade Condition

The operator maintains one condition:

    Upgrading

Stable reasons include:

- `NoUpgrade`
- `UpgradeInProgress`
- `UpgradeComplete`
- `UpgradeFailed`

Existing runtime conditions continue to represent overall health and
availability.

## Upgrade Detection

An image change is classified as an upgrade when:

1. A previous successful image exists.
2. The desired image differs from that image.

Changes to configuration or other Pod-template settings do not constitute an
image upgrade when the desired image remains unchanged.

## Rollout Completion

A non-zero replica rollout completes only when:

- Deployment observed generation reaches the Deployment generation
- Updated replicas equal desired replicas
- Total replicas equal desired replicas
- Ready replicas equal desired replicas
- Available replicas equal desired replicas
- Unavailable replicas equal zero

A zero-replica rollout completes after the latest Deployment generation is
observed and all replica counts are zero.

## Rollout Failure

The operator recognizes Deployment failure states including:

- `ProgressDeadlineExceeded`
- `ReplicaFailure`

A failed upgrade does not modify the desired image.

The previous `lastSuccessfulImage` remains unchanged.

## Automatic Rollback

The operator will not automatically roll back runtime image changes.

The custom-resource specification remains authoritative.

Rollback is an explicit desired-state change made by a user or higher-level
automation.

## Deployment Progress Deadline

Managed Deployments use:

    progressDeadlineSeconds: 600

This provides a deterministic Kubernetes signal for stalled rollouts.

## Configuration Rollouts

The operator computes a deterministic SHA-256 checksum of the managed ConfigMap
data and places it on the Deployment Pod template using:

    runtime.trussium.io/config-checksum

This causes configuration changes to create a new Deployment revision.

Checksum generation:

- Sorts keys deterministically
- Includes operator-managed ConfigMap data
- Excludes Secret values
- Excludes timestamps
- Excludes Kubernetes resource versions
- Produces stable output for semantically identical configuration

## Events

The operator emits:

- `RuntimeUpgradeStarted`
- `RuntimeUpgradeCompleted`
- `RuntimeUpgradeFailed`

Events are transition-based and are not emitted repeatedly during unchanged
reconciliation.

## Compatibility

Upgrade tracking treats image references as identities.

It does not require semantic-version parsing.

Compatibility is documented separately and is not yet enforced through
admission or reconciliation.

## Consequences

### Positive

- No additional Pod RBAC
- No ReplicaSet RBAC
- Kubernetes-native rollout observation
- Explicit upgrade status
- Stable successful-image history
- Configuration-triggered rollouts
- No polling loop
- No hidden mutation of desired state

### Negative

- Deployment state is not equivalent to application-level semantic health.
- Secret-value changes do not trigger rollouts automatically.
- Failed upgrades require explicit operator or user action.
- Compatibility is documented rather than enforced.

## Alternatives Considered

### Read Pods Directly

Rejected because Deployment status already exposes the rollout information
required by this milestone and direct Pod access would broaden RBAC.

### Read ReplicaSets Directly

Rejected because updated and total replica state is already available through
Deployment status.

### Automatic Rollback

Rejected because silently mutating the desired image would violate the
declarative custom-resource contract and could hide upgrade failures.

### Periodic Polling

Rejected because owned Deployment status changes already trigger
reconciliation.

## Follow-Up

The next milestone will validate these contracts using envtest and real Kind
clusters.
