# Development Guide

## Prerequisites

Install:

- Go at the version declared by `go.mod`
- Docker
- kubectl
- Kubebuilder
- Git
- GitHub CLI

Project tools such as `controller-gen`, `kustomize`, `setup-envtest`, and
`golangci-lint` are installed through Makefile targets.

Use the project-pinned tools during validation rather than relying on unrelated
global installations.

## Clone

```bash
gh repo clone trussium/trussium-operator
cd trussium-operator
```

## Dependencies

```bash
go mod download
go mod verify
```

## Project Generation

Generate Go helper code:

```bash
make generate
```

Generate Kubernetes manifests:

```bash
make manifests
```

Generated code and manifests must be committed whenever their source markers or
API definitions change.

## Formatting

```bash
make fmt
```

## Static Analysis

```bash
make vet
make lint
```

## Tests

```bash
make test
```

The project uses:

- Standard Go tests for pure resource builders
- envtest for controller integration tests
- Kind for real-cluster end-to-end tests

Run the complete Kind suite with:

```bash
make test-e2e
```

This target creates an isolated Kind cluster, builds and side-loads the
operator image, runs the E2E suite, and removes the cluster afterward.

## Complete Validation

```bash
make generate
make manifests
make fmt
make vet
make test
make lint
make test-e2e
```

Before committing, stage the intended files and verify generation stability:

```bash
git add -A

make generate
make manifests
make fmt

git diff --exit-code
git diff --cached --check
```

`git diff --exit-code` confirms that regeneration produced no additional
unstaged changes after the intended files were staged.

## Run the Manager Locally

A Kubernetes cluster is required:

```bash
make run
```

The manager reconciles `TrussiumRuntime` resources against the connected
Kubernetes cluster.

## Build the Manager

```bash
go build ./cmd/main.go
```

## Build the Operator Container

```bash
make docker-build IMG=trussium-operator:dev
```

Tagged releases publish multi-platform operator images to:

```text
ghcr.io/trussiumhq/trussium-operator:<version>
```

Each GitHub release also includes `install.yaml`, a single manifest containing
the CRD and controller deployment. Generate the equivalent local bundle with:

```bash
make build-installer IMG=ghcr.io/trussiumhq/trussium-operator:v0.1.0
```

The target writes `dist/install.yaml` and leaves the tracked Kustomize files
unchanged.

## Release Versioning

The operator follows Semantic Versioning and uses Conventional Commits to
derive release versions. Merges to `main` run the semantic release workflow,
which updates `VERSION`, creates a `vMAJOR.MINOR.PATCH` Git tag and GitHub
release, and dispatches container publication for that tag.

Published images receive full, major/minor, major, and `latest` tags. Release
images include OCI source, revision, version, and build-date metadata, plus
SBOM and provenance attestations.

## Workflow

Every change follows:

```text
Issue
  → fresh branch from main
  → implementation
  → validation
  → roadmap and documentation update
  → Conventional Commit
  → push
  → pull request
  → squash merge
  → branch cleanup
```

Every pull request description must begin with:

```text
Closes #<issue-number>
```

That line must appear before `## Summary`.

## Commit Conventions

Examples:

```text
chore: bootstrap operator project foundation
feat(api): add TrussiumRuntime v1alpha1 contract
test(controller): add ConfigMap reconciliation coverage
fix(status): preserve observed generation during retries
docs: document runtime image compatibility
```
