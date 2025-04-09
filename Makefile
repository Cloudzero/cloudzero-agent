# Allow overriding local variables by setting them in local-config.mk
-include local-config.mk

# Dependency executables
#
# These are dependencies that are expected to be installed system-wide. For
# tools we install via `make install-tools` there is no need to allow overriding
# the path to the executable.
GO     ?= go
AWK    ?= awk
CC     ?= $(shell $(GO) env CC)
CXX    ?= $(shell $(GO) env CXX)
CURL   ?= curl
DOCKER ?= docker
GREP   ?= grep
HELM   ?= helm
NPM    ?= npm
PROTOC ?= protoc
RM     ?= rm
XARGS  ?= xargs

# Build configuration
GO_MODULE      ?= $(shell $(GO) list -m)
IMAGE_PREFIX   ?= $(subst github.com,ghcr.io,$(GO_MODULE))
IMAGE_NAME     ?= $(IMAGE_PREFIX)/$(notdir $(GO_MODULE))
BUILD_TIME     ?= $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
REVISION       ?= $(shell git rev-parse HEAD)
TAG            ?= dev-$(REVISION)
OUTPUT_BIN_DIR ?= bin

# Default Helm configuration
CLOUDZERO_HOST   ?= dev-api.cloudzero.com
CLOUD_ACCOUNT_ID ?= "12345"
CSP_REGION       ?= "us-east-1"
CLUSTER_NAME     ?= "insights-controller-integration-test"

