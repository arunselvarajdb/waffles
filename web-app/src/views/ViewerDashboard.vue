<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Navigation -->
    <NavBar />

    <!-- Main Content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Page Header -->
      <div class="mb-8">
        <h2 class="text-2xl font-bold text-gray-900">Viewer Dashboard</h2>
        <p class="mt-1 text-sm text-gray-600">View server configurations and execute tools</p>
      </div>

      <!-- Server Stats (Read-Only) -->
      <ServerStats />

      <!-- Available Servers -->
      <div class="bg-white rounded-lg shadow overflow-hidden mt-6">
        <div class="px-6 py-4 border-b border-gray-200">
          <h3 class="text-lg font-medium text-gray-900">Available Servers</h3>
          <p class="mt-1 text-sm text-gray-600">Servers you have access to</p>
        </div>

        <div class="p-12 text-center">
          <svg class="mx-auto h-16 w-16 text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <h3 class="text-lg font-medium text-gray-900 mb-2">Tool Execution Interface Coming Soon</h3>
          <p class="text-gray-600 max-w-md mx-auto">
            This interface will allow viewers to browse available servers and execute MCP tools
            with configured parameters. Full implementation planned for a future release.
          </p>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useServersStore } from '@/stores/servers'
import NavBar from '@/components/layout/NavBar.vue'
import ServerStats from '@/components/servers/ServerStats.vue'

const serversStore = useServersStore()

onMounted(async () => {
  await serversStore.fetchServers()
})
</script>
