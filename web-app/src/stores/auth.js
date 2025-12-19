import { defineStore } from 'pinia'
import api from '@/services/api'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null,
    roles: [],
    isAuthenticated: false,
    loading: false,
    error: null,
    authEnabled: null // null = unknown, true = enabled, false = disabled
  }),

  getters: {
    isAdmin: (state) => state.roles.includes('admin'),
    isOperator: (state) => state.roles.includes('operator') || state.roles.includes('admin'),
    isViewer: (state) => state.roles.includes('viewer') || state.roles.includes('operator') || state.roles.includes('admin'),
    // For backwards compatibility
    role: (state) => {
      if (state.roles.includes('admin')) return 'admin'
      if (state.roles.includes('operator')) return 'operator'
      if (state.roles.includes('viewer')) return 'viewer'
      return 'user'
    }
  },

  actions: {
    // Login with email and password
    async login(email, password) {
      this.loading = true
      this.error = null

      try {
        const response = await api.post('/auth/login', { email, password })
        this.user = response.user
        this.roles = response.user.roles || []
        this.isAuthenticated = true

        // Handle redirect after login
        const redirectPath = sessionStorage.getItem('redirectAfterLogin')
        if (redirectPath) {
          sessionStorage.removeItem('redirectAfterLogin')
          return redirectPath
        }
        return '/admin'
      } catch (error) {
        this.error = error.response?.data?.message || 'Login failed'
        throw error
      } finally {
        this.loading = false
      }
    },

    // Logout
    async logout() {
      this.loading = true

      try {
        await api.post('/auth/logout')
      } catch (error) {
        // Ignore errors on logout - session might already be expired
        console.error('Logout error:', error)
      } finally {
        this.user = null
        this.roles = []
        this.isAuthenticated = false
        this.loading = false
      }
    },

    // Check authentication status (get current user)
    async checkAuth() {
      try {
        const user = await api.get('/me')
        this.user = user
        this.roles = user.roles || []
        this.isAuthenticated = true
        return true
      } catch (error) {
        this.user = null
        this.roles = []
        this.isAuthenticated = false
        return false
      }
    },

    // Check if authentication is enabled on the server
    async checkAuthConfig() {
      try {
        const status = await api.get('/status')
        this.authEnabled = status.auth?.enabled ?? true

        // If auth is disabled, auto-login as admin for development
        if (this.authEnabled === false && !this.isAuthenticated) {
          this.loginAsAdmin()
        }

        return this.authEnabled
      } catch (error) {
        // If we can't reach the server, assume auth is enabled
        this.authEnabled = true
        return true
      }
    },

    // Demo login - actually authenticate with backend using test credentials
    async loginAsAdmin() {
      try {
        const response = await api.post('/auth/login', {
          email: 'admin@example.com',
          password: 'admin123'
        })
        this.user = response.user
        this.roles = response.user.roles || ['admin']
        this.isAuthenticated = true
        return true
      } catch (error) {
        // Fallback to mock if backend auth fails
        console.warn('Backend admin login failed, using mock:', error)
        this.user = { id: 'demo-admin', name: 'Admin User', email: 'admin@example.com' }
        this.roles = ['admin']
        this.isAuthenticated = true
        return false
      }
    },

    async loginAsViewer() {
      try {
        const response = await api.post('/auth/login', {
          email: 'viewer@example.com',
          password: 'viewer123'
        })
        this.user = response.user
        this.roles = response.user.roles || ['viewer']
        this.isAuthenticated = true
        return true
      } catch (error) {
        // Fallback to mock if backend auth fails
        console.warn('Backend viewer login failed, using mock:', error)
        this.user = { id: 'demo-viewer', name: 'Viewer User', email: 'viewer@example.com' }
        this.roles = ['viewer']
        this.isAuthenticated = true
        return false
      }
    },

    // Clear any errors
    clearError() {
      this.error = null
    }
  },

  // Persist auth state in sessionStorage
  persist: {
    storage: sessionStorage,
    paths: ['user', 'roles', 'isAuthenticated', 'authEnabled']
  }
})
