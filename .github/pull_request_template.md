Closes #

## Summary

Describe the completed change.

## Motivation

Explain the problem and why this change is needed.

## Changes

- 
- 
- 

## Validation

```bash
make generate
make manifests
make fmt
make vet
make test
make lint
```

Results:

- [ ] Generation passed
- [ ] Formatting passed
- [ ] Vet passed
- [ ] Tests passed
- [ ] Lint passed
- [ ] Generated files are stable

## Documentation

Describe documentation and roadmap updates.

## Compatibility

- Operator compatibility impact:
- Trussium runtime compatibility impact:
- Kubernetes compatibility impact:
- CRD/API compatibility impact:

## Security

Describe any effect on:

- RBAC
- Secrets
- Container security
- Cross-namespace access
- Image selection
- Supply chain

Write `No security impact` where applicable.

## Checklist

- [ ] The change is scoped to the linked issue.
- [ ] Tests cover the new behaviour where applicable.
- [ ] Generated artifacts are committed.
- [ ] Public API changes are documented.
- [ ] No credentials or sensitive information are included.
- [ ] The pull request title follows Conventional Commits.
