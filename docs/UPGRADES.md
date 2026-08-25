# Runtime Upgrade Lifecycle

## Operator Upgrade and Rollback

Upgrade the operator independently from managed runtime images. The operator
upgrade does not modify existing `TrussiumRuntime.spec.image` values.

For the released manifest bundle, apply the target version:

```bash
kubectl apply -f https://github.com/trussiumhq/trussium-operator/releases/download/v0.3.1/install.yaml
kubectl rollout status deployment/trussium-operator-controller-manager -n trussium-operator-system
```

For Helm installations, upgrade to an explicit chart version:

```bash
helm upgrade trussium-operator oci://ghcr.io/trussiumhq/charts/trussium-operator \
  --version 0.3.1 --namespace trussium-operator-system
```

CI continuously validates release-to-release upgrades from representative
published chart versions (`v0.3.1`, `v0.5.0`, `v0.7.0`, and `v0.9.0`) to the
current chart. Each matrix entry creates a runtime, upgrades the operator,
checks the managed resource and controller rollout, then rolls back to the
previous chart revision and checks the rollout again. The matrix is updated
as supported release families change.

Each release also publishes the tested matrix as the
`UPGRADE-MATRIX.md` release asset.

Before upgrading, review the target release notes and back up custom resources.
Afterwards, confirm the controller Deployment is available and observe existing
`TrussiumRuntime` conditions. To roll back, apply or upgrade to the previously
known-good operator version; do not delete CRDs as part of a rollback.

The Trussium Operator provides an explicit lifecycle for runtime image
transitions managed through `TrussiumRuntime`.

The operator observes Kubernetes Deployment rollout state rather than directly
reading Pods or ReplicaSets.

## Image State

`TrussiumRuntime.status` distinguishes three image concepts.

### Desired Image

`status.desiredImage` is the fully rendered image requested by the current
custom-resource specification.

For example:

    ghcr.io/trussiumhq/trussium:v0.24.0

or:

    ghcr.io/trussiumhq/trussium@sha256:<digest>

### Current Image

`status.currentImage` is the image currently configured on the managed
Deployment.

This preserves the existing status contract established before upgrade
lifecycle support was introduced.

### Last Successful Image

`status.lastSuccessfulImage` records the image whose Deployment rollout most
recently completed successfully.

During an upgrade, the desired and configured Deployment image may already be
the new image while `lastSuccessfulImage` continues to identify the previous
successful runtime.

Example:

    desiredImage: ghcr.io/trussiumhq/trussium:v0.24.0
    currentImage: ghcr.io/trussiumhq/trussium:v0.24.0
    lastSuccessfulImage: ghcr.io/trussiumhq/trussium:v0.23.0

After successful rollout completion:

    desiredImage: ghcr.io/trussiumhq/trussium:v0.24.0
    currentImage: ghcr.io/trussiumhq/trussium:v0.24.0
    lastSuccessfulImage: ghcr.io/trussiumhq/trussium:v0.24.0

## Initial Deployment

The first successful runtime rollout establishes `lastSuccessfulImage`.

Initial deployment is not classified as an upgrade.

Its upgrade condition is:

    type: Upgrading
    status: "False"
    reason: NoUpgrade

This prevents initial provisioning from being confused with a transition from
one previously successful runtime image to another.

## Upgrade Detection

An image transition is considered an upgrade when:

1. `lastSuccessfulImage` is non-empty.
2. The currently desired image differs from `lastSuccessfulImage`.

A Deployment rollout caused only by configuration, Pod metadata, scheduling, or
another Pod-template change is not classified as a runtime image upgrade when
the image remains unchanged.

Both tagged and digest-based image references participate in the same upgrade
lifecycle.

## Upgrading Condition

The operator maintains one upgrade-specific condition:

    Upgrading

### NoUpgrade

    type: Upgrading
    status: "False"
    reason: NoUpgrade

No runtime image transition is active.

### UpgradeInProgress

    type: Upgrading
    status: "True"
    reason: UpgradeInProgress

A previously successful runtime image is being replaced by another desired
image.

### UpgradeComplete

    type: Upgrading
    status: "False"
    reason: UpgradeComplete

The new image successfully completed its Deployment rollout.

The completed state is preserved during unchanged reconciliation.

### UpgradeFailed

    type: Upgrading
    status: "False"
    reason: UpgradeFailed

The Deployment reported rollout failure while transitioning to the desired
image.

The previous `lastSuccessfulImage` remains unchanged.

## Deployment Rollout Observation

The operator derives rollout state from the managed Deployment.

It observes:

- Deployment generation
- Deployment observed generation
- Desired replicas
- Total replicas
- Updated replicas
- Ready replicas
- Available replicas
- Unavailable replicas
- Deployment Progressing condition
- Deployment ReplicaFailure condition

The operator does not read Pods or ReplicaSets.

## Rollout Completion

