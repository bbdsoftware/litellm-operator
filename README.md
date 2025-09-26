# litellm-operator

A Kubernetes operator for managing litellm resources.

[![CI](https://github.com/bbdsoftware/litellm-operator/actions/workflows/ci.yml/badge.svg)](https://github.com/bbdsoftware/litellm-operator/actions/workflows/ci.yml)
[![Release](https://github.com/bbdsoftware/litellm-operator/actions/workflows/release.yml/badge.svg)](https://github.com/bbdsoftware/litellm-operator/actions/workflows/release.yml)
[![Documentation](https://img.shields.io/badge/docs-GitHub%20Pages-blue.svg)](https://bbdsoftware.github.io/litellm-operator/)
[![Go Report Card](https://goreportcard.com/badge/github.com/bbd/litellm-operator)](https://goreportcard.com/report/github.com/bbd/litellm-operator)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Go Version](https://img.shields.io/badge/Go-1.22+-blue.svg)](https://golang.org)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.11+-blue.svg)](https://kubernetes.io)

## Description

The operator is used to manage REST API operations on litellm resources:

- Virtual Keys
- Users
- Teams

## Getting Started

It is expected that the operator will be deployed in the same namespace as the litellm service.

### Prerequisites
- go version v1.22.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.
- Helm v3.8+ (for Helm installation method)

### To Deploy on the cluster

#### Option 1: Using Helm (Recommended)

**Authenticate with GitHub Container Registry:**

```sh
helm registry login ghcr.io -u YOUR_GITHUB_USERNAME -p YOUR_GITHUB_TOKEN
```

**Install the operator using Helm:**

```sh
helm install litellm-operator oci://ghcr.io/bbdsoftware/charts/litellm-operator --version <VERSION>
```

> **NOTE**: Replace `<VERSION>` with the desired version (e.g., `0.0.1`). You can find available versions in the [releases page](https://github.com/bbdsoftware/litellm-operator/releases).

#### Option 2: Manual Deployment

**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/litellm-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don't work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/litellm-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall

#### If installed with Helm:

```sh
helm uninstall litellm-operator
```

#### If installed manually:

**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/litellm-operator:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/litellm-operator/<tag or branch>/dist/install.yaml
```

## Monitoring and Observability

The LiteLLM Operator exposes comprehensive Prometheus metrics for monitoring controller health, performance, and resource management.

### Metrics Available

- **Controller Metrics**: Reconciliation loops, error rates, and latency per controller
- **Resource Metrics**: Status and health of managed LiteLLM resources
- **Performance Metrics**: Latency histograms and throughput measurements

### Accessing Metrics

Metrics are exposed on the controller manager's metrics endpoint (default: `:8443/metrics`) and can be scraped by Prometheus or any compatible monitoring system.

### Documentation

For detailed metrics documentation, monitoring setup, and alerting guidelines, see:

- **[Complete Metrics Documentation](docs/metrics.md)** - Comprehensive guide with all metrics, queries, and alerts
- **[Controller Metrics Package](internal/controller/metrics/README.md)** - Technical documentation for developers

### Quick Start Monitoring

```promql
# Controller reconciliation rate
rate(litellm_reconcile_loops_total[5m])

# Error percentage by controller  
rate(litellm_reconcile_errors_total[5m]) / rate(litellm_reconcile_loops_total[5m]) * 100

# 95th percentile reconciliation latency
histogram_quantile(0.95, rate(litellm_reconcile_latency_seconds_bucket[5m]))
```

## Contributing
See [CONTRIBUTING](CONTRIBUTING.md).

## License
The LiteLLM Operator is released under the Apache 2.0 license. See the [LICENSE](./LICENSE) file for details.
