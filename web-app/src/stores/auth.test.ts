import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from './auth'

// Mock the API module
vi.mock('@/services/api', () => ({
  default: {
    post: vi.fn(),
    get: vi.fn(),
  },
}))

import api from '@/services/api'

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('initial state', () => {
    it('has correct initial state', () => {
      const store = useAuthStore()
      expect(store.user).toBeNull()
      expect(store.roles).toEqual([])
      expect(store.isAuthenticated).toBe(false)
      expect(store.loading).toBe(false)
      expect(store.error).toBeNull()
      expect(store.authEnabled).toBeNull()
    })
  })

  describe('getters', () => {
    it('isAdmin returns true when user has admin role', () => {
      const store = useAuthStore()
      store.roles = ['admin']
      expect(store.isAdmin).toBe(true)
    })

    it('isAdmin returns false when user does not have admin role', () => {
      const store = useAuthStore()
      store.roles = ['viewer']
      expect(store.isAdmin).toBe(false)
    })

    it('isOperator returns true for operator role', () => {
      const store = useAuthStore()
      store.roles = ['operator']
      expect(store.isOperator).toBe(true)
    })

    it('isOperator returns true for admin role', () => {
      const store = useAuthStore()
      store.roles = ['admin']
      expect(store.isOperator).toBe(true)
    })

    it('isViewer returns true for any valid role', () => {
      const store = useAuthStore()

      store.roles = ['viewer']
      expect(store.isViewer).toBe(true)

      store.roles = ['operator']
      expect(store.isViewer).toBe(true)

      store.roles = ['admin']
      expect(store.isViewer).toBe(true)
    })

    it('role getter returns highest role', () => {
      const store = useAuthStore()

      store.roles = ['admin', 'viewer']
      expect(store.role).toBe('admin')

      store.roles = ['operator', 'viewer']
      expect(store.role).toBe('operator')

      store.roles = ['viewer']
      expect(store.role).toBe('viewer')

      store.roles = []
      expect(store.role).toBe('user')
    })
  })

  describe('login action', () => {
    it('successfully logs in user', async () => {
      const mockUser = {
        id: '123',
        email: 'test@example.com',
        name: 'Test User',
        roles: ['admin'],
      }
      vi.mocked(api.post).mockResolvedValue({ user: mockUser })

      const store = useAuthStore()
      const result = await store.login('test@example.com', 'password123')

      expect(api.post).toHaveBeenCalledWith('/auth/login', {
        email: 'test@example.com',
        password: 'password123',
      })
      expect(store.user).toEqual(mockUser)
      expect(store.roles).toEqual(['admin'])
      expect(store.isAuthenticated).toBe(true)
      expect(store.loading).toBe(false)
      expect(result).toBe('/admin')
    })

    it('sets loading state during login', async () => {
      vi.mocked(api.post).mockImplementation(() => new Promise(() => {}))

      const store = useAuthStore()
      store.login('test@example.com', 'password')

      expect(store.loading).toBe(true)
    })

    it('handles login error', async () => {
      const error = {
        response: { data: { message: 'Invalid credentials' } },
      }
      vi.mocked(api.post).mockRejectedValue(error)

      const store = useAuthStore()

      await expect(store.login('test@example.com', 'wrong')).rejects.toEqual(error)
      expect(store.error).toBe('Invalid credentials')
      expect(store.isAuthenticated).toBe(false)
      expect(store.loading).toBe(false)
    })

    it('uses redirect path from sessionStorage', async () => {
      const mockUser = { id: '123', email: 'test@example.com', roles: ['viewer'] }
      vi.mocked(api.post).mockResolvedValue({ user: mockUser })
      vi.mocked(sessionStorage.getItem).mockReturnValue('/servers')

      const store = useAuthStore()
      const result = await store.login('test@example.com', 'password')

      expect(result).toBe('/servers')
      expect(sessionStorage.removeItem).toHaveBeenCalledWith('redirectAfterLogin')
    })
  })

  describe('logout action', () => {
    it('successfully logs out user', async () => {
      vi.mocked(api.post).mockResolvedValue({})

      const store = useAuthStore()
      store.user = { id: '123', email: 'test@example.com' }
      store.roles = ['admin']
      store.isAuthenticated = true

      await store.logout()

      expect(api.post).toHaveBeenCalledWith('/auth/logout')
      expect(store.user).toBeNull()
      expect(store.roles).toEqual([])
      expect(store.isAuthenticated).toBe(false)
    })

    it('clears state even if logout API fails', async () => {
      vi.mocked(api.post).mockRejectedValue(new Error('Network error'))

      const store = useAuthStore()
      store.user = { id: '123', email: 'test@example.com' }
      store.roles = ['admin']
      store.isAuthenticated = true

      await store.logout()

      expect(store.user).toBeNull()
      expect(store.roles).toEqual([])
      expect(store.isAuthenticated).toBe(false)
    })
  })

  describe('checkAuth action', () => {
    it('returns true and sets user when authenticated', async () => {
      const mockUser = { id: '123', email: 'test@example.com', roles: ['admin'] }
      vi.mocked(api.get).mockResolvedValue(mockUser)

      const store = useAuthStore()
      const result = await store.checkAuth()

      expect(api.get).toHaveBeenCalledWith('/me')
      expect(result).toBe(true)
      expect(store.user).toEqual(mockUser)
      expect(store.roles).toEqual(['admin'])
      expect(store.isAuthenticated).toBe(true)
    })

    it('returns false when not authenticated', async () => {
      vi.mocked(api.get).mockRejectedValue(new Error('Unauthorized'))

      const store = useAuthStore()
      const result = await store.checkAuth()

      expect(result).toBe(false)
      expect(store.user).toBeNull()
      expect(store.isAuthenticated).toBe(false)
    })
  })

  describe('checkAuthConfig action', () => {
    it('sets authEnabled from status endpoint', async () => {
      vi.mocked(api.get).mockResolvedValue({ auth: { enabled: true } })

      const store = useAuthStore()
      const result = await store.checkAuthConfig()

      expect(api.get).toHaveBeenCalledWith('/status')
      expect(result).toBe(true)
      expect(store.authEnabled).toBe(true)
    })

    it('defaults to true when status check fails', async () => {
      vi.mocked(api.get).mockRejectedValue(new Error('Network error'))

      const store = useAuthStore()
      const result = await store.checkAuthConfig()

      expect(result).toBe(true)
      expect(store.authEnabled).toBe(true)
    })
  })

  describe('clearError action', () => {
    it('clears error state', () => {
      const store = useAuthStore()
      store.error = 'Some error'

      store.clearError()

      expect(store.error).toBeNull()
    })
  })
})
