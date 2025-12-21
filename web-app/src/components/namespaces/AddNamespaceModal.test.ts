import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import AddNamespaceModal from './AddNamespaceModal.vue'

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
    template: '<input class="input-mock" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
    props: ['modelValue', 'label', 'placeholder', 'required', 'error', 'hint'],
    emits: ['update:modelValue']
  }
}))

// Mock BaseButton
vi.mock('@/components/common/BaseButton.vue', () => ({
  default: {
    template: '<button class="button-mock" :disabled="disabled"><slot /></button>',
    props: ['variant', 'disabled']
  }
}))

// Mock namespaces store
const mockCreateNamespace = vi.fn()
vi.mock('@/stores/namespaces', () => ({
  useNamespacesStore: vi.fn(() => ({
    createNamespace: mockCreateNamespace
  }))
}))

describe('AddNamespaceModal', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('does not render when modelValue is false', () => {
    const wrapper = mount(AddNamespaceModal, {
      props: { modelValue: false }
    })
    expect(wrapper.find('.modal-mock').exists()).toBe(false)
  })

  it('renders when modelValue is true', () => {
    const wrapper = mount(AddNamespaceModal, {
      props: { modelValue: true }
    })
    expect(wrapper.find('.modal-mock').exists()).toBe(true)
  })

  it('displays Create Namespace title', () => {
    const wrapper = mount(AddNamespaceModal, {
      props: { modelValue: true }
    })
    expect(wrapper.text()).toContain('Create Namespace')
  })

  it('has name input', () => {
    const wrapper = mount(AddNamespaceModal, {
      props: { modelValue: true }
    })
    expect(wrapper.find('.input-mock').exists()).toBe(true)
  })

  it('has description textarea', () => {
    const wrapper = mount(AddNamespaceModal, {
      props: { modelValue: true }
    })
    expect(wrapper.find('textarea').exists()).toBe(true)
  })

  it('has Cancel button', () => {
    const wrapper = mount(AddNamespaceModal, {
      props: { modelValue: true }
    })
    expect(wrapper.text()).toContain('Cancel')
  })

  it('has Create Namespace button', () => {
    const wrapper = mount(AddNamespaceModal, {
      props: { modelValue: true }
    })
    const buttons = wrapper.findAll('.button-mock')
    expect(buttons.some(b => b.text().includes('Create Namespace'))).toBe(true)
  })

  it('emits update:modelValue false on cancel', async () => {
    const wrapper = mount(AddNamespaceModal, {
      props: { modelValue: true }
    })

    const cancelButton = wrapper.findAll('.button-mock').find(b => b.text() === 'Cancel')
    await cancelButton?.trigger('click')

    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([false])
  })

  it('calls createNamespace on successful submit', async () => {
    mockCreateNamespace.mockResolvedValue({})

    const wrapper = mount(AddNamespaceModal, {
      props: { modelValue: true }
    })

    // Set name value
    const input = wrapper.find('.input-mock')
    await input.setValue('test-namespace')

    // Click Create button
    const createButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Create Namespace'))
    await createButton?.trigger('click')
    await flushPromises()

    expect(mockCreateNamespace).toHaveBeenCalled()
  })

  it('emits success on successful create', async () => {
    mockCreateNamespace.mockResolvedValue({})

    const wrapper = mount(AddNamespaceModal, {
      props: { modelValue: true }
    })

    const input = wrapper.find('.input-mock')
    await input.setValue('test-namespace')

    const createButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Create Namespace'))
    await createButton?.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('success')).toBeTruthy()
  })

  it('does not create with empty name', async () => {
    const wrapper = mount(AddNamespaceModal, {
      props: { modelValue: true }
    })

    // Click Create without setting name
    const createButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Create Namespace'))
    await createButton?.trigger('click')
    await flushPromises()

    expect(mockCreateNamespace).not.toHaveBeenCalled()
  })

  it('handles create error gracefully', async () => {
    mockCreateNamespace.mockRejectedValue(new Error('Create failed'))

    const wrapper = mount(AddNamespaceModal, {
      props: { modelValue: true }
    })

    const input = wrapper.find('.input-mock')
    await input.setValue('test-namespace')

    const createButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Create Namespace'))
    await createButton?.trigger('click')
    await flushPromises()

    // Should not emit success on error
    expect(wrapper.emitted('success')).toBeFalsy()
  })
})
