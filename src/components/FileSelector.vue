<script setup>
// 文件选择器组件 - 选择要发送的文件和目标设备
defineProps({
  selectedItems: { type: Array, default: () => [] },    // 选中的文件列表
  selectedPeer: { type: String, default: null },         // 选中的目标设备
  peers: { type: Array, default: () => [] },             // 可用设备列表
  sending: { type: Boolean, default: false },            // 是否正在发送
  canSend: { type: Boolean, default: false },            // 是否可以发送
})

defineEmits(['update:selectedItems', 'update:selectedPeer', 'browse-files', 'browse-dir', 'remove-path', 'send'])
</script>

<template>
  <div class="p-4 border-b border-gray-100">
    <div class="flex items-center gap-3 mb-3">
      <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide flex-1">Send Files</h2>
      <select
        class="input w-48 text-sm"
        :value="selectedPeer"
        @change="$emit('update:selectedPeer', $event.target.value)"
      >
        <option value="">Select peer...</option>
        <option v-for="p in peers" :key="p.addr" :value="p.addr">
          {{ p.name }} ({{ p.addr }})
        </option>
      </select>
    </div>

    <div class="flex gap-2 mb-3">
      <button @click="$emit('browse-files')" class="btn-secondary text-sm flex-1">
        <svg class="w-4 h-4 mr-1 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        Add Files
      </button>
      <button @click="$emit('browse-dir')" class="btn-secondary text-sm flex-1">
        <svg class="w-4 h-4 mr-1 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
        </svg>
        Add Folder
      </button>
    </div>

    <div v-if="selectedItems.length === 0" class="border-2 border-dashed border-gray-200 rounded-lg p-6 text-center">
      <p class="text-sm text-gray-400">Select files or folders to share</p>
      <p class="text-xs text-gray-300 mt-1">Use the buttons above to browse</p>
    </div>
    <div v-else class="space-y-1 max-h-40 overflow-y-auto scrollbar-thin mb-3">
      <div
        v-for="(item, i) in selectedItems"
        :key="item.path"
        class="flex items-center gap-2 px-3 py-2 bg-gray-50 rounded-lg text-sm"
      >
        <svg v-if="item.isFile" class="w-4 h-4 text-blue-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
        </svg>
        <svg v-else class="w-4 h-4 text-amber-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
        </svg>
        <span class="flex-1 truncate text-gray-700">{{ item.path }}</span>
        <button
          @click="$emit('remove-path', i)"
          class="text-gray-400 hover:text-red-500 shrink-0"
          :disabled="sending"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>

    <button
      @click="$emit('send')"
      :disabled="!canSend"
      class="btn-primary w-full"
      :class="{ 'opacity-50 cursor-not-allowed': !canSend }"
    >
      <span v-if="sending" class="inline-flex items-center gap-2">
        <svg class="animate-spin h-4 w-4" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
        Sending...
      </span>
      <span v-else>
        <svg class="w-4 h-4 mr-1 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
            d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
        </svg>
        Send
      </span>
    </button>
  </div>
</template>
