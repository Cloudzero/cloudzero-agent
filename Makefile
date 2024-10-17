project_root := $(shell pwd)
applications := remotewrite
stackname ?= fastapi
region := us-east-1

GO := go
ARCH := arm64

default: menu
.PHONY: default

menu:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(lastword $(MAKEFILE_LIST)) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: menu

build: ## Builds the applications
	${MAKE} ${MAKEOPTS} $(foreach app,${applications}, build-${app})
.PHONY: build

build-%:
	cd app/$* && GOOS=linux GOARCH=$(ARCH) CGO_ENABLED=0 ${GO} build -o bootstrap
.PHONY: build-%

clean: ## Cleans the build artifacts
	@rm -fr dist
	@go clean -cache
	@rm $(foreach app,${applications}, app/${app}/bootstrap)
.PHONY: clean

deploy: ## Deploy the AWS CloudFormation Stack
	@if [ -f samconfig.toml ]; \
		then sam deploy --stack-name ${stackname}; \
		else sam deploy -g --stack-name ${stackname}; \
	fi
.PHONY: deploy

delete: ## Delete the AWS CloudFormation Stack
	@sam delete --stack-name ${stackname}
.PHONY: delete
