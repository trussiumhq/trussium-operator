# Security Policy

## Reporting Vulnerabilities

Do not report suspected vulnerabilities through public GitHub issues,
Discussions, or pull requests.

Use GitHub private vulnerability reporting:

```text
Repository
  → Security
  → Advisories
  → Report a vulnerability
```

Include:

- Affected component
- Affected version or commit
- Reproduction steps
- Security impact
- Suggested mitigation where known
- Whether the issue is already public

Do not include real provider credentials, customer data, prompts, or other
sensitive information.

## Response Process

Maintainers will:

1. Acknowledge the report.
2. Assess severity and affected versions.
3. Reproduce the issue where possible.
4. Prepare a fix and regression coverage.
5. Coordinate disclosure and release timing.
6. Publish remediation guidance.

Response times are best effort while the project is pre-release and
community-maintained.

## Supported Versions

| Version | Security support |
|---|---|
| `main` | Yes |
| Pre-release tags | Best effort |
| Stable releases | None published yet |

This table will be updated when the first operator release is published.

## Security Scope

Security-sensitive areas include:

- Kubernetes RBAC
- Secret references
- Provider credential projection
- Container security contexts
- Admission and validation logic
- Cross-namespace access
- Owner references and deletion behaviour
- Image selection
- Supply-chain metadata
- Runtime configuration
- Operator metrics and debug endpoints

## Secret Handling

The operator must not:

- Accept plaintext provider credentials in custom resources
- Log Secret contents
- Store credentials in status
- Copy credentials into Kubernetes Events
- Expose credentials through metrics
- Include credentials in reconciliation errors

Secret values belong in Kubernetes Secret objects or supported external
secret-management systems.
