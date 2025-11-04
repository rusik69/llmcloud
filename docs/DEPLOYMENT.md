# Deployment Guide

Complete guide for deploying llmcloud-operator with k0s cluster and web UI.

## Quick Start

Deploy everything to a remote host:

```bash
make deploy
```

**Default**: Deploys to `rusik@192.168.1.79`  
**Override**: `SSH_HOST=user@hostname make deploy`

After deployment:
- **Web UI**: http://192.168.1.79:8080
- **API**: http://192.168.1.79:8080/api/v1

## Architecture

```
Remote Host
├── k0s cluster
│   ├── KubeVirt (VM management)
│   ├── CDI (data import)
│   └── CRDs (Projects, VMs, Models, Services)
└── llmcloud-operator (systemd service)
    ├── Controllers (reconciliation loops)
    ├── API Server (REST endpoints on :8080)
    └── Web UI (embedded Vue.js app)
```

## What Gets Deployed

### 1. k0s Cluster
- k0s v1.29.1+k0s.0 (configurable via `K0S_VERSION`)
- Controller + Worker on single node
- KubeVirt v1.1.1 for VM management
- CDI v1.58.0 for data volumes

### 2. LLMCloud Operator
- Deployed as systemd service at `/opt/llmcloud-operator/manager`
- Listens on port 8080 (API + UI)
- Custom Resource Definitions (CRDs):
  - Project - Multi-tenant workspaces
  - VirtualMachine - KubeVirt VMs
  - LLMModel - Ollama-based LLM deployments
  - Service - Helm chart catalog

### 3. Web UI
- Vue.js 3 SPA embedded in operator binary
- Full CRUD operations for all CRDs
- Real-time status updates

## Storage Configuration

The operator uses a dedicated block device for persistent storage.

### Default Configuration
- Storage device: `/dev/sda`
- Mount point: `/mnt`
- Filesystem: ext4

### Custom Storage Device

```bash
make deploy STORAGE_DEVICE=/dev/nvme0n1
```

### Storage Layout

```
/mnt/
├── k0s/              # k0s cluster data (etcd, etc.)
├── containerd/       # Container images and layers
├── vm-disks/         # Virtual machine disk images
├── llm-models/       # LLM model files
└── services-data/   # Service persistent data
```

### Component Storage Paths

- **k0s**: `/mnt/k0s` (etcd data: `/mnt/k0s/etcd`)
- **Containerd**: `/mnt/containerd` (all container images)
- **Local Path Provisioner**: `/mnt/vm-disks` (VM disk PVCs)
- **LLM Models**: `/mnt/llm-models` (model storage and caching)
- **Services**: `/mnt/services-data` (persistent service data)

### Storage Requirements

- Block device must exist on target host
- Device should be empty (will be formatted during deployment)
- Recommended: 100GB minimum
- Root/sudo access required

### Uninstall Behavior

When running `make uninstall`, the system will:
1. Stop all services
2. Delete all data in `/mnt/*`
3. Unmount `/mnt`
4. Remove the fstab entry
5. Leave the block device unformatted

**Note:** The storage device itself is not wiped. To manually wipe:
```bash
ssh user@host "sudo wipefs -a /dev/sda"
```

## Deployment Steps

### Step-by-Step Deployment

```bash
# 1. Deploy k0s cluster
make deploy-remote-k0s

# 2. Verify cluster
export KUBECONFIG=~/.kube/config-k0s-remote
kubectl get nodes

# 3. Deploy operator
make deploy-remote-operator
```

### Full Deployment

```bash
make deploy
```

This will:
1. Install k0s (or skip if already installed)
2. Install KubeVirt and CDI
3. Build Vue.js frontend
4. Build operator binary
5. Deploy to remote host
6. Start systemd service
7. Install CRDs and RBAC

## Configuration

### Environment Variables

**k0s deployment:**
- `SSH_HOST` - Remote host (default: `rusik@192.168.1.79`)
- `K0S_VERSION` - k0s version (default: `v1.29.1+k0s.0`)
- `STORAGE_DEVICE` - Storage device (default: `/dev/sda`)

**Operator deployment:**
- `SSH_HOST` - Remote host (default: `rusik@192.168.1.79`)
- `KUBECONFIG` - Local kubeconfig path (default: `~/.kube/config-k0s-remote`)

### Systemd Service

Service file: `/etc/systemd/system/llmcloud-operator.service`

```bash
# View logs
ssh rusik@192.168.1.79 'sudo journalctl -u llmcloud-operator -f'

# Restart service
ssh rusik@192.168.1.79 'sudo systemctl restart llmcloud-operator'

# Check status
ssh rusik@192.168.1.79 'sudo systemctl status llmcloud-operator'
```

## Usage Examples

### Create Project

```bash
kubectl apply -f - <<EOF
apiVersion: llmcloud.llmcloud.io/v1alpha1
kind: Project
metadata:
  name: my-project
spec:
  description: "My first project"
  members:
    - username: admin
      role: owner
EOF
```

### Deploy VM

```bash
kubectl apply -f - <<EOF
apiVersion: llmcloud.llmcloud.io/v1alpha1
kind: VirtualMachine
metadata:
  name: test-vm
  namespace: project-my-project
spec:
  cpus: 2
  memory: 2Gi
  image: quay.io/kubevirt/cirros-container-disk-demo:latest
EOF
```

### Deploy LLM Model

```bash
kubectl apply -f - <<EOF
apiVersion: llmcloud.llmcloud.io/v1alpha1
kind: LLMModel
metadata:
  name: deepseek
  namespace: project-my-project
spec:
  model: deepseek-r1:1.5b
  replicas: 1
  gpu: false
EOF
```

