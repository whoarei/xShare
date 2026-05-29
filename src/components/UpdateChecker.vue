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
let updateInstance = null

async function checkForUpdate(silent = true) {
  if (checking.value || downloading.value) return
  checking.value = true
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
    } else if (!silent) {
      updateAvailable.value = false
    }
  } catch (e) {
    if (!silent) throw e
  } finally {
    checking.value = false
  }
}

async function installAndRestart() {
  if (!updateInstance) return
  try {
    await updateInstance.install()
  } catch (e) {
    throw e
  }
  await relaunch()
}

provide('updateChecker', {
  updateAvailable,
  downloading,
  downloaded,
  newVersion,
  updateBody,
  checking,
  checkForUpdate,
  installAndRestart
})

defineExpose({
  checkForUpdate,
  updateAvailable,
  downloaded
})
</script>

<template>
  <slot />
</template>
