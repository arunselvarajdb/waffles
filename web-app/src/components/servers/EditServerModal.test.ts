import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import EditServerModal from './EditServerModal.vue'

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
    props: ['authType', 'authConfig', 'id'],
    emits: ['update:auth-type', 'update:auth-config']
  }
}))

// Mock API
vi.mock('@/services/api', () => ({
  default: {
    post: vi.fn()
  }
}))
import api from '@/services/api'

// Mock servers store
const mockUpdateServer = vi.fn()
vi.mock('@/stores/servers', () => ({
  useServersStore: vi.fn(() => ({
    updateServer: mockUpdateServer
  }))
}))

describe('EditServerModal', () => {
  const mockServer = {
    id: 'server-123',
    name: 'Test Server',
    description: 'Test description',
    url: 'http://localhost:9001',
    protocol_version: '1.0.0',
    transport: 'streamable_http',
    auth_type: 'none',
    auth_config: {},
    health_check_url: 'http://localhost:9001/health',
    health_check_interval: 60,
    timeout: 30,
    max_connections: 10,
    tags: ['tag1', 'tag2'],
    allowed_tools: ['tool1']
  }

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('does not render when modelValue is false', () => {
    const wrapper = mount(EditServerModal, {
      props: { modelValue: false, server: mockServer }
    })
    expect(wrapper.find('.modal-mock').exists()).toBe(false)
  })

  it('renders when modelValue is true', () => {
    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.find('.modal-mock').exists()).toBe(true)
  })

  it('displays Edit Server title', () => {
    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Edit Server')
  })

  it('has server name input', () => {
    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    const inputs = wrapper.findAll('.input-mock')
    expect(inputs.length).toBeGreaterThan(0)
  })

  it('has description textarea', () => {
    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.find('textarea').exists()).toBe(true)
  })

  it('has transport type selector', () => {
    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.find('select').exists()).toBe(true)
    expect(wrapper.text()).toContain('Transport Type')
  })

  it('has health check configuration section', () => {
    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Health Check Configuration')
  })

  it('has Cancel button', () => {
    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Cancel')
  })

  it('has Save Changes button', () => {
    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Save Changes')
  })

  it('has Test Connection button', () => {
    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Test Connection')
  })

  it('includes AuthConfigForm component', () => {
    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.find('.auth-config-mock').exists()).toBe(true)
  })

  it('emits update:modelValue false on cancel', async () => {
    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })

    const cancelButton = wrapper.findAll('.button-mock').find(b => b.text() === 'Cancel')
    await cancelButton?.trigger('click')

    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([false])
  })

  it('has transport type options', () => {
    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    const options = wrapper.findAll('option')
    expect(options.some(o => o.text().includes('Streamable HTTP'))).toBe(true)
    expect(options.some(o => o.text().includes('SSE'))).toBe(true)
    expect(options.some(o => o.text().includes('HTTP (Legacy)'))).toBe(true)
  })

  it('has Test Connection button that can be clicked', async () => {
    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })

    const testButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Test Connection'))
    expect(testButton?.exists()).toBe(true)
  })

  it('renders modal with server prop', () => {
    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    // The modal should render with the server data
    expect(wrapper.find('.modal-mock').exists()).toBe(true)
  })

  it('calls updateServer on submit', async () => {
    mockUpdateServer.mockResolvedValue({})

    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })

    const saveButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Save Changes'))
    await saveButton?.trigger('click')
    await flushPromises()

    expect(mockUpdateServer).toHaveBeenCalledWith('server-123', expect.any(Object))
  })

  it('emits success on successful submit', async () => {
    mockUpdateServer.mockResolvedValue({})

    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })

    const saveButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Save Changes'))
    await saveButton?.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('success')).toBeTruthy()
  })

  it('emits update:modelValue false on successful submit', async () => {
    mockUpdateServer.mockResolvedValue({})

    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })

    const saveButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Save Changes'))
    await saveButton?.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([false])
  })

  it('does not submit when server is null', async () => {
    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: null }
    })

    const saveButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Save Changes'))
    await saveButton?.trigger('click')
    await flushPromises()

    expect(mockUpdateServer).not.toHaveBeenCalled()
  })

  it('handles submit error gracefully', async () => {
    mockUpdateServer.mockRejectedValue(new Error('Update failed'))

    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })

    const saveButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Save Changes'))
    await saveButton?.trigger('click')
    await flushPromises()

    // Should not emit success on error
    expect(wrapper.emitted('success')).toBeFalsy()
  })

  it('parses tags from comma-separated string on submit', async () => {
    mockUpdateServer.mockResolvedValue({})

    const wrapper = mount(EditServerModal, {
      props: { modelValue: true, server: mockServer }
    })

    const saveButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Save Changes'))
    await saveButton?.trigger('click')
    await flushPromises()

    expect(mockUpdateServer).toHaveBeenCalledWith(
      'server-123',
      expect.objectContaining({
        tags: expect.any(Array)
      })
    )
  })
})
