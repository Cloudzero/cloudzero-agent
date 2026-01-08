# Allow overriding local variables by setting them in local-config.mk
-include local-config.mk

# -----------------------------------------------------------------------------
# Verbosity Control
# -----------------------------------------------------------------------------
# Default to silent mode. Override with 'make V=1'
V ?= 0

# $(Q) is the silence prefix.
# If V=0, Q=@ (silences command). If V=1, Q is empty (shows command).
_Q_0 := @
_Q_1 :=
Q = $(_Q_$(V))

# $(LOG) prints a pretty status message only when V=0.
# Usage: $(call LOG, ACTION, target_name)
_LOG_0 = @printf "  %-12s %s\n" "$(1)" "$(2)"
_LOG_1 = @:
LOG = $(call _LOG_$(V),$(1),$(2))

# $(call VPRINT,<command>) - Print command only in verbose mode
# Use before always-silenced commands to show a (possibly sanitized) version
_VPRINT_0 :=
_VPRINT_1 = @echo '$(1)';
VPRINT = $(_VPRINT_$(V))

# Placeholder for masked secrets in verbose output
MASKED_SECRET := ********

# Dependency executables
#
# These are dependencies that are expected to be installed system-wide. For
# tools we install via `make install-tools` there is no need to allow overriding
# the path to the executable.
GO          ?= go
AWK         ?= awk
CC          ?= $(shell $(GO) env CC)
CXX         ?= $(shell $(GO) env CXX)
CHECKOV     ?= checkov
CURL        ?= curl
DOCKER      ?= docker
GREP        ?= grep
KUBECTL     ?= kubectl
NPM         ?= npm
NPM_INSTALL ?= install
PROTOC      ?= protoc
RM          ?= rm
XARGS       ?= xargs

# Dependencies we install via `make install-tools`
DYFF          ?= .tools/bin/dyff
GOFUMPT       ?= .tools/bin/gofumpt
GOJQ          ?= .tools/bin/gojq
GOLANGCI_LINT ?= .tools/bin/golangci-lint
ACTIONLINT    ?= .tools/bin/actionlint
HELM          ?= .tools/bin/helm
KIND          ?= .tools/bin/kind
KUBECONFORM   ?= .tools/bin/kubeconform
KUTTL         ?= .tools/bin/kubectl-kuttl
MARKDOWNLINT  ?= .tools/node_modules/.bin/markdownlint-cli2
MMDC          ?= .tools/node_modules/.bin/mmdc
MOCKGEN       ?= .tools/bin/mockgen
PRETTIER      ?= .tools/node_modules/.bin/prettier
STATICCHECK   ?= .tools/bin/staticcheck

# Build configuration
GO_MODULE      ?= $(shell $(GO) list -m)
IMAGE_PREFIX   ?= $(subst github.com,ghcr.io,$(GO_MODULE))
IMAGE_NAME     ?= $(IMAGE_PREFIX)/$(notdir $(GO_MODULE))
BUILD_TIME     ?= $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
REVISION       ?= $(shell git rev-parse HEAD)
TAG            ?= dev-$(REVISION)
OUTPUT_BIN_DIR ?= bin
GIT_ROOT       ?= $(shell git rev-parse --show-toplevel)
REGENERATE     ?= auto

# Default Helm configuration
CLOUDZERO_HOST        ?= dev-api.cloudzero.com
CLOUD_ACCOUNT_ID      ?= "123456789012"
CSP_REGION            ?= "us-east-1"
HELM_PLUGINS_DIR      := $(PWD)/.tools/helm-plugins
HELM_ENV              ?= HELM_PLUGINS="$(HELM_PLUGINS_DIR)"
HELM_CMD              := $(HELM_ENV) $(HELM)

