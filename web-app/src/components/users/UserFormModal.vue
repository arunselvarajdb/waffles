<template>
  <BaseModal
    :model-value="modelValue"
    @update:model-value="$emit('update:modelValue', $event)"
    :title="isEditing ? 'Edit User' : 'Create User'"
    size="md"
  >
    <form @submit.prevent="handleSubmit" class="space-y-4">
      <!-- Name -->
      <div>
        <label for="name" class="block text-sm font-medium text-gray-700">
          Name <span class="text-red-500">*</span>
        </label>
        <BaseInput
          id="name"
          v-model="form.name"
          type="text"
          placeholder="Enter user's full name"
          required
        />
      </div>

      <!-- Email -->
      <div>
        <label for="email" class="block text-sm font-medium text-gray-700">
          Email <span class="text-red-500">*</span>
        </label>
        <BaseInput
          id="email"
          v-model="form.email"
          type="email"
          placeholder="user@example.com"
          required
          :disabled="isEditing"
        />
        <p v-if="isEditing" class="mt-1 text-xs text-gray-500">
          Email cannot be changed after creation
        </p>
      </div>

      <!-- Password (only for create) -->
      <div v-if="!isEditing">
        <label for="password" class="block text-sm font-medium text-gray-700">
          Password
        </label>
        <BaseInput
          id="password"
          v-model="form.password"
          type="password"
          placeholder="Leave empty to generate temp password"
        />
        <p class="mt-1 text-xs text-gray-500">
          If left empty, a temporary password will be generated
        </p>
      </div>

      <!-- Role (only for create) -->
      <div v-if="!isEditing">
        <label for="role" class="block text-sm font-medium text-gray-700">
          Initial Role
        </label>
        <select
          id="role"
          v-model="form.role"
          class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
        >
          <option value="user">User</option>
          <option value="readonly">Read Only</option>
          <option value="operator">Operator</option>
          <option value="admin">Admin</option>
        </select>
      </div>

      <!-- Active Status (only for edit) -->
      <div v-if="isEditing" class="flex items-center">
        <input
          id="is_active"
          v-model="form.is_active"
          type="checkbox"
          class="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
        />
        <label for="is_active" class="ml-2 block text-sm text-gray-900">
          User is active
        </label>
      </div>

      <!-- Error Message -->
      <div v-if="error" class="rounded-md bg-red-50 p-4">
        <div class="flex">
          <div class="text-sm text-red-700">
            {{ error }}
          </div>
        </div>
      </div>

      <!-- Temp Password Display -->
      <div v-if="tempPassword" class="rounded-md bg-green-50 p-4">
        <div class="flex flex-col">
          <p class="text-sm font-medium text-green-800">User created successfully!</p>
          <p class="text-sm text-green-700 mt-1">Temporary password:</p>
          <code class="mt-1 bg-green-100 px-2 py-1 rounded text-green-900 font-mono">
            {{ tempPassword }}
          </code>
          <p class="text-xs text-green-600 mt-2">
            Please share this password securely with the user. They will be prompted to change it on first login.
          </p>
        </div>
      </div>

      <!-- Actions -->
      <div class="flex justify-end space-x-3 pt-4">
        <BaseButton
          type="button"
          variant="secondary"
          @click="handleCancel"
        >
          {{ tempPassword ? 'Close' : 'Cancel' }}
        </BaseButton>
        <BaseButton
          v-if="!tempPassword"
          type="submit"
          variant="primary"
          :disabled="loading"
        >
          {{ loading ? 'Saving...' : (isEditing ? 'Update' : 'Create') }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import BaseModal from '@/components/common/BaseModal.vue'
import BaseInput from '@/components/common/BaseInput.vue'
import BaseButton from '@/components/common/BaseButton.vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  user: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:modelValue', 'submit'])

const loading = ref(false)
const error = ref('')
const tempPassword = ref('')

const form = ref({
  name: '',
  email: '',
  password: '',
  role: 'user',
  is_active: true
})

const isEditing = computed(() => !!props.user)

// Watch for modal open/close and user changes
watch(() => props.modelValue, (newVal) => {
  if (newVal) {
    resetForm()
  }
})

watch(() => props.user, (newUser) => {
  if (newUser) {
    form.value = {
      name: newUser.name || '',
      email: newUser.email || '',
      password: '',
      role: 'user',
      is_active: newUser.is_active ?? true
    }
  }
}, { immediate: true })

const resetForm = () => {
  error.value = ''
  tempPassword.value = ''
  if (props.user) {
    form.value = {
      name: props.user.name || '',
      email: props.user.email || '',
      password: '',
      role: 'user',
      is_active: props.user.is_active ?? true
    }
  } else {
    form.value = {
      name: '',
      email: '',
      password: '',
      role: 'user',
      is_active: true
    }
  }
}

const handleSubmit = async () => {
  loading.value = true
  error.value = ''

  try {
    const result = await new Promise((resolve, reject) => {
      emit('submit', {
        data: { ...form.value },
        isEditing: isEditing.value,
        resolve,
        reject
      })
    })

    // If creating and temp password returned, show it
    if (!isEditing.value && result?.temp_password) {
      tempPassword.value = result.temp_password
    } else {
      emit('update:modelValue', false)
    }
  } catch (err) {
    error.value = err.message || 'An error occurred'
  } finally {
    loading.value = false
  }
}

const handleCancel = () => {
  emit('update:modelValue', false)
}
</script>
