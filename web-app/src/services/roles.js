import api from './api'

const rolesApi = {
  /**
   * List all roles with user counts
   * @returns {Promise<Object>} Roles list
   */
  list() {
    return api.get('/admin/roles')
  },

  /**
   * Get a single role by ID with permissions
   * @param {string} id - Role ID
   * @returns {Promise<Object>} Role with permissions
   */
  getById(id) {
    return api.get(`/admin/roles/${id}`)
  },

  /**
   * Create a new custom role
   * @param {Object} roleData - Role data
   * @param {string} roleData.name - Role name
   * @param {string} roleData.description - Role description
   * @param {string[]} roleData.permissions - Permission IDs
   * @returns {Promise<Object>} Created role
   */
  create(roleData) {
    return api.post('/admin/roles', roleData)
  },

  /**
   * Update a role
   * @param {string} id - Role ID
   * @param {Object} roleData - Role data to update
   * @returns {Promise<Object>} Updated role
   */
  update(id, roleData) {
    return api.put(`/admin/roles/${id}`, roleData)
  },

  /**
   * Delete a custom role
   * @param {string} id - Role ID
   * @returns {Promise<void>}
   */
  delete(id) {
    return api.delete(`/admin/roles/${id}`)
  },

  /**
   * List all available permissions
   * @returns {Promise<Object>} Permissions list
   */
  listPermissions() {
    return api.get('/admin/permissions')
  }
}

export default rolesApi