For a runtime with replicas greater than zero, successful rollout requires:

- The Deployment controller has observed the current Deployment generation.
- Updated replicas equal desired replicas.
- Total replicas equal desired replicas.
- Ready replicas equal desired replicas.
- Available replicas equal desired replicas.
- Unavailable replicas equal zero.

This prevents the operator from reporting upgrade completion while old
Deployment replicas are still present.

## Scale to Zero

A runtime with:

    spec.replicas: 0

can still complete an image upgrade.

The Deployment must have observed its latest generation and all replica counts
must be zero.

No Pod needs to be started merely to establish the desired image state.

## Rollout Failure

The operator recognizes Deployment rollout failures including:

### Progress Deadline Exceeded

    type: Progressing
    status: "False"
    reason: ProgressDeadlineExceeded

### Replica Failure

    type: ReplicaFailure
    status: "True"

During an image upgrade, failure results in:

    Upgrading=False
    reason=UpgradeFailed

General runtime conditions also reflect the failure:

    Ready=False
    Degraded=True

The requested image remains unchanged and `lastSuccessfulImage` remains the
last successfully rolled-out image.

## No Automatic Rollback

The operator does not automatically change `TrussiumRuntimeSpec` or restore a
previous image after rollout failure.

Rollback remains an explicit user or automation decision.

This preserves the declarative relationship between the custom-resource
specification and the managed Deployment.

## Progress Deadline

Managed Deployments use:

    progressDeadlineSeconds: 600

This gives Kubernetes an explicit boundary for reporting stalled Deployment
progress.

The value is operator-managed in the current API version.

## Configuration-Triggered Rollouts

Runtime configuration is projected through a managed ConfigMap and consumed by
the runtime container using `envFrom`.

Kubernetes does not create a new Deployment revision when only the referenced
ConfigMap data changes.

The operator therefore adds this Pod-template annotation:

    runtime.trussium.io/config-checksum

The annotation contains a deterministic SHA-256 checksum of the fully rendered
operator-managed ConfigMap data.

When runtime configuration changes:

1. ConfigMap data changes.
2. The checksum changes.
3. The Deployment Pod template changes.
4. Kubernetes creates a new Deployment revision.
5. The rollout is observed through normal Deployment status.

## Checksum Security Boundary

The configuration checksum contains only operator-managed ConfigMap data.

It never contains or derives from:

- Secret values
- Secret contents
- Secret resource versions
- Timestamps
- Random values
- Kubernetes object resource versions

Provider credentials remain outside the checksum.

Changing the value stored inside an existing Secret therefore does not
automatically trigger a runtime rollout during this milestone.

Changing a Secret reference in the custom-resource specification changes the
Pod template normally.

## Upgrade Events

The operator emits transition-based Kubernetes Events.

### RuntimeUpgradeStarted

Normal Event emitted when an existing successfully deployed runtime begins
transitioning to another desired image.

### RuntimeUpgradeCompleted

Normal Event emitted when the new runtime image successfully completes its
Deployment rollout.

### RuntimeUpgradeFailed

Warning Event emitted when an active runtime image upgrade fails.

Upgrade Events include the previous successful image and desired image where
applicable.

They never include Secret values.

Initial runtime deployment does not emit an upgrade-start Event.

Unchanged reconciliation does not emit duplicate upgrade Events.

## Existing Runtime Conditions

Upgrade state complements rather than replaces the existing runtime conditions:

- `ConfigurationValid`
- `Progressing`
- `Available`
- `Ready`
- `Degraded`

`Upgrading` describes image-transition lifecycle.

The other conditions continue to describe overall runtime health and
availability.

## Reconciliation Model

Upgrade observation remains event-driven.

The operator watches the owned Deployment, so Deployment status changes enqueue
reconciliation.

No polling loop or periodic upgrade requeue is required.

Status-only changes on the `TrussiumRuntime` remain filtered from the primary
watch.

## RBAC Boundary

Upgrade lifecycle support adds no permissions for:

- Pods
- ReplicaSets
- Nodes
- Secret values

Existing Deployment access is sufficient for rollout observation.

## Compatibility

Runtime upgrade tracking does not parse or enforce semantic versions.

Tags and digests are treated as complete image identities.

Compatibility policy is documented separately in
[COMPATIBILITY.md](COMPATIBILITY.md).

## Current Limitations

This milestone does not provide:

- Automatic rollback
- Manual rollback API
- Upgrade waves
- Maintenance windows
- Multi-cluster rollout coordination
- Runtime registry queries
- Signature verification
- Semantic-version admission
- Secret-content-triggered rollouts
- Direct application health probing by the controller

## Release-to-Release Upgrade Validation

CI installs a published previous operator chart in Kind, creates a
`TrussiumRuntime`, upgrades to the checked-out chart, and confirms that the
custom resource and controller rollout remain available. This validates the
upgrade contract across released chart boundaries.
