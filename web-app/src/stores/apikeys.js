import { defineStore } from 'pinia'
import apiKeysApi from '@/services/apikeys'

export const useApiKeysStore = defineStore('apikeys', {
  state: () => ({
    apiKeys: [],
    currentKey: null,
    loading: false,
    error: null,
    // Store the newly created key temporarily (shown only once)
    newlyCreatedKey: null
  }),

  getters: {
    activeKeys: (state) => state.apiKeys.filter(k => !k.expires_at || new Date(k.expires_at) > new Date()),

    expiredKeys: (state) => state.apiKeys.filter(k => k.expires_at && new Date(k.expires_at) <= new Date()),

    keyStats: (state) => {
      const total = state.apiKeys.length
      const active = state.apiKeys.filter(k => !k.expires_at || new Date(k.expires_at) > new Date()).length
      const expired = total - active
      const recentlyUsed = state.apiKeys.filter(k => {
        if (!k.last_used_at) return false
        const lastUsed = new Date(k.last_used_at)
        const dayAgo = new Date()
        dayAgo.setDate(dayAgo.getDate() - 1)
        return lastUsed > dayAgo
      }).length
      return { total, active, expired, recentlyUsed }
    }
  },

  actions: {
    async fetchApiKeys() {
      this.loading = true
      this.error = null
      try {
        const data = await apiKeysApi.list()
        this.apiKeys = data.api_keys || []
      } catch (error) {
        this.error = error.response?.data?.error || error.message
        console.error('Failed to fetch API keys:', error)
      } finally {
        this.loading = false
      }
    },

    async fetchApiKey(id) {
      this.loading = true
      this.error = null
      try {
        const key = await apiKeysApi.getById(id)
        this.currentKey = key
        return key
      } catch (error) {
        this.error = error.response?.data?.error || error.message
        throw error
      } finally {
        this.loading = false
      }
    },

    async createApiKey(keyData) {
      this.loading = true
      this.error = null
      try {
        const response = await apiKeysApi.create(keyData)
        // Store the newly created key with plain-text key (shown only once)
        this.newlyCreatedKey = {
          ...response,
          key: response.key // Plain-text key
        }
        // Add to list (without the plain-text key for security)
        this.apiKeys.unshift({
          id: response.id,
          name: response.name,
          key_prefix: response.key?.substring(0, 14) || 'mcpgw_****',
          expires_at: response.expires_at,
          created_at: response.created_at
        })
        return response
      } catch (error) {
        this.error = error.response?.data?.message || error.response?.data?.error || error.message
        throw error
      } finally {
        this.loading = false
      }
    },

    async deleteApiKey(id) {
      this.loading = true
      this.error = null
      try {
        await apiKeysApi.delete(id)
        this.apiKeys = this.apiKeys.filter(k => k.id !== id)
      } catch (error) {
        this.error = error.response?.data?.error || error.message
        throw error
      } finally {
        this.loading = false
      }
    },

    clearNewlyCreatedKey() {
      this.newlyCreatedKey = null
    },

    clearError() {
      this.error = null
    }
  }
})
