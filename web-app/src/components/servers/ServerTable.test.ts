import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import ServerTable from './ServerTable.vue'
import BaseBadge from '@/components/common/BaseBadge.vue'
import BaseToggle from '@/components/common/BaseToggle.vue'

describe('ServerTable', () => {
  const mockServers = [
    {
      id: '1',
      name: 'API Server',
      description: 'Production API',
      url: 'http://api.example.com',
      protocol_version: '1.0',
      auth_type: 'bearer',
      is_active: true,
      health: { status: 'healthy', last_checked: new Date().toISOString() }
    },
    {
      id: '2',
      name: 'Dev Server',
      description: null,
      url: 'http://dev.example.com',
      protocol_version: null,
      auth_type: 'none',
      is_active: false,
      health: { status: 'unhealthy' }
    }
  ]

  const mountTable = (servers = mockServers) => {
    return mount(ServerTable, {
      props: { servers },
      global: {
        components: { BaseBadge, BaseToggle }
      }
    })
  }

  it('renders table element', () => {
    const wrapper = mountTable()
    expect(wrapper.find('table').exists()).toBe(true)
  })

  it('renders table headers', () => {
    const wrapper = mountTable()
    const headers = wrapper.findAll('th')
    expect(headers.length).toBe(8)
    expect(wrapper.text()).toContain('Status')
    expect(wrapper.text()).toContain('Server Name')
    expect(wrapper.text()).toContain('URL')
    expect(wrapper.text()).toContain('Protocol')
    expect(wrapper.text()).toContain('Auth Type')
    expect(wrapper.text()).toContain('Health')
    expect(wrapper.text()).toContain('Active')
    expect(wrapper.text()).toContain('Actions')
  })

  it('renders empty state when no servers', () => {
    const wrapper = mountTable([])
    expect(wrapper.text()).toContain('No servers found')
    expect(wrapper.text()).toContain('Get started by adding a new server')
  })

  it('renders server rows', () => {
    const wrapper = mountTable()
    const rows = wrapper.findAll('tbody tr')
    expect(rows.length).toBe(2)
  })

  it('displays server name', () => {
    const wrapper = mountTable()
    expect(wrapper.text()).toContain('API Server')
    expect(wrapper.text()).toContain('Dev Server')
  })

  it('displays server description', () => {
    const wrapper = mountTable()
    expect(wrapper.text()).toContain('Production API')
    expect(wrapper.text()).toContain('No description')
  })

  it('displays server URL', () => {
    const wrapper = mountTable()
    expect(wrapper.text()).toContain('http://api.example.com')
    expect(wrapper.text()).toContain('http://dev.example.com')
  })

  it('displays protocol version or N/A', () => {
    const wrapper = mountTable()
    expect(wrapper.text()).toContain('1.0')
    expect(wrapper.text()).toContain('N/A')
  })

  it('shows green status for healthy server', () => {
    const wrapper = mountTable()
    expect(wrapper.find('.bg-green-500').exists()).toBe(true)
  })

  it('shows red status for unhealthy server', () => {
    const wrapper = mountTable()
    expect(wrapper.find('.bg-red-500').exists()).toBe(true)
  })

  it('shows gray status for unknown health', () => {
    const wrapper = mountTable([{ id: '1', name: 'Test', is_active: true }])
    expect(wrapper.find('.bg-gray-400').exists()).toBe(true)
  })

  it('formats auth type correctly', () => {
    const wrapper = mountTable()
    expect(wrapper.text()).toContain('Bearer')
    expect(wrapper.text()).toContain('None')
  })

  it('formats health status correctly', () => {
    const wrapper = mountTable()
    expect(wrapper.text()).toContain('Healthy')
    expect(wrapper.text()).toContain('Unhealthy')
  })

  it('displays Unknown for missing health status', () => {
    const wrapper = mountTable([{ id: '1', name: 'Test', is_active: true }])
    expect(wrapper.text()).toContain('Unknown')
  })

  it('emits view-server on View click', async () => {
    const wrapper = mountTable()
    const viewButton = wrapper.findAll('button').find(b => b.text() === 'View')
    await viewButton?.trigger('click')
    expect(wrapper.emitted('view-server')).toBeTruthy()
    expect(wrapper.emitted('view-server')![0]).toEqual([mockServers[0]])
  })

  it('emits edit-server on Edit click', async () => {
    const wrapper = mountTable()
    const editButton = wrapper.findAll('button').find(b => b.text() === 'Edit')
    await editButton?.trigger('click')
    expect(wrapper.emitted('edit-server')).toBeTruthy()
    expect(wrapper.emitted('edit-server')![0]).toEqual([mockServers[0]])
  })

  it('emits delete-server on Delete click', async () => {
    const wrapper = mountTable()
    const deleteButton = wrapper.findAll('button').find(b => b.text() === 'Delete')
    await deleteButton?.trigger('click')
    expect(wrapper.emitted('delete-server')).toBeTruthy()
    expect(wrapper.emitted('delete-server')![0]).toEqual([mockServers[0]])
  })

  it('emits toggle-server on toggle change', async () => {
    const wrapper = mountTable()
    const toggle = wrapper.findComponent(BaseToggle)
    await toggle.trigger('click')
    expect(wrapper.emitted('toggle-server')).toBeTruthy()
    expect(wrapper.emitted('toggle-server')![0]).toEqual(['1'])
  })

  it('renders action buttons for each server', () => {
    const wrapper = mountTable()
    const viewButtons = wrapper.findAll('button').filter(b => b.text() === 'View')
    const editButtons = wrapper.findAll('button').filter(b => b.text() === 'Edit')
    const deleteButtons = wrapper.findAll('button').filter(b => b.text() === 'Delete')
    expect(viewButtons.length).toBe(2)
    expect(editButtons.length).toBe(2)
    expect(deleteButtons.length).toBe(2)
  })

  it('uses correct badge variant for oauth auth', () => {
    const servers = [{ id: '1', name: 'OAuth Server', auth_type: 'oauth', is_active: true }]
    const wrapper = mountTable(servers)
    const badge = wrapper.findComponent(BaseBadge)
    expect(badge.props('variant')).toBe('success')
  })

  it('uses correct badge variant for healthy status', () => {
    const wrapper = mountTable()
    // Find health badge by checking variant
    const badges = wrapper.findAllComponents(BaseBadge)
    const healthyBadge = badges.find(b => b.text() === 'Healthy')
    expect(healthyBadge?.props('variant')).toBe('success')
  })

  it('uses correct badge variant for degraded status', () => {
    const servers = [{ id: '1', name: 'Test', is_active: true, health: { status: 'degraded' } }]
    const wrapper = mountTable(servers)
    expect(wrapper.text()).toContain('Degraded')
    expect(wrapper.find('.bg-yellow-500').exists()).toBe(true)
  })

  it('formats recent timestamp as "Just now"', () => {
    const servers = [{
      id: '1',
      name: 'Test',
      is_active: true,
      health: { status: 'healthy', last_checked: new Date().toISOString() }
    }]
    const wrapper = mountTable(servers)
    expect(wrapper.text()).toContain('Just now')
  })

  it('has hover effect on rows', () => {
    const wrapper = mountTable()
    const rows = wrapper.findAll('tbody tr.hover\\:bg-gray-50')
    expect(rows.length).toBe(2)
  })
})
