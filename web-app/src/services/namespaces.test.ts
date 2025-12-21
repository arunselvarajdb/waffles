import { describe, it, expect, vi, beforeEach } from 'vitest'
import namespacesService from './namespaces'

// Mock the api module
vi.mock('./api', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  }
}))

import api from './api'

describe('namespaces service', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('list', () => {
    it('calls api.get with correct endpoint', async () => {
      vi.mocked(api.get).mockResolvedValue({ namespaces: [] })
      await namespacesService.list()
      expect(api.get).toHaveBeenCalledWith('/namespaces', { params: {} })
    })

    it('passes params to api.get', async () => {
      vi.mocked(api.get).mockResolvedValue({ namespaces: [] })
      await namespacesService.list({ search: 'test' })
      expect(api.get).toHaveBeenCalledWith('/namespaces', { params: { search: 'test' } })
    })
  })

  describe('get', () => {
    it('calls api.get with namespace id', async () => {
      vi.mocked(api.get).mockResolvedValue({ id: '1' })
      await namespacesService.get('1')
      expect(api.get).toHaveBeenCalledWith('/namespaces/1')
    })
  })

  describe('create', () => {
    it('calls api.post with namespace data', async () => {
      const data = { name: 'test-ns', description: 'Test namespace' }
      vi.mocked(api.post).mockResolvedValue({ id: '1', ...data })
      await namespacesService.create(data)
      expect(api.post).toHaveBeenCalledWith('/namespaces', data)
    })
  })

  describe('update', () => {
    it('calls api.put with namespace id and data', async () => {
      const data = { name: 'updated-ns' }
      vi.mocked(api.put).mockResolvedValue({ id: '1', ...data })
      await namespacesService.update('1', data)
      expect(api.put).toHaveBeenCalledWith('/namespaces/1', data)
    })
  })

  describe('delete', () => {
    it('calls api.delete with namespace id', async () => {
      vi.mocked(api.delete).mockResolvedValue(undefined)
      await namespacesService.delete('1')
      expect(api.delete).toHaveBeenCalledWith('/namespaces/1')
    })
  })

  describe('listServers', () => {
    it('calls api.get with servers endpoint', async () => {
      vi.mocked(api.get).mockResolvedValue({ servers: [] })
      await namespacesService.listServers('ns1')
      expect(api.get).toHaveBeenCalledWith('/namespaces/ns1/servers')
    })
  })

  describe('addServer', () => {
    it('calls api.post with server data', async () => {
      vi.mocked(api.post).mockResolvedValue(undefined)
      await namespacesService.addServer('ns1', 'server1')
      expect(api.post).toHaveBeenCalledWith('/namespaces/ns1/servers', { server_id: 'server1' })
    })
  })

  describe('removeServer', () => {
    it('calls api.delete with server id', async () => {
      vi.mocked(api.delete).mockResolvedValue(undefined)
      await namespacesService.removeServer('ns1', 'server1')
      expect(api.delete).toHaveBeenCalledWith('/namespaces/ns1/servers/server1')
    })
  })

  describe('listRoleAccess', () => {
    it('calls api.get with access endpoint', async () => {
      vi.mocked(api.get).mockResolvedValue({ access_entries: [] })
      await namespacesService.listRoleAccess('ns1')
      expect(api.get).toHaveBeenCalledWith('/namespaces/ns1/access')
    })
  })

  describe('setRoleAccess', () => {
    it('calls api.post with role access data', async () => {
      vi.mocked(api.post).mockResolvedValue(undefined)
      await namespacesService.setRoleAccess('ns1', 'admin', 'execute')
      expect(api.post).toHaveBeenCalledWith('/namespaces/ns1/access', {
        role_name: 'admin',
        access_level: 'execute'
      })
    })
  })

  describe('removeRoleAccess', () => {
    it('calls api.delete with role id', async () => {
      vi.mocked(api.delete).mockResolvedValue(undefined)
      await namespacesService.removeRoleAccess('ns1', 'role1')
      expect(api.delete).toHaveBeenCalledWith('/namespaces/ns1/access/role1')
    })
  })
})
