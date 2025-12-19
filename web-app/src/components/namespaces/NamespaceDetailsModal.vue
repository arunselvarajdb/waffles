<template>
  <BaseModal
    :modelValue="modelValue"
    @update:modelValue="$emit('update:modelValue', $event)"
    :title="namespace?.name || 'Namespace Details'"
    size="lg"
  >
    <div class="space-y-6">
      <!-- Namespace Info -->
      <div class="bg-gray-50 rounded-lg p-4">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-xs font-medium text-gray-500 uppercase">Name</label>
            <p class="mt-1 text-sm text-gray-900">{{ namespace?.name }}</p>
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-500 uppercase">Created</label>
            <p class="mt-1 text-sm text-gray-900">{{ formatDate(namespace?.created_at) }}</p>
          </div>
          <div class="col-span-2">
            <label class="block text-xs font-medium text-gray-500 uppercase">Description</label>
            <p class="mt-1 text-sm text-gray-900">{{ namespace?.description || 'No description' }}</p>
          </div>
        </div>
      </div>

      <!-- Tabs -->
      <div class="border-b border-gray-200">
        <nav class="-mb-px flex space-x-8">
          <button
            @click="activeTab = 'servers'"
            :class="[
              activeTab === 'servers'
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300',
              'whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm'
            ]"
          >
            Servers ({{ servers.length }})
          </button>
          <button
            @click="activeTab = 'access'"
            :class="[
              activeTab === 'access'
                ? 'border-blue-500 text-blue-600'
                : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300',
              'whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm'
            ]"
          >
            Role Access ({{ accessEntries.length }})
          </button>
        </nav>
      </div>

      <!-- Tab Content -->
      <div class="min-h-[200px]">
        <!-- Servers Tab -->
        <div v-if="activeTab === 'servers'" class="space-y-4">
          <!-- Add Server -->
          <div class="flex gap-2">
            <select
              v-model="selectedServerId"
              class="flex-1 px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="">Select a server to add...</option>
              <option
                v-for="server in availableServers"
                :key="server.id"
                :value="server.id"
              >
                {{ server.name }}
              </option>
            </select>
            <BaseButton
              variant="primary"
              @click="handleAddServer"
              :disabled="!selectedServerId || addingServer"
            >
              {{ addingServer ? 'Adding...' : 'Add' }}
            </BaseButton>
          </div>

          <!-- Server List -->
          <div v-if="loadingServers" class="text-center py-8">
            <p class="text-gray-500">Loading servers...</p>
          </div>
          <div v-else-if="servers.length === 0" class="text-center py-8">
            <p class="text-gray-500">No servers in this namespace</p>
          </div>
          <ul v-else class="divide-y divide-gray-200 border border-gray-200 rounded-md">
            <li
              v-for="server in servers"
              :key="server.server_id"
              class="flex items-center justify-between px-4 py-3"
            >
              <span class="text-sm text-gray-900">{{ server.server_name || server.server_id }}</span>
              <button
                @click="handleRemoveServer(server.server_id)"
                class="text-red-600 hover:text-red-800 text-sm"
              >
                Remove
              </button>
            </li>
          </ul>
        </div>

        <!-- Access Tab -->
        <div v-if="activeTab === 'access'" class="space-y-4">
          <!-- Add Access -->
          <div class="flex gap-2">
            <select
              v-model="selectedRoleName"
              class="flex-1 px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="">Select a role...</option>
              <option
                v-for="role in availableRoles"
                :key="role.id"
                :value="role.name"
              >
                {{ role.name }}
              </option>
            </select>
            <select
              v-model="selectedAccessLevel"
              class="w-32 px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="view">View</option>
              <option value="execute">Execute</option>
            </select>
            <BaseButton
              variant="primary"
              @click="handleSetAccess"
              :disabled="!selectedRoleName || settingAccess"
            >
              {{ settingAccess ? 'Setting...' : 'Set' }}
            </BaseButton>
          </div>

          <!-- Access List -->
          <div v-if="loadingAccess" class="text-center py-8">
            <p class="text-gray-500">Loading access entries...</p>
          </div>
          <div v-else-if="accessEntries.length === 0" class="text-center py-8">
            <p class="text-gray-500">No role access configured</p>
          </div>
          <ul v-else class="divide-y divide-gray-200 border border-gray-200 rounded-md">
            <li
              v-for="entry in accessEntries"
              :key="entry.role_id"
              class="flex items-center justify-between px-4 py-3"
            >
              <div class="flex items-center gap-3">
                <span class="text-sm text-gray-900">{{ entry.role_name }}</span>
                <BaseBadge :variant="entry.access_level === 'execute' ? 'success' : 'info'">
                  {{ entry.access_level }}
                </BaseBadge>
              </div>
              <button
                @click="handleRemoveAccess(entry.role_id)"
                class="text-red-600 hover:text-red-800 text-sm"
              >
                Remove
              </button>
            </li>
          </ul>
        </div>
      </div>
    </div>

    <template #footer>
      <BaseButton variant="secondary" @click="handleClose">
        Close
      </BaseButton>
    </template>
  </BaseModal>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import BaseModal from '@/components/common/BaseModal.vue'
