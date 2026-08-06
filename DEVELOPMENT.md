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

The project will eventually use:

- Standard Go tests for pure resource builders
- envtest for controller integration tests
- Kind for real-cluster end-to-end tests

## Complete Validation

```bash
make generate
make manifests
make fmt
make vet
make test
make lint
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

The foundation release has no Trussium custom resource or controller. The
manager therefore starts without a Trussium reconciliation loop.

## Build the Manager

```bash
go build ./cmd/main.go
```

## Build the Operator Container

```bash
make docker-build IMG=trussium-operator:dev
```

Image publication will be introduced in a later milestone.

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
