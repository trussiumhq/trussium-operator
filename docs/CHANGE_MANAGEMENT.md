# Change Management

Trussium is released through three independently versioned repositories:

- `trussium` — runtime behaviour and runtime image releases
- `trussium-helm` — the runtime Helm distribution
- `trussium-operator` — Kubernetes lifecycle and custom-resource behaviour

This repository uses a lightweight change-management process to keep those
contracts aligned without requiring a separate change-management system.

## Change classes

Every non-trivial issue and pull request declares one change class:

- **Patch/security** — documentation, tests, dependency, or security fixes
  without an intended public behaviour change.
- **Compatible feature** — additive API, chart, or controller behaviour that
  preserves existing valid resources and defaults.
- **Contract change** — a change to runtime environment variables, health,
  security defaults, packaging, or release compatibility.
- **Breaking change** — removal or incompatible alteration of a public API,
  chart value, managed-resource contract, or supported upgrade path.

If multiple classes apply, use the highest-impact class.

## Required impact record

Issues and pull requests must record impact on:

- Runtime repository and image contract
- Runtime Helm chart
- Operator API and controller behaviour
- Kubernetes version support
- Upgrade and rollback paths
- Security, RBAC, and Secret handling
- Documentation and roadmap

An explicit `No impact` statement is required when a dimension is unaffected.

## Release gates

Before a compatible or contract change is released:

1. The compatibility manifest is updated.
2. Relevant runtime, chart, and operator tests pass.
3. Upgrade and rollback behavior is validated when managed resources change.
4. Public documentation and release notes are updated.
5. A maintainer reviews the compatibility impact.

Breaking changes additionally require a migration and deprecation plan before
implementation.

## Rollout and rollback

Every release-affecting change documents its rollout success conditions, the
previous known-good versions, the explicit rollback procedure, and any
migration that rollback cannot reverse.

The operator does not silently rewrite a user's runtime image or CRD during
rollback.

## Source of truth

[`compatibility.yaml`](compatibility.yaml) is the machine-readable record of
tested release combinations. `COMPATIBILITY.md` and the published
`UPGRADE-MATRIX.md` asset must remain consistent with it.

Runtime behavior remains authoritative in the runtime repository. Chart
packaging and chart defaults remain authoritative in the runtime Helm
repository.
