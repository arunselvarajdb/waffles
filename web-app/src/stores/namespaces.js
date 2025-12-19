import { defineStore } from 'pinia'
import namespacesApi from '@/services/namespaces'

export const useNamespacesStore = defineStore('namespaces', {
  state: () => ({
    namespaces: [],
    currentNamespace: null,
    currentNamespaceServers: [],
    currentNamespaceAccess: [],
    loading: false,
    error: null,
    filters: {
      search: ''
    },
    pagination: {
      page: 1,
      limit: 10,
      total: 0
    }
  }),

  getters: {
    filteredNamespaces: (state) => {
      let filtered = state.namespaces

      // Search filter
      if (state.filters.search) {
        const search = state.filters.search.toLowerCase()
        filtered = filtered.filter(n =>
          n.name.toLowerCase().includes(search) ||
          n.description?.toLowerCase().includes(search)
        )
      }

      return filtered
    },

    namespaceCount: (state) => state.namespaces.length
  },

  actions: {
    async fetchNamespaces(params = {}) {
      this.loading = true
      this.error = null
      try {
        const data = await namespacesApi.list({
          page: this.pagination.page,
          limit: this.pagination.limit,
          ...params
        })
        this.namespaces = data.namespaces || data
        this.pagination.total = data.count || this.namespaces.length
      } catch (error) {
        this.error = error.message
        console.error('Failed to fetch namespaces:', error)
      } finally {
        this.loading = false
      }
    },

    async fetchNamespace(id) {
      this.loading = true
      this.error = null
      try {
        const namespace = await namespacesApi.get(id)
        this.currentNamespace = namespace
        return namespace
      } catch (error) {
        this.error = error.message
        throw error
      } finally {
        this.loading = false
      }
    },

    async createNamespace(namespaceData) {
      this.loading = true
      this.error = null
      try {
        const newNamespace = await namespacesApi.create(namespaceData)
        this.namespaces.push(newNamespace)
        return newNamespace
      } catch (error) {
        this.error = error.message
        throw error
      } finally {
        this.loading = false
      }
    },

    async updateNamespace(id, namespaceData) {
      this.loading = true
      this.error = null
      try {
        const updated = await namespacesApi.update(id, namespaceData)
        const index = this.namespaces.findIndex(n => n.id === id)
        if (index !== -1) {
          this.namespaces[index] = updated
        }
        if (this.currentNamespace?.id === id) {
          this.currentNamespace = updated
        }
        return updated
      } catch (error) {
        this.error = error.message
        throw error
      } finally {
        this.loading = false
      }
    },

    async deleteNamespace(id) {
      this.loading = true
      this.error = null
      try {
        await namespacesApi.delete(id)
        this.namespaces = this.namespaces.filter(n => n.id !== id)
        if (this.currentNamespace?.id === id) {
          this.currentNamespace = null
        }
      } catch (error) {
        this.error = error.message
        throw error
      } finally {
        this.loading = false
      }
    },

    // Server membership management
    async fetchNamespaceServers(namespaceId) {
      try {
        const data = await namespacesApi.listServers(namespaceId)
        this.currentNamespaceServers = data.servers || []
        return this.currentNamespaceServers
      } catch (error) {
        console.error('Failed to fetch namespace servers:', error)
        throw error
      }
    },

    async addServerToNamespace(namespaceId, serverId) {
      try {
        await namespacesApi.addServer(namespaceId, serverId)
        await this.fetchNamespaceServers(namespaceId)
      } catch (error) {
        this.error = error.message
        throw error
      }
    },

    async removeServerFromNamespace(namespaceId, serverId) {
      try {
        await namespacesApi.removeServer(namespaceId, serverId)
        this.currentNamespaceServers = this.currentNamespaceServers.filter(
          s => s.server_id !== serverId
        )
      } catch (error) {
        this.error = error.message
        throw error
      }
    },

    // Role access management
    async fetchNamespaceAccess(namespaceId) {
      try {
        const data = await namespacesApi.listRoleAccess(namespaceId)
        this.currentNamespaceAccess = data.access_entries || []
        return this.currentNamespaceAccess
      } catch (error) {
        console.error('Failed to fetch namespace access:', error)
        throw error
      }
    },

    async setRoleAccess(namespaceId, roleName, accessLevel) {
      try {
        await namespacesApi.setRoleAccess(namespaceId, roleName, accessLevel)
        await this.fetchNamespaceAccess(namespaceId)
      } catch (error) {
        this.error = error.message
        throw error
      }
    },

    async removeRoleAccess(namespaceId, roleId) {
      try {
        await namespacesApi.removeRoleAccess(namespaceId, roleId)
        this.currentNamespaceAccess = this.currentNamespaceAccess.filter(
          a => a.role_id !== roleId
        )
      } catch (error) {
        this.error = error.message
        throw error
      }
    },

    setFilters(filters) {
      this.filters = { ...this.filters, ...filters }
    },

    setPage(page) {
      this.pagination.page = page
      this.fetchNamespaces()
    },

    clearCurrentNamespace() {
      this.currentNamespace = null
      this.currentNamespaceServers = []
      this.currentNamespaceAccess = []
    }
  }
})
