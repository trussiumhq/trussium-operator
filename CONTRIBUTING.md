# Contributing to Trussium Operator

Thank you for contributing to the Trussium Operator.

## Before Starting

Search existing issues before opening a new one.

Substantial changes must begin with a GitHub issue defining:

- Motivation
- Scope
- Out-of-scope work
- Acceptance criteria
- Validation expectations

Security vulnerabilities must not be reported through public issues. Follow
[SECURITY.md](SECURITY.md).

## Development Workflow

1. Create or select a GitHub issue.
2. Update local `main`.
3. Create a focused branch.
4. Implement the smallest complete change.
5. Add or update tests.
6. Run all validation.
7. Update documentation and the roadmap where applicable.
8. Commit using Conventional Commits.
9. Push the branch.
10. Open a pull request.
11. Address checks and review comments.
12. Squash merge.
13. Delete the feature branch.

## Branches

Use one of:

```text
feature/<description>
fix/<description>
docs/<description>
chore/<description>
refactor/<description>
test/<description>
```

## Generated Artifacts

When API markers, RBAC markers, or generated types change, run:

```bash
make generate
make manifests
```

Generated artifacts must be committed with the source change that produced
them.

Do not manually edit generated deep-copy files or generated CRD manifests.

## Validation

Before opening a pull request:

```bash
make generate
make manifests
make fmt
make vet
make test
make lint
```

Then verify generation stability:

```bash
git add -A
make generate
make manifests
make fmt
git diff --exit-code
git diff --cached --check
```

## Pull Requests

A pull request must:

- Begin with `Closes #<issue-number>`.
- Explain the motivation and implementation.
- Include validation commands and results.
- Identify compatibility and security implications.
- Update documentation where public behaviour changes.
- Remain focused on one issue.

## Commit Messages

Use Conventional Commits.

Examples:

```text
feat(api): add TrussiumRuntime image contract
fix(controller): avoid unnecessary Deployment updates
test(api): validate image tag and digest exclusivity
docs: add operator upgrade guidance
chore: update Kubernetes dependencies
```

## API Changes

Custom-resource APIs are public contracts.

Changes must consider:

- Defaulting
- Validation
- Backward compatibility
- Upgrade behaviour
- Existing persisted resources
- Generated OpenAPI schemas
- Future conversion requirements

Breaking API changes require explicit discussion before implementation.

## Licence

By contributing, you agree that your contributions are licensed under the
Apache License 2.0.
