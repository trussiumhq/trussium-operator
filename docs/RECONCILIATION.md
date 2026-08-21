# Core Runtime Reconciliation

The Trussium Operator reconciles a namespaced `TrussiumRuntime` resource into
the Kubernetes resources required to run one Trussium runtime instance.

## Managed Resources

For each `TrussiumRuntime`, the controller manages:

- ConfigMap
- ServiceAccount
- Service
- Deployment

Each managed resource:

- Uses the same name and namespace as its `TrussiumRuntime`
- Has stable Kubernetes application labels
- Has a controller owner reference
- Is recreated when deleted
- Is corrected when its managed configuration drifts

Kubernetes garbage collection removes managed resources when the owning
`TrussiumRuntime` is deleted.

No finalizer is required because this milestone does not manage external
resources.

## Reconciliation Flow

A reconciliation iteration performs:

1. Read the `TrussiumRuntime`.
2. Ignore requests for resources that no longer exist.
3. Ignore resources already being deleted.
4. Create or update the ConfigMap.
5. Create or update the ServiceAccount.
6. Create or update the Service.
7. Create or update the Deployment.
8. Return without periodic requeueing.

The controller also watches owned resources. A change or deletion affecting a
managed ConfigMap, ServiceAccount, Service, or Deployment triggers another
reconciliation of the owning `TrussiumRuntime`.

## Resource Names

The initial naming contract uses the custom-resource name directly.

For a resource declared as:

    metadata:
      name: private-ai
      namespace: trussium

The controller creates:

    ConfigMap/private-ai
    ServiceAccount/private-ai
    Service/private-ai
    Deployment/private-ai

All resources remain in the `trussium` namespace.

## Labels

Managed resources use:

    app.kubernetes.io/name: trussium
    app.kubernetes.io/instance: <TrussiumRuntime name>
    app.kubernetes.io/component: runtime
    app.kubernetes.io/managed-by: trussium-operator

Services and Deployments select runtime pods using:

    app.kubernetes.io/name: trussium
    app.kubernetes.io/instance: <TrussiumRuntime name>

The selector intentionally excludes mutable or descriptive labels.

## ConfigMap

The managed ConfigMap projects non-secret runtime configuration.

Always projected:

| Environment variable | Value |
|---|---|
| `TRUSSIUM_ENVIRONMENT` | `production` |
| `TRUSSIUM_RUNTIME__HOST` | `0.0.0.0` |
| `TRUSSIUM_RUNTIME__PORT` | `9000` |
| `TRUSSIUM_PROVIDER__NAME` | Selected provider type |

Conditionally projected:

| Environment variable | Custom-resource source |
|---|---|
| `TRUSSIUM_PROVIDER__BASE_URL` | `spec.provider.baseURL` |
| `TRUSSIUM_TIMEOUTS__PROVIDER_REQUEST_SECONDS` | `spec.runtime.providerRequestTimeoutSeconds` |
| `TRUSSIUM_TIMEOUTS__STREAM_IDLE_SECONDS` | `spec.runtime.streamIdleTimeoutSeconds` |
| `TRUSSIUM_RUNTIME__GRACEFUL_SHUTDOWN_SECONDS` | `spec.runtime.shutdownDrainTimeoutSeconds` |

Optional values remain absent when not configured. This allows the selected
Trussium runtime release to apply its own defaults.

`spec.provider.model` remains part of the public Kubernetes API, but this
milestone does not invent a runtime environment variable for a process-wide
default model.

## Provider Credentials

Provider credentials are projected directly from the referenced Kubernetes
Secret into the runtime container:

    TRUSSIUM_PROVIDER__API_KEY

The operator does not:

- Read the Secret value
- Copy the Secret value into a ConfigMap
- Store the value in status
- Include the value in logs
- Include the value in Kubernetes Events

The controller therefore does not require Secret read permissions.

## ServiceAccount

The controller creates a dedicated ServiceAccount for each runtime.

The ServiceAccount:

- Uses the runtime name
- Disables automatic Kubernetes API token mounting
- Includes configured image-pull Secret references
- Has a controller owner reference

The runtime workload does not need Kubernetes API access during this
milestone.

## Service

The managed Service:

- Uses `spec.service.type`
- Uses `spec.service.port`
- Targets the container port named `http`
- Selects only pods belonging to the owning runtime
- Preserves Kubernetes-assigned Service fields such as `clusterIP`

Supported Service types are:

- `ClusterIP`
- `NodePort`
- `LoadBalancer`

## Deployment

The managed Deployment:

- Uses `spec.replicas`
- Uses the selected image tag or digest
- Uses the dedicated ServiceAccount
- Disables ServiceAccount token mounting
- Disables Kubernetes Service environment-variable injection
- Exposes TCP port `9000` as `http`
- Loads non-secret settings through the managed ConfigMap
- Projects provider credentials through a Secret key reference
- Applies image-pull Secret references
- Applies configured resource requests and limits

Tag-based image example:

    ghcr.io/trussiumhq/trussium:v0.23.0

Digest-based image example:

    ghcr.io/trussiumhq/trussium@sha256:<digest>

## Ownership and Drift Correction

Every managed resource has a controller owner reference pointing to its
`TrussiumRuntime`.

The controller uses create-or-update reconciliation:

- Missing resources are created.
- Managed fields are restored after drift.
- Deleted managed resources are recreated.
- Repeated reconciliation produces the same desired configuration.

User-controlled fields outside the operator's managed contract should not be
added directly to generated resources because the controller may overwrite
them.

Additional supported customization must be introduced through the
`TrussiumRuntime` API.

## RBAC

The controller requires:

- Read access to `TrussiumRuntime` resources
- Create, read, update, patch, delete, list, and watch access for ConfigMaps
- Equivalent management access for ServiceAccounts
- Equivalent management access for Services
- Equivalent management access for Deployments
- Equivalent management access for PodDisruptionBudgets
- Equivalent management access for NetworkPolicies
- Read access to referenced Secrets
- Status updates and Kubernetes Event creation

The controller does not currently require:

- Pod reads

## Runtime Upgrade Reconciliation

Runtime image transitions use the existing Deployment reconciliation path.

The controller does not create separate upgrade resources.

After reconciling the desired Deployment, the controller observes Deployment
status and determines whether the rollout is:

- Waiting for the Deployment controller
- Progressing
- Complete
- Failed

Successful rollout advances:

    status.lastSuccessfulImage

Failed rollout preserves the previous successful image.

The controller never automatically rewrites `spec.image`.

### Deployment Progress Deadline

Managed Deployments use:

    progressDeadlineSeconds: 600

This allows stalled rollouts to surface through the standard Deployment
`Progressing` condition.

### Configuration Rollout Checksum

Managed runtime configuration is represented in the ConfigMap.

The controller calculates a deterministic SHA-256 checksum of that ConfigMap
data and projects it onto the Pod template:

    runtime.trussium.io/config-checksum

A runtime configuration change therefore modifies the Pod template and causes
the Deployment controller to create a new revision.

Secret values are never included in checksum calculation.

### Upgrade Observation Boundary

The controller observes only the Deployment.

It does not require:

- Pod reads
- ReplicaSet reads
- Periodic rollout polling
- Runtime HTTP probes from the controller

## Current Scope Boundary

The controller intentionally does not reconcile arbitrary Pod specifications,
finalizers, or egress NetworkPolicy rules. Runtime NetworkPolicy support is
opt-in and manages ingress only; this preserves DNS and provider egress while
requiring explicit client selectors for ingress. CPU-based HPA support is also
opt-in and preserves the autoscaler's Deployment replica decisions.
