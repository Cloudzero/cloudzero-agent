REPO_NAME ?= $(shell basename `git rev-parse --show-toplevel`)
IMAGE_NAME ?= ghcr.io/cloudzero/cloudzero-agent-validator/cloudzero-agent-validator
TAG ?= dev-$(git rev-parse --short HEAD)

# Docker is the default container tool (and buildx buildkit)
CONTAINER_TOOL ?= docker
BUILDX_CONTAINER_EXISTS := $(shell $(CONTAINER_TOOL) buildx ls --format "{{.Name}}: {{.DriverEndpoint}}" | grep -c "container:")

BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
REVISION ?= $(shell git rev-parse HEAD)

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
	@CGO_ENABLED=0 go build \
		-mod=readonly \
		-trimpath \
		-ldflags="-s -w -X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Time=${BUILD_TIME} -X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Rev=${REVISION} -X github.com/cloudzero/cloudzero-agent-validator/pkg/build.Tag=${TAG}" \
		-o bin/cloudzero-agent-validator \
		cmd/cloudzero-agent-validator/main.go

.PHONY: clean
clean: ## Clean the binary
	@rm -rf bin log.json

.PHONY: test
test: ## Run the unit tests
	@go test -timeout 60s ./... -race -cover

.PHONY: login
login: ## Docker login to GHCR
	@echo $(GHCR_PAT) | $(CONTAINER_TOOL) login ghcr.io -u $(GHCR_USER) --password-stdin

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

.PHONY: generate
generate: ## Generate the status protobuf definition package
	$(MAKE) -C $(CURDIR)/pkg/status generate