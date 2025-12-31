<template>
  <BaseModal v-model="isOpen" title="API Key Created" size="md" :closable="false">
    <div class="space-y-6">
      <!-- Success Message -->
      <div class="flex items-center space-x-3 text-green-600">
        <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <span class="font-medium">Your API key has been created successfully!</span>
      </div>

      <!-- Warning -->
      <div class="bg-yellow-50 border border-yellow-200 rounded-md p-4">
        <div class="flex">
          <svg class="h-5 w-5 text-yellow-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div class="ml-3">
            <h3 class="text-sm font-medium text-yellow-800">
              Save this key now!
            </h3>
            <p class="text-sm text-yellow-700 mt-1">
              This is the only time you will see this key. Make sure to copy it and store it securely.
            </p>
          </div>
        </div>
      </div>

      <!-- Key Display -->
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-2">
          Your API Key
        </label>
        <div class="flex items-center space-x-2">
          <code class="flex-1 bg-gray-100 px-4 py-3 rounded-md font-mono text-sm break-all select-all">
            {{ apiKey.key }}
          </code>
          <button
            @click="copyKey"
            class="flex-shrink-0 inline-flex items-center px-3 py-2 border border-gray-300 shadow-sm text-sm leading-4 font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
          >
            <svg v-if="!copied" class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" />
            </svg>
            <svg v-else class="h-4 w-4 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
            <span class="ml-1">{{ copied ? 'Copied!' : 'Copy' }}</span>
          </button>
        </div>
      </div>

      <!-- Key Details -->
      <div class="bg-gray-50 rounded-md p-4">
        <h4 class="text-sm font-medium text-gray-900 mb-3">Key Details</h4>
        <dl class="space-y-2 text-sm">
          <div class="flex justify-between">
            <dt class="text-gray-500">Name</dt>
            <dd class="text-gray-900 font-medium">{{ apiKey.name }}</dd>
          </div>
          <div class="flex justify-between">
            <dt class="text-gray-500">Expires</dt>
            <dd class="text-gray-900">{{ apiKey.expires_at ? formatDate(apiKey.expires_at) : 'Never' }}</dd>
          </div>
          <div class="flex justify-between">
            <dt class="text-gray-500">Created</dt>
            <dd class="text-gray-900">{{ formatDate(apiKey.created_at) }}</dd>
          </div>
        </dl>
      </div>

      <!-- Actions -->
      <div class="flex justify-end pt-4 border-t">
        <BaseButton variant="primary" @click="close">
          I've saved my key
        </BaseButton>
      </div>
    </div>
  </BaseModal>
</template>

<script setup>
import { ref, computed } from 'vue'
import BaseModal from '@/components/common/BaseModal.vue'
import BaseButton from '@/components/common/BaseButton.vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  apiKey: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['update:modelValue', 'close'])

const isOpen = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const copied = ref(false)

const copyKey = async () => {
  try {
    await navigator.clipboard.writeText(props.apiKey.key)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch (err) {
    console.error('Failed to copy:', err)
  }
}

const formatDate = (dateStr) => {
  if (!dateStr) return 'N/A'
  const date = new Date(dateStr)
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}

const close = () => {
  isOpen.value = false
  emit('close')
}
</script>
