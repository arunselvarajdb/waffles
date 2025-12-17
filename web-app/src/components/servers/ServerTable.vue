<template>
  <div class="bg-white rounded-lg shadow overflow-hidden">
    <!-- Table -->
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Status
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Server Name
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              URL
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Protocol
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Auth Type
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Health
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Active
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Actions
            </th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <!-- Empty State -->
          <tr v-if="servers.length === 0">
            <td colspan="8" class="px-6 py-12 text-center">
              <div class="text-gray-500">
                <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"></path>
                </svg>
                <p class="mt-2 text-sm font-medium">No servers found</p>
                <p class="mt-1 text-sm">Get started by adding a new server.</p>
              </div>
            </td>
          </tr>

          <!-- Server Rows -->
          <tr v-for="server in servers" :key="server.id" class="hover:bg-gray-50">
            <!-- Status Indicator -->
            <td class="px-6 py-4 whitespace-nowrap">
              <span
                :class="[
                  'h-3 w-3 rounded-full inline-block',
                  getStatusColor(server.health?.status)
                ]"
              />
            </td>

            <!-- Server Name & Description -->
            <td class="px-6 py-4 whitespace-nowrap">
              <div class="text-sm font-medium text-gray-900">{{ server.name }}</div>
              <div class="text-sm text-gray-500">{{ server.description || 'No description' }}</div>
            </td>

            <!-- URL -->
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              {{ server.url }}
            </td>

            <!-- Protocol Version -->
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              {{ server.protocol_version || 'N/A' }}
            </td>

            <!-- Auth Type -->
            <td class="px-6 py-4 whitespace-nowrap">
              <BaseBadge :variant="getAuthBadgeVariant(server.auth_type)">
                {{ formatAuthType(server.auth_type) }}
              </BaseBadge>
            </td>

            <!-- Health -->
            <td class="px-6 py-4 whitespace-nowrap">
              <BaseBadge :variant="getHealthBadgeVariant(server.health?.status)">
                {{ formatHealthStatus(server.health?.status) }}
              </BaseBadge>
              <div v-if="server.health?.last_checked" class="text-xs text-gray-400 mt-1">
                {{ formatTimestamp(server.health.last_checked) }}
              </div>
            </td>

            <!-- Active Toggle -->
            <td class="px-6 py-4 whitespace-nowrap">
              <BaseToggle
                :model-value="server.is_active"
                @change="$emit('toggle-server', server.id)"
              />
            </td>

            <!-- Actions -->
            <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
              <button
                @click="$emit('view-server', server)"
                class="text-gray-600 hover:text-gray-900 mr-3"
              >
                View
              </button>
              <button
                @click="$emit('edit-server', server)"
                class="text-blue-600 hover:text-blue-900 mr-3"
              >
                Edit
              </button>
              <button
                @click="$emit('delete-server', server)"
                class="text-red-600 hover:text-red-900"
              >
                Delete
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
import BaseToggle from '@/components/common/BaseToggle.vue'

defineProps({
  servers: {
    type: Array,
    default: () => []
  }
})

defineEmits(['view-server', 'edit-server', 'delete-server', 'toggle-server'])

const getStatusColor = (healthStatus) => {
  switch (healthStatus) {
    case 'healthy':
      return 'bg-green-500'
    case 'degraded':
      return 'bg-yellow-500'
    case 'unhealthy':
      return 'bg-red-500'
    default:
      return 'bg-gray-400'
  }
}

const getAuthBadgeVariant = (authType) => {
  switch (authType) {
    case 'none':
      return 'secondary'
    case 'basic':
      return 'info'
    case 'bearer':
      return 'info'
    case 'oauth':
      return 'success'
    default:
      return 'secondary'
  }
}

const formatAuthType = (authType) => {
  if (!authType) return 'None'
  return authType.charAt(0).toUpperCase() + authType.slice(1)
}

const getHealthBadgeVariant = (healthStatus) => {
  switch (healthStatus) {
    case 'healthy':
      return 'success'
    case 'degraded':
      return 'warning'
    case 'unhealthy':
      return 'danger'
    default:
      return 'secondary'
  }
}

const formatHealthStatus = (healthStatus) => {
  if (!healthStatus) return 'Unknown'
  return healthStatus.charAt(0).toUpperCase() + healthStatus.slice(1)
}

const formatTimestamp = (timestamp) => {
  if (!timestamp) return ''
  const now = new Date()
  const then = new Date(timestamp)
  const diffMs = now - then
  const diffMins = Math.floor(diffMs / 60000)

  if (diffMins < 1) return 'Just now'
  if (diffMins < 60) return `${diffMins} mins ago`
  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours} hours ago`
  const diffDays = Math.floor(diffHours / 24)
  return `${diffDays} days ago`
}
</script>
