import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ServiceAccounts from './ServiceAccounts.vue'

// Mock child components
vi.mock('@/components/layout/NavBar.vue', () => ({
  default: { template: '<div class="navbar-mock">NavBar</div>' }
}))
vi.mock('@/components/common/BaseBadge.vue', () => ({
  default: {
    template: '<span class="badge-mock" :class="variant"><slot /></span>',
    props: ['variant']
  }
}))
vi.mock('@/components/common/BaseButton.vue', () => ({
  default: {
    template: '<button class="button-mock"><slot /></button>',
    props: ['variant']
  }
}))
vi.mock('@/components/common/BaseInput.vue', () => ({
  default: {
    template: '<input class="input-mock" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
    props: ['modelValue', 'placeholder'],
    emits: ['update:modelValue']
  }
}))
vi.mock('@/components/apikeys/CreateApiKeyModal.vue', () => ({
  default: {
    template: '<div class="create-modal-mock" v-if="modelValue">CreateApiKeyModal</div>',
    props: ['modelValue'],
    emits: ['update:modelValue', 'submit']
  }
}))
vi.mock('@/components/apikeys/ApiKeyCreatedModal.vue', () => ({
  default: {
    template: '<div class="created-modal-mock" v-if="modelValue">ApiKeyCreatedModal</div>',
    props: ['modelValue', 'apiKey'],
    emits: ['update:modelValue']
  }
}))

// Mock API service
const mockGet = vi.fn()
const mockPost = vi.fn()
const mockDelete = vi.fn()

vi.mock('@/services/api', () => ({
  default: {
    get: (...args: unknown[]) => mockGet(...args),
    post: (...args: unknown[]) => mockPost(...args),
    delete: (...args: unknown[]) => mockDelete(...args)
  }
}))

