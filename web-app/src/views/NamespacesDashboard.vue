<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Navigation -->
    <NavBar />

    <!-- Main Content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Page Header -->
      <div class="flex justify-between items-center mb-8">
        <div>
          <h2 class="text-2xl font-bold text-gray-900">Namespace Management</h2>
          <p class="mt-1 text-sm text-gray-600">Organize servers into namespaces and manage role-based access</p>
        </div>
        <BaseButton variant="primary" @click="showAddModal = true">
          + Create Namespace
        </BaseButton>
      </div>

      <!-- Statistics -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
        <div class="bg-white rounded-lg shadow p-6">
          <div class="flex items-center">
            <div class="p-3 rounded-full bg-blue-100 text-blue-600">
              <svg class="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
              </svg>
            </div>
            <div class="ml-4">
              <p class="text-sm font-medium text-gray-500">Total Namespaces</p>
              <p class="text-2xl font-semibold text-gray-900">{{ namespaceCount }}</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Search Filter -->
      <div class="bg-white rounded-lg shadow p-4 mb-6">
        <div class="max-w-md">
          <BaseInput
            v-model="searchQuery"
            placeholder="Search namespaces..."
            @input="handleSearch"
          >
            <template #prefix>
              <svg class="h-5 w-5 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
            </template>
          </BaseInput>
        </div>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="bg-white rounded-lg shadow p-12 text-center">
        <p class="text-gray-500">Loading namespaces...</p>
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="bg-white rounded-lg shadow p-12 text-center">
        <p class="text-red-500">{{ error }}</p>
        <BaseButton variant="secondary" class="mt-4" @click="fetchData">
          Retry
        </BaseButton>
      </div>

      <!-- Namespace List -->
      <NamespaceTable
        v-else
        :namespaces="filteredNamespaces"
        @view-namespace="handleViewNamespace"
        @edit-namespace="handleEditNamespace"
        @delete-namespace="handleDeleteNamespace"
      />

      <!-- Pagination -->
      <BasePagination
        v-if="filteredNamespaces.length > 0"
        :current-page="currentPage"
        :total="pagination.total"
        :per-page="pagination.limit"
        @page-change="handlePageChange"
      />
    </main>

    <!-- Modals -->
    <AddNamespaceModal
      v-model="showAddModal"
      @success="handleNamespaceAdded"
    />

    <EditNamespaceModal
      v-model="showEditModal"
      :namespace="selectedNamespace"
      @success="handleNamespaceUpdated"
    />

    <DeleteNamespaceModal
      v-model="showDeleteModal"
      :namespace="selectedNamespace"
      @success="handleNamespaceDeleted"
    />

    <NamespaceDetailsModal
      v-model="showDetailsModal"
      :namespace="selectedNamespace"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useNamespacesStore } from '@/stores/namespaces'
import NavBar from '@/components/layout/NavBar.vue'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseInput from '@/components/common/BaseInput.vue'
import BasePagination from '@/components/common/BasePagination.vue'
import NamespaceTable from '@/components/namespaces/NamespaceTable.vue'
import AddNamespaceModal from '@/components/namespaces/AddNamespaceModal.vue'
import EditNamespaceModal from '@/components/namespaces/EditNamespaceModal.vue'
import DeleteNamespaceModal from '@/components/namespaces/DeleteNamespaceModal.vue'
import NamespaceDetailsModal from '@/components/namespaces/NamespaceDetailsModal.vue'

const namespacesStore = useNamespacesStore()

const showAddModal = ref(false)
const showEditModal = ref(false)
const showDeleteModal = ref(false)
const showDetailsModal = ref(false)
const selectedNamespace = ref(null)
const searchQuery = ref('')

const loading = computed(() => namespacesStore.loading)
const error = computed(() => namespacesStore.error)
const filteredNamespaces = computed(() => namespacesStore.filteredNamespaces)
const namespaceCount = computed(() => namespacesStore.namespaceCount)
const pagination = computed(() => namespacesStore.pagination)
const currentPage = computed(() => namespacesStore.pagination.page)

onMounted(async () => {
  await fetchData()
})

const fetchData = async () => {
  await namespacesStore.fetchNamespaces()
}

const handleSearch = () => {
  namespacesStore.setFilters({ search: searchQuery.value })
}

const handleViewNamespace = (namespace) => {
  selectedNamespace.value = namespace
  showDetailsModal.value = true
}

const handleEditNamespace = (namespace) => {
  selectedNamespace.value = namespace
  showEditModal.value = true
}

const handleDeleteNamespace = (namespace) => {
  selectedNamespace.value = namespace
  showDeleteModal.value = true
}

const handlePageChange = (page) => {
  namespacesStore.setPage(page)
}

const handleNamespaceAdded = () => {
  showAddModal.value = false
  namespacesStore.fetchNamespaces()
}

const handleNamespaceUpdated = () => {
  namespacesStore.fetchNamespaces()
}

const handleNamespaceDeleted = () => {
  namespacesStore.fetchNamespaces()
}
</script>
