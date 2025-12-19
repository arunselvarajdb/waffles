<template>
  <div class="bg-gradient-to-br from-blue-50 to-indigo-100 min-h-screen flex items-center justify-center">
    <div class="max-w-md w-full mx-4">
      <!-- Login Card -->
      <div class="bg-white rounded-lg shadow-xl p-8">
        <!-- Logo/Header -->
        <div class="text-center mb-8">
          <h1 class="text-3xl font-bold text-blue-600 mb-2">MCP Gateway</h1>
          <p class="text-gray-600">Sign in to your account</p>
        </div>

        <!-- SSO Login Button -->
        <div v-if="ssoEnabled" class="mb-6">
          <button
            @click="loginWithSSO"
            class="w-full flex items-center justify-center px-4 py-3 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-md shadow-sm transition duration-200"
          >
            <svg class="h-5 w-5 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
            </svg>
            <span>Sign in with SSO</span>
          </button>
        </div>

        <!-- Divider (only if SSO is enabled) -->
        <div v-if="ssoEnabled" class="relative mb-6">
          <div class="absolute inset-0 flex items-center">
            <div class="w-full border-t border-gray-300"></div>
          </div>
          <div class="relative flex justify-center text-sm">
            <span class="px-2 bg-white text-gray-500">Or sign in with email</span>
          </div>
        </div>

        <!-- Email/Password Login Form -->
        <form @submit.prevent="handleEmailLogin" class="space-y-4 mb-6">
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Email</label>
            <input
              v-model="email"
              type="email"
              required
              class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="you@example.com"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Password</label>
            <input
              v-model="password"
              type="password"
              required
              class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="Enter your password"
            />
          </div>

          <!-- Error Message -->
          <div v-if="error" class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded text-sm">
            {{ error }}
          </div>

          <button
            type="submit"
            :disabled="loading"
            class="w-full bg-gray-800 hover:bg-gray-900 disabled:bg-gray-400 text-white font-medium py-2 px-4 rounded-md transition duration-200"
          >
            <span v-if="loading">Signing in...</span>
            <span v-else>Sign In with Email</span>
          </button>
        </form>

        <!-- Demo Access Section -->
        <div class="mt-8 pt-6 border-t border-gray-200">
          <p class="text-sm text-gray-600 text-center mb-4">Demo Access (No Auth Required)</p>
          <div class="space-y-2">
            <button
              @click="loginAsAdmin"
              class="w-full bg-purple-600 hover:bg-purple-700 text-white font-medium py-2 px-4 rounded-md transition duration-200 text-sm"
            >
              Demo: Login as Admin
            </button>
            <button
              @click="loginAsViewer"
              class="w-full bg-green-600 hover:bg-green-700 text-white font-medium py-2 px-4 rounded-md transition duration-200 text-sm"
            >
              Demo: Login as Viewer
            </button>
          </div>
        </div>
      </div>

      <!-- Info Box -->
      <div class="mt-6 bg-blue-50 border border-blue-200 rounded-lg p-4">
        <p class="text-sm text-blue-800">
          <strong>Note:</strong> SSO works with any OIDC provider (Keycloak, Okta, Auth0, Azure AD, etc.).
          Configure your provider's credentials in <code class="bg-blue-100 px-1 rounded">config.yaml</code>.
          Use the demo buttons to explore admin and viewer roles.
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const email = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')
const ssoEnabled = ref(false)

onMounted(async () => {
  // Check for error in URL (from SSO callback)
  if (route.query.error) {
    error.value = route.query.error
  }

  // Check if SSO is enabled
  try {
    const baseUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080'
    const response = await fetch(`${baseUrl}/api/v1/auth/sso/status`, {
      credentials: 'include'
    })
    if (response.ok) {
      const data = await response.json()
      ssoEnabled.value = data.enabled
    }
  } catch (err) {
    console.warn('Failed to check SSO status:', err)
  }
})

const handleEmailLogin = async () => {
  error.value = ''
  loading.value = true

  try {
    const redirectPath = await authStore.login(email.value, password.value)
    router.push(redirectPath)
  } catch (err) {
    error.value = err.response?.data?.message || 'Invalid email or password'
  } finally {
    loading.value = false
  }
}

const loginWithSSO = () => {
  // Redirect to SSO endpoint
  const baseUrl = import.meta.env.VITE_API_URL || 'http://localhost:8080'
  window.location.href = `${baseUrl}/api/v1/auth/sso`
}

const loginAsAdmin = async () => {
  loading.value = true
  error.value = ''
  try {
    await authStore.loginAsAdmin()
    router.push('/admin')
  } catch (err) {
    error.value = 'Demo admin login failed'
  } finally {
    loading.value = false
  }
}

const loginAsViewer = async () => {
  loading.value = true
  error.value = ''
  try {
    await authStore.loginAsViewer()
    router.push('/dashboard')
  } catch (err) {
    error.value = 'Demo viewer login failed'
  } finally {
    loading.value = false
  }
}
</script>
