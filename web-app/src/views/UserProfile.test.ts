import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import UserProfile from './UserProfile.vue'

// Mock child components
vi.mock('@/components/layout/NavBar.vue', () => ({
  default: { template: '<div class="navbar-mock">NavBar</div>' }
}))
vi.mock('@/components/common/BaseBadge.vue', () => ({
  default: {
    template: '<span class="badge-mock"><slot /></span>',
    props: ['variant']
  }
}))
vi.mock('@/components/common/BaseButton.vue', () => ({
  default: {
    template: '<button class="button-mock" :type="type"><slot /></button>',
    props: ['variant', 'type', 'disabled']
  }
}))
vi.mock('@/components/common/BaseInput.vue', () => ({
  default: {
    template: '<input class="input-mock" :type="type" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />',
    props: ['modelValue', 'type', 'placeholder', 'error'],
    emits: ['update:modelValue']
  }
}))
vi.mock('@/components/apikeys/ApiKeyStats.vue', () => ({
  default: { template: '<div class="apikey-stats-mock">ApiKeyStats</div>', props: ['keys'] }
}))
vi.mock('@/components/apikeys/ApiKeyTable.vue', () => ({
  default: { template: '<div class="apikey-table-mock">ApiKeyTable</div>', props: ['keys', 'loading'] }
}))
vi.mock('@/components/apikeys/CreateApiKeyModal.vue', () => ({
  default: { template: '<div class="create-modal-mock" v-if="modelValue">CreateApiKeyModal</div>', props: ['modelValue'] }
}))
vi.mock('@/components/apikeys/ApiKeyCreatedModal.vue', () => ({
  default: { template: '<div class="created-modal-mock" v-if="modelValue">ApiKeyCreatedModal</div>', props: ['modelValue', 'apiKey'] }
}))

// Mock API service
vi.mock('@/services/api', () => ({
  default: {
    post: vi.fn(),
    get: vi.fn(),
    delete: vi.fn()
  }
}))

// Create mock functions for stores
const mockFetchApiKeys = vi.fn()
const mockCreateApiKey = vi.fn()
const mockDeleteApiKey = vi.fn()

const createMockAuthStore = (overrides = {}) => ({
  user: { id: 'user-123', email: 'test@example.com' },
  role: 'admin',
  isAdmin: true,
  authProvider: 'local',
  canChangePassword: true,
  ...overrides
})

const createMockApiKeysStore = (overrides = {}) => ({
  apiKeys: [],
  loading: false,
  fetchApiKeys: mockFetchApiKeys,
  createApiKey: mockCreateApiKey,
  deleteApiKey: mockDeleteApiKey,
  ...overrides
})

vi.mock('@/stores/auth', () => ({
  useAuthStore: vi.fn(() => createMockAuthStore())
}))

vi.mock('@/stores/apikeys', () => ({
  useApiKeysStore: vi.fn(() => createMockApiKeysStore())
}))

import { useAuthStore } from '@/stores/auth'
import { useApiKeysStore } from '@/stores/apikeys'

