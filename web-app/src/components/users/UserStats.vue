<template>
  <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
    <!-- Total Users -->
    <div class="bg-white rounded-lg shadow p-4">
      <div class="flex items-center">
        <div class="flex-shrink-0 bg-blue-100 rounded-md p-3">
          <svg class="h-6 w-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197m13.5-9a2.25 2.25 0 11-4.5 0 2.25 2.25 0 014.5 0z"></path>
          </svg>
        </div>
        <div class="ml-4">
          <p class="text-sm font-medium text-gray-500">Total Users</p>
          <p class="text-2xl font-semibold text-gray-900">{{ stats.total }}</p>
        </div>
      </div>
    </div>

    <!-- Active Users -->
    <div class="bg-white rounded-lg shadow p-4">
      <div class="flex items-center">
        <div class="flex-shrink-0 bg-green-100 rounded-md p-3">
          <svg class="h-6 w-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
          </svg>
        </div>
        <div class="ml-4">
          <p class="text-sm font-medium text-gray-500">Active</p>
          <p class="text-2xl font-semibold text-green-600">{{ stats.active }}</p>
        </div>
      </div>
    </div>

    <!-- Inactive Users -->
    <div class="bg-white rounded-lg shadow p-4">
      <div class="flex items-center">
        <div class="flex-shrink-0 bg-gray-100 rounded-md p-3">
          <svg class="h-6 w-6 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636"></path>
          </svg>
        </div>
        <div class="ml-4">
          <p class="text-sm font-medium text-gray-500">Inactive</p>
          <p class="text-2xl font-semibold text-gray-600">{{ stats.inactive }}</p>
        </div>
      </div>
    </div>

    <!-- Admin Users -->
    <div class="bg-white rounded-lg shadow p-4">
      <div class="flex items-center">
        <div class="flex-shrink-0 bg-red-100 rounded-md p-3">
          <svg class="h-6 w-6 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"></path>
          </svg>
        </div>
        <div class="ml-4">
          <p class="text-sm font-medium text-gray-500">Admins</p>
          <p class="text-2xl font-semibold text-red-600">{{ stats.admins }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  stats: {
    type: Object,
    default: () => ({
      total: 0,
      active: 0,
      inactive: 0,
      admins: 0
    })
  }
})
</script>
