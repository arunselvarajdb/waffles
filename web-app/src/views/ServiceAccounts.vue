<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />
    <div class="max-w-7xl mx-auto py-8 px-4 sm:px-6 lg:px-8">
      <!-- Header -->
      <div class="flex justify-between items-center mb-8">
        <div>
          <h1 class="text-2xl font-bold text-gray-900">Service Accounts</h1>
          <p class="mt-1 text-sm text-gray-500">
            Manage API keys for service accounts and automated systems
          </p>
        </div>
        <BaseButton variant="primary" @click="showCreateModal = true">
          Create Service Account Key
        </BaseButton>
      </div>

      <!-- Filters -->
      <div class="bg-white shadow rounded-lg mb-6 p-4">
        <div class="flex flex-wrap gap-4">
          <div class="flex-1 min-w-[200px]">
            <BaseInput
              v-model="filters.search"
              placeholder="Search by name or user..."
            />
          </div>
          <div>
            <select
              v-model="filters.status"
              class="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            >
              <option value="">All Status</option>
              <option value="active">Active</option>
              <option value="expired">Expired</option>
            </select>
          </div>
        </div>
      </div>

      <!-- Service Accounts Table -->
      <div class="bg-white shadow rounded-lg overflow-hidden">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Name
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Owner
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Scopes
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Status
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Last Used
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Created
              </th>
              <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                Actions
              </th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <tr v-if="loading">
              <td colspan="7" class="px-6 py-12 text-center text-gray-500">
                Loading service accounts...
              </td>
            </tr>
            <tr v-else-if="filteredKeys.length === 0">
              <td colspan="7" class="px-6 py-12 text-center text-gray-500">
                No service accounts found
              </td>
            </tr>
            <tr v-else v-for="key in filteredKeys" :key="key.id" class="hover:bg-gray-50">
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="flex items-center">
                  <div>
                    <div class="text-sm font-medium text-gray-900">{{ key.name }}</div>
                    <div class="text-sm text-gray-500 font-mono">{{ key.key_prefix }}</div>
                  </div>
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900">{{ key.user_email || 'Unknown' }}</div>
                <div class="text-sm text-gray-500 font-mono">{{ key.user_id?.substring(0, 8) }}...</div>
              </td>
              <td class="px-6 py-4">
                <div class="flex flex-wrap gap-1">
                  <BaseBadge
                    v-for="scope in (key.scopes || []).slice(0, 3)"
                    :key="scope"
                    variant="secondary"
                    class="text-xs"
                  >
                    {{ scope }}
                  </BaseBadge>
                  <BaseBadge
                    v-if="(key.scopes || []).length > 3"
                    variant="secondary"
                    class="text-xs"
                  >
                    +{{ key.scopes.length - 3 }}
                  </BaseBadge>
                  <span v-if="!key.scopes || key.scopes.length === 0" class="text-xs text-gray-500">
                    Full access
                  </span>
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <BaseBadge :variant="getStatusVariant(key)">
                  {{ getStatusText(key) }}
                </BaseBadge>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                {{ key.last_used_at ? formatDate(key.last_used_at) : 'Never' }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                {{ formatDate(key.created_at) }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <button
                  @click="handleRevokeKey(key)"
                  class="text-red-600 hover:text-red-900"
                >
                  Revoke
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination placeholder -->
      <div class="mt-4 flex justify-between items-center">
        <p class="text-sm text-gray-500">
          Showing {{ filteredKeys.length }} service account(s)
        </p>
      </div>
    </div>

    <!-- Create Service Account Modal -->
    <CreateApiKeyModal
      v-model="showCreateModal"
      @submit="handleCreateKey"
    />

    <!-- API Key Created Modal -->
    <ApiKeyCreatedModal
      v-model="showKeyCreatedModal"
      :api-key="createdKey"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import NavBar from '@/components/layout/NavBar.vue'
import BaseBadge from '@/components/common/BaseBadge.vue'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseInput from '@/components/common/BaseInput.vue'
import CreateApiKeyModal from '@/components/apikeys/CreateApiKeyModal.vue'
import ApiKeyCreatedModal from '@/components/apikeys/ApiKeyCreatedModal.vue'
import api from '@/services/api'

const loading = ref(false)
const allKeys = ref([])
const filters = ref({
  search: '',
  status: ''
})

const showCreateModal = ref(false)
const showKeyCreatedModal = ref(false)
const createdKey = ref(null)

const filteredKeys = computed(() => {
  let result = allKeys.value

  // Filter by search
  if (filters.value.search) {
    const search = filters.value.search.toLowerCase()
    result = result.filter(key =>
      key.name.toLowerCase().includes(search) ||
      (key.user_email && key.user_email.toLowerCase().includes(search))
    )
  }

  // Filter by status
  if (filters.value.status) {
    const now = new Date()
    result = result.filter(key => {
      const isExpired = key.expires_at && new Date(key.expires_at) < now
      return filters.value.status === 'expired' ? isExpired : !isExpired
    })
  }

  return result
})

onMounted(async () => {
  await fetchAllKeys()
})

const fetchAllKeys = async () => {
  loading.value = true
  try {
    // Admin endpoint to list all API keys across all users
    const response = await api.get('/admin/api-keys')
    allKeys.value = response.api_keys || []
  } catch (error) {
    console.error('Failed to fetch service accounts:', error)
    allKeys.value = []
  } finally {
    loading.value = false
  }
}

const getStatusVariant = (key) => {
  if (key.expires_at && new Date(key.expires_at) < new Date()) {
    return 'error'
  }
  return 'success'
}

const getStatusText = (key) => {
  if (key.expires_at && new Date(key.expires_at) < new Date()) {
    return 'Expired'
  }
  return 'Active'
}

const formatDate = (dateString) => {
  if (!dateString) return 'N/A'
  return new Date(dateString).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}

const handleCreateKey = async (data) => {
  try {
    const response = await api.post('/api-keys', data)
    createdKey.value = response
    showCreateModal.value = false
    showKeyCreatedModal.value = true
    await fetchAllKeys()
  } catch (error) {
    console.error('Failed to create service account key:', error)
  }
}

const handleRevokeKey = async (key) => {
  if (!confirm(`Are you sure you want to revoke the API key "${key.name}"? This action cannot be undone.`)) {
    return
  }

  try {
    await api.delete(`/admin/api-keys/${key.id}`)
    await fetchAllKeys()
  } catch (error) {
    console.error('Failed to revoke API key:', error)
  }
}
</script>
