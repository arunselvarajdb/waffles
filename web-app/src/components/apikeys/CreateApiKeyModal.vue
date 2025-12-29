<template>
  <BaseModal v-model="isOpen" title="Create API Key" size="lg">
    <form @submit.prevent="handleSubmit" class="space-y-6">
      <!-- Name -->
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">
          Name <span class="text-red-500">*</span>
        </label>
        <BaseInput
          v-model="form.name"
          placeholder="e.g., Production Client Key"
          required
          :error="errors.name"
        />
        <p class="mt-1 text-xs text-gray-500">A descriptive name for this API key</p>
      </div>

      <!-- Description -->
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">
          Description
        </label>
        <textarea
          v-model="form.description"
          rows="2"
          class="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
          placeholder="Optional description of what this key is used for"
        ></textarea>
      </div>

      <!-- Expiration -->
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-1">
          Expiration
        </label>
        <select
          v-model="form.expires_in_days"
          class="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
        >
          <option :value="null">Never expires</option>
          <option :value="7">7 days</option>
          <option :value="30">30 days</option>
          <option :value="90">90 days</option>
          <option :value="365">1 year</option>
        </select>
      </div>

      <!-- Scopes -->
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-2">
          Permissions
        </label>
        <p class="text-xs text-gray-500 mb-3">
          Select the permissions for this API key. Leave empty for full access.
        </p>
        <div class="space-y-2 max-h-48 overflow-y-auto border rounded-md p-3 bg-gray-50">
          <label
            v-for="scope in availableScopes"
            :key="scope.value"
            class="flex items-start space-x-3 p-2 hover:bg-white rounded cursor-pointer"
          >
            <input
              type="checkbox"
              :value="scope.value"
              v-model="form.scopes"
              class="mt-0.5 h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <div>
              <span class="text-sm font-medium text-gray-900">{{ scope.label }}</span>
              <p class="text-xs text-gray-500">{{ scope.description }}</p>
            </div>
          </label>
        </div>
      </div>

      <!-- Read Only Toggle -->
      <div class="flex items-center justify-between p-3 bg-gray-50 rounded-md">
        <div>
          <span class="text-sm font-medium text-gray-900">Read-only Access</span>
          <p class="text-xs text-gray-500">Only allow GET requests (no create/update/delete)</p>
        </div>
        <BaseToggle v-model="form.read_only" />
      </div>

      <!-- Advanced Options (collapsed by default, admin only) -->
      <details v-if="authStore.isAdmin" class="border rounded-md">
        <summary class="px-4 py-3 cursor-pointer text-sm font-medium text-gray-700 hover:bg-gray-50">
          Advanced Options
        </summary>
        <div class="px-4 py-3 space-y-4 border-t">
          <!-- IP Whitelist -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">
              IP Whitelist
            </label>
            <BaseInput
              v-model="ipInput"
              placeholder="e.g., 192.168.1.0/24 or 10.0.0.1"
              @keydown.enter.prevent="addIp"
            />
            <p class="mt-1 text-xs text-gray-500">
              Press Enter to add. Leave empty to allow all IPs.
            </p>
            <div v-if="form.ip_whitelist.length" class="mt-2 flex flex-wrap gap-2">
              <span
                v-for="(ip, index) in form.ip_whitelist"
                :key="index"
                class="inline-flex items-center px-2 py-1 text-xs font-medium rounded bg-gray-100 text-gray-800"
              >
                {{ ip }}
                <button
                  type="button"
                  @click="removeIp(index)"
                  class="ml-1 text-gray-500 hover:text-gray-700"
                >
                  &times;
                </button>
              </span>
            </div>
          </div>

          <!-- Allowed Servers -->
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">
              Restrict to Servers
            </label>
            <BaseInput
              v-model="serverInput"
              placeholder="Enter server UUID"
              @keydown.enter.prevent="addServer"
            />
            <p class="mt-1 text-xs text-gray-500">
              Leave empty to allow access to all servers.
            </p>
            <div v-if="form.allowed_servers.length" class="mt-2 flex flex-wrap gap-2">
              <span
                v-for="(server, index) in form.allowed_servers"
                :key="index"
                class="inline-flex items-center px-2 py-1 text-xs font-medium rounded bg-blue-100 text-blue-800"
              >
                {{ server.substring(0, 8) }}...
                <button
                  type="button"
                  @click="removeServer(index)"
                  class="ml-1 text-blue-500 hover:text-blue-700"
                >
                  &times;
                </button>
              </span>
            </div>
          </div>
        </div>
      </details>

      <!-- Actions -->
      <div class="flex justify-end space-x-3 pt-4 border-t">
        <BaseButton variant="secondary" type="button" @click="close">
          Cancel
        </BaseButton>
        <BaseButton variant="primary" type="submit" :disabled="submitting">
          {{ submitting ? 'Creating...' : 'Create API Key' }}
        </BaseButton>
      </div>
    </form>
  </BaseModal>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import BaseModal from '@/components/common/BaseModal.vue'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseInput from '@/components/common/BaseInput.vue'
import BaseToggle from '@/components/common/BaseToggle.vue'
import { getScopesForRole } from '@/services/apikeys'
import { useAuthStore } from '@/stores/auth'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue', 'submit'])

const authStore = useAuthStore()

const isOpen = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

// Filter scopes based on user role using Casbin policy-aligned permissions
const availableScopes = computed(() => {
  return getScopesForRole(authStore.role)
})

const initialForm = () => ({
  name: '',
  description: '',
  expires_in_days: 90,
  scopes: [],
  allowed_servers: [],
  allowed_tools: [],
  namespaces: [],
  ip_whitelist: [],
  read_only: false
})

const form = ref(initialForm())
const errors = ref({})
const submitting = ref(false)
const ipInput = ref('')
const serverInput = ref('')

// Reset form when modal opens
watch(isOpen, (value) => {
  if (value) {
    form.value = initialForm()
    errors.value = {}
    ipInput.value = ''
    serverInput.value = ''
  }
})

const addIp = () => {
  const ip = ipInput.value.trim()
  if (ip && !form.value.ip_whitelist.includes(ip)) {
    form.value.ip_whitelist.push(ip)
    ipInput.value = ''
  }
}

const removeIp = (index) => {
  form.value.ip_whitelist.splice(index, 1)
}

const addServer = () => {
  const server = serverInput.value.trim()
  if (server && !form.value.allowed_servers.includes(server)) {
    form.value.allowed_servers.push(server)
    serverInput.value = ''
  }
}

const removeServer = (index) => {
  form.value.allowed_servers.splice(index, 1)
}

const handleSubmit = async () => {
  errors.value = {}

  if (!form.value.name.trim()) {
    errors.value.name = 'Name is required'
    return
  }

  submitting.value = true

  try {
    const data = {
      name: form.value.name.trim(),
      description: form.value.description.trim() || undefined,
      expires_in_days: form.value.expires_in_days || undefined,
      scopes: form.value.scopes.length ? form.value.scopes : undefined,
      allowed_servers: form.value.allowed_servers.length ? form.value.allowed_servers : undefined,
      allowed_tools: form.value.allowed_tools.length ? form.value.allowed_tools : undefined,
      namespaces: form.value.namespaces.length ? form.value.namespaces : undefined,
      ip_whitelist: form.value.ip_whitelist.length ? form.value.ip_whitelist : undefined,
      read_only: form.value.read_only || undefined
    }

    emit('submit', data)
  } finally {
    submitting.value = false
  }
}

const close = () => {
  isOpen.value = false
}
</script>
