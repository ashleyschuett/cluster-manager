SHELL=/bin/bash
PROJECT_NAME := "cloud-agent"
AGENT_IMAGE_TAG ?= "latest"
COORDINATOR_IMAGE_TAG ?= "latest"
PKG := "github.com/containership/$(PROJECT_NAME)"
PKG_LIST := $(shell glide novendor)
GO_FILES := $(shell find . -type f -not -path './vendor/*' -name '*.go')

# TODO golint has no way to ignore specific files or directories, so we have to
# manually build a lint list. This workaround can go away and we can use
# $PKG_LIST when the k8s code generator is updated to follow golang conventions
# for generated files. See https://github.com/kubernetes/code-generator/issues/30.
LINT_LIST := $(shell go list ./... | grep -v '/pkg/client')

.PHONY: all fmt-check lint test vet release

all: agent coordinator ## (default) Build and deploy agent and coordinator

fmt-check: ## Check the file format
	@gofmt -s -e -d ${GO_FILES}

lint: ## Lint the files
	@golint -set_exit_status ${LINT_LIST}

test: ## Run unittests
	@go test -short ${PKG_LIST}

vet: ## Vet the files
	@go vet ${PKG_LIST}

## Read about data race https://golang.org/doc/articles/race_detector.html
## to not test file for race use `// +build !race` at top
race: ## Run data race detector
	@go test -race -short ${PKG_LIST}

msan: ## Run memory sanitizer (only works on linux/amd64)
	@go test -msan -short ${PKG_LIST}

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

### Commands for local development
mount: ## Set up minikube mount
	@echo "Setting up mount as symlink in /Users/.minikube-mounts folder"
	$(shell sudo mkdir /Users/.minikube-mounts)
	$(shell sudo ln -s ${PWD}/internal/ /Users/.minikube-mounts/internal)
	@echo $(shell ls /Users/.minikube-mounts)

deploy-crds: ## Deploy all CRDs
	kubectl apply -f deploy/crd

deploy-common: deploy-crds ## Deploy all common yamls
	kubectl apply -f deploy/common/containership-core-namespace.yaml
	kubectl apply -f deploy/rbac
	kubectl apply -f deploy/eventrouter/eventrouter.yaml
	kubectl apply -f deploy/common/containership-env-configmap.yaml
	kubectl apply -f deploy/common/containership-coordinator-service.yaml
	kubectl apply -f deploy/common/containership-service-account.yaml

deploy-agent: deploy-common ## Deploy the agent
	kubectl apply -f deploy/development/agent.yaml

deploy-coordinator: deploy-common ## Deploy the coordinator
	kubectl apply -f deploy/development/coordinator.yaml

deploy: deploy-agent deploy-coordinator # Deploy everything

build-agent: ## Build the agent in Docker
	@eval $$(minikube docker-env) ;\
	docker image build -t containership/cloud-agent:$(AGENT_IMAGE_TAG) \
		--build-arg GIT_DESCRIBE=`git describe --dirty` \
		--build-arg GIT_COMMIT=`git rev-parse --short HEAD` \
		-f Dockerfile.agent .

agent: build-agent deploy-agent ## Build and deploy the agent

build-coordinator: ## Build the coordinator in Docker
	@eval $$(minikube docker-env) ;\
	docker image build -t containership/cloud-coordinator:$(COORDINATOR_IMAGE_TAG) \
		--build-arg GIT_DESCRIBE=`git describe --dirty` \
		--build-arg GIT_COMMIT=`git rev-parse --short HEAD` \
		-f Dockerfile.coordinator .

coordinator: build-coordinator deploy-coordinator ## Build and deploy the coordinator

release: ## Build release images for agent and coordinator (must be on semver tag)
	@./scripts/build/release.sh
