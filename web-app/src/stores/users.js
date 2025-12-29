import { defineStore } from 'pinia'
import usersApi from '@/services/users'

export const useUsersStore = defineStore('users', {
  state: () => ({
    users: [],
    currentUser: null,
    loading: false,
    error: null,
    filters: {
      search: '',
      status: 'all', // all, active, inactive
      role: 'all'    // all, admin, operator, user, readonly
    },
    pagination: {
      page: 1,
      pageSize: 10,
      total: 0,
      totalPages: 0
    }
  }),

  getters: {
    filteredUsers: (state) => {
      let filtered = state.users

      // Search filter (client-side for current page)
      if (state.filters.search) {
        const search = state.filters.search.toLowerCase()
        filtered = filtered.filter(u =>
          u.email?.toLowerCase().includes(search) ||
          u.name?.toLowerCase().includes(search)
        )
      }

      // Status filter
      if (state.filters.status !== 'all') {
        const isActive = state.filters.status === 'active'
        filtered = filtered.filter(u => u.is_active === isActive)
      }

      // Role filter
      if (state.filters.role !== 'all') {
        filtered = filtered.filter(u =>
          u.roles?.includes(state.filters.role)
        )
      }

      return filtered
    },

    activeUsers: (state) => state.users.filter(u => u.is_active),

    adminUsers: (state) => state.users.filter(u => u.roles?.includes('admin')),

    userStats: (state) => {
      const total = state.pagination.total
      const active = state.users.filter(u => u.is_active).length
      const inactive = state.users.filter(u => !u.is_active).length
      const admins = state.users.filter(u => u.roles?.includes('admin')).length
      return { total, active, inactive, admins }
    }
  },

  actions: {
    async fetchUsers(params = {}) {
      this.loading = true
      this.error = null
      try {
        const data = await usersApi.list({
          page: this.pagination.page,
          page_size: this.pagination.pageSize,
          search: this.filters.search || undefined,
          ...params
        })
        this.users = data.users || []
        this.pagination.total = data.total || 0
        this.pagination.totalPages = data.total_pages || 0
        this.pagination.page = data.page || 1
        this.pagination.pageSize = data.page_size || 10
      } catch (error) {
        this.error = error.response?.data?.error || error.message
        console.error('Failed to fetch users:', error)
      } finally {
        this.loading = false
      }
    },

    async fetchUser(id) {
      this.loading = true
      this.error = null
      try {
        const user = await usersApi.getById(id)
        this.currentUser = user
        return user
      } catch (error) {
        this.error = error.response?.data?.error || error.message
        throw error
      } finally {
        this.loading = false
      }
    },

    async createUser(userData) {
      this.loading = true
      this.error = null
      try {
        const response = await usersApi.create(userData)
        // Response contains { user, temp_password }
        if (response.user) {
          this.users.unshift(response.user)
        }
        return response
      } catch (error) {
        this.error = error.response?.data?.error || error.message
        throw error
      } finally {
        this.loading = false
      }
    },

    async updateUser(id, userData) {
      this.loading = true
      this.error = null
      try {
        const updated = await usersApi.update(id, userData)
        const index = this.users.findIndex(u => u.id === id)
        if (index !== -1) {
          this.users[index] = updated
        }
        return updated
      } catch (error) {
        this.error = error.response?.data?.error || error.message
        throw error
      } finally {
        this.loading = false
      }
    },

    async deactivateUser(id) {
      this.loading = true
      this.error = null
      try {
        await usersApi.deactivate(id)
        const index = this.users.findIndex(u => u.id === id)
        if (index !== -1) {
          this.users[index].is_active = false
        }
      } catch (error) {
        this.error = error.response?.data?.error || error.message
        throw error
      } finally {
        this.loading = false
      }
    },

    async updateUserRoles(id, roles) {
      this.loading = true
      this.error = null
      try {
        const updated = await usersApi.updateRoles(id, roles)
        const index = this.users.findIndex(u => u.id === id)
        if (index !== -1) {
          this.users[index] = updated
        }
        return updated
      } catch (error) {
        this.error = error.response?.data?.error || error.message
        throw error
      } finally {
        this.loading = false
      }
    },

    async resetUserPassword(id) {
      this.loading = true
      this.error = null
      try {
        const response = await usersApi.resetPassword(id)
        return response // Contains temp_password
      } catch (error) {
        this.error = error.response?.data?.error || error.message
        throw error
      } finally {
        this.loading = false
      }
    },

    setFilters(filters) {
      this.filters = { ...this.filters, ...filters }
    },

    setPage(page) {
      this.pagination.page = page
      this.fetchUsers()
    },

    setPageSize(pageSize) {
      this.pagination.pageSize = pageSize
      this.pagination.page = 1
      this.fetchUsers()
    },

    clearError() {
      this.error = null
    }
  }
})
