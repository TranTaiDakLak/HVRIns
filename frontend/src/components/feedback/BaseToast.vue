<!-- BaseToast.vue — Hiển thị danh sách thông báo toast
     Đọc notifications từ useAppStore
     Tự động biến mất theo duration trong store
     Vị trí: góc trên bên phải -->

<script setup lang="ts">
import { useAppStore } from '../../stores/app.store'

const appStore = useAppStore()

// Map loại notification sang icon path (SVG)
const iconPaths: Record<string, string> = {
  success: 'M8 1a7 7 0 1 1 0 14A7 7 0 0 1 8 1Zm2.8 4.6L7.2 9.2 5.4 7.4',
  error: 'M8 1a7 7 0 1 1 0 14A7 7 0 0 1 8 1ZM5.5 5.5l5 5M10.5 5.5l-5 5',
  warning: 'M8 1a7 7 0 1 1 0 14A7 7 0 0 1 8 1ZM8 5v3M8 10.5v.5',
  info: 'M8 1a7 7 0 1 1 0 14A7 7 0 0 1 8 1ZM8 5.5v.5M8 8v3',
}
</script>

<template>
  <Teleport to="body">
    <div class="base-toast__container">
      <TransitionGroup name="toast">
        <div
          v-for="notif in appStore.notifications"
          :key="notif.id"
          class="base-toast"
          :class="`base-toast--${notif.type}`"
        >
          <!-- Icon theo loại thông báo -->
          <svg
            class="base-toast__icon"
            viewBox="0 0 16 16"
            fill="none"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path :d="iconPaths[notif.type]" />
          </svg>

          <!-- Nội dung thông báo -->
          <span class="base-toast__message">{{ notif.message }}</span>

          <!-- Nút đóng -->
          <button
            class="base-toast__close"
            @click="appStore.removeNotification(notif.id)"
            aria-label="Đóng"
          >
            <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
              <path d="M4 4 12 12M12 4 4 12" />
            </svg>
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<style scoped>
/* === Container — góc dưới bên phải === */
.base-toast__container {
  position: fixed;
  bottom: var(--space-4);
  right: var(--space-4);
  display: flex;
  flex-direction: column-reverse;
  gap: var(--space-2);
  z-index: var(--z-toast);
  pointer-events: none;
  max-width: 380px;
}

/* === Mỗi toast === */
.base-toast {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-md);
  pointer-events: auto;
  min-width: 280px;
}

/* === Icon === */
.base-toast__icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  margin-top: 1px;
}

/* === Nội dung === */
.base-toast__message {
  flex: 1;
  font-family: var(--font-family);
  font-size: var(--font-size-base);
  color: var(--text-primary);
  line-height: 1.4;
}

/* === Nút đóng === */
.base-toast__close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  padding: 0;
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
  transition: color var(--transition-fast);
}

.base-toast__close:hover {
  color: var(--text-primary);
}

.base-toast__close svg {
  width: 12px;
  height: 12px;
}

/* === Màu viền trái theo loại === */
.base-toast--success {
  border-left: 3px solid var(--success-solid);
}

.base-toast--success .base-toast__icon {
  color: var(--success-text);
}

.base-toast--error {
  border-left: 3px solid var(--danger-solid);
}

.base-toast--error .base-toast__icon {
  color: var(--danger-text);
}

.base-toast--warning {
  border-left: 3px solid var(--warning-solid);
}

.base-toast--warning .base-toast__icon {
  color: var(--warning-text);
}

.base-toast--info {
  border-left: 3px solid var(--info-solid);
}

.base-toast--info .base-toast__icon {
  color: var(--info-text);
}

/* === Transition hiệu ứng vào/ra === */
.toast-enter-active {
  transition: all var(--transition-normal);
}

.toast-leave-active {
  transition: all var(--transition-fast);
}

.toast-enter-from {
  opacity: 0;
  transform: translateX(100%);
}

.toast-leave-to {
  opacity: 0;
  transform: translateX(40px);
}

.toast-move {
  transition: transform var(--transition-normal);
}
</style>
