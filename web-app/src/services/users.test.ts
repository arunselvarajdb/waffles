import { describe, it, expect, vi, beforeEach } from 'vitest'
import usersApi from './users'
import api from './api'

vi.mock('./api')

describe('usersApi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('list', () => {
    it('should call api.get with correct params', async () => {
      const mockResponse = { users: [], total: 0 }
      vi.mocked(api.get).mockResolvedValue(mockResponse)

      const params = { page: 1, page_size: 10 }
      const result = await usersApi.list(params)

      expect(api.get).toHaveBeenCalledWith('/admin/users', { params })
      expect(result).toEqual(mockResponse)
    })

    it('should call api.get without params when none provided', async () => {
      const mockResponse = { users: [], total: 0 }
      vi.mocked(api.get).mockResolvedValue(mockResponse)

      await usersApi.list()

      expect(api.get).toHaveBeenCalledWith('/admin/users', { params: {} })
    })
  })

  describe('getById', () => {
    it('should call api.get with correct path', async () => {
      const mockUser = { id: 'user-1', name: 'Test User' }
      vi.mocked(api.get).mockResolvedValue(mockUser)

      const result = await usersApi.getById('user-1')

      expect(api.get).toHaveBeenCalledWith('/admin/users/user-1')
      expect(result).toEqual(mockUser)
    })
  })

  describe('create', () => {
    it('should call api.post with user data', async () => {
      const userData = { email: 'test@example.com', name: 'Test User' }
      const mockResponse = { user: { ...userData, id: 'user-1' } }
      vi.mocked(api.post).mockResolvedValue(mockResponse)

      const result = await usersApi.create(userData)

      expect(api.post).toHaveBeenCalledWith('/admin/users', userData)
      expect(result).toEqual(mockResponse)
    })
  })

  describe('update', () => {
    it('should call api.put with correct path and data', async () => {
      const userData = { name: 'Updated Name' }
      const mockResponse = { id: 'user-1', name: 'Updated Name' }
      vi.mocked(api.put).mockResolvedValue(mockResponse)

      const result = await usersApi.update('user-1', userData)

      expect(api.put).toHaveBeenCalledWith('/admin/users/user-1', userData)
      expect(result).toEqual(mockResponse)
    })
  })

  describe('deactivate', () => {
    it('should call api.delete with correct path', async () => {
      vi.mocked(api.delete).mockResolvedValue(undefined)

      await usersApi.deactivate('user-1')

      expect(api.delete).toHaveBeenCalledWith('/admin/users/user-1')
    })
  })

  describe('updateRoles', () => {
    it('should call api.put with roles array', async () => {
      const mockResponse = { id: 'user-1', roles: ['admin', 'user'] }
      vi.mocked(api.put).mockResolvedValue(mockResponse)

      const result = await usersApi.updateRoles('user-1', ['admin', 'user'])

      expect(api.put).toHaveBeenCalledWith('/admin/users/user-1/roles', { roles: ['admin', 'user'] })
      expect(result).toEqual(mockResponse)
    })
  })

  describe('resetPassword', () => {
    it('should call api.post and return temp password', async () => {
      const mockResponse = { temp_password: 'abc123' }
      vi.mocked(api.post).mockResolvedValue(mockResponse)

      const result = await usersApi.resetPassword('user-1')

      expect(api.post).toHaveBeenCalledWith('/admin/users/user-1/reset-password')
      expect(result).toEqual(mockResponse)
    })
  })

  describe('listSessions', () => {
    it('should call api.get for sessions endpoint', async () => {
      const mockResponse = { sessions: [] }
      vi.mocked(api.get).mockResolvedValue(mockResponse)

      const result = await usersApi.listSessions('user-1')

      expect(api.get).toHaveBeenCalledWith('/admin/users/user-1/sessions')
      expect(result).toEqual(mockResponse)
    })
  })

  describe('revokeSession', () => {
    it('should call api.delete for specific session', async () => {
      vi.mocked(api.delete).mockResolvedValue(undefined)

      await usersApi.revokeSession('user-1', 'session-1')

      expect(api.delete).toHaveBeenCalledWith('/admin/users/user-1/sessions/session-1')
    })
  })

  describe('revokeAllSessions', () => {
    it('should call api.delete for all sessions', async () => {
      vi.mocked(api.delete).mockResolvedValue(undefined)

      await usersApi.revokeAllSessions('user-1')

      expect(api.delete).toHaveBeenCalledWith('/admin/users/user-1/sessions')
    })
  })
})
