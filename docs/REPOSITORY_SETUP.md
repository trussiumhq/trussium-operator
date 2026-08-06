# Repository Setup

This document records the recommended GitHub settings for the
`trussium-operator` repository.

## General Settings

Enable:

- Issues
- Discussions
- Squash merging
- Automatically delete head branches
- Allow pull request branches to be updated

Disable:

- Merge commits
- Rebase merging
- Wiki

Use the pull request title and description for squash commit messages.

## Default Branch

The default branch is:

```text
main
```

Direct feature development on `main` is not permitted.

## Branch Ruleset

After the first CI workflow has run, create a branch ruleset:

```text
Settings
  → Rules
  → Rulesets
  → New ruleset
  → New branch ruleset
```

Configure:

```text
Name: Protect main
Enforcement: Active
Target: Default branch
```

Enable:

- Restrict deletions
- Block force pushes
- Require a pull request before merging
- Require status checks to pass
- Require branches to be up to date before merging
- Require conversation resolution before merging
- Require linear history

Required status check:

```text
Quality
```

For an early solo-maintained repository, zero mandatory external approvals is
acceptable. Increase this to one approval when another maintainer joins.

## Merge Strategy

Use squash merging.

Pull request titles must follow Conventional Commits:

```text
feat(api): add TrussiumRuntime v1alpha1 contract
fix(controller): preserve runtime status during retries
docs: document provider Secret references
chore: update controller-runtime dependencies
```

## Branch Naming

Use:

```text
feature/<description>
fix/<description>
docs/<description>
chore/<description>
refactor/<description>
test/<description>
```

Examples:

```text
chore/operator-project-foundation
feature/trussium-runtime-api
feature/configmap-reconciliation
test/controller-envtest
```

## Security Settings

Enable where available:

- Dependabot alerts
- Dependabot security updates
- Secret scanning
- Secret scanning push protection
- Private vulnerability reporting
- CodeQL for Go

CodeQL can be introduced in a dedicated security-automation issue after the
project foundation is merged.
