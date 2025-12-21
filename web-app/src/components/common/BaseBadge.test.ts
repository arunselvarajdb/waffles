import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseBadge from './BaseBadge.vue'

describe('BaseBadge', () => {
  it('renders slot content', () => {
    const wrapper = mount(BaseBadge, {
      slots: { default: 'Active' },
    })
    expect(wrapper.text()).toBe('Active')
  })

  it('applies info variant by default', () => {
    const wrapper = mount(BaseBadge)
    expect(wrapper.classes()).toContain('bg-blue-100')
    expect(wrapper.classes()).toContain('text-blue-800')
  })

  it('applies success variant', () => {
    const wrapper = mount(BaseBadge, {
      props: { variant: 'success' },
    })
    expect(wrapper.classes()).toContain('bg-green-100')
    expect(wrapper.classes()).toContain('text-green-800')
  })

  it('applies warning variant', () => {
    const wrapper = mount(BaseBadge, {
      props: { variant: 'warning' },
    })
    expect(wrapper.classes()).toContain('bg-yellow-100')
    expect(wrapper.classes()).toContain('text-yellow-800')
  })

  it('applies danger variant', () => {
    const wrapper = mount(BaseBadge, {
      props: { variant: 'danger' },
    })
    expect(wrapper.classes()).toContain('bg-red-100')
    expect(wrapper.classes()).toContain('text-red-800')
  })

  it('applies secondary variant', () => {
    const wrapper = mount(BaseBadge, {
      props: { variant: 'secondary' },
    })
    expect(wrapper.classes()).toContain('bg-gray-100')
    expect(wrapper.classes()).toContain('text-gray-800')
  })

  it('applies md size by default', () => {
    const wrapper = mount(BaseBadge)
    expect(wrapper.classes()).toContain('px-2.5')
    expect(wrapper.classes()).toContain('text-sm')
  })

  it('applies sm size', () => {
    const wrapper = mount(BaseBadge, {
      props: { size: 'sm' },
    })
    expect(wrapper.classes()).toContain('px-2')
    expect(wrapper.classes()).toContain('text-xs')
  })

  it('applies lg size', () => {
    const wrapper = mount(BaseBadge, {
      props: { size: 'lg' },
    })
    expect(wrapper.classes()).toContain('px-3')
    expect(wrapper.classes()).toContain('text-base')
  })

  it('renders as span element', () => {
    const wrapper = mount(BaseBadge)
    expect(wrapper.element.tagName).toBe('SPAN')
  })

  it('has rounded-full class', () => {
    const wrapper = mount(BaseBadge)
    expect(wrapper.classes()).toContain('rounded-full')
  })
})
