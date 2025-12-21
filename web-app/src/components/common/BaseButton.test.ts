import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseButton from './BaseButton.vue'

describe('BaseButton', () => {
  it('renders slot content', () => {
    const wrapper = mount(BaseButton, {
      slots: {
        default: 'Click me',
      },
    })
    expect(wrapper.text()).toContain('Click me')
  })

  it('applies primary variant by default', () => {
    const wrapper = mount(BaseButton)
    expect(wrapper.classes()).toContain('bg-blue-600')
  })

  it('applies danger variant when specified', () => {
    const wrapper = mount(BaseButton, {
      props: {
        variant: 'danger',
      },
    })
    expect(wrapper.classes()).toContain('bg-red-600')
  })

  it('applies success variant when specified', () => {
    const wrapper = mount(BaseButton, {
      props: {
        variant: 'success',
      },
    })
    expect(wrapper.classes()).toContain('bg-green-600')
  })

  it('applies secondary variant when specified', () => {
    const wrapper = mount(BaseButton, {
      props: {
        variant: 'secondary',
      },
    })
    expect(wrapper.classes()).toContain('bg-gray-200')
  })

  it('is disabled when disabled prop is true', () => {
    const wrapper = mount(BaseButton, {
      props: {
        disabled: true,
      },
    })
    expect(wrapper.attributes('disabled')).toBeDefined()
    expect(wrapper.classes()).toContain('cursor-not-allowed')
  })

  it('is disabled when loading prop is true', () => {
    const wrapper = mount(BaseButton, {
      props: {
        loading: true,
      },
    })
    expect(wrapper.attributes('disabled')).toBeDefined()
    expect(wrapper.classes()).toContain('cursor-not-allowed')
  })

  it('shows loading spinner when loading', () => {
    const wrapper = mount(BaseButton, {
      props: {
        loading: true,
      },
    })
    expect(wrapper.find('svg').exists()).toBe(true)
  })

  it('does not show loading spinner when not loading', () => {
    const wrapper = mount(BaseButton, {
      props: {
        loading: false,
      },
    })
    expect(wrapper.find('svg').exists()).toBe(false)
  })

  it('emits click event when clicked', async () => {
    const wrapper = mount(BaseButton)
    await wrapper.trigger('click')
    expect(wrapper.emitted('click')).toBeTruthy()
  })

  it('applies correct size classes for sm size', () => {
    const wrapper = mount(BaseButton, {
      props: {
        size: 'sm',
      },
    })
    expect(wrapper.classes()).toContain('px-3')
    expect(wrapper.classes()).toContain('py-1.5')
  })

  it('applies correct size classes for lg size', () => {
    const wrapper = mount(BaseButton, {
      props: {
        size: 'lg',
      },
    })
    expect(wrapper.classes()).toContain('px-6')
    expect(wrapper.classes()).toContain('py-3')
  })

  it('has button type by default', () => {
    const wrapper = mount(BaseButton)
    expect(wrapper.attributes('type')).toBe('button')
  })

  it('can have submit type', () => {
    const wrapper = mount(BaseButton, {
      props: {
        type: 'submit',
      },
    })
    expect(wrapper.attributes('type')).toBe('submit')
  })
})
