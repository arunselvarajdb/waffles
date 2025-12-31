<template>
  <div class="min-h-screen bg-gray-50">
    <NavBar />
    <div class="max-w-4xl mx-auto py-8 px-4 sm:px-6 lg:px-8">
      <!-- Header -->
      <div class="mb-8">
        <h1 class="text-2xl font-bold text-gray-900">Account Settings</h1>
        <p class="mt-1 text-sm text-gray-500">Manage your profile, security settings, and API keys</p>
      </div>

      <!-- Tabs -->
      <div class="border-b border-gray-200 mb-6">
        <nav class="-mb-px flex space-x-8">
          <button
            v-for="tab in tabs"
            :key="tab.id"
            @click="activeTab = tab.id"
            :class="[
              activeTab === tab.id
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300',
              'whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm'
            ]"
          >
            {{ tab.name }}
          </button>
        </nav>
      </div>

      <!-- Tab Content -->
      <div class="bg-white shadow rounded-lg">
        <!-- Profile Tab -->
        <div v-if="activeTab === 'profile'" class="p-6">
          <h2 class="text-lg font-medium text-gray-900 mb-4">Profile Information</h2>
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-gray-700">Email</label>
              <p class="mt-1 text-sm text-gray-900">{{ authStore.user?.email || 'Not available' }}</p>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700">Role</label>
              <BaseBadge :variant="authStore.isAdmin ? 'info' : 'secondary'" class="mt-1">
                {{ authStore.role || 'User' }}
              </BaseBadge>
            </div>
            <div>
              <label class="block text-sm font-medium text-gray-700">User ID</label>
              <p class="mt-1 text-sm text-gray-500 font-mono">{{ authStore.user?.id || 'Not available' }}</p>
            </div>
          </div>
        </div>

        <!-- Security Tab -->
        <div v-if="activeTab === 'security'" class="p-6">
          <h2 class="text-lg font-medium text-gray-900 mb-4">Security Settings</h2>

          <!-- Password Change Section - Only for local/LDAP auth -->
          <div v-if="authStore.canChangePassword" class="border-b border-gray-200 pb-6 mb-6">
            <h3 class="text-sm font-medium text-gray-900 mb-4">Change Password</h3>
            <form @submit.prevent="handlePasswordChange" class="space-y-4 max-w-md">
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">Current Password</label>
                <BaseInput
                  v-model="passwordForm.currentPassword"
                  type="password"
                  placeholder="Enter current password"
                  :error="passwordErrors.currentPassword"
                />
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">New Password</label>
                <BaseInput
                  v-model="passwordForm.newPassword"
                  type="password"
                  placeholder="Enter new password"
                  :error="passwordErrors.newPassword"
                />
                <p class="mt-1 text-xs text-gray-500">Minimum 8 characters</p>
              </div>
              <div>
                <label class="block text-sm font-medium text-gray-700 mb-1">Confirm New Password</label>
                <BaseInput
                  v-model="passwordForm.confirmPassword"
                  type="password"
                  placeholder="Confirm new password"
                  :error="passwordErrors.confirmPassword"
                />
              </div>
              <div class="flex items-center space-x-4">
                <BaseButton type="submit" variant="primary" :disabled="changingPassword">
                  {{ changingPassword ? 'Changing...' : 'Change Password' }}
                </BaseButton>
                <span v-if="passwordSuccess" class="text-sm text-green-600">Password changed successfully</span>
              </div>
            </form>
          </div>

          <!-- OAuth notice -->
          <div v-else-if="authStore.authProvider === 'oauth'" class="border-b border-gray-200 pb-6 mb-6">
            <h3 class="text-sm font-medium text-gray-900 mb-2">Password Management</h3>
            <p class="text-sm text-gray-500">
              Your account is managed through Single Sign-On (SSO).
              Please contact your identity provider administrator to change your password.
            </p>
          </div>

          <!-- Active Sessions Section (placeholder for future) -->
          <div>
            <h3 class="text-sm font-medium text-gray-900 mb-2">Active Sessions</h3>
            <p class="text-sm text-gray-500">Session management coming soon.</p>
          </div>
        </div>

        <!-- API Keys Tab -->
        <div v-if="activeTab === 'api-keys'" class="p-6">
          <div class="flex justify-between items-center mb-6">
            <div>
              <h2 class="text-lg font-medium text-gray-900">API Keys</h2>
              <p class="mt-1 text-sm text-gray-500">Manage your personal API keys for programmatic access</p>
            </div>
            <BaseButton variant="primary" @click="showCreateModal = true">
              Create API Key
            </BaseButton>
          </div>

          <!-- API Key Stats -->
          <ApiKeyStats :keys="apiKeyStore.apiKeys" class="mb-6" />

          <!-- API Keys Table -->
          <ApiKeyTable
            :api-keys="apiKeyStore.apiKeys"
            :loading="apiKeyStore.loading"
            @view-key="handleViewKey"
            @delete-key="handleDeleteKey"
          />
        </div>
      </div>
    </div>

    <!-- Create API Key Modal -->
    <CreateApiKeyModal
      v-model="showCreateModal"
      @submit="handleCreateKey"
    />

    <!-- API Key Created Modal -->
    <ApiKeyCreatedModal
      v-model="showKeyCreatedModal"
      :api-key="createdKey"
    />

    <!-- View API Key Modal -->
    <ViewApiKeyModal
      v-model="showViewModal"
      :api-key="selectedKey"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useApiKeysStore } from '@/stores/apikeys'
