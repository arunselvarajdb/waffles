import { defineStore } from 'pinia'
import serversApi from '@/services/servers'

export const useServersStore = defineStore('servers', {
  state: () => ({
    servers: [],
    currentServer: null,
    loading: false,
    error: null,
    filters: {
      search: '',
      status: 'all', // all, active, inactive
      health: 'all', // all, healthy, degraded, unhealthy
    },
    pagination: {
      page: 1,
      limit: 10,
      total: 0
    }
  }),

  getters: {
    filteredServers: (state) => {
      let filtered = state.servers

      // Search filter
      if (state.filters.search) {
        const search = state.filters.search.toLowerCase()
        filtered = filtered.filter(s =>
          s.name.toLowerCase().includes(search) ||
          s.description?.toLowerCase().includes(search)
        )
      }

      // Status filter
      if (state.filters.status !== 'all') {
        const isActive = state.filters.status === 'active'
        filtered = filtered.filter(s => s.is_active === isActive)
      }

      // Health filter
      if (state.filters.health !== 'all') {
        filtered = filtered.filter(s => s.health?.status === state.filters.health)
      }

      return filtered
    },

    activeServers: (state) => state.servers.filter(s => s.is_active),
    healthyServers: (state) => state.servers.filter(s => s.health?.status === 'healthy'),
  },

  actions: {
    async fetchServers(params = {}) {
      this.loading = true
      this.error = null
      try {
        const data = await serversApi.list({
          page: this.pagination.page,
          limit: this.pagination.limit,
          ...params
        })
        this.servers = data.servers || data
        this.pagination.total = data.total || this.servers.length
      } catch (error) {
        this.error = error.message
        console.error('Failed to fetch servers:', error)
      } finally {
        this.loading = false
      }
    },

    async createServer(serverData) {
      this.loading = true
      this.error = null
      try {
        const newServer = await serversApi.create(serverData)
        this.servers.push(newServer)
        return newServer
      } catch (error) {
        this.error = error.message
        throw error
      } finally {
        this.loading = false
      }
    },

    async updateServer(id, serverData) {
      this.loading = true
      this.error = null
      try {
        const updated = await serversApi.update(id, serverData)
        const index = this.servers.findIndex(s => s.id === id)
        if (index !== -1) {
          this.servers[index] = updated
        }
        return updated
      } catch (error) {
        this.error = error.message
        throw error
      } finally {
        this.loading = false
      }
    },

    async deleteServer(id) {
      this.loading = true
      this.error = null
      try {
        await serversApi.delete(id)
        this.servers = this.servers.filter(s => s.id !== id)
      } catch (error) {
        this.error = error.message
        throw error
      } finally {
        this.loading = false
      }
    },

    async toggleServer(id) {
      try {
        const updated = await serversApi.toggle(id)
        const index = this.servers.findIndex(s => s.id === id)
        if (index !== -1) {
          this.servers[index] = updated
        }
        return updated
      } catch (error) {
        this.error = error.message
        throw error
      }
    },

    async checkHealth(id) {
      try {
        const health = await serversApi.checkHealth(id)
        const server = this.servers.find(s => s.id === id)
        if (server) {
          server.health = health
        }
        return health
      } catch (error) {
        console.error('Failed to check health:', error)
        throw error
      }
    },

    setFilters(filters) {
      this.filters = { ...this.filters, ...filters }
    },

    setPage(page) {
      this.pagination.page = page
      this.fetchServers()
    }
  }
})
