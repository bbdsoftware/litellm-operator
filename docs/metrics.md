# LiteLLM Operator Metrics Documentation

This document describes all the metrics exposed by the LiteLLM Operator for monitoring and observability.

## Overview

The LiteLLM Operator exposes Prometheus metrics to provide visibility into:
- Reconciliation loop activity and performance
- Error rates and failure patterns
- Resource management status
- Operator health and performance

All metrics are prefixed with `litellm_operator` to avoid conflicts with other components.

## Controller Metrics

These metrics track the behavior of Kubernetes controllers that manage LiteLLM resources.

### litellm_operator_reconcile_loops_total

**Type**: Counter  
**Description**: Total number of reconciliation loops executed per controller.  
**Labels**:
- `controller`: Name of the controller (e.g., `model`, `team`, `user`, `virtualkey`, `teammemberassociation`, `litellminstance`)

**Usage**: 
- Monitor controller activity levels
- Identify controllers processing high volumes of changes
- Correlate with cluster events or resource modifications
- Calculate reconciliation rates over time

**Example Queries**:
```promql
# Reconciliation rate per controller (per minute)
rate(litellm_operator_reconcile_loops_total[1m]) * 60

# Total reconciliations in last hour by controller
increase(litellm_operator_reconcile_loops_total[1h])

# Most active controllers
topk(5, rate(litellm_operator_reconcile_loops_total[5m]))
```

**Alerting Examples**:
```promql
# Alert if any controller is processing > 100 reconciliations per minute
rate(litellm_operator_reconcile_loops_total[1m]) * 60 > 100

# Alert if controller activity drops to zero (may indicate controller failure)
rate(litellm_operator_reconcile_loops_total[5m]) == 0
```

### litellm_operator_reconcile_errors_total

**Type**: Counter  
**Description**: Total number of reconciliation errors encountered per controller.  
**Labels**:
- `controller`: Name of the controller

**Usage**:
- Monitor error rates and identify problematic controllers
- Track system reliability over time
- Correlate errors with deployments or configuration changes
- Calculate error percentages

**Example Queries**:
```promql
# Error rate per controller (per minute)  
rate(litellm_operator_reconcile_errors_total[1m]) * 60

# Error percentage by controller
rate(litellm_operator_reconcile_errors_total[5m]) / rate(litellm_operator_reconcile_loops_total[5m]) * 100

# Controllers with highest error rates
topk(5, rate(litellm_operator_reconcile_errors_total[5m]))
```

**Alerting Examples**:
```promql
# Alert if error rate exceeds 10% for any controller
rate(litellm_operator_reconcile_errors_total[5m]) / rate(litellm_operator_reconcile_loops_total[5m]) > 0.1

# Alert if absolute error count is high (> 5 errors per minute)
rate(litellm_operator_reconcile_errors_total[1m]) * 60 > 5
```

### litellm_operator_reconcile_latency_seconds

**Type**: Histogram  
**Description**: Duration of reconciliation loops in seconds per controller.  
**Labels**:
- `controller`: Name of the controller

**Buckets**: Default Prometheus histogram buckets (0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, +Inf)

**Usage**:
- Monitor reconciliation performance
- Identify performance degradation
- Set SLAs for reconciliation latency
- Detect resource contention or external service delays

**Example Queries**:
```promql
# 95th percentile reconciliation latency by controller
histogram_quantile(0.95, rate(litellm_operator_reconcile_latency_seconds_bucket[5m]))

# Average reconciliation latency
rate(litellm_operator_reconcile_latency_seconds_sum[5m]) / rate(litellm_operator_reconcile_latency_seconds_count[5m])

# Reconciliations taking longer than 1 second
rate(litellm_operator_reconcile_latency_seconds_bucket{le="1"}[5m])
```

**Alerting Examples**:
```promql
# Alert if 95th percentile latency exceeds 5 seconds
histogram_quantile(0.95, rate(litellm_operator_reconcile_latency_seconds_bucket[5m])) > 5

# Alert if average latency exceeds 2 seconds
rate(litellm_operator_reconcile_latency_seconds_sum[5m]) / rate(litellm_operator_reconcile_latency_seconds_count[5m]) > 2
```

## LiteLLM Instance Specific Metrics

These metrics are specific to the LiteLLMInstance controller and track managed Kubernetes resources.

### litellm_managed_resource_active

**Type**: Gauge  
**Description**: Indicates whether a specific managed resource for a LiteLLMInstance is active (1) or inactive (0).  
**Labels**:
- `instance`: Name of the LiteLLMInstance
- `namespace`: Namespace of the LiteLLMInstance  
- `resource`: Type of managed resource (configmap, secret, deployment, service, ingress, etc.)

**Usage**:
- Monitor which resources are currently active for each LiteLLMInstance
- Track resource lifecycle changes
- Identify missing or inactive resources that should be present

**Example Queries**:
```promql
# Count of active resources per instance
sum by (instance, namespace) (litellm_managed_resource_active)

# Instances with inactive critical resources
litellm_managed_resource_active{resource=~"deployment|service"} == 0

# Resource availability across all instances
avg by (resource) (litellm_managed_resource_active)
```

### litellm_managed_resource_status

**Type**: Gauge  
**Description**: Status of a managed resource for a LiteLLMInstance. Value is 1 for the current status, 0 for other statuses.  
**Labels**:
- `instance`: Name of the LiteLLMInstance
- `namespace`: Namespace of the LiteLLMInstance
- `resource`: Type of managed resource
- `status`: Current status (inactive, missing, created, ready, not_ready, etc.)

**Usage**:
- Monitor detailed status of managed resources
- Track resource readiness and health
- Identify resources stuck in transitional states
- Generate alerts for unhealthy resources

**Example Queries**:
```promql
# Resources in 'not_ready' state
sum by (instance, resource) (litellm_managed_resource_status{status="not_ready"})

# Instances with any missing resources
sum by (instance, namespace) (litellm_managed_resource_status{status="missing"}) > 0

# Resource status distribution
sum by (status) (litellm_managed_resource_status)
```

**Alerting Examples**:
```promql
# Alert if any resource is missing for more than 5 minutes
litellm_managed_resource_status{status="missing"} == 1

# Alert if deployment is not ready
litellm_managed_resource_status{resource="deployment", status="not_ready"} == 1
```