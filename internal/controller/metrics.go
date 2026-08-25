/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package controller

import (
	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	runtimeReconciliationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "trussium_operator",
			Name:      "runtime_reconciliations_total",
			Help:      "Total TrussiumRuntime reconciliation attempts by result.",
		},
		[]string{"result"},
	)
	runtimeReconciliationDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "trussium_operator",
			Name:      "runtime_reconciliation_duration_seconds",
			Help:      "TrussiumRuntime reconciliation duration in seconds by result.",
		},
		[]string{"result"},
	)
)

func init() {
	ctrlmetrics.Registry.MustRegister(
		runtimeReconciliationsTotal,
		runtimeReconciliationDurationSeconds,
	)
}
