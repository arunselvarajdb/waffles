import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import AdminDashboard from './AdminDashboard.vue'

// Mock child components
vi.mock('@/components/layout/NavBar.vue', () => ({
  default: { template: '<div class="navbar-mock">NavBar</div>' }
}))
vi.mock('@/components/common/BaseButton.vue', () => ({
  default: { template: '<button class="button-mock"><slot /></button>', props: ['variant'] }
}))
vi.mock('@/components/common/BasePagination.vue', () => ({
  default: { template: '<div class="pagination-mock">Pagination</div>', props: ['currentPage', 'total', 'perPage'], emits: ['page-change'] }
}))
vi.mock('@/components/servers/ServerStats.vue', () => ({
  default: { template: '<div class="stats-mock">ServerStats</div>' }
}))
vi.mock('@/components/servers/ServerFilters.vue', () => ({
  default: { template: '<div class="filters-mock">ServerFilters</div>' }
}))
vi.mock('@/components/servers/ServerTable.vue', () => ({
  default: {
    template: '<div class="table-mock">ServerTable</div>',
    props: ['servers'],
    emits: ['view-server', 'edit-server', 'delete-server', 'toggle-server']
  }
}))
vi.mock('@/components/servers/AddServerModal.vue', () => ({
  default: { template: '<div class="add-modal-mock" v-if="modelValue">AddServerModal</div>', props: ['modelValue'], emits: ['update:modelValue', 'success'] }
}))
vi.mock('@/components/servers/EditServerModal.vue', () => ({
  default: { template: '<div class="edit-modal-mock" v-if="modelValue">EditServerModal</div>', props: ['modelValue', 'server'], emits: ['update:modelValue', 'success'] }
}))
vi.mock('@/components/servers/DeleteServerModal.vue', () => ({
  default: { template: '<div class="delete-modal-mock" v-if="modelValue">DeleteServerModal</div>', props: ['modelValue', 'server'], emits: ['update:modelValue', 'success'] }
}))
vi.mock('@/components/servers/ServerDetailsModal.vue', () => ({
  default: { template: '<div class="details-modal-mock" v-if="modelValue">ServerDetailsModal</div>', props: ['modelValue', 'server'], emits: ['update:modelValue', 'edit-server'] }
}))

// Mock the servers store
vi.mock('@/stores/servers', () => ({
  useServersStore: vi.fn(() => ({
    servers: [],
    filteredServers: [],
    pagination: { page: 1, limit: 10, total: 0 },
    loading: false,
    error: null,
    fetchServers: vi.fn(),
    toggleServer: vi.fn(),
    setPage: vi.fn()
  }))
}))

import { useServersStore } from '@/stores/servers'

describe('AdminDashboard', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders the dashboard', () => {
    const wrapper = mount(AdminDashboard)
    expect(wrapper.find('.min-h-screen').exists()).toBe(true)
  })

  it('displays page header', () => {
    const wrapper = mount(AdminDashboard)
    expect(wrapper.text()).toContain('Server Management')
    expect(wrapper.text()).toContain('Manage MCP servers and their configurations')
  })

  it('renders NavBar component', () => {
    const wrapper = mount(AdminDashboard)
    expect(wrapper.find('.navbar-mock').exists()).toBe(true)
  })

  it('renders Add Server button', () => {
    const wrapper = mount(AdminDashboard)
    expect(wrapper.text()).toContain('+ Add Server')
  })

  it('renders ServerStats component', () => {
    const wrapper = mount(AdminDashboard)
    expect(wrapper.find('.stats-mock').exists()).toBe(true)
  })

  it('renders ServerFilters component', () => {
    const wrapper = mount(AdminDashboard)
    expect(wrapper.find('.filters-mock').exists()).toBe(true)
  })

  it('renders ServerTable component', () => {
    const wrapper = mount(AdminDashboard)
    expect(wrapper.find('.table-mock').exists()).toBe(true)
  })

  it('fetches servers on mount', async () => {
    const mockFetchServers = vi.fn()
    vi.mocked(useServersStore).mockReturnValue({
      servers: [],
      filteredServers: [],
      pagination: { page: 1, limit: 10, total: 0 },
      loading: false,
      error: null,
      fetchServers: mockFetchServers,
      toggleServer: vi.fn(),
      setPage: vi.fn()
    })

    mount(AdminDashboard)
    await flushPromises()

    expect(mockFetchServers).toHaveBeenCalled()
  })

  it('opens add modal when Add Server button is clicked', async () => {
    const wrapper = mount(AdminDashboard)

    // Initially modal should not be visible
    expect(wrapper.find('.add-modal-mock').exists()).toBe(false)

    // Click add button
    await wrapper.find('.button-mock').trigger('click')

    // Modal should now be visible
    expect(wrapper.find('.add-modal-mock').exists()).toBe(true)
  })

  it('shows pagination when there are servers', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [{ id: '1', name: 'Test' }],
      filteredServers: [{ id: '1', name: 'Test' }],
      pagination: { page: 1, limit: 10, total: 1 },
      loading: false,
      error: null,
      fetchServers: vi.fn(),
      toggleServer: vi.fn(),
      setPage: vi.fn()
    })

    const wrapper = mount(AdminDashboard)
    expect(wrapper.find('.pagination-mock').exists()).toBe(true)
  })

  it('hides pagination when no servers', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [],
      filteredServers: [],
      pagination: { page: 1, limit: 10, total: 0 },
      loading: false,
      error: null,
      fetchServers: vi.fn(),
      toggleServer: vi.fn(),
      setPage: vi.fn()
    })

    const wrapper = mount(AdminDashboard)
    expect(wrapper.find('.pagination-mock').exists()).toBe(false)
  })

  it('has responsive layout classes', () => {
    const wrapper = mount(AdminDashboard)
    expect(wrapper.find('.max-w-7xl').exists()).toBe(true)
    expect(wrapper.find('.px-4').exists()).toBe(true)
  })
})
