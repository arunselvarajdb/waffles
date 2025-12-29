import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import LoginView from '@/views/LoginView.vue'
import AdminDashboard from '@/views/AdminDashboard.vue'
import AdminUsers from '@/views/AdminUsers.vue'
import AdminRoles from '@/views/AdminRoles.vue'
import ServerInspector from '@/views/ServerInspector.vue'
import ViewerDashboard from '@/views/ViewerDashboard.vue'
import NamespacesDashboard from '@/views/NamespacesDashboard.vue'
import ApiKeys from '@/views/ApiKeys.vue'

const routes = [
  {
    path: '/',
    redirect: '/login'
  },
  {
    path: '/login',
    name: 'Login',
    component: LoginView,
    meta: { guest: true } // Only accessible when not logged in
  },
  {
    path: '/admin',
    name: 'AdminDashboard',
    component: AdminDashboard,
    meta: { requiresAuth: true, requiresAdmin: true }
  },
  {
    path: '/admin/users',
    name: 'AdminUsers',
    component: AdminUsers,
    meta: { requiresAuth: true, requiresAdmin: true }
  },
  {
    path: '/admin/roles',
    name: 'AdminRoles',
    component: AdminRoles,
    meta: { requiresAuth: true, requiresAdmin: true }
  },
  {
    path: '/admin/namespaces',
    name: 'NamespacesDashboard',
    component: NamespacesDashboard,
    meta: { requiresAuth: true, requiresAdmin: true }
  },
  {
    path: '/admin/inspector',
    name: 'ServerInspector',
    component: ServerInspector,
    meta: { requiresAuth: true, requiresAdmin: true }
  },
  {
    path: '/dashboard',
    name: 'ViewerDashboard',
    component: ViewerDashboard,
    meta: { requiresAuth: true }
  },
  {
    path: '/settings/api-keys',
    name: 'ApiKeys',
    component: ApiKeys,
    meta: { requiresAuth: true }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// Navigation guards
router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()

  // Check auth config if we haven't yet (this auto-logins when auth is disabled)
  if (authStore.authEnabled === null) {
    await authStore.checkAuthConfig()
  }

  // If auth is disabled, skip all auth checks
  if (authStore.authEnabled === false) {
    // Still redirect from login page to admin when auth is disabled
    if (to.path === '/login' || to.path === '/') {
      return next('/admin')
    }
    return next()
  }

  // Check auth status if we have stored state but haven't verified with server
  if (authStore.isAuthenticated && !authStore.user?.id) {
    await authStore.checkAuth()
  }

  // Route requires authentication
  if (to.meta.requiresAuth) {
    // If not authenticated in local state, try to verify with server
    // This handles SSO login where the session cookie exists but localStorage doesn't have the state
    if (!authStore.isAuthenticated) {
      await authStore.checkAuth()
    }

    // Still not authenticated after checking with server
    if (!authStore.isAuthenticated) {
      // Store the intended destination
      sessionStorage.setItem('redirectAfterLogin', to.fullPath)
      return next('/login')
    }

    // Route requires admin role
    if (to.meta.requiresAdmin && !authStore.isAdmin) {
      // Redirect non-admins to viewer dashboard
      return next('/dashboard')
    }
  }

  // Guest-only routes (like login)
  if (to.meta.guest && authStore.isAuthenticated) {
    // Redirect logged-in users to dashboard
    return next(authStore.isAdmin ? '/admin' : '/dashboard')
  }

  next()
})

export default router