# Colors
ERROR_COLOR ?= \033[1;31m
INFO_COLOR  ?= \033[1;32m
WARN_COLOR  ?= \033[1;33m
NO_COLOR    ?= \033[0m	

# Docker is the default container tool (and buildx buildkit)
CONTAINER_TOOL ?= $(shell command -v $(DOCKER) 2>/dev/null)
ifdef CONTAINER_TOOL
BUILDX_CONTAINER_EXISTS := $(shell $(CONTAINER_TOOL) buildx ls --format "{{.Name}}: {{.DriverEndpoint}}" | grep -c "container:")
endif

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
GOLANGCI_LINT_VERSION ?= v1.64.4
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

TARGET_OS      ?= $(shell go env GOOS)
TARGET_ARCH    ?= $(shell go env GOARCH)

# The name of the architecture used by the toolchain often doesn't match the
# name of the architecture in GOARCH. This maps from the GOARCH name to the
# toolchain name. For additional details about the various architectures
# supported by go (i.e., GOARCH values), see:
# https://gist.github.com/asukakenji/f15ba7e588ac42795f421b48b8aede63
ifeq ($(TARGET_ARCH),amd64)
  TOOLCHAIN_ARCH ?= x86_64
else ifeq ($(TARGET_ARCH),arm64)
  TOOLCHAIN_ARCH ?= aarch64
else
  TOOLCHAIN_ARCH ?= $(TARGET_ARCH)
endif

ifeq ($(ENABLE_ZIG),true)
  TOOLCHAIN_CC  ?= "zig cc  -target $(TOOLCHAIN_ARCH)-$(TARGET_OS)-musl"
  TOOLCHAIN_CXX ?= "zig c++ -target $(TOOLCHAIN_ARCH)-$(TARGET_OS)-musl"
else
  TOOLCHAIN_CC  ?= $(CC)
  TOOLCHAIN_CXX ?= $(CXX)
endif

define generate-go-command-target
build: $(OUTPUT_BIN_DIR)/cloudzero-$(notdir $1)

.PHONY: $(OUTPUT_BIN_DIR)/cloudzero-$(notdir $1)
$(OUTPUT_BIN_DIR)/cloudzero-$(notdir $1):
	@mkdir -p $(OUTPUT_BIN_DIR)
	GOOS=$(TARGET_OS) GOARCH=$(TARGET_ARCH) \
	CC=$(TOOLCHAIN_CC) CXX=$(TOOLCHAIN_CXX) \
	CGO_ENABLED=1 \
	$(GO) build \
		-mod=readonly \
		-trimpath \
		-ldflags="-s -w -X $(GO_MODULE)/pkg/build.Time=$(BUILD_TIME) -X $(GO_MODULE)/app/build.Rev=$(REVISION) -X $(GO_MODULE)/app/build.Tag=$(TAG) -X $(GO_MODULE)/pkg/build.Time=$(BUILD_TIME) -X $(GO_MODULE)/pkg/build.Rev=$(REVISION) -X $(GO_MODULE)/pkg/build.Tag=$(TAG)" \
		-tags 'netgo osusergo' \
		-o $$@ \
		./$1/

endef

GO_BINARY_DIRS = \
	cmd \
	app/functions \
	$(NULL)

GO_COMMAND_PACKAGE_DIRS = \
	$(foreach parent_dir,$(GO_BINARY_DIRS),$(foreach src_dir,$(wildcard $(parent_dir)/*/),$(patsubst %/,%,$(src_dir)))) \
	$(NULL)

GO_BINARIES = \
	$(foreach bin,$(GO_COMMAND_PACKAGE_DIRS),$(OUTPUT_BIN_DIR)/cloudzero-$(notdir $(bin))) \
	$(NULL)

$(eval $(foreach target,$(GO_COMMAND_PACKAGE_DIRS),$(call generate-go-command-target,$(target))))

CLEANFILES += $(GO_BINARIES)

CLEANFILES += \
	log.json \
	certs \
	$(NULL)

# ----------- TESTING ------------

.PHONY: api-tests-check-env
api-tests-check-env:
	@test -z "$(CLOUDZERO_DEV_API_KEY)" && echo "CLOUDZERO_DEV_API_KEY is not set but is required for smoke tests. Consider adding to local-config.mk." && exit 1 || true

.PHONY: test
test: ## Run the unit tests
	$(GO) test -test.short -timeout 60s ./... -race -cover

.PHONY: test-integration
test-integration: api-tests-check-env
test-integration: ## Run the integration tests
	@CLOUDZERO_HOST=$(CLOUDZERO_HOST) \
	CLOUDZERO_DEV_API_KEY=$(CLOUDZERO_DEV_API_KEY) \
	CLOUD_ACCOUNT_ID=$(CLOUD_ACCOUNT_ID) \
	CSP_REGION=$(CSP_REGION) \
	CLUSTER_NAME=$(CLUSTER_NAME) \
	$(GO) test -run Integration -timeout 60s -race ./...

.PHONY: test-smoke
test-smoke: api-tests-check-env
test-smoke: ## Run the smoke tests
	@CLOUDZERO_HOST=$(CLOUDZERO_HOST) \
	CLOUDZERO_DEV_API_KEY=$(CLOUDZERO_DEV_API_KEY) \
	CLOUD_ACCOUNT_ID=$(CLOUD_ACCOUNT_ID) \
	CSP_REGION=$(CSP_REGION) \
	CLUSTER_NAME=$(CLUSTER_NAME) \
	$(GO) test -run Smoke -v -timeout 10m ./tests/smoke/...

# ----------- DOCKER IMAGE ------------

define generate-container-build-target
.PHONY: $1
$1:
ifeq ($(BUILDX_CONTAINER_EXISTS), 0)
	$(CONTAINER_TOOL) buildx create --name container --driver=docker-container --use
endif
	$(CONTAINER_TOOL) buildx build \
		--progress=plain \
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

# ----------- HELM CHART ------------

.PHONY: helm-lint
helm-lint: ## Lint the Helm chart
	@$(HELM) lint \
		--set-string cloudAccountId="\"$(CLOUD_ACCOUNT_ID)\"" \
		--set-string clusterName="$(CLUSTER_NAME)" \
		--set-string region="$(CSP_REGION)" \
		--set-string host="$(CLOUDZERO_HOST)" \
		--set-string apiKey="$(CLOUDZERO_DEV_API_KEY)" \
		helm/

lint: helm-lint

# ----------- CODE GENERATION ------------

.PHONY: generate
generate: ## (Re)generate generated code
	@$(GO) generate ./...

# We don't yet have a good way to install a specific version of protoc /
# protoc-gen-go, so for now we'll keep this out of the automatic regeneration
# path. If you want to regenerate it using the system protoc, manually remove
# pkg/status/cluster_status.pb.go, then run `make generate`.
generate: pkg/status/cluster_status.pb.go
pkg/status/cluster_status.pb.go: pkg/status/cluster_status.proto
	@$(PROTOC) --proto_path=$(dir $@) --go_out=$(dir $<) pkg/status/cluster_status.proto
