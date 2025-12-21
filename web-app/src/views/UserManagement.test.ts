import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import UserManagement from './UserManagement.vue'

// Mock NavBar component
vi.mock('@/components/layout/NavBar.vue', () => ({
  default: {
    template: '<nav class="navbar-mock">NavBar</nav>'
  }
}))

describe('UserManagement', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders the NavBar component', () => {
    const wrapper = mount(UserManagement)
    expect(wrapper.find('.navbar-mock').exists()).toBe(true)
  })

  it('displays User Management heading', () => {
    const wrapper = mount(UserManagement)
    expect(wrapper.text()).toContain('User Management')
  })

  it('displays description text', () => {
    const wrapper = mount(UserManagement)
    expect(wrapper.text()).toContain('Manage users and assign server permissions')
  })

  it('displays coming soon message', () => {
    const wrapper = mount(UserManagement)
    expect(wrapper.text()).toContain('User Management Coming Soon')
  })

  it('displays feature description', () => {
    const wrapper = mount(UserManagement)
    expect(wrapper.text()).toContain('manage users, assign roles')
  })

  it('has correct page structure', () => {
    const wrapper = mount(UserManagement)
    expect(wrapper.find('.min-h-screen').exists()).toBe(true)
    expect(wrapper.find('main').exists()).toBe(true)
  })

  it('displays placeholder icon', () => {
    const wrapper = mount(UserManagement)
    expect(wrapper.find('svg').exists()).toBe(true)
  })

  it('has max-width container', () => {
    const wrapper = mount(UserManagement)
    expect(wrapper.find('.max-w-7xl').exists()).toBe(true)
  })
})
