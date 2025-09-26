# Controller Metrics Package

This package provides shared Prometheus metrics for all LiteLLM Operator controllers, enabling consistent monitoring and observability across the entire system.

## Overview

The `controllermetrics` package defines common metrics that are used by all controllers in the LiteLLM Operator to track reconciliation activity, errors, and performance. This ensures consistent monitoring patterns and reduces code duplication.

## Metrics Provided

### Core Controller Metrics

All metrics include a `controller` label to differentiate between different controller types:

- `model` - Manages Model resources
- `team` - Manages Team resources  
- `user` - Manages User resources
- `virtualkey` - Manages VirtualKey resources
- `teammemberassociation` - Manages team membership
- `litellminstance` - Manages LiteLLMInstance resources

#### `litellm_operator_reconcile_loops_total`
**Type**: Counter  
**Description**: Total number of reconciliation loops executed per controller.

```promql
# Examples
rate(litellm_operator_reconcile_loops_total[1m]) * 60  # Reconciliations per minute
increase(litellm_operator_reconcile_loops_total[1h])   # Total reconciliations in last hour
```

#### `litellm_operator_reconcile_errors_total`
**Type**: Counter  
**Description**: Total number of reconciliation errors per controller.

```promql
# Examples  
rate(litellm_operator_reconcile_errors_total[5m]) / rate(litellm_operator_reconcile_loops_total[5m])  # Error rate
sum(rate(litellm_operator_reconcile_errors_total[1m]))  # Total error rate across controllers
```

#### `litellm_operator_reconcile_latency_seconds`
**Type**: Histogram  
**Description**: Duration of reconciliation operations per controller.

```promql
# Examples
histogram_quantile(0.95, rate(litellm_operator_reconcile_latency_seconds_bucket[5m]))  # 95th percentile
rate(litellm_operator_reconcile_latency_seconds_sum[5m]) / rate(litellm_operator_reconcile_latency_seconds_count[5m])  # Average latency
```

## Usage

### For Controller Developers

Controllers should integrate these metrics in their `Reconcile()` method:

```go
func (r *MyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Instrument reconcile loop and latency
    controllermetrics.InstrumentReconcileLoop("mycontroller")
    timer := controllermetrics.InstrumentReconcileLatency("mycontroller")
    defer timer.ObserveDuration()
    
    // Your reconciliation logic here...
    
    // Error metrics are handled automatically by base controller
    if err != nil {
        return r.HandleErrorRetryable(ctx, obj, err, "SomeReason")
    }
    
    return ctrl.Result{}, nil
}
```

### Using Base Controller Integration

The recommended approach is to use the base controller's built-in methods:

```go
// In controller constructor
func NewMyReconciler(client client.Client, scheme *runtime.Scheme) *MyReconciler {
    return &MyReconciler{
        BaseController: &base.BaseController[*v1alpha1.MyResource]{
            Client:         client,
            Scheme:         scheme,
            DefaultTimeout: 20 * time.Second,
            ControllerName: "mycontroller", // This enables automatic metrics
        },
    }
}

// In Reconcile method
func (r *MyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // These methods automatically use the ControllerName for metrics
    r.InstrumentReconcileLoop()
    timer := r.InstrumentReconcileLatency()
    defer timer.ObserveDuration()
    
    // Error handling automatically instruments errors
    if err := someOperation(); err != nil {
        return r.HandleErrorRetryable(ctx, obj, err, "OperationFailed")
    }
    
    return ctrl.Result{}, nil
}
```

## Monitoring and Alerting

### Recommended Alerts

```promql
# High error rate (> 10%)
rate(litellm_operator_reconcile_errors_total[5m]) / rate(litellm_operator_reconcile_loops_total[5m]) > 0.1

# High latency (95th percentile > 5 seconds)
histogram_quantile(0.95, rate(litellm_operator_reconcile_latency_seconds_bucket[5m])) > 5

# Controller not active (no reconciliations in 5 minutes)
rate(litellm_operator_reconcile_loops_total[5m]) == 0

# High reconciliation rate (> 100 per minute, may indicate resource churn)
rate(litellm_operator_reconcile_loops_total[1m]) * 60 > 100
```

### Dashboard Queries

```promql
# Controller activity overview
sum by (controller) (rate(litellm_operator_reconcile_loops_total[1m])) * 60

# Error rates by controller
sum by (controller) (rate(litellm_operator_reconcile_errors_total[5m])) / sum by (controller) (rate(litellm_operator_reconcile_loops_total[5m])) * 100

# Latency percentiles by controller
histogram_quantile(0.50, sum by (controller, le) (rate(litellm_operator_reconcile_latency_seconds_bucket[5m])))
histogram_quantile(0.95, sum by (controller, le) (rate(litellm_operator_reconcile_latency_seconds_bucket[5m])))
histogram_quantile(0.99, sum by (controller, le) (rate(litellm_operator_reconcile_latency_seconds_bucket[5m])))
```

## Implementation Notes

- All metrics are registered automatically when the package is imported
- Metrics are thread-safe and can be called concurrently
- The `controller` label should match the controller name used in logs and other observability
- Error instrumentation is handled automatically when using base controller error methods
- Metrics are exposed on the same endpoint as other controller-runtime metrics (typically `:8443/metrics`)

## Testing

The package includes comprehensive tests in `metrics_test.go` that verify:
- Metric initialization and registration
- Correct counter increments  
- Timer functionality
- Label values are properly set

Run tests with:
```bash
go test ./internal/controller/metrics/
```