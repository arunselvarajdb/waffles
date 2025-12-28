<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Navigation -->
    <NavBar />

    <!-- Main Content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Page Header -->
      <div class="flex justify-between items-center mb-8">
        <div>
          <h2 class="text-2xl font-bold text-gray-900">User Management</h2>
          <p class="mt-1 text-sm text-gray-600">Manage users and their access permissions</p>
        </div>
        <BaseButton variant="primary" @click="showCreateModal = true">
          + Add User
        </BaseButton>
      </div>

      <!-- Statistics -->
      <UserStats :stats="userStats" />

      <!-- Filters -->
      <UserFilters
        :filters="filters"
        @update:filters="handleFiltersUpdate"
      />

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
          <button @click="fetchUsers" class="ml-auto text-red-600 hover:text-red-800">
            Retry
          </button>
        </div>
      </div>

      <!-- User List -->
      <UserTable
        v-else
        :users="filteredUsers"
        @edit-user="handleEditUser"
        @manage-roles="handleManageRoles"
        @reset-password="handleResetPassword"
        @deactivate-user="handleDeactivateUser"
      />

      <!-- Pagination -->
      <BasePagination
        v-if="filteredUsers.length > 0"
        :current-page="pagination.page"
        :total="pagination.total"
        :per-page="pagination.pageSize"
        @page-change="handlePageChange"
        class="mt-6"
      />
    </main>

    <!-- Create/Edit User Modal -->
    <UserFormModal
      v-model="showCreateModal"
      :user="selectedUser"
      @submit="handleUserSubmit"
    />

    <!-- Role Assignment Modal -->
    <RoleAssignmentModal
      v-model="showRolesModal"
      :user="selectedUser"
      @submit="handleRolesSubmit"
    />

    <!-- Password Reset Confirmation -->
    <BaseModal
      v-model="showResetPasswordModal"
      title="Reset Password"
      size="sm"
    >
      <div v-if="!tempPassword" class="space-y-4">
        <p class="text-sm text-gray-600">
          Are you sure you want to reset the password for
          <span class="font-medium">{{ selectedUser?.name }}</span>?
        </p>
        <p class="text-sm text-gray-500">
          A new temporary password will be generated.
        </p>
        <div class="flex justify-end space-x-3">
          <BaseButton variant="secondary" @click="showResetPasswordModal = false">
            Cancel
          </BaseButton>
          <BaseButton variant="warning" @click="confirmResetPassword" :disabled="resetting">
            {{ resetting ? 'Resetting...' : 'Reset Password' }}
          </BaseButton>
        </div>
      </div>
      <div v-else class="space-y-4">
        <p class="text-sm text-green-700 font-medium">Password reset successfully!</p>
        <p class="text-sm text-gray-600">New temporary password:</p>
        <code class="block bg-gray-100 px-3 py-2 rounded font-mono text-sm">
          {{ tempPassword }}
        </code>
        <p class="text-xs text-gray-500">
          Please share this password securely with the user.
        </p>
        <div class="flex justify-end">
          <BaseButton variant="primary" @click="closeResetModal">
            Done
          </BaseButton>
        </div>
      </div>
    </BaseModal>

    <!-- Deactivate Confirmation -->
    <BaseModal
      v-model="showDeactivateModal"
      title="Deactivate User"
      size="sm"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-600">
          Are you sure you want to deactivate
          <span class="font-medium">{{ selectedUser?.name }}</span>?
        </p>
        <p class="text-sm text-gray-500">
          The user will no longer be able to log in or access any resources.
        </p>
        <div class="flex justify-end space-x-3">
          <BaseButton variant="secondary" @click="showDeactivateModal = false">
            Cancel
          </BaseButton>
          <BaseButton variant="danger" @click="confirmDeactivate" :disabled="deactivating">
            {{ deactivating ? 'Deactivating...' : 'Deactivate' }}
          </BaseButton>
        </div>
      </div>
    </BaseModal>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useUsersStore } from '@/stores/users'
import NavBar from '@/components/layout/NavBar.vue'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseModal from '@/components/common/BaseModal.vue'
import BasePagination from '@/components/common/BasePagination.vue'
import UserStats from '@/components/users/UserStats.vue'
import UserFilters from '@/components/users/UserFilters.vue'
import UserTable from '@/components/users/UserTable.vue'
import UserFormModal from '@/components/users/UserFormModal.vue'
import RoleAssignmentModal from '@/components/users/RoleAssignmentModal.vue'

const usersStore = useUsersStore()

// Modal states
const showCreateModal = ref(false)
const showRolesModal = ref(false)
const showResetPasswordModal = ref(false)
const showDeactivateModal = ref(false)
const selectedUser = ref(null)
const tempPassword = ref('')
const resetting = ref(false)
const deactivating = ref(false)

// Computed properties from store
const loading = computed(() => usersStore.loading)
const error = computed(() => usersStore.error)
const filteredUsers = computed(() => usersStore.filteredUsers)
const pagination = computed(() => usersStore.pagination)
const filters = computed(() => usersStore.filters)
const userStats = computed(() => usersStore.userStats)

// Fetch users on mount
onMounted(async () => {
  await fetchUsers()
})

const fetchUsers = async () => {
  await usersStore.fetchUsers()
}

const handleFiltersUpdate = (newFilters) => {
  usersStore.setFilters(newFilters)
}

const handlePageChange = (page) => {
  usersStore.setPage(page)
}

const handleEditUser = (user) => {
  selectedUser.value = user
  showCreateModal.value = true
}

const handleUserSubmit = async ({ data, isEditing, resolve, reject }) => {
  try {
    let result
    if (isEditing) {
      result = await usersStore.updateUser(selectedUser.value.id, {
        name: data.name,
        is_active: data.is_active
      })
    } else {
      result = await usersStore.createUser({
        email: data.email,
        name: data.name,
        password: data.password || undefined,
        role: data.role
      })
    }
    resolve(result)
    if (isEditing) {
      showCreateModal.value = false
    }
    selectedUser.value = null
  } catch (err) {
    reject(err)
  }
}

const handleManageRoles = (user) => {
  selectedUser.value = user
  showRolesModal.value = true
}

const handleRolesSubmit = async ({ userId, roles, resolve, reject }) => {
  try {
    await usersStore.updateUserRoles(userId, roles)
    resolve()
  } catch (err) {
    reject(err)
  }
}

const handleResetPassword = (user) => {
  selectedUser.value = user
  tempPassword.value = ''
  showResetPasswordModal.value = true
}

const confirmResetPassword = async () => {
  resetting.value = true
  try {
    const result = await usersStore.resetUserPassword(selectedUser.value.id)
    tempPassword.value = result.temp_password
  } catch (err) {
    console.error('Failed to reset password:', err)
  } finally {
    resetting.value = false
  }
}

const closeResetModal = () => {
  showResetPasswordModal.value = false
  tempPassword.value = ''
  selectedUser.value = null
}

const handleDeactivateUser = (user) => {
  selectedUser.value = user
  showDeactivateModal.value = true
}

const confirmDeactivate = async () => {
  deactivating.value = true
  try {
    await usersStore.deactivateUser(selectedUser.value.id)
    showDeactivateModal.value = false
    selectedUser.value = null
  } catch (err) {
    console.error('Failed to deactivate user:', err)
  } finally {
    deactivating.value = false
  }
}
</script>
