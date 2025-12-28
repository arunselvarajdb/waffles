<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Navigation -->
    <NavBar />

    <!-- Main Content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Page Header -->
      <div class="flex justify-between items-center mb-8">
        <div>
          <h2 class="text-2xl font-bold text-gray-900">Role Management</h2>
          <p class="mt-1 text-sm text-gray-600">Manage roles and permissions</p>
        </div>
        <BaseButton variant="primary" @click="openCreateModal">
          + Create Role
        </BaseButton>
      </div>

      <!-- Loading State -->
      <div v-if="loading" class="flex justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>

      <!-- Error State -->
      <div v-else-if="error" class="rounded-md bg-red-50 p-4 mb-6">
        <div class="flex">
          <div class="text-sm text-red-700">{{ error }}</div>
          <button @click="fetchRoles" class="ml-auto text-red-600 hover:text-red-800">
            Retry
          </button>
        </div>
      </div>

      <!-- Roles Grid -->
      <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <div
          v-for="role in roles"
          :key="role.id"
          class="bg-white rounded-lg shadow p-6"
        >
          <!-- Role Header -->
          <div class="flex items-start justify-between">
            <div>
              <div class="flex items-center space-x-2">
                <h3 class="text-lg font-medium text-gray-900">{{ role.name }}</h3>
                <BaseBadge v-if="role.is_built_in" variant="secondary" size="sm">
                  Built-in
                </BaseBadge>
              </div>
              <p class="mt-1 text-sm text-gray-500">{{ role.description || 'No description' }}</p>
            </div>
          </div>

          <!-- Role Stats -->
          <div class="mt-4 flex items-center text-sm text-gray-500">
            <svg class="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4.354a4 4 0 110 5.292M15 21H3v-1a6 6 0 0112 0v1zm0 0h6v-1a6 6 0 00-9-5.197"></path>
            </svg>
            {{ role.user_count || 0 }} users
          </div>

          <!-- Actions -->
          <div class="mt-4 flex space-x-3">
            <button
              @click="viewRole(role)"
              class="text-sm text-blue-600 hover:text-blue-800"
            >
              View Details
            </button>
            <button
              v-if="!role.is_built_in"
              @click="editRole(role)"
              class="text-sm text-gray-600 hover:text-gray-800"
            >
              Edit
            </button>
            <button
              v-if="!role.is_built_in"
              @click="confirmDelete(role)"
              class="text-sm text-red-600 hover:text-red-800"
            >
              Delete
            </button>
          </div>
        </div>
      </div>

      <!-- Empty State -->
      <div v-if="!loading && !error && roles.length === 0" class="text-center py-12">
        <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"></path>
        </svg>
        <p class="mt-2 text-sm font-medium text-gray-900">No roles found</p>
        <p class="mt-1 text-sm text-gray-500">Get started by creating a new role.</p>
      </div>
    </main>

    <!-- Role Details Modal -->
    <BaseModal
      v-model="showDetailsModal"
      :title="selectedRole?.name || 'Role Details'"
      size="md"
    >
      <div v-if="selectedRole" class="space-y-4">
        <div>
          <h4 class="text-sm font-medium text-gray-700">Description</h4>
          <p class="mt-1 text-sm text-gray-600">{{ selectedRole.description || 'No description' }}</p>
        </div>

        <div>
          <h4 class="text-sm font-medium text-gray-700">Users with this role</h4>
          <p class="mt-1 text-sm text-gray-600">{{ selectedRole.user_count || 0 }} users</p>
        </div>

        <div v-if="selectedRole.permissions?.length > 0">
          <h4 class="text-sm font-medium text-gray-700">Permissions</h4>
          <div class="mt-2 flex flex-wrap gap-2">
            <BaseBadge
              v-for="perm in selectedRole.permissions"
              :key="perm.id"
              variant="info"
            >
              {{ perm.name }}
            </BaseBadge>
          </div>
        </div>

        <div class="flex justify-end pt-4">
          <BaseButton variant="secondary" @click="showDetailsModal = false">
            Close
          </BaseButton>
        </div>
      </div>
    </BaseModal>

    <!-- Create/Edit Role Modal -->
    <BaseModal
      v-model="showFormModal"
      :title="editingRole ? 'Edit Role' : 'Create Role'"
      size="md"
    >
      <form @submit.prevent="handleSubmit" class="space-y-4">
        <div>
          <label for="name" class="block text-sm font-medium text-gray-700">
            Role Name <span class="text-red-500">*</span>
          </label>
          <BaseInput
            id="name"
            v-model="form.name"
            type="text"
            placeholder="e.g., developer"
            required
            :disabled="editingRole"
          />
        </div>

        <div>
          <label for="description" class="block text-sm font-medium text-gray-700">
            Description
          </label>
          <textarea
            id="description"
            v-model="form.description"
            rows="3"
            class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
            placeholder="Describe what this role allows..."
          ></textarea>
        </div>

        <div v-if="permissions.length > 0">
          <label class="block text-sm font-medium text-gray-700 mb-2">
            Permissions
          </label>
          <div class="max-h-48 overflow-y-auto border rounded-md p-3 space-y-2">
            <div
              v-for="perm in permissions"
              :key="perm.id"
              class="flex items-center"
            >
              <input
                :id="`perm-${perm.id}`"
                v-model="form.permissions"
                :value="perm.id"
                type="checkbox"
                class="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <label :for="`perm-${perm.id}`" class="ml-2 text-sm text-gray-700">
                {{ perm.name }}
                <span v-if="perm.description" class="text-gray-400">- {{ perm.description }}</span>
              </label>
            </div>
          </div>
        </div>

        <div v-if="formError" class="rounded-md bg-red-50 p-3">
          <p class="text-sm text-red-700">{{ formError }}</p>
        </div>

        <div class="flex justify-end space-x-3 pt-4">
          <BaseButton type="button" variant="secondary" @click="showFormModal = false">
            Cancel
          </BaseButton>
          <BaseButton type="submit" variant="primary" :disabled="submitting">
            {{ submitting ? 'Saving...' : (editingRole ? 'Update' : 'Create') }}
          </BaseButton>
        </div>
      </form>
    </BaseModal>

    <!-- Delete Confirmation Modal -->
    <BaseModal
      v-model="showDeleteModal"
      title="Delete Role"
      size="sm"
    >
      <div class="space-y-4">
        <p class="text-sm text-gray-600">
          Are you sure you want to delete the role
          <span class="font-medium">{{ roleToDelete?.name }}</span>?
        </p>
        <p v-if="roleToDelete?.user_count > 0" class="text-sm text-yellow-600">
          Warning: {{ roleToDelete.user_count }} users have this role assigned.
        </p>
        <div class="flex justify-end space-x-3">
          <BaseButton variant="secondary" @click="showDeleteModal = false">
            Cancel
          </BaseButton>
          <BaseButton variant="danger" @click="handleDelete" :disabled="deleting">
            {{ deleting ? 'Deleting...' : 'Delete' }}
          </BaseButton>
        </div>
      </div>
    </BaseModal>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import rolesApi from '@/services/roles'
