import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ServerDetailsModal from './ServerDetailsModal.vue'

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

describe('ServerDetailsModal', () => {
  const mockServer = {
    id: 'server-123',
    name: 'Test MCP Server',
    description: 'A test server for MCP',
    url: 'http://localhost:9001',
    is_active: true,
    transport: 'streamable_http',
    protocol_version: '2025-11-25',
    tools: [
      { name: 'tool1', description: 'First tool' },
      { name: 'tool2', description: 'Second tool' }
    ],
    auth_type: 'none',
    created_at: '2024-01-15T10:00:00Z'
  }

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('does not render when modelValue is false', () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: false, server: mockServer }
    })
    expect(wrapper.find('.modal-mock').exists()).toBe(false)
  })

  it('renders when modelValue is true', () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.find('.modal-mock').exists()).toBe(true)
  })

  it('displays Server Details title', () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Server Details')
  })

  it('displays server name', () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Test MCP Server')
  })

  it('displays server description', () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('A test server for MCP')
  })

  it('displays active badge for active server', () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Active')
  })

  it('displays inactive badge for inactive server', () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: { ...mockServer, is_active: false } }
    })
    expect(wrapper.text()).toContain('Inactive')
  })

  it('has configuration tabs', () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Claude Code')
    expect(wrapper.text()).toContain('Cursor')
    expect(wrapper.text()).toContain('VS Code')
  })

  it('has Close button', () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Close')
  })

  it('emits update:modelValue false when Close is clicked', async () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })

    const closeButton = wrapper.findAll('.button-mock').find(b => b.text() === 'Close')
    await closeButton?.trigger('click')

    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([false])
  })

  it('shows Claude Code configuration by default', () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('claude mcp add')
  })

  it('shows CLI command option', () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Add via CLI')
  })

  it('shows manual configuration option', () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Manual Configuration')
  })

  it('has Copy buttons for code snippets', () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })
    const copyButtons = wrapper.findAll('button').filter(b => b.text().includes('Copy'))
    expect(copyButtons.length).toBeGreaterThan(0)
  })

  it('can switch to Cursor tab', async () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })

    const cursorTab = wrapper.findAll('button').find(b => b.text() === 'Cursor')
    await cursorTab?.trigger('click')

    expect(wrapper.text()).toContain('Add to Cursor')
  })

  it('shows One-Click Install button for Cursor', async () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })

    const cursorTab = wrapper.findAll('button').find(b => b.text() === 'Cursor')
    await cursorTab?.trigger('click')

    expect(wrapper.text()).toContain('One-Click Install')
  })

  it('can switch to VS Code tab', async () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })

    const vscodeTab = wrapper.findAll('button').find(b => b.text() === 'VS Code')
    await vscodeTab?.trigger('click')

    expect(wrapper.text()).toContain('.vscode/settings.json')
  })

  it('displays API key note', () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Replace')
    expect(wrapper.text()).toContain('api-key')
  })

  it('handles null server gracefully', () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: null }
    })
    expect(wrapper.find('.modal-mock').exists()).toBe(true)
  })

  it('displays code blocks with styling', () => {
    const wrapper = mount(ServerDetailsModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.find('pre').exists()).toBe(true)
    expect(wrapper.find('.bg-gray-900').exists()).toBe(true)
  })
})
