<template>
  <BaseModal
    :model-value="modelValue"
    title="MCP Server Inspector"
    size="xl"
    @update:model-value="$emit('update:modelValue', $event)"
    @close="handleClose"
  >
    <div class="space-y-6">
      <!-- Connection Form -->
      <div class="bg-gray-50 rounded-lg p-4">
        <h3 class="text-sm font-medium text-gray-900 mb-3">Server Connection</h3>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div class="md:col-span-2">
            <BaseInput
              v-model="connectionConfig.url"
              type="url"
              label="Server URL"
              placeholder="http://localhost:9001"
              required
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 mb-1">Transport Type</label>
            <select
              v-model="connectionConfig.transport"
              class="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:border-blue-500 focus:ring-blue-500"
            >
              <option value="streamable_http">Streamable HTTP (MCP 2025)</option>
              <option value="sse">SSE (Server-Sent Events)</option>
              <option value="http">HTTP (Legacy)</option>
            </select>
          </div>
          <div>
            <BaseInput
              v-model="connectionConfig.protocol_version"
              label="Protocol Version"
              placeholder="2025-11-25"
            />
          </div>
        </div>
        <div class="mt-4 flex items-center gap-3">
          <BaseButton
            variant="primary"
            size="sm"
            :loading="connecting"
            :disabled="!connectionConfig.url"
            @click="connect"
          >
            {{ isConnected ? 'Reconnect' : 'Connect' }}
          </BaseButton>
          <span v-if="connectionStatus" class="text-sm" :class="connectionStatus.success ? 'text-green-600' : 'text-red-600'">
            {{ connectionStatus.success ? 'Connected' : 'Failed' }}
            <span v-if="connectionStatus.response_time_ms" class="text-gray-500">
              ({{ connectionStatus.response_time_ms }}ms)
            </span>
          </span>
        </div>
        <p v-if="connectionStatus && !connectionStatus.success && connectionStatus.error_message" class="mt-2 text-sm text-red-600">
          {{ connectionStatus.error_message }}
        </p>
      </div>

      <!-- Server Info -->
      <div v-if="isConnected && serverInfo" class="bg-blue-50 rounded-lg p-4">
        <h3 class="text-sm font-medium text-blue-900 mb-2">Server Information</h3>
        <div class="text-sm text-blue-800">
          <div v-if="serverInfo.name"><span class="font-medium">Name:</span> {{ serverInfo.name }}</div>
          <div v-if="serverInfo.version"><span class="font-medium">Version:</span> {{ serverInfo.version }}</div>
        </div>
      </div>

      <!-- Tabs for Tools/Resources -->
      <div v-if="isConnected">
        <div class="border-b border-gray-200">
          <nav class="-mb-px flex space-x-8">
            <button
              type="button"
              class="py-2 px-1 text-sm font-medium border-b-2 transition-colors"
              :class="activeTab === 'tools' ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'"
              @click="activeTab = 'tools'"
            >
              Tools ({{ tools.length }})
            </button>
            <button
              type="button"
              class="py-2 px-1 text-sm font-medium border-b-2 transition-colors"
              :class="activeTab === 'test' ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'"
              @click="activeTab = 'test'"
            >
              Test Tool
            </button>
          </nav>
        </div>

        <!-- Tools Tab -->
        <div v-show="activeTab === 'tools'" class="mt-4">
          <div v-if="tools.length === 0" class="text-center py-8 text-gray-500">
            No tools discovered from this server.
          </div>
          <div v-else class="space-y-3 max-h-80 overflow-y-auto">
            <div
              v-for="tool in tools"
              :key="tool.name"
              class="border border-gray-200 rounded-lg p-3 hover:bg-gray-50 cursor-pointer"
              @click="selectToolForTest(tool)"
            >
              <div class="flex items-start justify-between">
                <div class="flex-1 min-w-0">
                  <div class="font-medium text-gray-900">{{ tool.name }}</div>
                  <div v-if="tool.description" class="text-sm text-gray-500 mt-1">{{ tool.description }}</div>
                </div>
                <button
                  type="button"
                  class="ml-2 text-xs text-blue-600 hover:text-blue-800 font-medium"
                  @click.stop="selectToolForTest(tool)"
                >
                  Test
                </button>
              </div>
              <!-- Tool Schema Preview -->
              <div v-if="tool.inputSchema && tool.inputSchema.properties" class="mt-2">
                <div class="text-xs text-gray-400 mb-1">Parameters:</div>
                <div class="flex flex-wrap gap-1">
                  <span
                    v-for="(prop, propName) in tool.inputSchema.properties"
                    :key="propName"
                    class="inline-flex items-center px-2 py-0.5 rounded text-xs"
                    :class="tool.inputSchema.required?.includes(propName) ? 'bg-orange-100 text-orange-800' : 'bg-gray-100 text-gray-600'"
                  >
                    {{ propName }}
                    <span class="ml-1 text-gray-400">{{ prop.type }}</span>
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Test Tool Tab -->
        <div v-show="activeTab === 'test'" class="mt-4">
          <div v-if="!selectedTool" class="text-center py-8 text-gray-500">
            Select a tool from the Tools tab to test it.
          </div>
          <div v-else class="space-y-4">
            <div class="bg-gray-50 rounded-lg p-3">
              <div class="font-medium text-gray-900">{{ selectedTool.name }}</div>
              <div v-if="selectedTool.description" class="text-sm text-gray-500 mt-1">{{ selectedTool.description }}</div>
            </div>

            <!-- Dynamic Form for Tool Arguments -->
            <div v-if="selectedTool.inputSchema && selectedTool.inputSchema.properties" class="space-y-3">
              <h4 class="text-sm font-medium text-gray-700">Arguments</h4>
              <div
                v-for="(prop, propName) in selectedTool.inputSchema.properties"
                :key="propName"
              >
                <label class="block text-sm font-medium text-gray-700 mb-1">
                  {{ propName }}
                  <span v-if="selectedTool.inputSchema.required?.includes(propName)" class="text-red-500">*</span>
                  <span class="text-gray-400 font-normal ml-1">({{ prop.type }})</span>
                </label>
                <input
                  v-if="prop.type === 'string' || prop.type === 'number'"
                  v-model="toolArguments[propName]"
                  :type="prop.type === 'number' ? 'number' : 'text'"
                  :placeholder="prop.description || propName"
                  class="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:border-blue-500 focus:ring-blue-500"
                />
                <select
                  v-else-if="prop.type === 'boolean'"
                  v-model="toolArguments[propName]"
                  class="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:border-blue-500 focus:ring-blue-500"
                >
                  <option :value="undefined">Select...</option>
                  <option :value="true">true</option>
                  <option :value="false">false</option>
                </select>
                <textarea
                  v-else
                  v-model="toolArguments[propName]"
                  :placeholder="prop.description || `JSON for ${propName}`"
                  rows="2"
                  class="block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:border-blue-500 focus:ring-blue-500"
                />
                <p v-if="prop.description" class="text-xs text-gray-500 mt-1">{{ prop.description }}</p>
              </div>
            </div>

            <BaseButton
              variant="primary"
              size="sm"
              :loading="callingTool"
              @click="callTool"
            >
              Execute Tool
            </BaseButton>

            <!-- Tool Result -->
            <div v-if="toolResult" class="mt-4">
              <h4 class="text-sm font-medium text-gray-700 mb-2">Result</h4>
              <div
                class="rounded-lg p-3 overflow-auto max-h-60"
                :class="toolResult.success && !toolResult.is_error ? 'bg-green-50 border border-green-200' : 'bg-red-50 border border-red-200'"
              >
                <pre v-if="toolResult.content" class="text-sm whitespace-pre-wrap">{{ formatContent(toolResult.content) }}</pre>
                <p v-if="toolResult.error_message" class="text-sm text-red-600">{{ toolResult.error_message }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Tool Selection for Adding Server -->
      <div v-if="isConnected && tools.length > 0" class="border-t border-gray-200 pt-4">
        <div class="flex items-center justify-between mb-3">
          <h4 class="text-sm font-medium text-gray-900">Select Tools to Expose</h4>
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
        <div class="space-y-2 max-h-40 overflow-y-auto border border-gray-200 rounded-lg p-2">
          <label
            v-for="tool in tools"
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
            </div>
          </label>
        </div>
        <p class="text-xs text-gray-500 mt-2">
          {{ selectedTools.length === 0 ? 'All tools will be available' : `${selectedTools.length} of ${tools.length} tools selected` }}
        </p>
      </div>
    </div>

    <template #footer>
      <BaseButton variant="secondary" @click="handleClose">
        Cancel
      </BaseButton>
      <BaseButton
        variant="primary"
        :disabled="!isConnected"
        @click="proceedToAdd"
      >
        Add Server
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import api from '@/services/api'
import BaseModal from '@/components/common/BaseModal.vue'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseInput from '@/components/common/BaseInput.vue'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue', 'proceed'])

// Connection state
const connectionConfig = ref({
  url: '',
  transport: 'streamable_http',
  protocol_version: '2025-11-25'
})
const connecting = ref(false)
const connectionStatus = ref(null)
const serverInfo = ref(null)
const tools = ref([])

// UI state
const activeTab = ref('tools')
const selectedTool = ref(null)
const toolArguments = ref({})
const callingTool = ref(false)
const toolResult = ref(null)
const selectedTools = ref([])

// Computed
const isConnected = computed(() => connectionStatus.value?.success === true)

// Methods
const connect = async () => {
  if (!connectionConfig.value.url || connecting.value) return

  connecting.value = true
  connectionStatus.value = null
  serverInfo.value = null
  tools.value = []
  selectedTools.value = []

  try {
    const result = await api.post('/servers/test-connection', {
      url: connectionConfig.value.url,
      transport: connectionConfig.value.transport,
      protocol_version: connectionConfig.value.protocol_version || '2025-11-25',
      timeout: 30
    })
    connectionStatus.value = result

    if (result.success) {
      serverInfo.value = result.server_info
      if (result.tools && Array.isArray(result.tools)) {
        tools.value = result.tools
      }
    }
  } catch (error) {
    connectionStatus.value = {
      success: false,
      error_message: error.response?.data?.error || error.message || 'Connection failed'
    }
  } finally {
    connecting.value = false
  }
}

const selectToolForTest = (tool) => {
  selectedTool.value = tool
  toolArguments.value = {}
  toolResult.value = null
  activeTab.value = 'test'
}

const callTool = async () => {
  if (!selectedTool.value || callingTool.value) return

  callingTool.value = true
  toolResult.value = null

  try {
    // Convert string arguments to proper types
    const args = {}
    if (selectedTool.value.inputSchema?.properties) {
      for (const [key, prop] of Object.entries(selectedTool.value.inputSchema.properties)) {
        const value = toolArguments.value[key]
        if (value !== undefined && value !== '') {
          if (prop.type === 'number') {
            args[key] = Number(value)
          } else if (prop.type === 'boolean') {
            args[key] = value === true || value === 'true'
          } else if (prop.type === 'object' || prop.type === 'array') {
            try {
              args[key] = JSON.parse(value)
            } catch {
              args[key] = value
            }
          } else {
            args[key] = value
          }
        }
      }
    }

    const result = await api.post('/servers/call-tool', {
      url: connectionConfig.value.url,
      transport: connectionConfig.value.transport,
      protocol_version: connectionConfig.value.protocol_version || '2025-11-25',
      tool_name: selectedTool.value.name,
      arguments: args,
      timeout: 30
    })
    toolResult.value = result
  } catch (error) {
    toolResult.value = {
      success: false,
      error_message: error.response?.data?.error || error.message || 'Tool call failed'
    }
  } finally {
    callingTool.value = false
  }
}

const formatContent = (content) => {
  if (typeof content === 'string') return content
  if (Array.isArray(content)) {
    return content.map(item => {
      if (typeof item === 'object' && item.text) return item.text
      return JSON.stringify(item, null, 2)
    }).join('\n')
  }
  return JSON.stringify(content, null, 2)
}

const selectAllTools = () => {
  selectedTools.value = tools.value.map(t => t.name)
}

const deselectAllTools = () => {
  selectedTools.value = []
}

const proceedToAdd = () => {
  emit('proceed', {
    url: connectionConfig.value.url,
    transport: connectionConfig.value.transport,
    protocol_version: connectionConfig.value.protocol_version,
    server_info: serverInfo.value,
    tools: tools.value,
    allowed_tools: selectedTools.value
  })
}

const handleClose = () => {
  emit('update:modelValue', false)
}

// Reset when modal opens
watch(() => props.modelValue, (newVal) => {
  if (newVal) {
    connectionConfig.value = {
      url: '',
      transport: 'streamable_http',
      protocol_version: '2025-11-25'
    }
    connectionStatus.value = null
    serverInfo.value = null
    tools.value = []
    selectedTools.value = []
    selectedTool.value = null
    toolArguments.value = {}
    toolResult.value = null
    activeTab.value = 'tools'
  }
})
</script>
