<template>
  <nav class="bg-white shadow-sm border-b border-gray-200">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="flex justify-between h-16">
        <!-- Left side: Logo and Navigation -->
        <div class="flex">
          <div class="flex-shrink-0 flex items-center">
            <h1 class="text-2xl font-bold text-blue-600">MCP Gateway</h1>
            <BaseBadge v-if="userRole" :variant="roleBadgeVariant" class="ml-3">
              {{ userRole === 'admin' ? 'Admin' : 'Viewer' }}
            </BaseBadge>
          </div>

          <!-- Desktop Navigation Links -->
          <div v-if="isAdmin" class="hidden sm:ml-6 sm:flex sm:space-x-8">
            <router-link
              to="/admin"
              class="border-transparent hover:border-gray-300 hover:text-gray-700 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium"
              :class="isActive('/admin') ? 'border-blue-500 text-gray-900' : 'text-gray-500'"
            >
              Servers
            </router-link>
            <router-link
              to="/admin/inspector"
              class="border-transparent hover:border-gray-300 hover:text-gray-700 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium"
              :class="isActive('/admin/inspector') ? 'border-blue-500 text-gray-900' : 'text-gray-500'"
            >
              Inspector
            </router-link>
            <router-link
              to="/admin/users"
              class="border-transparent hover:border-gray-300 hover:text-gray-700 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium"
              :class="isActive('/admin/users') ? 'border-blue-500 text-gray-900' : 'text-gray-500'"
            >
              Users
            </router-link>
            <router-link
              to="/admin/namespaces"
              class="border-transparent hover:border-gray-300 hover:text-gray-700 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium"
              :class="isActive('/admin/namespaces') ? 'border-blue-500 text-gray-900' : 'text-gray-500'"
            >
              Namespaces
            </router-link>
            <router-link
              to="/admin/service-accounts"
              class="border-transparent hover:border-gray-300 hover:text-gray-700 inline-flex items-center px-1 pt-1 border-b-2 text-sm font-medium"
              :class="isActive('/admin/service-accounts') ? 'border-blue-500 text-gray-900' : 'text-gray-500'"
            >
              Service Accounts
            </router-link>
          </div>
        </div>

        <!-- Right side: User Menu -->
        <div class="flex-shrink-0 flex items-center space-x-4">
          <!-- User dropdown menu -->
          <div class="relative" ref="userMenuRef">
            <button
              @click="showUserMenu = !showUserMenu"
              class="flex items-center space-x-2 text-sm text-gray-700 hover:text-gray-900 focus:outline-none"
            >
              <span class="hidden sm:inline">{{ userEmail }}</span>
              <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
              </svg>
            </button>

            <!-- Dropdown menu -->
            <div
              v-if="showUserMenu"
              class="absolute right-0 mt-2 w-48 bg-white rounded-md shadow-lg ring-1 ring-black ring-opacity-5 z-50"
            >
              <div class="py-1">
                <router-link
                  to="/profile"
                  class="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
                  @click="showUserMenu = false"
                >
                  Profile
                </router-link>
                <router-link
                  to="/profile?tab=api-keys"
                  class="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
                  @click="showUserMenu = false"
                >
                  API Keys
                </router-link>
                <router-link
                  to="/profile?tab=security"
                  class="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
                  @click="showUserMenu = false"
                >
                  Security
                </router-link>
                <hr class="my-1 border-gray-200" />
                <button
                  @click="handleLogout"
                  class="block w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-100"
                >
                  Logout
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </nav>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import BaseBadge from '@/components/common/BaseBadge.vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const userRole = computed(() => authStore.role)
const userEmail = computed(() => authStore.user?.email)
const isAdmin = computed(() => authStore.isAdmin)

const showUserMenu = ref(false)
const userMenuRef = ref(null)

const roleBadgeVariant = computed(() => {
  return authStore.isAdmin ? 'info' : 'secondary'
})

const isActive = (path) => {
  return route.path === path
}

const handleLogout = async () => {
  showUserMenu.value = false
  await authStore.logout()
  router.push('/login')
}

// Close dropdown when clicking outside
const handleClickOutside = (event) => {
  if (userMenuRef.value && !userMenuRef.value.contains(event.target)) {
    showUserMenu.value = false
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>
