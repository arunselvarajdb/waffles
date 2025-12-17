<template>
  <div class="grid grid-cols-1 md:grid-cols-4 gap-6 mb-8">
    <!-- Total Servers Card -->
    <div class="bg-white rounded-lg shadow p-6">
      <div class="flex items-center">
        <div class="flex-shrink-0 bg-blue-100 rounded-md p-3">
          <svg class="h-6 w-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"></path>
          </svg>
        </div>
        <div class="ml-4">
          <p class="text-sm font-medium text-gray-500">Total Servers</p>
          <p class="text-2xl font-semibold text-gray-900">{{ totalServers }}</p>
        </div>
      </div>
    </div>

    <!-- Active Servers Card -->
    <div class="bg-white rounded-lg shadow p-6">
      <div class="flex items-center">
        <div class="flex-shrink-0 bg-green-100 rounded-md p-3">
          <svg class="h-6 w-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
          </svg>
        </div>
        <div class="ml-4">
          <p class="text-sm font-medium text-gray-500">Active Servers</p>
          <p class="text-2xl font-semibold text-gray-900">{{ activeServers }}</p>
        </div>
      </div>
    </div>

    <!-- Healthy Servers Card -->
    <div class="bg-white rounded-lg shadow p-6">
      <div class="flex items-center">
        <div class="flex-shrink-0 bg-green-100 rounded-md p-3">
          <svg class="h-6 w-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4.318 6.318a4.5 4.5 0 000 6.364L12 20.364l7.682-7.682a4.5 4.5 0 00-6.364-6.364L12 7.636l-1.318-1.318a4.5 4.5 0 00-6.364 0z"></path>
          </svg>
        </div>
        <div class="ml-4">
          <p class="text-sm font-medium text-gray-500">Healthy Servers</p>
          <p class="text-2xl font-semibold text-gray-900">{{ healthyServers }}</p>
        </div>
      </div>
    </div>

    <!-- Issues Card -->
    <div class="bg-white rounded-lg shadow p-6">
      <div class="flex items-center">
        <div class="flex-shrink-0 bg-red-100 rounded-md p-3">
          <svg class="h-6 w-6 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
          </svg>
        </div>
        <div class="ml-4">
          <p class="text-sm font-medium text-gray-500">Issues</p>
          <p class="text-2xl font-semibold text-gray-900">{{ issuesCount }}</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useServersStore } from '@/stores/servers'

const serversStore = useServersStore()

const totalServers = computed(() => serversStore.servers.length)
const activeServers = computed(() => serversStore.activeServers.length)
const healthyServers = computed(() => serversStore.healthyServers.length)
const issuesCount = computed(() => {
  return serversStore.servers.filter(s =>
    s.health?.status === 'degraded' || s.health?.status === 'unhealthy'
  ).length
})
</script>
