import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import NavBar from './NavBar.vue'

// Mock vue-router
const mockPush = vi.fn()
vi.mock('vue-router', () => ({
  useRouter: () => ({ push: mockPush }),
  useRoute: () => ({ path: '/admin' })
}))

// Mock BaseBadge component
vi.mock('@/components/common/BaseBadge.vue', () => ({
  default: {
    template: '<span class="badge-mock"><slot /></span>',
    props: ['variant']
  }
}))

// Mock auth store - needs to be after imports but return a fresh mock each time
const createMockAuthStore = (overrides = {}) => ({
  role: 'admin',
  user: { email: 'admin@example.com' },
  isAdmin: true,
  logout: vi.fn(),
  ...overrides
})

vi.mock('@/stores/auth', () => ({
  useAuthStore: vi.fn(() => createMockAuthStore())
}))

import { useAuthStore } from '@/stores/auth'

describe('NavBar', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    // Reset to admin mock by default
    vi.mocked(useAuthStore).mockReturnValue(createMockAuthStore())
  })

  it('renders the nav element', () => {
    const wrapper = mount(NavBar, {
      global: {
        stubs: { RouterLink: true }
      }
    })
    expect(wrapper.find('nav').exists()).toBe(true)
  })

  it('displays MCP Gateway title', () => {
    const wrapper = mount(NavBar, {
      global: {
        stubs: { RouterLink: true }
      }
    })
    expect(wrapper.text()).toContain('MCP Gateway')
  })

  it('shows admin badge for admin users', () => {
    const wrapper = mount(NavBar, {
      global: {
        stubs: { RouterLink: true }
      }
    })
    expect(wrapper.text()).toContain('Admin')
  })

  it('shows viewer badge for viewer users', () => {
    vi.mocked(useAuthStore).mockReturnValue(createMockAuthStore({
      role: 'viewer',
      user: { email: 'viewer@example.com' },
      isAdmin: false
    }))

    const wrapper = mount(NavBar, {
      global: {
        stubs: { RouterLink: true }
      }
    })
    expect(wrapper.text()).toContain('Viewer')
  })

  it('displays user email', () => {
    const wrapper = mount(NavBar, {
      global: {
        stubs: { RouterLink: true }
      }
    })
    expect(wrapper.text()).toContain('admin@example.com')
  })

  it('shows navigation links for admin users', () => {
    const wrapper = mount(NavBar, {
      global: {
        stubs: {
          RouterLink: {
            template: '<a><slot /></a>'
          }
        }
      }
    })
    expect(wrapper.text()).toContain('Servers')
    expect(wrapper.text()).toContain('Inspector')
    expect(wrapper.text()).toContain('Users')
    expect(wrapper.text()).toContain('Namespaces')
  })

  it('hides navigation links for non-admin users', () => {
    vi.mocked(useAuthStore).mockReturnValue(createMockAuthStore({
      role: 'viewer',
      user: { email: 'viewer@example.com' },
      isAdmin: false
    }))

    const wrapper = mount(NavBar, {
      global: {
        stubs: { RouterLink: true }
      }
    })
    // Navigation links should be hidden
    expect(wrapper.find('.sm\\:flex.sm\\:space-x-8').exists()).toBe(false)
  })

  it('has logout button', () => {
    const wrapper = mount(NavBar, {
      global: {
        stubs: { RouterLink: true }
      }
    })
    expect(wrapper.text()).toContain('Logout')
  })

  it('calls logout and redirects on logout click', async () => {
    const mockLogout = vi.fn()
    vi.mocked(useAuthStore).mockReturnValue(createMockAuthStore({
      logout: mockLogout
    }))

    const wrapper = mount(NavBar, {
      global: {
        stubs: { RouterLink: true }
      }
    })

    const logoutButton = wrapper.findAll('button').find(b => b.text() === 'Logout')
    await logoutButton?.trigger('click')

    expect(mockLogout).toHaveBeenCalled()
    expect(mockPush).toHaveBeenCalledWith('/login')
  })

  it('uses info variant badge for admin', () => {
    const wrapper = mount(NavBar, {
      global: {
        stubs: { RouterLink: true }
      }
    })
    const badge = wrapper.findComponent({ name: 'BaseBadge' })
    expect(badge.props('variant')).toBe('info')
  })

  it('uses secondary variant badge for viewer', () => {
    vi.mocked(useAuthStore).mockReturnValue(createMockAuthStore({
      role: 'viewer',
      user: { email: 'viewer@example.com' },
      isAdmin: false
    }))

    const wrapper = mount(NavBar, {
      global: {
        stubs: { RouterLink: true }
      }
    })
    const badge = wrapper.findComponent({ name: 'BaseBadge' })
    expect(badge.props('variant')).toBe('secondary')
  })

  it('has responsive classes', () => {
    const wrapper = mount(NavBar, {
      global: {
        stubs: { RouterLink: true }
      }
    })
    expect(wrapper.find('.max-w-7xl').exists()).toBe(true)
  })

  it('does not show badge when no role', () => {
    vi.mocked(useAuthStore).mockReturnValue(createMockAuthStore({
      role: null,
      user: null,
      isAdmin: false
    }))

    const wrapper = mount(NavBar, {
      global: {
        stubs: { RouterLink: true }
      }
    })
    expect(wrapper.find('.badge-mock').exists()).toBe(false)
  })
})
