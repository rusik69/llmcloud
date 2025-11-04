# Dev Container Setup

This project includes a devcontainer configuration for consistent development environments.

## Prerequisites

- [Docker](https://www.docker.com/get-started) installed and running
- [VS Code](https://code.visualstudio.com/) with the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

## Getting Started

### Option 1: VS Code Command Palette

1. Open the project in VS Code
2. Press `F1` or `Cmd+Shift+P` (Mac) / `Ctrl+Shift+P` (Windows/Linux)
3. Select **"Dev Containers: Reopen in Container"**
4. Wait for the container to build and start

### Option 2: VS Code Popup

1. Open the project in VS Code
2. You should see a popup: **"Folder contains a Dev Container configuration file. Reopen folder to develop in a container"**
3. Click **"Reopen in Container"**

## What's Included

The devcontainer includes:

- **Go 1.24** - Matching the project's Go version
- **Docker-in-Docker** - For building container images
- **kubectl** - Kubernetes CLI tool
- **Helm** - Kubernetes package manager
- **Node.js 20** - For frontend development
- **Development Tools**:
  - `make` - Build automation
  - `git` - Version control
  - `curl`, `wget` - Utilities

## VS Code Extensions

Automatically installed:

- **Go** - Go language support
- **Kubernetes** - Kubernetes tools
- **YAML** - YAML language support
- **Makefile Tools** - Makefile support
- **Volar** - Vue.js language support
- **GitLens** - Git supercharged

## Port Forwarding

The following ports are automatically forwarded:

- **8080** - Operator API/UI
- **3000** - Frontend dev server
- **8081** - Health probe endpoint

## First-Time Setup

After the container starts, run:

```bash
# Install Go tools
make tools

# Install frontend dependencies (if not already done)
cd web && npm install
```

## Common Tasks

### Build the Operator

```bash
make build
```

### Run Tests

```bash
make test
```

### Run Operator Locally

```bash
make dev
```

### Frontend Development

```bash
# Terminal 1: Run operator
make dev

# Terminal 2: Run frontend dev server
make web-dev
```

### Build Docker Images

```bash
make docker-build
```

## Docker Access

Docker-in-Docker is configured, so you can:

```bash
# Build images
docker build -t llmcloud-operator:dev .

# Run containers
docker run ...

# Use docker-compose
docker compose up
```

## Kubernetes Access

kubectl is installed and configured. To use it:

```bash
# Set up kubeconfig (if deploying to remote)
export KUBECONFIG=~/.kube/config-llmcloud

# Use kubectl
kubectl get nodes
```

## Troubleshooting

### Container Won't Start

1. Ensure Docker is running: `docker ps`
2. Check Docker has enough resources (CPU/Memory)
3. Try rebuilding: **"Dev Containers: Rebuild Container"**

### Port Forwarding Issues

1. Check if ports are already in use
2. Manually forward ports in VS Code

### Go Tools Not Found

```bash
make tools
```

### Frontend Dependencies Missing

```bash
cd web && npm install
```

### Permission Issues

The container runs as `vscode` user. If you need root:

```bash
sudo su
```

## Rebuilding the Container

To rebuild the container with fresh dependencies:

1. Press `F1` or `Cmd+Shift+P`
2. Select **"Dev Containers: Rebuild Container"**

## Exiting the Container

To return to local development:

1. Press `F1` or `Cmd+Shift+P`
2. Select **"Dev Containers: Reopen Folder Locally"**

## Customization

Edit `.devcontainer/devcontainer.json` to customize:

- Add more VS Code extensions
- Install additional tools
- Configure environment variables
- Add more port forwards

## Learn More

- [Dev Containers Documentation](https://containers.dev/)
- [VS Code Dev Containers](https://code.visualstudio.com/docs/devcontainers/containers)

