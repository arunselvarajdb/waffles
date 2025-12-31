<template>
  <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
    <!-- Total Keys -->
    <div class="bg-white rounded-lg shadow-sm p-4">
      <div class="flex items-center">
        <div class="flex-shrink-0">
          <div class="p-2 bg-blue-100 rounded-lg">
            <svg class="h-6 w-6 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
            </svg>
          </div>
        </div>
        <div class="ml-4">
          <p class="text-sm font-medium text-gray-500">Total Keys</p>
          <p class="text-2xl font-semibold text-gray-900">{{ stats.total }}</p>
        </div>
      </div>
    </div>

    <!-- Active Keys -->
    <div class="bg-white rounded-lg shadow-sm p-4">
      <div class="flex items-center">
        <div class="flex-shrink-0">
          <div class="p-2 bg-green-100 rounded-lg">
            <svg class="h-6 w-6 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
        </div>
        <div class="ml-4">
          <p class="text-sm font-medium text-gray-500">Active</p>
          <p class="text-2xl font-semibold text-gray-900">{{ stats.active }}</p>
        </div>
      </div>
    </div>

    <!-- Expired Keys -->
    <div class="bg-white rounded-lg shadow-sm p-4">
      <div class="flex items-center">
        <div class="flex-shrink-0">
          <div class="p-2 bg-red-100 rounded-lg">
            <svg class="h-6 w-6 text-red-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
        </div>
        <div class="ml-4">
          <p class="text-sm font-medium text-gray-500">Expired</p>
          <p class="text-2xl font-semibold text-gray-900">{{ stats.expired }}</p>
        </div>
      </div>
    </div>

    <!-- Recently Used -->
    <div class="bg-white rounded-lg shadow-sm p-4">
      <div class="flex items-center">
        <div class="flex-shrink-0">
          <div class="p-2 bg-purple-100 rounded-lg">
            <svg class="h-6 w-6 text-purple-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
        </div>
        <div class="ml-4">
          <p class="text-sm font-medium text-gray-500">Used Today</p>
          <p class="text-2xl font-semibold text-gray-900">{{ stats.recentlyUsed }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  stats: {
    type: Object,
    required: true,
    default: () => ({
      total: 0,
      active: 0,
      expired: 0,
      recentlyUsed: 0
    })
  }
})
</script>
