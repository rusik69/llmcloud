import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Content-Type': 'application/json'
  }
})

// Add request interceptor to include auth token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Add response interceptor to handle 401 errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Clear auth data and redirect to login
      localStorage.removeItem('token')
      localStorage.removeItem('username')
      localStorage.removeItem('isAdmin')
      localStorage.removeItem('projects')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export const projectsApi = {
  list: () => api.get('/projects'),
  get: (name) => api.get(`/projects/${name}`),
  create: (data) => api.post('/projects', data),
  delete: (name) => api.delete(`/projects/${name}`)
}

export const vmsApi = {
  list: (namespace) => api.get(`/namespaces/${namespace}/vms`),
  get: (namespace, name) => api.get(`/namespaces/${namespace}/vms/${name}`),
  create: (namespace, data) => api.post(`/namespaces/${namespace}/vms`, data),
  delete: (namespace, name) => api.delete(`/namespaces/${namespace}/vms/${name}`),
  start: (namespace, name) => api.post(`/actions/vm/${namespace}/${name}/start`),
  stop: (namespace, name) => api.post(`/actions/vm/${namespace}/${name}/stop`),
  reboot: (namespace, name) => api.post(`/actions/vm/${namespace}/${name}/reboot`),
  describe: (namespace, name) => api.get(`/describe/vm/${namespace}/${name}`),
  events: (namespace, name) => api.get(`/events/vm/${namespace}/${name}`)
}

export const modelsApi = {
  list: (namespace) => api.get(`/namespaces/${namespace}/models`),
  get: (namespace, name) => api.get(`/namespaces/${namespace}/models/${name}`),
  create: (namespace, data) => api.post(`/namespaces/${namespace}/models`, data),
  delete: (namespace, name) => api.delete(`/namespaces/${namespace}/models/${name}`)
}

export const servicesApi = {
  list: (namespace) => api.get(`/namespaces/${namespace}/services`),
  get: (namespace, name) => api.get(`/namespaces/${namespace}/services/${name}`),
  create: (namespace, data) => api.post(`/namespaces/${namespace}/services`, data),
  delete: (namespace, name) => api.delete(`/namespaces/${namespace}/services/${name}`)
}

export const nodesApi = {
  list: (namespace) => api.get(`/namespaces/${namespace}/nodes`),
  get: (namespace, name) => api.get(`/namespaces/${namespace}/nodes/${name}`),
  create: (namespace, data) => api.post(`/namespaces/${namespace}/nodes`, data),
  delete: (namespace, name) => api.delete(`/namespaces/${namespace}/nodes/${name}`),
  reboot: (namespace, name) => api.post(`/actions/node/${namespace}/${name}/reboot`),
  upgrade: (namespace, name) => api.post(`/actions/node/${namespace}/${name}/upgrade`)
}

export const namespacesApi = {
  list: () => projectsApi.list().then(res => ({
    data: { items: (res.data.items || []).map(p => ({ metadata: { name: p.metadata.name } })) }
  }))
}

export const authApi = {
  login: (username, password) => api.post('/auth/login', { username, password })
}
