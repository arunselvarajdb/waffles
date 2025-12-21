import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import EditNamespaceModal from './EditNamespaceModal.vue'

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
    template: '<button class="button-mock" :disabled="disabled" @click="$emit(\'click\')"><slot /></button>',
    props: ['variant', 'disabled'],
    emits: ['click']
  }
}))

// Mock namespaces store
const mockUpdateNamespace = vi.fn()
vi.mock('@/stores/namespaces', () => ({
  useNamespacesStore: vi.fn(() => ({
    updateNamespace: mockUpdateNamespace
  }))
}))

describe('EditNamespaceModal', () => {
  const mockNamespace = {
    id: 'ns-123',
    name: 'test-namespace',
    description: 'Test description'
  }

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('does not render when modelValue is false', () => {
    const wrapper = mount(EditNamespaceModal, {
      props: { modelValue: false, namespace: mockNamespace }
    })
    expect(wrapper.find('.modal-mock').exists()).toBe(false)
  })

  it('renders when modelValue is true', () => {
    const wrapper = mount(EditNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    expect(wrapper.find('.modal-mock').exists()).toBe(true)
  })

  it('displays Edit Namespace title', () => {
    const wrapper = mount(EditNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    expect(wrapper.text()).toContain('Edit Namespace')
  })

  it('has name input', () => {
    const wrapper = mount(EditNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    expect(wrapper.find('.input-mock').exists()).toBe(true)
  })

  it('has description textarea', () => {
    const wrapper = mount(EditNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    expect(wrapper.find('textarea').exists()).toBe(true)
  })

  it('has Cancel button', () => {
    const wrapper = mount(EditNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    expect(wrapper.text()).toContain('Cancel')
  })

  it('has Save Changes button', () => {
    const wrapper = mount(EditNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    expect(wrapper.text()).toContain('Save Changes')
  })

  it('emits update:modelValue false on cancel', async () => {
    const wrapper = mount(EditNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })

    const cancelButton = wrapper.findAll('.button-mock').find(b => b.text() === 'Cancel')
    await cancelButton?.trigger('click')

    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([false])
  })

  it('calls updateNamespace on successful submit', async () => {
    mockUpdateNamespace.mockResolvedValue({})

    const wrapper = mount(EditNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })

    // Set name value
    const input = wrapper.find('.input-mock')
    await input.setValue('updated-namespace')

    // Click Save button
    const saveButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Save Changes'))
    await saveButton?.trigger('click')
    await flushPromises()

    expect(mockUpdateNamespace).toHaveBeenCalled()
  })

  it('emits success on successful update', async () => {
    mockUpdateNamespace.mockResolvedValue({})

    const wrapper = mount(EditNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })

    const input = wrapper.find('.input-mock')
    await input.setValue('updated-namespace')

    const saveButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Save Changes'))
    await saveButton?.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('success')).toBeTruthy()
  })

  it('does not update with empty name', async () => {
    const wrapper = mount(EditNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })

    // Clear the name field
    const input = wrapper.find('.input-mock')
    await input.setValue('')

    const saveButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Save Changes'))
    await saveButton?.trigger('click')
    await flushPromises()

    expect(mockUpdateNamespace).not.toHaveBeenCalled()
  })

  it('does not update with name shorter than 2 chars', async () => {
    const wrapper = mount(EditNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })

    const input = wrapper.find('.input-mock')
    await input.setValue('a')

    const saveButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Save Changes'))
    await saveButton?.trigger('click')
    await flushPromises()

    expect(mockUpdateNamespace).not.toHaveBeenCalled()
  })

  it('handles update error gracefully', async () => {
    mockUpdateNamespace.mockRejectedValue({
      response: { data: { error: 'Update failed' } }
    })

    const wrapper = mount(EditNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })

    const input = wrapper.find('.input-mock')
    await input.setValue('updated-namespace')

    const saveButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Save Changes'))
    await saveButton?.trigger('click')
    await flushPromises()

    // Should not emit success on error
    expect(wrapper.emitted('success')).toBeFalsy()
  })

  it('shows Saving... text while loading', async () => {
    mockUpdateNamespace.mockImplementation(() => new Promise(() => {})) // Never resolves

    const wrapper = mount(EditNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })

    const input = wrapper.find('.input-mock')
    await input.setValue('updated-namespace')

    const saveButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Save Changes'))
    await saveButton?.trigger('click')

    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Saving...')
  })
})
