import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ViewerDashboard from './ViewerDashboard.vue'

// Mock child components
vi.mock('@/components/layout/NavBar.vue', () => ({
  default: { template: '<div class="navbar-mock">NavBar</div>' }
}))
vi.mock('@/components/servers/ServerStats.vue', () => ({
  default: { template: '<div class="stats-mock">ServerStats</div>' }
}))
vi.mock('@/components/servers/ViewerServerDetailsModal.vue', () => ({
  default: {
    template: '<div class="details-modal-mock" v-if="modelValue">ViewerServerDetailsModal</div>',
    props: ['modelValue', 'server'],
    emits: ['update:modelValue']
  }
}))

// Mock the servers store
vi.mock('@/stores/servers', () => ({
  useServersStore: vi.fn(() => ({
    servers: [],
    loading: false,
    error: null,
    fetchServers: vi.fn()
  }))
}))

import { useServersStore } from '@/stores/servers'

describe('ViewerDashboard', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders the dashboard', () => {
    const wrapper = mount(ViewerDashboard)
    expect(wrapper.find('.min-h-screen').exists()).toBe(true)
  })

  it('displays page header', () => {
    const wrapper = mount(ViewerDashboard)
    expect(wrapper.text()).toContain('Viewer Dashboard')
    expect(wrapper.text()).toContain('View server configurations and execute tools')
  })

  it('renders NavBar component', () => {
    const wrapper = mount(ViewerDashboard)
    expect(wrapper.find('.navbar-mock').exists()).toBe(true)
  })

  it('renders ServerStats component', () => {
    const wrapper = mount(ViewerDashboard)
    expect(wrapper.find('.stats-mock').exists()).toBe(true)
  })

  it('shows loading state', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [],
      loading: true,
      error: null,
      fetchServers: vi.fn()
    })

    const wrapper = mount(ViewerDashboard)
    expect(wrapper.text()).toContain('Loading servers...')
    expect(wrapper.find('.animate-spin').exists()).toBe(true)
  })

  it('shows empty state when no servers', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [],
      loading: false,
      error: null,
      fetchServers: vi.fn()
    })

    const wrapper = mount(ViewerDashboard)
    expect(wrapper.text()).toContain('No Servers Available')
    expect(wrapper.text()).toContain("You don't have access to any servers yet")
  })

  it('renders server list when servers exist', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [
        { id: '1', name: 'Test Server', url: 'http://test.com', is_active: true, transport: 'http' }
      ],
      loading: false,
      error: null,
      fetchServers: vi.fn()
    })

    const wrapper = mount(ViewerDashboard)
    expect(wrapper.text()).toContain('Test Server')
    expect(wrapper.text()).toContain('http://test.com')
  })

  it('shows active badge for active servers', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [
        { id: '1', name: 'Active Server', url: 'http://test.com', is_active: true }
      ],
      loading: false,
      error: null,
      fetchServers: vi.fn()
    })

    const wrapper = mount(ViewerDashboard)
    expect(wrapper.text()).toContain('Active')
    expect(wrapper.find('.bg-green-500').exists()).toBe(true)
  })

  it('shows inactive badge for inactive servers', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [
        { id: '1', name: 'Inactive Server', url: 'http://test.com', is_active: false }
      ],
      loading: false,
      error: null,
      fetchServers: vi.fn()
    })

    const wrapper = mount(ViewerDashboard)
    expect(wrapper.text()).toContain('Inactive')
    expect(wrapper.find('.bg-gray-400').exists()).toBe(true)
  })

  it('fetches servers on mount', async () => {
    const mockFetchServers = vi.fn()
    vi.mocked(useServersStore).mockReturnValue({
      servers: [],
      loading: false,
      error: null,
      fetchServers: mockFetchServers
    })

    mount(ViewerDashboard)
    await flushPromises()

    expect(mockFetchServers).toHaveBeenCalled()
  })

  it('opens details modal on View click', async () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [
        { id: '1', name: 'Test Server', url: 'http://test.com', is_active: true }
      ],
      loading: false,
      error: null,
      fetchServers: vi.fn()
    })

    const wrapper = mount(ViewerDashboard)

    // Initially modal should not be visible
    expect(wrapper.find('.details-modal-mock').exists()).toBe(false)

    // Click view button
    await wrapper.find('button').trigger('click')

    // Modal should now be visible
    expect(wrapper.find('.details-modal-mock').exists()).toBe(true)
  })

  it('displays server description when available', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [
        { id: '1', name: 'Test Server', url: 'http://test.com', is_active: true, description: 'A test server' }
      ],
      loading: false,
      error: null,
      fetchServers: vi.fn()
    })

    const wrapper = mount(ViewerDashboard)
    expect(wrapper.text()).toContain('A test server')
  })

  it('displays transport type', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [
        { id: '1', name: 'Test Server', url: 'http://test.com', is_active: true, transport: 'sse' }
      ],
      loading: false,
      error: null,
      fetchServers: vi.fn()
    })

    const wrapper = mount(ViewerDashboard)
    expect(wrapper.text()).toContain('sse')
  })

  it('has Available Servers section header', () => {
    const wrapper = mount(ViewerDashboard)
    expect(wrapper.text()).toContain('Available Servers')
    expect(wrapper.text()).toContain('Servers you have access to')
  })

  it('has hover effect on server rows', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [
        { id: '1', name: 'Test Server', url: 'http://test.com', is_active: true }
      ],
      loading: false,
      error: null,
      fetchServers: vi.fn()
    })

    const wrapper = mount(ViewerDashboard)
    expect(wrapper.find('.hover\\:bg-gray-50').exists()).toBe(true)
  })

  it('has responsive layout classes', () => {
    const wrapper = mount(ViewerDashboard)
    expect(wrapper.find('.max-w-7xl').exists()).toBe(true)
  })
})
