# Operator Upgrade Compatibility Matrix

This matrix is published with each operator release and records the chart
versions exercised by the release-to-release upgrade workflow.

| Target operator release | Previous chart versions tested | Rollback tested |
| --- | --- | --- |
| v0.13.1 / current release | v0.3.1, v0.5.0, v0.7.0, v0.9.0 | Yes |

Each entry installs the published previous chart from the Trussium OCI
registry, creates a `TrussiumRuntime`, upgrades to the current chart, verifies
the custom resource and controller rollout, then rolls back to the previous
Helm revision and verifies the rollout again.

This is an advisory compatibility record, not a guarantee for unlisted chart
versions. The supported test points are expanded as release families change.
