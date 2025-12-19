<template>
  <div class="bg-white rounded-lg shadow overflow-hidden">
    <!-- Table -->
    <div class="overflow-x-auto">
      <table class="min-w-full divide-y divide-gray-200">
        <thead class="bg-gray-50">
          <tr>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Name
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Description
            </th>
            <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Servers
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
          <tr v-if="namespaces.length === 0">
            <td colspan="5" class="px-6 py-12 text-center">
              <div class="text-gray-500">
                <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path>
                </svg>
                <p class="mt-2 text-sm font-medium">No namespaces found</p>
                <p class="mt-1 text-sm">Get started by creating a new namespace.</p>
              </div>
            </td>
          </tr>

          <!-- Namespace Rows -->
          <tr v-for="namespace in namespaces" :key="namespace.id" class="hover:bg-gray-50">
            <!-- Name -->
            <td class="px-6 py-4 whitespace-nowrap">
              <div class="text-sm font-medium text-gray-900">{{ namespace.name }}</div>
            </td>

            <!-- Description -->
            <td class="px-6 py-4">
              <div class="text-sm text-gray-500 max-w-md truncate">
                {{ namespace.description || 'No description' }}
              </div>
            </td>

            <!-- Server Count -->
            <td class="px-6 py-4 whitespace-nowrap">
              <BaseBadge variant="info">
                {{ namespace.server_count || 0 }} servers
              </BaseBadge>
            </td>

            <!-- Created -->
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              {{ formatDate(namespace.created_at) }}
            </td>

            <!-- Actions -->
            <td class="px-6 py-4 whitespace-nowrap text-sm font-medium">
              <button
                @click="$emit('view-namespace', namespace)"
                class="text-gray-600 hover:text-gray-900 mr-3"
              >
                View
              </button>
              <button
                @click="$emit('edit-namespace', namespace)"
                class="text-blue-600 hover:text-blue-900 mr-3"
              >
                Edit
              </button>
              <button
                @click="$emit('delete-namespace', namespace)"
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

defineProps({
  namespaces: {
    type: Array,
    default: () => []
  }
})

defineEmits(['view-namespace', 'edit-namespace', 'delete-namespace'])

const formatDate = (dateStr) => {
  if (!dateStr) return 'N/A'
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}
</script>
