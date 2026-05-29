<script setup>
// xShare主界面组件
import { ref, reactive, onMounted, onUnmounted, computed } from 'vue'

// 应用版本号 (由 vite.config.js 在构建时注入)
const appVersion = import.meta.env.VITE_APP_VERSION || 'dev'
import { invoke } from '@tauri-apps/api/core'
import { listen } from '@tauri-apps/api/event'
import DeviceList from './components/DeviceList.vue'
import FileSelector from './components/FileSelector.vue'
import TransferProgress from './components/TransferProgress.vue'
import About from './components/About.vue'
import Changelog from './components/Changelog.vue'

// 关于页面状态
const showAbout = ref(false)
const showChangelog = ref(false)

// 服务器状态
const serverRunning = ref(false)
const serverPort = ref(9527)
const serverDir = ref('./received')
const serverOutput = ref([])

// 保存目录历史记录
const saveDirHistory = ref([])
const showHistory = ref(false)

// 网络接口配置
const networkInterfaces = ref([])
const selectedInterface = ref('')
const availableIPs = ref([])
const selectedIP = ref('')

// 设备发现状态
const peers = ref([])
const discovering = ref(false)

// 文件选择和发送状态
const selectedItems = ref([])
const selectedPeer = ref(null)
const sending = ref(false)

// 传输进度状态
const transferActive = ref(false)
const currentTask = ref(null)
const progressItems = ref([])

// 日志记录
const logs = ref([])

// 事件监听器清理函数
let unlisteners = []

// addLog 添加日志记录
function addLog(msg, type = 'info') {
  logs.value.unshift({ time: new Date().toLocaleTimeString(), msg, type })
  if (logs.value.length > 200) logs.value.pop()
}

// loadSettings 加载应用设置
async function loadSettings() {
  try {
    const settings = await invoke('load_settings')
    if (settings.saveDir) serverDir.value = settings.saveDir
    saveDirHistory.value = settings.saveDirHistory || []
  } catch (e) {
    addLog('Failed to load settings: ' + e, 'warn')
  }
}

// saveSettings 保存应用设置
async function saveSettings() {
  try {
    await invoke('save_settings', {
      saveDir: serverDir.value,
      history: saveDirHistory.value
    })
  } catch (e) {
    addLog('Failed to save settings: ' + e, 'warn')
  }
}

// addToHistory 添加目录到历史记录
function addToHistory(path) {
  const now = new Date().toISOString()
  const existing = saveDirHistory.value.findIndex(h => h.path === path)
  if (existing >= 0) {
    saveDirHistory.value[existing].lastUsed = now
  } else {
    saveDirHistory.value.unshift({ path, lastUsed: now })
  }
  if (saveDirHistory.value.length > 100) {
    saveDirHistory.value = saveDirHistory.value.slice(0, 100)
  }
  saveSettings()
}

// selectHistory 从历史记录选择目录
function selectHistory(path) {
  serverDir.value = path
  showHistory.value = false
  addToHistory(path)
}

