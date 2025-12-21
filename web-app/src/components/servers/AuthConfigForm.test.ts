import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import AuthConfigForm from './AuthConfigForm.vue'

// Mock BaseInput
vi.mock('@/components/common/BaseInput.vue', () => ({
  default: {
    template: '<input class="input-mock" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" :placeholder="placeholder" :type="type" />',
    props: ['modelValue', 'label', 'placeholder', 'required', 'type', 'hint'],
    emits: ['update:modelValue']
  }
}))

describe('AuthConfigForm', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders the auth form', () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'none',
        authConfig: {}
      }
    })
    expect(wrapper.find('.space-y-4').exists()).toBe(true)
  })

  it('displays Authentication Type label', () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'none',
        authConfig: {}
      }
    })
    expect(wrapper.text()).toContain('Authentication Type')
  })

  it('displays all auth type options', () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'none',
        authConfig: {}
      }
    })
    expect(wrapper.text()).toContain('None')
    expect(wrapper.text()).toContain('Basic Auth')
    expect(wrapper.text()).toContain('Bearer Token')
    expect(wrapper.text()).toContain('OAuth 2.0')
  })

  it('has four radio inputs for auth types', () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'none',
        authConfig: {}
      }
    })
    const radios = wrapper.findAll('input[type="radio"]')
    expect(radios.length).toBe(4)
  })

  it('does not show auth config when type is none', () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'none',
        authConfig: {}
      }
    })
    expect(wrapper.find('.bg-gray-50').exists()).toBe(false)
  })

  it('shows Basic Authentication config when type is basic', () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'basic',
        authConfig: { username: '', password: '' }
      }
    })
    expect(wrapper.text()).toContain('Basic Authentication Config')
  })

  it('shows username and password inputs for basic auth', () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'basic',
        authConfig: { username: '', password: '' }
      }
    })
    const inputs = wrapper.findAll('.input-mock')
    expect(inputs.length).toBe(2)
  })

  it('shows Bearer Token config when type is bearer', () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'bearer',
        authConfig: { token: '' }
      }
    })
    expect(wrapper.text()).toContain('Bearer Token Config')
  })

  it('shows token input for bearer auth', () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'bearer',
        authConfig: { token: '' }
      }
    })
    const inputs = wrapper.findAll('.input-mock')
    expect(inputs.length).toBe(1)
  })

  it('shows OAuth 2.0 config when type is oauth', () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'oauth',
        authConfig: { client_id: '', client_secret: '', token_url: '', scopes: '' }
      }
    })
    expect(wrapper.text()).toContain('OAuth 2.0 Config')
  })

  it('shows all OAuth inputs', () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'oauth',
        authConfig: { client_id: '', client_secret: '', token_url: '', scopes: '' }
      }
    })
    const inputs = wrapper.findAll('.input-mock')
    expect(inputs.length).toBe(4) // client_id, client_secret, token_url, scopes
  })

  it('emits update:authType when radio is clicked', async () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'none',
        authConfig: {}
      }
    })

    const basicRadio = wrapper.findAll('input[type="radio"]').find(r => r.element.value === 'basic')
    await basicRadio?.setValue(true)
    await flushPromises()

    expect(wrapper.emitted('update:authType')).toBeTruthy()
    expect(wrapper.emitted('update:authType')![0]).toEqual(['basic'])
  })

  it('emits update:authConfig when config changes', async () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'bearer',
        authConfig: { token: '' }
      }
    })

    const tokenInput = wrapper.find('.input-mock')
    await tokenInput.setValue('my-token')
    await flushPromises()

    expect(wrapper.emitted('update:authConfig')).toBeTruthy()
  })

  it('selects correct radio based on authType prop', () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'basic',
        authConfig: { username: '', password: '' }
      }
    })

    const basicRadio = wrapper.find('input[value="basic"]')
    expect((basicRadio.element as HTMLInputElement).checked).toBe(true)
  })

  it('displays hint text about MCP protocol standards', () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'none',
        authConfig: {}
      }
    })
    expect(wrapper.text()).toContain('MCP protocol standards')
  })

  it('generates unique id when not provided', () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'none',
        authConfig: {}
      }
    })
    const radio = wrapper.find('input[type="radio"]')
    expect(radio.attributes('name')).toMatch(/auth_type_auth-/)
  })

  it('uses provided id for radio name attribute', () => {
    const wrapper = mount(AuthConfigForm, {
      props: {
        authType: 'none',
        authConfig: {},
        id: 'custom-id'
      }
    })
    const radio = wrapper.find('input[type="radio"]')
    expect(radio.attributes('name')).toBe('auth_type_custom-id')
  })
})
