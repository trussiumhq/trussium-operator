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
    repository: ghcr.io/trussiumhq/trussium
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

### `podMetadata`

Optional additional metadata applied to the runtime Pod template.

Example:

    podMetadata:
      labels:
        team: ai-platform
      annotations:
        example.com/owner: inference

Supported fields:

| Field | Type | Required | Description |
|---|---|---|---|
| `labels` | map[string]string | No | Additional Pod labels |
| `annotations` | map[string]string | No | Additional Pod annotations |

Operator-owned labels cannot be overridden.

These reserved labels maintain workload identity, selectors, and ownership.

### `scheduling`

Optional Kubernetes scheduling controls for runtime Pods.

Example:

    scheduling:
      nodeSelector:
        workload: ai

      tolerations:
        - key: workload
          operator: Equal
          value: ai
          effect: NoSchedule

Supported fields:

| Field | Type | Required | Description |
|---|---|---|---|
| `nodeSelector` | map[string]string | No | Required node labels |
| `tolerations` | Kubernetes toleration list | No | Supported taint tolerations |
| `affinity` | Kubernetes Affinity | No | Node, Pod affinity and anti-affinity |

The operator's default topology-spread configuration remains active regardless
of these settings.

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
  repository: ghcr.io/trussiumhq/trussium
  tag: v0.10.0
```

Digest example:

```yaml
image:
  repository: ghcr.io/trussiumhq/trussium
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
  currentImage: ghcr.io/trussiumhq/trussium:v0.10.0
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

### `status.desiredImage`

Fully rendered image requested by the current `spec.image`.

Example:

    ghcr.io/trussiumhq/trussium:v0.24.0

### `status.currentImage`

Image currently configured on the managed Deployment.

### `status.lastSuccessfulImage`

Image whose Deployment rollout most recently completed successfully.

This value remains unchanged while an upgrade is progressing or after an
upgrade failure.

### `Upgrading` Condition

Upgrade lifecycle is represented through:

    type: Upgrading

Reasons:

| Reason | Status | Meaning |
|---|---|---|
| `NoUpgrade` | `False` | No image upgrade is active |
| `UpgradeInProgress` | `True` | A previously successful image is transitioning |
| `UpgradeComplete` | `False` | The desired image completed successfully |
| `UpgradeFailed` | `False` | The requested image rollout failed |

Initial provisioning is not classified as an upgrade.

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

## PodDisruptionBudget Reconciliation

Each `TrussiumRuntime` owns one `policy/v1` PodDisruptionBudget with the same
name and namespace as the runtime resource.

The controller manages:

- Stable labels
- Runtime selector
- `maxUnavailable: 1`
- Controller owner reference

Drift is corrected through normal create-or-update reconciliation.

Deletion of the PodDisruptionBudget triggers reconciliation through the owned
resource watch and the resource is recreated while the `TrussiumRuntime`
continues to exist.

The PodDisruptionBudget is deleted through Kubernetes owner-reference garbage
collection when the `TrussiumRuntime` is deleted.
