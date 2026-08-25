/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package controller

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileRecordsSuccessMetric(t *testing.T) {
	before := testutil.ToFloat64(runtimeReconciliationsTotal.WithLabelValues("success"))
	reconciler := TrussiumRuntimeReconciler{Client: fake.NewClientBuilder().WithScheme(newTestScheme(t)).Build(), Scheme: newTestScheme(t)}
	if _, err := reconciler.Reconcile(context.Background(), ctrl.Request{}); err != nil {
		t.Fatalf("reconcile missing resource: %v", err)
	}
	after := testutil.ToFloat64(runtimeReconciliationsTotal.WithLabelValues("success"))
	if after != before+1 {
		t.Fatalf("expected success metric increment, before=%v after=%v", before, after)
	}
}
