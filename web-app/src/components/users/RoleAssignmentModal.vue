<template>
  <BaseModal
    :model-value="modelValue"
    @update:model-value="$emit('update:modelValue', $event)"
    title="Manage User Roles"
    size="sm"
  >
    <div class="space-y-4">
      <!-- User Info -->
      <div v-if="user" class="flex items-center space-x-3 pb-4 border-b">
        <div class="h-10 w-10 rounded-full bg-gray-300 flex items-center justify-center text-gray-600 font-medium">
          {{ getInitials(user.name) }}
        </div>
        <div>
          <p class="text-sm font-medium text-gray-900">{{ user.name }}</p>
          <p class="text-sm text-gray-500">{{ user.email }}</p>
        </div>
      </div>

      <!-- Role Selection -->
      <div class="space-y-3">
        <p class="text-sm font-medium text-gray-700">Select Roles:</p>

        <div
          v-for="role in availableRoles"
          :key="role.name"
          class="flex items-start"
        >
          <div class="flex items-center h-5">
            <input
              :id="`role-${role.name}`"
              v-model="selectedRoles"
              :value="role.name"
              type="checkbox"
              class="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
          </div>
          <div class="ml-3">
            <label :for="`role-${role.name}`" class="text-sm font-medium text-gray-700">
              {{ role.label }}
            </label>
            <p class="text-xs text-gray-500">{{ role.description }}</p>
          </div>
        </div>
      </div>

      <!-- Warning for no roles -->
      <div v-if="selectedRoles.length === 0" class="rounded-md bg-yellow-50 p-3">
        <p class="text-sm text-yellow-700">
          Warning: User will have no roles and limited access
        </p>
      </div>

      <!-- Error Message -->
      <div v-if="error" class="rounded-md bg-red-50 p-3">
        <p class="text-sm text-red-700">{{ error }}</p>
      </div>

      <!-- Actions -->
      <div class="flex justify-end space-x-3 pt-4">
        <BaseButton
          type="button"
          variant="secondary"
          @click="$emit('update:modelValue', false)"
        >
          Cancel
        </BaseButton>
        <BaseButton
          type="button"
          variant="primary"
          :disabled="loading"
          @click="handleSubmit"
        >
          {{ loading ? 'Saving...' : 'Save Roles' }}
        </BaseButton>
      </div>
    </div>
  </BaseModal>
</template>

<script setup>
import { ref, watch } from 'vue'
import BaseModal from '@/components/common/BaseModal.vue'
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
const selectedRoles = ref([])

const availableRoles = [
  {
    name: 'admin',
    label: 'Admin',
    description: 'Full access to all features including user management'
  },
  {
    name: 'operator',
    label: 'Operator',
    description: 'Can manage servers and gateway, view audit logs'
  },
  {
    name: 'viewer',
    label: 'Viewer',
    description: 'Read-only access to servers and health info'
  }
]

// Watch for user changes
watch(() => props.user, (newUser) => {
  if (newUser) {
    selectedRoles.value = [...(newUser.roles || [])]
  }
}, { immediate: true })

// Reset on modal open
watch(() => props.modelValue, (newVal) => {
  if (newVal && props.user) {
    selectedRoles.value = [...(props.user.roles || [])]
    error.value = ''
  }
})

const getInitials = (name) => {
  if (!name) return '?'
  const parts = name.split(' ')
  if (parts.length >= 2) {
    return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
  }
  return name.substring(0, 2).toUpperCase()
}

const handleSubmit = async () => {
  loading.value = true
  error.value = ''

  try {
    await new Promise((resolve, reject) => {
      emit('submit', {
        userId: props.user.id,
        roles: [...selectedRoles.value],
        resolve,
        reject
      })
    })
    emit('update:modelValue', false)
  } catch (err) {
    error.value = err.message || 'Failed to update roles'
  } finally {
    loading.value = false
  }
}
</script>
