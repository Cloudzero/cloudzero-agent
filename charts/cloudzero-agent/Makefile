# Allow for overrriding of these values
REPO_NAME ?= $(shell basename `git rev-parse --show-toplevel`)
IMAGE_NAME ?= ghcr.io/cloudzero/$(REPO_NAME)/cloudzero-agent-validator
TAG ?= latest

# Docker is the default container tool (and buildx buildkit)
CONTAINER_TOOL ?= docker
BUILDX_CONTAINER_EXISTS := $(shell $(CONTAINER_TOOL) buildx ls --format "{{.Name}}: {{.DriverEndpoint}}" | grep -c "container:")

# Directories
SRC_DIR := src
TEST_DIR := tests
VIRTUALENV := venv
REQUIREMENTS := $(SRC_DIR)/requirements.txt
TEST_REQUIREMENTS := $(TEST_DIR)/requirements.txt

# Colors
ERROR_COLOR = \033[1;31m
INFO_COLOR = \033[1;32m
WARN_COLOR = \033[1;33m
NO_COLOR = \033[0m

# Help target to list all available targets with descriptions
.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z_-]+:.*##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# Checks your active version of python
.PHONY: check-python-version
PYTHON_VERSION_SUPPORTED := 3.8 3.9 3.10 3.11 3.12
PYTHON_VERSION_SHELL := $(shell python --version | cut -f2 -d' ' | cut -f1,2 -d'.')
check-python-version: 
ifeq ($(findstring $(PYTHON_VERSION_SHELL),$(PYTHON_VERSION_SUPPORTED)),)
	@printf "$(WARN_COLOR)You have python $(ERROR_COLOR)$(PYTHON_VERSION_SHELL)$(WARN_COLOR); you need python one of $(INFO_COLOR)($(PYTHON_VERSION_SUPPORTED))$(NO_COLOR).\n"
	@exit 1
endif

# Create a virtual environment
.PHONY: $(VIRTUALENV)
$(VIRTUALENV): check-python-version ## creates a virtual environment
	@python -m venv $(VIRTUALENV)

# Initialize the virtual environment
PYTHON_ARGS := . $(VIRTUALENV)/bin/activate && PYTHONPATH=.
LIBS := $(VIRTUALENV)/lib/python$(PYTHON_VERSION_SHELL)/site-packages
.PHONY: init
init: $(LIBS) ## initializes a virtual environment, and installs the requirements
$(LIBS): $(VIRTUALENV) $(REQUIREMENTS)
	@$(PYTHON_ARGS) pip install -r $(REQUIREMENTS)
	@touch $(LIBS)

# Install development dependencies
.PHONY: init-dev
init-dev: check-python-version init ## Install development dependencies
	@$(PYTHON_ARGS) pip install -r $(TEST_REQUIREMENTS)
	@touch $(LIBS)

# Docker login to GHCR
.PHONY: login
login: ## Docker login to GHCR
	@echo $(GHCR_PAT) | $(CONTAINER_TOOL) login ghcr.io -u $(GHCR_USER) --password-stdin

# Build the Docker image
.PHONY: build
build: check-python-version ## Build the Docker image
ifeq ($(BUILDX_CONTAINER_EXISTS), 0)
	@$(CONTAINER_TOOL) buildx create --name container --driver=docker-container --use
endif
	@$(CONTAINER_TOOL) buildx build --builder=container --platform linux/amd64,linux/arm64 --push -t $(IMAGE_NAME):$(TAG) -f $(SRC_DIR)/Dockerfile $(SRC_DIR)

# Clean up any dangling images
.PHONY: clean
clean: ## Clean up dangling Docker images
	@rm -fr $(VIRTUALENV)
	@find src -name __pycache__ -exec rm -rf {} +
	@find tests -name .pytest_cache -exec rm -rf {} +

# Run unit tests
.PHONY: fmt
fmt: init-dev ## runs code formatter
	@$(PYTHON_ARGS) black $(SRC_DIR)/*.py $(TEST_DIR)/*.py

.PHONY: fmt-check
fmt-check: fmt ## Check if the code is properly formatted
	$(eval CHANGED := $(shell git status -s | grep "\.py" | wc -c))
	@[ $(CHANGED) -eq 0 ] || { echo "changed files"; exit 1; }

.PHONY: lint
lint: init-dev ## runs code linter
	@$(PYTHON_ARGS) ruff check $(SRC_DIR) $(TEST_DIR)

.PHONY: check
check: fmt-check lint test ## Run code linter and unit tests

.PHONY: test
test: check-python-version init-dev ## Run unit tests
	@$(PYTHON_ARGS) pytest -v $(TEST_DIR)
