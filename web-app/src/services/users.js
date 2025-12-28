import api from './api'

const usersApi = {
  /**
   * List all users with pagination
   * @param {Object} params - Query parameters
   * @param {number} params.page - Page number
   * @param {number} params.page_size - Items per page
   * @param {string} params.search - Search query
   * @returns {Promise<Object>} Users list with pagination info
   */
  list(params = {}) {
    return api.get('/admin/users', { params })
  },

  /**
   * Get a single user by ID
   * @param {string} id - User ID
   * @returns {Promise<Object>} User details with roles
   */
  getById(id) {
    return api.get(`/admin/users/${id}`)
  },

  /**
   * Create a new user
   * @param {Object} userData - User data
   * @param {string} userData.email - User email
   * @param {string} userData.name - User name
   * @param {string} userData.password - Optional password
   * @param {string} userData.role - Optional role (defaults to 'user')
   * @returns {Promise<Object>} Created user with temp_password if generated
   */
  create(userData) {
    return api.post('/admin/users', userData)
  },

  /**
   * Update user information
   * @param {string} id - User ID
   * @param {Object} userData - User data to update
   * @returns {Promise<Object>} Updated user
   */
  update(id, userData) {
    return api.put(`/admin/users/${id}`, userData)
  },

  /**
   * Deactivate a user
   * @param {string} id - User ID
   * @returns {Promise<void>}
   */
  deactivate(id) {
    return api.delete(`/admin/users/${id}`)
  },

  /**
   * Update user roles
   * @param {string} id - User ID
   * @param {string[]} roles - Array of role names
   * @returns {Promise<Object>} Updated user with roles
   */
  updateRoles(id, roles) {
    return api.put(`/admin/users/${id}/roles`, { roles })
  },

  /**
   * Reset user password (generates temp password)
   * @param {string} id - User ID
   * @returns {Promise<Object>} Object containing temp_password
   */
  resetPassword(id) {
    return api.post(`/admin/users/${id}/reset-password`)
  },

  /**
   * List user sessions
   * @param {string} id - User ID
   * @returns {Promise<Object>} User sessions
   */
  listSessions(id) {
    return api.get(`/admin/users/${id}/sessions`)
  },

  /**
   * Revoke a specific session
   * @param {string} userId - User ID
   * @param {string} sessionId - Session ID
   * @returns {Promise<void>}
   */
  revokeSession(userId, sessionId) {
    return api.delete(`/admin/users/${userId}/sessions/${sessionId}`)
  },

  /**
   * Revoke all user sessions
   * @param {string} id - User ID
   * @returns {Promise<void>}
   */
  revokeAllSessions(id) {
    return api.delete(`/admin/users/${id}/sessions`)
  }
}

export default usersApi
