import api from './api'

export default {
  // List all roles
  // Note: If the backend adds a /roles endpoint, we'll use that
  // For now, we'll return the known roles from the auth system
  async list() {
    try {
      // Try to fetch from backend if endpoint exists
      const response = await api.get('/roles')
      return response.roles || response
    } catch (error) {
      // If endpoint doesn't exist (404), return known default roles
      if (error.response?.status === 404) {
        return [
          { id: 'admin', name: 'admin', description: 'Full administrative access' },
          { id: 'operator', name: 'operator', description: 'Server management access' },
          { id: 'viewer', name: 'viewer', description: 'Read-only access' }
        ]
      }
      throw error
    }
  }
}
