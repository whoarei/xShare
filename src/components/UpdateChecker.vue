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
    const update = await check({ timeout: 30000 })
    if (update) {
      updateAvailable.value = true
      newVersion.value = update.version
      updateBody.value = update.body || ''
      updateInstance = update
      if (!downloaded.value) {
        downloading.value = true
        await update.download(() => {}, { timeout: 60000 })
        downloading.value = false
        downloaded.value = true
      }
    } else {
      noUpdate.value = true
    }
  } catch (e) {
    console.error('[UpdateChecker] Check for update failed:', e)
    const msg = e?.message || String(e)
    if (msg.includes('timeout') || msg.includes('timed out')) {
      error.value = '网络超时，请检查网络连接或代理设置'
    } else if (msg.includes('release JSON') || msg.includes('404')) {
      error.value = '未找到可用的更新源，请确认已发布包含 latest.json 的版本'
    } else {
      error.value = msg || '检查更新失败'
    }
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
  installAndRestart,
  updateAvailable,
  downloaded,
  checking,
  error,
  noUpdate,
  newVersion
})
</script>

<template>
  <slot />
</template>
