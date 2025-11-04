<template>
  <div class="users-page">
    <div class="page-header">
      <h1>User Management</h1>
      <button @click="showCreateForm = true" class="btn btn-primary">Create User</button>
    </div>

    <!-- Create User Modal -->
    <div v-if="showCreateForm" class="modal-overlay" @click.self="showCreateForm = false">
      <div class="modal">
        <div class="modal-header">
          <h2>Create New User</h2>
          <button @click="showCreateForm = false" class="close-btn">&times;</button>
        </div>
        <form @submit.prevent="createUser" class="modal-body">
          <div class="form-group">
            <label for="username">Username *</label>
            <input
              type="text"
              id="username"
              v-model="newUser.username"
              required
              pattern="[a-z0-9-]+"
              placeholder="lowercase-with-dashes"
            />
          </div>

          <div class="form-group">
            <label for="email">Email</label>
            <input
              type="email"
              id="email"
              v-model="newUser.email"
              placeholder="user@example.com"
            />
          </div>

          <div class="form-group">
            <label for="password">Password *</label>
            <input
              type="password"
              id="password"
              v-model="newUser.password"
              required
              minlength="8"
              placeholder="Min 8 characters"
            />
          </div>

          <div class="form-group">
            <label>
              <input type="checkbox" v-model="newUser.isAdmin" />
              Administrator
            </label>
          </div>

          <div class="form-group">
            <label for="projects">Projects (comma-separated)</label>
            <input
              type="text"
              id="projects"
              v-model="newUser.projectsInput"
              placeholder="project1, project2"
            />
            <small>Leave empty for admin users to access all projects</small>
          </div>

          <div v-if="createError" class="error-message">{{ createError }}</div>

          <div class="modal-footer">
            <button type="button" @click="showCreateForm = false" class="btn btn-secondary">Cancel</button>
            <button type="submit" class="btn btn-primary" :disabled="creating">
              {{ creating ? 'Creating...' : 'Create User' }}
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Users List -->
    <div v-if="loading" class="loading">Loading users...</div>
    <div v-else-if="error" class="error">{{ error }}</div>
    <div v-else-if="!users || users.length === 0" class="empty-state">
      <p>No users found</p>
    </div>
    <table v-else class="users-table">
      <thead>
        <tr>
          <th>Username</th>
          <th>Email</th>
          <th>Role</th>
          <th>Projects</th>
          <th>Status</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="user in users" :key="user.metadata.name">
          <td>
            <strong>{{ user.spec.username }}</strong>
          </td>
          <td>{{ user.spec.email || '-' }}</td>
          <td>
            <span :class="['badge', user.spec.isAdmin ? 'badge-admin' : 'badge-user']">
              {{ user.spec.isAdmin ? 'Admin' : 'User' }}
            </span>
          </td>
          <td>
            <span v-if="user.spec.isAdmin" class="text-muted">All Projects</span>
            <span v-else-if="!user.spec.projects || user.spec.projects.length === 0" class="text-muted">None</span>
            <span v-else>{{ user.spec.projects.join(', ') }}</span>
          </td>
          <td>
            <span :class="['badge', user.spec.disabled ? 'badge-disabled' : 'badge-active']">
              {{ user.spec.disabled ? 'Disabled' : 'Active' }}
            </span>
          </td>
          <td>
            <button
              v-if="user.spec.username !== 'root'"
              @click="toggleUserStatus(user)"
              class="btn btn-sm btn-secondary"
              :disabled="toggling === user.metadata.name"
            >
              {{ user.spec.disabled ? 'Enable' : 'Disable' }}
            </button>
            <button
              v-if="user.spec.username !== 'root'"
              @click="deleteUser(user)"
              class="btn btn-sm btn-danger"
              :disabled="deleting === user.metadata.name"
            >
              Delete
            </button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'

const router = useRouter()

const users = ref([])
const loading = ref(true)
const error = ref('')
const showCreateForm = ref(false)
const creating = ref(false)
const createError = ref('')
const toggling = ref(null)
const deleting = ref(null)

const newUser = ref({
  username: '',
  email: '',
  password: '',
  isAdmin: false,
  projectsInput: ''
})

// Check if current user is admin
const isAdmin = localStorage.getItem('isAdmin') === 'true'
if (!isAdmin) {
  router.push('/projects')
}

const api = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Authorization': `Bearer ${localStorage.getItem('token')}`,
    'Content-Type': 'application/json'
  }
})

const loadUsers = async () => {
  loading.value = true
  error.value = ''
  try {
    const response = await api.get('/users')
    users.value = response.data.items || []
  } catch (err) {
    error.value = 'Failed to load users: ' + (err.response?.data || err.message)
  } finally {
    loading.value = false
  }
}