import NavBar from '@/components/layout/NavBar.vue'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseModal from '@/components/common/BaseModal.vue'
import BaseInput from '@/components/common/BaseInput.vue'
import BaseBadge from '@/components/common/BaseBadge.vue'

// State
const roles = ref([])
const permissions = ref([])
const loading = ref(false)
const error = ref('')

// Modal states
const showDetailsModal = ref(false)
const showFormModal = ref(false)
const showDeleteModal = ref(false)
const selectedRole = ref(null)
const editingRole = ref(null)
const roleToDelete = ref(null)

// Form state
const form = ref({
  name: '',
  description: '',
  permissions: []
})
const formError = ref('')
const submitting = ref(false)
const deleting = ref(false)

// Fetch roles on mount
onMounted(async () => {
  await fetchRoles()
  await fetchPermissions()
})

const fetchRoles = async () => {
  loading.value = true
  error.value = ''
  try {
    const data = await rolesApi.list()
    roles.value = data.roles || data || []
  } catch (err) {
    error.value = err.response?.data?.error || err.message
  } finally {
    loading.value = false
  }
}

const fetchPermissions = async () => {
  try {
    const data = await rolesApi.listPermissions()
    permissions.value = data.permissions || data || []
  } catch (err) {
    console.error('Failed to fetch permissions:', err)
  }
}

const viewRole = async (role) => {
  try {
    const data = await rolesApi.getById(role.id)
    selectedRole.value = data
    showDetailsModal.value = true
  } catch (err) {
    console.error('Failed to fetch role details:', err)
    // Show basic info if fetch fails
    selectedRole.value = role
    showDetailsModal.value = true
  }
}

const openCreateModal = () => {
  editingRole.value = null
  form.value = {
    name: '',
    description: '',
    permissions: []
  }
  formError.value = ''
  showFormModal.value = true
}

const editRole = async (role) => {
  try {
    const data = await rolesApi.getById(role.id)
    editingRole.value = data
    form.value = {
      name: data.role?.name || role.name,
      description: data.role?.description || role.description || '',
      permissions: data.permissions?.map(p => p.id) || []
    }
    formError.value = ''
    showFormModal.value = true
  } catch (err) {
    console.error('Failed to fetch role for editing:', err)
  }
}

const handleSubmit = async () => {
  submitting.value = true
  formError.value = ''

  try {
    if (editingRole.value) {
      await rolesApi.update(editingRole.value.role?.id || editingRole.value.id, {
        description: form.value.description,
        permissions: form.value.permissions
      })
    } else {
      await rolesApi.create({
        name: form.value.name,
        description: form.value.description,
        permissions: form.value.permissions
      })
    }
    showFormModal.value = false
    await fetchRoles()
  } catch (err) {
    formError.value = err.response?.data?.error || err.message
  } finally {
    submitting.value = false
  }
}

const confirmDelete = (role) => {
  roleToDelete.value = role
  showDeleteModal.value = true
}

const handleDelete = async () => {
  deleting.value = true
  try {
    await rolesApi.delete(roleToDelete.value.id)
    showDeleteModal.value = false
    roleToDelete.value = null
    await fetchRoles()
  } catch (err) {
    console.error('Failed to delete role:', err)
  } finally {
    deleting.value = false
  }
}
</script>
