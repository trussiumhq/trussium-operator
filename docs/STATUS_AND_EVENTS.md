# Runtime Status and Kubernetes Events

The Trussium Operator reports observed runtime state through the
`TrussiumRuntime.status` subresource and Kubernetes Events.

Status provides the current machine-readable state. Events provide
human-readable lifecycle transitions and failures for operators using
`kubectl describe`.

## Status Fields

The controller maintains:

| Field | Description |
|---|---|
| `observedGeneration` | Latest `TrussiumRuntime` generation processed |
| `readyReplicas` | Ready replicas reported by the managed Deployment |
| `availableReplicas` | Available replicas reported by the Deployment |
| `currentImage` | Runtime image observed in the managed Deployment |
| `endpoint` | Internal Kubernetes Service endpoint |
| `conditions` | Kubernetes-native runtime conditions |

Example:

    status:
      observedGeneration: 4
      readyReplicas: 2
      availableReplicas: 2
      currentImage: ghcr.io/trussium/trussium:v0.23.0
      endpoint: http://private-ai.trussium.svc.cluster.local:9000
      conditions:
        - type: ConfigurationValid
          status: "True"
          reason: ReferencesResolved
          message: All referenced Secrets are available.
          observedGeneration: 4
        - type: Ready
          status: "True"
          reason: RuntimeReady
          message: All requested runtime replicas are ready and available.
          observedGeneration: 4

## Observed Generation

`status.observedGeneration` identifies the most recent custom-resource
generation processed by the controller.

Consumers should compare:

    metadata.generation

with:

    status.observedGeneration

When the values differ, the reported status may describe an earlier desired
configuration.

Each condition also records the generation it represents.

## Conditions

The controller maintains five conditions:

- `ConfigurationValid`
- `Progressing`
- `Available`
- `Ready`
- `Degraded`

Conditions use stable UpperCamelCase reasons and preserve
`lastTransitionTime` while their status remains unchanged.

A condition reason or message may change without resetting the transition
time when the underlying condition status remains the same.

## ConfigurationValid

`ConfigurationValid=True` when every referenced Secret exists in the
`TrussiumRuntime` namespace.

Successful reason:

- `ReferencesResolved`

Failure reasons:

- `SecretNotFound`
- `ReferenceCheckFailed`

The controller validates:

- `spec.provider.credentialsSecretRef`
- `spec.imagePullSecrets`

It checks only whether each Secret exists.

The controller never:

- Reads credential values
- Logs Secret contents
- Copies Secret values into status
- Copies Secret values into Events
- Validates provider credential values

## Progressing

`Progressing=True` while the managed Deployment is converging toward the
desired state.

Reasons include:

- `DeploymentProgressing`
- `DeploymentGenerationPending`

`Progressing=False` reasons include:

- `ReconciliationComplete`
- `ConfigurationInvalid`
- `DeploymentFailed`
- `ScaledToZero`
- `ReconciliationFailed`

## Available

`Available=True` when at least one requested runtime replica is available.

Reasons include:

- `RuntimeAvailable`
- `RuntimeUnavailable`
- `ScaledToZero`

For an intentionally scaled-to-zero runtime, `Available=False` is expected and
does not represent degradation.

## Ready

`Ready=True` when:

- Referenced Secrets are valid
- The Deployment has observed its current generation
- Ready replicas equal desired replicas
- Available replicas equal desired replicas

Reasons include:

- `RuntimeReady`
- `RuntimeNotReady`
- `ConfigurationInvalid`
- `DeploymentFailed`
- `ScaledToZero`
- `ReconciliationFailed`

For `spec.replicas: 0`, `Ready=True` means that the requested scaled-to-zero
state has been reached.

## Degraded

`Degraded=True` when the runtime cannot progress normally.

Reasons include:

- `ConfigurationInvalid`
- `ProgressDeadlineExceeded`
- `ReplicaFailure`
- `ReconciliationFailed`

`Degraded=False` uses:

- `AsExpected`

## Deployment Observation

The controller projects Deployment status rather than inspecting Pods
directly.

It observes:

- Deployment generation
- Deployment observed generation
- Ready replicas
- Available replicas
- Deployment progress conditions
- Deployment replica-failure conditions
- Runtime container image

The operator requires no Pod permissions for this milestone.

## Service Endpoint

The controller reports the internal Kubernetes endpoint:

    http://<runtime-name>.<namespace>.svc.cluster.local:<service-port>

Example:

    http://private-ai.trussium.svc.cluster.local:9000

The initial implementation does not discover external LoadBalancer addresses.

## Missing Secret Behaviour

When a referenced Secret is missing:

- `ConfigurationValid=False`
- `Progressing=False`
- `Ready=False`
- `Degraded=True`
- A Warning Event is emitted
- Deployment creation or update is blocked

The ConfigMap, ServiceAccount, Service, and Deployment are not reconciled until
the reference is resolved.

When the Secret is later created, the Secret watch enqueues every runtime in
the same namespace that references it.

## Kubernetes Events

The operator uses `events.k8s.io/v1`.

Events are emitted only for meaningful transitions and failures.

### Normal Events

| Reason | Meaning |
|---|---|
| `RuntimeProgressing` | The runtime is converging |
| `RuntimeReady` | All requested replicas are ready |
| `RuntimeRecovered` | A degraded runtime recovered |
| `RuntimeScaledToZero` | The requested zero-replica state was reached |

### Warning Events

| Reason | Meaning |
|---|---|
| `ConfigurationInvalid` | A referenced Secret is missing or cannot be checked |
| `ReconciliationFailed` | A Kubernetes reconciliation operation failed |
| `RuntimeDegraded` | Deployment or configuration state is degraded |

Events include:

- Type
- Reason
- Action
- Human-readable note
- Reference to the affected `TrussiumRuntime`

Unchanged repeated reconciliation does not emit duplicate transition Events.

## Watch Behaviour

The controller watches:

- `TrussiumRuntime` generation changes
- Owned ConfigMaps
- Owned ServiceAccounts
- Owned Services
- Owned Deployments
- Referenced Secrets

Primary `TrussiumRuntime` updates are filtered by generation. A status-only
write therefore does not enqueue another reconciliation loop.

Owned Deployment status changes still enqueue reconciliation so readiness and
failure conditions remain current.

## Status Writes

The controller writes through the Kubernetes status subresource.

Before writing, it:

1. Builds the desired observed status.
2. Preserves condition transition times when condition status is unchanged.
3. Compares desired and stored status.
4. Skips the write when there is no semantic difference.

This reduces unnecessary API-server writes and prevents status-only loops.

## RBAC

The controller requires:

- Read access to `TrussiumRuntime`
- Status update and patch access
- Read-only Secret access
- Create and patch access for `events.k8s.io/events`
- Existing managed-resource permissions

It does not require:

- Secret mutation
- Pod access
- Finalizer access
- Cross-namespace Secret access

## Current Boundary

This milestone does not implement:

- Runtime HTTP health probing
- Direct Pod inspection
- External provider connectivity validation
- Credential-value validation
- External LoadBalancer endpoint discovery
- Production probes
- PodDisruptionBudget
- HorizontalPodAutoscaler
- NetworkPolicy
- Configuration checksum rollouts
- Automated rollback

Production workload hardening is the next milestone.