const createUser = async () => {
  creating.value = true
  createError.value = ''

  try {
    const projects = newUser.value.projectsInput
      ? newUser.value.projectsInput.split(',').map(p => p.trim()).filter(p => p)
      : []

    const userData = {
      apiVersion: 'llmcloud.llmcloud.io/v1alpha1',
      kind: 'User',
      metadata: {
        name: newUser.value.username
      },
      spec: {
        username: newUser.value.username,
        email: newUser.value.email,
        password: newUser.value.password, // Backend will hash this
        isAdmin: newUser.value.isAdmin,
        projects: projects,
        disabled: false
      }
    }

    await api.post('/users', userData)

    showCreateForm.value = false
    newUser.value = {
      username: '',
      email: '',
      password: '',
      isAdmin: false,
      projectsInput: ''
    }

    await loadUsers()
  } catch (err) {
    createError.value = err.response?.data || err.message
  } finally {
    creating.value = false
  }
}

const toggleUserStatus = async (user) => {
  if (!confirm(`${user.spec.disabled ? 'Enable' : 'Disable'} user ${user.spec.username}?`)) {
    return
  }

  toggling.value = user.metadata.name
  try {
    const updated = {
      ...user,
      spec: {
        ...user.spec,
        disabled: !user.spec.disabled
      }
    }
    delete updated.spec.password // Don't send password in update

    await api.put(`/users/${user.metadata.name}`, updated)
    await loadUsers()
  } catch (err) {
    alert('Failed to update user: ' + (err.response?.data || err.message))
  } finally {
    toggling.value = null
  }
}

const deleteUser = async (user) => {
  if (!confirm(`Delete user ${user.spec.username}? This action cannot be undone.`)) {
    return
  }

  deleting.value = user.metadata.name
  try {
    await api.delete(`/users/${user.metadata.name}`)
    await loadUsers()
  } catch (err) {
    alert('Failed to delete user: ' + (err.response?.data || err.message))
  } finally {
    deleting.value = null
  }
}

onMounted(() => {
  loadUsers()
})
</script>

<style scoped>
.users-page {
  max-width: 1200px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
}

.page-header h1 {
  margin: 0;
}

.users-table {
  width: 100%;
  border-collapse: collapse;
  background: white;
  border-radius: 8px;
  overflow: hidden;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.users-table th {
  background: #f5f5f5;
  padding: 1rem;
  text-align: left;
  font-weight: 600;
  border-bottom: 2px solid #ddd;
}

.users-table td {
  padding: 1rem;
  border-bottom: 1px solid #eee;
}

.users-table tbody tr:hover {
  background: #f9f9f9;
}

.badge {
  display: inline-block;
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
}

.badge-admin {
  background: #667eea;
  color: white;
}

.badge-user {
  background: #e0e0e0;
  color: #666;
}

.badge-active {
  background: #4caf50;
  color: white;
}

.badge-disabled {
  background: #f44336;
  color: white;
}

.text-muted {
  color: #999;
  font-style: italic;
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

.btn-secondary {
  background: #e0e0e0;
  color: #333;
  margin-right: 0.5rem;
}

.btn-secondary:hover:not(:disabled) {
  background: #d0d0d0;
}

.btn-danger {
  background: #f44336;
  color: white;
}

.btn-danger:hover:not(:disabled) {
  background: #d32f2f;
}

.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.8rem;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.modal-overlay {
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

.modal {
  background: white;
  border-radius: 8px;
  width: 90%;
  max-width: 500px;
  max-height: 90vh;
  overflow: auto;
  box-shadow: 0 10px 40px rgba(0,0,0,0.3);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid #eee;
}

.modal-header h2 {
  margin: 0;
  font-size: 1.25rem;
}

.close-btn {
  background: none;
  border: none;
  font-size: 2rem;
  cursor: pointer;
  color: #999;
  line-height: 1;
  padding: 0;
  width: 2rem;
  height: 2rem;
}

.close-btn:hover {
  color: #333;
}

.modal-body {
  padding: 1.5rem;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  padding-top: 1rem;
  border-top: 1px solid #eee;
  margin-top: 1rem;
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

.form-group input[type="text"],
.form-group input[type="email"],
.form-group input[type="password"] {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 0.875rem;
}

.form-group input[type="text"]:focus,
.form-group input[type="email"]:focus,
.form-group input[type="password"]:focus {
  outline: none;
  border-color: #667eea;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

.form-group input[type="checkbox"] {
  margin-right: 0.5rem;
}

.form-group small {
  display: block;
  margin-top: 0.25rem;
  color: #666;
  font-size: 0.75rem;
}

.error-message {
  background: #fee;
  color: #c33;
  padding: 0.75rem;
  border-radius: 4px;
  margin-bottom: 1rem;
  font-size: 0.875rem;
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
</style>
