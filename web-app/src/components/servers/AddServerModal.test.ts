import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import AddServerModal from './AddServerModal.vue'

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

// Mock BaseInput
vi.mock('@/components/common/BaseInput.vue', () => ({
  default: {
    template: '<input class="input-mock" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" :placeholder="placeholder" />',
    props: ['modelValue', 'label', 'placeholder', 'required', 'type', 'hint'],
    emits: ['update:modelValue']
  }
}))

// Mock BaseButton
vi.mock('@/components/common/BaseButton.vue', () => ({
  default: {
    template: '<button class="button-mock" :disabled="disabled || loading" @click="$emit(\'click\')"><slot /></button>',
    props: ['variant', 'disabled', 'loading', 'size', 'type'],
    emits: ['click']
  }
}))

// Mock AuthConfigForm
vi.mock('./AuthConfigForm.vue', () => ({
  default: {
    template: '<div class="auth-config-mock">Auth Config</div>',
    props: ['authType', 'authConfig'],
    emits: ['update:auth-type', 'update:auth-config']
  }
}))

// Mock API
const mockApiPost = vi.fn()
vi.mock('@/services/api', () => ({
  default: {
    post: vi.fn()
  }
}))
import api from '@/services/api'

// Mock servers store
const mockCreateServer = vi.fn()
vi.mock('@/stores/servers', () => ({
  useServersStore: vi.fn(() => ({
    createServer: mockCreateServer
  }))
}))

