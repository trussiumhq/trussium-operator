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

- Change class: Patch/security, Compatible feature, Contract change, or Breaking change
- Operator compatibility impact:
- Trussium runtime compatibility impact:
- Runtime Helm chart compatibility impact:
- Kubernetes compatibility impact:
- CRD/API compatibility impact:
- Upgrade and rollback impact:

## Security

Describe any effect on:

- RBAC
- Secrets
- Container security
- Cross-namespace access
- Image selection
- Supply chain

Write `No security impact` where applicable.

## Change management

- [ ] `docs/compatibility.yaml` is updated when release compatibility changes.
- [ ] Rollout success conditions and rollback procedure are documented.
- [ ] Runtime and runtime Helm chart changes have linked coordination issues or PRs.
- [ ] Documentation and roadmap are updated.

## Checklist

- [ ] The change is scoped to the linked issue.
- [ ] Tests cover the new behaviour where applicable.
- [ ] Generated artifacts are committed.
- [ ] Public API changes are documented.
- [ ] No credentials or sensitive information are included.
- [ ] The pull request title follows Conventional Commits.
