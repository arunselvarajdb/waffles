import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseToggle from './BaseToggle.vue'

describe('BaseToggle', () => {
  it('renders as a button', () => {
    const wrapper = mount(BaseToggle)
    expect(wrapper.element.tagName).toBe('BUTTON')
  })

  it('has switch role', () => {
    const wrapper = mount(BaseToggle)
    expect(wrapper.attributes('role')).toBe('switch')
  })

  it('has aria-checked attribute matching modelValue', () => {
    const wrapperOff = mount(BaseToggle, { props: { modelValue: false } })
    expect(wrapperOff.attributes('aria-checked')).toBe('false')

    const wrapperOn = mount(BaseToggle, { props: { modelValue: true } })
    expect(wrapperOn.attributes('aria-checked')).toBe('true')
  })

  it('is off by default', () => {
    const wrapper = mount(BaseToggle)
    expect(wrapper.classes()).toContain('bg-gray-300')
  })

  it('shows green background when on', () => {
    const wrapper = mount(BaseToggle, {
      props: { modelValue: true },
    })
    expect(wrapper.classes()).toContain('bg-green-600')
  })

  it('emits update:modelValue on click', async () => {
    const wrapper = mount(BaseToggle, {
      props: { modelValue: false },
    })
    await wrapper.trigger('click')
    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([true])
  })

  it('emits change event on click', async () => {
    const wrapper = mount(BaseToggle, {
      props: { modelValue: false },
    })
    await wrapper.trigger('click')
    expect(wrapper.emitted('change')).toBeTruthy()
    expect(wrapper.emitted('change')![0]).toEqual([true])
  })

  it('toggles from true to false', async () => {
    const wrapper = mount(BaseToggle, {
      props: { modelValue: true },
    })
    await wrapper.trigger('click')
    expect(wrapper.emitted('update:modelValue')![0]).toEqual([false])
  })

  it('is disabled when disabled prop is true', () => {
    const wrapper = mount(BaseToggle, {
      props: { disabled: true },
    })
    expect(wrapper.attributes('disabled')).toBeDefined()
    expect(wrapper.classes()).toContain('cursor-not-allowed')
    expect(wrapper.classes()).toContain('opacity-50')
  })

  it('does not emit events when disabled', async () => {
    const wrapper = mount(BaseToggle, {
      props: { disabled: true, modelValue: false },
    })
    await wrapper.trigger('click')
    expect(wrapper.emitted('update:modelValue')).toBeFalsy()
    expect(wrapper.emitted('change')).toBeFalsy()
  })

  it('has cursor-pointer when not disabled', () => {
    const wrapper = mount(BaseToggle, {
      props: { disabled: false },
    })
    expect(wrapper.classes()).toContain('cursor-pointer')
  })

  it('switch knob moves when toggled', () => {
    const wrapperOff = mount(BaseToggle, { props: { modelValue: false } })
    const switchOff = wrapperOff.find('span')
    expect(switchOff.classes()).toContain('translate-x-1')

    const wrapperOn = mount(BaseToggle, { props: { modelValue: true } })
    const switchOn = wrapperOn.find('span')
    expect(switchOn.classes()).toContain('translate-x-6')
  })
})
