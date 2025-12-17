<template>
  <div class="space-y-4">
    <!-- Auth Type Selection -->
    <div>
      <label class="block text-sm font-medium text-gray-700 mb-2">Authentication Type</label>
      <div class="grid grid-cols-2 gap-3">
        <label class="inline-flex items-center cursor-pointer">
          <input
            type="radio"
            :name="`auth_type_${id}`"
            value="none"
            v-model="localAuthType"
            class="form-radio h-4 w-4 text-blue-600"
          />
          <span class="ml-2 text-sm text-gray-700">None</span>
        </label>
        <label class="inline-flex items-center cursor-pointer">
          <input
            type="radio"
            :name="`auth_type_${id}`"
            value="basic"
            v-model="localAuthType"
            class="form-radio h-4 w-4 text-blue-600"
          />
          <span class="ml-2 text-sm text-gray-700">Basic Auth</span>
        </label>
        <label class="inline-flex items-center cursor-pointer">
          <input
            type="radio"
            :name="`auth_type_${id}`"
            value="bearer"
            v-model="localAuthType"
            class="form-radio h-4 w-4 text-blue-600"
          />
          <span class="ml-2 text-sm text-gray-700">Bearer Token</span>
        </label>
        <label class="inline-flex items-center cursor-pointer">
          <input
            type="radio"
            :name="`auth_type_${id}`"
            value="oauth"
            v-model="localAuthType"
            class="form-radio h-4 w-4 text-blue-600"
          />
          <span class="ml-2 text-sm text-gray-700">OAuth 2.0</span>
        </label>
      </div>
      <p class="mt-1 text-xs text-gray-500">Select authentication method per MCP protocol standards</p>
    </div>

    <!-- Basic Auth Config -->
    <div v-if="localAuthType === 'basic'" class="space-y-3 p-4 bg-gray-50 rounded-md border border-gray-200">
      <div class="text-xs font-medium text-gray-600 mb-2">Basic Authentication Config</div>
      <BaseInput
        v-model="localAuthConfig.username"
        label="Username"
        placeholder="username"
        required
      />
      <BaseInput
        v-model="localAuthConfig.password"
        type="password"
        label="Password"
        placeholder="••••••••"
        required
      />
    </div>

    <!-- Bearer Token Config -->
    <div v-if="localAuthType === 'bearer'" class="p-4 bg-gray-50 rounded-md border border-gray-200">
      <div class="text-xs font-medium text-gray-600 mb-2">Bearer Token Config</div>
      <BaseInput
        v-model="localAuthConfig.token"
        label="Token"
        placeholder="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
        hint="JWT or API token"
        required
      />
    </div>

    <!-- OAuth 2.0 Config -->
    <div v-if="localAuthType === 'oauth'" class="space-y-3 p-4 bg-gray-50 rounded-md border border-gray-200">
      <div class="text-xs font-medium text-gray-600 mb-2">OAuth 2.0 Config</div>
      <BaseInput
        v-model="localAuthConfig.client_id"
        label="Client ID"
        placeholder="your-client-id"
        required
      />
      <BaseInput
        v-model="localAuthConfig.client_secret"
        type="password"
        label="Client Secret"
        placeholder="your-client-secret"
        required
      />
      <BaseInput
        v-model="localAuthConfig.token_url"
        type="url"
        label="Token URL"
        placeholder="https://oauth.example.com/token"
        required
      />
      <BaseInput
        v-model="localAuthConfig.scopes"
        label="Scopes (optional)"
        placeholder="read write"
        hint="Space-separated list of OAuth scopes"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import BaseInput from '@/components/common/BaseInput.vue'

const props = defineProps({
  authType: {
    type: String,
    default: 'none'
  },
  authConfig: {
    type: Object,
    default: () => ({})
  },
  id: {
    type: String,
    default: () => `auth-${Math.random().toString(36).substr(2, 9)}`
  }
})

const emit = defineEmits(['update:authType', 'update:authConfig'])

const localAuthType = ref(props.authType)
const localAuthConfig = ref({ ...props.authConfig })

// Watch for changes and emit to parent
watch(localAuthType, (newType) => {
  emit('update:authType', newType)
  // Clear auth config when type changes
  if (newType === 'none') {
    localAuthConfig.value = {}
  } else if (newType === 'basic') {
    localAuthConfig.value = { username: '', password: '' }
  } else if (newType === 'bearer') {
    localAuthConfig.value = { token: '' }
  } else if (newType === 'oauth') {
    localAuthConfig.value = { client_id: '', client_secret: '', token_url: '', scopes: '' }
  }
  emit('update:authConfig', localAuthConfig.value)
})

watch(localAuthConfig, (newConfig) => {
  emit('update:authConfig', newConfig)
}, { deep: true })

// Watch for external changes
watch(() => props.authType, (newType) => {
  localAuthType.value = newType
})

watch(() => props.authConfig, (newConfig) => {
  localAuthConfig.value = { ...newConfig }
}, { deep: true })
</script>
