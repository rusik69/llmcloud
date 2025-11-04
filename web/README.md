# LLMCloud Operator Web UI

Vue.js-based web interface for managing llmcloud-operator resources.

## Features

- **Projects**: Create and manage multi-tenant workspaces with RBAC
- **Virtual Machines**: Deploy and manage KubeVirt VMs
- **LLM Models**: Deploy Ollama-based language models (Deepseek, Llama2, Mistral, etc.)
- **Services**: Install catalog services (PostgreSQL, MySQL, Gitea, etc.)

## Development

```bash
# Install dependencies
npm install

# Start development server (with API proxy)
npm run dev
```

The dev server runs on http://localhost:3000 and proxies API requests to http://localhost:8080

## Building

```bash
# Build for production
npm run build

# Output goes to ../static/
```

## Integration

The built files are embedded into the Go operator binary using `//go:embed` and served at the root path.

Build operator with UI:

```bash
make build-ui
```

## API Endpoints

- `GET /api/v1/projects` - List projects
- `POST /api/v1/projects` - Create project
- `DELETE /api/v1/projects/{name}` - Delete project
- `GET /api/v1/namespaces/{ns}/vms` - List VMs
- `POST /api/v1/namespaces/{ns}/vms` - Create VM
- `DELETE /api/v1/namespaces/{ns}/vms/{name}` - Delete VM
- `GET /api/v1/namespaces/{ns}/models` - List LLM models
- `POST /api/v1/namespaces/{ns}/models` - Deploy model
- `DELETE /api/v1/namespaces/{ns}/models/{name}` - Delete model
- `GET /api/v1/namespaces/{ns}/services` - List services
- `POST /api/v1/namespaces/{ns}/services` - Install service
- `DELETE /api/v1/namespaces/{ns}/services/{name}` - Uninstall service
