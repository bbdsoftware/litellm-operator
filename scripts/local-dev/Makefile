

##@ Local Development

.PHONY: kind-cluster
kind-cluster: ## Create a Kind cluster.
	@if kind get clusters | grep -q "kind"; then \
		echo "Kind cluster already exists. Skipping creation."; \
	else \
		echo "Creating Kind cluster..."; \
		kind create cluster; \
	fi


.PHONY: kind-cluster-delete
kind-cluster-delete: ## Delete a Kind cluster.
	@if kind get clusters | grep -q "kind"; then \
		echo "Deleting Kind cluster..."; \
		kind delete cluster; \
	else \
		echo "No Kind cluster found to delete."; \
	fi


install-samples-extra: ## Install extra samples into the K8s cluster specified in ~/.kube/config.
	$(KUBECTL) apply -k config/samples/extra


.PHONY: dev-cluster-bootstrap
dev-cluster-bootstrap: kind-cluster manifests install install-samples-extra install-samples ## Bootstrap the development cluster.
	
dev-cluster-recreate: kind-cluster-delete dev-cluster-bootstrap




