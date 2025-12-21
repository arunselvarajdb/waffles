import { describe, it, expect, vi, beforeEach } from 'vitest'
import rolesService from './roles'

// Mock the api module
vi.mock('./api', () => ({
  default: {
    get: vi.fn()
  }
}))

import api from './api'

describe('roles service', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('list', () => {
    it('returns roles from API when available', async () => {
      const mockRoles = [
        { id: '1', name: 'admin' },
        { id: '2', name: 'viewer' }
      ]
      vi.mocked(api.get).mockResolvedValue({ roles: mockRoles })

      const result = await rolesService.list()

      expect(api.get).toHaveBeenCalledWith('/roles')
      expect(result).toEqual(mockRoles)
    })

    it('returns roles array directly when response is array', async () => {
      const mockRoles = [
        { id: '1', name: 'admin' },
        { id: '2', name: 'viewer' }
      ]
      vi.mocked(api.get).mockResolvedValue(mockRoles)

      const result = await rolesService.list()

      expect(result).toEqual(mockRoles)
    })

    it('returns default roles on 404 error', async () => {
      const error = { response: { status: 404 } }
      vi.mocked(api.get).mockRejectedValue(error)

      const result = await rolesService.list()

      expect(result).toEqual([
        { id: 'admin', name: 'admin', description: 'Full administrative access' },
        { id: 'operator', name: 'operator', description: 'Server management access' },
        { id: 'viewer', name: 'viewer', description: 'Read-only access' }
      ])
    })

    it('throws error on non-404 errors', async () => {
      const error = { response: { status: 500 }, message: 'Server error' }
      vi.mocked(api.get).mockRejectedValue(error)

      await expect(rolesService.list()).rejects.toEqual(error)
    })
  })
})