describe('ServiceAccounts', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    mockGet.mockResolvedValue({ api_keys: [] })
  })

  describe('rendering', () => {
    it('renders the service accounts page', async () => {
      const wrapper = mount(ServiceAccounts)
      await flushPromises()
      expect(wrapper.find('.min-h-screen').exists()).toBe(true)
    })

    it('displays page header', async () => {
      const wrapper = mount(ServiceAccounts)
      await flushPromises()
      expect(wrapper.text()).toContain('Service Accounts')
      expect(wrapper.text()).toContain('Manage API keys for service accounts')
    })

    it('renders NavBar component', async () => {
      const wrapper = mount(ServiceAccounts)
      await flushPromises()
      expect(wrapper.find('.navbar-mock').exists()).toBe(true)
    })

    it('shows Create Service Account Key button', async () => {
      const wrapper = mount(ServiceAccounts)
      await flushPromises()
      expect(wrapper.text()).toContain('Create Service Account Key')
    })
  })

  describe('table headers', () => {
    it('displays all table headers', async () => {
      const wrapper = mount(ServiceAccounts)
      await flushPromises()

      expect(wrapper.text()).toContain('Name')
      expect(wrapper.text()).toContain('Owner')
      expect(wrapper.text()).toContain('Scopes')
      expect(wrapper.text()).toContain('Status')
      expect(wrapper.text()).toContain('Last Used')
      expect(wrapper.text()).toContain('Created')
      expect(wrapper.text()).toContain('Actions')
    })
  })

  describe('loading state', () => {
    it('shows loading message while fetching', async () => {
      // Use a never-resolving promise to keep loading state
      let resolvePromise: (value: unknown) => void
      mockGet.mockImplementation(() => new Promise((resolve) => {
        resolvePromise = resolve
      }))

      const wrapper = mount(ServiceAccounts)
      // Need to wait for the component to start the fetch and update loading state
      await flushPromises()

      expect(wrapper.text()).toContain('Loading service accounts...')
    })
  })

  describe('empty state', () => {
    it('shows empty message when no service accounts', async () => {
      mockGet.mockResolvedValue({ api_keys: [] })

      const wrapper = mount(ServiceAccounts)
      await flushPromises()

      expect(wrapper.text()).toContain('No service accounts found')
    })
  })

  describe('data display', () => {
    const mockKeys = [
      {
        id: 'key-1',
        name: 'Test Service Key',
        key_prefix: 'mcp_abc123',
        user_id: 'user-123-456-789',
        user_email: 'service@example.com',
        scopes: ['servers:read', 'gateway:execute'],
        expires_at: null,
        last_used_at: '2024-01-15T10:30:00Z',
        created_at: '2024-01-01T00:00:00Z'
      }
    ]

    it('displays service account data', async () => {
      mockGet.mockResolvedValue({ api_keys: mockKeys })

      const wrapper = mount(ServiceAccounts)
      await flushPromises()

      expect(wrapper.text()).toContain('Test Service Key')
      expect(wrapper.text()).toContain('mcp_abc123...')
      expect(wrapper.text()).toContain('service@example.com')
      expect(wrapper.text()).toContain('user-123...')
    })

    it('displays scopes as badges', async () => {
      mockGet.mockResolvedValue({ api_keys: mockKeys })

      const wrapper = mount(ServiceAccounts)
      await flushPromises()

      expect(wrapper.text()).toContain('servers:read')
      expect(wrapper.text()).toContain('gateway:execute')
    })

    it('shows Active status for non-expired keys', async () => {
      mockGet.mockResolvedValue({ api_keys: mockKeys })

      const wrapper = mount(ServiceAccounts)
      await flushPromises()

      expect(wrapper.text()).toContain('Active')
    })

    it('shows Expired status for expired keys', async () => {
      const expiredKey = {
        ...mockKeys[0],
        expires_at: '2020-01-01T00:00:00Z'
      }
      mockGet.mockResolvedValue({ api_keys: [expiredKey] })

      const wrapper = mount(ServiceAccounts)
      await flushPromises()

      expect(wrapper.text()).toContain('Expired')
    })

    it('shows Full access when no scopes', async () => {
      const noScopeKey = {
        ...mockKeys[0],
        scopes: []
      }
      mockGet.mockResolvedValue({ api_keys: [noScopeKey] })

      const wrapper = mount(ServiceAccounts)
      await flushPromises()

      expect(wrapper.text()).toContain('Full access')
    })

    it('shows +N badge for more than 3 scopes', async () => {
      const manyScopes = {
        ...mockKeys[0],
        scopes: ['scope1', 'scope2', 'scope3', 'scope4', 'scope5']
      }
      mockGet.mockResolvedValue({ api_keys: [manyScopes] })

      const wrapper = mount(ServiceAccounts)
      await flushPromises()

      expect(wrapper.text()).toContain('+2')
    })

    it('shows Never for keys that have not been used', async () => {
      const unusedKey = {
        ...mockKeys[0],
        last_used_at: null
      }
      mockGet.mockResolvedValue({ api_keys: [unusedKey] })

      const wrapper = mount(ServiceAccounts)
      await flushPromises()

      expect(wrapper.text()).toContain('Never')
    })

    it('shows Revoke action button', async () => {
      mockGet.mockResolvedValue({ api_keys: mockKeys })

      const wrapper = mount(ServiceAccounts)
      await flushPromises()

      expect(wrapper.text()).toContain('Revoke')
    })
  })

  describe('filters', () => {
    it('has search input', async () => {
      const wrapper = mount(ServiceAccounts)
      await flushPromises()

      expect(wrapper.find('.input-mock').exists()).toBe(true)
    })

    it('has status filter dropdown', async () => {
      const wrapper = mount(ServiceAccounts)
      await flushPromises()

      const select = wrapper.find('select')
      expect(select.exists()).toBe(true)
      expect(wrapper.text()).toContain('All Status')
      expect(wrapper.text()).toContain('Active')
      expect(wrapper.text()).toContain('Expired')
    })

    it('filters by search term', async () => {
      const mockKeys = [
        { id: '1', name: 'Production Key', key_prefix: 'mcp_prod', user_email: 'prod@example.com', scopes: [], created_at: '2024-01-01' },
        { id: '2', name: 'Development Key', key_prefix: 'mcp_dev', user_email: 'dev@example.com', scopes: [], created_at: '2024-01-01' }
      ]
      mockGet.mockResolvedValue({ api_keys: mockKeys })

      const wrapper = mount(ServiceAccounts)
      await flushPromises()

      // Both should be visible initially
      expect(wrapper.text()).toContain('Production Key')
      expect(wrapper.text()).toContain('Development Key')

      // Type in search - need to find the actual input and set its value
      const input = wrapper.find('.input-mock')
      await input.setValue('Production')
      await flushPromises()

      expect(wrapper.text()).toContain('Production Key')
      expect(wrapper.text()).not.toContain('Development Key')
    })
  })

  describe('API calls', () => {
    it('fetches all API keys on mount', async () => {
      mount(ServiceAccounts)
      await flushPromises()

      expect(mockGet).toHaveBeenCalledWith('/admin/api-keys')
    })
  })

  describe('pagination info', () => {
    it('shows count of service accounts', async () => {
      const mockKeys = [
        { id: '1', name: 'Key 1', key_prefix: 'mcp_1', scopes: [], created_at: '2024-01-01' },
        { id: '2', name: 'Key 2', key_prefix: 'mcp_2', scopes: [], created_at: '2024-01-01' }
      ]
      mockGet.mockResolvedValue({ api_keys: mockKeys })

      const wrapper = mount(ServiceAccounts)
      await flushPromises()

      expect(wrapper.text()).toContain('Showing 2 service account(s)')
    })
  })

  describe('responsive layout', () => {
    it('has responsive container classes', async () => {
      const wrapper = mount(ServiceAccounts)
      await flushPromises()
      expect(wrapper.find('.max-w-7xl').exists()).toBe(true)
    })
  })
})
