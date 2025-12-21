import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useServersStore } from './servers'

// Mock the servers API
vi.mock('@/services/servers', () => ({
  default: {
    list: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    toggle: vi.fn(),
    checkHealth: vi.fn(),
  },
}))

import serversApi from '@/services/servers'

describe('useServersStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('initial state', () => {
    it('has correct initial state', () => {
      const store = useServersStore()
      expect(store.servers).toEqual([])
      expect(store.currentServer).toBeNull()
      expect(store.loading).toBe(false)
      expect(store.error).toBeNull()
      expect(store.filters.search).toBe('')
      expect(store.filters.status).toBe('all')
      expect(store.filters.health).toBe('all')
      expect(store.pagination.page).toBe(1)
      expect(store.pagination.limit).toBe(10)
    })
  })

  describe('getters', () => {
    it('filteredServers returns all servers when no filters', () => {
      const store = useServersStore()
      store.servers = [
        { id: '1', name: 'Server 1', is_active: true },
        { id: '2', name: 'Server 2', is_active: false },
      ]
      expect(store.filteredServers).toHaveLength(2)
    })

    it('filteredServers filters by search', () => {
      const store = useServersStore()
      store.servers = [
        { id: '1', name: 'API Server', is_active: true },
        { id: '2', name: 'Database', is_active: true },
      ]
      store.filters.search = 'api'
      expect(store.filteredServers).toHaveLength(1)
      expect(store.filteredServers[0].name).toBe('API Server')
    })

    it('filteredServers filters by description', () => {
      const store = useServersStore()
      store.servers = [
        { id: '1', name: 'Server 1', description: 'Production API', is_active: true },
        { id: '2', name: 'Server 2', description: 'Dev', is_active: true },
      ]
      store.filters.search = 'production'
      expect(store.filteredServers).toHaveLength(1)
    })

    it('filteredServers filters by active status', () => {
      const store = useServersStore()
      store.servers = [
        { id: '1', name: 'Server 1', is_active: true },
        { id: '2', name: 'Server 2', is_active: false },
      ]
      store.filters.status = 'active'
      expect(store.filteredServers).toHaveLength(1)
      expect(store.filteredServers[0].is_active).toBe(true)
    })

    it('filteredServers filters by inactive status', () => {
      const store = useServersStore()
      store.servers = [
        { id: '1', name: 'Server 1', is_active: true },
        { id: '2', name: 'Server 2', is_active: false },
      ]
      store.filters.status = 'inactive'
      expect(store.filteredServers).toHaveLength(1)
      expect(store.filteredServers[0].is_active).toBe(false)
    })

    it('filteredServers filters by health status', () => {
      const store = useServersStore()
      store.servers = [
        { id: '1', name: 'Server 1', health: { status: 'healthy' } },
        { id: '2', name: 'Server 2', health: { status: 'unhealthy' } },
      ]
      store.filters.health = 'healthy'
      expect(store.filteredServers).toHaveLength(1)
    })

    it('activeServers returns only active servers', () => {
      const store = useServersStore()
      store.servers = [
        { id: '1', name: 'Server 1', is_active: true },
        { id: '2', name: 'Server 2', is_active: false },
        { id: '3', name: 'Server 3', is_active: true },
      ]
      expect(store.activeServers).toHaveLength(2)
    })

    it('healthyServers returns only healthy servers', () => {
      const store = useServersStore()
      store.servers = [
        { id: '1', name: 'Server 1', health: { status: 'healthy' } },
        { id: '2', name: 'Server 2', health: { status: 'unhealthy' } },
      ]
      expect(store.healthyServers).toHaveLength(1)
    })
  })

  describe('fetchServers action', () => {
    it('fetches servers successfully', async () => {
      const mockServers = [
        { id: '1', name: 'Server 1' },
        { id: '2', name: 'Server 2' },
      ]
      vi.mocked(serversApi.list).mockResolvedValue({ servers: mockServers, total: 2 })

      const store = useServersStore()
      await store.fetchServers()

      expect(serversApi.list).toHaveBeenCalledWith({ page: 1, limit: 10 })
      expect(store.servers).toEqual(mockServers)
      expect(store.pagination.total).toBe(2)
      expect(store.loading).toBe(false)
    })

    it('handles array response without wrapper', async () => {
      const mockServers = [{ id: '1', name: 'Server 1' }]
      vi.mocked(serversApi.list).mockResolvedValue(mockServers)

      const store = useServersStore()
      await store.fetchServers()

      expect(store.servers).toEqual(mockServers)
    })

    it('sets error on fetch failure', async () => {
      vi.mocked(serversApi.list).mockRejectedValue(new Error('Network error'))

      const store = useServersStore()
      await store.fetchServers()

      expect(store.error).toBe('Network error')
      expect(store.loading).toBe(false)
    })
  })

  describe('createServer action', () => {
    it('creates server and adds to list', async () => {
      const newServer = { id: '3', name: 'New Server' }
      vi.mocked(serversApi.create).mockResolvedValue(newServer)

      const store = useServersStore()
      store.servers = [{ id: '1', name: 'Existing' }]

      const result = await store.createServer({ name: 'New Server' })

      expect(serversApi.create).toHaveBeenCalledWith({ name: 'New Server' })
      expect(result).toEqual(newServer)
      expect(store.servers).toHaveLength(2)
    })

    it('throws error on create failure', async () => {
      vi.mocked(serversApi.create).mockRejectedValue(new Error('Create failed'))

      const store = useServersStore()

      await expect(store.createServer({ name: 'Test' })).rejects.toThrow('Create failed')
      expect(store.error).toBe('Create failed')
    })
  })

  describe('updateServer action', () => {
    it('updates server in list', async () => {
      const updated = { id: '1', name: 'Updated Name' }
      vi.mocked(serversApi.update).mockResolvedValue(updated)

      const store = useServersStore()
      store.servers = [{ id: '1', name: 'Old Name' }]

      const result = await store.updateServer('1', { name: 'Updated Name' })

      expect(result).toEqual(updated)
      expect(store.servers[0].name).toBe('Updated Name')
    })

    it('throws error on update failure', async () => {
      vi.mocked(serversApi.update).mockRejectedValue(new Error('Update failed'))

      const store = useServersStore()

      await expect(store.updateServer('1', {})).rejects.toThrow('Update failed')
    })
  })

  describe('deleteServer action', () => {
    it('removes server from list', async () => {
      vi.mocked(serversApi.delete).mockResolvedValue(undefined)

      const store = useServersStore()
      store.servers = [
        { id: '1', name: 'Server 1' },
        { id: '2', name: 'Server 2' },
      ]

      await store.deleteServer('1')

      expect(store.servers).toHaveLength(1)
      expect(store.servers[0].id).toBe('2')
    })

    it('throws error on delete failure', async () => {
      vi.mocked(serversApi.delete).mockRejectedValue(new Error('Delete failed'))

      const store = useServersStore()

      await expect(store.deleteServer('1')).rejects.toThrow('Delete failed')
    })
  })

  describe('toggleServer action', () => {
    it('updates server active status', async () => {
      const updated = { id: '1', name: 'Server', is_active: false }
      vi.mocked(serversApi.toggle).mockResolvedValue(updated)

      const store = useServersStore()
      store.servers = [{ id: '1', name: 'Server', is_active: true }]

      await store.toggleServer('1')

      expect(store.servers[0].is_active).toBe(false)
    })
  })

  describe('checkHealth action', () => {
    it('updates server health', async () => {
      const health = { status: 'healthy', latency: 50 }
      vi.mocked(serversApi.checkHealth).mockResolvedValue(health)

      const store = useServersStore()
      store.servers = [{ id: '1', name: 'Server' }]

      await store.checkHealth('1')

      expect(store.servers[0].health).toEqual(health)
    })
  })

  describe('setFilters action', () => {
    it('updates filters', () => {
      const store = useServersStore()

      store.setFilters({ search: 'test', status: 'active' })

      expect(store.filters.search).toBe('test')
      expect(store.filters.status).toBe('active')
      expect(store.filters.health).toBe('all') // unchanged
    })
  })

  describe('setPage action', () => {
    it('updates page and fetches servers', async () => {
      vi.mocked(serversApi.list).mockResolvedValue({ servers: [], total: 0 })

      const store = useServersStore()
      await store.setPage(2)

      expect(store.pagination.page).toBe(2)
      expect(serversApi.list).toHaveBeenCalled()
    })
  })
})
