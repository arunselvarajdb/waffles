import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import NamespaceDetailsModal from './NamespaceDetailsModal.vue'

// Mock BaseModal
vi.mock('@/components/common/BaseModal.vue', () => ({
  default: {
    template: `
      <div v-if="modelValue" class="modal-mock">
        <h3>{{ title }}</h3>
        <slot />
        <div class="footer"><slot name="footer" /></div>
      </div>
    `,
    props: ['modelValue', 'title', 'size'],
    emits: ['update:modelValue', 'close']
  }
}))

// Mock BaseButton
vi.mock('@/components/common/BaseButton.vue', () => ({
  default: {
    template: '<button class="button-mock" :disabled="disabled" @click="$emit(\'click\')"><slot /></button>',
    props: ['variant', 'disabled', 'loading'],
    emits: ['click']
  }
}))

// Mock BaseBadge
vi.mock('@/components/common/BaseBadge.vue', () => ({
  default: {
    template: '<span class="badge-mock"><slot /></span>',
    props: ['variant']
  }
}))

// Mock namespaces store
const mockFetchNamespaceServers = vi.fn()
const mockFetchNamespaceAccess = vi.fn()
const mockAddServerToNamespace = vi.fn()
const mockRemoveServerFromNamespace = vi.fn()
const mockSetRoleAccess = vi.fn()
const mockRemoveRoleAccess = vi.fn()

vi.mock('@/stores/namespaces', () => ({
  useNamespacesStore: vi.fn(() => ({
    currentNamespaceServers: [],
    currentNamespaceAccess: [],
    fetchNamespaceServers: mockFetchNamespaceServers,
    fetchNamespaceAccess: mockFetchNamespaceAccess,
    addServerToNamespace: mockAddServerToNamespace,
    removeServerFromNamespace: mockRemoveServerFromNamespace,
    setRoleAccess: mockSetRoleAccess,
    removeRoleAccess: mockRemoveRoleAccess
  }))
}))

// Mock servers store
const mockFetchServers = vi.fn()
vi.mock('@/stores/servers', () => ({
  useServersStore: vi.fn(() => ({
    servers: [
      { id: 'server-1', name: 'Server 1' },
      { id: 'server-2', name: 'Server 2' }
    ],
    fetchServers: mockFetchServers
  }))
}))

// Mock roles store
const mockFetchRoles = vi.fn()
vi.mock('@/stores/roles', () => ({
  useRolesStore: vi.fn(() => ({
    roles: [
      { id: 'role-1', name: 'admin' },
      { id: 'role-2', name: 'viewer' }
    ],
    fetchRoles: mockFetchRoles
  }))
}))

describe('NamespaceDetailsModal', () => {
  const mockNamespace = {
    id: 'ns-123',
    name: 'engineering',
    description: 'Engineering namespace',
    created_at: '2024-01-15T10:00:00Z'
  }

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockFetchNamespaceServers.mockResolvedValue([])
    mockFetchNamespaceAccess.mockResolvedValue([])
    mockFetchServers.mockResolvedValue([])
    mockFetchRoles.mockResolvedValue([])
  })

  it('does not render when modelValue is false', () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: false, namespace: mockNamespace }
    })
    expect(wrapper.find('.modal-mock').exists()).toBe(false)
  })

  it('renders when modelValue is true', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    await flushPromises()
    expect(wrapper.find('.modal-mock').exists()).toBe(true)
  })

  it('displays namespace name in title', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    await flushPromises()
    expect(wrapper.text()).toContain('engineering')
  })

  it('displays namespace description', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    await flushPromises()
    expect(wrapper.text()).toContain('Engineering namespace')
  })

  it('displays "No description" when description is empty', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: { ...mockNamespace, description: '' } }
    })
    await flushPromises()
    expect(wrapper.text()).toContain('No description')
  })

  it('displays formatted creation date', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    await flushPromises()
    expect(wrapper.text()).toContain('Jan 15, 2024')
  })

  it('has Servers and Role Access tabs', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    await flushPromises()
    expect(wrapper.text()).toContain('Servers')
    expect(wrapper.text()).toContain('Role Access')
  })

  it('shows Close button in footer', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    await flushPromises()
    expect(wrapper.text()).toContain('Close')
  })

  it('emits update:modelValue false when Close is clicked', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    await flushPromises()

    const closeButton = wrapper.findAll('.button-mock').find(b => b.text() === 'Close')
    await closeButton?.trigger('click')

    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([false])
  })

  it('has server select dropdown', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    await flushPromises()

    const select = wrapper.find('select')
    expect(select.exists()).toBe(true)
  })

  it('has Add button for adding servers', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Add')
  })

  it('shows "No servers in this namespace" when empty', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('No servers in this namespace')
  })

  it('can switch to Role Access tab', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    await flushPromises()

    const accessTab = wrapper.findAll('button').find(b => b.text().includes('Role Access'))
    await accessTab?.trigger('click')

    expect(wrapper.text()).toContain('No role access configured')
  })

  it('has role select dropdown on access tab', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    await flushPromises()

    const accessTab = wrapper.findAll('button').find(b => b.text().includes('Role Access'))
    await accessTab?.trigger('click')

    const selects = wrapper.findAll('select')
    expect(selects.length).toBeGreaterThanOrEqual(1)
  })

  it('has access level dropdown with View and Execute options', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    await flushPromises()

    const accessTab = wrapper.findAll('button').find(b => b.text().includes('Role Access'))
    await accessTab?.trigger('click')

    const options = wrapper.findAll('option')
    expect(options.some(o => o.text() === 'View')).toBe(true)
    expect(options.some(o => o.text() === 'Execute')).toBe(true)
  })

  it('has Set button for setting role access', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    await flushPromises()

    const accessTab = wrapper.findAll('button').find(b => b.text().includes('Role Access'))
    await accessTab?.trigger('click')

    expect(wrapper.text()).toContain('Set')
  })

  it('has correct structure when rendered', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    await flushPromises()

    // Verify the component renders with expected structure
    expect(wrapper.find('.modal-mock').exists()).toBe(true)
    expect(wrapper.findAll('select').length).toBeGreaterThan(0)
    expect(wrapper.findAll('button').length).toBeGreaterThan(0)
  })

  it('displays namespace info section', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    await flushPromises()

    expect(wrapper.find('.bg-gray-50').exists()).toBe(true)
    expect(wrapper.text()).toContain('Name')
    expect(wrapper.text()).toContain('Created')
    expect(wrapper.text()).toContain('Description')
  })

  it('handles null namespace gracefully', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: null }
    })
    await flushPromises()

    // Should render with default title
    expect(wrapper.text()).toContain('Namespace Details')
  })

  it('shows N/A for null dates', async () => {
    const wrapper = mount(NamespaceDetailsModal, {
      props: { modelValue: true, namespace: { ...mockNamespace, created_at: null } }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('N/A')
  })
})
