import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import NamespacesDashboard from './NamespacesDashboard.vue'

// Mock child components
vi.mock('@/components/layout/NavBar.vue', () => ({
  default: { template: '<div class="navbar-mock">NavBar</div>' }
}))
vi.mock('@/components/common/BaseButton.vue', () => ({
  default: { template: '<button class="button-mock"><slot /></button>', props: ['variant'] }
}))
vi.mock('@/components/common/BaseInput.vue', () => ({
  default: { template: '<input class="input-mock" />', props: ['modelValue', 'placeholder'], emits: ['update:modelValue', 'input'] }
}))
vi.mock('@/components/common/BasePagination.vue', () => ({
  default: { template: '<div class="pagination-mock">Pagination</div>', props: ['currentPage', 'total', 'perPage'], emits: ['page-change'] }
}))
vi.mock('@/components/namespaces/NamespaceTable.vue', () => ({
  default: { template: '<div class="table-mock">NamespaceTable</div>', props: ['namespaces'], emits: ['view-namespace', 'edit-namespace', 'delete-namespace'] }
}))
vi.mock('@/components/namespaces/AddNamespaceModal.vue', () => ({
  default: { template: '<div class="add-modal-mock" v-if="modelValue">AddModal</div>', props: ['modelValue'], emits: ['update:modelValue', 'success'] }
}))
vi.mock('@/components/namespaces/EditNamespaceModal.vue', () => ({
  default: { template: '<div class="edit-modal-mock" v-if="modelValue">EditModal</div>', props: ['modelValue', 'namespace'], emits: ['update:modelValue', 'success'] }
}))
vi.mock('@/components/namespaces/DeleteNamespaceModal.vue', () => ({
  default: { template: '<div class="delete-modal-mock" v-if="modelValue">DeleteModal</div>', props: ['modelValue', 'namespace'], emits: ['update:modelValue', 'success'] }
}))
vi.mock('@/components/namespaces/NamespaceDetailsModal.vue', () => ({
  default: { template: '<div class="details-modal-mock" v-if="modelValue">DetailsModal</div>', props: ['modelValue', 'namespace'], emits: ['update:modelValue'] }
}))

// Mock namespaces store - factory pattern for per-test overrides
const createMockNamespacesStore = (overrides = {}) => ({
  namespaces: [],
  filteredNamespaces: [],
  namespaceCount: 0,
  pagination: { page: 1, limit: 10, total: 0 },
  loading: false,
  error: null,
  fetchNamespaces: vi.fn(),
  setFilters: vi.fn(),
  setPage: vi.fn(),
  ...overrides
})

vi.mock('@/stores/namespaces', () => ({
  useNamespacesStore: vi.fn(() => createMockNamespacesStore())
}))

import { useNamespacesStore } from '@/stores/namespaces'

describe('NamespacesDashboard', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    // Reset to default mock
    vi.mocked(useNamespacesStore).mockReturnValue(createMockNamespacesStore())
  })

  it('renders the dashboard', () => {
    const wrapper = mount(NamespacesDashboard)
    expect(wrapper.find('.min-h-screen').exists()).toBe(true)
  })

  it('displays page header', () => {
    const wrapper = mount(NamespacesDashboard)
    expect(wrapper.text()).toContain('Namespace Management')
    expect(wrapper.text()).toContain('Organize servers into namespaces')
  })

  it('renders NavBar component', () => {
    const wrapper = mount(NamespacesDashboard)
    expect(wrapper.find('.navbar-mock').exists()).toBe(true)
  })

  it('renders Create Namespace button', () => {
    const wrapper = mount(NamespacesDashboard)
    expect(wrapper.text()).toContain('+ Create Namespace')
  })

  it('shows namespace count in statistics', () => {
    vi.mocked(useNamespacesStore).mockReturnValue(createMockNamespacesStore({
      namespaceCount: 5,
      pagination: { page: 1, limit: 10, total: 5 }
    }))

    const wrapper = mount(NamespacesDashboard)
    expect(wrapper.text()).toContain('5')
    expect(wrapper.text()).toContain('Total Namespaces')
  })

  it('shows loading state', () => {
    vi.mocked(useNamespacesStore).mockReturnValue(createMockNamespacesStore({
      loading: true
    }))

    const wrapper = mount(NamespacesDashboard)
    expect(wrapper.text()).toContain('Loading namespaces...')
  })

  it('shows error state with retry button', () => {
    vi.mocked(useNamespacesStore).mockReturnValue(createMockNamespacesStore({
      error: 'Failed to load'
    }))

    const wrapper = mount(NamespacesDashboard)
    expect(wrapper.text()).toContain('Failed to load')
    expect(wrapper.text()).toContain('Retry')
  })

  it('renders NamespaceTable when not loading and no error', () => {
    const wrapper = mount(NamespacesDashboard)
    expect(wrapper.find('.table-mock').exists()).toBe(true)
  })

  it('fetches namespaces on mount', async () => {
    const mockFetch = vi.fn()
    vi.mocked(useNamespacesStore).mockReturnValue(createMockNamespacesStore({
      fetchNamespaces: mockFetch
    }))

    mount(NamespacesDashboard)
    await flushPromises()

    expect(mockFetch).toHaveBeenCalled()
  })

  it('opens add modal on button click', async () => {
    const wrapper = mount(NamespacesDashboard)

    expect(wrapper.find('.add-modal-mock').exists()).toBe(false)

    await wrapper.find('.button-mock').trigger('click')

    expect(wrapper.find('.add-modal-mock').exists()).toBe(true)
  })

  it('shows pagination when namespaces exist', () => {
    vi.mocked(useNamespacesStore).mockReturnValue(createMockNamespacesStore({
      namespaces: [{ id: '1' }],
      filteredNamespaces: [{ id: '1' }],
      namespaceCount: 1,
      pagination: { page: 1, limit: 10, total: 1 }
    }))

    const wrapper = mount(NamespacesDashboard)
    expect(wrapper.find('.pagination-mock').exists()).toBe(true)
  })

  it('hides pagination when no namespaces', () => {
    const wrapper = mount(NamespacesDashboard)
    expect(wrapper.find('.pagination-mock').exists()).toBe(false)
  })

  it('has search input', () => {
    const wrapper = mount(NamespacesDashboard)
    expect(wrapper.find('.input-mock').exists()).toBe(true)
  })

  it('has responsive layout', () => {
    const wrapper = mount(NamespacesDashboard)
    expect(wrapper.find('.max-w-7xl').exists()).toBe(true)
  })
})
