import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseModal from './BaseModal.vue'

describe('BaseModal', () => {
  beforeEach(() => {
    // Reset body overflow before each test
    document.body.style.overflow = ''
  })

  afterEach(() => {
    document.body.style.overflow = ''
  })

  it('does not render when modelValue is false', () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: false },
      global: {
        stubs: { Teleport: true }
      }
    })
    expect(wrapper.find('.fixed').exists()).toBe(false)
  })

  it('renders when modelValue is true', () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true },
      global: {
        stubs: { Teleport: true }
      }
    })
    expect(wrapper.find('.fixed').exists()).toBe(true)
  })

  it('displays title prop', () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true, title: 'Test Modal Title' },
      global: {
        stubs: { Teleport: true }
      }
    })
    expect(wrapper.text()).toContain('Test Modal Title')
  })

  it('renders header slot content', () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true },
      slots: { header: 'Custom Header' },
      global: {
        stubs: { Teleport: true }
      }
    })
    expect(wrapper.text()).toContain('Custom Header')
  })

  it('renders default slot content', () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true },
      slots: { default: 'Modal body content' },
      global: {
        stubs: { Teleport: true }
      }
    })
    expect(wrapper.text()).toContain('Modal body content')
  })

  it('renders footer slot content', () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true },
      slots: { footer: '<button>Save</button>' },
      global: {
        stubs: { Teleport: true }
      }
    })
    expect(wrapper.text()).toContain('Save')
  })

  it('applies sm size class', () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true, size: 'sm' },
      global: {
        stubs: { Teleport: true }
      }
    })
    expect(wrapper.find('.max-w-sm').exists()).toBe(true)
  })

  it('applies md size class by default', () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true },
      global: {
        stubs: { Teleport: true }
      }
    })
    expect(wrapper.find('.max-w-md').exists()).toBe(true)
  })

  it('applies lg size class', () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true, size: 'lg' },
      global: {
        stubs: { Teleport: true }
      }
    })
    expect(wrapper.find('.max-w-lg').exists()).toBe(true)
  })

  it('applies xl size class', () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true, size: 'xl' },
      global: {
        stubs: { Teleport: true }
      }
    })
    expect(wrapper.find('.max-w-2xl').exists()).toBe(true)
  })

  it('shows close button when closable is true', () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true, title: 'Test', closable: true },
      global: {
        stubs: { Teleport: true }
      }
    })
    expect(wrapper.find('button svg').exists()).toBe(true)
  })

  it('hides close button when closable is false', () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true, title: 'Test', closable: false },
      global: {
        stubs: { Teleport: true }
      }
    })
    // Header with title but no close button
    const header = wrapper.find('.border-b')
    expect(header.find('button').exists()).toBe(false)
  })

  it('emits update:modelValue and close on close button click', async () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true, title: 'Test' },
      global: {
        stubs: { Teleport: true }
      }
    })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([false])
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('closes on backdrop click when closeOnBackdrop is true', async () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true, closeOnBackdrop: true },
      global: {
        stubs: { Teleport: true }
      }
    })
    // Click on the outer container
    await wrapper.find('.fixed.inset-0.z-50').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('close')).toBeTruthy()
  })

  it('does not close on backdrop click when closeOnBackdrop is false', async () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true, closeOnBackdrop: false },
      global: {
        stubs: { Teleport: true }
      }
    })
    await wrapper.find('.fixed.inset-0.z-50').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toBeFalsy()
  })

  it('does not close on backdrop click when closable is false', async () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true, closable: false, closeOnBackdrop: true },
      global: {
        stubs: { Teleport: true }
      }
    })
    await wrapper.find('.fixed.inset-0.z-50').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toBeFalsy()
  })

  it('does not close when clicking modal content', async () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true },
      slots: { default: 'Content' },
      global: {
        stubs: { Teleport: true }
      }
    })
    await wrapper.find('.bg-white.rounded-lg').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toBeFalsy()
  })

  it('has backdrop with bg-black bg-opacity-50', () => {
    const wrapper = mount(BaseModal, {
      props: { modelValue: true },
      global: {
        stubs: { Teleport: true }
      }
    })
    expect(wrapper.find('.bg-black.bg-opacity-50').exists()).toBe(true)
  })
})
