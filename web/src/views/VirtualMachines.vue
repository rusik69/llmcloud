<template>
  <div class="vms">
    <div class="header">
      <h2>Virtual Machines</h2>
      <div class="header-actions">
        <select v-model="selectedNamespace" @change="loadVMs" class="namespace-select">
          <option value="">Select Namespace</option>
          <option v-for="ns in namespaces" :key="ns" :value="ns">{{ ns }}</option>
        </select>
        <button @click="showCreateDialog = true" :disabled="!selectedNamespace" class="btn btn-primary">
          Create VM
        </button>
      </div>
    </div>

    <div class="vms-table">
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>CPUs</th>
            <th>Memory</th>
            <th>Disk</th>
            <th>OS</th>
            <th>Status</th>
            <th>Node</th>
            <th>IP Address</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="vm in vms" :key="vm.metadata.name">
            <td>
              <router-link :to="`/vms/${selectedNamespace}/${vm.metadata.name}`" class="vm-link">
                {{ vm.metadata.name }}
              </router-link>
            </td>
            <td>{{ vm.spec.cpus }}</td>
            <td>{{ vm.spec.memory }}</td>
            <td>{{ vm.spec.diskSize || '10Gi' }}</td>
            <td>{{ vm.spec.os }}{{ vm.spec.osVersion ? ':' + vm.spec.osVersion : '' }}</td>
            <td><span :class="['badge', vm.status.phase]">{{ vm.status.phase }}</span></td>
            <td>{{ vm.status.node || '-' }}</td>
            <td>{{ vm.status.ipAddress || '-' }}</td>
            <td>
              <div class="action-buttons">
                <button @click="startVM(vm.metadata.name)" class="btn btn-sm btn-success" :disabled="vm.spec.runStrategy === 'Always'">Start</button>
                <button @click="stopVM(vm.metadata.name)" class="btn btn-sm btn-warning" :disabled="vm.spec.runStrategy === 'Halted'">Stop</button>
                <button @click="rebootVM(vm.metadata.name)" class="btn btn-sm btn-info">Reboot</button>
                <button @click="deleteVM(vm.metadata.name)" class="btn btn-sm btn-danger">Delete</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-if="showCreateDialog" class="modal" @click="showCreateDialog = false">
      <div class="modal-content" @click.stop>
        <h3>Create Virtual Machine</h3>
        <form @submit.prevent="createVM">
          <div class="form-group">
            <label>Name</label>
            <input v-model="newVM.name" required class="form-control" />
          </div>
          <div class="form-group">
            <label>CPUs</label>
            <input v-model.number="newVM.cpus" type="number" min="1" required class="form-control" />
          </div>
          <div class="form-group">
            <label>Memory</label>
            <input v-model="newVM.memory" placeholder="e.g., 1Gi" required class="form-control" />
          </div>
          <div class="form-group">
            <label>Disk Size</label>
            <input v-model="newVM.diskSize" placeholder="e.g., 10Gi" required class="form-control" />
          </div>
          <div class="form-group">
            <label>Operating System</label>
            <select v-model="newVM.os" required class="form-control">
              <option value="">Select OS</option>
              <option value="ubuntu">Ubuntu</option>
              <option value="fedora">Fedora</option>
              <option value="debian">Debian</option>
              <option value="centos">CentOS Stream</option>
              <option value="alpine">Alpine</option>
              <option value="freebsd">FreeBSD</option>
              <option value="cirros">Cirros (Test)</option>
            </select>
          </div>
          <div class="form-group">
            <label>OS Version (optional)</label>
            <input v-model="newVM.osVersion" placeholder="e.g., 22.04" class="form-control" />
          </div>
          <div class="form-actions">
            <button type="button" @click="showCreateDialog = false" class="btn">Cancel</button>
            <button type="submit" class="btn btn-primary">Create</button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import { vmsApi, projectsApi } from '../api/client'

