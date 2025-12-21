import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useNamespacesStore } from './namespaces'

// Mock the namespaces API
vi.mock('@/services/namespaces', () => ({
  default: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    listServers: vi.fn(),
    addServer: vi.fn(),
    removeServer: vi.fn(),
    listRoleAccess: vi.fn(),
    setRoleAccess: vi.fn(),
    removeRoleAccess: vi.fn()
  }
}))

import namespacesApi from '@/services/namespaces'

describe('useNamespacesStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('initial state', () => {
    it('has correct initial state', () => {
      const store = useNamespacesStore()
      expect(store.namespaces).toEqual([])
      expect(store.currentNamespace).toBeNull()
      expect(store.currentNamespaceServers).toEqual([])
      expect(store.currentNamespaceAccess).toEqual([])
      expect(store.loading).toBe(false)
      expect(store.error).toBeNull()
      expect(store.filters.search).toBe('')
      expect(store.pagination.page).toBe(1)
      expect(store.pagination.limit).toBe(10)
    })
  })

  describe('getters', () => {
    it('filteredNamespaces returns all namespaces when no filter', () => {
      const store = useNamespacesStore()
      store.namespaces = [
        { id: '1', name: 'prod', description: 'Production' },
        { id: '2', name: 'dev', description: 'Development' }
      ]
      expect(store.filteredNamespaces).toHaveLength(2)
    })

    it('filteredNamespaces filters by name', () => {
      const store = useNamespacesStore()
      store.namespaces = [
        { id: '1', name: 'production', description: 'Prod env' },
        { id: '2', name: 'development', description: 'Dev env' }
      ]
      store.filters.search = 'prod'
      expect(store.filteredNamespaces).toHaveLength(1)
      expect(store.filteredNamespaces[0].name).toBe('production')
    })

    it('filteredNamespaces filters by description', () => {
      const store = useNamespacesStore()
      store.namespaces = [
        { id: '1', name: 'ns1', description: 'Production environment' },
        { id: '2', name: 'ns2', description: 'Dev environment' }
      ]
      store.filters.search = 'production'
      expect(store.filteredNamespaces).toHaveLength(1)
    })

    it('namespaceCount returns correct count', () => {
      const store = useNamespacesStore()
      store.namespaces = [{ id: '1' }, { id: '2' }, { id: '3' }]
      expect(store.namespaceCount).toBe(3)
    })
  })

  describe('fetchNamespaces action', () => {
    it('fetches namespaces successfully', async () => {
      const mockNamespaces = [
        { id: '1', name: 'ns1' },
        { id: '2', name: 'ns2' }
      ]
      vi.mocked(namespacesApi.list).mockResolvedValue({ namespaces: mockNamespaces, count: 2 })

      const store = useNamespacesStore()
      await store.fetchNamespaces()

      expect(namespacesApi.list).toHaveBeenCalled()
      expect(store.namespaces).toEqual(mockNamespaces)
      expect(store.pagination.total).toBe(2)
      expect(store.loading).toBe(false)
    })

    it('handles array response', async () => {
      const mockNamespaces = [{ id: '1', name: 'ns1' }]
      vi.mocked(namespacesApi.list).mockResolvedValue(mockNamespaces)

      const store = useNamespacesStore()
      await store.fetchNamespaces()

      expect(store.namespaces).toEqual(mockNamespaces)
    })

    it('sets error on fetch failure', async () => {
      vi.mocked(namespacesApi.list).mockRejectedValue(new Error('Network error'))

      const store = useNamespacesStore()
      await store.fetchNamespaces()

      expect(store.error).toBe('Network error')
      expect(store.loading).toBe(false)
    })
  })

  describe('fetchNamespace action', () => {
    it('fetches single namespace', async () => {
      const mockNamespace = { id: '1', name: 'production' }
      vi.mocked(namespacesApi.get).mockResolvedValue(mockNamespace)

      const store = useNamespacesStore()
      const result = await store.fetchNamespace('1')

      expect(result).toEqual(mockNamespace)
      expect(store.currentNamespace).toEqual(mockNamespace)
    })

    it('throws error on failure', async () => {
      vi.mocked(namespacesApi.get).mockRejectedValue(new Error('Not found'))

      const store = useNamespacesStore()
      await expect(store.fetchNamespace('1')).rejects.toThrow('Not found')
    })
  })

  describe('createNamespace action', () => {
    it('creates namespace and adds to list', async () => {
      const newNamespace = { id: '3', name: 'new-ns' }
      vi.mocked(namespacesApi.create).mockResolvedValue(newNamespace)

      const store = useNamespacesStore()
      store.namespaces = [{ id: '1', name: 'existing' }]

      const result = await store.createNamespace({ name: 'new-ns' })

      expect(result).toEqual(newNamespace)
      expect(store.namespaces).toHaveLength(2)
    })

    it('throws error on create failure', async () => {
      vi.mocked(namespacesApi.create).mockRejectedValue(new Error('Create failed'))

      const store = useNamespacesStore()
      await expect(store.createNamespace({ name: 'test' })).rejects.toThrow('Create failed')
    })
  })

  describe('updateNamespace action', () => {
    it('updates namespace in list', async () => {
      const updated = { id: '1', name: 'updated-name' }
      vi.mocked(namespacesApi.update).mockResolvedValue(updated)

      const store = useNamespacesStore()
      store.namespaces = [{ id: '1', name: 'old-name' }]

      const result = await store.updateNamespace('1', { name: 'updated-name' })

      expect(result).toEqual(updated)
      expect(store.namespaces[0].name).toBe('updated-name')
    })

    it('updates currentNamespace if it matches', async () => {
      const updated = { id: '1', name: 'updated' }
      vi.mocked(namespacesApi.update).mockResolvedValue(updated)

      const store = useNamespacesStore()
      store.namespaces = [{ id: '1', name: 'old' }]
      store.currentNamespace = { id: '1', name: 'old' }

      await store.updateNamespace('1', { name: 'updated' })

      expect(store.currentNamespace.name).toBe('updated')
    })
  })

  describe('deleteNamespace action', () => {
    it('removes namespace from list', async () => {
      vi.mocked(namespacesApi.delete).mockResolvedValue(undefined)

      const store = useNamespacesStore()
      store.namespaces = [
        { id: '1', name: 'ns1' },
        { id: '2', name: 'ns2' }
      ]

      await store.deleteNamespace('1')

      expect(store.namespaces).toHaveLength(1)
      expect(store.namespaces[0].id).toBe('2')
    })

    it('clears currentNamespace if deleted', async () => {
      vi.mocked(namespacesApi.delete).mockResolvedValue(undefined)

      const store = useNamespacesStore()
      store.namespaces = [{ id: '1', name: 'ns1' }]
      store.currentNamespace = { id: '1', name: 'ns1' }

      await store.deleteNamespace('1')

      expect(store.currentNamespace).toBeNull()
    })
  })

  describe('fetchNamespaceServers action', () => {
    it('fetches namespace servers', async () => {
      const mockServers = [{ server_id: '1' }, { server_id: '2' }]
      vi.mocked(namespacesApi.listServers).mockResolvedValue({ servers: mockServers })

      const store = useNamespacesStore()
      const result = await store.fetchNamespaceServers('ns1')

      expect(result).toEqual(mockServers)
      expect(store.currentNamespaceServers).toEqual(mockServers)
    })
  })

  describe('addServerToNamespace action', () => {
    it('adds server and refreshes list', async () => {
      vi.mocked(namespacesApi.addServer).mockResolvedValue(undefined)
      vi.mocked(namespacesApi.listServers).mockResolvedValue({ servers: [{ server_id: '1' }] })

      const store = useNamespacesStore()
      await store.addServerToNamespace('ns1', 's1')

      expect(namespacesApi.addServer).toHaveBeenCalledWith('ns1', 's1')
      expect(namespacesApi.listServers).toHaveBeenCalled()
    })
  })

  describe('removeServerFromNamespace action', () => {
    it('removes server from list', async () => {
      vi.mocked(namespacesApi.removeServer).mockResolvedValue(undefined)

      const store = useNamespacesStore()
      store.currentNamespaceServers = [
        { server_id: '1' },
        { server_id: '2' }
      ]

      await store.removeServerFromNamespace('ns1', '1')

      expect(store.currentNamespaceServers).toHaveLength(1)
    })
  })

  describe('role access management', () => {
    it('fetchNamespaceAccess fetches access entries', async () => {
      const mockAccess = [{ role_id: '1', access_level: 'view' }]
      vi.mocked(namespacesApi.listRoleAccess).mockResolvedValue({ access_entries: mockAccess })

      const store = useNamespacesStore()
      const result = await store.fetchNamespaceAccess('ns1')

      expect(result).toEqual(mockAccess)
      expect(store.currentNamespaceAccess).toEqual(mockAccess)
    })

    it('setRoleAccess sets access and refreshes', async () => {
      vi.mocked(namespacesApi.setRoleAccess).mockResolvedValue(undefined)
      vi.mocked(namespacesApi.listRoleAccess).mockResolvedValue({ access_entries: [] })

      const store = useNamespacesStore()
      await store.setRoleAccess('ns1', 'admin', 'execute')

      expect(namespacesApi.setRoleAccess).toHaveBeenCalledWith('ns1', 'admin', 'execute')
    })

    it('removeRoleAccess removes from list', async () => {
      vi.mocked(namespacesApi.removeRoleAccess).mockResolvedValue(undefined)

      const store = useNamespacesStore()
      store.currentNamespaceAccess = [
        { role_id: '1', access_level: 'view' },
        { role_id: '2', access_level: 'execute' }
      ]

      await store.removeRoleAccess('ns1', '1')

      expect(store.currentNamespaceAccess).toHaveLength(1)
    })
  })

  describe('setFilters action', () => {
    it('updates filters', () => {
      const store = useNamespacesStore()
      store.setFilters({ search: 'test' })

      expect(store.filters.search).toBe('test')
    })
  })

  describe('setPage action', () => {
    it('updates page and fetches', async () => {
      vi.mocked(namespacesApi.list).mockResolvedValue({ namespaces: [], count: 0 })

      const store = useNamespacesStore()
      await store.setPage(2)

      expect(store.pagination.page).toBe(2)
      expect(namespacesApi.list).toHaveBeenCalled()
    })
  })

  describe('clearCurrentNamespace action', () => {
    it('clears all current namespace data', () => {
      const store = useNamespacesStore()
      store.currentNamespace = { id: '1' }
      store.currentNamespaceServers = [{ server_id: '1' }]
      store.currentNamespaceAccess = [{ role_id: '1' }]

      store.clearCurrentNamespace()

      expect(store.currentNamespace).toBeNull()
      expect(store.currentNamespaceServers).toEqual([])
      expect(store.currentNamespaceAccess).toEqual([])
    })
  })
})
