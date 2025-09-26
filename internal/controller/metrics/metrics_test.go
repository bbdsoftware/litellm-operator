package controllermetrics

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
)

func TestMetricsInitialization(t *testing.T) {
	// Test that metrics are properly initialized
	if ReconcileLoopsTotal == nil {
		t.Fatal("ReconcileLoopsTotal should not be nil")
	}
	if ReconcileErrorsTotal == nil {
		t.Fatal("ReconcileErrorsTotal should not be nil")
	}
	if ReconcileLatency == nil {
		t.Fatal("ReconcileLatency should not be nil")
	}
}

func TestInstrumentReconcileLoop(t *testing.T) {
	controllerName := "test-controller"

	// Get initial count
	metric := &dto.Metric{}
	counter, err := ReconcileLoopsTotal.GetMetricWithLabelValues(controllerName)
	if err != nil {
		t.Fatal(err)
	}
	counter.Write(metric)
	initialValue := metric.GetCounter().GetValue()

	// Increment
	InstrumentReconcileLoop(controllerName)

	// Check increment
	counter.Write(metric)
	newValue := metric.GetCounter().GetValue()

	if newValue != initialValue+1 {
		t.Errorf("Expected %f, got %f", initialValue+1, newValue)
	}
}

func TestInstrumentReconcileError(t *testing.T) {
	controllerName := "test-controller"

	// Get initial count
	metric := &dto.Metric{}
	counter, err := ReconcileErrorsTotal.GetMetricWithLabelValues(controllerName)
	if err != nil {
		t.Fatal(err)
	}
	counter.Write(metric)
	initialValue := metric.GetCounter().GetValue()

	// Increment
	InstrumentReconcileError(controllerName)

	// Check increment
	counter.Write(metric)
	newValue := metric.GetCounter().GetValue()

	if newValue != initialValue+1 {
		t.Errorf("Expected %f, got %f", initialValue+1, newValue)
	}
}

func TestInstrumentReconcileLatency(t *testing.T) {
	controllerName := "test-controller"

	// This just tests that we can create a timer without error
	timer := InstrumentReconcileLatency(controllerName)
	if timer == nil {
		t.Fatal("Timer should not be nil")
	}

	// Simulate observing the duration
	timer.ObserveDuration()
}
