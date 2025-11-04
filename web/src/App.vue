<template>
  <div class="app">
    <nav v-if="isAuthenticated" class="navbar">
      <div class="navbar-brand">
        <h1>LLMCloud Operator</h1>
      </div>
      <div class="navbar-menu">
        <router-link to="/projects" class="nav-item">Projects</router-link>
        <router-link to="/vms" class="nav-item">Virtual Machines</router-link>
        <router-link to="/models" class="nav-item">LLM Models</router-link>
        <router-link to="/services" class="nav-item">Services</router-link>
        <router-link v-if="isAdmin" to="/nodes" class="nav-item">Cluster Nodes</router-link>
        <router-link v-if="isAdmin" to="/users" class="nav-item">Users</router-link>
      </div>
      <div class="navbar-user">
        <span class="username">{{ username }}</span>
        <button @click="handleLogout" class="btn-logout">Logout</button>
      </div>
    </nav>
    <main class="content" :class="{ 'no-nav': !isAuthenticated }">
      <router-view />
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'

const router = useRouter()
const route = useRoute()

const username = ref('')
const isAuthenticated = ref(false)
const isAdmin = ref(false)

const updateAuthState = () => {
  const token = localStorage.getItem('token')
  isAuthenticated.value = !!token
  username.value = localStorage.getItem('username') || ''
  isAdmin.value = localStorage.getItem('isAdmin') === 'true'
}

const handleLogout = () => {
  localStorage.removeItem('token')
  localStorage.removeItem('username')
  localStorage.removeItem('isAdmin')
  localStorage.removeItem('projects')
  updateAuthState()
  router.push('/login')
}

onMounted(() => {
  updateAuthState()

  // Update auth state when route changes (after login)
  router.afterEach(() => {
    updateAuthState()
  })
})

// Watch route to update auth state
watch(() => route.path, () => {
  updateAuthState()
})
</script>

<style scoped>
.app {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
}

.navbar {
  background: #2c3e50;
  color: white;
  padding: 1rem 2rem;
  display: flex;
  justify-content: space-between;
  align-items: center;
  box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.navbar-brand h1 {
  font-size: 1.5rem;
  font-weight: 600;
}

.navbar-menu {
  display: flex;
  gap: 2rem;
  flex: 1;
  margin-left: 3rem;
}

.navbar-user {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.username {
  color: #ecf0f1;
  font-weight: 500;
}

.btn-logout {
  background: #e74c3c;
  color: white;
  border: none;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.9rem;
  transition: background 0.2s;
}

.btn-logout:hover {
  background: #c0392b;
}

.nav-item {
  color: #ecf0f1;
  text-decoration: none;
  padding: 0.5rem 1rem;
  border-radius: 4px;
  transition: background 0.2s;
}

.nav-item:hover {
  background: rgba(255,255,255,0.1);
}

.nav-item.router-link-active {
  background: #3498db;
}

.content {
  flex: 1;
  padding: 2rem;
  max-width: 1400px;
  margin: 0 auto;
  width: 100%;
}

.content.no-nav {
  padding: 0;
  max-width: none;
}
</style>
