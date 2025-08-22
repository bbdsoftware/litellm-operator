

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


.PHONY: install-samples-extra
install-samples-extra: ## Install extra samples into the K8s cluster specified in ~/.kube/config.
	$(KUBECTL) apply -k config/samples/extra

## Deploy and patch the image pull policy to Never for the controller image. Wait for pods to be ready.
.PHONY: deploy-dev
deploy-dev: deploy
	$(KUBECTL) patch deployment litellm-operator-controller-manager -n litellm-operator-system --type=json -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/imagePullPolicy", "value": "Never"}]'
	$(KUBECTL) apply -k config/samples/extra/cnpg
	sleep 10
	$(KUBECTL) wait --for=condition=Ready --timeout=120s -n cnpg-system pod -l app.kubernetes.io/name=cloudnative-pg
	$(KUBECTL) apply -k config/samples/extra
	sleep 10
	$(KUBECTL) wait --for=condition=complete --timeout=120s -n litellm job/litellm-postgres-1-initdb
	sleep 5
	$(KUBECTL) wait --for=condition=Ready --timeout=120s -n litellm pod -l cnpg.io/instanceName=litellm-postgres-1,cnpg.io/instanceRole=primary


.PHONY: dev-cluster-bootstrap
dev-cluster-bootstrap: kind-cluster install install-samples-extra install-samples ## Bootstrap the development cluster.

.PHONY: dev-cluster-bootstrap-full
dev-cluster-bootstrap-full: IMG=controller:dev
dev-cluster-bootstrap-full: kind-cluster docker-build docker-load install deploy-dev install-samples ## bootstrap the cluster and deploy a locally built controller image

.PHONY: dev-cluster-recreate
dev-cluster-recreate: kind-cluster-delete dev-cluster-bootstrap ## Recreate the Kind cluster and bootstrap