import NavBar from '@/components/layout/NavBar.vue'
import BaseBadge from '@/components/common/BaseBadge.vue'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseInput from '@/components/common/BaseInput.vue'
import ApiKeyStats from '@/components/apikeys/ApiKeyStats.vue'
import ApiKeyTable from '@/components/apikeys/ApiKeyTable.vue'
import CreateApiKeyModal from '@/components/apikeys/CreateApiKeyModal.vue'
import ApiKeyCreatedModal from '@/components/apikeys/ApiKeyCreatedModal.vue'
import ViewApiKeyModal from '@/components/apikeys/ViewApiKeyModal.vue'
import api from '@/services/api'

const authStore = useAuthStore()
const apiKeyStore = useApiKeysStore()

const tabs = [
  { id: 'profile', name: 'Profile' },
  { id: 'security', name: 'Security' },
  { id: 'api-keys', name: 'API Keys' }
]

const activeTab = ref('profile')

// Password change state
const passwordForm = ref({
  currentPassword: '',
  newPassword: '',
  confirmPassword: ''
})
const passwordErrors = ref({})
const changingPassword = ref(false)
const passwordSuccess = ref(false)

// API Key state
const showCreateModal = ref(false)
const showKeyCreatedModal = ref(false)
const showViewModal = ref(false)
const createdKey = ref(null)
const selectedKey = ref({})

onMounted(() => {
  // Check if we should show a specific tab based on route query
  const urlParams = new URLSearchParams(window.location.search)
  const tab = urlParams.get('tab')
  if (tab && tabs.some(t => t.id === tab)) {
    activeTab.value = tab
  }

  // Load API keys when component mounts
  apiKeyStore.fetchApiKeys()
})

const handlePasswordChange = async () => {
  passwordErrors.value = {}
  passwordSuccess.value = false

  // Validation
  if (!passwordForm.value.currentPassword) {
    passwordErrors.value.currentPassword = 'Current password is required'
    return
  }
  if (!passwordForm.value.newPassword) {
    passwordErrors.value.newPassword = 'New password is required'
    return
  }
  if (passwordForm.value.newPassword.length < 8) {
    passwordErrors.value.newPassword = 'Password must be at least 8 characters'
    return
  }
  if (passwordForm.value.newPassword !== passwordForm.value.confirmPassword) {
    passwordErrors.value.confirmPassword = 'Passwords do not match'
    return
  }

  changingPassword.value = true
  try {
    await api.post('/auth/change-password', {
      current_password: passwordForm.value.currentPassword,
      new_password: passwordForm.value.newPassword
    })
    passwordSuccess.value = true
    passwordForm.value = { currentPassword: '', newPassword: '', confirmPassword: '' }
  } catch (error) {
    if (error.response?.status === 400) {
      passwordErrors.value.currentPassword = 'Current password is incorrect'
    } else {
      passwordErrors.value.currentPassword = 'Failed to change password'
    }
  } finally {
    changingPassword.value = false
  }
}

const handleCreateKey = async (data) => {
  try {
    const key = await apiKeyStore.createApiKey(data)
    createdKey.value = key
    showCreateModal.value = false
    showKeyCreatedModal.value = true
  } catch (error) {
    console.error('Failed to create API key:', error)
  }
}

const handleViewKey = (key) => {
  selectedKey.value = key
  showViewModal.value = true
}

const handleDeleteKey = async (key) => {
  if (confirm(`Are you sure you want to delete the API key "${key.name}"? This action cannot be undone.`)) {
    await apiKeyStore.deleteApiKey(key.id)
  }
}
</script>
