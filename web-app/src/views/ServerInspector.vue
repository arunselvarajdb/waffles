<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Navigation -->
    <NavBar />

    <!-- Main Content -->
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <!-- Page Header -->
      <div class="mb-8">
        <h2 class="text-2xl font-bold text-gray-900">Server Inspector</h2>
        <p class="mt-1 text-sm text-gray-600">Connect to an MCP server to discover its capabilities before adding it to the gateway</p>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <!-- Left Panel: Connection Form -->
        <div class="lg:col-span-1">
          <div class="bg-white rounded-lg shadow p-6">
            <h3 class="text-lg font-medium text-gray-900 mb-4">Connection Settings</h3>

            <div class="space-y-4">
              <BaseInput
                v-model="connectionConfig.url"
                type="url"
                label="Server URL"
                placeholder="http://localhost:9001/mcp"
                required
              />

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

              <BaseInput
                v-model="connectionConfig.protocol_version"
                label="Protocol Version"
                placeholder="2025-11-25"
              />

              <BaseButton
                variant="primary"
                class="w-full"
                :loading="connecting"
                :disabled="!connectionConfig.url"
                @click="connect"
              >
                {{ isConnected ? 'Reconnect' : 'Connect' }}
              </BaseButton>

              <!-- Connection Status -->
              <div v-if="connectionStatus" class="mt-4">
                <div
                  class="rounded-lg p-3"
                  :class="connectionStatus.success ? 'bg-green-50 border border-green-200' : 'bg-red-50 border border-red-200'"
                >
                  <div class="flex items-center">
                    <span
                      class="h-2 w-2 rounded-full mr-2"
                      :class="connectionStatus.success ? 'bg-green-500' : 'bg-red-500'"
                    ></span>
                    <span class="text-sm font-medium" :class="connectionStatus.success ? 'text-green-800' : 'text-red-800'">
                      {{ connectionStatus.success ? 'Connected' : 'Connection Failed' }}
                    </span>
                    <span v-if="connectionStatus.response_time_ms" class="ml-2 text-xs text-gray-500">
                      ({{ connectionStatus.response_time_ms }}ms)
                    </span>
                  </div>
                  <p v-if="!connectionStatus.success && connectionStatus.error_message" class="mt-1 text-sm text-red-600">
                    {{ connectionStatus.error_message }}
                  </p>
                </div>
              </div>

              <!-- Server Info -->
              <div v-if="isConnected && serverInfo" class="bg-blue-50 rounded-lg p-4 mt-4">
                <h4 class="text-sm font-medium text-blue-900 mb-2">Server Information</h4>
                <div class="text-sm text-blue-800 space-y-1">
                  <div v-if="serverInfo.name"><span class="font-medium">Name:</span> {{ serverInfo.name }}</div>
                  <div v-if="serverInfo.version"><span class="font-medium">Version:</span> {{ serverInfo.version }}</div>
                </div>
              </div>
            </div>
          </div>

          <!-- Add Server Button -->
          <div v-if="isConnected" class="mt-6">
            <BaseButton
              variant="primary"
              class="w-full"
              @click="proceedToAddServer"
            >
              Add This Server to Gateway
            </BaseButton>
          </div>
        </div>

        <!-- Right Panel: Tools & Test -->
        <div class="lg:col-span-2">
          <div class="bg-white rounded-lg shadow">
            <!-- Tabs -->
            <div class="border-b border-gray-200">
              <nav class="flex -mb-px">
                <button
                  type="button"
                  class="py-4 px-6 text-sm font-medium border-b-2 transition-colors"
                  :class="activeTab === 'tools' ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'"
                  @click="activeTab = 'tools'"
                >
                  Tools ({{ tools.length }})
                </button>
                <button
                  type="button"
                  class="py-4 px-6 text-sm font-medium border-b-2 transition-colors"
                  :class="activeTab === 'test' ? 'border-blue-500 text-blue-600' : 'border-transparent text-gray-500 hover:text-gray-700'"
                  @click="activeTab = 'test'"
                >
                  Test Tool
                </button>
              </nav>
            </div>

            <div class="p-6">
              <!-- Tools Tab -->
              <div v-show="activeTab === 'tools'">
                <div v-if="!isConnected" class="text-center py-12 text-gray-500">
                  <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
                  </svg>
                  <p class="mt-4">Connect to a server to discover its tools</p>
                </div>

                <div v-else-if="tools.length === 0" class="text-center py-12 text-gray-500">
                  No tools discovered from this server.
                </div>

                <div v-else class="space-y-3">
                  <!-- Tool Selection Header -->
                  <div class="flex items-center justify-between mb-4">
                    <p class="text-sm text-gray-600">
                      Select tools to expose ({{ selectedTools.length === 0 ? 'all tools' : `${selectedTools.length} selected` }})
                    </p>
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

                  <!-- Tools List -->
                  <div class="space-y-2 max-h-[500px] overflow-y-auto">
                    <div
                      v-for="tool in tools"
                      :key="tool.name"
                      class="border border-gray-200 rounded-lg p-4 hover:bg-gray-50 cursor-pointer transition-colors"
                      @click="selectToolForTest(tool)"
                    >
                      <div class="flex items-start">
                        <input
                          type="checkbox"
                          :value="tool.name"
                          v-model="selectedTools"
                          class="mt-1 h-4 w-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                          @click.stop
                        />
                        <div class="ml-3 flex-1 min-w-0">
                          <div class="flex items-center justify-between">
                            <span class="font-medium text-gray-900">{{ tool.name }}</span>
                            <button
                              type="button"
                              class="text-xs text-blue-600 hover:text-blue-800 font-medium"
                              @click.stop="selectToolForTest(tool)"
                            >
                              Test
                            </button>
                          </div>
                          <p v-if="tool.description" class="text-sm text-gray-500 mt-1">{{ tool.description }}</p>

                          <!-- Parameters Preview -->
                          <div v-if="tool.inputSchema && tool.inputSchema.properties" class="mt-2">
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
                  </div>
                </div>
              </div>

              <!-- Test Tool Tab -->
              <div v-show="activeTab === 'test'">
                <div v-if="!selectedTool" class="text-center py-12 text-gray-500">
                  <svg class="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
                  </svg>
                  <p class="mt-4">Select a tool from the Tools tab to test it</p>
                </div>

                <div v-else class="space-y-4">
                  <!-- Selected Tool Info -->
                  <div class="bg-gray-50 rounded-lg p-4">
                    <div class="font-medium text-gray-900">{{ selectedTool.name }}</div>
                    <p v-if="selectedTool.description" class="text-sm text-gray-500 mt-1">{{ selectedTool.description }}</p>
                  </div>

                  <!-- Tool Arguments Form -->
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
                    :loading="callingTool"
                    @click="callTool"
                  >
                    Execute Tool
                  </BaseButton>

                  <!-- Tool Result -->
                  <div v-if="toolResult" class="mt-4">
                    <h4 class="text-sm font-medium text-gray-700 mb-2">Result</h4>
                    <div
                      class="rounded-lg p-4 overflow-auto max-h-80"
                      :class="toolResult.success && !toolResult.is_error ? 'bg-green-50 border border-green-200' : 'bg-red-50 border border-red-200'"
                    >
                      <pre v-if="toolResult.content" class="text-sm whitespace-pre-wrap font-mono">{{ formatContent(toolResult.content) }}</pre>
                      <p v-if="toolResult.error_message" class="text-sm text-red-600">{{ toolResult.error_message }}</p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>

    <!-- Add Server Modal -->
    <AddServerModal
      v-model="showAddModal"
      :prefill-data="addServerData"
      @success="handleServerAdded"
    />
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/services/api'
import NavBar from '@/components/layout/NavBar.vue'
import BaseInput from '@/components/common/BaseInput.vue'
import BaseButton from '@/components/common/BaseButton.vue'
import AddServerModal from '@/components/servers/AddServerModal.vue'

const router = useRouter()

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

// Add server modal
const showAddModal = ref(false)
const addServerData = ref(null)

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
  selectedTool.value = null
  toolResult.value = null

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

const proceedToAddServer = () => {
  addServerData.value = {
    url: connectionConfig.value.url,
    transport: connectionConfig.value.transport,
    protocol_version: connectionConfig.value.protocol_version,
    server_info: serverInfo.value,
    tools: tools.value,
    allowed_tools: selectedTools.value
  }
  showAddModal.value = true
}

const handleServerAdded = () => {
  showAddModal.value = false
  router.push('/admin')
}
</script>