// formatDate 格式化日期显示
function formatDate(isoString) {
  if (!isoString) return ''
  const date = new Date(isoString)
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

// startServer 启动文件接收服务器
async function startServer() {
  try {
    const result = await invoke('start_server', {
      port: serverPort.value,
      dir: serverDir.value,
      ip: selectedIP.value || null
    })
    serverRunning.value = true
    addLog('Server started: ' + result, 'success')
  } catch (e) {
    addLog('Failed to start server: ' + e, 'error')
  }
}

// stopServer 停止文件接收服务器
async function stopServer() {
  try {
    const result = await invoke('stop_server')
    serverRunning.value = false
    addLog('Server stopped: ' + result, 'success')
  } catch (e) {
    addLog('Failed to stop server: ' + e, 'error')
  }
}

// discoverPeers 发现局域网内的其他xShare设备
async function discoverPeers() {
  discovering.value = true
  try {
    const output = await invoke('discover_peers', {
      timeout: 5,
      ip: selectedIP.value || null
    })
    const data = JSON.parse(output)
    if (data.type === 'peers') {
      peers.value = data.peers || []
      addLog(`Found ${peers.value.length} peer(s)`, 'success')
    }
  } catch (e) {
    addLog('Discovery failed: ' + e, 'error')
  } finally {
    discovering.value = false
  }
}

// browseFiles 打开文件选择对话框
async function browseFiles() {
  try {
    const path = await invoke('open_file_dialog')
    if (path && !selectedItems.value.some(i => i.path === path)) {
      selectedItems.value.push({ path, isFile: true })
    }
  } catch (e) {
    addLog('File dialog error: ' + e, 'error')
  }
}

// browseDir 打开目录选择对话框
async function browseDir() {
  try {
    const path = await invoke('open_dir_dialog')
    if (path && !selectedItems.value.some(i => i.path === path)) {
      selectedItems.value.push({ path, isFile: false })
    }
  } catch (e) {
    addLog('Directory dialog error: ' + e, 'error')
  }
}

// browseSaveDir 打开目录选择对话框用于选择保存目录
async function browseSaveDir() {
  if (transferActive.value) {
    addLog('Cannot change directory during transfer', 'error')
    return
  }
  try {
    const path = await invoke('open_dir_dialog', { dir: serverDir.value })
    if (path) {
      serverDir.value = path
      addToHistory(path)
      if (serverRunning.value) {
        addLog('Directory changed, restarting server...', 'info')
        await stopServer()
        await startServer()
      }
    }
  } catch (e) {
    addLog('Directory dialog error: ' + e, 'error')
  }
}

// removePath 从选择列表中移除路径
function removePath(index) {
  selectedItems.value.splice(index, 1)
}

// sendFiles 发送选中的文件到目标设备
async function sendFiles() {
  if (!selectedPeer.value || selectedItems.value.length === 0) return
  sending.value = true
  transferActive.value = true
  currentTask.value = null
  progressItems.value = []

  try {
    for (const item of selectedItems.value) {
      await invoke('send_files', {
        peer: selectedPeer.value,
        path: item.path
      })
    }
  } catch (e) {
    addLog('Send error: ' + e, 'error')
    sending.value = false
    transferActive.value = false
  }
}

// handleProgress 处理传输进度事件
function handleProgress(payload) {
  try {
    const data = JSON.parse(payload)
    switch (data.type) {
      case 'task':
        currentTask.value = data
        addLog(`Receiving task: ${data.id} (${data.item_count} items, ${formatSize(data.total_size)})`)
        break
      case 'progress':
        progressItems.value.push(data)
        break
      case 'complete':
        addLog('Transfer complete!', 'success')
        transferActive.value = false
        sending.value = false
        break
      case 'error':
        addLog('Transfer error: ' + data.msg, 'error')
        transferActive.value = false
        sending.value = false
        break
    }
  } catch {}
}

// handleServerEvent 处理服务器事件
function handleServerEvent(payload) {
  try {
    const data = JSON.parse(payload)
    if (data.type === 'ready') {
      serverRunning.value = true
      addLog(`Server ready on port ${data.port}, dir: ${data.dir}`, 'success')
    } else if (data.type === 'task') {
      currentTask.value = data
      transferActive.value = true
    } else if (data.type === 'progress') {
      progressItems.value.push(data)
    } else if (data.type === 'complete') {
      addLog('Receive complete!', 'success')
      transferActive.value = false
    } else if (data.type === 'error') {
      addLog('Server error: ' + data.msg, 'error')
    } else {
      serverOutput.value.unshift(payload)
      if (serverOutput.value.length > 100) serverOutput.value.pop()
    }
  } catch {
    serverOutput.value.unshift(payload)
    if (serverOutput.value.length > 100) serverOutput.value.pop()
  }
}

// handleServerError 处理服务器错误事件
function handleServerError(payload) {
  addLog('Server error: ' + payload, 'error')
}

// handleTransferError 处理传输错误事件
function handleTransferError(payload) {
  addLog('Transfer error: ' + payload, 'error')
}

// handleTransferComplete 处理传输完成事件
function handleTransferComplete(payload) {
  addLog(`Transfer finished with code: ${payload.code || 0}`)
  sending.value = false
  transferActive.value = false
}

// listIPs 获取本机可用IP地址列表
async function listIPs() {
  try {
    const output = await invoke('list_ips')
    const data = JSON.parse(output)
    if (data.type === 'ips') {
      availableIPs.value = data.ips || []
      addLog(`Found ${availableIPs.value.length} IP address(es)`, 'info')
    }
  } catch (e) {
    addLog('Failed to list IPs: ' + e, 'warn')
  }
}

// formatSize 格式化文件大小显示
function formatSize(bytes) {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(1)} ${units[i]}`
}

// canSend 计算属性：是否可以发送文件
const canSend = computed(() => !!selectedPeer.value && selectedItems.value.length > 0 && !sending.value)

// closeHistoryDropdown 关闭历史记录下拉菜单
function closeHistoryDropdown(e) {
  if (!e.target.closest('.relative')) {
    showHistory.value = false
  }
}

// onMounted 组件挂载时初始化事件监听
onMounted(async () => {
  loadSettings()
  listIPs()
  document.addEventListener('click', closeHistoryDropdown)
  unlisteners.push(
    await listen('transfer-progress', (e) => handleProgress(e.payload)),
    await listen('transfer-error', (e) => handleTransferError(e.payload)),
    await listen('transfer-complete', (e) => handleTransferComplete(e.payload)),
    await listen('server-event', (e) => handleServerEvent(e.payload)),
    await listen('server-error', (e) => handleServerError(e.payload)),
    await listen('server-terminated', () => {
      serverRunning.value = false
      addLog('Server process terminated', 'warn')
    })
  )
})

// onUnmounted 组件卸载时清理事件监听
onUnmounted(() => {
  unlisteners.forEach(fn => fn())
  document.removeEventListener('click', closeHistoryDropdown)
})
</script>

<template>
  <div class="flex flex-col h-screen bg-gray-50">
    <!-- Header -->
    <header class="flex items-center justify-between px-6 py-3 bg-white border-b border-gray-200 shadow-sm">
      <div class="flex items-center gap-3">
        <div class="flex items-center gap-2">
          <div class="w-8 h-8 bg-primary-600 rounded-lg flex items-center justify-center">
            <span class="text-white font-bold text-sm">xS</span>
          </div>
          <h1 class="text-xl font-bold text-gray-800">xShare</h1>
        </div>
        <span class="text-xs text-gray-400 bg-gray-100 px-2 py-0.5 rounded">v{{ appVersion }}</span>
      </div>
      <div class="flex items-center gap-4">
        <button @click="showAbout = true" class="text-sm text-gray-500 hover:text-gray-700 transition-colors">关于</button>
        <div class="flex items-center gap-2">
          <span
            class="w-2 h-2 rounded-full"
            :class="serverRunning ? 'bg-emerald-500 animate-pulse' : 'bg-gray-300'"
          ></span>
          <span class="text-sm text-gray-600">
            {{ serverRunning ? `Listening :${serverPort}` : 'Server stopped' }}
          </span>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <div class="flex flex-1 overflow-hidden">
      <!-- Left Panel -->
      <aside class="w-80 flex flex-col border-r border-gray-200 bg-white">
        <!-- Server Controls -->
        <div class="p-4 border-b border-gray-100">
          <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">Server</h2>
          <div class="space-y-2">
            <div class="flex gap-2">
              <input
                v-model.number="serverPort"
                type="number"
                class="input w-24"
                placeholder="Port"
                :disabled="serverRunning"
              />
              <div class="flex gap-1 flex-1 relative">
                <input
                  v-model="serverDir"
                  type="text"
                  class="input flex-1"
                  placeholder="Receive directory"
                  @change="saveSettings"
                />
                <button
                  @click="showHistory = !showHistory"
                  class="btn-secondary px-2"
                  title="History"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                </button>
                <button
                  @click="browseSaveDir"
                  class="btn-secondary px-2"
                  title="Browse directory"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                  </svg>
                </button>
                <!-- 历史记录下拉菜单 -->
                <div
                  v-if="showHistory"
                  class="absolute top-full left-0 right-0 z-10 mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-60 overflow-y-auto"
                >
                  <div
                    v-for="item in saveDirHistory"
                    :key="item.path"
                    @click="selectHistory(item.path)"
                    class="px-3 py-2 hover:bg-gray-100 cursor-pointer text-sm border-b border-gray-100 last:border-b-0"
                  >
                    <div class="truncate text-gray-800">{{ item.path }}</div>
                    <div class="text-xs text-gray-400">{{ formatDate(item.lastUsed) }}</div>
                  </div>
                  <div v-if="saveDirHistory.length === 0" class="px-3 py-2 text-gray-400 text-sm italic">
                    No history
                  </div>
                </div>
              </div>
            </div>
            <div class="flex gap-2">
              <button
                v-if="!serverRunning"
                @click="startServer"
                class="btn-primary flex-1 text-sm"
              >
                Start Server
              </button>
              <button
                v-else
                @click="stopServer"
                class="btn-danger flex-1 text-sm"
              >
                Stop Server
              </button>
            </div>
            <div class="pt-1">
              <label class="text-xs text-gray-400 mb-1 block">Bind IP</label>
              <select
                v-model="selectedIP"
                class="input w-full text-sm"
                :disabled="serverRunning"
              >
                <option value="">Auto (default)</option>
                <option
                  v-for="info in availableIPs"
                  :key="info.ip"
                  :value="info.ip"
                >
                  {{ info.ip }} ({{ info.iface }}, {{ info.family }})
                </option>
              </select>
            </div>
          </div>
        </div>

        <!-- Device List -->
        <DeviceList
          v-model:selectedPeer="selectedPeer"
          :peers="peers"
          :discovering="discovering"
          @discover="discoverPeers"
        />

        <!-- Logs -->
        <div class="flex-1 p-4 overflow-hidden flex flex-col min-h-0">
          <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-2">Log</h2>
          <div class="flex-1 overflow-y-auto scrollbar-thin bg-gray-50 rounded-lg p-2 text-xs font-mono">
            <div
              v-for="(log, i) in logs"
              :key="i"
              class="py-0.5"
              :class="{
                'text-emerald-600': log.type === 'success',
                'text-red-600': log.type === 'error',
                'text-yellow-600': log.type === 'warn',
                'text-gray-500': log.type === 'info'
              }"
            >
              <span class="text-gray-400">{{ log.time }}</span>
              {{ log.msg }}
            </div>
            <div v-if="logs.length === 0" class="text-gray-400 italic">No logs yet</div>
          </div>
        </div>
      </aside>

      <!-- Right Panel -->
      <main class="flex-1 flex flex-col overflow-hidden">
        <!-- File Selector & Send -->
        <FileSelector
          v-model:selectedItems="selectedItems"
          v-model:selectedPeer="selectedPeer"
          :peers="peers"
          :sending="sending"
          :canSend="canSend"
          @browse-files="browseFiles"
          @browse-dir="browseDir"
          @remove-path="removePath"
          @send="sendFiles"
        />

        <!-- Transfer Progress -->
        <TransferProgress
          :active="transferActive"
          :task="currentTask"
          :items="progressItems"
        />

        <!-- Server Output (receive side) -->
        <div v-if="serverRunning && serverOutput.length > 0" class="flex-1 p-4 overflow-hidden flex flex-col min-h-0">
          <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-2">Server Events</h2>
          <div class="flex-1 overflow-y-auto scrollbar-thin bg-gray-50 rounded-lg p-2 text-xs font-mono">
            <div v-for="(line, i) in serverOutput" :key="i" class="py-0.5 text-gray-600">
              {{ line }}
            </div>
          </div>
        </div>
      </main>
    </div>
  </div>
  <About :visible="showAbout" @close="showAbout = false" @show-changelog="showAbout = false; showChangelog = true" />
  <Changelog :visible="showChangelog" @close="showChangelog = false" />
</template>
