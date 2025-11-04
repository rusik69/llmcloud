import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import Login from './views/Login.vue'
import Projects from './views/Projects.vue'
import VirtualMachines from './views/VirtualMachines.vue'
import VMDetails from './views/VMDetails.vue'
import LLMModels from './views/LLMModels.vue'
import Services from './views/Services.vue'
import Nodes from './views/Nodes.vue'
import Users from './views/Users.vue'

const routes = [
  { path: '/login', component: Login, meta: { requiresAuth: false } },
  { path: '/', redirect: '/projects' },
  { path: '/projects', component: Projects, meta: { requiresAuth: true } },
  { path: '/vms', component: VirtualMachines, meta: { requiresAuth: true } },
  { path: '/vms/:namespace/:name', component: VMDetails, meta: { requiresAuth: true } },
  { path: '/models', component: LLMModels, meta: { requiresAuth: true } },
  { path: '/services', component: Services, meta: { requiresAuth: true } },
  { path: '/nodes', component: Nodes, meta: { requiresAuth: true, requiresAdmin: true } },
  { path: '/users', component: Users, meta: { requiresAuth: true, requiresAdmin: true } }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// Navigation guard to check authentication
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  const isAdmin = localStorage.getItem('isAdmin') === 'true'
  const requiresAuth = to.matched.some(record => record.meta.requiresAuth !== false)
  const requiresAdmin = to.matched.some(record => record.meta.requiresAdmin === true)

  if (requiresAuth && !token) {
    // Redirect to login if trying to access protected route without token
    next('/login')
  } else if (requiresAdmin && !isAdmin) {
    // Redirect to projects if trying to access admin route without admin privileges
    next('/projects')
  } else if (to.path === '/login' && token) {
    // Redirect to projects if already logged in and trying to access login page
    next('/projects')
  } else {
    next()
  }
})

createApp(App).use(router).mount('#app')