### Install Service

```bash
kubectl apply -f - <<EOF
apiVersion: llmcloud.llmcloud.io/v1alpha1
kind: Service
metadata:
  name: postgres
  namespace: project-my-project
spec:
  type: postgresql
  version: "15.5.0"
  config:
    password: "mysecretpassword"
    database: "mydb"
EOF
```

## Verification Checklist

### Pre-Deployment

- [ ] SSH access to remote host works
- [ ] Sudo access without password prompts
- [ ] Block device exists and is available
- [ ] At least 10GB free disk space
- [ ] Go 1.21+ installed locally
- [ ] Node.js/npm installed locally

### Post-Deployment

- [ ] `make deploy` completes without errors
- [ ] Web UI accessible at http://192.168.1.79:8080
- [ ] API responds at http://192.168.1.79:8080/api/v1
- [ ] k0s cluster is running: `ssh host 'sudo k0s status'`
- [ ] Operator service is active: `ssh host 'sudo systemctl status llmcloud-operator'`
- [ ] CRDs are installed: `kubectl get crd | grep llmcloud`
- [ ] Can create Project via UI
- [ ] Can create VM via UI
- [ ] Can deploy LLM Model via UI
- [ ] Can install Service via UI

### Test Deployment

```bash
# 1. Deploy k0s
make deploy-remote-k0s

# 2. Verify k0s
ssh rusik@192.168.1.79 'sudo k0s status'
export KUBECONFIG=~/.kube/config-k0s-remote
kubectl get nodes

# 3. Deploy operator
make deploy-remote-operator

# 4. Test API
curl http://192.168.1.79:8080/api/v1/projects

# 5. Test UI
curl -I http://192.168.1.79:8080/
open http://192.168.1.79:8080
```

## Troubleshooting

### Operator not starting

```bash
# Check logs
ssh rusik@192.168.1.79 'sudo journalctl -u llmcloud-operator -n 100'

# Check k0s status
ssh rusik@192.168.1.79 'sudo k0s status'

# Verify CRDs installed
export KUBECONFIG=~/.kube/config-k0s-remote
kubectl get crd | grep llmcloud
```

### UI not accessible

```bash
# Check operator is running
ssh rusik@192.168.1.79 'sudo systemctl status llmcloud-operator'

# Check port is listening
ssh rusik@192.168.1.79 'sudo netstat -tlnp | grep 8080'

# Check firewall
ssh rusik@192.168.1.79 'sudo ufw status'
```

### KubeVirt VMs not starting

```bash
# Check KubeVirt status
export KUBECONFIG=~/.kube/config-k0s-remote
kubectl get kubevirt -n kubevirt

# Check emulation enabled
kubectl get kubevirt kubevirt -n kubevirt -o yaml | grep useEmulation

# Check VM status
kubectl get vmi -A
```

### Storage issues

```bash
# Check device exists
ssh host 'lsblk | grep sda'

# Check mount
ssh host 'mount | grep /mnt'

# Check filesystem
ssh host 'df -h /mnt'
```

### Build fails

```bash
# Check Go version
go version  # Should be 1.21+

# Check npm packages
cd web && npm install && npm run build
ls -la static/  # Should show index.html, assets/
```

## Uninstallation

### Quick Uninstall

```bash
# Uninstall everything (operator + k0s)
make uninstall

# Or step by step:
make uninstall-remote-operator  # Remove operator, CRDs, RBAC
make uninstall-remote-k0s      # Remove k0s cluster
```

### What Gets Removed

**`make uninstall-remote-operator`:**
- Stops and disables llmcloud-operator systemd service
- Removes `/opt/llmcloud-operator/` directory
- Removes `/etc/systemd/system/llmcloud-operator.service`
- Deletes all Projects (cascades to VMs, Models, Services)
- Removes CRDs from cluster
- Removes RBAC (ClusterRole, RoleBindings)
- k0s cluster remains running

**`make uninstall-remote-k0s`:**
- Stops k0s
- Resets k0s (deletes all Kubernetes data)
- Removes k0s binary and directories
- Removes local kubeconfig (`~/.kube/config-k0s-remote`)

**`make uninstall`:**
- Runs both uninstall-remote-operator and uninstall-remote-k0s
- Complete removal of all components

### Manual Uninstall

If scripts fail:

```bash
# Stop and remove operator
ssh rusik@192.168.1.79 'sudo systemctl stop llmcloud-operator'
ssh rusik@192.168.1.79 'sudo systemctl disable llmcloud-operator'
ssh rusik@192.168.1.79 'sudo rm -rf /opt/llmcloud-operator'
ssh rusik@192.168.1.79 'sudo rm /etc/systemd/system/llmcloud-operator.service'

# Remove CRDs
export KUBECONFIG=~/.kube/config-k0s-remote
kubectl delete projects --all
kubectl delete -f config/crd/bases/
kubectl delete -f config/rbac/

# Stop and remove k0s
ssh rusik@192.168.1.79 'sudo k0s stop'
ssh rusik@192.168.1.79 'sudo k0s reset --yes'
```

## Production Considerations

1. **High Availability**: Deploy k0s in HA mode with multiple controllers
2. **Persistent Storage**: Configure storage classes for VMs and services
3. **TLS/HTTPS**: Add reverse proxy (nginx/traefik) with TLS certificates
4. **Authentication**: Implement OAuth2/OIDC for web UI
5. **Monitoring**: Add Prometheus metrics and Grafana dashboards
6. **Backup**: Regular etcd backups and PV snapshots
7. **Resource Quotas**: Enforce project-level resource limits

