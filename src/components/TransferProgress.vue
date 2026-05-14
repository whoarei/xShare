<script setup>
import { computed } from 'vue'

const props = defineProps({
  active: { type: Boolean, default: false },
  task: { type: Object, default: null },
  items: { type: Array, default: () => [] },
})

const dirCount = computed(() => props.items.filter(i => i.kind === 'mkdir').length)
const fileCount = computed(() => props.items.filter(i => i.kind === 'file' && i.done).length)
const currentFile = computed(() => {
  const files = props.items.filter(i => i.kind === 'file' && !i.done)
  return files.length > 0 ? files[files.length - 1] : null
})

const totalItems = computed(() => props.task?.item_count || 0)
const completedItems = computed(() => dirCount.value + fileCount.value)
const progressPercent = computed(() => {
  if (!totalItems.value || totalItems.value === 0) return 0
  return Math.round((completedItems.value / totalItems.value) * 100)
})
</script>

<template>
  <div
    v-if="active"
    class="p-4 border-b border-gray-100"
  >
    <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide mb-3">Transfer Progress</h2>

    <!-- Task Info -->
    <div v-if="task" class="mb-3 p-3 bg-primary-50 rounded-lg">
      <div class="flex items-center justify-between mb-1">
        <span class="text-sm font-medium text-primary-700">Task: {{ task.id }}</span>
        <span class="text-xs text-primary-500">{{ completedItems }} / {{ totalItems }}</span>
      </div>
      <div class="w-full bg-primary-200 rounded-full h-2">
        <div
          class="bg-primary-600 h-2 rounded-full transition-all duration-300"
          :style="{ width: progressPercent + '%' }"
        ></div>
      </div>
    </div>

    <!-- Current File -->
    <div v-if="currentFile" class="text-sm text-gray-600 mb-2">
      <span class="text-gray-400">Transferring:</span>
      <span class="font-medium ml-1 truncate">{{ currentFile.path }}</span>
    </div>

    <!-- Status -->
    <div v-if="!task && items.length === 0" class="text-sm text-gray-400 text-center py-2">
      Waiting for transfer to start...
    </div>

    <!-- Recent Items -->
    <div v-if="items.length > 0" class="max-h-32 overflow-y-auto scrollbar-thin space-y-0.5">
      <div
        v-for="(item, i) in items.slice().reverse().slice(0, 20)"
        :key="i"
        class="flex items-center gap-2 text-xs"
      >
        <span
          v-if="item.kind === 'mkdir'"
          class="w-1.5 h-1.5 rounded-full bg-yellow-400 shrink-0"
        ></span>
        <span
          v-else-if="item.done"
          class="w-1.5 h-1.5 rounded-full bg-emerald-400 shrink-0"
        ></span>
        <span
          v-else
          class="w-1.5 h-1.5 rounded-full bg-blue-400 animate-pulse shrink-0"
        ></span>
        <span class="text-gray-500 truncate">{{ item.path }}</span>
        <span class="text-gray-400 shrink-0">{{ item.kind }}</span>
      </div>
    </div>
  </div>
</template>
