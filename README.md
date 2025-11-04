# LLMCloud Operator

Kubernetes operator for multi-tenant LLM infrastructure with VMs, models, and services.

## Features

- 🏗️ Multi-tenant projects with RBAC
- 🖥️ Virtual machines (KubeVirt)
- 🤖 LLM models (Ollama: Deepseek, Llama2, Mistral)
- 📦 Service catalog (PostgreSQL, MySQL, Gitea)
- 🌐 Web UI (Vue.js)
- 🔌 REST API

## Quick Start

### Using Dev Container (Recommended)

Open in VS Code and select **"Reopen in Container"** for a consistent development environment. See [.devcontainer/README.md](.devcontainer/README.md) for details.

### Local Development

```bash
make deploy    # Deploy k0s + operator + web UI
```

Access: **http://192.168.1.79:8080**

## Documentation

- [Dev Container Setup](.devcontainer/README.md) - Development environment setup
- [Deployment Guide](docs/DEPLOYMENT.md) - Complete deployment instructions
- [Makefile Reference](docs/MAKEFILE.md) - All make targets and commands
- [Testing Guide](docs/TESTING.md) - Testing and coverage information

## Commands

| Task | Command |
|------|---------|
| Deploy | `make deploy` |
| Update operator | `make deploy-remote-operator` |
| Uninstall | `make uninstall` |
| View logs | `make logs` |
| Check status | `make status` |
| Frontend dev | `make web-dev` |
| Run tests | `make test` |

## Configuration

```bash
SSH_HOST=user@host make deploy          # Change target host
STORAGE_DEVICE=/dev/nvme0n1 make deploy # Change storage device
K0S_VERSION=v1.30.0+k0s.0 make deploy  # Change k0s version
```

## License

Apache 2.0
