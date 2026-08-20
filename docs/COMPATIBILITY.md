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

The following combinations are validated by the operator's release and Kind
test suites:

| Operator version | Kubernetes | Runtime image contract | Status |
|---|---|---|---|
| v0.3.1 | >=1.25 | documented v0.x Trussium runtime contract | Tested |

`Tested` means the operator lifecycle is validated against Kind and the
documented runtime integration contract. It is not a promise that every
arbitrary runtime tag is compatible.

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

    ghcr.io/trussiumhq/trussium:v0.24.0

and:

    ghcr.io/trussiumhq/trussium@sha256:<digest>

The operator does not currently interpret image tags as semantic versions.

A digest is treated as an immutable image identity.

## Enforcement

The operator currently documents compatibility but does not reject a runtime
image based on:

- Tag format
- Semantic version
- Digest
- Registry
- Runtime release metadata

This avoids inventing compatibility guarantees before independent operator
releases establish a stable versioning contract.

## Upgrade Guidance

Before changing a production runtime image:

1. Review the operator release notes.
2. Review the runtime release notes.
3. Confirm the combination is documented as supported or tested.
4. Use immutable image digests where reproducibility is required.
5. Observe `status.conditions` and `status.lastSuccessfulImage` during rollout.

The operator does not automatically roll back an incompatible or failed
runtime image.

## Open-Source Boundary

The public operator will continue to support ordinary runtime upgrades and
compatibility documentation.

Commercial Trussium products may later provide higher-level capabilities such
as:

- Fleet-wide compatibility reporting
- Upgrade waves
- Maintenance windows
- Approval workflows
- Cross-cluster rollout policy
- Enterprise release channels

The public operator remains independently usable without those services.
