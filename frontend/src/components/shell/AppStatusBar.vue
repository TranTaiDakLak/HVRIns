<script setup lang="ts">
// AppStatusBar.vue - global bottom status bar.

import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { getBridgeMode, getResourceUsageService, getAppInfoService } from '@/bridge/client'
import { useAppStore } from '@/stores/app.store'
import { useUploadLogStore } from '@/stores/uploadLog.store'

const bridgeMode = getBridgeMode()
const appStore = useAppStore()
const uploadLogStore = useUploadLogStore()

const appVersion = ref('...')
const ramMb = ref(0)
const cpuPct = ref(0)
const cleaning = ref(false)
let timer: ReturnType<typeof setTimeout> | null = null
let windowHidden = false

async function softCleanup() {
  if (cleaning.value) return
  cleaning.value = true
  try {
    uploadLogStore.logs = []
    window.dispatchEvent(new CustomEvent('app:soft-cleanup'))
    await nextTick()
    const w = window as any
    if (typeof w?.go?.main?.App?.ForceMemoryCleanup === 'function') {
      const r = await w.go.main.App.ForceMemoryCleanup()
      appStore.notify('success', `Đã dọn ${r.iosSessionsClosed + r.androidSessionsClosed} session, giải phóng ${r.freedMB} MB`)
    } else {
      appStore.notify('success', 'Đã dọn buffer giao diện')
    }
    await fetchUsage()
  } finally {
    cleaning.value = false
  }
}

function hardReload() {
  if (!confirm('Reload UI sẽ mất state hiện tại (tab, scroll, log đang xem). Workers Go vẫn chạy. Tiếp tục?')) return
  window.location.reload()
}

async function fetchUsage() {
  try {
    const svc = await getResourceUsageService()
    const usage = await svc.get()
    ramMb.value = usage.ramMb
    cpuPct.value = usage.cpuPct
  } catch {
    // Mock mode or disconnected bridge.
  }
}

function nextInterval(): number {
  if (windowHidden) return 15_000
  if (cpuPct.value > 5) return 2_000
  return 5_000
}

function scheduleNext() {
  if (timer) clearTimeout(timer)
  timer = setTimeout(async () => {
    await fetchUsage()
    scheduleNext()
  }, nextInterval())
}

function onVisibilityChange() {
  windowHidden = document.hidden
}

onMounted(async () => {
  const infoSvc = await getAppInfoService()
  appVersion.value = await infoSvc.getVersion()
  await fetchUsage()
  document.addEventListener('visibilitychange', onVisibilityChange)
  windowHidden = document.hidden
  scheduleNext()
})

onUnmounted(() => {
  if (timer) clearTimeout(timer)
  timer = null
  document.removeEventListener('visibilitychange', onVisibilityChange)
})
</script>

<template>
  <footer class="status-bar">
    <!-- Page-specific slot — filled by active page via Teleport -->
    <div id="status-bar-page-slot" class="status-bar__page-slot"></div>

    <!-- Resource usage -->
    <span class="status-bar__item status-bar__resource">CPU: {{ cpuPct.toFixed(1) }}%</span>
    <span class="status-bar__divider">|</span>
    <span class="status-bar__item status-bar__resource">RAM: {{ ramMb.toFixed(1) }} MB</span>
    <span class="status-bar__divider">|</span>

    <!-- Icon-only buttons với CSS tooltip -->
    <div class="status-bar__icon-wrap" data-tip="Dọn RAM (buffer + idle TCP)">
      <button class="status-bar__icon-btn" :class="{ 'status-bar__icon-btn--spinning': cleaning }"
              :disabled="cleaning" @click="softCleanup" aria-label="Dọn RAM">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M3 6h18M8 6V4h8v2M19 6l-1 14H6L5 6"/>
          <path d="M10 11v6M14 11v6"/>
        </svg>
      </button>
    </div>
    <div class="status-bar__icon-wrap" data-tip="Reload UI (mất state hiện tại)">
      <button class="status-bar__icon-btn status-bar__icon-btn--warn" @click="hardReload" aria-label="Reload UI">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor"
             stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/>
          <path d="M3 3v5h5"/>
        </svg>
      </button>
    </div>
    <span class="status-bar__divider">|</span>

    <span class="status-bar__item">Hạ Vũ {{ appVersion }}</span>
  </footer>
</template>

<style scoped>
.status-bar {
  height: var(--statusbar-height);
  background: var(--surface-elevated);
  border-top: 1px solid var(--border-subtle);
  display: flex;
  align-items: center;
  padding: 0 var(--space-3);
  gap: var(--space-3);
  font-size: var(--font-size-xs);
  color: var(--text-muted);
  flex-shrink: 0;
}

.status-bar__page-slot {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  overflow: hidden;
}

.status-bar__item { display: flex; align-items: center; white-space: nowrap; }
.status-bar__resource { color: var(--text-secondary); }
.status-bar__divider { color: var(--border-default); }

/* Icon-only button + CSS tooltip */
.status-bar__icon-wrap { position: relative; display: flex; align-items: center; }
.status-bar__icon-wrap::after {
  content: attr(data-tip);
  position: absolute;
  bottom: calc(100% + 6px);
  left: 50%;
  transform: translateX(-50%);
  white-space: nowrap;
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  color: var(--text-primary);
  font-size: 11px;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  pointer-events: none;
  opacity: 0;
  transition: opacity 120ms;
  box-shadow: var(--shadow-md);
  z-index: var(--z-tooltip);
}

.status-bar__icon-wrap:hover::after { opacity: 1; }

.status-bar__icon-btn {
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-muted);
  width: 22px;
  height: 22px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 120ms, color 120ms, border-color 120ms;
  padding: 0;
}

.status-bar__icon-btn svg {
  fill: none;
  stroke: currentColor;
}

.status-bar__icon-btn:hover:not(:disabled) {
  background: var(--surface-hover-strong);
  border-color: var(--border-default);
  color: var(--text-primary);
}

.status-bar__icon-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.status-bar__icon-btn--warn:hover { border-color: var(--warning-solid); color: var(--warning-text); }
.status-bar__icon-btn--spinning { animation: spin 0.8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }
</style>
