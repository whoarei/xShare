<script setup>
defineProps({
  peers: { type: Array, default: () => [] },
  discovering: { type: Boolean, default: false },
  selectedPeer: { type: String, default: null },
})

defineEmits(['discover', 'update:selectedPeer'])
</script>

<template>
  <div class="p-4 border-b border-gray-100">
    <div class="flex items-center justify-between mb-3">
      <h2 class="text-sm font-semibold text-gray-500 uppercase tracking-wide">Peers</h2>
      <button
        @click="$emit('discover')"
        :disabled="discovering"
        class="btn-primary text-xs py-1 px-3"
      >
        <span v-if="discovering" class="inline-flex items-center gap-1">
          <svg class="animate-spin h-3 w-3" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
          </svg>
          Scanning...
        </span>
        <span v-else>Discover</span>
      </button>
    </div>
    <div v-if="peers.length === 0" class="text-sm text-gray-400 text-center py-4">
      <p v-if="discovering">Searching for peers...</p>
      <p v-else>No peers discovered. Click Discover to scan.</p>
    </div>
    <div v-else class="space-y-1 max-h-48 overflow-y-auto scrollbar-thin">
      <button
        v-for="peer in peers"
        :key="peer.addr"
        @click="$emit('update:selectedPeer', peer.addr)"
        class="w-full text-left px-3 py-2 rounded-lg text-sm transition-colors"
        :class="selectedPeer === peer.addr
          ? 'bg-primary-100 text-primary-700 ring-1 ring-primary-300'
          : 'hover:bg-gray-100 text-gray-700'"
      >
        <div class="font-medium truncate">{{ peer.name }}</div>
        <div class="text-xs text-gray-400">{{ peer.addr }}</div>
      </button>
    </div>
  </div>
</template>
