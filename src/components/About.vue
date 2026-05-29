<script setup>
defineProps({
  visible: {
    type: Boolean,
    default: false
  },
  updateDownloaded: {
    type: Boolean,
    default: false
  },
  updateDownloading: {
    type: Boolean,
    default: false
  },
  updateChecking: {
    type: Boolean,
    default: false
  },
  newVersion: {
    type: String,
    default: ''
  }
})

const emit = defineEmits(['close', 'show-changelog', 'check-update', 'install-update'])

const appVersion = import.meta.env.VITE_APP_VERSION || 'dev'

function close() {
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="visible" class="fixed inset-0 z-50 flex items-center justify-center" @click.self="close">
        <div class="absolute inset-0 bg-black/50 backdrop-blur-sm"></div>
        <div class="relative bg-white rounded-2xl shadow-xl max-w-md w-full mx-4 overflow-hidden">
          <div class="p-6 text-center">
            <div class="w-16 h-16 bg-primary-600 rounded-2xl flex items-center justify-center mx-auto mb-4">
              <span class="text-white font-bold text-2xl">xS</span>
            </div>
            <h2 class="text-2xl font-bold text-gray-800 mb-1">xShare</h2>
            <p class="text-sm text-gray-500 mb-4">局域网文件传输工具</p>
            <div class="inline-flex items-center gap-1.5 bg-gray-100 rounded-full px-3 py-1 mb-6">
              <span class="text-xs font-medium text-gray-600">版本</span>
              <span class="text-xs font-bold text-primary-600">v{{ appVersion }}</span>
            </div>
            <div class="text-left space-y-3 mb-6">
              <div class="flex items-start gap-3">
                <div class="w-8 h-8 bg-blue-100 rounded-lg flex items-center justify-center shrink-0">
                  <svg class="w-4 h-4 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path>
                  </svg>
                </div>
                <div>
                  <p class="text-sm font-medium text-gray-700">快速传输</p>
                  <p class="text-xs text-gray-500">基于局域网的高速文件传输</p>
                </div>
              </div>
              <div class="flex items-start gap-3">
                <div class="w-8 h-8 bg-green-100 rounded-lg flex items-center justify-center shrink-0">
                  <svg class="w-4 h-4 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9"></path>
                  </svg>
                </div>
                <div>
                  <p class="text-sm font-medium text-gray-700">跨平台</p>
                  <p class="text-xs text-gray-500">支持 Windows、macOS 和 Linux</p>
                </div>
              </div>
              <div class="flex items-start gap-3">
                <div class="w-8 h-8 bg-purple-100 rounded-lg flex items-center justify-center shrink-0">
                  <svg class="w-4 h-4 text-purple-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"></path>
                  </svg>
                </div>
                <div>
                  <p class="text-sm font-medium text-gray-700">安全可靠</p>
                  <p class="text-xs text-gray-500">点对点传输，无需云端服务器</p>
                </div>
              </div>
            </div>
            <div class="bg-gray-50 rounded-lg p-3 mb-6">
              <p class="text-xs text-gray-500 mb-2">技术栈</p>
              <div class="flex flex-wrap justify-center gap-2">
                <span class="px-2 py-1 bg-green-100 text-green-700 rounded text-xs font-medium">Vue 3</span>
                <span class="px-2 py-1 bg-blue-100 text-blue-700 rounded text-xs font-medium">Tauri v2</span>
                <span class="px-2 py-1 bg-cyan-100 text-cyan-700 rounded text-xs font-medium">Go</span>
                <span class="px-2 py-1 bg-purple-100 text-purple-700 rounded text-xs font-medium">Tailwind CSS</span>
              </div>
            </div>
            <p class="text-xs text-gray-400">Copyright © 2024 xShare Contributors</p>
            <div class="mt-3 flex flex-col items-center gap-2">
              <button @click="emit('show-changelog')" class="text-xs text-primary-600 hover:text-primary-700 underline">查看更新日志</button>
              <div v-if="updateDownloaded" class="flex flex-col items-center gap-2">
                <span class="text-xs text-gray-500">新版本 v{{ newVersion }} 已就绪</span>
                <button @click="emit('install-update')" class="btn-primary text-sm px-4 py-1.5">安装并重启</button>
              </div>
              <div v-else-if="updateDownloading" class="flex items-center gap-2 text-xs text-gray-500">
                <div class="animate-spin rounded-full h-3.5 w-3.5 border-b-2 border-primary-600"></div>
                正在下载新版本...
              </div>
              <div v-else-if="updateChecking" class="flex items-center gap-2 text-xs text-gray-500">
                <div class="animate-spin rounded-full h-3.5 w-3.5 border-b-2 border-primary-600"></div>
                检查更新中...
              </div>
              <button v-else @click="emit('check-update')" class="text-xs text-gray-500 hover:text-primary-600 transition-colors">检查更新</button>
            </div>
          </div>
          <div class="border-t border-gray-100 px-6 py-4">
            <button @click="close" class="btn-primary w-full">关闭</button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.3s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-active .relative,
.modal-leave-active .relative {
  transition: transform 0.3s ease, opacity 0.3s ease;
}

.modal-enter-from .relative,
.modal-leave-to .relative {
  transform: scale(0.95);
  opacity: 0;
}
</style>
