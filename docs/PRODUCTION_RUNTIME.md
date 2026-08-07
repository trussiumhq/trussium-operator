# Production Runtime Contract

The Trussium Operator applies a production Kubernetes workload contract to every
managed `TrussiumRuntime`.

This contract is intentionally opinionated. Security, health checks, rollout
behaviour, and disruption protection are controlled by the operator rather than
exposed as arbitrary Pod configuration.

## Runtime Image

Examples use the canonical Trussium runtime image repository:

    ghcr.io/trussiumhq/trussium

A `TrussiumRuntime` may reference another compatible image repository through
`spec.image.repository`.

## Health Probes

The runtime container exposes Kubernetes health endpoints through the named
`http` port.

### Startup Probe

    path: /health/live
    periodSeconds: 2
    timeoutSeconds: 1
    failureThreshold: 30

The startup probe prevents liveness and readiness checks from interfering with
runtime initialization.

### Liveness Probe

    path: /health/live
    periodSeconds: 10
    timeoutSeconds: 2
    failureThreshold: 3

### Readiness Probe

    path: /health/ready
    periodSeconds: 5
    timeoutSeconds: 2
    failureThreshold: 3

Probe configuration is part of the supported runtime contract and is not
currently user-configurable.

## Pod Security

Runtime Pods use:

    runAsNonRoot: true
    runAsUser: 10001
    runAsGroup: 10001

The Pod seccomp profile is:

    RuntimeDefault

The runtime container uses:

    allowPrivilegeEscalation: false
    readOnlyRootFilesystem: true

All Linux capabilities are dropped.

The workload is not privileged.

## Service Account Isolation

Every runtime receives a dedicated ServiceAccount.

The runtime Pod also uses:

    automountServiceAccountToken: false
    enableServiceLinks: false

The runtime workload therefore receives no Kubernetes API credentials unless a
future explicitly supported capability requires them.

## Graceful Termination

The Trussium runtime supports graceful request draining during process
termination.

By default:

    runtime drain timeout = 30 seconds
    Kubernetes safety margin = 6 seconds
    terminationGracePeriodSeconds = 36

When:

    spec.runtime.shutdownDrainTimeoutSeconds

is configured, the operator calculates:

    terminationGracePeriodSeconds =
        shutdownDrainTimeoutSeconds + 6

The margin gives the runtime additional time for cancellation cleanup and
process exit after draining completes.

## Rolling Updates

Managed Deployments use:

    strategy:
      type: RollingUpdate
      rollingUpdate:
        maxUnavailable: 0
        maxSurge: 1

Deployment revision history is limited to:

    revisionHistoryLimit: 3

The zero-unavailable strategy prioritizes runtime availability during ordinary
updates when sufficient cluster capacity exists.

## Topology Spreading

The operator adds a hostname topology-spread constraint:

    maxSkew: 1
    topologyKey: kubernetes.io/hostname
    whenUnsatisfiable: ScheduleAnyway

The selector matches the stable labels belonging to the corresponding
`TrussiumRuntime`.

This encourages replicas to spread across nodes without making scheduling
impossible when the cluster has limited topology diversity.

## PodDisruptionBudget

Every runtime receives an owned `policy/v1` PodDisruptionBudget.

The default policy is:

    maxUnavailable: 1

The PodDisruptionBudget uses the same stable runtime selector labels as the
Deployment.

The operator:

- Creates it
- Corrects configuration drift
- Recreates it after deletion
- Watches it as an owned resource
- Relies on Kubernetes garbage collection when the runtime is deleted

## Pod Metadata

Additional Pod labels and annotations may be configured:

    spec:
      podMetadata:
        labels:
          team: ai-platform
        annotations:
          example.com/owner: inference

Operator-owned labels cannot be overridden.

This protects Deployment selectors, ownership identity, and controller
behaviour.

User annotations are copied to the Pod template.

Changes to Pod metadata modify the Pod template and therefore trigger the normal
Deployment rollout process.

## Scheduling

Supported scheduling controls are exposed under:

    spec.scheduling

Supported fields are:

- `nodeSelector`
- `tolerations`
- `affinity`

Example:

    spec:
      scheduling:
        nodeSelector:
          workload: ai

        tolerations:
          - key: workload
            operator: Equal
            value: ai
            effect: NoSchedule

Scheduling configuration uses Kubernetes-native structures while avoiding
arbitrary PodSpec exposure.

The operator-managed topology-spread constraint remains enabled independently
of user affinity settings.

## Managed Resource Set

A `TrussiumRuntime` now owns:

- ConfigMap
- ServiceAccount
- Service
- Deployment
- PodDisruptionBudget

All managed resources use deterministic names, stable labels, and controller
owner references.

## RBAC

The operator has lifecycle permissions for:

    policy/poddisruptionbudgets

It still requires no direct Pod permissions.

The runtime workload itself receives no Kubernetes API RBAC through its
ServiceAccount.

## Configuration Boundary

The operator does not expose arbitrary configuration for:

- Containers
- Init containers
- Sidecars
- Host networking
- Host namespaces
- Linux capabilities
- Seccomp profiles
- Runtime user IDs
- Health probes
- PodDisruptionBudget policy
- HostPath volumes
- Arbitrary PodSpec fields

These restrictions preserve a stable, secure runtime contract.

## Next Milestone

The next milestone focuses on runtime upgrade lifecycle and rollout behaviour,
including explicit compatibility and upgrade-state handling.
