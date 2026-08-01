.PHONY: deploy-manifests-dev
deploy-manifests-dev: manifests kustomize
	$(KUBECTL) apply -k config/samples/extra/cnpg
	sleep 10
	$(KUBECTL) wait --for=condition=Ready --timeout=120s -n cnpg-system pod -l app.kubernetes.io/name=cloudnative-pg
	$(KUBECTL) apply -k config/samples/extra
	sleep 10
	$(KUBECTL) wait --for=condition=complete --timeout=120s -n litellm job/litellm-postgres-1-initdb
	sleep 5
	$(KUBECTL) wait --for=condition=Ready --timeout=120s -n litellm pod -l cnpg.io/instanceName=litellm-postgres-1,cnpg.io/instanceRole=primary

.PHONY: deploy-controller-dev
deploy-controller-dev: manifests kustomize
	@TMPDIR=$$(mktemp -d) && \
	cp -r config/* $$TMPDIR && \
	cd $$TMPDIR/manager && $(KUSTOMIZE) edit set image controller=${IMG} && \
	cd - > /dev/null && \
	$(KUSTOMIZE) build $$TMPDIR/default | $(KUBECTL) apply -f - && \
	rm -rf $$TMPDIR

##@ Local Development

.PHONY: vind-create
vind-create: ## Create a vCluster in Docker.
	@if vcluster list | grep -q "litellm-operator"; then \
		echo "vCluster litellm-operator already exists. Reconnecting..."; \
		vcluster connect litellm-operator; \
	else \
		echo "Creating vCluster litellm-operator..."; \
		vcluster create litellm-operator -f scripts/local-dev/vcluster.yaml && \
		$(KUBECTL) apply -f scripts/local-dev/storageclass.yaml; \
	fi

.PHONY: vind-delete
vind-delete: ## Delete a vCluster in Docker.
	@if vcluster list | grep -q "litellm-operator"; then \
		echo "Deleting vCluster litellm-operator..."; \
		vcluster delete litellm-operator; \
	else \
		echo "No vCluster litellm-operator found to delete."; \
	fi

.PHONY: vind-bootstrap-full
vind-bootstrap-full: IMG=litellm-operator:dev
vind-bootstrap-full: vind-create docker-build install deploy-controller-dev deploy-manifests-dev install-samples-local-dev ## Bootstrap the vCluster with the dev image.

.PHONY: vind-bootstrap-full-recreate
vind-bootstrap-full-recreate: vind-delete vind-bootstrap-full ## Recreate the vCluster and bootstrap.

.PHONY: vind-bootstrap-debug
vind-bootstrap-debug: vind-create install deploy-manifests-dev install-samples-local-dev ## Bootstrap the vCluster without the litellm-operator.

.PHONY: vind-bootstrap-debug-recreate
vind-bootstrap-debug-recreate: vind-delete vind-bootstrap-debug ## Recreate the vCluster and bootstrap without the litellm-operator.

.PHONY: install-samples-local-dev
install-samples-local-dev: install-samples ## Install samples with LoadBalancer service for local development.
	$(KUBECTL) apply -k config/samples/local-dev

.PHONY: vind-get-loadbalancer-ip
vind-get-loadbalancer-ip: ## Get the IP address of the load balancer service.
	$(KUBECTL) get svc litellm-example-loadbalancer -n litellm -o jsonpath='{.status.loadBalancer.ingress[0].ip}'

.PHONY: debug-port-forward-litellm
debug-port-forward-litellm: ## Port forward the litellm example service for a debug session of the operator.
	$(KUBECTL) port-forward svc/litellm-example-service 4000:4000 -n litellm

.PHONY: vind-get-admin-password
vind-get-admin-password: ## Get the admin password for the litellm example service.
	$(KUBECTL) get secret litellm-example-secrets -n litellm -o jsonpath='{.data.masterkey}' | base64 -d
