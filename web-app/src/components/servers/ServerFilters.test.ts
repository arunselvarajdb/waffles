import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ServerFilters from './ServerFilters.vue'

// Mock BaseInput component
vi.mock('@/components/common/BaseInput.vue', () => ({
  default: {
    template: '<input class="input-mock" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
    props: ['modelValue', 'placeholder'],
    emits: ['update:modelValue']
  }
}))

// Mock the servers store - factory pattern for per-test overrides
const mockSetFilters = vi.fn()
const createMockServersStore = (overrides = {}) => ({
  filters: {
    search: '',
    status: 'all',
    health: 'all'
  },
  setFilters: mockSetFilters,
  ...overrides
})

vi.mock('@/stores/servers', () => ({
  useServersStore: vi.fn(() => createMockServersStore())
}))

import { useServersStore } from '@/stores/servers'

describe('ServerFilters', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    // Reset to default mock
    vi.mocked(useServersStore).mockReturnValue(createMockServersStore())
  })

  it('renders filter container', () => {
    const wrapper = mount(ServerFilters)
    expect(wrapper.find('.bg-white.rounded-lg').exists()).toBe(true)
  })

  it('renders search input', () => {
    const wrapper = mount(ServerFilters)
    expect(wrapper.find('.input-mock').exists()).toBe(true)
  })

  it('renders status filter dropdown', () => {
    const wrapper = mount(ServerFilters)
    const selects = wrapper.findAll('select')
    expect(selects.length).toBe(2)
  })

  it('has status filter options', () => {
    const wrapper = mount(ServerFilters)
    expect(wrapper.text()).toContain('All Statuses')
    expect(wrapper.text()).toContain('Active Only')
    expect(wrapper.text()).toContain('Inactive Only')
  })

  it('has health filter options', () => {
    const wrapper = mount(ServerFilters)
    expect(wrapper.text()).toContain('All Health')
    expect(wrapper.text()).toContain('Healthy Only')
    expect(wrapper.text()).toContain('Degraded Only')
    expect(wrapper.text()).toContain('Unhealthy Only')
  })

  it('calls setFilters on search input change', async () => {
    const wrapper = mount(ServerFilters)
    const input = wrapper.find('.input-mock')
    await input.setValue('test search')
    await input.trigger('input')

    expect(mockSetFilters).toHaveBeenCalled()
  })

  it('calls setFilters on status filter change', async () => {
    const wrapper = mount(ServerFilters)
    const selects = wrapper.findAll('select')
    await selects[0].setValue('active')
    await selects[0].trigger('change')

    expect(mockSetFilters).toHaveBeenCalled()
  })

  it('calls setFilters on health filter change', async () => {
    const wrapper = mount(ServerFilters)
    const selects = wrapper.findAll('select')
    await selects[1].setValue('healthy')
    await selects[1].trigger('change')

    expect(mockSetFilters).toHaveBeenCalled()
  })

  it('initializes with store filter values', async () => {
    vi.mocked(useServersStore).mockReturnValue(createMockServersStore({
      filters: {
        search: 'initial search',
        status: 'active',
        health: 'healthy'
      }
    }))

    const wrapper = mount(ServerFilters)
    await flushPromises()

    const selects = wrapper.findAll('select')
    expect(selects[0].element.value).toBe('active')
    expect(selects[1].element.value).toBe('healthy')
  })

  it('has responsive grid layout', () => {
    const wrapper = mount(ServerFilters)
    expect(wrapper.find('.grid-cols-1').exists()).toBe(true)
    expect(wrapper.find('.md\\:grid-cols-4').exists()).toBe(true)
  })

  it('search input spans 2 columns on md screens', () => {
    const wrapper = mount(ServerFilters)
    expect(wrapper.find('.md\\:col-span-2').exists()).toBe(true)
  })

  it('has proper styling on select elements', () => {
    const wrapper = mount(ServerFilters)
    const select = wrapper.find('select')
    expect(select.classes()).toContain('rounded-lg')
    expect(select.classes()).toContain('border')
  })

  it('passes all filters to setFilters', async () => {
    const wrapper = mount(ServerFilters)
    const selects = wrapper.findAll('select')

    await selects[0].setValue('inactive')
    await selects[0].trigger('change')

    expect(mockSetFilters).toHaveBeenCalledWith({
      search: '',
      status: 'inactive',
      health: 'all'
    })
  })
})
