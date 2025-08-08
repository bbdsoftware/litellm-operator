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

.PHONY: docs-serve
docs-serve: gen-crd-ref-docs 
	docker run --rm -it -p 8000:8000 -v $(PWD):/docs squidfunk/mkdocs-material:latest serve -a 0.0.0.0:8000


