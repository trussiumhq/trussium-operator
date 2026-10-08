# Operator and Runtime Compatibility

The Trussium Operator and Trussium runtime are independently maintained
components with an explicit integration boundary.

The operator manages Kubernetes lifecycle.

The runtime repository remains authoritative for runtime process behaviour,
provider integrations, runtime APIs, runtime configuration semantics, health
endpoints, and runtime container releases.

## Canonical Runtime Image

The canonical public runtime image repository is:

    ghcr.io/trussiumhq/trussium

The operator also permits another compatible image repository to be supplied
through `spec.image.repository`.

## Compatibility Dimensions

Compatibility involves:

- Trussium Operator version
- Trussium runtime release or immutable image identity
- Kubernetes API compatibility
- Runtime configuration contract compatibility
- Runtime health endpoint compatibility
- Container security and execution contract compatibility

## Released Compatibility Matrix

The machine-readable source of truth is
[`compatibility.yaml`](compatibility.yaml). It records the currently validated
operator, runtime image, chart, Kubernetes, upgrade, and rollback combinations.
The following is the current release snapshot:

| Operator release / chart | Runtime version | Runtime chart | Kubernetes | Status |
|---|---|---|---|---|
| v1.0.0 | v1.0.0 | v1.0.0 | >=1.25 | Tested |
| v1.0.0 | v1.17.0 | v1.1.0 | >=1.25 | Tested |
| v1.0.2 | v1.22.0 | v1.3.0 | >=1.25 | Tested |
| v1.0.2 | v1.27.0 | v1.3.1 | >=1.25 | Tested |
| v1.0.5 | v1.27.0, v1.29.1 | v1.3.1 | >=1.25 | Tested |

`Tested` means the operator lifecycle is validated against Kind and the
documented runtime integration contract. It is not a promise that every
arbitrary runtime tag is compatible.

Runtime Helm chart `v1.3.1` is the latest published chart and its default
`appVersion` remains `1.27.0`. The `v1.29.1` row uses an explicit
`TrussiumRuntime.spec.image.tag` override; it does not claim that the Runtime
Helm chart default changed. The Operator chart tested by this row is `v1.0.5`;
it includes the missing HorizontalPodAutoscaler and NetworkPolicy permissions
required by the controller.

The `1.0.0` validation uses the stable runtime image and chart contract with
an explicit runtime image override for lifecycle testing.

The `v1.17.0` / `v1.1.0` row is validated by the real Operator E2E lifecycle;
the runtime workload is reconciled, reaches its health contract, and survives
the tested image upgrade path in Kind. The `v1.22.0` row is validated by the
real Operator E2E lifecycle upgrade. The `v1.27.0` baseline is checked in the
Helm Chart CI runtime-compatibility job before and after the runtime image
transition.

The `v1.29.1` row is validated by that Kind job by upgrading the published
Operator chart `v1.0.4` to the current chart from the checkout. The job builds
the current operator image from that checkout and loads it into Kind, so the
upgrade exercises the candidate controller rather than reusing the old
`1.0.4` controller image. The chart corrects the published chart's missing
HorizontalPodAutoscaler and NetworkPolicy permissions. The job reconciles a
runtime on image `1.27.0`, verifies the updated service account permissions,
restarts the controller so its caches initialize with those permissions, rolls
forward to `1.29.1`, and waits for the operator's
Ready condition and successful-image status, then rolls the runtime image back
to `1.27.0` and verifies the restored workload. It does not roll the operator
chart back to the known-incomplete `v1.0.4` chart.
The Runtime Helm chart `v1.3.1` remains the referenced chart release and
defaults to runtime `1.27.0`; the tested `1.29.1` image is an explicit
operator-managed override.

## Runtime Contract Expected by the Operator

The operator currently expects compatible runtime images to support:

- The configured runtime HTTP port
- `/health/live`
- `/health/ready`
- Runtime environment-variable configuration
- Graceful process shutdown
- Numeric non-root execution compatible with UID/GID `10001`
- Read-only root filesystem operation

Runtime configuration semantics remain defined by the Trussium runtime
repository.

## Image Identity

Upgrade lifecycle tracking operates on the complete rendered image reference.

Supported examples include:

    ghcr.io/trussiumhq/trussium:1.0.0

and:

    ghcr.io/trussiumhq/trussium@sha256:<digest>

The operator does not currently interpret image tags as semantic versions.

A digest is treated as an immutable image identity.

## Enforcement

The operator uses an advisory-first compatibility policy. It documents tested
combinations and preserves support for compatible private mirrors, custom
repositories, tags, and digests. It does not reject a runtime image based on:

- Tag format
- Semantic version
- Digest
- Registry
- Runtime release metadata

This avoids inventing compatibility guarantees before independent operator
releases establish a stable versioning contract.

## Future Strict Mode

Strict compatibility enforcement is intentionally deferred. If introduced, it
will be an explicit opt-in setting backed by a mature, maintained compatibility
matrix. It must never silently change the current permissive default or block
compatible private runtime builds.

Until then, operators should use the released compatibility matrix as advisory
guidance and observe `status.conditions` during image rollouts.

## Upgrade Guidance

Before changing a production runtime image:

1. Review the operator release notes.
2. Review the runtime release notes.
3. Confirm the combination is documented as supported or tested.
4. Use immutable image digests where reproducibility is required.
5. Observe `status.conditions` and `status.lastSuccessfulImage` during rollout.

The operator does not automatically roll back an incompatible or failed
runtime image.
