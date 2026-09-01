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

| Operator version | Runtime version | Runtime chart | Kubernetes | Status |
|---|---|---|---|---|
| v1.0.0 | v1.0.0 | v1.0.0 | >=1.25 | Tested |
| v1.0.0 | v1.17.0 | v1.1.0 | >=1.25 | Proposed |

`Tested` means the operator lifecycle is validated against Kind and the
documented runtime integration contract. It is not a promise that every
arbitrary runtime tag is compatible.

The `1.0.0` validation uses the stable runtime image and chart contract with
an explicit runtime image override for lifecycle testing.

The `v1.17.0` / `v1.1.0` row is a compatibility proposal. It must not be
treated as tested until the runtime workload reaches its health contract in a
real Operator lifecycle test.

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
