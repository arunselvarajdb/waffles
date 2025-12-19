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

        <!-- Loading State -->
        <div v-if="serversStore.loading" class="p-12 text-center">
          <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
          <p class="mt-4 text-gray-600">Loading servers...</p>
        </div>

        <!-- Empty State -->
        <div v-else-if="serversStore.servers.length === 0" class="p-12 text-center">
          <svg class="mx-auto h-16 w-16 text-gray-400 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
          </svg>
          <h3 class="text-lg font-medium text-gray-900 mb-2">No Servers Available</h3>
          <p class="text-gray-600 max-w-md mx-auto">
            You don't have access to any servers yet. Contact an administrator to get access to namespaces.
          </p>
        </div>

        <!-- Server List -->
        <div v-else class="divide-y divide-gray-200">
          <div
            v-for="server in serversStore.servers"
            :key="server.id"
            class="px-6 py-4 hover:bg-gray-50"
          >
            <div class="flex items-center justify-between">
              <div class="flex items-center space-x-4">
                <!-- Status Indicator -->
                <div
                  :class="[
                    'w-3 h-3 rounded-full',
                    server.is_active ? 'bg-green-500' : 'bg-gray-400'
                  ]"
                ></div>
                <div>
                  <h4 class="text-sm font-medium text-gray-900">{{ server.name }}</h4>
                  <p class="text-xs text-gray-500">{{ server.url }}</p>
                </div>
              </div>
              <div class="flex items-center space-x-4">
                <span
                  :class="[
                    'px-2 py-1 text-xs rounded-full',
                    server.is_active
                      ? 'bg-green-100 text-green-800'
                      : 'bg-gray-100 text-gray-800'
                  ]"
                >
                  {{ server.is_active ? 'Active' : 'Inactive' }}
                </span>
                <span class="text-xs text-gray-500">{{ server.transport }}</span>
                <button
                  @click="handleViewServer(server)"
                  class="px-3 py-1 text-sm font-medium text-blue-600 hover:text-blue-800 hover:bg-blue-50 rounded-md transition-colors"
                >
                  View
                </button>
              </div>
            </div>
            <p v-if="server.description" class="mt-2 text-sm text-gray-600 ml-7">
              {{ server.description }}
            </p>
          </div>
        </div>
      </div>
    </main>

    <!-- Server Details Modal (Read-Only) -->
    <ViewerServerDetailsModal
      v-model="showDetailsModal"
      :server="selectedServer"
    />
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useServersStore } from '@/stores/servers'
import NavBar from '@/components/layout/NavBar.vue'
import ServerStats from '@/components/servers/ServerStats.vue'
import ViewerServerDetailsModal from '@/components/servers/ViewerServerDetailsModal.vue'

const serversStore = useServersStore()

const showDetailsModal = ref(false)
const selectedServer = ref(null)

const handleViewServer = (server) => {
  selectedServer.value = server
  showDetailsModal.value = true
}

onMounted(async () => {
  await serversStore.fetchServers()
})
</script>
