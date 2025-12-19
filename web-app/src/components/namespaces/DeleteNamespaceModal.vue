<template>
  <BaseModal
    :modelValue="modelValue"
    @update:modelValue="$emit('update:modelValue', $event)"
    title="Delete Namespace"
    size="sm"
  >
    <div class="space-y-4">
      <div class="flex items-center justify-center w-12 h-12 mx-auto bg-red-100 rounded-full">
        <svg class="w-6 h-6 text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
      </div>

      <div class="text-center">
        <p class="text-sm text-gray-500">
          Are you sure you want to delete the namespace
          <span class="font-semibold text-gray-900">{{ namespace?.name }}</span>?
        </p>
        <p class="mt-2 text-sm text-gray-500">
          This will remove all server memberships and role access configurations.
          This action cannot be undone.
        </p>
      </div>

      <div v-if="error" class="p-3 bg-red-50 border border-red-200 rounded-md">
        <p class="text-sm text-red-600">{{ error }}</p>
      </div>
    </div>

    <template #footer>
      <BaseButton variant="secondary" @click="handleClose">
        Cancel
      </BaseButton>
      <BaseButton
        variant="danger"
        @click="handleDelete"
        :disabled="loading"
      >
        {{ loading ? 'Deleting...' : 'Delete Namespace' }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup>
import { ref } from 'vue'
import BaseModal from '@/components/common/BaseModal.vue'
import BaseButton from '@/components/common/BaseButton.vue'
import { useNamespacesStore } from '@/stores/namespaces'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  namespace: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:modelValue', 'success'])

const namespacesStore = useNamespacesStore()

const loading = ref(false)
const error = ref(null)

const handleDelete = async () => {
  if (!props.namespace?.id) return

  loading.value = true
  error.value = null

  try {
    await namespacesStore.deleteNamespace(props.namespace.id)
    emit('success')
    emit('update:modelValue', false)
  } catch (err) {
    console.error('Failed to delete namespace:', err)
    error.value = err.response?.data?.error || 'Failed to delete namespace'
  } finally {
    loading.value = false
  }
}

const handleClose = () => {
  error.value = null
  emit('update:modelValue', false)
}
</script>
