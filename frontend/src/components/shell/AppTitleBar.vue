<script setup lang="ts">
// AppTitleBar.vue — Custom frameless title bar
// Thay thế title bar Windows native, có drag area + window controls

import { ref, onMounted, onBeforeUnmount } from 'vue'
import ConfirmDialog from '@/components/feedback/ConfirmDialog.vue'

// Wails runtime window controls
function minimise() { (window as any).runtime?.WindowMinimise() }
function toggleMaximise() { (window as any).runtime?.WindowToggleMaximise() }

// Quit pipeline:
//   1. User nhấn X / Alt+F4 / taskbar close → OnBeforeClose backend block + emit "app:request-quit-confirm"
//   2. Listener mở ConfirmDialog với cảnh báo "đang chạy reg/verify" nếu có
//   3. User confirm → gọi App.RequestQuit() → backend set flag + runtime.Quit()
//   4. Wails close pipeline chạy lại → OnBeforeClose thấy flag=true → cho close
const showQuitDialog = ref(false)
const dialogMessage = ref('Bạn có chắc muốn thoát ứng dụng?')

function quitClick() {
  // X button click → request quit (đi qua OnBeforeClose để FE nhận event + show dialog).
  // Không gọi trực tiếp ConfirmDialog ở đây — để 1 luồng duy nhất qua backend → tránh
  // race khi 2 nguồn (X button vs Alt+F4) cùng trigger và 2 dialog đồng thời.
  ;(window as any).runtime?.Quit()
}

function onQuitConfirm() {
  // User confirm → gọi backend RequestQuit để set flag + close
  const w = window as any
  if (typeof w?.go?.app?.App?.RequestQuit === 'function') {
    w.go.app.App.RequestQuit()
  } else {
    // Fallback: gọi runtime.Quit() lần nữa (flag chưa set → sẽ block lại).
    // Trường hợp này chỉ xảy ra trong dev khi Wails bindings chưa regen.
    w.runtime?.Quit?.()
  }
}

let unsub: (() => void) | undefined
onMounted(() => {
  // Backend emit event khi user trigger close (X / Alt+F4 / taskbar close).
  const w = window as any
  if (typeof w?.runtime?.EventsOn === 'function') {
    unsub = w.runtime.EventsOn('app:request-quit-confirm', (data: { registerRunning?: boolean; verifyRunning?: boolean }) => {
      if (data?.registerRunning || data?.verifyRunning) {
        const tasks: string[] = []
        if (data.registerRunning) tasks.push('Đăng ký')
        if (data.verifyRunning) tasks.push('Verify')
        dialogMessage.value = `⚠️ ${tasks.join(' + ')} đang chạy. Thoát bây giờ sẽ dừng hết các luồng. Bạn có chắc muốn thoát?`
      } else {
        dialogMessage.value = 'Bạn có chắc muốn thoát ứng dụng?'
      }
      showQuitDialog.value = true
    })
  }
})

onBeforeUnmount(() => {
  if (unsub) try { unsub() } catch { /* ignore */ }
})

const isMaximised = ref(false)

// Track maximise state
async function checkMaximised() {
  try {
    isMaximised.value = await (window as any).runtime?.WindowIsMaximised()
  } catch { /* ignore */ }
}

// Double-click title bar to toggle maximise
function onDblClick() {
  toggleMaximise()
  setTimeout(checkMaximised, 100)
}

// Update state after button click
function onToggleMax() {
  toggleMaximise()
  setTimeout(checkMaximised, 100)
}
</script>

<template>
  <div class="titlebar" @dblclick="onDblClick">
    <!-- Drag region — wails uses --wails-draggable attribute -->
    <div class="titlebar__drag" style="--wails-draggable: drag">
      <svg class="titlebar__icon" width="14" height="14" viewBox="0 0 24 24" fill="none">
        <defs>
          <linearGradient id="ig-grad-title" x1="0%" y1="100%" x2="100%" y2="0%">
            <stop offset="0%" stop-color="#FCAF45"/>
            <stop offset="35%" stop-color="#E1306C"/>
            <stop offset="70%" stop-color="#833AB4"/>
            <stop offset="100%" stop-color="#405DE6"/>
          </linearGradient>
        </defs>
        <rect x="2" y="2" width="20" height="20" rx="6" ry="6" stroke="url(#ig-grad-title)" stroke-width="2" fill="none"/>
        <circle cx="12" cy="12" r="4.5" stroke="url(#ig-grad-title)" stroke-width="2" fill="none"/>
        <circle cx="17.5" cy="6.5" r="1.2" fill="url(#ig-grad-title)"/>
      </svg>
      <span class="titlebar__text">Hạ Vũ</span>
    </div>

    <!-- Window controls -->
    <div class="titlebar__controls">
      <button class="titlebar__btn" @click="minimise" title="Minimize">
        <svg width="10" height="10" viewBox="0 0 10 10"><line x1="0" y1="5" x2="10" y2="5" stroke="currentColor" stroke-width="1.2"/></svg>
      </button>
      <button class="titlebar__btn" @click="onToggleMax" title="Maximize">
        <svg v-if="!isMaximised" width="10" height="10" viewBox="0 0 10 10"><rect x="0.5" y="0.5" width="9" height="9" rx="1" fill="none" stroke="currentColor" stroke-width="1.2"/></svg>
        <svg v-else width="10" height="10" viewBox="0 0 10 10"><rect x="2" y="0" width="8" height="8" rx="1" fill="none" stroke="currentColor" stroke-width="1.1"/><rect x="0" y="2" width="8" height="8" rx="1" fill="var(--surface-elevated)" stroke="currentColor" stroke-width="1.1"/></svg>
      </button>
      <button class="titlebar__btn titlebar__btn--close" @click="quitClick" title="Close">
        <svg width="10" height="10" viewBox="0 0 10 10"><line x1="1" y1="1" x2="9" y2="9" stroke="currentColor" stroke-width="1.3"/><line x1="9" y1="1" x2="1" y2="9" stroke="currentColor" stroke-width="1.3"/></svg>
      </button>
    </div>
  </div>

  <!-- Confirm dialog: hiện khi backend emit "app:request-quit-confirm" (sau khi user nhấn X / Alt+F4) -->
  <ConfirmDialog
    v-model:show="showQuitDialog"
    title="Thoát ứng dụng"
    :message="dialogMessage"
    confirm-text="Thoát"
    cancel-text="Hủy"
    variant="primary"
    @confirm="onQuitConfirm"
  />
</template>

<style scoped>
.titlebar {
  height: 32px;
  background: var(--surface-elevated);
  border-bottom: 1px solid var(--border-default);
  display: flex;
  align-items: center;
  flex-shrink: 0;
  user-select: none;
  -webkit-app-region: drag;
}

.titlebar__drag {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 6px;
  padding-left: 12px;
  height: 100%;
}

.titlebar__icon {
  color: var(--brand-primary);
  flex-shrink: 0;
}

.titlebar__text {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-muted);
  letter-spacing: 0.02em;
}

.titlebar__controls {
  display: flex;
  align-items: stretch;
  height: 100%;
  -webkit-app-region: no-drag;
}

.titlebar__btn {
  width: 46px;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-secondary);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: background 0.1s, color 0.1s;
}

.titlebar__btn:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}

.titlebar__btn--close:hover {
  background: #dc2626;
  color: white;
}
</style>
