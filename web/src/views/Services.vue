<template>
  <div class="services">
    <div class="header">
      <h2>Services</h2>
      <div class="header-actions">
        <select v-model="selectedNamespace" @change="loadServices" class="namespace-select">
          <option value="">Select Namespace</option>
          <option v-for="ns in namespaces" :key="ns" :value="ns">{{ ns }}</option>
        </select>
        <button @click="showCreateDialog = true" :disabled="!selectedNamespace" class="btn btn-primary">
          Create Service
        </button>
      </div>
    </div>

    <div class="services-table">
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Image</th>
            <th>Replicas</th>
            <th>Status</th>
            <th>Endpoint</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="service in services" :key="service.metadata.name">
            <td><strong>{{ service.metadata.name }}</strong></td>
            <td>{{ service.spec.type }}</td>
            <td class="image-cell">{{ service.spec.image }}</td>
            <td>{{ service.status.readyReplicas || 0 }} / {{ service.spec.replicas || 1 }}</td>
            <td>
              <span :class="['badge', service.status.phase]">{{ service.status.phase || 'Pending' }}</span>
            </td>
            <td>{{ service.status.endpoint || '-' }}</td>
            <td>
              <button @click="deleteService(service.metadata.name)" class="btn btn-sm btn-danger">Delete</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Service Dialog -->
    <div v-if="showCreateDialog" class="modal" @click="showCreateDialog = false">
      <div class="modal-content" @click.stop>
        <h3>Create Service</h3>
        <form @submit.prevent="createService">
          <div class="form-group">
            <label>Name *</label>
            <input v-model="newService.name" required class="form-control" />
          </div>
          <div class="form-group">
            <label>Type *</label>
            <select v-model="newService.type" required class="form-control">
              <option value="">Select Type</option>
              <option value="api">API</option>
              <option value="web">Web</option>
              <option value="worker">Worker</option>
              <option value="database">Database</option>
            </select>
          </div>
          <div class="form-group">
            <label>Image *</label>
            <input v-model="newService.image" required placeholder="e.g., nginx:latest" class="form-control" />
          </div>
          <div class="form-group">
            <label>Replicas</label>
            <input v-model.number="newService.replicas" type="number" min="1" class="form-control" />
          </div>
          <div class="form-group">
            <label>Port</label>
            <input v-model.number="newService.port" type="number" placeholder="e.g., 80" class="form-control" />
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

<script setup>
import { ref, onMounted } from 'vue'
import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('token')}`,
    'Content-Type': 'application/json'
  }
})

const services = ref([])
const namespaces = ref([])
const selectedNamespace = ref('')
const showCreateDialog = ref(false)
const newService = ref({
  name: '',
  type: '',
  image: '',
  replicas: 1,
  port: null
})

const loadNamespaces = async () => {
  try {
    const response = await axios.get('/api/v1/projects', {
      headers: { 'Authorization': `Bearer ${localStorage.getItem('token')}` }
    })
    namespaces.value = response.data.items.map(p => p.status.namespace).filter(Boolean)
  } catch (error) {
    console.error('Failed to load namespaces:', error)
  }
}

const loadServices = async () => {
  if (!selectedNamespace.value) return

  try {
    const response = await api.get(`/namespaces/${selectedNamespace.value}/services`)
    services.value = response.data.items || []
  } catch (error) {
    console.error('Failed to load services:', error)
  }
}

const createService = async () => {
  try {
    const serviceData = {
      apiVersion: 'llmcloud.llmcloud.io/v1alpha1',
      kind: 'Service',
      metadata: {
        name: newService.value.name,
        namespace: selectedNamespace.value
      },
      spec: {
        type: newService.value.type,
        image: newService.value.image,
        replicas: newService.value.replicas || 1
      }
    }

    if (newService.value.port) {
      serviceData.spec.ports = [{
        port: newService.value.port,
        targetPort: newService.value.port,
        protocol: 'TCP'
      }]
    }

    await api.post(`/namespaces/${selectedNamespace.value}/services`, serviceData)
    showCreateDialog.value = false
    newService.value = { name: '', type: '', image: '', replicas: 1, port: null }
    await loadServices()
  } catch (error) {
    console.error('Failed to create service:', error)
    alert('Failed to create service: ' + (error.response?.data || error.message))
  }
}

const deleteService = async (name) => {
  if (!confirm(`Delete service ${name}?`)) return

  try {
    await api.delete(`/namespaces/${selectedNamespace.value}/services/${name}`)
    await loadServices()
  } catch (error) {
    console.error('Failed to delete service:', error)
    alert('Failed to delete service: ' + (error.response?.data || error.message))
  }
}

onMounted(loadNamespaces)
</script>

<style scoped>
.services {
  max-width: 1400px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
}

.header h2 {
  margin: 0;
  font-size: 1.75rem;
}

.header-actions {
  display: flex;
  gap: 1rem;
}

.namespace-select {
  padding: 0.5rem 1rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 0.9rem;
}

.services-table {
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

table {
  width: 100%;
  border-collapse: collapse;
}

th {
  background: #f5f5f5;
  padding: 1rem;
  text-align: left;
  font-weight: 600;
  border-bottom: 2px solid #ddd;
}

td {
  padding: 1rem;
  border-bottom: 1px solid #eee;
}

.image-cell {
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: monospace;
  font-size: 0.85rem;
}

tbody tr:hover {
  background: #f9f9f9;
}

.badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
  background: #e0e0e0;
  color: #666;
}

.badge.Running {
  background: #4caf50;
  color: white;
}

.badge.Pending {
  background: #ff9800;
  color: white;
}

.badge.Failed {
  background: #f44336;
  color: white;
}

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
  transition: all 0.2s;
}

.btn-primary {
  background: #667eea;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #5568d3;
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.8rem;
}

.btn-danger {
  background: #f44336;
  color: white;
}

.btn-danger:hover {
  background: #d32f2f;
}

.modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0,0,0,0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  background: white;
  border-radius: 8px;
  padding: 2rem;
  width: 90%;
  max-width: 500px;
  box-shadow: 0 10px 40px rgba(0,0,0,0.3);
}

.modal-content h3 {
  margin-top: 0;
  margin-bottom: 1.5rem;
}

.form-group {
  margin-bottom: 1.5rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  font-weight: 500;
  color: #333;
}

.form-control {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 0.875rem;
}

.form-control:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 1.5rem;
}
</style>
