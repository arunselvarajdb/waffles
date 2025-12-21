import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useRolesStore } from './roles'

// Mock the roles API
vi.mock('@/services/roles', () => ({
  default: {
    list: vi.fn()
  }
}))

import rolesApi from '@/services/roles'

describe('useRolesStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('initial state', () => {
    it('has correct initial state', () => {
      const store = useRolesStore()
      expect(store.roles).toEqual([])
      expect(store.loading).toBe(false)
      expect(store.error).toBeNull()
    })
  })

  describe('getters', () => {
    it('getRoleByName finds role', () => {
      const store = useRolesStore()
      store.roles = [
        { id: '1', name: 'admin' },
        { id: '2', name: 'viewer' }
      ]

      const adminRole = store.getRoleByName('admin')
      expect(adminRole).toEqual({ id: '1', name: 'admin' })
    })

    it('getRoleByName returns undefined for missing role', () => {
      const store = useRolesStore()
      store.roles = [{ id: '1', name: 'admin' }]

      const role = store.getRoleByName('nonexistent')
      expect(role).toBeUndefined()
    })
  })

  describe('fetchRoles action', () => {
    it('fetches roles successfully', async () => {
      const mockRoles = [
        { id: '1', name: 'admin' },
        { id: '2', name: 'viewer' }
      ]
      vi.mocked(rolesApi.list).mockResolvedValue(mockRoles)

      const store = useRolesStore()
      await store.fetchRoles()

      expect(rolesApi.list).toHaveBeenCalled()
      expect(store.roles).toEqual(mockRoles)
      expect(store.loading).toBe(false)
    })

    it('sets loading state during fetch', async () => {
      let resolvePromise
      vi.mocked(rolesApi.list).mockImplementation(() => new Promise(resolve => {
        resolvePromise = resolve
      }))

      const store = useRolesStore()
      const fetchPromise = store.fetchRoles()

      expect(store.loading).toBe(true)

      resolvePromise([])
      await fetchPromise

      expect(store.loading).toBe(false)
    })

    it('sets error on fetch failure', async () => {
      vi.mocked(rolesApi.list).mockRejectedValue(new Error('Network error'))

      const store = useRolesStore()
      await store.fetchRoles()

      expect(store.error).toBe('Network error')
      expect(store.loading).toBe(false)
    })

    it('clears error before fetching', async () => {
      vi.mocked(rolesApi.list).mockResolvedValue([])

      const store = useRolesStore()
      store.error = 'Previous error'

      await store.fetchRoles()

      expect(store.error).toBeNull()
    })
  })
})
