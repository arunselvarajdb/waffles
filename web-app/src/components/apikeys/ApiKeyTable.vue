<template>
  <div class="bg-white shadow-sm rounded-lg overflow-hidden">
    <table class="min-w-full divide-y divide-gray-200">
      <thead class="bg-gray-50">
        <tr>
          <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
            Name
          </th>
          <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
            Key Prefix
          </th>
          <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
            Status
          </th>
          <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
            Last Used
          </th>
          <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
            Created
          </th>
          <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
            Actions
          </th>
        </tr>
      </thead>
      <tbody class="bg-white divide-y divide-gray-200">
        <tr v-for="key in apiKeys" :key="key.id" class="hover:bg-gray-50">
          <td class="px-6 py-4 whitespace-nowrap">
            <div class="text-sm font-medium text-gray-900">{{ key.name }}</div>
          </td>
          <td class="px-6 py-4 whitespace-nowrap">
            <code class="text-sm text-gray-600 bg-gray-100 px-2 py-1 rounded">
              {{ key.key_prefix }}
            </code>
          </td>
          <td class="px-6 py-4 whitespace-nowrap">
            <BaseBadge :variant="getStatusVariant(key)">
              {{ getStatusText(key) }}
            </BaseBadge>
          </td>
          <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
            {{ key.last_used_at ? formatDate(key.last_used_at) : 'Never' }}
          </td>
          <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
            {{ formatDate(key.created_at) }}
          </td>
          <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
            <button
              @click="$emit('view-key', key)"
              class="text-blue-600 hover:text-blue-900 mr-3"
            >
              View
            </button>
            <button
              @click="$emit('delete-key', key)"
              class="text-red-600 hover:text-red-900"
            >
              Revoke
            </button>
          </td>
        </tr>
        <tr v-if="apiKeys.length === 0">
          <td colspan="6" class="px-6 py-12 text-center text-gray-500">
            No API keys found. Create one to get started.
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup>
import BaseBadge from '@/components/common/BaseBadge.vue'

defineProps({
  apiKeys: {
    type: Array,
    required: true
  }
})

defineEmits(['view-key', 'delete-key'])

const isExpired = (key) => {
  if (!key.expires_at) return false
  return new Date(key.expires_at) <= new Date()
}

const getStatusVariant = (key) => {
  if (isExpired(key)) return 'danger'
  return 'success'
}

const getStatusText = (key) => {
  if (isExpired(key)) return 'Expired'
  if (key.expires_at) {
    const days = Math.ceil((new Date(key.expires_at) - new Date()) / (1000 * 60 * 60 * 24))
    return days <= 7 ? `Expires in ${days}d` : 'Active'
  }
  return 'Active'
}

const formatDate = (dateStr) => {
  if (!dateStr) return 'N/A'
  const date = new Date(dateStr)
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}
</script>
