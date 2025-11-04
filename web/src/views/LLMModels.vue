<template>
  <div class="models">
    <div class="header">
      <h2>LLM Models</h2>
      <div class="header-actions">
        <select v-model="selectedNamespace" @change="loadModels" class="namespace-select">
          <option value="">Select Namespace</option>
          <option v-for="ns in namespaces" :key="ns" :value="ns">{{ ns }}</option>
        </select>
        <button @click="showCreateDialog = true" :disabled="!selectedNamespace" class="btn btn-primary">
          Deploy Model
        </button>
      </div>
    </div>

    <div class="models-table">
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Model</th>
            <th>Size</th>
            <th>Provider</th>
            <th>Replicas</th>
            <th>Status</th>
            <th>Endpoint</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="model in models" :key="model.metadata.name">
            <td><strong>{{ model.metadata.name }}</strong></td>
            <td>{{ model.spec.modelName }}</td>
            <td>{{ model.spec.modelSize || '-' }}</td>
            <td>{{ model.spec.provider || 'ollama' }}</td>
            <td>{{ model.status.readyReplicas || 0 }} / {{ model.spec.replicas || 1 }}</td>
            <td>
              <span :class="['badge', model.status.phase]">{{ model.status.phase || 'Pending' }}</span>
            </td>
            <td>{{ model.status.endpoint || '-' }}</td>
            <td>
              <button @click="deleteModel(model.metadata.name)" class="btn btn-sm btn-danger">Delete</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Model Dialog -->
    <div v-if="showCreateDialog" class="modal" @click="showCreateDialog = false">
      <div class="modal-content" @click.stop>
        <h3>Deploy LLM Model</h3>
        <form @submit.prevent="createModel">
          <div class="form-group">
            <label>Name *</label>
            <input v-model="newModel.name" required class="form-control" />
          </div>
          <div class="form-group">
            <label>Model *</label>
            <select v-model="newModel.modelName" required class="form-control">
              <option value="">Select Model</option>
              <option value="llama2">Llama 2</option>
              <option value="llama3">Llama 3</option>
              <option value="mistral">Mistral</option>
              <option value="mixtral">Mixtral</option>
              <option value="codellama">Code Llama</option>
              <option value="phi">Phi</option>
            </select>
          </div>
          <div class="form-group">
            <label>Size</label>
            <select v-model="newModel.modelSize" class="form-control">
              <option value="">Default</option>
              <option value="7b">7B</option>
              <option value="13b">13B</option>
              <option value="70b">70B</option>
            </select>
          </div>
          <div class="form-group">
            <label>Provider</label>
            <input v-model="newModel.provider" placeholder="ollama" class="form-control" />
          </div>
          <div class="form-group">
            <label>Replicas</label>
            <input v-model.number="newModel.replicas" type="number" min="1" class="form-control" />
          </div>
          <div class="form-actions">
            <button type="button" @click="showCreateDialog = false" class="btn">Cancel</button>
            <button type="submit" class="btn btn-primary">Deploy</button>
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

const models = ref([])
const namespaces = ref([])
const selectedNamespace = ref('')
const showCreateDialog = ref(false)
const newModel = ref({
  name: '',
  modelName: '',
  modelSize: '',
  provider: 'ollama',
  replicas: 1
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

const loadModels = async () => {
  if (!selectedNamespace.value) return

  try {
    const response = await api.get(`/namespaces/${selectedNamespace.value}/models`)
    models.value = response.data.items || []
  } catch (error) {
    console.error('Failed to load models:', error)
  }
}

const createModel = async () => {
  try {
    const modelData = {
      apiVersion: 'llmcloud.llmcloud.io/v1alpha1',
      kind: 'LLMModel',
      metadata: {
        name: newModel.value.name,
        namespace: selectedNamespace.value
      },
      spec: {
        modelName: newModel.value.modelName,
        provider: newModel.value.provider || 'ollama',
        replicas: newModel.value.replicas || 1
      }
    }

    if (newModel.value.modelSize) {
      modelData.spec.modelSize = newModel.value.modelSize
    }

    await api.post(`/namespaces/${selectedNamespace.value}/models`, modelData)
    showCreateDialog.value = false
    newModel.value = { name: '', modelName: '', modelSize: '', provider: 'ollama', replicas: 1 }
    await loadModels()
  } catch (error) {
    console.error('Failed to create model:', error)
    alert('Failed to create model: ' + (error.response?.data || error.message))
  }
}

const deleteModel = async (name) => {
  if (!confirm(`Delete model ${name}?`)) return

  try {
    await api.delete(`/namespaces/${selectedNamespace.value}/models/${name}`)
    await loadModels()
  } catch (error) {
    console.error('Failed to delete model:', error)
    alert('Failed to delete model: ' + (error.response?.data || error.message))
  }
}

onMounted(loadNamespaces)
</script>

<style scoped>
.models {
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

.models-table {
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
