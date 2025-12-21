import { describe, it, expect, vi, beforeEach } from 'vitest'
import serversService from './servers'

// Mock the api module
vi.mock('./api', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
    patch: vi.fn()
  }
}))

import api from './api'

describe('servers service', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('list', () => {
    it('calls api.get with correct endpoint', async () => {
      vi.mocked(api.get).mockResolvedValue({ servers: [] })
      await serversService.list()
      expect(api.get).toHaveBeenCalledWith('/servers', { params: {} })
    })

    it('passes params to api.get', async () => {
      vi.mocked(api.get).mockResolvedValue({ servers: [] })
      await serversService.list({ status: 'active' })
      expect(api.get).toHaveBeenCalledWith('/servers', { params: { status: 'active' } })
    })
  })

  describe('get', () => {
    it('calls api.get with server id', async () => {
      vi.mocked(api.get).mockResolvedValue({ id: '1' })
      await serversService.get('1')
      expect(api.get).toHaveBeenCalledWith('/servers/1')
    })
  })

  describe('create', () => {
    it('calls api.post with server data', async () => {
      const serverData = { name: 'Test Server', url: 'http://example.com' }
      vi.mocked(api.post).mockResolvedValue({ id: '1', ...serverData })
      await serversService.create(serverData)
      expect(api.post).toHaveBeenCalledWith('/servers', serverData)
    })
  })

  describe('update', () => {
    it('calls api.put with server id and data', async () => {
      const updateData = { name: 'Updated Server' }
      vi.mocked(api.put).mockResolvedValue({ id: '1', ...updateData })
      await serversService.update('1', updateData)
      expect(api.put).toHaveBeenCalledWith('/servers/1', updateData)
    })
  })

  describe('delete', () => {
    it('calls api.delete with server id', async () => {
      vi.mocked(api.delete).mockResolvedValue(undefined)
      await serversService.delete('1')
      expect(api.delete).toHaveBeenCalledWith('/servers/1')
    })
  })

  describe('toggle', () => {
    it('calls api.patch with toggle endpoint', async () => {
      vi.mocked(api.patch).mockResolvedValue({ is_active: true })
      await serversService.toggle('1')
      expect(api.patch).toHaveBeenCalledWith('/servers/1/toggle')
    })
  })

  describe('getHealth', () => {
    it('calls api.get with health endpoint', async () => {
      vi.mocked(api.get).mockResolvedValue({ status: 'healthy' })
      await serversService.getHealth('1')
      expect(api.get).toHaveBeenCalledWith('/servers/1/health')
    })
  })

  describe('checkHealth', () => {
    it('calls api.post with health endpoint', async () => {
      vi.mocked(api.post).mockResolvedValue({ status: 'healthy' })
      await serversService.checkHealth('1')
      expect(api.post).toHaveBeenCalledWith('/servers/1/health')
    })
  })
})
