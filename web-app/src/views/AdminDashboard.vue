<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Navigation -->
    <NavBar />

    <!-- Main Content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Page Header -->
      <div class="flex justify-between items-center mb-8">
        <div>
          <h2 class="text-2xl font-bold text-gray-900">Server Management</h2>
          <p class="mt-1 text-sm text-gray-600">Manage MCP servers and their configurations</p>
        </div>
        <BaseButton variant="primary" @click="showAddModal = true">
          + Add Server
        </BaseButton>
      </div>

      <!-- Dashboard Statistics -->
      <ServerStats />

      <!-- Filters -->
      <ServerFilters />

      <!-- Server List -->
      <ServerTable
        :servers="filteredServers"
        @view-server="handleViewServer"
        @edit-server="handleEditServer"
        @delete-server="handleDeleteServer"
        @toggle-server="handleToggleServer"
      />

      <!-- Pagination -->
      <BasePagination
        v-if="filteredServers.length > 0"
        :current-page="currentPage"
        :total="pagination.total"
        :per-page="pagination.limit"
        @page-change="handlePageChange"
      />
    </main>

    <!-- Modals -->
    <AddServerModal
      v-model="showAddModal"
      @success="handleServerAdded"
    />

    <EditServerModal
      v-model="showEditModal"
      :server="selectedServer"
      @success="handleServerUpdated"
    />

    <DeleteServerModal
      v-model="showDeleteModal"
      :server="selectedServer"
      @success="handleServerDeleted"
    />

    <ServerDetailsModal
      v-model="showDetailsModal"
      :server="selectedServer"
      @edit-server="handleEditFromDetails"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useServersStore } from '@/stores/servers'
import NavBar from '@/components/layout/NavBar.vue'
import BaseButton from '@/components/common/BaseButton.vue'
import BasePagination from '@/components/common/BasePagination.vue'
import ServerStats from '@/components/servers/ServerStats.vue'
import ServerFilters from '@/components/servers/ServerFilters.vue'
import ServerTable from '@/components/servers/ServerTable.vue'
import AddServerModal from '@/components/servers/AddServerModal.vue'
import EditServerModal from '@/components/servers/EditServerModal.vue'
import DeleteServerModal from '@/components/servers/DeleteServerModal.vue'
import ServerDetailsModal from '@/components/servers/ServerDetailsModal.vue'

const serversStore = useServersStore()

const showAddModal = ref(false)
const showEditModal = ref(false)
const showDeleteModal = ref(false)
const showDetailsModal = ref(false)
const selectedServer = ref(null)

const filteredServers = computed(() => serversStore.filteredServers)
const pagination = computed(() => serversStore.pagination)
const currentPage = computed(() => serversStore.pagination.page)

onMounted(async () => {
  await serversStore.fetchServers()
})

const handleViewServer = (server) => {
  selectedServer.value = server
  showDetailsModal.value = true
}

const handleEditServer = (server) => {
  selectedServer.value = server
  showEditModal.value = true
}

const handleEditFromDetails = (server) => {
  showDetailsModal.value = false
  selectedServer.value = server
  showEditModal.value = true
}

const handleDeleteServer = (server) => {
  selectedServer.value = server
  showDeleteModal.value = true
}

const handleToggleServer = async (serverId) => {
  try {
    await serversStore.toggleServer(serverId)
  } catch (error) {
    console.error('Failed to toggle server:', error)
  }
}

const handlePageChange = (page) => {
  serversStore.setPage(page)
}

const handleServerAdded = () => {
  showAddModal.value = false
  serversStore.fetchServers()
}

const handleServerUpdated = () => {
  serversStore.fetchServers()
}

const handleServerDeleted = () => {
  serversStore.fetchServers()
}
</script>
