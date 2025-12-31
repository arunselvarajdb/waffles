<template>
  <BaseModal
    :model-value="modelValue"
    title="Server Details"
    size="lg"
    @update:model-value="$emit('update:modelValue', $event)"
    @close="$emit('update:modelValue', false)"
  >
    <div v-if="server" class="space-y-6">
      <!-- Server Info Header -->
      <div class="flex items-start justify-between">
        <div>
          <h3 class="text-lg font-medium text-gray-900">{{ server.name }}</h3>
          <p v-if="server.description" class="mt-1 text-sm text-gray-500">{{ server.description }}</p>
        </div>
        <BaseBadge :variant="server.is_active ? 'success' : 'secondary'">
          {{ server.is_active ? 'Active' : 'Inactive' }}
        </BaseBadge>
      </div>

      <!-- Client Configuration Tabs -->
      <div class="border border-gray-200 rounded-lg overflow-hidden">
        <div class="bg-gray-50 border-b border-gray-200">
          <nav class="flex -mb-px">
            <button
              v-for="tab in configTabs"
              :key="tab.id"
              type="button"
              class="py-3 px-4 text-sm font-medium border-b-2 transition-colors"
              :class="activeConfigTab === tab.id
                ? 'border-blue-500 text-blue-600 bg-white'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:bg-gray-100'"
              @click="activeConfigTab = tab.id"
            >
              {{ tab.label }}
            </button>
          </nav>
        </div>

        <!-- Claude Code Config -->
        <div v-show="activeConfigTab === 'claude-code'" class="p-4">
          <div class="space-y-4">
            <!-- CLI Command -->
            <div>
              <p class="text-sm font-medium text-gray-700 mb-2">Option 1: Add via CLI</p>
              <p class="text-sm text-gray-600 mb-2">
                Run this command to add the server:
              </p>
              <div class="bg-gray-900 rounded-lg p-3 relative">
                <button
                  type="button"
                  class="absolute top-2 right-2 text-xs text-gray-400 hover:text-white"
                  @click="copyToClipboard(claudeCodeCliCommand)"
                >
                  Copy
                </button>
                <pre class="text-xs text-green-400 font-mono overflow-x-auto whitespace-pre-wrap">{{ claudeCodeCliCommand }}</pre>
              </div>
            </div>

            <!-- Manual Config -->
            <div>
              <p class="text-sm font-medium text-gray-700 mb-2">Option 2: Manual Configuration</p>
              <p class="text-sm text-gray-600 mb-2">
                Add this to your <code class="bg-gray-100 px-1 rounded">~/.claude/settings.json</code> or project's <code class="bg-gray-100 px-1 rounded">.claude/settings.json</code>:
              </p>
              <div class="bg-gray-900 rounded-lg p-3 relative">
                <button
                  type="button"
                  class="absolute top-2 right-2 text-xs text-gray-400 hover:text-white"
                  @click="copyToClipboard(claudeCodeConfig)"
                >
                  Copy
                </button>
                <pre class="text-xs text-green-400 font-mono overflow-x-auto whitespace-pre-wrap">{{ claudeCodeConfig }}</pre>
              </div>
            </div>

            <div class="bg-blue-50 rounded-lg p-3 text-xs text-blue-700">
              <strong>How to get your API key:</strong>
              <ol class="mt-2 list-decimal list-inside space-y-1">
                <li>Go to <a href="/profile?tab=api-keys" class="text-blue-600 hover:underline font-medium">Account Settings → API Keys</a></li>
                <li>Click "Create API Key" and give it a name</li>
                <li>Copy the key (it's only shown once!)</li>
                <li>Replace <code class="bg-blue-100 px-1 rounded">your-api-key</code> in the config above</li>
              </ol>
            </div>
          </div>
        </div>

        <!-- Cursor Config -->
        <div v-show="activeConfigTab === 'cursor'" class="p-4">
          <div class="space-y-4">
            <!-- One-Click Install Button -->
            <div>
              <p class="text-sm font-medium text-gray-700 mb-2">Option 1: One-Click Install</p>
              <a
                :href="cursorDeepLink"
                class="inline-flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-purple-600 to-indigo-600 text-white text-sm font-medium rounded-lg hover:from-purple-700 hover:to-indigo-700 transition-all shadow-sm"
              >
                <svg class="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"/>
                </svg>
                Add to Cursor
              </a>
              <p class="text-xs text-gray-500 mt-2">
                Opens Cursor and prompts to add this MCP server. Uses OAuth for authentication.
              </p>
            </div>

            <!-- Manual Config -->
            <div>
              <p class="text-sm font-medium text-gray-700 mb-2">Option 2: Manual Configuration</p>
              <p class="text-sm text-gray-600 mb-2">
                Add this to your Cursor MCP settings (<code class="bg-gray-100 px-1 rounded">~/.cursor/mcp.json</code>):
              </p>
              <div class="bg-gray-900 rounded-lg p-3 relative">
                <button
                  type="button"
                  class="absolute top-2 right-2 text-xs text-gray-400 hover:text-white"
                  @click="copyToClipboard(cursorConfig)"
                >
                  Copy
                </button>
                <pre class="text-xs text-green-400 font-mono overflow-x-auto whitespace-pre-wrap">{{ cursorConfig }}</pre>
              </div>
              <div class="bg-blue-50 rounded-lg p-3 text-xs text-blue-700 mt-2">
                <strong>How to get your API key:</strong>
                <ol class="mt-2 list-decimal list-inside space-y-1">
                  <li>Go to <a href="/profile?tab=api-keys" class="text-blue-600 hover:underline font-medium">Account Settings → API Keys</a></li>
                  <li>Click "Create API Key" and give it a name</li>
                  <li>Copy the key and replace <code class="bg-blue-100 px-1 rounded">your-api-key</code> above</li>
                  <li>Restart Cursor after updating the configuration</li>
                </ol>
              </div>
            </div>
          </div>
        </div>

        <!-- VS Code Config -->
        <div v-show="activeConfigTab === 'vscode'" class="p-4">
          <div class="space-y-3">
            <p class="text-sm text-gray-600">
              Add this to your VS Code settings (<code class="bg-gray-100 px-1 rounded">.vscode/settings.json</code>):
            </p>
            <div class="bg-gray-900 rounded-lg p-3 relative">
              <button
                type="button"
                class="absolute top-2 right-2 text-xs text-gray-400 hover:text-white"
                @click="copyToClipboard(vscodeConfig)"
              >
                Copy
              </button>
              <pre class="text-xs text-green-400 font-mono overflow-x-auto whitespace-pre-wrap">{{ vscodeConfig }}</pre>
            </div>
            <div class="bg-blue-50 rounded-lg p-3 text-xs text-blue-700">
              <strong>Setup Instructions:</strong>
              <ol class="mt-2 list-decimal list-inside space-y-1">
                <li>Install a VS Code extension that supports MCP (like Cline or Continue)</li>
                <li>Go to <a href="/profile?tab=api-keys" class="text-blue-600 hover:underline font-medium">Account Settings → API Keys</a> to create an API key</li>
                <li>Copy the key and replace <code class="bg-blue-100 px-1 rounded">your-api-key</code> above</li>
                <li>Check your extension's documentation for the exact config location</li>
              </ol>
            </div>
          </div>
        </div>
      </div>

      <!-- Authentication Info -->
      <div class="bg-yellow-50 rounded-lg p-4 border border-yellow-100">
        <div class="flex items-start gap-2">
          <svg class="h-5 w-5 text-yellow-600 mt-0.5 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
          <div>
            <h4 class="text-sm font-medium text-yellow-800">Authentication Required</h4>
            <p class="text-xs text-yellow-700 mt-1">
              All gateway endpoints require authentication. For MCP clients, use an API key:
            </p>
            <ul class="text-xs text-yellow-700 mt-2 list-disc list-inside space-y-1">
              <li><strong>API Key (Recommended):</strong> Pass <code class="bg-yellow-100 px-1 rounded">X-API-Key: mcpgw_...</code> header</li>
              <li><strong>Bearer Token:</strong> Pass <code class="bg-yellow-100 px-1 rounded">Authorization: Bearer mcpgw_...</code></li>
            </ul>
            <a
              href="/profile?tab=api-keys"
              class="inline-flex items-center mt-3 text-xs font-medium text-yellow-800 hover:text-yellow-900 underline"
            >
              Create an API Key →
            </a>
          </div>
        </div>
      </div>

      <!-- Server Configuration Details -->
      <div class="border-t border-gray-200 pt-4">
        <h4 class="text-sm font-medium text-gray-900 mb-3">Server Configuration</h4>
        <dl class="grid grid-cols-2 gap-4 text-sm">
          <div>
            <dt class="text-gray-500">Direct URL</dt>
            <dd class="font-mono text-gray-900 truncate">{{ server.url }}</dd>
          </div>
          <div>
            <dt class="text-gray-500">Transport</dt>
            <dd class="text-gray-900">{{ formatTransport(server.transport) }}</dd>
          </div>
          <div>
            <dt class="text-gray-500">Protocol Version</dt>
            <dd class="text-gray-900">{{ server.protocol_version || 'N/A' }}</dd>
          </div>
          <div>
            <dt class="text-gray-500">Auth Type</dt>
            <dd class="text-gray-900">{{ formatAuthType(server.auth_type) }}</dd>
          </div>
          <div>
            <dt class="text-gray-500">Timeout</dt>
            <dd class="text-gray-900">{{ server.timeout || 30 }}s</dd>
          </div>
          <div>
            <dt class="text-gray-500">Max Connections</dt>
            <dd class="text-gray-900">{{ server.max_connections || 10 }}</dd>
          </div>
        </dl>
      </div>

      <!-- Tags -->
      <div v-if="server.tags && server.tags.length > 0" class="border-t border-gray-200 pt-4">
        <h4 class="text-sm font-medium text-gray-900 mb-2">Tags</h4>
        <div class="flex flex-wrap gap-2">
          <span
            v-for="tag in server.tags"
            :key="tag"
            class="inline-flex items-center px-2 py-1 rounded-md text-xs font-medium bg-gray-100 text-gray-800"
          >
            {{ tag }}
          </span>
        </div>
      </div>
    </div>

    <template #footer>
      <BaseButton variant="secondary" @click="$emit('update:modelValue', false)">
        Close
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup>
import { ref, computed } from 'vue'
import BaseModal from '@/components/common/BaseModal.vue'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseBadge from '@/components/common/BaseBadge.vue'

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

defineEmits(['update:modelValue'])

// Client configuration tabs
const activeConfigTab = ref('claude-code')

const configTabs = [
  { id: 'claude-code', label: 'Claude Code' },
  { id: 'cursor', label: 'Cursor' },
  { id: 'vscode', label: 'VS Code' }
]

// Compute the gateway base URL (in production, this would come from config)
const gatewayBaseUrl = computed(() => {
  // Use the current window location for the gateway URL
  const protocol = window.location.protocol
  const host = window.location.hostname
  const currentPort = window.location.port
  // In development, Vite runs on ports like 5173, 5174, 5175, 5176, etc.
  // Map any Vite dev port to backend port 8080
  const isViteDevPort = currentPort && currentPort.startsWith('517')
  const port = isViteDevPort ? '8080' : currentPort
  return `${protocol}//${host}${port ? ':' + port : ''}/api/v1/gateway`
})

// Claude Code CLI command
const claudeCodeCliCommand = computed(() => {
  if (!props.server?.id) return ''
  const base = gatewayBaseUrl.value
  const serverName = props.server.name.toLowerCase().replace(/\s+/g, '-')
  return `claude mcp add ${serverName} ${base}/${props.server.id} --transport http`
})

// Client configuration for Claude Code
const claudeCodeConfig = computed(() => {
  if (!props.server?.id) return ''
  const base = gatewayBaseUrl.value
  const config = {
    mcpServers: {
      [props.server.name.toLowerCase().replace(/\s+/g, '-')]: {
        url: `${base}/${props.server.id}`,
        transport: 'streamable-http',
        headers: {
          'X-API-Key': 'your-api-key'
        }
      }
    }
  }
  return JSON.stringify(config, null, 2)
})

// Client configuration for Cursor
const cursorConfig = computed(() => {
  if (!props.server?.id) return ''
  const base = gatewayBaseUrl.value
  const config = {
    mcpServers: {
      [props.server.name.toLowerCase().replace(/\s+/g, '-')]: {
        url: `${base}/${props.server.id}`,
        transport: 'streamable-http',
        headers: {
          'X-API-Key': 'your-api-key'
        }
      }
    }
  }
  return JSON.stringify(config, null, 2)
})

// Cursor deep link for one-click install
// Format: cursor://anysphere.cursor-deeplink/mcp/install?name=$NAME&config=$BASE64_CONFIG
const cursorDeepLink = computed(() => {
  if (!props.server?.id) return ''
  const base = gatewayBaseUrl.value
  const serverName = props.server.name.toLowerCase().replace(/\s+/g, '-')

  // Config for remote HTTP server (OAuth-enabled, no API key needed)
  const config = {
    url: `${base}/${props.server.id}`
  }

  // Base64 encode the config
  const configBase64 = btoa(JSON.stringify(config))

  return `cursor://anysphere.cursor-deeplink/mcp/install?name=${encodeURIComponent(serverName)}&config=${configBase64}`
})

// Client configuration for VS Code (Cline/Continue)
const vscodeConfig = computed(() => {
  if (!props.server?.id) return ''
  const base = gatewayBaseUrl.value
  const config = {
    'mcp.servers': {
      [props.server.name.toLowerCase().replace(/\s+/g, '-')]: {
        url: `${base}/${props.server.id}`,
        transport: 'streamable-http',
        headers: {
          'X-API-Key': 'your-api-key'
        }
      }
    }
  }
  return JSON.stringify(config, null, 2)
})

const formatTransport = (transport) => {
  const transports = {
    'streamable_http': 'Streamable HTTP (MCP 2025)',
    'sse': 'SSE (Server-Sent Events)',
    'http': 'HTTP (Legacy)'
  }
  return transports[transport] || transport || 'Unknown'
}

const formatAuthType = (authType) => {
  if (!authType || authType === 'none') return 'None'
  return authType.charAt(0).toUpperCase() + authType.slice(1)
}

const copyToClipboard = async (text) => {
  try {
    await navigator.clipboard.writeText(text)
    // Could add a toast notification here
  } catch (err) {
    console.error('Failed to copy:', err)
  }
}
</script>