describe('UserProfile', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(useAuthStore).mockReturnValue(createMockAuthStore())
    vi.mocked(useApiKeysStore).mockReturnValue(createMockApiKeysStore())
    // Reset window.location.search
    Object.defineProperty(window, 'location', {
      value: { search: '' },
      writable: true
    })
  })

  describe('rendering', () => {
    it('renders the profile page', () => {
      const wrapper = mount(UserProfile)
      expect(wrapper.find('.min-h-screen').exists()).toBe(true)
    })

    it('displays page header', () => {
      const wrapper = mount(UserProfile)
      expect(wrapper.text()).toContain('Account Settings')
      expect(wrapper.text()).toContain('Manage your profile, security settings, and API keys')
    })

    it('renders NavBar component', () => {
      const wrapper = mount(UserProfile)
      expect(wrapper.find('.navbar-mock').exists()).toBe(true)
    })

    it('renders all three tabs', () => {
      const wrapper = mount(UserProfile)
      expect(wrapper.text()).toContain('Profile')
      expect(wrapper.text()).toContain('Security')
      expect(wrapper.text()).toContain('API Keys')
    })
  })

  describe('Profile tab', () => {
    it('shows profile tab by default', () => {
      const wrapper = mount(UserProfile)
      expect(wrapper.text()).toContain('Profile Information')
      expect(wrapper.text()).toContain('Email')
    })

    it('displays user email', () => {
      const wrapper = mount(UserProfile)
      expect(wrapper.text()).toContain('test@example.com')
    })

    it('displays user role', () => {
      const wrapper = mount(UserProfile)
      expect(wrapper.find('.badge-mock').exists()).toBe(true)
    })

    it('displays user ID', () => {
      const wrapper = mount(UserProfile)
      expect(wrapper.text()).toContain('user-123')
    })
  })

  describe('Security tab', () => {
    it('shows security tab when clicked', async () => {
      const wrapper = mount(UserProfile)

      const securityTab = wrapper.findAll('button').find(b => b.text() === 'Security')
      await securityTab?.trigger('click')

      expect(wrapper.text()).toContain('Security Settings')
    })

    it('shows password change form when canChangePassword is true', async () => {
      vi.mocked(useAuthStore).mockReturnValue(createMockAuthStore({
        canChangePassword: true,
        authProvider: 'local'
      }))

      const wrapper = mount(UserProfile)
      const securityTab = wrapper.findAll('button').find(b => b.text() === 'Security')
      await securityTab?.trigger('click')

      expect(wrapper.text()).toContain('Change Password')
      expect(wrapper.text()).toContain('Current Password')
      expect(wrapper.text()).toContain('New Password')
      expect(wrapper.text()).toContain('Confirm New Password')
    })

    it('shows OAuth notice when authProvider is oauth', async () => {
      vi.mocked(useAuthStore).mockReturnValue(createMockAuthStore({
        canChangePassword: false,
        authProvider: 'oauth'
      }))

      const wrapper = mount(UserProfile)
      const securityTab = wrapper.findAll('button').find(b => b.text() === 'Security')
      await securityTab?.trigger('click')

      expect(wrapper.text()).toContain('Single Sign-On (SSO)')
      expect(wrapper.text()).toContain('identity provider administrator')
    })

    it('shows active sessions section', async () => {
      const wrapper = mount(UserProfile)
      const securityTab = wrapper.findAll('button').find(b => b.text() === 'Security')
      await securityTab?.trigger('click')

      expect(wrapper.text()).toContain('Active Sessions')
      expect(wrapper.text()).toContain('Session management coming soon')
    })
  })

  describe('API Keys tab', () => {
    it('shows API Keys tab when clicked', async () => {
      const wrapper = mount(UserProfile)

      const apiKeysTab = wrapper.findAll('button').find(b => b.text() === 'API Keys')
      await apiKeysTab?.trigger('click')

      expect(wrapper.text()).toContain('API Keys')
      expect(wrapper.text()).toContain('Manage your personal API keys')
    })

    it('shows Create API Key button', async () => {
      const wrapper = mount(UserProfile)
      const apiKeysTab = wrapper.findAll('button').find(b => b.text() === 'API Keys')
      await apiKeysTab?.trigger('click')

      expect(wrapper.text()).toContain('Create API Key')
    })

    it('renders ApiKeyStats component', async () => {
      const wrapper = mount(UserProfile)
      const apiKeysTab = wrapper.findAll('button').find(b => b.text() === 'API Keys')
      await apiKeysTab?.trigger('click')

      expect(wrapper.find('.apikey-stats-mock').exists()).toBe(true)
    })

    it('renders ApiKeyTable component', async () => {
      const wrapper = mount(UserProfile)
      const apiKeysTab = wrapper.findAll('button').find(b => b.text() === 'API Keys')
      await apiKeysTab?.trigger('click')

      expect(wrapper.find('.apikey-table-mock').exists()).toBe(true)
    })
  })

  describe('tab switching from URL', () => {
    it('opens api-keys tab when tab=api-keys in URL', async () => {
      Object.defineProperty(window, 'location', {
        value: { search: '?tab=api-keys' },
        writable: true
      })

      const wrapper = mount(UserProfile)
      await flushPromises()
      expect(wrapper.text()).toContain('Manage your personal API keys')
    })

    it('opens security tab when tab=security in URL', async () => {
      Object.defineProperty(window, 'location', {
        value: { search: '?tab=security' },
        writable: true
      })

      const wrapper = mount(UserProfile)
      await flushPromises()
      expect(wrapper.text()).toContain('Security Settings')
    })
  })

  describe('API key store integration', () => {
    it('fetches API keys on mount', async () => {
      mount(UserProfile)
      await flushPromises()

      expect(mockFetchApiKeys).toHaveBeenCalled()
    })
  })

  describe('responsive layout', () => {
    it('has responsive container classes', () => {
      const wrapper = mount(UserProfile)
      expect(wrapper.find('.max-w-4xl').exists()).toBe(true)
    })
  })
})
