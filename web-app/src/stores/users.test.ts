import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useUsersStore } from './users'
import usersApi from '@/services/users'

vi.mock('@/services/users')

describe('useUsersStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('initial state', () => {
    it('should have correct initial state', () => {
      const store = useUsersStore()

      expect(store.users).toEqual([])
      expect(store.currentUser).toBeNull()
      expect(store.loading).toBe(false)
      expect(store.error).toBeNull()
      expect(store.filters).toEqual({
        search: '',
        status: 'all',
        role: 'all'
      })
      expect(store.pagination).toEqual({
        page: 1,
        pageSize: 10,
        total: 0,
        totalPages: 0
      })
    })
  })

  describe('getters', () => {
    it('filteredUsers should filter by search', () => {
      const store = useUsersStore()
      store.users = [
        { id: '1', name: 'John Doe', email: 'john@example.com', is_active: true, roles: ['user'] },
        { id: '2', name: 'Jane Smith', email: 'jane@example.com', is_active: true, roles: ['admin'] }
      ]
      store.filters.search = 'john'

      expect(store.filteredUsers).toHaveLength(1)
      expect(store.filteredUsers[0].name).toBe('John Doe')
    })

    it('filteredUsers should filter by status', () => {
      const store = useUsersStore()
      store.users = [
        { id: '1', name: 'Active User', is_active: true, roles: [] },
        { id: '2', name: 'Inactive User', is_active: false, roles: [] }
      ]
      store.filters.status = 'active'

      expect(store.filteredUsers).toHaveLength(1)
      expect(store.filteredUsers[0].name).toBe('Active User')
    })

    it('filteredUsers should filter by role', () => {
      const store = useUsersStore()
      store.users = [
        { id: '1', name: 'Admin', is_active: true, roles: ['admin'] },
        { id: '2', name: 'User', is_active: true, roles: ['user'] }
      ]
      store.filters.role = 'admin'

      expect(store.filteredUsers).toHaveLength(1)
      expect(store.filteredUsers[0].name).toBe('Admin')
    })

    it('activeUsers should return only active users', () => {
      const store = useUsersStore()
      store.users = [
        { id: '1', is_active: true },
        { id: '2', is_active: false },
        { id: '3', is_active: true }
      ]

      expect(store.activeUsers).toHaveLength(2)
    })

    it('adminUsers should return users with admin role', () => {
      const store = useUsersStore()
      store.users = [
        { id: '1', roles: ['admin'] },
        { id: '2', roles: ['user'] },
        { id: '3', roles: ['admin', 'user'] }
      ]

      expect(store.adminUsers).toHaveLength(2)
    })

    it('userStats should return correct statistics', () => {
      const store = useUsersStore()
      store.users = [
        { id: '1', is_active: true, roles: ['admin'] },
        { id: '2', is_active: true, roles: ['user'] },
        { id: '3', is_active: false, roles: ['user'] }
      ]
      store.pagination.total = 3

      const stats = store.userStats
      expect(stats.total).toBe(3)
      expect(stats.active).toBe(2)
      expect(stats.inactive).toBe(1)
      expect(stats.admins).toBe(1)
    })
  })

  describe('actions', () => {
    describe('fetchUsers', () => {
      it('should fetch users and update state', async () => {
        const store = useUsersStore()
        const mockData = {
          users: [{ id: '1', name: 'Test User' }],
          total: 1,
          total_pages: 1,
          page: 1,
          page_size: 10
        }
        vi.mocked(usersApi.list).mockResolvedValue(mockData)

        await store.fetchUsers()

        expect(store.users).toEqual(mockData.users)
        expect(store.pagination.total).toBe(1)
        expect(store.loading).toBe(false)
        expect(store.error).toBeNull()
      })

      it('should handle fetch error', async () => {
        const store = useUsersStore()
        const error = new Error('Network error')
        vi.mocked(usersApi.list).mockRejectedValue(error)

        await store.fetchUsers()

        expect(store.error).toBe('Network error')
        expect(store.loading).toBe(false)
      })
    })

    describe('createUser', () => {
      it('should create user and add to list', async () => {
        const store = useUsersStore()
        const newUser = { email: 'new@example.com', name: 'New User' }
        const mockResponse = { user: { id: '1', ...newUser } }
        vi.mocked(usersApi.create).mockResolvedValue(mockResponse)

        const result = await store.createUser(newUser)

        expect(usersApi.create).toHaveBeenCalledWith(newUser)
        expect(store.users[0]).toEqual(mockResponse.user)
        expect(result).toEqual(mockResponse)
      })
    })

    describe('updateUser', () => {
      it('should update user in list', async () => {
        const store = useUsersStore()
        store.users = [{ id: '1', name: 'Old Name' }]
        const mockResponse = { id: '1', name: 'New Name' }
        vi.mocked(usersApi.update).mockResolvedValue(mockResponse)

        await store.updateUser('1', { name: 'New Name' })

        expect(store.users[0].name).toBe('New Name')
      })
    })

    describe('deactivateUser', () => {
      it('should set user is_active to false', async () => {
        const store = useUsersStore()
        store.users = [{ id: '1', is_active: true }]
        vi.mocked(usersApi.deactivate).mockResolvedValue(undefined)

        await store.deactivateUser('1')

        expect(store.users[0].is_active).toBe(false)
      })
    })

    describe('updateUserRoles', () => {
      it('should update user roles', async () => {
        const store = useUsersStore()
        store.users = [{ id: '1', roles: ['user'] }]
        const mockResponse = { id: '1', roles: ['admin', 'user'] }
        vi.mocked(usersApi.updateRoles).mockResolvedValue(mockResponse)

        await store.updateUserRoles('1', ['admin', 'user'])

        expect(store.users[0].roles).toEqual(['admin', 'user'])
      })
    })

    describe('resetUserPassword', () => {
      it('should return temp password', async () => {
        const store = useUsersStore()
        const mockResponse = { temp_password: 'abc123' }
        vi.mocked(usersApi.resetPassword).mockResolvedValue(mockResponse)

        const result = await store.resetUserPassword('1')

        expect(result.temp_password).toBe('abc123')
      })
    })

    describe('setFilters', () => {
      it('should update filters', () => {
        const store = useUsersStore()

        store.setFilters({ search: 'test', status: 'active' })

        expect(store.filters.search).toBe('test')
        expect(store.filters.status).toBe('active')
        expect(store.filters.role).toBe('all') // unchanged
      })
    })

    describe('setPage', () => {
      it('should update page and fetch users', async () => {
        const store = useUsersStore()
        vi.mocked(usersApi.list).mockResolvedValue({ users: [], total: 0 })

        store.setPage(2)

        expect(store.pagination.page).toBe(2)
        expect(usersApi.list).toHaveBeenCalled()
      })
    })

    describe('clearError', () => {
      it('should clear error', () => {
        const store = useUsersStore()
        store.error = 'Some error'

        store.clearError()

        expect(store.error).toBeNull()
      })
    })
  })
})
