import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import DeleteNamespaceModal from './DeleteNamespaceModal.vue'

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
    props: ['variant', 'disabled'],
    emits: ['click']
  }
}))

// Mock namespaces store
const mockDeleteNamespace = vi.fn()
vi.mock('@/stores/namespaces', () => ({
  useNamespacesStore: vi.fn(() => ({
    deleteNamespace: mockDeleteNamespace
  }))
}))

describe('DeleteNamespaceModal', () => {
  const mockNamespace = {
    id: 'ns-123',
    name: 'test-namespace',
    description: 'Test namespace'
  }

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('does not render when modelValue is false', () => {
    const wrapper = mount(DeleteNamespaceModal, {
      props: { modelValue: false, namespace: mockNamespace }
    })
    expect(wrapper.find('.modal-mock').exists()).toBe(false)
  })

  it('renders when modelValue is true', () => {
    const wrapper = mount(DeleteNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    expect(wrapper.find('.modal-mock').exists()).toBe(true)
  })

  it('displays Delete Namespace title', () => {
    const wrapper = mount(DeleteNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    expect(wrapper.text()).toContain('Delete Namespace')
  })

  it('displays the namespace name in confirmation message', () => {
    const wrapper = mount(DeleteNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    expect(wrapper.text()).toContain('test-namespace')
  })

  it('displays warning about deletion being permanent', () => {
    const wrapper = mount(DeleteNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    expect(wrapper.text()).toContain('cannot be undone')
  })

  it('has Cancel button', () => {
    const wrapper = mount(DeleteNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    expect(wrapper.text()).toContain('Cancel')
  })

  it('has Delete Namespace button', () => {
    const wrapper = mount(DeleteNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    expect(wrapper.text()).toContain('Delete Namespace')
  })

  it('emits update:modelValue false on cancel', async () => {
    const wrapper = mount(DeleteNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })

    const cancelButton = wrapper.findAll('.button-mock').find(b => b.text() === 'Cancel')
    await cancelButton?.trigger('click')

    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([false])
  })

  it('calls deleteNamespace on delete click', async () => {
    mockDeleteNamespace.mockResolvedValue({})

    const wrapper = mount(DeleteNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })

    const deleteButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Delete Namespace'))
    await deleteButton?.trigger('click')
    await flushPromises()

    expect(mockDeleteNamespace).toHaveBeenCalledWith('ns-123')
  })

  it('emits success on successful delete', async () => {
    mockDeleteNamespace.mockResolvedValue({})

    const wrapper = mount(DeleteNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })

    const deleteButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Delete Namespace'))
    await deleteButton?.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('success')).toBeTruthy()
  })

  it('emits update:modelValue false on successful delete', async () => {
    mockDeleteNamespace.mockResolvedValue({})

    const wrapper = mount(DeleteNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })

    const deleteButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Delete Namespace'))
    await deleteButton?.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([false])
  })

  it('does not delete when namespace is null', async () => {
    const wrapper = mount(DeleteNamespaceModal, {
      props: { modelValue: true, namespace: null }
    })

    const deleteButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Delete Namespace'))
    await deleteButton?.trigger('click')
    await flushPromises()

    expect(mockDeleteNamespace).not.toHaveBeenCalled()
  })

  it('shows error message on delete failure', async () => {
    mockDeleteNamespace.mockRejectedValue({
      response: { data: { error: 'Cannot delete namespace' } }
    })

    const wrapper = mount(DeleteNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })

    const deleteButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Delete Namespace'))
    await deleteButton?.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('Cannot delete namespace')
  })

  it('does not emit success on delete error', async () => {
    mockDeleteNamespace.mockRejectedValue(new Error('Delete failed'))

    const wrapper = mount(DeleteNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })

    const deleteButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Delete Namespace'))
    await deleteButton?.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('success')).toBeFalsy()
  })

  it('shows Deleting... text while loading', async () => {
    mockDeleteNamespace.mockImplementation(() => new Promise(() => {})) // Never resolves

    const wrapper = mount(DeleteNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })

    const deleteButton = wrapper.findAll('.button-mock').find(b => b.text().includes('Delete Namespace'))
    await deleteButton?.trigger('click')

    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Deleting...')
  })

  it('displays warning icon', () => {
    const wrapper = mount(DeleteNamespaceModal, {
      props: { modelValue: true, namespace: mockNamespace }
    })
    expect(wrapper.find('svg').exists()).toBe(true)
  })
})
