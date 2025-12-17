<template>
  <BaseModal
    v-model="isOpen"
    title="Edit Server"
    size="lg"
    @close="handleClose"
  >
    <form @submit.prevent="handleSubmit">
      <div class="space-y-4">
        <!-- Basic Info -->
        <BaseInput
          v-model="formData.name"
          label="Server Name"
          placeholder="e.g., Filesystem Server"
          required
        />

        <div>
          <label class="block text-sm font-medium text-gray-700 mb-1">Description</label>
          <textarea
            v-model="formData.description"
            rows="2"
            placeholder="Brief description of this server..."
            class="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:border-blue-500 focus:ring-blue-500"
          />
        </div>

        <!-- Server URL with Test Connection -->
        <div>
          <BaseInput
            v-model="formData.url"
            type="url"
            label="Server URL"
            placeholder="http://localhost:9001"
            required
          />
          <div class="mt-2 flex items-center gap-2">
            <BaseButton
              type="button"
              variant="secondary"
              size="sm"
              :loading="testingConnection"
              :disabled="!formData.url"
              @click="testConnection"
            >
              Test Connection
            </BaseButton>
            <span v-if="connectionTestResult" class="text-sm" :class="connectionTestResult.success ? 'text-green-600' : 'text-red-600'">
              {{ connectionTestResult.success ? 'Connected' : 'Failed' }}
              <span v-if="connectionTestResult.success && connectionTestResult.tool_count !== undefined">
                ({{ connectionTestResult.tool_count }} tools)
              </span>
              <span v-if="connectionTestResult.response_time_ms" class="text-gray-500">
                - {{ connectionTestResult.response_time_ms }}ms
              </span>
            </span>
          </div>
          <!-- Connection test error message -->
          <p v-if="connectionTestResult && !connectionTestResult.success && connectionTestResult.error_message" class="mt-1 text-sm text-red-600">
            {{ connectionTestResult.error_message }}
          </p>
        </div>

        <div class="grid grid-cols-2 gap-4">
          <BaseInput
            v-model="formData.protocol_version"
            label="Protocol Version"
            placeholder="1.0.0"
          />

          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Transport Type</label>
            <select
              v-model="formData.transport"
              class="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:border-blue-500 focus:ring-blue-500"
            >
              <option value="streamable_http">Streamable HTTP (MCP 2025)</option>
              <option value="sse">SSE (Server-Sent Events)</option>
              <option value="http">HTTP (Legacy)</option>
            </select>
            <p class="text-xs text-gray-500 mt-1">Select the transport protocol for this MCP server</p>
          </div>
        </div>

        <!-- Auth Config -->
        <AuthConfigForm
          :auth-type="formData.auth_type"
          :auth-config="formData.auth_config"
          @update:auth-type="formData.auth_type = $event"
          @update:auth-config="formData.auth_config = $event"
          id="edit-auth-form"
        />

        <!-- Health Check Config -->
        <div class="pt-4 border-t border-gray-200">
          <h4 class="text-sm font-medium text-gray-900 mb-3">Health Check Configuration</h4>

          <BaseInput
            v-model="formData.health_check_url"
            type="url"
            label="Health Check URL"
            placeholder="http://localhost:9001/health"
          />

          <div class="grid grid-cols-3 gap-3 mt-3">
            <BaseInput
              v-model.number="formData.health_check_interval"
              type="number"
              label="Check Interval (s)"
              placeholder="60"
            />

            <BaseInput
              v-model.number="formData.timeout"
              type="number"
              label="Timeout (s)"
              placeholder="30"
            />

            <BaseInput
              v-model.number="formData.max_connections"
              type="number"
              label="Max Connections"
              placeholder="10"
            />
          </div>
        </div>

        <!-- Tags -->
        <BaseInput
          v-model="formData.tags"
          label="Tags (optional)"
          placeholder="tag1, tag2, tag3"
          hint="Comma-separated list of tags"
        />

        <!-- Tool Selection (shown after successful connection test) -->
        <div v-if="discoveredTools.length > 0" class="pt-4 border-t border-gray-200">
          <div class="flex items-center justify-between mb-3">
            <h4 class="text-sm font-medium text-gray-900">Available Tools</h4>
            <div class="flex gap-2">
              <button
                type="button"
                class="text-xs text-blue-600 hover:text-blue-800"
                @click="selectAllTools"
              >
                Select All
              </button>
              <span class="text-gray-300">|</span>
              <button
                type="button"
                class="text-xs text-blue-600 hover:text-blue-800"
                @click="deselectAllTools"
              >
                Deselect All
              </button>
            </div>
          </div>
          <p class="text-xs text-gray-500 mb-3">
            Select which tools to expose to users. If none are selected, all tools will be available.
          </p>
          <div class="space-y-2 max-h-48 overflow-y-auto border border-gray-200 rounded-lg p-2">
            <label
              v-for="tool in discoveredTools"
              :key="tool.name"
              class="flex items-start gap-2 p-2 hover:bg-gray-50 rounded cursor-pointer"
            >
              <input
                type="checkbox"
                :value="tool.name"
                v-model="selectedTools"
                class="mt-0.5 h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
              />
              <div class="flex-1 min-w-0">
                <div class="text-sm font-medium text-gray-900">{{ tool.name }}</div>
                <div v-if="tool.description" class="text-xs text-gray-500 truncate">{{ tool.description }}</div>
              </div>
            </label>
          </div>
          <p class="text-xs text-gray-500 mt-2">
            {{ selectedTools.length === 0 ? 'All tools will be available' : `${selectedTools.length} of ${discoveredTools.length} tools selected` }}
          </p>
        </div>

        <!-- Show previously saved allowed_tools if no new test connection -->
        <div v-else-if="formData.allowed_tools && formData.allowed_tools.length > 0" class="pt-4 border-t border-gray-200">
          <h4 class="text-sm font-medium text-gray-900 mb-2">Currently Allowed Tools</h4>
          <p class="text-xs text-gray-500 mb-2">
            Run a connection test to modify allowed tools.
          </p>
          <div class="flex flex-wrap gap-2">
            <span
              v-for="toolName in formData.allowed_tools"
              :key="toolName"
              class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-blue-100 text-blue-800"
            >
              {{ toolName }}
            </span>
          </div>
        </div>
      </div>
    </form>

    <template #footer>
      <BaseButton variant="secondary" @click="handleClose">
        Cancel
      </BaseButton>
      <BaseButton variant="primary" :loading="loading" @click="handleSubmit">
        Save Changes
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup>
import { ref, watch } from 'vue'
import { useServersStore } from '@/stores/servers'
import api from '@/services/api'
import BaseModal from '@/components/common/BaseModal.vue'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseInput from '@/components/common/BaseInput.vue'
import AuthConfigForm from './AuthConfigForm.vue'

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
const testingConnection = ref(false)
const connectionTestResult = ref(null)
const discoveredTools = ref([])
const selectedTools = ref([])

