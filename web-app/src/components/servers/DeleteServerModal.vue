<template>
  <BaseModal
    v-model="isOpen"
    title="Delete Server"
    size="sm"
    @close="handleClose"
  >
    <div class="text-sm text-gray-600">
      <p class="mb-4">
        Are you sure you want to delete
        <span class="font-semibold text-gray-900">{{ server?.name }}</span>?
      </p>
      <p class="text-red-600">This action cannot be undone.</p>
    </div>

    <template #footer>
      <BaseButton variant="secondary" @click="handleClose">
        Cancel
      </BaseButton>
      <BaseButton variant="danger" :loading="loading" @click="handleDelete">
        Delete Server
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useServersStore } from '@/stores/servers'
import BaseModal from '@/components/common/BaseModal.vue'
import BaseButton from '@/components/common/BaseButton.vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  server: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['update:modelValue', 'success'])

const serversStore = useServersStore()

const isOpen = ref(props.modelValue)
const loading = ref(false)

watch(() => props.modelValue, (newVal) => {
  isOpen.value = newVal
})

watch(isOpen, (newVal) => {
  emit('update:modelValue', newVal)
})

const handleClose = () => {
  isOpen.value = false
}

const handleDelete = async () => {
  if (!props.server?.id) return

  loading.value = true
  try {
    await serversStore.deleteServer(props.server.id)
    emit('success')
    handleClose()
  } catch (error) {
    console.error('Failed to delete server:', error)
  } finally {
    loading.value = false
  }
}
</script>
