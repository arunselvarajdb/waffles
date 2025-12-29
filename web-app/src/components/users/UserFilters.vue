<template>
  <div class="bg-white rounded-lg shadow p-4 mb-6">
    <div class="flex flex-wrap items-center gap-4">
      <!-- Search -->
      <div class="flex-1 min-w-[200px]">
        <BaseInput
          v-model="localFilters.search"
          type="text"
          placeholder="Search by name or email..."
          @input="debouncedUpdate"
        >
          <template #prefix>
            <svg class="h-5 w-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
            </svg>
          </template>
        </BaseInput>
      </div>

      <!-- Status Filter -->
      <div class="w-40">
        <select
          v-model="localFilters.status"
          class="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          @change="updateFilters"
        >
          <option value="all">All Status</option>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
        </select>
      </div>

      <!-- Role Filter -->
      <div class="w-40">
        <select
          v-model="localFilters.role"
          class="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          @change="updateFilters"
        >
          <option value="all">All Roles</option>
          <option value="admin">Admin</option>
          <option value="operator">Operator</option>
          <option value="viewer">Viewer</option>
        </select>
      </div>

      <!-- Clear Filters -->
      <BaseButton
        v-if="hasActiveFilters"
        variant="secondary"
        size="sm"
        @click="clearFilters"
      >
        Clear Filters
      </BaseButton>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import BaseInput from '@/components/common/BaseInput.vue'
import BaseButton from '@/components/common/BaseButton.vue'

const props = defineProps({
  filters: {
    type: Object,
    default: () => ({
      search: '',
      status: 'all',
      role: 'all'
    })
  }
})

const emit = defineEmits(['update:filters'])

const localFilters = ref({ ...props.filters })

// Watch for external filter changes
watch(() => props.filters, (newFilters) => {
  localFilters.value = { ...newFilters }
}, { deep: true })

const hasActiveFilters = computed(() => {
  return localFilters.value.search !== '' ||
    localFilters.value.status !== 'all' ||
    localFilters.value.role !== 'all'
})

let debounceTimer = null
const debouncedUpdate = () => {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    updateFilters()
  }, 300)
}

const updateFilters = () => {
  emit('update:filters', { ...localFilters.value })
}

const clearFilters = () => {
  localFilters.value = {
    search: '',
    status: 'all',
    role: 'all'
  }
  updateFilters()
}
</script>
