<template>
  <BaseModal v-model="isOpen" title="API Key Details" size="lg">
    <div class="space-y-6">
      <!-- Key Info -->
      <div>
        <label class="block text-sm font-medium text-gray-700 mb-2">
          Key Identifier
        </label>
        <code class="block bg-gray-100 px-4 py-3 rounded-md font-mono text-sm">
          {{ apiKey.key_prefix }}
        </code>
      </div>

      <!-- Key Details -->
      <div class="bg-gray-50 rounded-md p-4">
        <h4 class="text-sm font-medium text-gray-900 mb-3">Details</h4>
        <dl class="space-y-3 text-sm">
          <div class="flex justify-between">
            <dt class="text-gray-500">Name</dt>
            <dd class="text-gray-900 font-medium">{{ apiKey.name }}</dd>
          </div>
          <div class="flex justify-between">
            <dt class="text-gray-500">Status</dt>
            <dd>
              <span
                :class="[
                  'inline-flex px-2 py-1 text-xs font-medium rounded-full',
                  isExpired ? 'bg-red-100 text-red-800' : 'bg-green-100 text-green-800'
                ]"
              >
                {{ isExpired ? 'Expired' : 'Active' }}
              </span>
              <span
                v-if="apiKey.read_only"
                class="ml-2 inline-flex px-2 py-1 text-xs font-medium bg-yellow-100 text-yellow-800 rounded-full"
              >
                Read Only
              </span>
            </dd>
          </div>
          <div class="flex justify-between">
            <dt class="text-gray-500">Expires</dt>
            <dd class="text-gray-900">{{ apiKey.expires_at ? formatDate(apiKey.expires_at) : 'Never' }}</dd>
          </div>
          <div class="flex justify-between">
            <dt class="text-gray-500">Last Used</dt>
            <dd class="text-gray-900">{{ apiKey.last_used_at ? formatDate(apiKey.last_used_at) : 'Never' }}</dd>
          </div>
          <div class="flex justify-between">
            <dt class="text-gray-500">Created</dt>
            <dd class="text-gray-900">{{ formatDate(apiKey.created_at) }}</dd>
          </div>
        </dl>
      </div>

      <!-- Permissions Section -->
      <div class="border border-gray-200 rounded-md overflow-hidden">
        <div class="bg-gray-50 px-4 py-3 border-b border-gray-200">
          <h4 class="text-sm font-medium text-gray-900">Permissions</h4>
        </div>
        <div class="p-4 space-y-4">
          <!-- Scopes -->
          <div>
            <dt class="text-sm font-medium text-gray-700 mb-2">Scopes</dt>
            <dd>
              <div v-if="hasScopes" class="flex flex-wrap gap-2">
                <span
                  v-for="scope in apiKey.scopes"
                  :key="scope"
                  class="inline-flex px-2 py-1 text-xs font-medium bg-blue-100 text-blue-800 rounded-full"
                >
                  {{ scope }}
                </span>
              </div>
              <span v-else class="text-sm text-gray-500 italic">Full access (no scope restrictions)</span>
            </dd>
          </div>

          <!-- Allowed Servers -->
          <div>
            <dt class="text-sm font-medium text-gray-700 mb-2">Allowed Servers</dt>
            <dd>
              <div v-if="hasAllowedServers" class="flex flex-wrap gap-2">
                <span
                  v-for="server in apiKey.allowed_servers"
                  :key="server"
                  class="inline-flex px-2 py-1 text-xs font-medium bg-purple-100 text-purple-800 rounded-full font-mono"
                >
                  {{ truncateId(server) }}
                </span>
              </div>
              <span v-else class="text-sm text-gray-500 italic">All servers</span>
            </dd>
          </div>

          <!-- Allowed Tools -->
          <div>
            <dt class="text-sm font-medium text-gray-700 mb-2">Allowed Tools</dt>
            <dd>
              <div v-if="hasAllowedTools" class="flex flex-wrap gap-2">
                <span
                  v-for="tool in apiKey.allowed_tools"
                  :key="tool"
                  class="inline-flex px-2 py-1 text-xs font-medium bg-green-100 text-green-800 rounded-full"
                >
                  {{ tool }}
                </span>
              </div>
              <span v-else class="text-sm text-gray-500 italic">All tools</span>
            </dd>
          </div>

          <!-- Namespaces -->
          <div>
            <dt class="text-sm font-medium text-gray-700 mb-2">Namespaces</dt>
            <dd>
              <div v-if="hasNamespaces" class="flex flex-wrap gap-2">
                <span
                  v-for="ns in apiKey.namespaces"
                  :key="ns"
                  class="inline-flex px-2 py-1 text-xs font-medium bg-indigo-100 text-indigo-800 rounded-full font-mono"
                >
                  {{ truncateId(ns) }}
                </span>
              </div>
              <span v-else class="text-sm text-gray-500 italic">All namespaces</span>
            </dd>
          </div>

          <!-- IP Whitelist -->
          <div>
            <dt class="text-sm font-medium text-gray-700 mb-2">IP Whitelist</dt>
            <dd>
              <div v-if="hasIPWhitelist" class="flex flex-wrap gap-2">
                <span
                  v-for="ip in apiKey.ip_whitelist"
                  :key="ip"
                  class="inline-flex px-2 py-1 text-xs font-medium bg-orange-100 text-orange-800 rounded-full font-mono"
                >
                  {{ ip }}
                </span>
              </div>
              <span v-else class="text-sm text-gray-500 italic">Any IP address</span>
            </dd>
          </div>
        </div>
      </div>

      <!-- Note -->
      <div class="bg-gray-50 border border-gray-200 rounded-md p-3">
        <p class="text-xs text-gray-500">
          For security reasons, the full API key is only shown when it is first created.
          If you need a new key, you can revoke this one and create a new one.
        </p>
      </div>

      <!-- Actions -->
      <div class="flex justify-end pt-4 border-t">
        <BaseButton variant="secondary" @click="close">
          Close
        </BaseButton>
      </div>
    </div>
  </BaseModal>
</template>

<script setup>
import { computed } from 'vue'
import BaseModal from '@/components/common/BaseModal.vue'
import BaseButton from '@/components/common/BaseButton.vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  apiKey: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['update:modelValue'])

const isOpen = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const isExpired = computed(() => {
  if (!props.apiKey?.expires_at) return false
  return new Date(props.apiKey.expires_at) <= new Date()
})

// Permission checks
const hasScopes = computed(() => {
  return props.apiKey?.scopes && props.apiKey.scopes.length > 0
})

const hasAllowedServers = computed(() => {
  return props.apiKey?.allowed_servers && props.apiKey.allowed_servers.length > 0
})

const hasAllowedTools = computed(() => {
  return props.apiKey?.allowed_tools && props.apiKey.allowed_tools.length > 0
})

const hasNamespaces = computed(() => {
  return props.apiKey?.namespaces && props.apiKey.namespaces.length > 0
})

const hasIPWhitelist = computed(() => {
  return props.apiKey?.ip_whitelist && props.apiKey.ip_whitelist.length > 0
})

const formatDate = (dateStr) => {
  if (!dateStr) return 'N/A'
  const date = new Date(dateStr)
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

// Truncate UUIDs for display
const truncateId = (id) => {
  if (!id || id.length <= 12) return id
  return id.substring(0, 8) + '...'
}

const close = () => {
  isOpen.value = false
}
</script>
