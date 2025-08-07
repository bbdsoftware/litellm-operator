# Helm-specific Makefile for litellm-operator
# This file contains all Helm-related targets and configurations

# Detect the correct chart path by checking if the chart directory exists
# This works whether we're running from project root or deploy directory
CHART_PATH = $(shell if [ -d "charts/litellm-operator" ]; then echo "charts/litellm-operator"; elif [ -d "deploy/charts/litellm-operator" ]; then echo "deploy/charts/litellm-operator"; else echo "charts/litellm-operator"; fi)

##@ Helm

.PHONY: helm-lint
helm-lint: ## Lint the Helm chart.
	helm lint $(CHART_PATH)

.PHONY: helm-template
helm-template: ## Template the Helm chart.
	helm template litellm-operator $(CHART_PATH)

.PHONY: helm-package
helm-package: ## Package the Helm chart.
	helm package $(CHART_PATH)
	mkdir -p dist && mv litellm-operator-*.tgz dist/

.PHONY: helm-install
helm-install: ## Install the Helm chart locally.
	helm dependency update $(CHART_PATH)
	helm install -n litellm-operator-system --create-namespace litellm-operator $(CHART_PATH)

.PHONY: helm-uninstall
helm-uninstall: ## Uninstall the Helm chart locally.
	helm uninstall -n litellm-operator-system litellm-operator

.PHONY: helm-upgrade
helm-upgrade: ## Upgrade the Helm chart locally.
	helm upgrade -n litellm-operator-system litellm-operator $(CHART_PATH)

.PHONY: helm-test
helm-test: ## Test the Helm chart.
	helm test -n litellm-operator-system litellm-operator

.PHONY: helm-docs
helm-docs: ## Generate Helm chart documentation.
	helm-docs --chart-search-root=helm --output-file-template=helm/README.md.gotmpl

.PHONY: helm-push-oci
helm-push-oci: helm-package ## Push Helm chart to OCI registry.
	helm push litellm-operator-*.tgz oci://ghcr.io/bbd/charts

HELMIFY ?= $(LOCALBIN)/helmify

.PHONY: run-helmify
run-helmify: ## Run helmify.
	$(KUSTOMIZE) build config/default | $(HELMIFY) -crd-dir -cert-manager-as-subchart -add-webhook-option $(CHART_PATH)

.PHONY: helmify
helmify: $(HELMIFY) ## Download helmify locally if necessary.
$(HELMIFY): $(LOCALBIN)
	test -s $(LOCALBIN)/helmify || GOBIN=$(LOCALBIN) go install github.com/arttor/helmify/cmd/helmify@latest

.PHONY: helm-gen
helm-gen: manifests kustomize helmify run-helmify  ## Generate Helm chart from Kustomize output.
	$(KUSTOMIZE) build config/default | $(HELMIFY) -crd-dir -cert-manager-as-subchart -add-webhook-option $(CHART_PATH) 