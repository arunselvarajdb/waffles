import api from './api'

const apiKeysApi = {
  /**
   * List all API keys for the current user
   * @returns {Promise<Object>} API keys list with total count
   */
  list() {
    return api.get('/api-keys')
  },

  /**
   * Get a single API key by ID
   * @param {string} id - API key ID
   * @returns {Promise<Object>} API key details
   */
  getById(id) {
    return api.get(`/api-keys/${id}`)
  },

  /**
   * Create a new API key
   * @param {Object} keyData - API key data
   * @param {string} keyData.name - Key name (required)
   * @param {string} keyData.description - Key description (optional)
   * @param {number} keyData.expires_in_days - Days until expiry (optional)
   * @param {string[]} keyData.scopes - Permission scopes (optional)
   * @param {string[]} keyData.allowed_servers - Allowed server UUIDs (optional)
   * @param {string[]} keyData.allowed_tools - Allowed tool names (optional)
   * @param {string[]} keyData.namespaces - Allowed namespace UUIDs (optional)
   * @param {string[]} keyData.ip_whitelist - Allowed IP/CIDR ranges (optional)
   * @param {boolean} keyData.read_only - Read-only access (optional)
   * @returns {Promise<Object>} Created API key with plain-text key (only returned once!)
   */
  create(keyData) {
    return api.post('/api-keys', keyData)
  },

  /**
   * Delete an API key
   * @param {string} id - API key ID
   * @returns {Promise<void>}
   */
  delete(id) {
    return api.delete(`/api-keys/${id}`)
  }
}

// Available scopes for API keys
// Based on Casbin policy roles:
// - admin: full access to everything
// - operator: servers (full), gateway (full), audit (read), namespaces (read), health (read)
// - viewer: servers (read-only), health (read)
// - user: profile and API keys only
export const API_KEY_SCOPES = [
  // Viewer+ scopes (viewer, operator, admin)
  { value: 'servers:read', label: 'Servers Read', description: 'List and view MCP servers', minRole: 'viewer' },

  // Operator+ scopes (operator, admin)
  { value: 'servers:write', label: 'Servers Write', description: 'Create, update, and delete MCP servers', minRole: 'operator' },
  { value: 'gateway:execute', label: 'Gateway Execute', description: 'Execute tools and access resources through the gateway', minRole: 'operator' },
  { value: 'audit:read', label: 'Audit Read', description: 'View audit logs', minRole: 'operator' },
  { value: 'namespaces:read', label: 'Namespaces Read', description: 'List and view namespaces', minRole: 'operator' },

  // Admin-only scopes
  { value: 'namespaces:write', label: 'Namespaces Write', description: 'Create, update, and delete namespaces', minRole: 'admin' },
  { value: 'users:read', label: 'Users Read', description: 'List and view users', minRole: 'admin' },
  { value: 'users:write', label: 'Users Write', description: 'Create, update, and delete users', minRole: 'admin' },
  { value: 'roles:read', label: 'Roles Read', description: 'List and view roles', minRole: 'admin' },
  { value: 'roles:write', label: 'Roles Write', description: 'Create, update, and delete roles', minRole: 'admin' }
]

// Helper to get scopes available for a specific role
export const getScopesForRole = (role) => {
  const roleHierarchy = {
    'admin': ['admin', 'operator', 'viewer'],
    'operator': ['operator', 'viewer'],
    'viewer': ['viewer']
  }

  const allowedMinRoles = roleHierarchy[role] || ['viewer']
  return API_KEY_SCOPES.filter(scope => allowedMinRoles.includes(scope.minRole))
}

export default apiKeysApi
