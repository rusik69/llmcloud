<template>
  <div class="projects">
    <div class="header">
      <h2>Projects</h2>
      <button @click="showCreateDialog = true" class="btn btn-primary">Create Project</button>
    </div>

    <div class="projects-grid">
      <div v-for="project in projects" :key="project.metadata.name" class="card">
        <div class="card-header">
          <h3>{{ project.metadata.name }}</h3>
          <span :class="['status', project.status.phase]">{{ project.status.phase }}</span>
        </div>
        <div class="card-body">
          <p v-if="project.spec.description">{{ project.spec.description }}</p>
          <div class="stats">
            <div class="stat">
              <span class="stat-label">Members</span>
              <span class="stat-value">{{ project.spec.members?.length || 0 }}</span>
            </div>
            <div class="stat">
              <span class="stat-label">VMs</span>
              <span class="stat-value">{{ project.status.vmCount || 0 }}</span>
            </div>
            <div class="stat">
              <span class="stat-label">Models</span>
              <span class="stat-value">{{ project.status.llmModelCount || 0 }}</span>
            </div>
            <div class="stat">
              <span class="stat-label">Services</span>
              <span class="stat-value">{{ project.status.serviceCount || 0 }}</span>
            </div>
          </div>
        </div>
        <div class="card-footer">
          <button @click="deleteProject(project.metadata.name)" class="btn btn-danger">Delete</button>
        </div>
      </div>
    </div>

    <div v-if="showCreateDialog" class="modal" @click="showCreateDialog = false">
      <div class="modal-content" @click.stop>
        <h3>Create Project</h3>
        <form @submit.prevent="createProject">
          <div class="form-group">
            <label>Name</label>
            <input v-model="newProject.name" required class="form-control" />
          </div>
          <div class="form-group">
            <label>Description</label>
            <textarea v-model="newProject.description" class="form-control"></textarea>
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
import { projectsApi } from '../api/client'

export default {
  setup() {
    const projects = ref([])
    const showCreateDialog = ref(false)
    const newProject = ref({ name: '', description: '' })

    const loadProjects = async () => {
      try {
        const response = await projectsApi.list()
        projects.value = response.data.items || []
      } catch (error) {
        console.error('Failed to load projects:', error)
      }
    }

    const createProject = async () => {
      try {
        await projectsApi.create({
          apiVersion: 'llmcloud.llmcloud.io/v1alpha1',
          kind: 'Project',
          metadata: { name: newProject.value.name },
          spec: { description: newProject.value.description, members: [] }
        })
        showCreateDialog.value = false
        newProject.value = { name: '', description: '' }
        await loadProjects()
      } catch (error) {
        console.error('Failed to create project:', error)
      }
    }

    const deleteProject = async (name) => {
      if (!confirm(`Delete project ${name}?`)) return
      try {
        await projectsApi.delete(name)
        await loadProjects()
      } catch (error) {
        console.error('Failed to delete project:', error)
      }
    }

    onMounted(loadProjects)

    return { projects, showCreateDialog, newProject, createProject, deleteProject }
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

.projects-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: 1.5rem;
}

.card {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  overflow: hidden;
}

.card-header {
  padding: 1.5rem;
  border-bottom: 1px solid #e0e0e0;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header h3 {
  font-size: 1.25rem;
  font-weight: 600;
}

.status {
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.875rem;
  font-weight: 500;
}

.status.Active {
  background: #e8f5e9;
  color: #2e7d32;
}

.status.Error {
  background: #ffebee;
  color: #c62828;
}

.card-body {
  padding: 1.5rem;
}

.stats {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
  margin-top: 1rem;
}

.stat {
  display: flex;
  flex-direction: column;
}

.stat-label {
  font-size: 0.875rem;
  color: #666;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: 600;
  color: #2c3e50;
}

.card-footer {
  padding: 1rem 1.5rem;
  background: #f5f5f5;
  border-top: 1px solid #e0e0e0;
  display: flex;
  justify-content: flex-end;
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

.btn-primary {
  background: #3498db;
  color: white;
}

.btn-primary:hover {
  background: #2980b9;
}

.btn-danger {
  background: #e74c3c;
  color: white;
}

.btn-danger:hover {
  background: #c0392b;
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

textarea.form-control {
  resize: vertical;
  min-height: 80px;
}

.form-actions {
  display: flex;
  gap: 1rem;
  justify-content: flex-end;
}
</style>
