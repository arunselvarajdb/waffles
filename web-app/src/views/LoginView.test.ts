import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, VueWrapper } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import LoginView from './LoginView.vue'

// Mock the auth store
const mockLogin = vi.fn()
const mockLoginAsAdmin = vi.fn()
const mockLoginAsViewer = vi.fn()

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    login: mockLogin,
    loginAsAdmin: mockLoginAsAdmin,
    loginAsViewer: mockLoginAsViewer,
    isAuthenticated: false,
    user: null,
    roles: [],
  }),
}))

// Mock fetch for SSO status check
global.fetch = vi.fn(() =>
  Promise.resolve({
    ok: true,
    json: () => Promise.resolve({ enabled: false }),
  })
) as unknown as typeof fetch

describe('LoginView', () => {
  let router: ReturnType<typeof createRouter>
  let wrapper: VueWrapper | null = null

  beforeEach(async () => {
    // Create a fresh router for each test using memory history (works in test environment)
    router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: { template: '<div>Home</div>' } },
        { path: '/login', component: LoginView },
        { path: '/admin', component: { template: '<div>Admin</div>' } },
        { path: '/dashboard', component: { template: '<div>Dashboard</div>' } },
      ],
    })

    setActivePinia(createPinia())
    vi.clearAllMocks()
    router.push('/login')
    await router.isReady()
  })

  afterEach(() => {
    // Cleanup wrapper to prevent async navigation after test teardown
    if (wrapper) {
      wrapper.unmount()
      wrapper = null
    }
  })

  const mountComponent = () => {
    wrapper = mount(LoginView, {
      global: {
        plugins: [router, createPinia()],
      },
    })
    return wrapper
  }

  describe('rendering', () => {
    it('renders login form', async () => {
      const wrapper = mountComponent()
      await flushPromises()

      expect(wrapper.find('h1').text()).toBe('MCP Gateway')
      expect(wrapper.find('input[type="email"]').exists()).toBe(true)
      expect(wrapper.find('input[type="password"]').exists()).toBe(true)
      expect(wrapper.find('button[type="submit"]').exists()).toBe(true)
    })

    it('renders demo login buttons', async () => {
      const wrapper = mountComponent()
      await flushPromises()

      const buttons = wrapper.findAll('button')
      const buttonTexts = buttons.map((b) => b.text())

      expect(buttonTexts).toContain('Demo: Login as Admin')
      expect(buttonTexts).toContain('Demo: Login as Viewer')
    })

    it('does not show SSO button when SSO is disabled', async () => {
      const wrapper = mountComponent()
      await flushPromises()

      expect(wrapper.text()).not.toContain('Sign in with SSO')
    })

    it('shows SSO button when SSO is enabled', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ enabled: true }),
      } as Response)

      const wrapper = mountComponent()
      await flushPromises()

      expect(wrapper.text()).toContain('Sign in with SSO')
    })
  })

  describe('email login', () => {
    it('calls login with email and password on form submit', async () => {
      mockLogin.mockResolvedValue('/admin')

      const wrapper = mountComponent()
      await flushPromises()

      await wrapper.find('input[type="email"]').setValue('test@example.com')
      await wrapper.find('input[type="password"]').setValue('password123')
      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(mockLogin).toHaveBeenCalledWith('test@example.com', 'password123')
    })

    it('shows loading state during login', async () => {
      mockLogin.mockImplementation(
        () => new Promise((resolve) => setTimeout(() => resolve('/admin'), 100))
      )

      const wrapper = mountComponent()
      await flushPromises()

      await wrapper.find('input[type="email"]').setValue('test@example.com')
      await wrapper.find('input[type="password"]').setValue('password123')
      await wrapper.find('form').trigger('submit')

      expect(wrapper.find('button[type="submit"]').text()).toContain('Signing in...')
    })

    it('shows error message on login failure', async () => {
      mockLogin.mockRejectedValue({
        response: { data: { message: 'Invalid credentials' } },
      })

      const wrapper = mountComponent()
      await flushPromises()

      await wrapper.find('input[type="email"]').setValue('test@example.com')
      await wrapper.find('input[type="password"]').setValue('wrongpassword')
      await wrapper.find('form').trigger('submit')
      await flushPromises()

      expect(wrapper.text()).toContain('Invalid credentials')
    })
  })

  describe('demo login', () => {
    it('calls loginAsAdmin when admin demo button is clicked', async () => {
      mockLoginAsAdmin.mockResolvedValue(true)

      const wrapper = mountComponent()
      await flushPromises()

      const adminButton = wrapper
        .findAll('button')
        .find((b) => b.text().includes('Demo: Login as Admin'))
      await adminButton?.trigger('click')
      await flushPromises()

      expect(mockLoginAsAdmin).toHaveBeenCalled()
    })

    it('calls loginAsViewer when viewer demo button is clicked', async () => {
      mockLoginAsViewer.mockResolvedValue(true)

      const wrapper = mountComponent()
      await flushPromises()

      const viewerButton = wrapper
        .findAll('button')
        .find((b) => b.text().includes('Demo: Login as Viewer'))
      await viewerButton?.trigger('click')
      await flushPromises()

      expect(mockLoginAsViewer).toHaveBeenCalled()
    })
  })

  describe('form validation', () => {
    it('has required attribute on email input', async () => {
      const wrapper = mountComponent()
      await flushPromises()

      const emailInput = wrapper.find('input[type="email"]')
      expect(emailInput.attributes('required')).toBeDefined()
    })

    it('has required attribute on password input', async () => {
      const wrapper = mountComponent()
      await flushPromises()

      const passwordInput = wrapper.find('input[type="password"]')
      expect(passwordInput.attributes('required')).toBeDefined()
    })
  })
})
