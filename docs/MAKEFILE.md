# Makefile Reference

Complete reference for all available make targets in llmcloud-operator.

## Quick Reference

| Task | Command |
|------|---------|
| Deploy everything | `make deploy` |
| Update operator | `make deploy-operator` |
| Remove everything | `make uninstall` |
| Build with UI | `make web` |
| Frontend dev | `make web-dev` |
| View logs | `make logs` |
| Check status | `make status` |
| Run tests | `make test` |
| List all targets | `make help` |

## Configuration

You can override these variables when running make:

```bash
make deploy SSH_HOST=user@host STORAGE_DEVICE=/dev/nvme0n1
```

### Environment Variables

**Deployment:**
- `SSH_HOST` - Remote host for deployment (default: `rusik@192.168.1.79`)
- `KUBECONFIG` - Kubeconfig path (default: `~/.kube/config-llmcloud`)
- `STORAGE_DEVICE` - Block device for storage (default: `/dev/sda`)
- `K0S_VERSION` - k0s version to install (default: `v1.29.1+k0s.0`)

**Docker:**
- `REGISTRY` - Container registry (default: `ghcr.io`)
- `GITHUB_USER` - GitHub username/organization (default: `rusik69`)
- `PROJECT_NAME` - Project name (default: `llmcloud-operator`)
- `IMG` - Docker image name for operator (default: `$(REGISTRY)/$(GITHUB_USER)/$(PROJECT_NAME):latest`)
- `FRONTEND_IMG` - Docker image name for frontend (default: `$(REGISTRY)/$(GITHUB_USER)/$(PROJECT_NAME)-frontend:latest`)
- `CONTAINER_TOOL` - Container tool (default: `docker`)

## Development

### Local Development

```bash
make dev          # Run operator locally with hot reload
make build        # Build operator binary (bin/manager)
make test         # Run unit tests
make clean        # Clean build artifacts
```

### Code Generation

```bash
make manifests    # Generate CRDs and RBAC
make generate     # Generate DeepCopy methods
make fmt          # Run go fmt
make vet          # Run go vet
```

## Remote Deployment

### Deploy

```bash
make deploy                    # Full deployment (k0s + operator + UI)
make deploy-remote-k0s        # Deploy only k0s cluster
make deploy-remote-operator   # Deploy only operator (requires k0s running)
```

**Default host:** `rusik@192.168.1.79`  
**Override:** `SSH_HOST=user@host make deploy`

**Output:**
- Web UI: http://192.168.1.79:8080
- API: http://192.168.1.79:8080/api/v1

### Uninstall

```bash
make uninstall                # Full uninstall (operator + k0s)
make uninstall-remote-operator # Uninstall only operator (keep k0s)
make uninstall-remote-k0s    # Uninstall only k0s cluster
```

**What gets removed:**
- `uninstall-remote-operator`: Systemd service, CRDs, RBAC, operator binary
- `uninstall-remote-k0s`: k0s cluster, all Kubernetes data, local kubeconfig
- `uninstall-remote`: Everything above

### Management

```bash
make logs      # View operator logs
make status    # Check operator status
```

## Web Frontend

### Development

```bash
make web-dev        # Run dev server on :3000 with hot reload
```

Dev server proxies `/api` requests to `localhost:8080`

### Build

```bash
make web            # Build frontend and operator together
```

Builds Vue.js frontend and embeds it into the operator binary.

## Docker

### Build and Push

```bash
make docker-build              # Build Docker image for operator
make docker-push               # Push Docker image for operator
make docker-buildx             # Multi-platform build for operator
make docker-build-frontend     # Build Docker image for frontend
make docker-push-frontend      # Push Docker image for frontend
make docker-buildx-frontend    # Multi-platform build for frontend
```

**Default images:**
- Operator: `ghcr.io/rusik69/llmcloud-operator:latest`
- Frontend: `ghcr.io/rusik69/llmcloud-operator-frontend:latest`

**Examples:**
```bash
# Use defaults
make docker-build docker-push

# Override image name
IMG=myregistry.com/llmcloud-operator:v1.0 make docker-build docker-push

# Override registry and user
REGISTRY=myregistry.com GITHUB_USER=myorg make docker-build docker-push
```

## Kubernetes Deployment

For standard Kubernetes deployments (not remote):

```bash
make install        # Install CRDs to cluster
make uninstall      # Uninstall CRDs from cluster
make deploy-k8s     # Deploy to cluster (kustomize method)
make undeploy       # Remove from cluster
```

## Tools

### Tool Installation

```bash
make tools          # Install all development tools
make kustomize      # Download kustomize
make controller-gen # Download controller-gen
make setup-envtest  # Setup envtest
```

## Common Workflows

### First Time Setup

```bash
# Deploy to remote host
make deploy

# Wait for deployment to complete
# Access UI at http://192.168.1.79:8080
```

### Update Operator

```bash
# Update code, then redeploy
make deploy-remote-operator
```

### Frontend Development

```bash
# Terminal 1: Run operator locally
export KUBECONFIG=~/.kube/config-k0s-remote
make dev

# Terminal 2: Run frontend dev server
make web-dev

# Access: http://localhost:3000
# API calls proxy to localhost:8080
```

### Clean Rebuild

```bash
make clean
make build
make deploy
```

## Examples

### Deploy to Different Host

```bash
make deploy SSH_HOST=admin@192.168.1.100
```

### Deploy with Custom Storage

```bash
make deploy STORAGE_DEVICE=/dev/nvme0n1
```

### Build and Deploy

```bash
make build
make deploy
```

### View Logs

```bash
make logs
```

## Directory Structure

```
llmcloud-operator/
├── bin/                    # Built binaries
├── config/                 # Kubernetes configs
│   ├── crd/bases/         # CRD manifests
│   ├── rbac/              # RBAC manifests
│   └── manager/           # Manager deployment
├── internal/
│   ├── api/               # API server
│   │   └── static/        # Embedded frontend (auto-generated)
│   └── controller/        # Controllers
├── web/                   # Vue.js frontend
│   ├── src/
│   ├── package.json
│   └── vite.config.js
└── static/                # Built frontend (auto-generated)
```

## Help

```bash
make help           # Show all available targets with descriptions
make -n <target>    # Dry run (show what would be executed)
```

