<script setup lang="ts">
// App.vue — Root component
import { onMounted, onUnmounted, nextTick } from 'vue'
import AppLayout from '@/layouts/AppLayout.vue'
import BaseToast from '@/components/feedback/BaseToast.vue'
import { useAppStore } from '@/stores/app.store'
import { useUploadLogStore } from '@/stores/uploadLog.store'
import { useAccountsStore } from '@/modules/accounts/store/useAccountsStore'

// Listen system:memory-warning từ Go memory watchdog (Phase 1 stability).
// Khi Go heap > 500 MB → notify user warning để cân nhắc restart sau batch.
const appStore = useAppStore()
const uploadLogStore = useUploadLogStore()
const accountsStore = useAccountsStore()
let lastMemWarnAt = 0
const MEM_WARN_COOLDOWN = 10 * 60 * 1000 // 10 phút — tránh spam notify

function onMemoryWarning(payload: { heapMB?: number; msg?: string }) {
  const now = Date.now()
  if (now - lastMemWarnAt < MEM_WARN_COOLDOWN) return
  lastMemWarnAt = now
  appStore.notify('warning', payload?.msg || `⚠️ RAM app cao (${payload?.heapMB || '?'} MB) — cân nhắc restart`)
}

// Soft UI cleanup (mỗi 12h) — KHÔNG reload Vue (giữ tab/scroll/state đang xem).
// Chỉ dọn buffer log + cache cũ → nhẹ Chromium JS heap mà KHÔNG mất UX.
//
// Trước đây code gọi window.location.reload() → user mất state đang nhập/xem.
// Giờ chỉ clear logs cũ + emit signal cho các page tự dọn cache → seamless.
async function onUIReload() {
  // 1. Clear log buffer của upload (lớn nhất, ~200 entries text)
  uploadLogStore.logs = []

  // 2. Clear cache proxy chạy realtime (sẽ tự rebuild khi có event mới)
  if ((accountsStore as any).runProxyCache?.clear) {
    (accountsStore as any).runProxyCache.clear()
  }
  if ((accountsStore as any).displayProxyCache?.clear) {
    (accountsStore as any).displayProxyCache.clear()
  }

  // 3. Phát signal cho các page tự dọn local state (registerLogs, scroll buffers...)
  window.dispatchEvent(new CustomEvent('app:soft-cleanup'))

  // 4. Trigger Vue flush + nhường tick cho V8 GC
  await nextTick()

  // 5. Notify nhẹ — không gây giật
  appStore.notify('info', '🧹 Đã dọn buffer giao diện (không reload)')
}

// Task 6: lưu unsub fn từ Wails EventsOn để cleanup chính xác per-listener.
// Trước đây gọi EventsOff(name) global → xóa MỌI listener của event đó (kể cả của
// component khác cùng nghe). Giữ unsub fn → chỉ xóa đúng listener của App.vue.
let unsubMemWarn: (() => void) | null = null
let unsubUIReload: (() => void) | null = null

// Notify backend khi window minimize/restore — backend dùng để throttle batch
// emitter (300ms → 2s) khi hidden, tiết kiệm IPC + JSON serialize CPU.
function notifyVisibility() {
  const w = window as any
  const fn = w?.go?.main?.App?.NotifyVisibilityChange
  if (typeof fn !== 'function') return
  try { fn(document.hidden) } catch { /* ignore */ }
}

onMounted(() => {
  const w = window as any
  if (w?.runtime?.EventsOn) {
    const u1 = w.runtime.EventsOn('system:memory-warning', onMemoryWarning)
    const u2 = w.runtime.EventsOn('system:ui-reload', onUIReload)
    if (typeof u1 === 'function') unsubMemWarn = u1
    if (typeof u2 === 'function') unsubUIReload = u2
  }
  // Visibility tracking — gọi NotifyVisibilityChange ngay + lắng nghe thay đổi
  notifyVisibility()
  document.addEventListener('visibilitychange', notifyVisibility)
})
onUnmounted(() => {
  unsubMemWarn?.()
  unsubUIReload?.()
  unsubMemWarn = null
  unsubUIReload = null
  document.removeEventListener('visibilitychange', notifyVisibility)
})
</script>

<template>
  <AppLayout />
  <BaseToast />
</template>
