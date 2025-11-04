<template>
  <div class="nodes">
    <div class="header">
      <h2>Cluster Nodes</h2>
      <button @click="showAddNodeDialog = true" class="btn-primary">Add Node</button>
    </div>

    <div v-if="loading" class="loading">Loading nodes...</div>
    <div v-else-if="error" class="error">{{ error }}</div>
    <div v-else-if="nodes.length === 0" class="empty-state">
      <p>No nodes found in cluster</p>
    </div>
    <div v-else class="nodes-table">
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Status</th>
            <th>Roles</th>
            <th>Age</th>
            <th>Version</th>
            <th>Internal IP</th>
            <th>OS / Arch</th>
            <th>Resources</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="node in nodes" :key="node.metadata.name">
            <td><strong>{{ node.metadata.name }}</strong></td>
            <td>
              <span :class="['status', getNodeStatus(node).toLowerCase()]">
                {{ getNodeStatus(node) }}
              </span>
            </td>
            <td>
              <span v-for="role in getNodeRoles(node)" :key="role" class="badge">
                {{ role }}
              </span>
            </td>
            <td>{{ formatAge(node.metadata.creationTimestamp) }}</td>
            <td>{{ getNodeVersion(node) }}</td>
            <td>{{ getNodeIP(node) }}</td>
            <td>{{ getOSInfo(node) }}</td>
            <td>
              <div class="resources">
                <div>CPU: {{ getResourceValue(node, 'cpu') }}</div>
                <div>Memory: {{ getResourceValue(node, 'memory') }}</div>
              </div>
            </td>
            <td>
              <button @click="confirmRemoveNode(node.metadata.name)" class="btn-danger btn-small">Remove</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Add Node Dialog -->
    <div v-if="showAddNodeDialog" class="modal-overlay" @click="closeAddNodeDialog">
      <div class="modal" @click.stop>
        <div class="modal-header">
          <h3>Add New Node</h3>
          <button @click="closeAddNodeDialog" class="close-btn">&times;</button>
        </div>
        <div class="modal-body">
          <div class="form-group">
            <label>SSH Host *</label>
            <input
              v-model="newNode.host"
              type="text"
              placeholder="user@hostname or hostname"
              required
            />
            <small>Example: root@192.168.1.100 or node2.example.com</small>
          </div>

          <div class="form-group">
            <label>Node Role *</label>
            <select v-model="newNode.role" required>
              <option value="">Select role...</option>
              <option value="master">Master (Control Plane)</option>
              <option value="worker">Worker</option>
            </select>
          </div>

          <div v-if="addNodeError" class="error-message">{{ addNodeError }}</div>
          <div v-if="addNodeSuccess" class="success-message">{{ addNodeSuccess }}</div>
        </div>
        <div class="modal-footer">
          <button @click="closeAddNodeDialog" class="btn-secondary">Cancel</button>
          <button @click="addNode" :disabled="addingNode" class="btn-primary">
            {{ addingNode ? 'Adding...' : 'Add Node' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'

const router = useRouter()

// Check if user is admin
const isAdmin = localStorage.getItem('isAdmin') === 'true'
if (!isAdmin) {
  router.push('/projects')
}

const nodes = ref([])
const loading = ref(true)
const error = ref('')

const showAddNodeDialog = ref(false)
const newNode = ref({ host: '', role: '' })
const addingNode = ref(false)
const addNodeError = ref('')
const addNodeSuccess = ref('')

const api = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('token')}`,
    'Content-Type': 'application/json'
  }
})

const loadNodes = async () => {
  loading.value = true
  error.value = ''
  try {
    const response = await api.get('/nodes')
    nodes.value = response.data.items || []
  } catch (err) {
    error.value = 'Failed to load nodes: ' + (err.response?.data || err.message)
  } finally {
    loading.value = false
  }
}

const getNodeStatus = (node) => {
  const conditions = node.status?.conditions || []
  const readyCondition = conditions.find(c => c.type === 'Ready')
  if (readyCondition?.status === 'True') {
    return 'Ready'
  }
  return 'NotReady'
}

const getNodeRoles = (node) => {
  const labels = node.metadata?.labels || {}
  const roles = []

  if (labels['node-role.kubernetes.io/control-plane'] !== undefined ||
      labels['node-role.kubernetes.io/master'] !== undefined) {
    roles.push('control-plane')
  }

  if (labels['node-role.kubernetes.io/worker'] !== undefined) {
    roles.push('worker')
  }

  // If no specific role labels, it's typically a worker
  if (roles.length === 0) {
    roles.push('worker')
  }

  return roles
}

const getNodeVersion = (node) => {
  return node.status?.nodeInfo?.kubeletVersion || 'N/A'
}

const getNodeIP = (node) => {
  const addresses = node.status?.addresses || []
  const internalIP = addresses.find(addr => addr.type === 'InternalIP')
  return internalIP?.address || 'N/A'
}

const getOSInfo = (node) => {
  const nodeInfo = node.status?.nodeInfo
  if (!nodeInfo) return 'N/A'

  const os = nodeInfo.operatingSystem || 'unknown'
  const arch = nodeInfo.architecture || 'unknown'
  return `${os} / ${arch}`
}

const getResourceValue = (node, resourceType) => {
  const capacity = node.status?.capacity || {}
  const value = capacity[resourceType]

  if (!value) return 'N/A'

  if (resourceType === 'memory') {
    // Convert Ki to Gi
    const kiloBytes = parseInt(value.replace('Ki', ''))
    const gigaBytes = (kiloBytes / (1024 * 1024)).toFixed(2)
    return `${gigaBytes} Gi`
  }

  return value
}

const formatAge = (timestamp) => {
  if (!timestamp) return 'N/A'

  const date = new Date(timestamp)
  const now = new Date()
  const diff = Math.floor((now - date) / 1000)

  if (diff < 60) return `${diff}s`
  if (diff < 3600) return `${Math.floor(diff / 60)}m`
  if (diff < 86400) return `${Math.floor(diff / 3600)}h`
  return `${Math.floor(diff / 86400)}d`
}

const closeAddNodeDialog = () => {
  showAddNodeDialog.value = false
  newNode.value = { host: '', role: '' }
  addNodeError.value = ''
  addNodeSuccess.value = ''
}

const addNode = async () => {
  if (!newNode.value.host || !newNode.value.role) {
    addNodeError.value = 'Please fill in all required fields'
    return
  }

  addingNode.value = true
  addNodeError.value = ''
  addNodeSuccess.value = ''

  try {
    await api.post('/nodes', {
      host: newNode.value.host,
      role: newNode.value.role
    })

    addNodeSuccess.value = 'Node is being added to the cluster. This may take a few minutes...'

    setTimeout(() => {
      closeAddNodeDialog()
      loadNodes()
    }, 2000)
  } catch (err) {
    addNodeError.value = 'Failed to add node: ' + (err.response?.data || err.message)
  } finally {
    addingNode.value = false
  }
}

const confirmRemoveNode = (nodeName) => {
  if (confirm(`Are you sure you want to remove node "${nodeName}" from the cluster?`)) {
    removeNode(nodeName)
  }
}

const removeNode = async (nodeName) => {
  try {
    await api.delete(`/nodes/${nodeName}`)
    loadNodes()
  } catch (err) {
    error.value = 'Failed to remove node: ' + (err.response?.data || err.message)
  }
}

onMounted(() => {
  loadNodes()
  // Refresh every 30 seconds
  setInterval(loadNodes, 30000)
})
</script>

<style scoped>
.nodes {
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

.nodes-table {
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
  background: #e0e0e0;
  color: #666;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
  margin-right: 0.25rem;
}

.status {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
}

.status.ready {
  background: #4caf50;
  color: white;
}

.status.notready {
  background: #f44336;
  color: white;
}

.resources {
  font-size: 0.875rem;
}

.resources div {
  margin: 0.25rem 0;
}

.loading,
.error,
.empty-state {
  text-align: center;
  padding: 3rem;
  color: #666;
}

.error {
  color: #c33;
}

.empty-state p {
  margin-bottom: 1rem;
  font-size: 1.1rem;
}

.btn-primary, .btn-secondary, .btn-danger {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-weight: 500;
}

.btn-primary {
  background: #2196f3;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #1976d2;
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-secondary {
  background: #757575;
  color: white;
}

.btn-secondary:hover {
  background: #616161;
}

.btn-danger {
  background: #f44336;
  color: white;
}

.btn-danger:hover {
  background: #d32f2f;
}

.btn-small {
  padding: 0.4rem 0.8rem;
  font-size: 0.875rem;
}

/* Modal styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: white;
  border-radius: 8px;
  max-width: 500px;
  width: 90%;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid #eee;
}

.modal-header h3 {
  margin: 0;
  font-size: 1.25rem;
}

.close-btn {
  background: none;
  border: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: #999;
  padding: 0;
  width: 30px;
  height: 30px;
  line-height: 1;
}

.close-btn:hover {
  color: #333;
}

.modal-body {
  padding: 1.5rem;
}

.modal-footer {
  padding: 1rem 1.5rem;
  border-top: 1px solid #eee;
  display: flex;
  gap: 0.75rem;
  justify-content: flex-end;
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

.form-group input,
.form-group select {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 1rem;
}

.form-group input:focus,
.form-group select:focus {
  outline: none;
  border-color: #2196f3;
}

.form-group small {
  display: block;
  margin-top: 0.25rem;
  color: #666;
  font-size: 0.875rem;
}

.error-message {
  padding: 0.75rem;
  background: #ffebee;
  color: #c62828;
  border-radius: 4px;
  margin-top: 1rem;
}

.success-message {
  padding: 0.75rem;
  background: #e8f5e9;
  color: #2e7d32;
  border-radius: 4px;
  margin-top: 1rem;
}
</style>
