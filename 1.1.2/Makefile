##@ Documentation

# Detect if we're running from the root directory or docs directory
# If docs/Makefile exists relative to current directory, we're in root
# Otherwise, we're in the docs directory
ifeq ($(wildcard docs/Makefile),docs/Makefile)
	# Running from root directory
	DOCS_DIR := docs
	API_DIR := api
	CONFIG_FILE := docs/.crd-ref-docs.yaml
	OUTPUT_DIR := docs/api-reference
	# Ensure we have access to the main Makefile variables
	LOCALBIN ?= $(shell pwd)/bin
	CRD_REF_DOCS ?= $(LOCALBIN)/crd-ref-docs
else
	# Running from docs directory
	DOCS_DIR := .
	API_DIR := ../api
	CONFIG_FILE := .crd-ref-docs.yaml
	OUTPUT_DIR := api-reference
	# Ensure we have access to the main Makefile variables
	LOCALBIN ?= $(shell pwd)/../bin
	CRD_REF_DOCS ?= $(LOCALBIN)/crd-ref-docs
endif

.PHONY: api-docs
gen-crd-ref-docs: ## Generate API reference documentation from Go types.
	@echo "Generating API reference documentation..."
	@mkdir -p $(OUTPUT_DIR)
	$(CRD_REF_DOCS) \
		--source-path=$(API_DIR) \
		--config=$(CONFIG_FILE) \
		--renderer=markdown \
		--output-path=$(OUTPUT_DIR) \
		--output-mode=group
	@echo "API documentation generated at $(OUTPUT_DIR)/api-generated.md"

.PHONY: docs

.PHONY: install-docs
install-docs: install-mkdocs ## Install all documentation dependencies

.PHONY: docs-serve
docs-serve: gen-crd-ref-docs install-mkdocs
	docker run --rm -it -p 8000:8000 -v $(DOCS_DIR)/..:/docs squidfunk/mkdocs-material:latest serve -a 0.0.0.0:8000
.PHONY: docs-build
docs-build: gen-crd-ref-docs
	docker run --rm -it -v $(DOCS_DIR)/..:/docs squidfunk/mkdocs-material:latest build 

.PHONY: docs-build-local
docs-build-local: gen-crd-ref-docs install-mkdocs
	cd $(DOCS_DIR)/.. && python3 -m mkdocs build

.PHONY: docs-build-strict
docs-build-strict: gen-crd-ref-docs install-mkdocs
	cd $(DOCS_DIR)/.. && python3 -m mkdocs build --strict

.PHONY: install-mkdocs
install-mkdocs: ## Install mkdocs and required dependencies
	@echo "📦 Checking mkdocs installation..."
	@if ! python3 -c "import mkdocs" >/dev/null 2>&1; then \
		echo "🔧 mkdocs not found. Installing dependencies..."; \
		python3 -m pip install --upgrade pip; \
		python3 -m pip install -r $(DOCS_DIR)/requirements.txt; \
		echo "✅ mkdocs and dependencies installed successfully"; \
	else \
		echo "✅ mkdocs already installed"; \
	fi