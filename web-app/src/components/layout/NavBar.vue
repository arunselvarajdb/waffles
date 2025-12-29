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
          </div>
        </div>

        <!-- Right side: API Keys, User Info and Logout -->
        <div class="flex-shrink-0 flex items-center space-x-4">
          <router-link
            to="/settings/api-keys"
            class="text-sm text-gray-500 hover:text-gray-700 transition-colors"
            :class="isActive('/settings/api-keys') ? 'text-blue-600 font-medium' : ''"
          >
            API Keys
          </router-link>
          <span v-if="userEmail" class="text-sm text-gray-700 whitespace-nowrap">{{ userEmail }}</span>
          <button
            @click="handleLogout"
            class="text-sm text-gray-500 hover:text-gray-700 transition-colors"
          >
            Logout
          </button>
        </div>
      </div>
    </div>
  </nav>
</template>

<script setup>
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import BaseBadge from '@/components/common/BaseBadge.vue'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

const userRole = computed(() => authStore.role)
const userEmail = computed(() => authStore.user?.email)
const isAdmin = computed(() => authStore.isAdmin)

const roleBadgeVariant = computed(() => {
  return authStore.isAdmin ? 'info' : 'secondary'
})

const isActive = (path) => {
  return route.path === path
}

const handleLogout = async () => {
  await authStore.logout()
  router.push('/login')
}
</script>
