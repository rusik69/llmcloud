# Configuration
REGISTRY ?= ghcr.io
GITHUB_USER ?= rusik69
PROJECT_NAME ?= llmcloud-operator
IMG ?= $(REGISTRY)/$(GITHUB_USER)/$(PROJECT_NAME):latest
FRONTEND_IMG ?= $(REGISTRY)/$(GITHUB_USER)/$(PROJECT_NAME)-frontend:latest
CONTAINER_TOOL ?= docker
SHELL = /bin/bash

# Paths
LOCALBIN ?= $(shell pwd)/bin
GOBIN ?= $(shell go env GOPATH)/bin

# Remote deployment
SSH_HOST ?= rusik@192.168.1.79
KUBECONFIG ?= $(HOME)/.kube/config-llmcloud
STORAGE_DEVICE ?= /dev/sda

# Tool binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint

# Tool versions
KUSTOMIZE_VERSION ?= v5.7.1
CONTROLLER_TOOLS_VERSION ?= v0.19.0
ENVTEST_K8S_VERSION ?= 1.34
GOLANGCI_LINT_VERSION ?= v2.4.0

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

##@ Development

.PHONY: dev
dev: ## Run operator locally with hot reload
	@go run cmd/main.go

.PHONY: build
build: ## Build operator binary
	@$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	@$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	@go fmt ./...
	@go vet ./...
	@go build -o bin/manager cmd/main.go

.PHONY: test
test: setup-envtest ## Run unit tests
	@KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Run golangci-lint
	@$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: $(GOLANGCI_LINT) ## Run golangci-lint with auto-fix
	@$(GOLANGCI_LINT) run --fix

.PHONY: clean
clean: ## Clean build artifacts
	@rm -rf bin/ dist/ cover.out coverage.html

##@ Remote Deployment

.PHONY: deploy
deploy: build ## Deploy to remote k0s cluster
	@./bin/manager deploy --ssh-host=$(SSH_HOST) --storage-device=$(STORAGE_DEVICE)

.PHONY: uninstall
uninstall: build ## Uninstall operator and k0s from remote
	@./bin/manager uninstall --ssh-host=$(SSH_HOST) --k0s

.PHONY: logs
logs: ## View operator logs
	@ssh $(SSH_HOST) 'sudo journalctl -u llmcloud-operator -f'

.PHONY: status
status: ## Check operator status
	@ssh $(SSH_HOST) 'sudo systemctl status llmcloud-operator --no-pager'

##@ Web Frontend

.PHONY: web
web: ## Build frontend and operator
	@cd web && npm install && npm run build
	@$(MAKE) build

.PHONY: web-dev
web-dev: ## Run frontend dev server
	@cd web && npm run dev

##@ Docker

.PHONY: docker-build
docker-build: ## Build docker image for the manager
	$(CONTAINER_TOOL) build -t ${IMG} -f Dockerfile .

.PHONY: docker-push
docker-push: ## Push docker image for the manager
	$(CONTAINER_TOOL) push ${IMG}

.PHONY: docker-build-frontend
docker-build-frontend: ## Build docker image for the frontend
	$(CONTAINER_TOOL) build -t ${FRONTEND_IMG} -f web/Dockerfile ./web

.PHONY: docker-push-frontend
docker-push-frontend: ## Push docker image for the frontend
	$(CONTAINER_TOOL) push ${FRONTEND_IMG}

.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name project-v3-builder || true
	$(CONTAINER_TOOL) buildx use project-v3-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=linux/amd64,linux/arm64 --tag ${IMG} -f Dockerfile.cross .
	rm Dockerfile.cross

.PHONY: docker-buildx-frontend
docker-buildx-frontend: ## Build and push docker image for the frontend for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' web/Dockerfile > web/Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name project-v3-builder || true
	$(CONTAINER_TOOL) buildx use project-v3-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=linux/amd64,linux/arm64 --tag ${FRONTEND_IMG} -f web/Dockerfile.cross ./web
	rm web/Dockerfile.cross

##@ Tools

.PHONY: tools
tools: $(KUSTOMIZE) $(CONTROLLER_GEN) $(ENVTEST) $(GOLANGCI_LINT) ## Install all tools

.PHONY: kustomize
kustomize: $(KUSTOMIZE)
$(KUSTOMIZE): $(LOCALBIN)
	@$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN)
$(CONTROLLER_GEN): $(LOCALBIN)
	@$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: setup-envtest
setup-envtest: envtest
	@$(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path

.PHONY: envtest
envtest: $(ENVTEST)
$(ENVTEST): $(LOCALBIN)
	@GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT)
$(GOLANGCI_LINT): $(LOCALBIN)
	@echo "Installing golangci-lint@$(GOLANGCI_LINT_VERSION)"
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(LOCALBIN) $(GOLANGCI_LINT_VERSION)
	@$(GOLANGCI_LINT) version

define go-install-tool
@[ -f "$(1)-$(3)" ] && [ "$$(readlink -- "$(1)" 2>/dev/null)" = "$(1)-$(3)" ] || { \
set -e; \
echo "Installing $(2)@$(3)" ;\
rm -f $(1) ;\
GOBIN=$(LOCALBIN) go install $(2)@$(3) ;\
mv $(1) $(1)-$(3) ;\
ln -sf $$(basename $(1)-$(3)) $(1) ;\
}
endef
