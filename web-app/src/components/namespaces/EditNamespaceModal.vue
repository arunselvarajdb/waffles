<template>
  <BaseModal
    :modelValue="modelValue"
    @update:modelValue="$emit('update:modelValue', $event)"
    title="Edit Namespace"
    size="md"
  >
    <form @submit.prevent="handleSubmit" class="space-y-4">
      <BaseInput
        v-model="formData.name"
        label="Name"
        placeholder="e.g., engineering"
        required
        :error="errors.name"
        hint="A unique identifier for this namespace"
      />

      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">
          Description
        </label>
        <textarea
          v-model="formData.description"
          rows="3"
          class="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
          placeholder="Optional description of this namespace"
        />
      </div>
    </form>

    <template #footer>
      <BaseButton variant="secondary" @click="handleClose">
        Cancel
      </BaseButton>
      <BaseButton
        variant="primary"
        @click="handleSubmit"
        :disabled="loading"
      >
        {{ loading ? 'Saving...' : 'Save Changes' }}
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup>
import { ref, watch } from 'vue'
import BaseModal from '@/components/common/BaseModal.vue'
import BaseInput from '@/components/common/BaseInput.vue'
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
const errors = ref({})
const formData = ref({
  name: '',
  description: ''
})

const resetForm = () => {
  if (props.namespace) {
    formData.value = {
      name: props.namespace.name || '',
      description: props.namespace.description || ''
    }
  }
  errors.value = {}
}

watch(() => props.modelValue, (isOpen) => {
  if (isOpen) {
    resetForm()
  }
})

watch(() => props.namespace, () => {
  if (props.modelValue) {
    resetForm()
  }
})

const validate = () => {
  errors.value = {}

  if (!formData.value.name?.trim()) {
    errors.value.name = 'Name is required'
    return false
  }

  if (formData.value.name.length < 2) {
    errors.value.name = 'Name must be at least 2 characters'
    return false
  }

  return true
}

const handleSubmit = async () => {
  if (!validate()) return

  loading.value = true
  try {
    await namespacesStore.updateNamespace(props.namespace.id, formData.value)
    emit('success')
    emit('update:modelValue', false)
  } catch (error) {
    console.error('Failed to update namespace:', error)
    if (error.response?.data?.error) {
      errors.value.name = error.response.data.error
    }
  } finally {
    loading.value = false
  }
}

const handleClose = () => {
  emit('update:modelValue', false)
}
</script>
