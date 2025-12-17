import api from './api'

export default {
  // List servers with filters
  list(params = {}) {
    return api.get('/servers', { params })
  },

  // Get single server
  get(id) {
    return api.get(`/servers/${id}`)
  },

  // Create server
  create(data) {
    return api.post('/servers', data)
  },

  // Update server
  update(id, data) {
    return api.put(`/servers/${id}`, data)
  },

  // Delete server
  delete(id) {
    return api.delete(`/servers/${id}`)
  },

  // Toggle active/inactive
  toggle(id) {
    return api.patch(`/servers/${id}/toggle`)
  },

  // Get health status
  getHealth(id) {
    return api.get(`/servers/${id}/health`)
  },

  // Check health now
  checkHealth(id) {
    return api.post(`/servers/${id}/health`)
  }
}