const formData = ref({
  name: '',
  description: '',
  url: '',
  protocol_version: '1.0.0',
  transport: 'streamable_http',
  auth_type: 'none',
  auth_config: {},
  health_check_url: '',
  health_check_interval: 60,
  timeout: 30,
  max_connections: 10,
  tags: ''
})

watch(() => props.modelValue, (newVal) => {
  isOpen.value = newVal
  if (newVal && props.server) {
    populateForm()
  }
})

watch(isOpen, (newVal) => {
  emit('update:modelValue', newVal)
})

const populateForm = () => {
  if (!props.server) return

  formData.value = {
    name: props.server.name || '',
    description: props.server.description || '',
    url: props.server.url || '',
    protocol_version: props.server.protocol_version || '1.0.0',
    transport: props.server.transport || 'streamable_http',
    auth_type: props.server.auth_type || 'none',
    auth_config: props.server.auth_config || {},
    health_check_url: props.server.health_check_url || '',
    health_check_interval: props.server.health_check_interval || 60,
    timeout: props.server.timeout || 30,
    max_connections: props.server.max_connections || 10,
    tags: Array.isArray(props.server.tags) ? props.server.tags.join(', ') : '',
    allowed_tools: props.server.allowed_tools || []
  }
  connectionTestResult.value = null
  discoveredTools.value = []
  selectedTools.value = props.server.allowed_tools || []
}

const testConnection = async () => {
  if (!formData.value.url || testingConnection.value) return

  testingConnection.value = true
  connectionTestResult.value = null
  discoveredTools.value = []

  try {
    const result = await api.post('/servers/test-connection', {
      url: formData.value.url,
      transport: formData.value.transport || 'streamable_http',
      protocol_version: formData.value.protocol_version || '2025-11-25',
      timeout: formData.value.timeout || 10
    })
    connectionTestResult.value = result

    // Extract discovered tools from response
    if (result.success && result.tools && Array.isArray(result.tools)) {
      discoveredTools.value = result.tools.map(tool => ({
        name: tool.name || 'Unknown',
        description: tool.description || ''
      }))
      // Keep previously selected tools that still exist, or select all if none were selected
      const existingToolNames = discoveredTools.value.map(t => t.name)
      const previouslySelected = selectedTools.value.filter(t => existingToolNames.includes(t))
      if (previouslySelected.length > 0) {
        selectedTools.value = previouslySelected
      }
    }
  } catch (error) {
    connectionTestResult.value = {
      success: false,
      error_message: error.response?.data?.error || error.message || 'Connection test failed'
    }
  } finally {
    testingConnection.value = false
  }
}

const selectAllTools = () => {
  selectedTools.value = discoveredTools.value.map(t => t.name)
}

const deselectAllTools = () => {
  selectedTools.value = []
}

const handleClose = () => {
  isOpen.value = false
  connectionTestResult.value = null
}

const handleSubmit = async () => {
  if (!props.server?.id) return

  loading.value = true
  try {
    const payload = {
      ...formData.value,
      tags: formData.value.tags ? formData.value.tags.split(',').map(t => t.trim()) : [],
      allowed_tools: discoveredTools.value.length > 0 ? selectedTools.value : formData.value.allowed_tools
    }

    await serversStore.updateServer(props.server.id, payload)
    emit('success')
    handleClose()
  } catch (error) {
    console.error('Failed to update server:', error)
  } finally {
    loading.value = false
  }
}
</script>
