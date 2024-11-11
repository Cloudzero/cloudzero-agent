project_root := $(shell pwd)
#FUNCTIONS := collector shipper
FUNCTIONS := collector
REGION := us-east-2

# Docker is the default container tool (and buildx buildkit)
CONTAINER_TOOL ?= docker
BUILDX_CONTAINER_EXISTS := $(shell $(CONTAINER_TOOL) buildx ls --format "{{.Name}}: {{.DriverEndpoint}}" | grep -c "container:")

BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%I:%M:%S%p')
REVISION ?= $(shell git rev-parse HEAD)
TAG ?= latest

GO := go

.PHONY: default menu build package clean lint

default: menu

menu: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(lastword $(MAKEFILE_LIST)) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

ci: build tests-unit

build: ## Build the functions
		${MAKE} ${MAKEOPTS} $(foreach function,${FUNCTIONS}, build-${function})

build-%:
		cd app/functions/$* && GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 ${GO} build -o bootstrap

clean: ## Clean up
	@rm $(foreach function,${FUNCTIONS}, app/functions/${function}/bootstrap)

fmt: ## Run go fmt against code
	@go fmt ./...

lint: ## Run the linter 
	@golangci-lint run

tests-unit: ## Run unit tests
	@go test -tags=unit -cover -race ./...

package: ## Package the functions
		${MAKE} ${MAKEOPTS} $(foreach function,${FUNCTIONS}, package-${function})

package-%:
ifeq ($(BUILDX_CONTAINER_EXISTS), 0)
	@$(CONTAINER_TOOL) buildx create --name container --driver=docker-container --use
endif
	$(CONTAINER_TOOL) buildx build \
		--builder=container \
		--platform linux/amd64,linux/arm64 \
		--build-arg APP_NAME=$* \
		--build-arg REVISION=$(REVISION) \
		--build-arg TAG=$(TAG) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--push -t ghcr.io/josephbarnett/hexigon/$*:$(TAG) -f app/docker/Dockerfile .
