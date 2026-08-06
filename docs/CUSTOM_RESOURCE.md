# TrussiumRuntime Custom Resource

## API Identity

```text
Group:   runtime.trussium.io
Version: v1alpha1
Kind:    TrussiumRuntime
Plural:  trussiumruntimes
Scope:   Namespaced
```

`TrussiumRuntime` declares the desired configuration of one Trussium runtime
instance.

The v1alpha1 API is experimental. Backward compatibility will be considered
carefully, but changes may occur before a stable API version is released.

## Minimal Example

```yaml
apiVersion: runtime.trussium.io/v1alpha1
kind: TrussiumRuntime
metadata:
  name: private-ai
  namespace: trussium
spec:
  image:
    repository: ghcr.io/trussium/trussium
    tag: v0.1.0

  provider:
    type: ollama
    model: llama3.2
    baseURL: http://ollama.ollama.svc.cluster.local:11434/v1
```

The Kubernetes API server applies these defaults:

```yaml
spec:
  replicas: 1

  image:
    pullPolicy: IfNotPresent

  service:
    type: ClusterIP
    port: 9000
```

## Specification

### `spec.image`

Selects the released Trussium runtime container.

| Field | Required | Description |
|---|---:|---|
| `repository` | Yes | Image repository without tag or digest |
| `tag` | Conditional | Versioned image tag |
| `digest` | Conditional | SHA-256 image digest |
| `pullPolicy` | No | Kubernetes image pull policy |

Exactly one of `tag` or `digest` must be supplied.

Tag example:

```yaml
image:
  repository: ghcr.io/trussium/trussium
  tag: v0.10.0
```

Digest example:

```yaml
image:
  repository: ghcr.io/trussium/trussium
  digest: sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
```

Production deployments should prefer immutable release tags or digests.

### `spec.replicas`

Defines the desired runtime pod count.

```yaml
replicas: 2
```

The default is `1`.

A replica count of `0` is valid and represents an intentionally scaled-down
runtime.

### `spec.imagePullSecrets`

References image registry Secrets in the same namespace:

```yaml
imagePullSecrets:
  - name: ghcr-credentials
```

The operator never stores registry credentials in the custom resource.

### `spec.provider`

Selects the provider and model.

| Field | Required | Description |
|---|---:|---|
| `type` | Yes | `openai` or `ollama` |
| `model` | Yes | Provider model identifier |
| `baseURL` | No | Provider or compatible API endpoint |
| `credentialsSecretRef` | No | Secret name and key containing credentials |

OpenAI example:

```yaml
provider:
  type: openai
  model: gpt-5
  credentialsSecretRef:
    name: openai-credentials
    key: api-key
```

Ollama example:

```yaml
provider:
  type: ollama
  model: llama3.2
  baseURL: http://ollama.ollama.svc.cluster.local:11434/v1
```

Credential values must never be embedded directly:

```yaml
# Invalid design — never supported
provider:
  apiKey: secret-value
```

### `spec.runtime`

Defines optional overrides for runtime-owned defaults:

```yaml
runtime:
  providerRequestTimeoutSeconds: 60
  streamIdleTimeoutSeconds: 30
  shutdownDrainTimeoutSeconds: 45
```

Omitting a field leaves its value under the control of the selected Trussium
runtime release.

### `spec.service`

Defines the Kubernetes Service:

```yaml
service:
  type: ClusterIP
  port: 9000
```

Supported Service types are:

- `ClusterIP`
- `NodePort`
- `LoadBalancer`

`ExternalName` is not supported because TrussiumRuntime Services select
operator-managed runtime pods.

### `spec.resources`

Uses the Kubernetes resource-requirements contract:

```yaml
resources:
  requests:
    cpu: 250m
    memory: 256Mi
  limits:
    cpu: "1"
    memory: 1Gi
```

## Status

The controller will populate status in a later milestone:

```yaml
status:
  observedGeneration: 3
  readyReplicas: 2
  availableReplicas: 2
  currentImage: ghcr.io/trussium/trussium:v0.10.0
  endpoint: http://private-ai.trussium.svc.cluster.local:9000
  conditions:
    - type: Ready
      status: "True"
      reason: RuntimeAvailable
      message: All requested runtime replicas are ready
```

The v1alpha1 status contract contains:

| Field | Description |
|---|---|
| `observedGeneration` | Most recent generation processed |
| `readyReplicas` | Ready runtime replicas |
| `availableReplicas` | Available runtime replicas |
| `currentImage` | Complete image observed by the controller |
| `endpoint` | Runtime Service endpoint |
| `conditions` | Standard Kubernetes conditions |

## Printer Columns

A future controller-populated resource will be displayed as:

```text
NAME         READY   DESIRED   PROVIDER   MODEL      VERSION   AGE
private-ai   2       2         ollama     llama3.2   v0.10.0   10m
```

## Security

- Credential values are never custom-resource fields.
- Secret references are namespaced.
- Secret contents must never be written to status or Kubernetes Events.
- An image tag or digest must be explicitly selected.
- The API does not expose arbitrary Pod or Deployment specifications.
- Cross-namespace Secret references are not supported.

## Reconciliation

This milestone defines only the API contract.

ConfigMap, ServiceAccount, Service, Deployment, and status reconciliation will
be introduced through separate issues.