# Colors
ERROR_COLOR ?= \033[1;31m
INFO_COLOR  ?= \033[1;32m
WARN_COLOR  ?= \033[1;33m
NO_COLOR    ?= \033[0m	

# Docker is the default container tool (and buildx buildkit)
CONTAINER_TOOL ?= $(shell command -v $(DOCKER) 2>/dev/null)
ifdef $(CONTAINER_TOOL)
BUILDX_CONTAINER_EXISTS := $(shell $(CONTAINER_TOOL) buildx ls --format "{{.Name}}: {{.DriverEndpoint}}" | grep -c "container:")
endif

.DEFAULT_GOAL := help

# Help target to list all available targets with descriptions
.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z_-]+:.*##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: FORCE
FORCE:

# ----------- CLEANUP ------------

CLEANFILES ?= \
	$(NULL)

MAINTAINER_CLEANFILES ?= \
	$(NULL)

.PHONY: clean
clean: ## Remove build artifacts
	$(call LOG,CLEAN,build artifacts)
	$(Q)$(RM) -rf $(CLEANFILES)

.PHONY: maintainer-clean
maintainer-clean: ## Remove build artifacts and maintainer-specific files
maintainer-clean: clean
	$(call LOG,CLEAN,maintainer files)
	$(Q)$(RM) -rf $(MAINTAINER_CLEANFILES)

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
	$(call LOG,INSTALL,go tools)
	$(Q)$(GREP) -E '^	_' .tools/tools.go | $(AWK) '{print $$2}' | GOBIN=$(PWD)/.tools/bin $(XARGS) $(GO) -C .tools install

.PHONY: install-tools-node
install-tools: install-tools-node
install-tools-node:
	$(call LOG,INSTALL,node tools)
	$(Q)$(NPM) $(NPM_INSTALL) --prefix ./.tools

# This is for installing tools using Homebrew that we assume are installed
# system-wide.
.PHONY: install-tools-homebrew
install-tools-homebrew: ## Install some tools via Homebrew
install-tools-homebrew:
	$(call LOG,INSTALL,homebrew packages)
	$(Q)brew install \
		checkov \
		npm \
		protoc-gen-go \
		$(NULL)

# golangci-lint is intentionally not installed via tools.go; see
# https://golangci-lint.run/welcome/install/#install-from-sources for details.
GOLANGCI_LINT_VERSION ?= v2.4.0
.PHONY: install-tools-golangci-lint
install-tools: install-tools-golangci-lint
install-tools-golangci-lint: install-tools-go
	$(call LOG,INSTALL,golangci-lint $(GOLANGCI_LINT_VERSION))
	$(Q)$(CURL) -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b .tools/bin $(GOLANGCI_LINT_VERSION)

# Helm unittest plugin installation (pinned to v1.0.2 due to platformHooks bug in v1.0.3)
# See: https://github.com/helm-unittest/helm-unittest/issues/790
HELM_UNITTEST_VERSION := v1.0.2
HELM_UNITTEST_PLUGIN  := $(HELM_PLUGINS_DIR)/helm-unittest/untt

$(HELM_UNITTEST_PLUGIN): | $(HELM)
	$(call LOG,INSTALL,helm-unittest $(HELM_UNITTEST_VERSION))
	$(Q)mkdir -p "$(HELM_PLUGINS_DIR)"
	$(Q)$(HELM_CMD) plugin install https://github.com/helm-unittest/helm-unittest --version $(HELM_UNITTEST_VERSION)

.PHONY: install-tools-helm-unittest
install-tools: install-tools-helm-unittest
install-tools-helm-unittest: $(HELM_UNITTEST_PLUGIN)

# Generate the secrets file used by the `act` tool for local GitHub Action development.
secrets-act:
	@if [[ "$(CLOUDZERO_DEV_API_KEY)" == "" ]] || [[ "$(GITHUB_TOKEN)" == "" ]]; then echo "CLOUDZERO_DEV_API_KEY and GITHUB_TOKEN are required to generate the .github/workflows/.secret file, but at least one of them is not set. Consider adding to local-config.mk."; exit 1; fi
	$(call LOG,GEN,.github/workflows/.secrets)
	$(call VPRINT,echo "CLOUDZERO_DEV_API_KEY=$(MASKED_SECRET)" > $(GIT_ROOT)/.github/workflows/.secrets)
	@echo "CLOUDZERO_DEV_API_KEY=$(CLOUDZERO_DEV_API_KEY)" > $(GIT_ROOT)/.github/workflows/.secrets
	$(call VPRINT,echo "GITHUB_TOKEN=$(MASKED_SECRET)" >> $(GIT_ROOT)/.github/workflows/.secrets)
	@echo "GITHUB_TOKEN=$(GITHUB_TOKEN)" >> $(GIT_ROOT)/.github/workflows/.secrets

# ----------- STANDARDS & PRACTICES ------------

.PHONY: format
format: ## Run go fmt against code

GOFUMPT_TARGET        ?= .

.PHONY: format-go
format: format-go
format-go:
	$(call LOG,FORMAT,go)
	$(Q)$(GOFUMPT) -w $(GOFUMPT_TARGET)
	$(Q)$(GO) mod tidy
	$(Q)$(GO) -C .tools/ mod tidy
	$(Q)$(GO) -C tests/ mod tidy

PRETTIER_TARGET       ?= .

.PHONY: format-prettier
format: format-prettier
format-prettier:
	$(call LOG,FORMAT,prettier)
	$(Q)$(PRETTIER) --write $(PRETTIER_TARGET) | grep -v '(unchanged)$$' || true

.PHONY: lint-go
lint-go:
	$(call LOG,LINT,go)
	$(Q)$(GOLANGCI_LINT) run ./...

.PHONY: %.md-lint-mermaid
%.md-lint-mermaid: %.md
	$(Q)if grep -q '```mermaid' "$<" 2>/dev/null; then \
		echo "  LINT         mermaid: $<"; \
		awk '/```mermaid/,/```/' "$<" | awk 'BEGIN{RS="```mermaid"; FS="```"} NR>1 {print $$1}' | while read -r diagram; do \
			if [ -n "$$diagram" ]; then \
				echo "$$diagram" | $(MMDC) -i - -e svg -o - --quiet --puppeteerConfigFile .tools/puppeteer-config.json >/dev/null || { \
					echo "$(ERROR_COLOR)Mermaid diagram validation failed in $<$(NO_COLOR)"; \
					exit 1; \
				}; \
			fi; \
		done; \
	fi

.PHONY: lint-mermaid
lint-mermaid: ## Run Mermaid diagram validation on all Markdown files
lint-mermaid: $(patsubst %.md,%.md-lint-mermaid,$(shell find . -name '*.md' -not -exec git check-ignore -q {} \; -print 2>/dev/null | sort -u | xargs grep -l '```mermaid' 2>/dev/null))

.PHONY: lint-markdown
lint-markdown: ## Run markdownlint on all Markdown files
	$(call LOG,LINT,markdown)
	$(Q)$(MARKDOWNLINT)

.PHONY: lint-action
lint-action: ## Run actionlint on GitHub Actions workflows
	$(call LOG,LINT,actions)
	$(Q)$(ACTIONLINT) -verbose

.PHONY: lint
lint: ## Run the linter
lint: lint-go lint-mermaid lint-markdown lint-action

.PHONY: analyze-go
analyze-go:
	$(call LOG,ANALYZE,go)
	$(Q)$(STATICCHECK) -checks all ./...

.PHONY: analyze
analyze: ## Run static analysis
analyze: analyze-go

.PHONY: analyze-staticcheck
analyze-staticcheck:
	$(call LOG,ANALYZE,staticcheck)
	$(Q)staticcheck -checks all ./...

.PHONY: analyze-checkov
analyze-checkov: $(addsuffix -analyze-checkov,$(wildcard tests/helm/template/*.yaml))
analyze: analyze-checkov

.PHONY: tests/helm/template/%.yaml-analyze-checkov
tests/helm/template/%.yaml-analyze-checkov: tests/helm/template/%.yaml
	$(call LOG,ANALYZE,checkov: $<)
	$(Q)checkov -f $< \
		--compact \
		--quiet \
		--framework kubernetes \
		$(NULL)

helm-test: analyze-checkov

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
	$(call LOG,GO,$$@)
	$(Q)mkdir -p $(OUTPUT_BIN_DIR)
	$(Q)GOOS=$(TARGET_OS) GOARCH=$(TARGET_ARCH) \
	CC=$(TOOLCHAIN_CC) CXX=$(TOOLCHAIN_CXX) \
	CGO_ENABLED=1 \
	$(GO) build \
		-mod=readonly \
		-trimpath \
		-ldflags="-s -w -X $(GO_MODULE)/app/build.Rev=$(REVISION) -X $(GO_MODULE)/app/build.Tag=$(TAG) -X $(GO_MODULE)/app/build.Time=$(BUILD_TIME)" \
		-tags 'netgo osusergo' \
		-o $$@ \
		./$1/

endef

GO_BINARY_DIRS = \
	cmd \
	app/functions \
	$(NULL)

GO_COMMAND_PACKAGE_DIRS = \
	$(patsubst %/,%,$(filter %/,$(foreach parent_dir,$(GO_BINARY_DIRS),$(wildcard $(parent_dir)/*/)))) \
	$(NULL)

GO_BINARIES = \
	$(foreach bin,$(GO_COMMAND_PACKAGE_DIRS),$(OUTPUT_BIN_DIR)/cloudzero-$(notdir $(bin))) \
	$(NULL)

# Generate embedded defaults for helmless (conditional on REGENERATE setting)
ifneq ($(REGENERATE),never)
app/functions/helmless/default-values.yaml: helm/values.yaml $(wildcard helm/*.yaml helm/templates/*.yaml helm/templates/*.tpl helm/*.yaml)
	$(call LOG,GEN,$@)
	$(Q)$(HELM_CMD) show values ./helm | $(PRETTIER) --stdin-filepath $@ > $@

bin/cloudzero-helmless: app/functions/helmless/default-values.yaml

# Add the embedded defaults file to dependencies
$(OUTPUT_BIN_DIR)/cloudzero-helmless: app/functions/helmless/default-values.yaml

generate: app/functions/helmless/default-values.yaml
endif

MAINTAINER_CLEANFILES += app/functions/helmless/default-values.yaml

$(eval $(foreach target,$(GO_COMMAND_PACKAGE_DIRS),$(call generate-go-command-target,$(target))))

CLEANFILES += $(GO_BINARIES)

CLEANFILES += \
	log.json \
	certs \
	$(NULL)

# ----------- TESTING ------------

.PHONY: api-tests-check-env
api-tests-check-env:
	@test -z "$(CLOUDZERO_DEV_API_KEY)" && echo "CLOUDZERO_DEV_API_KEY is not set but is required for smoke tests and helm chart installation. Consider adding to local-config.mk." && exit 1 || true

GO_TEST_TARGET        ?= ./...

.PHONY: test
test: ## Run the unit tests
	$(call LOG,TEST,unit)
	$(Q)$(GO) test -test.short -timeout 120s $(GO_TEST_TARGET) -race -cover

.PHONY: test-integration
test-integration: api-tests-check-env
test-integration: ## Run the integration tests
	$(call LOG,TEST,integration)
	$(call VPRINT,CLOUDZERO_HOST=$(CLOUDZERO_HOST) CLOUDZERO_DEV_API_KEY=$(MASKED_SECRET) CLOUD_ACCOUNT_ID=$(CLOUD_ACCOUNT_ID) CSP_REGION=$(CSP_REGION) CLUSTER_NAME=$(CLUSTER_NAME) $(GO) test -run Integration -timeout 60s -race ./...)
	@CLOUDZERO_HOST=$(CLOUDZERO_HOST) \
	CLOUDZERO_DEV_API_KEY=$(CLOUDZERO_DEV_API_KEY) \
	CLOUD_ACCOUNT_ID=$(CLOUD_ACCOUNT_ID) \
	CSP_REGION=$(CSP_REGION) \
	CLUSTER_NAME=$(CLUSTER_NAME) \
	$(GO) test -run Integration -timeout 60s -race ./...

.PHONY: test-smoke
test-smoke: api-tests-check-env
test-smoke: ## Run the smoke tests
	$(call LOG,TEST,smoke)
	$(call VPRINT,CLOUDZERO_HOST=$(CLOUDZERO_HOST) CLOUDZERO_DEV_API_KEY=$(MASKED_SECRET) CLOUD_ACCOUNT_ID=$(CLOUD_ACCOUNT_ID) CSP_REGION=$(CSP_REGION) CLUSTER_NAME=$(CLUSTER_NAME) $(GO) -C tests test -run Smoke -v -timeout 10m ./smoke/...)
	@CLOUDZERO_HOST=$(CLOUDZERO_HOST) \
	CLOUDZERO_DEV_API_KEY=$(CLOUDZERO_DEV_API_KEY) \
	CLOUD_ACCOUNT_ID=$(CLOUD_ACCOUNT_ID) \
	CSP_REGION=$(CSP_REGION) \
	CLUSTER_NAME=$(CLUSTER_NAME) \
	$(GO) -C tests test -run Smoke -v -timeout 10m ./smoke/...

.PHONY: test-all
test-all: ## Run all tests
test-all: helm-test test test-integration kind-test test-smoke

# ----------- LOCAL TESTING INFRASTRUCTURE ------------

# Generic testing configuration
TEST_K8S_VERSION          ?= v1.32.3
TEST_KIND_IMAGE_VERSION   ?= v1.33.2
TEST_PLATFORM             ?= linux/amd64
CLUSTER_NAME              ?= kind

# New unified test cluster system

.PHONY: kind-up
kind-up: # Create kind cluster for testing
kind-up: tests/kuttl/kubeconfig

.PHONY: kind-down
kind-down: ## Delete kind cluster and cleanup
	$(call LOG,KIND,delete cluster $(CLUSTER_NAME))
	$(Q)$(KIND) delete cluster --name $(CLUSTER_NAME) || true
	$(Q)$(RM) -f tests/kuttl/kubeconfig

# Complete KUTTL test workflow
.PHONY: kind-test
kind-test: ## Set up Kind cluster, install chart, run tests, unistall, delete cluster
kind-test:
	$(Q)set -e; \
	kind_up_success=0; \
	helm_install_success=0; \
	helm_test_success=0; \
	\
	$(MAKE) kind-up && kind_up_success=1; \
	if [ "$$kind_up_success" = "1" ]; then \
		$(MAKE) helm-install-current && helm_install_success=1; \
		if [ "$$helm_install_success" = "1" ]; then \
			$(MAKE) helm-test-kuttl && helm_test_success=1; \
		fi; \
	fi; \
	\
	if [ "$$helm_install_success" = "1" ]; then \
		$(MAKE) helm-uninstall || true; \
	fi; \
	if [ "$$kind_up_success" = "1" ]; then \
		$(MAKE) kind-down || true; \
	fi; \
	if [ "$$helm_test_success" = "0" ]; then \
		exit 1; \
	fi


# Kind cluster kubeconfig for testing
tests/kuttl/kubeconfig: ## Create kind cluster kubeconfig for testing
	$(call LOG,KIND,create cluster $(CLUSTER_NAME))
	$(Q)$(KIND) create cluster --name $(CLUSTER_NAME) --image kindest/node:$(TEST_KIND_IMAGE_VERSION)
	$(Q)$(KIND) get kubeconfig --name $(CLUSTER_NAME) > $@
	$(Q)chmod 600 $@
	$(Q)KUBECONFIG="$@" $(KUBECTL) wait --for=condition=Ready nodes --all --timeout=4m
	$(Q)KUBECONFIG="$@" $(KUBECTL) wait --for=condition=Available deployment/coredns --namespace kube-system --timeout=4m

# ----------- DOCKER IMAGE ------------

DEBUG_IMAGE ?= busybox:stable-uclibc

# Define targets for Docker image variants
#
#  $1: Target name (package, package-debug, package-build, package-build-debug)
#  $2: Whether to push to a registry, or only the local Docker (push, load)
#  $3: Whether to build a debug image (true, false)
define generate-container-build-target
.PHONY: $1
$1:
	$(call LOG,DOCKER,$1 $(IMAGE_NAME):$(TAG))
ifeq ($(BUILDX_CONTAINER_EXISTS), 0)
	$(Q)$(CONTAINER_TOOL) buildx create --name container --driver=docker-container --use
endif
	$(Q)$(CONTAINER_TOOL) buildx build \
		--progress=plain \
		--platform linux/amd64,linux/arm64 \
		--build-arg REVISION=$(REVISION) \
		--build-arg TAG=$(TAG) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		$(if $(filter true,$(3)),--build-arg DEPLOY_IMAGE=$(DEBUG_IMAGE),) \
		--$2 -t $(IMAGE_NAME):$(TAG) -f docker/Dockerfile .
endef

package: ## Build and push the Docker image
$(eval $(call generate-container-build-target,package,push,false,false))

package-debug: ## Build and push a debugging version of the Docker image
$(eval $(call generate-container-build-target,package-debug,push,true))

package-build: ## Build the Docker image
$(eval $(call generate-container-build-target,package-build,load,false))

package-build-debug: ## Build a debugging version of the Docker image
$(eval $(call generate-container-build-target,package-build-debug,load,true))

# ----------- HELM CHART ------------

PROMETHEUS_COMMUNITY_REPO  ?= https://prometheus-community.github.io/helm-charts
HELM_SCHEMA_TEST_NAMESPACE ?= cz-agent
HELM_SCHEMA_TEST_TARGET    ?= cz-agent
KUBE_VERSION               ?= 1.33.0

# Cluster configuration system
CLUSTER_CONFIG_NAME       ?= $(CLUSTER_NAME)
CLUSTER_CONFIG_FILE       ?= clusters/$(CLUSTER_CONFIG_NAME).yaml
CLUSTER_OVERRIDES_FILE    ?= clusters/$(CLUSTER_CONFIG_NAME)-overrides.yaml

# Additional helm overrides for specialized configurations (optional)
# Set HELM_EXTRA_OVERRIDES to path of additional values file that takes precedence
# Example: HELM_EXTRA_OVERRIDES=clusters/federated-mode-overrides.yaml
HELM_EXTRA_OVERRIDES      ?=

# =============================================================================
# CLUSTER CONFIGURATION HELPER FUNCTIONS
# =============================================================================
# These functions provide a unified interface for extracting properties from
# cluster configuration files and generating appropriate command-line arguments
# for Kubernetes tools (kubectl, helm, kuttl).

# -----------------------------------------------------------------------------
# Core Property Extraction Functions
# -----------------------------------------------------------------------------

# get-cluster-property - Extract a property from the cluster configuration file
#
# This runs <property> as a gojq query and returns the result.
#
# Usage: $(call get-cluster-property,<query>,<default>)
#
# Arguments:
#   $1: Query (e.g., '.namespace', '.release', '.kubeConfig')
#   $2: Default value if property is missing or null (e.g., '""', '""')
# Returns: the result of the query
#
# Examples:
#   $(call get-cluster-property,.namespace) -> "cza"
define get-cluster-property
$(strip $(shell $(GOJQ) --raw-output --yaml-input '$(1)$(if $(2), // $(2),)' $(CLUSTER_CONFIG_FILE)))
endef

# get-kubeconfig-path - Get the kubeconfig file path from cluster config
#
# Usage: $(call get-kubeconfig-path)
#
# Returns: Path to kubeconfig file or empty string if not specified.  Note that
#   an empty string means use system KUBECONFIG environment variable, falling back
#   on the default (~/.kube/config)
define get-kubeconfig-path
$(call get-cluster-property,.kubeConfig,"$(KUBECONFIG)")
endef

# get-kubeconfig-env - Get KUBECONFIG environment variable setting
#
# Usage: $(call get-kubeconfig-env)
#
# Returns: KUBECONFIG="<path>" if kubeConfig is set in cluster config, empty if null
define get-kubeconfig-env
$(shell $(GOJQ) --raw-output --yaml-input 'if .kubeConfig and .kubeConfig != null then "KUBECONFIG=\"\(.kubeConfig)\"" else "" end' < $(CLUSTER_CONFIG_FILE))
endef

# get-kubectx-arg - Get Kubernetes context flag with value
#
# Usage: $(call get-kubectx-arg,--context)
# Usage: $(call get-kubectx-arg,--kube-context)
#
# Returns: --context <value> or --kube-context <value> if context is set, empty if not
define get-kubectx-arg
$(shell $(GOJQ) --raw-output --yaml-input 'if .context and .context != null then "$(1) " + (.context | @json) else "" end' < $(CLUSTER_CONFIG_FILE))
endef

# -----------------------------------------------------------------------------
# Tool Invocation Helper Functions
# -----------------------------------------------------------------------------
# These functions generate complete command invocations for Kubernetes tools
# with appropriate environment variables and global arguments. They
# automatically handle kubeconfig, context, and namespace based on the cluster
# configuration.

# invoke-kubectl - Generate kubectl command with kubeconfig and context
#
# Usage: $(call invoke-kubectl) get pods
# Returns: KUBECONFIG=<path> kubectl --context <context> [additional args]
define invoke-kubectl
$(call get-kubeconfig-env) $(KUBECTL) $(call get-kubectx-arg,--context)
endef

# get-helm-extra-overrides - Generate additional --values argument if HELM_EXTRA_OVERRIDES is set
#
# This function enables specialized Helm chart configurations by allowing additional values files
# to be layered on top of standard cluster overrides. This is useful for testing specific
# deployment scenarios like federated mode, cert-manager integration, etc.
#
# Usage: $(call get-helm-extra-overrides)
# Returns: "--values <file>" if HELM_EXTRA_OVERRIDES is set and non-empty, empty otherwise
#
# Priority order for values (later takes precedence):
#   1. helm/values.yaml (chart defaults)
#   2. $(CLUSTER_OVERRIDES_FILE) (cluster-specific overrides)
#   3. $(HELM_EXTRA_OVERRIDES) (specialized configuration overrides)
#   4. --set arguments (command-line overrides)
define get-helm-extra-overrides
$(if $(HELM_EXTRA_OVERRIDES),--values $(HELM_EXTRA_OVERRIDES),)
endef

# invoke-helm - Generate helm command with kubeconfig, context, and namespace
# Returns: KUBECONFIG=<path> helm --kube-context <context> --namespace <namespace> [additional args]
# Usage: $(call invoke-helm) install my-release ./chart
# Note: Uses --kube-context (not --context) as required by helm
define invoke-helm
$(call get-kubeconfig-env) $(HELM_CMD) $(call get-kubectx-arg,--kube-context) --namespace "$(call get-cluster-property,.namespace)"
endef

# invoke-kuttl - Generate kuttl command with kubeconfig only
# Returns: KUBECONFIG=<path> kuttl [additional args]
# Usage: $(call invoke-kuttl) test
# Note: KUTTL only supports KUBECONFIG, not --context
define invoke-kuttl
$(call get-kubeconfig-env) $(KUTTL)
endef

# Use a timestamp file to track helm dependency installation
helm/charts/.stamp: helm/Chart.yaml
	$(call LOG,HELM,dependency build)
	$(Q)$(HELM_CMD) repo add --force-update prometheus-community $(PROMETHEUS_COMMUNITY_REPO)
	$(Q)$(HELM_CMD) repo update prometheus-community
	$(Q)$(HELM_CMD) dependency build ./helm
	$(Q)touch helm/charts/.stamp

.PHONY: helm-install-deps
helm-install-deps: helm/charts/.stamp

.PHONY: helm-install
helm-install: api-tests-check-env helm-install-deps $(CLUSTER_CONFIG_FILE) $(CLUSTER_OVERRIDES_FILE)
helm-install: ## Install the Helm chart (uses CLUSTER_CONFIG_NAME)
	$(call LOG,HELM,install $(CLUSTER_CONFIG_NAME))
	$(Q)$(call invoke-helm) upgrade --install "$(call get-cluster-property,.release)" \
		./helm \
		--create-namespace \
		--values $(CLUSTER_OVERRIDES_FILE) \
		$(call get-helm-extra-overrides) \
		$(NULL)

# helm-install-current is the same as helm-install, except that it
# will set the tag for the agent image to dev-$(git rev-parse HEAD).
.PHONY: helm-install-current
helm-install-current: api-tests-check-env helm-install-deps $(CLUSTER_CONFIG_FILE) $(CLUSTER_OVERRIDES_FILE)
helm-install-current: ## Install chart with test image
	$(call LOG,HELM,install $(CLUSTER_CONFIG_NAME) [dev])
	$(Q)$(call invoke-helm) upgrade --install "$(call get-cluster-property,.release)" \
		./helm \
		--create-namespace \
		--values $(CLUSTER_OVERRIDES_FILE) \
		$(call get-helm-extra-overrides) \
		--set components.agent.image.tag=dev-$(shell git rev-parse HEAD) \
		$(NULL)

.PHONY: helm-wait
helm-wait: ## Wait for chart to be ready after installation
	$(call LOG,HELM,wait $(CLUSTER_CONFIG_NAME))
	$(Q)$(call invoke-kubectl) wait --for=condition=Available \
		--namespace $(call get-cluster-property,.namespace) \
		--timeout=5m \
		$(foreach deployment,server webhook aggregator ksm,deployment/$(call get-cluster-property,.release)-cz-$(deployment)) \
		$(NULL)

.PHONY: helm-uninstall
helm-uninstall: $(CLUSTER_CONFIG_FILE) ## Uninstall the Helm chart (uses CLUSTER_CONFIG_NAME)
	$(call LOG,HELM,uninstall $(CLUSTER_CONFIG_NAME))
	$(Q)$(call invoke-helm) uninstall "$(call get-cluster-property,.release)" \
		$(NULL)

.PHONY: helm-lint
helm-lint: helm/values.schema.json $(CLUSTER_CONFIG_FILE) $(CLUSTER_OVERRIDES_FILE)
helm-lint: ## Lint the Helm chart (uses CLUSTER_CONFIG_NAME)
	$(call LOG,HELM,lint)
	$(Q)$(call invoke-helm) lint ./helm \
		--values $(CLUSTER_OVERRIDES_FILE) \
		$(call get-helm-extra-overrides) \
		$(NULL)

# Generate list of all schema test targets (file paths without .yaml extension)
SCHEMA_TEST_FILES := $(wildcard tests/helm/schema/*.yaml)
SCHEMA_TEST_TARGETS := $(SCHEMA_TEST_FILES:.yaml=)
SCHEMA_TEMPLATE_TARGETS := $(addsuffix -template,$(SCHEMA_TEST_TARGETS))
SCHEMA_KUBECONFORM_TARGETS := $(addsuffix -kubeconform,$(filter %pass,$(SCHEMA_TEST_TARGETS)))

# Targets that depend on both template and kubeconform validation
# For .pass tests, depend on both template and kubeconform targets
# For .fail tests, only depend on template target
$(filter %pass,$(SCHEMA_TEST_TARGETS)): %: %-template %-kubeconform

$(filter %fail,$(SCHEMA_TEST_TARGETS)): %: %-template

# Pattern rule for Helm template validation
tests/helm/schema/%-template: tests/helm/schema/%.yaml helm/charts/.stamp helm/values.schema.json
	$(Q)file="tests/helm/schema/$*.yaml"; \
	expected_result=$$(echo "$$file" | grep -q "\.pass\.yaml$$" && echo "pass" || echo "fail"); \
	output=$$($(HELM_CMD) template --kube-version "$(KUBE_VERSION)" "$(HELM_SCHEMA_TEST_TARGET)" ./helm --values "$$file" --set apiKey="not-a-real-key" 2>&1); \
	if [ $$? -eq 0 ]; then \
		result="pass"; \
	else \
		result="fail"; \
	fi; \
	if [ "$$result" = "$$expected_result" ]; then \
		echo "  TEST         ✓ $$file (Helm validation)"; \
	else \
		echo "  TEST         ✗ $$file (expected $$expected_result, got $$result)"; \
		echo "$$output" | grep -E "(Error:|execution error)" | head -5 || echo "$$output" | tail -10; \
		exit 1; \
	fi

# Pattern rule for kubeconform validation (only for .pass tests)
tests/helm/schema/%-kubeconform: tests/helm/schema/%.yaml helm/charts/.stamp helm/values.schema.json
	$(Q)file="tests/helm/schema/$*.yaml"; \
	kubeconform_output=$$($(HELM_CMD) template --kube-version "$(KUBE_VERSION)" "$(HELM_SCHEMA_TEST_TARGET)" ./helm --values "$$file" --set apiKey="not-a-real-key" 2>/dev/null | $(KUBECONFORM) \
		-kubernetes-version "$(KUBE_VERSION)" \
		-schema-location default \
		-schema-location 'https://raw.githubusercontent.com/datreeio/CRDs-catalog/main/{{.Group}}/{{.ResourceKind}}_{{.ResourceAPIVersion}}.json' \
		-strict \
		-summary \
		- 2>&1); \
	kubeconform_exit=$$?; \
	if [ $$kubeconform_exit -eq 0 ]; then \
		echo "  TEST         ✓ $$file (kubeconform validation)"; \
	else \
		echo "  TEST         ✗ $$file (kubeconform validation failed)"; \
		echo "kubeconform output:"; \
		echo "$$kubeconform_output"; \
		exit 1; \
	fi

.PHONY: helm-test-schema
helm-test-schema: ## Run the Helm values schema validation tests
helm-test-schema: helm/charts/.stamp
helm-test-schema: $(SCHEMA_TEST_TARGETS)

.PHONY: helm-test-schema-template
helm-test-schema-template: ## Run only the Helm template validation tests
helm-test-schema-template: helm/charts/.stamp
helm-test-schema-template: $(SCHEMA_TEMPLATE_TARGETS)

.PHONY: helm-test-schema-kubeconform
helm-test-schema-kubeconform: ## Run only the kubeconform validation tests
helm-test-schema-kubeconform: helm/charts/.stamp
helm-test-schema-kubeconform: $(SCHEMA_KUBECONFORM_TARGETS)

.PHONY: helm-test-subchart
helm-test-subchart: ## Run the Helm subchart validation tests
helm-test-subchart: helm/charts/.stamp
helm-test-subchart: helm/values.schema.json
	$(call LOG,TEST,helm subchart)
	$(Q)for dir in tests/helm/subchart/*/; do \
		if [ -d "$$dir/chart" ]; then \
			echo "  HELM         Building dependencies for $$(basename $$dir)..."; \
			$(HELM_CMD) dependency build "$$dir/chart"; \
		fi; \
	done
	$(Q)for dir in tests/helm/subchart/*/; do \
		if [ -d "$$dir/chart" ]; then \
			for file in $$dir*.yaml; do \
				if [ -f "$$file" ]; then \
					expected_result=$$(echo $$file | grep -q "\.pass\.yaml$$" && echo "pass" || echo "fail"); \
					output=$$($(HELM_CMD) template parent-test "$$dir/chart" --values "$$file" 2>&1); \
					if [ $$? -eq 0 ]; then \
						result="pass"; \
					else \
						result="fail"; \
					fi; \
					if [ "$$result" = "$$expected_result" ]; then \
						echo "  TEST         ✓ $$file"; \
					else \
						echo "  TEST         ✗ $$file (expected $$expected_result, got $$result)"; \
						if [ "$$expected_result" = "pass" ]; then \
							echo "Helm command output:"; \
							echo ""; \
							echo "$$output"; \
						fi; \
						exit 1; \
					fi; \
				fi; \
			done; \
		fi; \
	done

helm/tests/%.yaml-unittest: helm/tests/%.yaml $(HELM_UNITTEST_PLUGIN)
	$(Q)$(HELM_CMD) unittest ./helm --values helm/tests/values.yaml --file 'tests/$*.yaml'

.PHONY: helm-test-unittest
helm-test-unittest: ## Run Helm unittest tests
helm-test-unittest: $(HELM_UNITTEST_PLUGIN) helm/charts/.stamp
	$(call LOG,TEST,helm unittest)
	$(Q)$(HELM_CMD) unittest ./helm --values helm/tests/values.yaml

.PHONY: helm-test
helm-test: ## Run all Helm validation tests
helm-test: helm-test-schema helm-test-subchart helm-test-unittest helm-test-template

# Istio template test requires Istio CRD API versions to be available
tests/helm/template/istio.yaml: tests/helm/template/istio-overrides.yml helm/charts/.stamp helm/values.schema.json $(wildcard helm/templates/*.yaml) $(wildcard helm/templates/*.tpl) helm/values.yaml
	$(call LOG,HELM,template $@)
	$(Q)$(HELM_CMD) template --kube-version "$(KUBE_VERSION)" "$(HELM_SCHEMA_TEST_TARGET)" --namespace "$(HELM_SCHEMA_TEST_NAMESPACE)" \
		--api-versions networking.istio.io/v1 \
		./helm --values $< > $@

tests/helm/template/%.yaml: tests/helm/template/%-overrides.yml helm/charts/.stamp helm/values.schema.json $(wildcard helm/templates/*.yaml) $(wildcard helm/templates/*.tpl) helm/values.yaml
	$(call LOG,HELM,template $@)
	$(Q)$(HELM_CMD) template --kube-version "$(KUBE_VERSION)" "$(HELM_SCHEMA_TEST_TARGET)" --namespace "$(HELM_SCHEMA_TEST_NAMESPACE)" ./helm --values $< > $@

helm-test-template: $(patsubst %-overrides.yml,%.yaml,$(wildcard tests/helm/template/*-overrides.yml))
generate: helm-test-template

.PHONY: tests/kuttl/%/run
tests/kuttl/%/run: tests/kuttl/%/kuttl-test.yaml
	$(call LOG,KUTTL,$(dir $<))
	$(Q)$(call invoke-kuttl) test --config $< -v 1 $(dir $<)

.PHONY: helm-test-kuttl
helm-test-kuttl: ## Run KUTTL tests. Note that this assumes the infra is in place; see the kind-test target
helm-test-kuttl: $(patsubst tests/kuttl/%/kuttl-test.yaml,tests/kuttl/%/run,$(wildcard tests/kuttl/*/kuttl-test.yaml))


