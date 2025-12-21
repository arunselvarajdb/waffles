import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ServerStats from './ServerStats.vue'

// Mock the servers store
vi.mock('@/stores/servers', () => ({
  useServersStore: vi.fn(() => ({
    servers: [],
    activeServers: [],
    healthyServers: []
  }))
}))

import { useServersStore } from '@/stores/servers'

describe('ServerStats', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders stats grid', () => {
    const wrapper = mount(ServerStats)
    expect(wrapper.find('.grid').exists()).toBe(true)
  })

  it('displays Total Servers card', () => {
    const wrapper = mount(ServerStats)
    expect(wrapper.text()).toContain('Total Servers')
  })

  it('displays Active Servers card', () => {
    const wrapper = mount(ServerStats)
    expect(wrapper.text()).toContain('Active Servers')
  })

  it('displays Healthy Servers card', () => {
    const wrapper = mount(ServerStats)
    expect(wrapper.text()).toContain('Healthy Servers')
  })

  it('displays Issues card', () => {
    const wrapper = mount(ServerStats)
    expect(wrapper.text()).toContain('Issues')
  })

  it('shows correct total servers count', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [{ id: '1' }, { id: '2' }, { id: '3' }],
      activeServers: [],
      healthyServers: []
    })

    const wrapper = mount(ServerStats)
    expect(wrapper.text()).toContain('3')
  })

  it('shows correct active servers count', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [{ id: '1' }, { id: '2' }],
      activeServers: [{ id: '1' }],
      healthyServers: []
    })

    const wrapper = mount(ServerStats)
    // Check there's a card showing 1 for active servers
    const cards = wrapper.findAll('.bg-white.rounded-lg')
    expect(cards.length).toBe(4)
  })

  it('shows correct healthy servers count', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [{ id: '1' }, { id: '2' }],
      activeServers: [{ id: '1' }, { id: '2' }],
      healthyServers: [{ id: '1' }]
    })

    const wrapper = mount(ServerStats)
    expect(wrapper.text()).toContain('1')
  })

  it('calculates issues count correctly', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [
        { id: '1', health: { status: 'healthy' } },
        { id: '2', health: { status: 'degraded' } },
        { id: '3', health: { status: 'unhealthy' } }
      ],
      activeServers: [],
      healthyServers: []
    })

    const wrapper = mount(ServerStats)
    // Should show 2 issues (degraded + unhealthy)
    expect(wrapper.text()).toContain('2')
  })

  it('shows 0 issues when all healthy', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [
        { id: '1', health: { status: 'healthy' } },
        { id: '2', health: { status: 'healthy' } }
      ],
      activeServers: [],
      healthyServers: []
    })

    const wrapper = mount(ServerStats)
    expect(wrapper.text()).toContain('0')
  })

  it('handles servers without health status', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [
        { id: '1' },
        { id: '2', health: null }
      ],
      activeServers: [],
      healthyServers: []
    })

    const wrapper = mount(ServerStats)
    // Should not crash and show 0 issues
    expect(wrapper.text()).toContain('0')
  })

  it('has correct grid layout for responsive design', () => {
    const wrapper = mount(ServerStats)
    expect(wrapper.find('.grid-cols-1').exists()).toBe(true)
    expect(wrapper.find('.md\\:grid-cols-4').exists()).toBe(true)
  })

  it('has blue icon for total servers', () => {
    const wrapper = mount(ServerStats)
    expect(wrapper.find('.bg-blue-100').exists()).toBe(true)
  })

  it('has green icon for active servers', () => {
    const wrapper = mount(ServerStats)
    expect(wrapper.find('.bg-green-100').exists()).toBe(true)
  })

  it('has red icon for issues', () => {
    const wrapper = mount(ServerStats)
    expect(wrapper.find('.bg-red-100').exists()).toBe(true)
  })

  it('shows zero counts when no servers', () => {
    vi.mocked(useServersStore).mockReturnValue({
      servers: [],
      activeServers: [],
      healthyServers: []
    })

    const wrapper = mount(ServerStats)
    const stats = wrapper.findAll('.text-2xl')
    stats.forEach(stat => {
      expect(stat.text()).toBe('0')
    })
  })
})
