# Allow overriding local variables by setting them in local-config.mk
-include local-config.mk

REPO_NAME ?= $(shell basename `git rev-parse --show-toplevel`)
IMAGE_PREFIX ?= ghcr.io/cloudzero/$(REPO_NAME)
IMAGE_NAME ?= $(IMAGE_PREFIX)/$(REPO_NAME)

BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
REVISION ?= $(shell git rev-parse HEAD)
TAG ?= dev-$(REVISION)

# Directories
OUTPUT_BIN_DIR ?= bin

# Dependency executables
#
# These are dependencies that are expected to be installed system-wide. For
# tools we install via `make install-tools` there is no need to allow overriding
# the path to the executable.
AWK    ?= awk
CURL   ?= curl
DOCKER ?= docker
GO     ?= go
GREP   ?= grep
NPM    ?= npm
RM     ?= rm
XARGS  ?= xargs

# Colors
ERROR_COLOR ?= \033[1;31m
INFO_COLOR  ?= \033[1;32m
WARN_COLOR  ?= \033[1;33m
NO_COLOR    ?= \033[0m	

# Docker is the default container tool (and buildx buildkit)
CONTAINER_TOOL ?= $(DOCKER)
BUILDX_CONTAINER_EXISTS := $(shell $(CONTAINER_TOOL) buildx ls --format "{{.Name}}: {{.DriverEndpoint}}" | grep -c "container:")

.DEFAULT_GOAL := help

# Help target to list all available targets with descriptions
.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z_-]+:.*##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

# ----------- CLEANUP ------------

CLEANFILES ?= \
	$(NULL)

MAINTAINER_CLEANFILES ?= \
	$(NULL)

.PHONY: clean
clean: ## Remove build artifacts
	@$(RM) -rf $(CLEANFILES)

.PHONY: maintainer-clean
maintainer-clean: ## Remove build artifacts and maintainer-specific files
maintainer-clean: clean
	@$(RM) -rf $(MAINTAINER_CLEANFILES)

# ----------- DEVELOPMENT TOOL INSTALLATION ------------

ifeq ($(shell uname -s),Darwin)
export SHELL:=env PATH="$(PWD)/.tools/bin:$(PWD)/.tools/node_modules/.bin:$(PATH)" $(SHELL)
else
export PATH := $(PWD)/.tools/bin:$(PWD)/.tools/node_modules/.bin:$(PATH)
endif

MAINTAINER_CLEANFILES += \
	.tools/bin \
	.tools/node_modules/.bin \
	$(NULL)

.PHONY: install-tools
install-tools: ## Install development tools

.PHONY: install-tools-go
install-tools: install-tools-go
install-tools-go:
	@$(GREP) -E '^	_' tools.go | $(AWK) '{print $$2}' | GOBIN=$(PWD)/.tools/bin $(XARGS) $(GO) install

.PHONY: install-tools-node
install-tools: install-tools-node
install-tools-node:
	@$(NPM) install --prefix ./.tools

# golangci-lint is intentionally not installed via tools.go; see
# https://golangci-lint.run/welcome/install/#install-from-sources for details.
GOLANGCI_LINT_VERSION ?= v1.62.2
.PHONY: install-tools-golangci-lint
install-tools: install-tools-golangci-lint
install-tools-golangci-lint: install-tools-go
	@$(CURL) -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b .tools/bin $(GOLANGCI_LINT_VERSION)

# ----------- STANDARDS & PRACTICES ------------

.PHONY: format
format: ## Run go fmt against code

.PHONY: format-go
format: format-go
format-go:
	@gofumpt -w .
	@$(GO) mod tidy

.PHONY: format-prettier
format: format-prettier
format-prettier:
	@prettier --write .

.PHONY: lint
lint: ## Run the linter 
	@golangci-lint run

.PHONY: analyze
analyze: ## Run static analysis
	@staticcheck -checks all ./...

# ----------- COMPILATION ------------

.PHONY: build
build: ## Build the binaries

define generate-go-command-target
.PHONY: build-$1
build: build-$1
build-$1:
	@mkdir -p bin
	@CGO_ENABLED=0 go build \
		-mod=readonly \
		-trimpath \
		-ldflags="-s -w -X github.com/cloudzero/$(REPO_NAME)/pkg/build.Time=${BUILD_TIME} -X github.com/cloudzero/$(REPO_NAME)/pkg/build.Rev=${REVISION} -X github.com/cloudzero/$(REPO_NAME)/pkg/build.Tag=${TAG}" \
		${GO_BUILD_FLAGS} \
		-o ${OUTPUT_BIN_DIR}/$1 \
		./cmd/$1/
CLEANFILES += $(OUTPUT_BIN_DIR)/$1
endef

GO_COMMAND_TARGETS = \
	cloudzero-agent-validator \
	cloudzero-agent-inspector \
	$(NULL)

$(eval $(foreach target,$(GO_COMMAND_TARGETS),$(call generate-go-command-target,$(target))))

CLEANFILES += \
	log.json \
	certs \
	$(NULL)

# ----------- TESTING ------------

.PHONY: test
test: ## Run the unit tests
	@$(GO) test -timeout 60s ./... -race -cover

.PHONY: test-integration
test-integration: ## Run the integration tests
	@$(GO) test -tags=integration -timeout 60s -race ./... 

# ----------- DOCKER IMAGE ------------

define generate-container-build-target
.PHONY: $1
$1:
ifeq ($(BUILDX_CONTAINER_EXISTS), 0)
	@$(CONTAINER_TOOL) buildx create --name container --driver=docker-container --use
endif
	@$(CONTAINER_TOOL) buildx build \
		--builder=container \
		--platform linux/amd64,linux/arm64 \
		--build-arg REVISION=$(REVISION) \
		--build-arg TAG=$(TAG) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--$2 -t $(IMAGE_NAME):$(TAG) -f docker/Dockerfile .
	echo -e "$(INFO_COLOR)Image $(IMAGE_NAME):$(TAG) built successfully$(NO_COLOR)"
endef

package: ## Build and push the Docker image
$(eval $(call generate-container-build-target,package,push))

package-build: ## Build the Docker image
$(eval $(call generate-container-build-target,package-build,load))

# ----------- CODE GENERATION ------------

.PHONY: generate
generate: ## (Re)generate generated code
	@$(GO) generate ./...
