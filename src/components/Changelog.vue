<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  visible: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['close'])
const htmlContent = ref('')
const loading = ref(false)
const error = ref('')

async function loadChangelog() {
  if (htmlContent.value) return
  loading.value = true
  error.value = ''
  try {
    const { readTextFile, BaseDirectory } = await import('@tauri-apps/plugin-fs')
    htmlContent.value = await readTextFile('resources/CHANGELOG.html', { baseDir: BaseDirectory.Resource })
  } catch (e) {
    error.value = '无法加载更新日志'
    console.error('Failed to read CHANGELOG.html:', e)
  } finally {
    loading.value = false
  }
}

function close() {
  emit('close')
}

watch(() => props.visible, (val) => {
  if (val) loadChangelog()
}, { immediate: true })
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="visible" class="fixed inset-0 z-50 flex items-center justify-center" @click.self="close">
        <div class="absolute inset-0 bg-black/50 backdrop-blur-sm"></div>
        <div class="relative bg-white rounded-2xl shadow-xl max-w-2xl w-full mx-4 max-h-[80vh] flex flex-col overflow-hidden">
          <div class="px-6 py-4 border-b border-gray-100 flex items-center justify-between shrink-0">
            <h2 class="text-lg font-bold text-gray-800">更新日志</h2>
            <button @click="close" class="text-gray-400 hover:text-gray-600 transition-colors">
              <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
              </svg>
            </button>
          </div>
          <div class="flex-1 overflow-y-auto px-6 py-4">
            <div v-if="loading" class="flex items-center justify-center py-12">
              <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600"></div>
            </div>
            <div v-else-if="error" class="text-center py-12 text-gray-500">
              {{ error }}
            </div>
            <div v-else class="changelog-content" v-html="htmlContent"></div>
          </div>
          <div class="border-t border-gray-100 px-6 py-4 shrink-0">
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

.changelog-content :deep(h1) {
  display: none;
}

.changelog-content :deep(h2) {
  font-size: 1.125rem;
  font-weight: 700;
  color: #111827;
  margin: 1.25rem 0 0.5rem;
  padding-bottom: 0.375rem;
  border-bottom: 1px solid #e5e7eb;
}

.changelog-content :deep(h3) {
  font-size: 0.9375rem;
  font-weight: 600;
  color: #374151;
  margin: 1rem 0 0.375rem;
}

.changelog-content :deep(ul) {
  padding-left: 1.25rem;
  margin: 0.375rem 0;
}

.changelog-content :deep(li) {
  font-size: 0.875rem;
  color: #4b5563;
  margin: 0.25rem 0;
  line-height: 1.5;
}

.changelog-content :deep(code) {
  font-family: "Cascadia Code", "Fira Code", Consolas, monospace;
  background: #f3f4f6;
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 0.8125rem;
}

.changelog-content :deep(a) {
  color: #2563eb;
  text-decoration: none;
}

.changelog-content :deep(a:hover) {
  text-decoration: underline;
}
</style>
