<template>
  <div class="bg-white rounded-lg shadow overflow-hidden">
    <!-- Table -->
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              User
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Email
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Roles
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Status
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Created
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Actions
            </th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <!-- Empty State -->
          <tr v-if="users.length === 0">
            <td colspan="6" class="px-6 py-12 text-center">
              <div class="text-gray-500">
                <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197m13.5-9a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z"></path>
                </svg>
                <p class="mt-2 text-sm font-medium">No users found</p>
                <p class="mt-1 text-sm">Get started by adding a new user.</p>
              </div>
            </td>
          </tr>

          <!-- User Rows -->
          <tr v-for="user in users" :key="user.id" class="hover:bg-gray-50">
            <!-- User Name & Avatar -->
            <td class="px-6 py-4 whitespace-nowrap">
              <div class="flex items-center">
                <div class="flex-shrink-0 h-10 w-10">
                  <div class="h-10 w-10 rounded-full bg-gray-300 flex items-center justify-center text-gray-600 font-medium">
                    {{ getInitials(user.name) }}
                  </div>
                </div>
                <div class="ml-4">
                  <div class="text-sm font-medium text-gray-900">{{ user.name }}</div>
                  <div class="text-sm text-gray-500">ID: {{ user.id?.slice(0, 8) }}...</div>
                </div>
              </div>
            </td>

            <!-- Email -->
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              {{ user.email }}
            </td>

            <!-- Roles -->
            <td class="px-6 py-4 whitespace-nowrap">
              <div class="flex flex-wrap gap-1">
                <BaseBadge
                  v-for="role in user.roles || []"
                  :key="role"
                  :variant="getRoleBadgeVariant(role)"
                >
                  {{ role }}
                </BaseBadge>
                <span v-if="!user.roles || user.roles.length === 0" class="text-sm text-gray-400">
                  No roles
                </span>
              </div>
            </td>

            <!-- Status -->
            <td class="px-6 py-4 whitespace-nowrap">
              <BaseBadge :variant="user.is_active ? 'success' : 'danger'">
                {{ user.is_active ? 'Active' : 'Inactive' }}
              </BaseBadge>
            </td>

            <!-- Created -->
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              {{ formatDate(user.created_at) }}
            </td>

            <!-- Actions -->
            <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
              <button
                @click="$emit('edit-user', user)"
                class="text-blue-600 hover:text-blue-900 mr-3"
              >
                Edit
              </button>
              <button
                @click="$emit('manage-roles', user)"
                class="text-purple-600 hover:text-purple-900 mr-3"
              >
                Roles
              </button>
              <button
                @click="$emit('reset-password', user)"
                class="text-yellow-600 hover:text-yellow-900 mr-3"
              >
                Reset PW
              </button>
              <button
                v-if="user.is_active"
                @click="$emit('deactivate-user', user)"
                class="text-red-600 hover:text-red-900"
              >
                Deactivate
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup>
import BaseBadge from '@/components/common/BaseBadge.vue'

defineProps({
  users: {
    type: Array,
    default: () => []
  }
})

defineEmits(['edit-user', 'manage-roles', 'reset-password', 'deactivate-user'])

const getInitials = (name) => {
  if (!name) return '?'
  const parts = name.split(' ')
  if (parts.length >= 2) {
    return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
  }
  return name.substring(0, 2).toUpperCase()
}

const getRoleBadgeVariant = (role) => {
  switch (role) {
    case 'admin':
      return 'danger'
    case 'operator':
      return 'warning'
    case 'user':
      return 'info'
    case 'readonly':
      return 'secondary'
    default:
      return 'secondary'
  }
}

const formatDate = (dateString) => {
  if (!dateString) return 'N/A'
  const date = new Date(dateString)
  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}
</script>