import BaseButton from '@/components/common/BaseButton.vue'
import BaseBadge from '@/components/common/BaseBadge.vue'
import { useNamespacesStore } from '@/stores/namespaces'
import { useServersStore } from '@/stores/servers'
import { useRolesStore } from '@/stores/roles'

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

const emit = defineEmits(['update:modelValue'])

const namespacesStore = useNamespacesStore()
const serversStore = useServersStore()
const rolesStore = useRolesStore()

const activeTab = ref('servers')
const loadingServers = ref(false)
const loadingAccess = ref(false)
const addingServer = ref(false)
const settingAccess = ref(false)

const selectedServerId = ref('')
const selectedRoleName = ref('')
const selectedAccessLevel = ref('view')

const servers = computed(() => namespacesStore.currentNamespaceServers)
const accessEntries = computed(() => namespacesStore.currentNamespaceAccess)

// Servers not already in the namespace
const availableServers = computed(() => {
  const memberIds = new Set(servers.value.map(s => s.server_id))
  return serversStore.servers.filter(s => !memberIds.has(s.id))
})

// Roles not already configured
const availableRoles = computed(() => {
  const accessRoleIds = new Set(accessEntries.value.map(a => a.role_id))
  return rolesStore.roles.filter(r => !accessRoleIds.has(r.id))
})

const loadData = async () => {
  if (!props.namespace?.id) return

  loadingServers.value = true
  loadingAccess.value = true

  try {
    await Promise.all([
      namespacesStore.fetchNamespaceServers(props.namespace.id),
      namespacesStore.fetchNamespaceAccess(props.namespace.id),
      serversStore.fetchServers(),
      rolesStore.fetchRoles()
    ])
  } catch (error) {
    console.error('Failed to load namespace data:', error)
  } finally {
    loadingServers.value = false
    loadingAccess.value = false
  }
}

watch(() => props.modelValue, (isOpen) => {
  if (isOpen) {
    activeTab.value = 'servers'
    selectedServerId.value = ''
    selectedRoleName.value = ''
    selectedAccessLevel.value = 'view'
    loadData()
  }
})

const handleAddServer = async () => {
  if (!selectedServerId.value) return

  addingServer.value = true
  try {
    await namespacesStore.addServerToNamespace(props.namespace.id, selectedServerId.value)
    selectedServerId.value = ''
  } catch (error) {
    console.error('Failed to add server:', error)
  } finally {
    addingServer.value = false
  }
}

const handleRemoveServer = async (serverId) => {
  try {
    await namespacesStore.removeServerFromNamespace(props.namespace.id, serverId)
  } catch (error) {
    console.error('Failed to remove server:', error)
  }
}

const handleSetAccess = async () => {
  if (!selectedRoleName.value) return

  settingAccess.value = true
  try {
    await namespacesStore.setRoleAccess(
      props.namespace.id,
      selectedRoleName.value,
      selectedAccessLevel.value
    )
    selectedRoleName.value = ''
    selectedAccessLevel.value = 'view'
  } catch (error) {
    console.error('Failed to set access:', error)
  } finally {
    settingAccess.value = false
  }
}

const handleRemoveAccess = async (roleId) => {
  try {
    await namespacesStore.removeRoleAccess(props.namespace.id, roleId)
  } catch (error) {
    console.error('Failed to remove access:', error)
  }
}

const handleClose = () => {
  emit('update:modelValue', false)
}

const formatDate = (dateStr) => {
  if (!dateStr) return 'N/A'
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  })
}
</script>
