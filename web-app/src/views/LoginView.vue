<template>
  <div class="bg-gradient-to-br from-blue-50 to-indigo-100 min-h-screen flex items-center justify-center">
    <div class="max-w-md w-full mx-4">
      <!-- Login Card -->
      <div class="bg-white rounded-lg shadow-xl p-8">
        <!-- Logo/Header -->
        <div class="text-center mb-8">
          <h1 class="text-3xl font-bold text-blue-600 mb-2">MCP Gateway</h1>
          <p class="text-gray-600">Sign in with your organization</p>
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
            class="w-full bg-blue-600 hover:bg-blue-700 disabled:bg-blue-400 text-white font-medium py-2 px-4 rounded-md transition duration-200"
          >
            <span v-if="loading">Signing in...</span>
            <span v-else>Sign In</span>
          </button>
        </form>

        <!-- SSO Login Options -->
        <div class="space-y-3">
          <p class="text-sm font-medium text-gray-700 text-center mb-4">Choose your SSO provider:</p>

          <!-- Google SSO -->
          <button
            @click="loginWithSSO('google')"
            class="w-full flex items-center justify-center px-4 py-3 border border-gray-300 rounded-md shadow-sm bg-white hover:bg-gray-50 transition duration-200"
          >
            <svg class="h-5 w-5 mr-3" viewBox="0 0 24 24">
              <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"/>
              <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"/>
              <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"/>
              <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"/>
            </svg>
            <span class="text-sm font-medium text-gray-700">Continue with Google</span>
          </button>

          <!-- Microsoft SSO -->
          <button
            @click="loginWithSSO('microsoft')"
            class="w-full flex items-center justify-center px-4 py-3 border border-gray-300 rounded-md shadow-sm bg-white hover:bg-gray-50 transition duration-200"
          >
            <svg class="h-5 w-5 mr-3" viewBox="0 0 23 23">
              <path fill="#f35325" d="M0 0h11v11H0z"/>
              <path fill="#81bc06" d="M12 0h11v11H12z"/>
              <path fill="#05a6f0" d="M0 12h11v11H0z"/>
              <path fill="#ffba08" d="M12 12h11v11H12z"/>
            </svg>
            <span class="text-sm font-medium text-gray-700">Continue with Microsoft</span>
          </button>

          <!-- GitHub SSO -->
          <button
            @click="loginWithSSO('github')"
            class="w-full flex items-center justify-center px-4 py-3 border border-gray-300 rounded-md shadow-sm bg-white hover:bg-gray-50 transition duration-200"
          >
            <svg class="h-5 w-5 mr-3" fill="currentColor" viewBox="0 0 24 24">
              <path fill-rule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" clip-rule="evenodd"/>
            </svg>
            <span class="text-sm font-medium text-gray-700">Continue with GitHub</span>
          </button>

          <!-- Okta SSO -->
          <button
            @click="loginWithSSO('okta')"
            class="w-full flex items-center justify-center px-4 py-3 border border-gray-300 rounded-md shadow-sm bg-white hover:bg-gray-50 transition duration-200"
          >
            <svg class="h-5 w-5 mr-3" viewBox="0 0 24 24">
              <circle fill="#007DC1" cx="12" cy="12" r="11"/>
              <path fill="#fff" d="M12 6c-3.3 0-6 2.7-6 6s2.7 6 6 6 6-2.7 6-6-2.7-6-6-6zm0 10c-2.2 0-4-1.8-4-4s1.8-4 4-4 4 1.8 4 4-1.8 4-4 4z"/>
            </svg>
            <span class="text-sm font-medium text-gray-700">Continue with Okta</span>
          </button>

          <!-- Generic SAML SSO -->
          <button
            @click="loginWithSSO('saml')"
            class="w-full flex items-center justify-center px-4 py-3 border border-gray-300 rounded-md shadow-sm bg-white hover:bg-gray-50 transition duration-200"
          >
            <svg class="h-5 w-5 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
            </svg>
            <span class="text-sm font-medium text-gray-700">Continue with SAML SSO</span>
          </button>
        </div>

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
          <strong>Note:</strong> This is a demonstration interface. SSO providers are not yet integrated.
          Use the demo buttons to explore admin and viewer roles.
        </p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const email = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

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

const loginWithSSO = (provider) => {
  // TODO: Implement actual SSO integration
  console.log(`SSO login with ${provider} not yet implemented`)
  alert(`SSO integration with ${provider} is not yet implemented. Please use the demo login buttons.`)
}

const loginAsAdmin = () => {
  authStore.loginAsAdmin()
  router.push('/admin')
}

const loginAsViewer = () => {
  authStore.loginAsViewer()
  router.push('/dashboard')
}
</script>
