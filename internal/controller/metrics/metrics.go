/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package controllermetrics provides common metrics for all controllers
package controllermetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// ReconcileLoopsTotal tracks the total number of reconcile loops per controller.
	//
	// This counter is incremented every time a controller's Reconcile() method is called,
	// providing visibility into controller activity levels and reconciliation frequency.
	//
	// Labels:
	//   - controller: Name of the controller (model, team, user, virtualkey, teammemberassociation, litellminstance)
	ReconcileLoopsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "litellm_operator_reconcile_loops_total",
			Help: "Total number of reconcile loops per controller.",
		},
		[]string{"controller"},
	)

	// ReconcileErrorsTotal tracks the total number of reconciliation errors per controller
	ReconcileErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "litellm_operator_reconcile_errors_total",
			Help: "Total number of reconciliation errors per controller.",
		},
		[]string{"controller"},
	)

	// ReconcileLatency tracks the latency of reconciliation loops per controller
	ReconcileLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "litellm_operator_reconcile_latency_seconds",
			Help: "Latency of reconciliation loops per controller.",
		},
		[]string{"controller"},
	)
)

func init() {
	// Register metrics with the global prometheus registry.
	metrics.Registry.MustRegister(ReconcileLoopsTotal, ReconcileErrorsTotal, ReconcileLatency)
}

// InstrumentReconcileLoop increments the reconcile loops counter for a controller.
//
// This function should be called at the beginning of every Reconcile() method
// to track the total number of reconciliation operations performed by each controller.
//
// Parameters:
//   - controllerName: The name of the controller (e.g., "model", "team", "user")
//
// Example usage:
//
//	func (r *ModelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
//	    controllermetrics.InstrumentReconcileLoop("model")
//	    // ... rest of reconciliation logic
//	}
func InstrumentReconcileLoop(controllerName string) {
	ReconcileLoopsTotal.WithLabelValues(controllerName).Inc()
}

// InstrumentReconcileError increments the reconcile errors counter for a controller.
//
// This function should be called whenever a reconciliation operation encounters
// an error condition. In the LiteLLM Operator, this is typically called automatically
// by the base controller's error handling methods.
//
// Parameters:
//   - controllerName: The name of the controller experiencing the error
//
// Example usage:
//
//	if err := someOperation(); err != nil {
//	    controllermetrics.InstrumentReconcileError("model")
//	    return ctrl.Result{}, err
//	}
func InstrumentReconcileError(controllerName string) {
	ReconcileErrorsTotal.WithLabelValues(controllerName).Inc()
}

// InstrumentReconcileLatency creates a timer for measuring reconcile latency.
//
// This function returns a Prometheus Timer that should be used to measure
// the duration of reconciliation operations. The timer should be created at
// the start of the Reconcile() method and its ObserveDuration() method
// called via defer to ensure the measurement is recorded.
//
// Parameters:
//   - controllerName: The name of the controller being measured
//
// Returns:
//   - *prometheus.Timer: A timer instance for measuring duration
//
// Example usage:
//
//	func (r *ModelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
//	    timer := controllermetrics.InstrumentReconcileLatency("model")
//	    defer timer.ObserveDuration()
//	    // ... reconciliation logic
//	}
func InstrumentReconcileLatency(controllerName string) *prometheus.Timer {
	return prometheus.NewTimer(ReconcileLatency.WithLabelValues(controllerName))
}
