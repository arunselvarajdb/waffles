import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import NamespaceTable from './NamespaceTable.vue'
import BaseBadge from '@/components/common/BaseBadge.vue'

describe('NamespaceTable', () => {
  const mockNamespaces = [
    {
      id: '1',
      name: 'production',
      description: 'Production environment',
      server_count: 5,
      created_at: '2024-01-15T10:00:00Z'
    },
    {
      id: '2',
      name: 'development',
      description: null,
      server_count: 0,
      created_at: '2024-02-20T14:30:00Z'
    }
  ]

  const mountTable = (namespaces = mockNamespaces) => {
    return mount(NamespaceTable, {
      props: { namespaces },
      global: {
        components: { BaseBadge }
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
    expect(headers.length).toBe(5)
    expect(wrapper.text()).toContain('Name')
    expect(wrapper.text()).toContain('Description')
    expect(wrapper.text()).toContain('Servers')
    expect(wrapper.text()).toContain('Created')
    expect(wrapper.text()).toContain('Actions')
  })

  it('renders empty state when no namespaces', () => {
    const wrapper = mountTable([])
    expect(wrapper.text()).toContain('No namespaces found')
    expect(wrapper.text()).toContain('Get started by creating a new namespace')
  })

  it('renders namespace rows', () => {
    const wrapper = mountTable()
    const rows = wrapper.findAll('tbody tr')
    expect(rows.length).toBe(2)
  })

  it('displays namespace name', () => {
    const wrapper = mountTable()
    expect(wrapper.text()).toContain('production')
    expect(wrapper.text()).toContain('development')
  })

  it('displays namespace description', () => {
    const wrapper = mountTable()
    expect(wrapper.text()).toContain('Production environment')
    expect(wrapper.text()).toContain('No description')
  })

  it('displays server count in badge', () => {
    const wrapper = mountTable()
    expect(wrapper.text()).toContain('5 servers')
    expect(wrapper.text()).toContain('0 servers')
  })

  it('formats created date correctly', () => {
    const wrapper = mountTable()
    expect(wrapper.text()).toContain('Jan')
    expect(wrapper.text()).toContain('2024')
  })

  it('displays N/A for missing created_at', () => {
    const namespaces = [{ id: '1', name: 'test', created_at: null }]
    const wrapper = mountTable(namespaces)
    expect(wrapper.text()).toContain('N/A')
  })

  it('emits view-namespace on View click', async () => {
    const wrapper = mountTable()
    const viewButton = wrapper.findAll('button').find(b => b.text() === 'View')
    await viewButton?.trigger('click')
    expect(wrapper.emitted('view-namespace')).toBeTruthy()
    expect(wrapper.emitted('view-namespace')![0]).toEqual([mockNamespaces[0]])
  })

  it('emits edit-namespace on Edit click', async () => {
    const wrapper = mountTable()
    const editButton = wrapper.findAll('button').find(b => b.text() === 'Edit')
    await editButton?.trigger('click')
    expect(wrapper.emitted('edit-namespace')).toBeTruthy()
    expect(wrapper.emitted('edit-namespace')![0]).toEqual([mockNamespaces[0]])
  })

  it('emits delete-namespace on Delete click', async () => {
    const wrapper = mountTable()
    const deleteButton = wrapper.findAll('button').find(b => b.text() === 'Delete')
    await deleteButton?.trigger('click')
    expect(wrapper.emitted('delete-namespace')).toBeTruthy()
    expect(wrapper.emitted('delete-namespace')![0]).toEqual([mockNamespaces[0]])
  })

  it('renders action buttons for each namespace', () => {
    const wrapper = mountTable()
    const viewButtons = wrapper.findAll('button').filter(b => b.text() === 'View')
    const editButtons = wrapper.findAll('button').filter(b => b.text() === 'Edit')
    const deleteButtons = wrapper.findAll('button').filter(b => b.text() === 'Delete')
    expect(viewButtons.length).toBe(2)
    expect(editButtons.length).toBe(2)
    expect(deleteButtons.length).toBe(2)
  })

  it('uses info variant for server count badge', () => {
    const wrapper = mountTable()
    const badge = wrapper.findComponent(BaseBadge)
    expect(badge.props('variant')).toBe('info')
  })

  it('has hover effect on rows', () => {
    const wrapper = mountTable()
    const rows = wrapper.findAll('tbody tr.hover\\:bg-gray-50')
    expect(rows.length).toBe(2)
  })

  it('handles 0 server count correctly', () => {
    const namespaces = [{ id: '1', name: 'empty' }]
    const wrapper = mountTable(namespaces)
    expect(wrapper.text()).toContain('0 servers')
  })

  it('truncates long descriptions', () => {
    const wrapper = mountTable()
    expect(wrapper.find('.truncate').exists()).toBe(true)
  })
})
