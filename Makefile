REPO_NAME ?= $(shell basename `git rev-parse --show-toplevel`)
IMAGE_NAME ?= ghcr.io/cloudzero/cloudzero-insights-controller/cloudzero-insights-controller

# Docker is the default container tool (and buildx buildkit)
CONTAINER_TOOL ?= docker
BUILDX_CONTAINER_EXISTS := $(shell $(CONTAINER_TOOL) buildx ls --format "{{.Name}}: {{.DriverEndpoint}}" | grep -c "container:")

BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
REVISION ?= $(shell git rev-parse HEAD)
TAG ?= dev-$(REVISION)

# Directories
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

.PHONY: fmt
fmt: ## Run go fmt against code
	@go fmt ./...

.PHONY: lint
lint: ## Run the linter 
	@golangci-lint run

.PHONY: vet
vet: ## Run go vet against code
	@go vet ./...

.PHONY: build
build: ## Build the binary
	@mkdir -p bin
	@CGO_ENABLED=1 go build \
		-mod=readonly \
		-trimpath \
		-ldflags="-s -w -X github.com/cloudzero/cloudzero-insights-controller/pkg/build.Time=${BUILD_TIME} -X github.com/cloudzero/cloudzero-insights-controller/pkg/build.Rev=${REVISION} -X github.com/cloudzero/cloudzero-insights-controller/pkg/build.Tag=${TAG}" \
		-o bin/cloudzero-insights-controller \
		cmd/cloudzero-insights-controller/*.go

.PHONY: clean
clean: ## Clean the binary
	@rm -rf bin log.json certs

.PHONY: test
test: ## Run the unit tests
	@go test -timeout 60s ./... -race -cover


.PHONY: test-integration
test-integration: ## Run the integration tests
	@go test -tags=integration -timeout 60s -race ./... 

.PHONY: package
package:  ## Builds the Docker image
ifeq ($(BUILDX_CONTAINER_EXISTS), 0)
	@$(CONTAINER_TOOL) buildx create --name container --driver=docker-container --use
endif
	@$(CONTAINER_TOOL) buildx build \
		--builder=container \
		--platform linux/amd64,linux/arm64 \
		--build-arg REVISION=$(REVISION) \
		--build-arg TAG=$(TAG) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--push -t $(IMAGE_NAME):$(TAG) -f docker/Dockerfile .
	echo -e "$(INFO_COLOR)Image $(IMAGE_NAME):$(TAG) built and pushed successfully$(NO_COLOR)"

.PHONY: deploy-admission-controller
deploy-admission-controller: ## Deploy the admission controller
	@bash scripts/deploy-admission-controller.sh

.PHONY: undeploy-admission-controller
undeploy-admission-controller: ## Undeploy the admission controller
	@bash scripts/undeploy-admission-controller.sh


.PHONY: deploy-test-app
deploy-test-app: ## Deploy the test app
	@bash scripts/deploy-test-app.sh

.PHONY: undeploy-test-app
undeploy-test-app: ## Undeploy the test app
	@bash scripts/undeploy-test-app.sh


# ----------- MOCK GENERATION ------------
# ----------- MOCK GENERATION ------------
# Define the mockgen tool (ensure it's installed)
MOCKGEN := go.uber.org/mock/mockgen@latest

# Define directories
TYPES_DIR := pkg/types
MOCKS_DIR := $(TYPES_DIR)/mocks

# Find all .go files in TYPES_DIR, excluding the mocks directory and test files
SOURCE_FILES := $(wildcard $(TYPES_DIR)/*.go)
SOURCE_FILES := $(filter-out $(MOCKS_DIR)/*.go, $(SOURCE_FILES))
SOURCE_FILES := $(filter-out %_test.go, $(SOURCE_FILES))

# Define mock destination files with _mock.go suffix
MOCK_DEST_FILES := $(patsubst $(TYPES_DIR)/%.go, $(MOCKS_DIR)/%_mock.go, $(SOURCE_FILES))

.PHONY: generate-mocks ## Generate mocks for all Go files in types directory
generate-mocks: $(MOCK_DEST_FILES)

# Pattern rule to generate a mock for each source file
$(MOCKS_DIR)/%_mock.go: $(TYPES_DIR)/%.go | $(MOCKS_DIR)
	@echo "Generating mock for $<"
	@go run $(MOCKGEN) -source=$< -destination=$@ -package=mocks

# Ensure the mocks directory exists
$(MOCKS_DIR):
	mkdir -p $(MOCKS_DIR)

.PHONY: clean-mocks ## Delete all generated mocks
clean-mocks:
	rm -f $(MOCKS_DIR)/*_mock.go
