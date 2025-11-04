<template>
  <div class="vm-details">
    <div class="header">
      <div>
        <button @click="goBack" class="btn btn-secondary">← Back</button>
        <h2>{{ vmName }}</h2>
      </div>
      <div class="header-actions">
        <button @click="startVM" class="btn btn-sm btn-success" :disabled="vm?.spec?.runStrategy === 'Always'">Start</button>
        <button @click="stopVM" class="btn btn-sm btn-warning" :disabled="vm?.spec?.runStrategy === 'Halted'">Stop</button>
        <button @click="rebootVM" class="btn btn-sm btn-info">Reboot</button>
        <button @click="deleteVM" class="btn btn-sm btn-danger">Delete</button>
      </div>
    </div>

    <div v-if="loading" class="loading">Loading VM details...</div>

    <div v-else-if="vm" class="details-container">
      <!-- VM Information Card -->
      <div class="info-card">
        <h3>VM Information</h3>
        <div class="info-grid">
          <div class="info-item">
            <span class="info-label">Status</span>
            <span :class="['badge', vm.status?.phase]">{{ vm.status?.phase || 'Unknown' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Namespace</span>
            <span class="info-value">{{ namespace }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">CPUs</span>
            <span class="info-value">{{ vm.spec?.cpus || 1 }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Memory</span>
            <span class="info-value">{{ vm.spec?.memory || '1Gi' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Disk Size</span>
            <span class="info-value">{{ vm.spec?.diskSize || '10Gi' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">OS</span>
            <span class="info-value">{{ vm.spec?.os }}{{ vm.spec?.osVersion ? ':' + vm.spec.osVersion : '' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Node</span>
            <span class="info-value">{{ vm.status?.node || '-' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">IP Address</span>
            <span class="info-value">{{ vm.status?.ipAddress || '-' }}</span>
          </div>
          <div class="info-item">
            <span class="info-label">Run Strategy</span>
            <span class="info-value">{{ vm.spec?.runStrategy || 'Always' }}</span>
          </div>
        </div>
      </div>

      <!-- Console Card -->
      <div class="console-card">
        <div class="console-header">
          <h3>VM Console</h3>
          <div class="console-controls">
            <button @click="refreshConsole" class="btn btn-sm">Refresh</button>
            <button @click="clearConsole" class="btn btn-sm">Clear</button>
            <select v-model="consoleType" class="console-type-select">
              <option value="serial">Serial Console</option>
              <option value="vnc">VNC (Graphical)</option>
            </select>
          </div>
        </div>

        <div class="console-container">
          <div v-if="consoleType === 'serial'" class="serial-console">
            <div ref="consoleOutput" class="console-output">{{ consoleLog }}</div>
            <div class="console-input-area">
              <input
                v-model="consoleInput"
                @keyup.enter="sendCommand"
                placeholder="Type command and press Enter..."
                class="console-input"
              />
              <button @click="sendCommand" class="btn btn-sm btn-primary">Send</button>
            </div>
          </div>

          <div v-else class="vnc-console">
            <div class="vnc-placeholder">
              <p>VNC Console</p>
              <p class="vnc-info">Graphical console access via noVNC</p>
              <p class="vnc-note">Note: Requires VNC service to be running on the VM</p>
              <iframe
                v-if="vncUrl"
                :src="vncUrl"
                class="vnc-frame"
                title="VNC Console"
              ></iframe>
              <div v-else class="vnc-unavailable">
                VNC console not available. VM may not be running or VNC is not configured.
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- CloudInit Card -->
      <div class="cloudInit-card" v-if="vm.spec?.cloudInit">
        <h3>Cloud-Init Configuration</h3>
        <pre class="cloudInit-content">{{ vm.spec.cloudInit }}</pre>
      </div>

      <!-- SSH Keys Card -->
      <div class="ssh-keys-card" v-if="vm.spec?.sshKeys && vm.spec.sshKeys.length > 0">
        <h3>SSH Keys</h3>
        <div class="ssh-keys-list">
          <div v-for="(key, index) in vm.spec.sshKeys" :key="index" class="ssh-key-item">
            <code>{{ key.substring(0, 60) }}...</code>
          </div>
        </div>
      </div>

      <!-- KubeVirt Details Card -->
      <div class="kubevirt-details-card">
        <div class="kubevirt-header">
          <h3>KubeVirt VM Details</h3>
          <button @click="loadDescribe" class="btn btn-sm">Refresh</button>
        </div>

        <div class="tabs">
          <button
            @click="activeTab = 'describe'"
            :class="['tab', { active: activeTab === 'describe' }]">
            Describe
          </button>
          <button
            @click="activeTab = 'yaml-vm'"
            :class="['tab', { active: activeTab === 'yaml-vm' }]">
            VM YAML
          </button>
          <button
            @click="activeTab = 'yaml-vmi'"
            :class="['tab', { active: activeTab === 'yaml-vmi' }]"
            :disabled="!describeData?.yaml?.vmi">
            VMI YAML
          </button>
          <button
            @click="loadEvents(); activeTab = 'events'"
            :class="['tab', { active: activeTab === 'events' }]">
            Events
          </button>
        </div>

        <div class="tab-content">
          <div v-if="loadingDescribe && activeTab !== 'events'" class="loading-describe">
            Loading KubeVirt details...
          </div>
          <div v-else-if="describeError && activeTab !== 'events'" class="error-describe">
            {{ describeError }}
          </div>
          <pre v-else-if="activeTab === 'describe'" class="describe-output">{{ describeData?.describe }}</pre>
          <pre v-else-if="activeTab === 'yaml-vm'" class="yaml-output">{{ describeData?.yaml?.vm }}</pre>
          <pre v-else-if="activeTab === 'yaml-vmi'" class="yaml-output">{{ describeData?.yaml?.vmi || 'VMI not available' }}</pre>
          <div v-else-if="activeTab === 'events'">
            <div v-if="loadingEvents" class="loading-describe">Loading events...</div>
            <div v-else-if="eventsError" class="error-describe">{{ eventsError }}</div>
            <div v-else-if="!events || events.length === 0" class="no-events">No events found</div>
            <table v-else class="events-table">
              <thead>
                <tr>
                  <th>Type</th>
                  <th>Reason</th>
                  <th>Object</th>
                  <th>Source</th>
                  <th>Message</th>
                  <th>Count</th>
                  <th>Last Seen</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(event, index) in events" :key="index" :class="'event-' + (event.type || 'normal').toLowerCase()">
                  <td>{{ event.type || 'Normal' }}</td>
                  <td>{{ event.reason }}</td>
                  <td>{{ event.involvedObjectKind }}/{{ event.involvedObjectName }}</td>
                  <td>{{ event.source }}</td>
                  <td class="message-cell">{{ event.message }}</td>
                  <td>{{ event.count || 1 }}</td>
                  <td>{{ formatTimestamp(event.lastTimestamp || event.firstTimestamp) }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { vmsApi } from '../api/client'

export default {
  setup() {
    const route = useRoute()
    const router = useRouter()
    const namespace = ref(route.params.namespace)
    const vmName = ref(route.params.name)
    const vm = ref(null)
    const loading = ref(true)
    const consoleType = ref('serial')
    const consoleLog = ref('VM Console Output\n\nWaiting for VM to start...\n')
    const consoleInput = ref('')
    const consoleOutput = ref(null)
    const vncUrl = ref('')
    const activeTab = ref('describe')
    const describeData = ref(null)
    const loadingDescribe = ref(false)
    const describeError = ref('')
    const events = ref([])
    const loadingEvents = ref(false)
    const eventsError = ref('')
    let refreshInterval = null

    const loadVM = async () => {
      try {
        const response = await vmsApi.get(namespace.value, vmName.value)
        vm.value = response.data
        loading.value = false

        // Simulate console output for demonstration
        if (vm.value.status?.phase === 'Running') {
          updateConsoleOutput()
        }
      } catch (error) {
        console.error('Failed to load VM:', error)
        loading.value = false
      }
    }

    const updateConsoleOutput = () => {
      if (vm.value?.status?.phase === 'Running') {
        const timestamp = new Date().toLocaleTimeString()
        if (!consoleLog.value.includes('VM is running')) {
          consoleLog.value += `\n[${timestamp}] VM is running on node ${vm.value.status.node || 'unknown'}\n`
          consoleLog.value += `[${timestamp}] IP Address: ${vm.value.status.ipAddress || 'not assigned'}\n`
          consoleLog.value += `[${timestamp}] Console ready. Type commands below.\n`
          consoleLog.value += `\n> `
        }
      }
    }

    const refreshConsole = () => {
      loadVM()
      updateConsoleOutput()
    }

    const clearConsole = () => {
      consoleLog.value = 'Console cleared.\n\n> '
    }

    const sendCommand = () => {
      if (!consoleInput.value.trim()) return

      const timestamp = new Date().toLocaleTimeString()
      consoleLog.value += `${consoleInput.value}\n`
      consoleLog.value += `[${timestamp}] Command sent (simulated)\n`
      consoleLog.value += `> `

      consoleInput.value = ''

      // Auto-scroll to bottom
      if (consoleOutput.value) {
        consoleOutput.value.scrollTop = consoleOutput.value.scrollHeight
      }
    }

    const startVM = async () => {
      try {
        await vmsApi.start(namespace.value, vmName.value)
        await loadVM()
        consoleLog.value += `\n[${new Date().toLocaleTimeString()}] VM start requested\n> `
      } catch (error) {
        console.error('Failed to start VM:', error)
        alert('Failed to start VM: ' + error.message)
      }
    }

    const stopVM = async () => {
      if (!confirm(`Stop VM ${vmName.value}?`)) return
      try {
        await vmsApi.stop(namespace.value, vmName.value)
        await loadVM()
        consoleLog.value += `\n[${new Date().toLocaleTimeString()}] VM stop requested\n> `
      } catch (error) {
        console.error('Failed to stop VM:', error)
        alert('Failed to stop VM: ' + error.message)
      }
    }

    const rebootVM = async () => {
      if (!confirm(`Reboot VM ${vmName.value}?`)) return
      try {
        await vmsApi.reboot(namespace.value, vmName.value)
        await loadVM()
        consoleLog.value += `\n[${new Date().toLocaleTimeString()}] VM reboot requested\n> `
      } catch (error) {
        console.error('Failed to reboot VM:', error)
        alert('Failed to reboot VM: ' + error.message)
      }
    }

    const deleteVM = async () => {
      if (!confirm(`Delete VM ${vmName.value}? This action cannot be undone.`)) return
      try {
        await vmsApi.delete(namespace.value, vmName.value)
        router.push('/vms')
      } catch (error) {
        console.error('Failed to delete VM:', error)
        alert('Failed to delete VM: ' + error.message)
      }
    }

    const loadDescribe = async () => {
      loadingDescribe.value = true
      describeError.value = ''
      try {
        const response = await vmsApi.describe(namespace.value, vmName.value)
        describeData.value = response.data
      } catch (error) {
        console.error('Failed to load VM describe:', error)
        describeError.value = 'Failed to load KubeVirt details: ' + error.message
      } finally {
        loadingDescribe.value = false
      }
    }

    const loadEvents = async () => {
      loadingEvents.value = true
      eventsError.value = ''
      try {
        const response = await vmsApi.events(namespace.value, vmName.value)
        events.value = response.data.events || []
      } catch (error) {
        console.error('Failed to load events:', error)
        eventsError.value = 'Failed to load events: ' + error.message
      } finally {
        loadingEvents.value = false
      }
    }

    const formatTimestamp = (timestamp) => {
      if (!timestamp) return 'N/A'
      const date = new Date(timestamp)
      const now = new Date()
      const diff = Math.floor((now - date) / 1000)

      if (diff < 60) return `${diff}s ago`
      if (diff < 3600) return `${Math.floor(diff / 60)}m ago`
      if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`
      return `${Math.floor(diff / 86400)}d ago`
    }

    const goBack = () => {
      router.push('/vms')
    }

    onMounted(() => {
      loadVM()
      loadDescribe()
      // Refresh VM status and describe data every 10 seconds
      refreshInterval = setInterval(() => {
        loadVM()
        loadDescribe()
      }, 10000)

      // Set VNC URL if available (would come from API in real implementation)
      // vncUrl.value = `/api/v1/namespaces/${namespace.value}/vms/${vmName.value}/vnc`
    })

    onUnmounted(() => {
      if (refreshInterval) {
        clearInterval(refreshInterval)
      }
    })

    return {
      namespace,
      vmName,
      vm,
      loading,
      consoleType,
      consoleLog,
      consoleInput,
      consoleOutput,
      vncUrl,
      activeTab,
      describeData,
      loadingDescribe,
      describeError,
      events,
      loadingEvents,
      eventsError,
      loadVM,
      loadDescribe,
      loadEvents,
      formatTimestamp,
      refreshConsole,
      clearConsole,
      sendCommand,
      startVM,
      stopVM,
      rebootVM,
      deleteVM,
      goBack
    }
  }
}
</script>

<style scoped>
.vm-details {
  max-width: 1400px;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
}

.header > div:first-child {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.header h2 {
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 0.5rem;
}

.loading {
  text-align: center;
  padding: 3rem;
  color: #666;
}

.details-container {
  display: grid;
  gap: 2rem;
}

.info-card, .console-card, .cloudInit-card, .ssh-keys-card {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  padding: 1.5rem;
}

.info-card h3, .console-card h3, .cloudInit-card h3, .ssh-keys-card h3 {
  margin: 0 0 1.5rem 0;
  font-size: 1.25rem;
  font-weight: 600;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: 1.5rem;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.info-label {
  font-size: 0.875rem;
  color: #666;
  font-weight: 500;
}

.info-value {
  font-size: 1rem;
  color: #333;
}

.badge {
  padding: 0.25rem 0.75rem;
  border-radius: 12px;
  font-size: 0.875rem;
  font-weight: 500;
  display: inline-block;
  width: fit-content;
}

.badge.Running {
  background: #e8f5e9;
  color: #2e7d32;
}

.badge.Pending {
  background: #fff3e0;
  color: #ef6c00;
}

.badge.Stopped, .badge.Halted {
  background: #eeeeee;
  color: #666;
}

.badge.Error {
  background: #ffebee;
  color: #c62828;
}

.console-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.console-controls {
  display: flex;
  gap: 0.5rem;
  align-items: center;
}

.console-type-select {
  padding: 0.375rem 0.75rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 0.875rem;
}

.console-container {
  border: 1px solid #ddd;
  border-radius: 4px;
  overflow: hidden;
}

.serial-console {
  display: flex;
  flex-direction: column;
  height: 500px;
}

.console-output {
  flex: 1;
  background: #1e1e1e;
  color: #d4d4d4;
  font-family: 'Courier New', monospace;
  font-size: 0.875rem;
  padding: 1rem;
  overflow-y: auto;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.console-input-area {
  display: flex;
  border-top: 1px solid #ddd;
  background: #2d2d2d;
}

.console-input {
  flex: 1;
  padding: 0.75rem;
  border: none;
  background: #2d2d2d;
  color: #d4d4d4;
  font-family: 'Courier New', monospace;
  font-size: 0.875rem;
}

.console-input:focus {
  outline: none;
  background: #3d3d3d;
}

.vnc-console {
  height: 600px;
  background: #1e1e1e;
  display: flex;
  align-items: center;
  justify-content: center;
}

.vnc-placeholder {
  text-align: center;
  color: #d4d4d4;
}

.vnc-info {
  color: #888;
  margin-top: 0.5rem;
}

.vnc-note {
  color: #666;
  font-size: 0.875rem;
  margin-top: 1rem;
}

.vnc-frame {
  width: 100%;
  height: 100%;
  border: none;
}

.vnc-unavailable {
  color: #999;
  padding: 2rem;
}

.cloudInit-content {
  background: #f5f5f5;
  padding: 1rem;
  border-radius: 4px;
  overflow-x: auto;
  font-family: 'Courier New', monospace;
  font-size: 0.875rem;
  margin: 0;
}

.ssh-keys-list {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.ssh-key-item {
  background: #f5f5f5;
  padding: 0.75rem;
  border-radius: 4px;
  font-family: 'Courier New', monospace;
  font-size: 0.875rem;
}

.btn {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
  background: #e0e0e0;
  color: #333;
}

.btn:hover {
  background: #d0d0d0;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-secondary {
  background: #6c757d;
  color: white;
}

.btn-secondary:hover {
  background: #5a6268;
}

.btn-primary {
  background: #3498db;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #2980b9;
}

.btn-sm {
  padding: 0.375rem 0.75rem;
  font-size: 0.875rem;
}

.btn-success {
  background: #27ae60;
  color: white;
}

.btn-success:hover:not(:disabled) {
  background: #229954;
}

.btn-warning {
  background: #f39c12;
  color: white;
}

.btn-warning:hover:not(:disabled) {
  background: #e67e22;
}

.btn-info {
  background: #3498db;
  color: white;
}

.btn-info:hover:not(:disabled) {
  background: #2980b9;
}

.btn-danger {
  background: #e74c3c;
  color: white;
}

.btn-danger:hover:not(:disabled) {
  background: #c0392b;
}

.kubevirt-details-card {
  background: white;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
  padding: 1.5rem;
}

.kubevirt-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.kubevirt-header h3 {
  margin: 0;
  font-size: 1.25rem;
  font-weight: 600;
}

.tabs {
  display: flex;
  gap: 0.5rem;
  border-bottom: 2px solid #e0e0e0;
  margin-bottom: 1rem;
}

.tab {
  padding: 0.75rem 1.5rem;
  border: none;
  background: none;
  cursor: pointer;
  font-size: 0.875rem;
  font-weight: 500;
  color: #666;
  border-bottom: 3px solid transparent;
  margin-bottom: -2px;
  transition: all 0.2s;
}

.tab:hover:not(:disabled) {
  color: #3498db;
}

.tab.active {
  color: #3498db;
  border-bottom-color: #3498db;
}

.tab:disabled {
  color: #ccc;
  cursor: not-allowed;
}

.tab-content {
  min-height: 400px;
}

.loading-describe, .error-describe {
  text-align: center;
  padding: 3rem;
  color: #666;
}

.error-describe {
  color: #e74c3c;
}

.describe-output, .yaml-output {
  background: #f8f9fa;
  border: 1px solid #e0e0e0;
  border-radius: 4px;
  padding: 1.5rem;
  overflow-x: auto;
  font-family: 'Courier New', monospace;
  font-size: 0.875rem;
  line-height: 1.6;
  margin: 0;
  max-height: 600px;
  overflow-y: auto;
}

.describe-output {
  white-space: pre-wrap;
  word-wrap: break-word;
}

.yaml-output {
  white-space: pre;
}
.events-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.events-table th {
  text-align: left;
  padding: 0.75rem;
  background: #f5f5f5;
  border-bottom: 2px solid #ddd;
  font-weight: 600;
  color: #333;
}

.events-table td {
  padding: 0.75rem;
  border-bottom: 1px solid #eee;
  vertical-align: top;
}

.events-table .message-cell {
  max-width: 400px;
  word-wrap: break-word;
}

.event-warning {
  background-color: #fff3cd;
}

.event-error {
  background-color: #f8d7da;
}

.event-normal {
  background-color: white;
}

.no-events {
  padding: 2rem;
  text-align: center;
  color: #666;
  font-style: italic;
}
</style>
