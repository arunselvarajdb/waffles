import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import BaseInput from './BaseInput.vue'

describe('BaseInput', () => {
  it('renders input element', () => {
    const wrapper = mount(BaseInput)
    expect(wrapper.find('input').exists()).toBe(true)
  })

  it('renders label when provided', () => {
    const wrapper = mount(BaseInput, {
      props: { label: 'Email Address' },
    })
    expect(wrapper.find('label').text()).toContain('Email Address')
  })

  it('does not render label when not provided', () => {
    const wrapper = mount(BaseInput)
    expect(wrapper.find('label').exists()).toBe(false)
  })

  it('shows required indicator when required', () => {
    const wrapper = mount(BaseInput, {
      props: { label: 'Name', required: true },
    })
    expect(wrapper.find('label').text()).toContain('*')
  })

  it('binds modelValue correctly', () => {
    const wrapper = mount(BaseInput, {
      props: { modelValue: 'test value' },
    })
    expect(wrapper.find('input').element.value).toBe('test value')
  })

  it('emits update:modelValue on input', async () => {
    const wrapper = mount(BaseInput)
    await wrapper.find('input').setValue('new value')
    expect(wrapper.emitted('update:modelValue')).toBeTruthy()
    expect(wrapper.emitted('update:modelValue')![0]).toEqual(['new value'])
  })

  it('emits blur event', async () => {
    const wrapper = mount(BaseInput)
    await wrapper.find('input').trigger('blur')
    expect(wrapper.emitted('blur')).toBeTruthy()
  })

  it('emits focus event', async () => {
    const wrapper = mount(BaseInput)
    await wrapper.find('input').trigger('focus')
    expect(wrapper.emitted('focus')).toBeTruthy()
  })

  it('applies placeholder', () => {
    const wrapper = mount(BaseInput, {
      props: { placeholder: 'Enter email' },
    })
    expect(wrapper.find('input').attributes('placeholder')).toBe('Enter email')
  })

  it('sets input type', () => {
    const wrapper = mount(BaseInput, {
      props: { type: 'password' },
    })
    expect(wrapper.find('input').attributes('type')).toBe('password')
  })

  it('disables input when disabled prop is true', () => {
    const wrapper = mount(BaseInput, {
      props: { disabled: true },
    })
    expect(wrapper.find('input').attributes('disabled')).toBeDefined()
    expect(wrapper.find('input').classes()).toContain('cursor-not-allowed')
  })

  it('shows error message when error prop is set', () => {
    const wrapper = mount(BaseInput, {
      props: { error: 'Invalid email format' },
    })
    expect(wrapper.text()).toContain('Invalid email format')
    expect(wrapper.find('.text-red-600').exists()).toBe(true)
  })

  it('shows hint when provided and no error', () => {
    const wrapper = mount(BaseInput, {
      props: { hint: 'Enter your work email' },
    })
    expect(wrapper.text()).toContain('Enter your work email')
    expect(wrapper.find('.text-gray-500').exists()).toBe(true)
  })

  it('shows error instead of hint when both provided', () => {
    const wrapper = mount(BaseInput, {
      props: { error: 'Error message', hint: 'Hint message' },
    })
    expect(wrapper.text()).toContain('Error message')
    expect(wrapper.text()).not.toContain('Hint message')
  })

  it('applies error styling when error is present', () => {
    const wrapper = mount(BaseInput, {
      props: { error: 'Some error' },
    })
    expect(wrapper.find('input').classes()).toContain('border-red-300')
  })

  it('applies normal styling when no error', () => {
    const wrapper = mount(BaseInput)
    expect(wrapper.find('input').classes()).toContain('border-gray-300')
  })

  it('generates unique id when not provided', () => {
    const wrapper = mount(BaseInput, {
      props: { label: 'Test' },
    })
    const id = wrapper.find('input').attributes('id')
    expect(id).toMatch(/^input-/)
  })

  it('uses provided id', () => {
    const wrapper = mount(BaseInput, {
      props: { id: 'custom-id', label: 'Test' },
    })
    expect(wrapper.find('input').attributes('id')).toBe('custom-id')
    expect(wrapper.find('label').attributes('for')).toBe('custom-id')
  })
})
