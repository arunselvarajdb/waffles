import { defineStore } from 'pinia'
import rolesApi from '@/services/roles'

export const useRolesStore = defineStore('roles', {
  state: () => ({
    roles: [],
    loading: false,
    error: null
  }),

  getters: {
    getRoleByName: (state) => (name) => {
      return state.roles.find(r => r.name === name)
    }
  },

  actions: {
    async fetchRoles() {
      this.loading = true
      this.error = null
      try {
        this.roles = await rolesApi.list()
      } catch (error) {
        this.error = error.message
        console.error('Failed to fetch roles:', error)
      } finally {
        this.loading = false
      }
    }
  }
})
