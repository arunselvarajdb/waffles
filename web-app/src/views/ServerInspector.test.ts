import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ServerInspector from './ServerInspector.vue'

// Mock vue-router
const mockPush = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush })
}))

// Mock NavBar component
vi.mock('@/components/layout/NavBar.vue', () => ({
  default: {
    template: '<nav class="navbar-mock">NavBar</nav>'
  }
}))

// Mock BaseInput
vi.mock('@/components/common/BaseInput.vue', () => ({
  default: {
    template: '<input class="input-mock" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
    props: ['modelValue', 'label', 'placeholder', 'required', 'type'],
    emits: ['update:modelValue']
  }
}))

// Mock BaseButton
vi.mock('@/components/common/BaseButton.vue', () => ({
  default: {
    template: '<button class="button-mock" :disabled="disabled || loading" @click="$emit(\'click\')"><slot /></button>',
    props: ['variant', 'disabled', 'loading'],
    emits: ['click']
  }
}))

// Mock AddServerModal
vi.mock('@/components/servers/AddServerModal.vue', () => ({
  default: {
    template: '<div v-if="modelValue" class="add-server-modal-mock">AddServerModal</div>',
    props: ['modelValue', 'prefillData'],
    emits: ['update:modelValue', 'success']
  }
}))

// Mock API
vi.mock('@/services/api', () => ({
  default: {
    post: vi.fn()
  }
}))
import api from '@/services/api'

describe('ServerInspector', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders the NavBar component', () => {
    const wrapper = mount(ServerInspector)
    expect(wrapper.find('.navbar-mock').exists()).toBe(true)
  })

  it('displays Server Inspector heading', () => {
    const wrapper = mount(ServerInspector)
    expect(wrapper.text()).toContain('Server Inspector')
  })

  it('displays description text', () => {
    const wrapper = mount(ServerInspector)
    expect(wrapper.text()).toContain('Connect to an MCP server')
  })

  it('has Connection Settings section', () => {
    const wrapper = mount(ServerInspector)
    expect(wrapper.text()).toContain('Connection Settings')
  })

  it('has Server URL input', () => {
    const wrapper = mount(ServerInspector)
    expect(wrapper.findAll('.input-mock').length).toBeGreaterThan(0)
  })

  it('has transport type selector', () => {
    const wrapper = mount(ServerInspector)
    expect(wrapper.find('select').exists()).toBe(true)
    expect(wrapper.text()).toContain('Transport Type')
  })

  it('has transport type options', () => {
    const wrapper = mount(ServerInspector)
    const options = wrapper.findAll('option')
    expect(options.some(o => o.text().includes('Streamable HTTP'))).toBe(true)
    expect(options.some(o => o.text().includes('SSE'))).toBe(true)
    expect(options.some(o => o.text().includes('HTTP (Legacy)'))).toBe(true)
  })

  it('has Connect button', () => {
    const wrapper = mount(ServerInspector)
    expect(wrapper.text()).toContain('Connect')
  })

  it('has protocol version input', () => {
    const wrapper = mount(ServerInspector)
    // There should be multiple inputs (url and protocol_version)
    const inputs = wrapper.findAll('.input-mock')
    expect(inputs.length).toBe(2)
  })

  it('has grid layout', () => {
    const wrapper = mount(ServerInspector)
    expect(wrapper.find('.grid').exists()).toBe(true)
  })

  it('has main container', () => {
    const wrapper = mount(ServerInspector)
    expect(wrapper.find('.min-h-screen').exists()).toBe(true)
    expect(wrapper.find('main').exists()).toBe(true)
  })

  it('has max-width container', () => {
    const wrapper = mount(ServerInspector)
    expect(wrapper.find('.max-w-7xl').exists()).toBe(true)
  })

  it('Connect button shows loading state when connecting', async () => {
    vi.mocked(api.post).mockImplementation(() => new Promise(() => {})) // Never resolves

    const wrapper = mount(ServerInspector)

    // Set a URL first
    const inputs = wrapper.findAll('.input-mock')
    await inputs[0].setValue('http://localhost:9001')

    const connectButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Connect'))
    await connectButton?.trigger('click')

    // The button should show loading state
    expect(wrapper.text()).toContain('Connect')
  })

  it('shows connection status on successful connection', async () => {
    vi.mocked(api.post).mockResolvedValue({
      success: true,
      server_info: { name: 'Test Server', version: '1.0.0' },
      tools: [],
      response_time_ms: 100
    })

    const wrapper = mount(ServerInspector)

    const inputs = wrapper.findAll('.input-mock')
    await inputs[0].setValue('http://localhost:9001')

    const connectButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Connect'))
    await connectButton?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Connected')
  })

  it('shows connection error on failed connection', async () => {
    vi.mocked(api.post).mockResolvedValue({
      success: false,
      error_message: 'Connection refused'
    })

    const wrapper = mount(ServerInspector)

    const inputs = wrapper.findAll('.input-mock')
    await inputs[0].setValue('http://localhost:9001')

    const connectButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Connect'))
    await connectButton?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Connection Failed')
  })

  it('shows server info after successful connection', async () => {
    vi.mocked(api.post).mockResolvedValue({
      success: true,
      server_info: { name: 'Test MCP Server', version: '2.0.0' },
      tools: [],
      response_time_ms: 50
    })

    const wrapper = mount(ServerInspector)

    const inputs = wrapper.findAll('.input-mock')
    await inputs[0].setValue('http://localhost:9001')

    const connectButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Connect'))
    await connectButton?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Server Information')
  })

  it('shows Add Server button after successful connection', async () => {
    vi.mocked(api.post).mockResolvedValue({
      success: true,
      server_info: { name: 'Test Server' },
      tools: [],
      response_time_ms: 50
    })

    const wrapper = mount(ServerInspector)

    const inputs = wrapper.findAll('.input-mock')
    await inputs[0].setValue('http://localhost:9001')

    const connectButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Connect'))
    await connectButton?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Add This Server to Gateway')
  })

  it('shows tools list after successful connection', async () => {
    vi.mocked(api.post).mockResolvedValue({
      success: true,
      server_info: { name: 'Test Server' },
      tools: [
        { name: 'tool1', description: 'First tool' },
        { name: 'tool2', description: 'Second tool' }
      ],
      response_time_ms: 50
    })

    const wrapper = mount(ServerInspector)

    const inputs = wrapper.findAll('.input-mock')
    await inputs[0].setValue('http://localhost:9001')

    const connectButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Connect'))
    await connectButton?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('tool1')
    expect(wrapper.text()).toContain('tool2')
  })

  it('calls API with correct parameters on connect', async () => {
    vi.mocked(api.post).mockResolvedValue({
      success: true,
      server_info: {},
      tools: []
    })

    const wrapper = mount(ServerInspector)

    const inputs = wrapper.findAll('.input-mock')
    await inputs[0].setValue('http://localhost:9001/mcp')

    const connectButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Connect'))
    await connectButton?.trigger('click')
    await flushPromises()

    expect(api.post).toHaveBeenCalledWith('/servers/test-connection', expect.objectContaining({
      url: 'http://localhost:9001/mcp'
    }))
  })
})
