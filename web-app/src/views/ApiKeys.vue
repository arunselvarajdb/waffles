<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Navigation -->
    <NavBar />

    <!-- Main Content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Page Header -->
      <div class="flex justify-between items-center mb-8">
        <div>
          <h2 class="text-2xl font-bold text-gray-900">API Keys</h2>
          <p class="mt-1 text-sm text-gray-600">
            Manage your API keys for programmatic access to the MCP Gateway
          </p>
        </div>
        <BaseButton variant="primary" @click="showCreateModal = true">
          + Create API Key
        </BaseButton>
      </div>

      <!-- Statistics -->
      <ApiKeyStats :stats="keyStats" />

      <!-- Loading State -->
      <div v-if="loading" class="flex justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="rounded-md bg-red-50 p-4 mb-6">
        <div class="flex">
          <div class="text-sm text-red-700">
            {{ error }}
          </div>
          <button @click="fetchApiKeys" class="ml-auto text-red-600 hover:text-red-800">
            Retry
          </button>
        </div>
      </div>

      <!-- API Keys List -->
      <ApiKeyTable
        v-else
        :api-keys="apiKeys"
        @view-key="handleViewKey"
        @delete-key="handleDeleteKey"
      />
    </main>

    <!-- Create API Key Modal -->
    <CreateApiKeyModal
      v-model="showCreateModal"
      @submit="handleCreateKey"
    />

    <!-- Key Created Modal -->
    <ApiKeyCreatedModal
      v-if="newlyCreatedKey"
      v-model="showKeyCreatedModal"
      :api-key="newlyCreatedKey"
      @close="handleKeyCreatedClose"
    />

    <!-- View Key Details Modal -->
    <BaseModal
      v-model="showViewModal"
      title="API Key Details"
      size="md"
    >
      <div v-if="selectedKey" class="space-y-4">
        <dl class="divide-y divide-gray-200">
          <div class="py-3 flex justify-between">
            <dt class="text-sm font-medium text-gray-500">Name</dt>
            <dd class="text-sm text-gray-900">{{ selectedKey.name }}</dd>
          </div>
          <div class="py-3 flex justify-between">
            <dt class="text-sm font-medium text-gray-500">Key Prefix</dt>
            <dd class="text-sm font-mono text-gray-900">{{ selectedKey.key_prefix }}</dd>
          </div>
          <div class="py-3 flex justify-between">
            <dt class="text-sm font-medium text-gray-500">Created</dt>
            <dd class="text-sm text-gray-900">{{ formatDate(selectedKey.created_at) }}</dd>
          </div>
          <div class="py-3 flex justify-between">
            <dt class="text-sm font-medium text-gray-500">Expires</dt>
            <dd class="text-sm text-gray-900">
              {{ selectedKey.expires_at ? formatDate(selectedKey.expires_at) : 'Never' }}
            </dd>
          </div>
          <div class="py-3 flex justify-between">
            <dt class="text-sm font-medium text-gray-500">Last Used</dt>
            <dd class="text-sm text-gray-900">
              {{ selectedKey.last_used_at ? formatDate(selectedKey.last_used_at) : 'Never' }}
            </dd>
          </div>
        </dl>
        <div class="flex justify-end pt-4 border-t">
          <BaseButton variant="secondary" @click="showViewModal = false">
            Close
          </BaseButton>
        </div>
      </div>
    </BaseModal>

    <!-- Delete Confirmation Modal -->
    <BaseModal
      v-model="showDeleteModal"
      title="Revoke API Key"
      size="sm"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-600">
          Are you sure you want to revoke the API key
          <span class="font-medium">{{ selectedKey?.name }}</span>?
        </p>
        <p class="text-sm text-gray-500">
          This action cannot be undone. Any applications using this key will lose access.
        </p>
        <div class="flex justify-end space-x-3">
          <BaseButton variant="secondary" @click="showDeleteModal = false">
            Cancel
          </BaseButton>
          <BaseButton variant="danger" @click="confirmDelete" :disabled="deleting">
            {{ deleting ? 'Revoking...' : 'Revoke Key' }}
          </BaseButton>
        </div>
      </div>
    </BaseModal>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useApiKeysStore } from '@/stores/apikeys'
import NavBar from '@/components/layout/NavBar.vue'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseModal from '@/components/common/BaseModal.vue'
import ApiKeyStats from '@/components/apikeys/ApiKeyStats.vue'
import ApiKeyTable from '@/components/apikeys/ApiKeyTable.vue'
import CreateApiKeyModal from '@/components/apikeys/CreateApiKeyModal.vue'
import ApiKeyCreatedModal from '@/components/apikeys/ApiKeyCreatedModal.vue'

const apiKeysStore = useApiKeysStore()

// Modal states
const showCreateModal = ref(false)
const showKeyCreatedModal = ref(false)
const showViewModal = ref(false)
const showDeleteModal = ref(false)
const selectedKey = ref(null)
const deleting = ref(false)

// Computed properties from store
const loading = computed(() => apiKeysStore.loading)
const error = computed(() => apiKeysStore.error)
const apiKeys = computed(() => apiKeysStore.apiKeys)
const keyStats = computed(() => apiKeysStore.keyStats)
const newlyCreatedKey = computed(() => apiKeysStore.newlyCreatedKey)

// Fetch keys on mount
onMounted(async () => {
  await fetchApiKeys()
})

const fetchApiKeys = async () => {
  await apiKeysStore.fetchApiKeys()
}

const handleCreateKey = async (data) => {
  try {
    await apiKeysStore.createApiKey(data)
    showCreateModal.value = false
    showKeyCreatedModal.value = true
  } catch (err) {
    console.error('Failed to create API key:', err)
  }
}

const handleKeyCreatedClose = () => {
  apiKeysStore.clearNewlyCreatedKey()
  showKeyCreatedModal.value = false
}

const handleViewKey = (key) => {
  selectedKey.value = key
  showViewModal.value = true
}

const handleDeleteKey = (key) => {
  selectedKey.value = key
  showDeleteModal.value = true
}

const confirmDelete = async () => {
  deleting.value = true
  try {
    await apiKeysStore.deleteApiKey(selectedKey.value.id)
    showDeleteModal.value = false
    selectedKey.value = null
  } catch (err) {
    console.error('Failed to delete API key:', err)
  } finally {
    deleting.value = false
  }
}

const formatDate = (dateStr) => {
  if (!dateStr) return 'N/A'
  const date = new Date(dateStr)
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}
</script>
