<template>
  <div class="bg-white rounded-lg shadow p-6 mb-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <!-- Search Input -->
      <div class="md:col-span-2">
        <BaseInput
          v-model="searchQuery"
          type="text"
          placeholder="Search by name or description..."
          @update:modelValue="handleSearchChange"
        />
      </div>

      <!-- Status Filter -->
      <div>
        <select
          v-model="statusFilter"
          @change="handleFilterChange"
          class="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:border-blue-500 focus:ring-blue-500"
        >
          <option value="all">All Statuses</option>
          <option value="active">Active Only</option>
          <option value="inactive">Inactive Only</option>
        </select>
      </div>

      <!-- Health Filter -->
      <div>
        <select
          v-model="healthFilter"
          @change="handleFilterChange"
          class="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:border-blue-500 focus:ring-blue-500"
        >
          <option value="all">All Health</option>
          <option value="healthy">Healthy Only</option>
          <option value="degraded">Degraded Only</option>
          <option value="unhealthy">Unhealthy Only</option>
        </select>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useServersStore } from '@/stores/servers'
import BaseInput from '@/components/common/BaseInput.vue'

const serversStore = useServersStore()

const searchQuery = ref('')
const statusFilter = ref('all')
const healthFilter = ref('all')

onMounted(() => {
  // Initialize filters from store
  searchQuery.value = serversStore.filters.search
  statusFilter.value = serversStore.filters.status
  healthFilter.value = serversStore.filters.health
})

const handleSearchChange = () => {
  serversStore.setFilters({
    search: searchQuery.value,
    status: statusFilter.value,
    health: healthFilter.value
  })
}

const handleFilterChange = () => {
  serversStore.setFilters({
    search: searchQuery.value,
    status: statusFilter.value,
    health: healthFilter.value
  })
}
</script>
