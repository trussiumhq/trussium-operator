# Controller Metrics

The operator exposes authenticated HTTPS Prometheus metrics through its
controller metrics Service. Enable the chart's optional ServiceMonitor when
using Prometheus Operator.

## Reconciliation Metrics

| Metric | Type | Labels | Meaning |
|---|---|---|---|
| `trussium_operator_runtime_reconciliations_total` | Counter | `result` | Completed reconciliation attempts. |
| `trussium_operator_runtime_reconciliation_duration_seconds` | Histogram | `result` | Reconciliation duration. |

`result` is either `success` or `error`. Metrics intentionally exclude resource
name, namespace, image, Secret, and error-message labels to keep cardinality
bounded and avoid exposing sensitive operational details.
