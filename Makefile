# Configuration
REGISTRY ?= ghcr.io
GITHUB_USER ?= rusik69
PROJECT_NAME ?= llmcloud-operator
IMG ?= $(REGISTRY)/$(GITHUB_USER)/$(PROJECT_NAME):latest
FRONTEND_IMG ?= $(REGISTRY)/$(GITHUB_USER)/$(PROJECT_NAME)-frontend:latest
CONTAINER_TOOL ?= docker

# Paths
LOCALBIN := $(shell pwd)/bin
SSH_HOST ?= rusik@192.168.1.79
KUBECONFIG ?= $(HOME)/.kube/config-llmcloud
STORAGE_DEVICE ?= /dev/sda

# Tools
KUSTOMIZE := $(LOCALBIN)/kustomize
CONTROLLER_GEN := $(LOCALBIN)/controller-gen
ENVTEST := $(LOCALBIN)/setup-envtest
GOLANGCI_LINT := $(LOCALBIN)/golangci-lint

# Versions
KUSTOMIZE_VERSION ?= v5.7.1
CONTROLLER_TOOLS_VERSION ?= v0.19.0
ENVTEST_K8S_VERSION ?= 1.34
GOLANGCI_LINT_VERSION ?= v2.4.0

.DEFAULT_GOAL := help

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

##@ Development

.PHONY: dev build test lint lint-fix clean
dev: ## Run operator locally
	go run cmd/main.go

build: $(CONTROLLER_GEN) ## Build operator binary
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	go fmt ./...
	go vet ./...
	go build -o bin/manager cmd/main.go

test: $(ENVTEST) ## Run tests
	KUBEBUILDER_ASSETS="$$($(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

lint: $(GOLANGCI_LINT) ## Run linter
	GOFLAGS=-buildvcs=false $(GOLANGCI_LINT) run

lint-fix: $(GOLANGCI_LINT) ## Fix linting issues
	$(GOLANGCI_LINT) run --fix

clean: ## Clean build artifacts
	rm -rf bin/ dist/ cover.out coverage.html

##@ Deployment

.PHONY: deploy uninstall logs status
deploy: build ## Deploy to remote cluster
	./bin/manager deploy --ssh-host=$(SSH_HOST) --storage-device=$(STORAGE_DEVICE)

uninstall: build ## Uninstall from remote cluster
	./bin/manager uninstall --ssh-host=$(SSH_HOST) --k0s

logs: ## View operator logs
	ssh $(SSH_HOST) 'sudo journalctl -u llmcloud-operator -f'

status: ## Check operator status
	ssh $(SSH_HOST) 'sudo systemctl status llmcloud-operator --no-pager'

##@ Frontend

.PHONY: web web-dev
web: build ## Build frontend and operator
	cd web && npm install && npm run build
	$(MAKE) build

web-dev: ## Run frontend dev server
	cd web && npm run dev

##@ Docker

.PHONY: docker-build docker-push docker-buildx
docker-build: ## Build operator image
	$(CONTAINER_TOOL) build -t $(IMG) .

docker-push: ## Push operator image
	$(CONTAINER_TOOL) push $(IMG)

docker-buildx: ## Build and push multi-arch operator image
	$(CONTAINER_TOOL) buildx create --name llmcloud-builder --use 2>/dev/null || $(CONTAINER_TOOL) buildx use llmcloud-builder
	$(CONTAINER_TOOL) buildx build --push --platform=linux/amd64,linux/arm64 --tag $(IMG) .

##@ Tools

.PHONY: tools
tools: $(KUSTOMIZE) $(CONTROLLER_GEN) $(ENVTEST) $(GOLANGCI_LINT) ## Install all tools

$(KUSTOMIZE): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

$(GOLANGCI_LINT): $(LOCALBIN)
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCALBIN) $(GOLANGCI_LINT_VERSION)
