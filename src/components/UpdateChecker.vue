<script setup>
import { ref, provide } from 'vue'
import { check } from '@tauri-apps/plugin-updater'
import { relaunch } from '@tauri-apps/plugin-process'

const updateAvailable = ref(false)
const downloading = ref(false)
const downloaded = ref(false)
const newVersion = ref('')
const updateBody = ref('')
const checking = ref(false)
const error = ref('')
const noUpdate = ref(false)
let updateInstance = null

async function checkForUpdate(silent = true) {
  if (checking.value || downloading.value) return
  checking.value = true
  error.value = ''
  noUpdate.value = false
  try {
    const update = await check()
    if (update) {
      updateAvailable.value = true
      newVersion.value = update.version
      updateBody.value = update.body || ''
      updateInstance = update
      if (!downloaded.value) {
        downloading.value = true
        await update.download(() => {})
        downloading.value = false
        downloaded.value = true
      }
    } else {
      noUpdate.value = true
    }
  } catch (e) {
    error.value = e?.message || String(e) || '检查更新失败'
  } finally {
    checking.value = false
  }
}

async function installAndRestart() {
  if (!updateInstance) return
  await updateInstance.install()
  await relaunch()
}

provide('updateChecker', {
  updateAvailable,
  downloading,
  downloaded,
  newVersion,
  updateBody,
  checking,
  error,
  noUpdate,
  checkForUpdate,
  installAndRestart
})

defineExpose({
  checkForUpdate,
  updateAvailable,
  downloaded,
  error,
  noUpdate
})
</script>

<template>
  <slot />
</template>