describe('AddServerModal', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('does not render when modelValue is false', () => {
    const wrapper = mount(AddServerModal, {
      props: { modelValue: false }
    })
    expect(wrapper.find('.modal-mock').exists()).toBe(false)
  })

  it('renders when modelValue is true', () => {
    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })
    expect(wrapper.find('.modal-mock').exists()).toBe(true)
  })

  it('displays Add New Server title', () => {
    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })
    expect(wrapper.text()).toContain('Add New Server')
  })

  it('has server name input', () => {
    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })
    const inputs = wrapper.findAll('.input-mock')
    expect(inputs.length).toBeGreaterThan(0)
  })

  it('has description textarea', () => {
    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })
    expect(wrapper.find('textarea').exists()).toBe(true)
  })

  it('has transport type selector', () => {
    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })
    expect(wrapper.find('select').exists()).toBe(true)
    expect(wrapper.text()).toContain('Transport Type')
  })

  it('has health check configuration section', () => {
    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })
    expect(wrapper.text()).toContain('Health Check Configuration')
  })

  it('has Cancel button', () => {
    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })
    expect(wrapper.text()).toContain('Cancel')
  })

  it('has Add Server button', () => {
    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })
    expect(wrapper.text()).toContain('Add Server')
  })

  it('has Test Connection button', () => {
    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })
    expect(wrapper.text()).toContain('Test Connection')
  })

  it('includes AuthConfigForm component', () => {
    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })
    expect(wrapper.find('.auth-config-mock').exists()).toBe(true)
  })

  it('emits update:modelValue false on cancel', async () => {
    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })

    const cancelButton = wrapper.findAll('.button-mock').find(b => b.text() === 'Cancel')
    await cancelButton?.trigger('click')

    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([false])
  })

  it('has transport type options', () => {
    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })
    const options = wrapper.findAll('option')
    expect(options.some(o => o.text().includes('Streamable HTTP'))).toBe(true)
    expect(options.some(o => o.text().includes('SSE'))).toBe(true)
    expect(options.some(o => o.text().includes('HTTP (Legacy)'))).toBe(true)
  })

  it('calls testConnection when Test Connection clicked', async () => {
    vi.mocked(api.post).mockResolvedValue({ success: true, tools: [] })

    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })

    // Set URL first
    const urlInput = wrapper.findAll('.input-mock')[1] // URL input
    await urlInput.setValue('http://localhost:9001')
    await wrapper.vm.$nextTick()

    const testButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Test Connection'))
    await testButton?.trigger('click')
    await flushPromises()

    expect(api.post).toHaveBeenCalledWith('/servers/test-connection', expect.objectContaining({
      url: 'http://localhost:9001'
    }))
  })

  it('shows Connected on successful connection test', async () => {
    vi.mocked(api.post).mockResolvedValue({ success: true, tools: [], tool_count: 5 })

    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })

    const urlInput = wrapper.findAll('.input-mock')[1]
    await urlInput.setValue('http://localhost:9001')

    const testButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Test Connection'))
    await testButton?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Connected')
  })

  it('shows Failed on connection test error', async () => {
    vi.mocked(api.post).mockRejectedValue({
      response: { data: { error: 'Connection refused' } }
    })

    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })

    const urlInput = wrapper.findAll('.input-mock')[1]
    await urlInput.setValue('http://localhost:9001')

    const testButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Test Connection'))
    await testButton?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Failed')
  })

  it('shows discovered tools section after successful connection', async () => {
    vi.mocked(api.post).mockResolvedValue({
      success: true,
      tools: [
        { name: 'tool1', description: 'First tool' },
        { name: 'tool2', description: 'Second tool' }
      ],
      tool_count: 2
    })

    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })

    const urlInput = wrapper.findAll('.input-mock')[1]
    await urlInput.setValue('http://localhost:9001')

    const testButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Test Connection'))
    await testButton?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Available Tools')
    expect(wrapper.text()).toContain('tool1')
    expect(wrapper.text()).toContain('tool2')
  })

  it('has Select All and Deselect All buttons when tools discovered', async () => {
    vi.mocked(api.post).mockResolvedValue({
      success: true,
      tools: [
        { name: 'tool1', description: 'First tool' }
      ],
      tool_count: 1
    })

    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })

    const urlInput = wrapper.findAll('.input-mock')[1]
    await urlInput.setValue('http://localhost:9001')

    const testButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Test Connection'))
    await testButton?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Select All')
    expect(wrapper.text()).toContain('Deselect All')
  })

  it('calls createServer on submit', async () => {
    mockCreateServer.mockResolvedValue({})

    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })

    // Set required fields
    const inputs = wrapper.findAll('.input-mock')
    await inputs[0].setValue('Test Server') // name
    await inputs[1].setValue('http://localhost:9001') // url

    const addButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Add Server'))
    await addButton?.trigger('click')
    await flushPromises()

    expect(mockCreateServer).toHaveBeenCalled()
  })

  it('emits success on successful submit', async () => {
    mockCreateServer.mockResolvedValue({})

    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })

    const inputs = wrapper.findAll('.input-mock')
    await inputs[0].setValue('Test Server')
    await inputs[1].setValue('http://localhost:9001')

    const addButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Add Server'))
    await addButton?.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('success')).toBeTruthy()
  })

  it('emits update:modelValue false on successful submit', async () => {
    mockCreateServer.mockResolvedValue({})

    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })

    const inputs = wrapper.findAll('.input-mock')
    await inputs[0].setValue('Test Server')
    await inputs[1].setValue('http://localhost:9001')

    const addButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Add Server'))
    await addButton?.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([false])
  })

  it('handles submit error gracefully', async () => {
    mockCreateServer.mockRejectedValue(new Error('Create failed'))

    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })

    const inputs = wrapper.findAll('.input-mock')
    await inputs[0].setValue('Test Server')
    await inputs[1].setValue('http://localhost:9001')

    const addButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Add Server'))
    await addButton?.trigger('click')
    await flushPromises()

    // Should not emit success on error
    expect(wrapper.emitted('success')).toBeFalsy()
  })

  it('accepts prefillData prop', async () => {
    const prefillData = {
      url: 'http://prefill.example.com',
      protocol_version: '1.0.0',
      transport: 'sse',
      server_info: { name: 'Prefilled Server' },
      tools: [{ name: 'tool1', description: 'Tool 1' }],
      allowed_tools: ['tool1']
    }

    const wrapper = mount(AddServerModal, {
      props: { modelValue: true, prefillData }
    })

    // Modal should render with prefill data
    expect(wrapper.find('.modal-mock').exists()).toBe(true)
  })

  it('parses tags from comma-separated string', async () => {
    mockCreateServer.mockResolvedValue({})

    const wrapper = mount(AddServerModal, {
      props: { modelValue: true }
    })

    const inputs = wrapper.findAll('.input-mock')
    await inputs[0].setValue('Test Server')
    await inputs[1].setValue('http://localhost:9001')

    // Find the tags input (last one)
    const tagsInput = inputs[inputs.length - 1]
    await tagsInput.setValue('tag1, tag2, tag3')

    const addButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Add Server'))
    await addButton?.trigger('click')
    await flushPromises()

    expect(mockCreateServer).toHaveBeenCalledWith(
      expect.objectContaining({
        tags: ['tag1', 'tag2', 'tag3']
      })
    )
  })
})