lint: helm-lint

helm/values.schema.json: helm/values.schema.yaml helm/schema/k8s.json scripts/merge-json-schema.jq
	$(call LOG,GEN,$@)
	$(Q)$(GOJQ) --yaml-input . helm/values.schema.yaml | \
		$(GOJQ) --slurpfile k8s helm/schema/k8s.json -f scripts/merge-json-schema.jq | \
		$(PRETTIER) --stdin-filepath "$@" > "$@"

generate: helm/values.schema.json

.PHONY: helm-test-template-diff
helm-test-template-diff: $(patsubst %-overrides.yml,%.yaml-semdiff,$(wildcard tests/helm/template/*-overrides.yml))

# The JSON Schema for Kubernetes. For details, see:
# https://github.com/yannh/kubernetes-json-schema/
K8S_SCHEMA_UPSTREAM ?= https://raw.githubusercontent.com/yannh/kubernetes-json-schema/refs/heads/master/master-standalone-strict/_definitions.json

helm/schema/k8s.json:
	$(call LOG,FETCH,$@)
	$(Q)$(CURL) -sSL "$(K8S_SCHEMA_UPSTREAM)" | $(PRETTIER) --stdin-filepath "$@" > "$@"

generate: helm/schema/k8s.json

# ----------- JSON SEMANTIC DIFF ------------
#
# Compare JSON/YAML files with the version in git using dyff. This is a semantic
# diff tool, so it won't show any differences for things like formatting
# changes, reordering items, etc., making it invaluable when refactoring.
#
# Usage:
#   make path/to/file.json-semdiff
#   make path/to/file.yaml-semdiff
#   make path/to/file.json-semdiff SEMDIFF_REVISION=v1.2.3
#
# Variables:
#   DYFF
#   SEMDIFF_REVISION=<git-ref>       - Git revision to compare with (default: HEAD)

SEMDIFF_REVISION ?= HEAD

# Shared implementation for semantic diff using dyff
define semdiff_impl
$(call LOG,DIFF,$(patsubst %-semdiff,%,$@))
$(Q)temp_file=$$(mktemp); \
git show $(SEMDIFF_REVISION):"$(patsubst %-semdiff,%,$@)" > "$$temp_file" 2>/dev/null || { echo "File $(patsubst %-semdiff,%,$@) not found in $(SEMDIFF_REVISION)"; rm -f "$$temp_file"; exit 1; }; \
$(DYFF) between --set-exit-code "$$temp_file" "$(patsubst %-semdiff,%,$@)"; \
exit_code=$$?; \
rm -f "$$temp_file"; \
exit $$exit_code
endef

.PHONY: %.json-semdiff %.yaml-semdiff %.yml-semdiff

%.json-semdiff: %.json
	$(semdiff_impl)

%.yaml-semdiff: %.yaml
	$(semdiff_impl)

%.yml-semdiff: %.yml
	$(semdiff_impl)

# ----------- CODE GENERATION ------------

.PHONY: generate
generate: ## (Re)generate generated code

# ----------- PROTOBUF GENERATION ------------

# Protobuf files to generate
PROTOBUF_FILES := \
	app/types/status/cluster_status.pb.go \
	app/types/clusterconfig/clusterconfig.pb.go \
	$(NULL)

# We don't yet have a good way to install a specific version of protoc /
# protoc-gen-go, so for now we'll keep this out of the automatic regeneration
# path. If you want to regenerate it using the system protoc, manually remove
# the .pb.go files, then run `make generate`.
.PHONY: generate-protobuf
generate-protobuf: $(PROTOBUF_FILES)

generate: generate-protobuf

# Pattern rule for generating protobuf files
%.pb.go: %.proto
	$(call LOG,PROTOC,$@)
	$(Q)$(PROTOC) \
	  --plugin=.tools/bin/protoc-gen-go \
	  --proto_path=$(dir $@) \
	  --go_out=$(dir $<) \
	  $< \
	  $(NULL)

.PHONY: protobuf-clean
protobuf-clean:
	$(call LOG,CLEAN,protobuf)
	$(Q)$(RM) $(PROTOBUF_FILES)

maintainer-clean: protobuf-clean

# ----------- MOCK GENERATION ------------

# Mock files to generate
MOCK_FILES := \
	app/types/mocks/runnable_mock.go \
	app/types/mocks/resource_store_mock.go \
	app/types/mocks/store_mock.go \
	app/utils/scout/types/mocks/scout_mock.go \
	app/types/mocks/storage_mock.go \
	app/domain/certificate/mocks/kubernetes_client_mock.go \
	app/domain/k8s/mocks/certificate_client_mock.go \
	$(NULL)

.PHONY: generate-mocks
generate-mocks: $(MOCK_FILES)

generate: generate-mocks

.PHONY: mocks-clean
mocks-clean:
	$(call LOG,CLEAN,mocks)
	$(Q)$(RM) $(MOCK_FILES)

maintainer-clean: mocks-clean

# Convert snake_case to PascalCase using awk
# $(1) = snake_case string
define snake-to-pascal
$(shell echo "$(1)" | $(AWK) -F'_' '{for(i=1;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} {print}' OFS='')
endef

# Enable secondary expansion for automatic dependency calculation
.SECONDEXPANSION:

# Helper to calculate mock dependencies
define mock-deps
$(subst /mocks/,/,$(subst _mock.go,.go,$(1))) $(wildcard $(1:.go=.diff))
endef

# Pattern rule for generating mock files
%_mock.go: $$(call mock-deps,$$@)
	$(call LOG,MOCKGEN,$@)
	$(Q)$(MOCKGEN) \
		-destination=$@ \
		-package=mocks \
		$(GO_MODULE)/$(patsubst %/,%,$(patsubst %/mocks/,%/,$(dir $@))) \
		$(call snake-to-pascal,$(subst _mock,,$(basename $(notdir $@))))
	$(Q)$(if $(filter %.diff,$^),echo "  PATCH        $(filter %.diff,$^)"; patch -si "$(filter %.diff,$^)" "$@")
