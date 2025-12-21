import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import DeleteServerModal from './DeleteServerModal.vue'

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
    template: '<button class="button-mock" :disabled="loading"><slot /></button>',
    props: ['variant', 'loading']
  }
}))

// Mock servers store
const mockDeleteServer = vi.fn()
vi.mock('@/stores/servers', () => ({
  useServersStore: vi.fn(() => ({
    deleteServer: mockDeleteServer
  }))
}))

describe('DeleteServerModal', () => {
  const mockServer = { id: '1', name: 'Test Server' }

  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('does not render when modelValue is false', () => {
    const wrapper = mount(DeleteServerModal, {
      props: { modelValue: false, server: mockServer }
    })
    expect(wrapper.find('.modal-mock').exists()).toBe(false)
  })

  it('renders when modelValue is true', () => {
    const wrapper = mount(DeleteServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.find('.modal-mock').exists()).toBe(true)
  })

  it('displays Delete Server title', () => {
    const wrapper = mount(DeleteServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Delete Server')
  })

  it('displays server name in confirmation', () => {
    const wrapper = mount(DeleteServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Test Server')
  })

  it('shows warning about irreversible action', () => {
    const wrapper = mount(DeleteServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('This action cannot be undone')
  })

  it('has Cancel button', () => {
    const wrapper = mount(DeleteServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Cancel')
  })

  it('has Delete Server button', () => {
    const wrapper = mount(DeleteServerModal, {
      props: { modelValue: true, server: mockServer }
    })
    expect(wrapper.text()).toContain('Delete Server')
  })

  it('emits update:modelValue false on cancel', async () => {
    const wrapper = mount(DeleteServerModal, {
      props: { modelValue: true, server: mockServer }
    })

    const cancelButton = wrapper.findAll('.button-mock').find(b => b.text() === 'Cancel')
    await cancelButton?.trigger('click')

    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([false])
  })

  it('calls deleteServer on confirm', async () => {
    mockDeleteServer.mockResolvedValue(undefined)

    const wrapper = mount(DeleteServerModal, {
      props: { modelValue: true, server: mockServer }
    })

    const deleteButton = wrapper.findAll('.button-mock').find(b => b.text() === 'Delete Server')
    await deleteButton?.trigger('click')
    await flushPromises()

    expect(mockDeleteServer).toHaveBeenCalledWith('1')
  })

  it('emits success after successful delete', async () => {
    mockDeleteServer.mockResolvedValue(undefined)

    const wrapper = mount(DeleteServerModal, {
      props: { modelValue: true, server: mockServer }
    })

    const deleteButton = wrapper.findAll('.button-mock').find(b => b.text() === 'Delete Server')
    await deleteButton?.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('success')).toBeTruthy()
  })

  it('closes modal after successful delete', async () => {
    mockDeleteServer.mockResolvedValue(undefined)

    const wrapper = mount(DeleteServerModal, {
      props: { modelValue: true, server: mockServer }
    })

    const deleteButton = wrapper.findAll('.button-mock').find(b => b.text() === 'Delete Server')
    await deleteButton?.trigger('click')
    await flushPromises()

    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
  })

  it('does not delete when server is null', async () => {
    const wrapper = mount(DeleteServerModal, {
      props: { modelValue: true, server: null }
    })

    const deleteButton = wrapper.findAll('.button-mock').find(b => b.text() === 'Delete Server')
    await deleteButton?.trigger('click')
    await flushPromises()

    expect(mockDeleteServer).not.toHaveBeenCalled()
  })

  it('handles delete error gracefully', async () => {
    mockDeleteServer.mockRejectedValue(new Error('Delete failed'))

    const wrapper = mount(DeleteServerModal, {
      props: { modelValue: true, server: mockServer }
    })

    const deleteButton = wrapper.findAll('.button-mock').find(b => b.text() === 'Delete Server')
    await deleteButton?.trigger('click')
    await flushPromises()

    // Should not emit success on error
    expect(wrapper.emitted('success')).toBeFalsy()
  })

  it('syncs isOpen with modelValue prop', async () => {
    const wrapper = mount(DeleteServerModal, {
      props: { modelValue: false, server: mockServer }
    })

    expect(wrapper.find('.modal-mock').exists()).toBe(false)

    await wrapper.setProps({ modelValue: true })
    expect(wrapper.find('.modal-mock').exists()).toBe(true)
  })
})
