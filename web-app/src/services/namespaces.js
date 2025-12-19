import api from './api'

export default {
  // List namespaces with filters
  list(params = {}) {
    return api.get('/namespaces', { params })
  },

  // Get single namespace
  get(id) {
    return api.get(`/namespaces/${id}`)
  },

  // Create namespace
  create(data) {
    return api.post('/namespaces', data)
  },

  // Update namespace
  update(id, data) {
    return api.put(`/namespaces/${id}`, data)
  },

  // Delete namespace
  delete(id) {
    return api.delete(`/namespaces/${id}`)
  },

  // List servers in namespace
  listServers(namespaceId) {
    return api.get(`/namespaces/${namespaceId}/servers`)
  },

  // Add server to namespace
  addServer(namespaceId, serverId) {
    return api.post(`/namespaces/${namespaceId}/servers`, { server_id: serverId })
  },

  // Remove server from namespace
  removeServer(namespaceId, serverId) {
    return api.delete(`/namespaces/${namespaceId}/servers/${serverId}`)
  },

  // List role access for namespace
  listRoleAccess(namespaceId) {
    return api.get(`/namespaces/${namespaceId}/access`)
  },

  // Set role access for namespace
  setRoleAccess(namespaceId, roleName, accessLevel) {
    return api.post(`/namespaces/${namespaceId}/access`, {
      role_name: roleName,
      access_level: accessLevel
    })
  },

  // Remove role access from namespace
  removeRoleAccess(namespaceId, roleId) {
    return api.delete(`/namespaces/${namespaceId}/access/${roleId}`)
  }
}