export default {
  setup() {
    const vms = ref([])
    const namespaces = ref([])
    const selectedNamespace = ref('')
    const showCreateDialog = ref(false)
    const newVM = ref({ name: '', cpus: 1, memory: '1Gi', diskSize: '10Gi', os: '', osVersion: '' })

    const loadNamespaces = async () => {
      try {
        const response = await projectsApi.list()
        namespaces.value = response.data.items.map(p => p.status.namespace).filter(Boolean)
      } catch (error) {
        console.error('Failed to load namespaces:', error)
      }
    }

    const loadVMs = async () => {
      if (!selectedNamespace.value) return
      try {
        const response = await vmsApi.list(selectedNamespace.value)
        vms.value = response.data.items || []
      } catch (error) {
        console.error('Failed to load VMs:', error)
      }
    }

    const createVM = async () => {
      try {
        const spec = {
          cpus: newVM.value.cpus,
          memory: newVM.value.memory,
          diskSize: newVM.value.diskSize,
          os: newVM.value.os
        }
        if (newVM.value.osVersion) {
          spec.osVersion = newVM.value.osVersion
        }
        await vmsApi.create(selectedNamespace.value, {
          apiVersion: 'llmcloud.llmcloud.io/v1alpha1',
          kind: 'VirtualMachine',
          metadata: { name: newVM.value.name, namespace: selectedNamespace.value },
          spec
        })
        showCreateDialog.value = false
        newVM.value = { name: '', cpus: 1, memory: '1Gi', diskSize: '10Gi', os: '', osVersion: '' }
        await loadVMs()
      } catch (error) {
        console.error('Failed to create VM:', error)
      }
    }

    const deleteVM = async (name) => {
      if (!confirm(`Delete VM ${name}?`)) return
      try {
        await vmsApi.delete(selectedNamespace.value, name)
        await loadVMs()
      } catch (error) {
        console.error('Failed to delete VM:', error)
      }
    }

    const startVM = async (name) => {
      try {
        await vmsApi.start(selectedNamespace.value, name)
        await loadVMs()
      } catch (error) {
        console.error('Failed to start VM:', error)
        alert('Failed to start VM: ' + error.message)
      }
    }

    const stopVM = async (name) => {
      if (!confirm(`Stop VM ${name}?`)) return
      try {
        await vmsApi.stop(selectedNamespace.value, name)
        await loadVMs()
      } catch (error) {
        console.error('Failed to stop VM:', error)
        alert('Failed to stop VM: ' + error.message)
      }
    }

    const rebootVM = async (name) => {
      if (!confirm(`Reboot VM ${name}?`)) return
      try {
        await vmsApi.reboot(selectedNamespace.value, name)
        await loadVMs()
      } catch (error) {
        console.error('Failed to reboot VM:', error)
        alert('Failed to reboot VM: ' + error.message)
      }
    }

    onMounted(loadNamespaces)

    return {
      vms,
      namespaces,
      selectedNamespace,
      showCreateDialog,
      newVM,
      loadVMs,
      createVM,
      deleteVM,
      startVM,
      stopVM,
      rebootVM
    }
  }
}
</script>

<style scoped>
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
}

.header-actions {
  display: flex;
  gap: 1rem;
}

.namespace-select {
  padding: 0.5rem 1rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 0.875rem;
}

.vms-table {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  overflow: hidden;
}

table {
  width: 100%;
  border-collapse: collapse;
}

th, td {
  padding: 1rem;
  text-align: left;
}

th {
  background: #f5f5f5;
  font-weight: 600;
  border-bottom: 2px solid #e0e0e0;
}

td {
  border-bottom: 1px solid #e0e0e0;
}

.vm-link {
  color: #3498db;
  text-decoration: none;
  font-weight: 500;
}

.vm-link:hover {
  text-decoration: underline;
  color: #2980b9;
}

.truncate {
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.badge {
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.875rem;
  font-weight: 500;
}

.badge.Running {
  background: #e8f5e9;
  color: #2e7d32;
}

.badge.Pending {
  background: #fff3e0;
  color: #ef6c00;
}

.badge.Error {
  background: #ffebee;
  color: #c62828;
}

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
  background: #e0e0e0;
  color: #333;
}

.btn:hover {
  background: #d0d0d0;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary {
  background: #3498db;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #2980b9;
}

.btn-sm {
  padding: 0.25rem 0.75rem;
  font-size: 0.75rem;
}

.btn-danger {
  background: #e74c3c;
  color: white;
}

.btn-danger:hover {
  background: #c0392b;
}

.btn-success {
  background: #27ae60;
  color: white;
}

.btn-success:hover:not(:disabled) {
  background: #229954;
}

.btn-warning {
  background: #f39c12;
  color: white;
}

.btn-warning:hover:not(:disabled) {
  background: #e67e22;
}

.btn-info {
  background: #3498db;
  color: white;
}

.btn-info:hover:not(:disabled) {
  background: #2980b9;
}

.action-buttons {
  display: flex;
  gap: 0.25rem;
  flex-wrap: wrap;
}

.modal {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0,0,0,0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  padding: 2rem;
  border-radius: 8px;
  width: 500px;
  max-width: 90%;
}

.form-group {
  margin-bottom: 1.5rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
}

.form-control {
  width: 100%;
  padding: 0.5rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 1rem;
}

.form-control:focus {
  outline: none;
  border-color: #3498db;
}

.form-actions {
  display: flex;
  gap: 1rem;
  justify-content: flex-end;
}
</style>
