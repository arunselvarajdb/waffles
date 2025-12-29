import { describe, it, expect, vi, beforeEach } from 'vitest'
import rolesApi from './roles'
import api from './api'

vi.mock('./api')

describe('rolesApi', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('list', () => {
    it('should call api.get with correct endpoint', async () => {
      const mockResponse = { roles: [{ id: '1', name: 'admin' }] }
      vi.mocked(api.get).mockResolvedValue(mockResponse)

      const result = await rolesApi.list()

      expect(api.get).toHaveBeenCalledWith('/admin/roles')
      expect(result).toEqual(mockResponse)
    })
  })

  describe('getById', () => {
    it('should call api.get with correct path', async () => {
      const mockRole = { id: 'role-1', name: 'admin', permissions: [] }
      vi.mocked(api.get).mockResolvedValue(mockRole)

      const result = await rolesApi.getById('role-1')

      expect(api.get).toHaveBeenCalledWith('/admin/roles/role-1')
      expect(result).toEqual(mockRole)
    })
  })

  describe('create', () => {
    it('should call api.post with role data', async () => {
      const roleData = { name: 'custom-role', description: 'A custom role', permissions: ['read'] }
      const mockResponse = { id: 'new-role', ...roleData }
      vi.mocked(api.post).mockResolvedValue(mockResponse)

      const result = await rolesApi.create(roleData)

      expect(api.post).toHaveBeenCalledWith('/admin/roles', roleData)
      expect(result).toEqual(mockResponse)
    })
  })

  describe('update', () => {
    it('should call api.put with correct path and data', async () => {
      const roleData = { description: 'Updated description' }
      const mockResponse = { id: 'role-1', name: 'admin', ...roleData }
      vi.mocked(api.put).mockResolvedValue(mockResponse)

      const result = await rolesApi.update('role-1', roleData)

      expect(api.put).toHaveBeenCalledWith('/admin/roles/role-1', roleData)
      expect(result).toEqual(mockResponse)
    })
  })

  describe('delete', () => {
    it('should call api.delete with correct path', async () => {
      vi.mocked(api.delete).mockResolvedValue(undefined)

      await rolesApi.delete('role-1')

      expect(api.delete).toHaveBeenCalledWith('/admin/roles/role-1')
    })
  })

  describe('listPermissions', () => {
    it('should call api.get for permissions endpoint', async () => {
      const mockResponse = { permissions: [{ id: 'read', name: 'Read' }] }
      vi.mocked(api.get).mockResolvedValue(mockResponse)

      const result = await rolesApi.listPermissions()

      expect(api.get).toHaveBeenCalledWith('/admin/permissions')
      expect(result).toEqual(mockResponse)
    })
  })
})
