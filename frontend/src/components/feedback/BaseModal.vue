<!-- BaseModal.vue — Modal dialog cơ bản
     Hỗ trợ ESC đóng, click backdrop đóng
     4 kích thước: sm, md, lg, xl -->

<script setup lang="ts">
import { watch, onUnmounted } from 'vue'

// --- Props ---
const props = withDefaults(defineProps<{
  show: boolean
  title?: string
  size?: 'sm' | 'md' | 'lg' | 'xl'
}>(), {
  title: '',
  size: 'md',
})

const emit = defineEmits<{
  'update:show': [value: boolean]
  close: []
}>()

// Đóng modal
function close() {
  emit('update:show', false)
  emit('close')
}

// Xử lý phím ESC
function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    close()
  }
}

// Lắng nghe / gỡ event keyboard khi show thay đổi
watch(() => props.show, (val) => {
  if (val) {
    document.addEventListener('keydown', onKeydown)
    // Khóa scroll body khi modal mở
    document.body.style.overflow = 'hidden'
  } else {
    document.removeEventListener('keydown', onKeydown)
    document.body.style.overflow = ''
  }
}, { immediate: true })

onUnmounted(() => {
  document.removeEventListener('keydown', onKeydown)
  document.body.style.overflow = ''
})

// Map kích thước sang max-width
const sizeMap: Record<string, string> = {
  sm: '480px',
  md: '680px',
  lg: '920px',
  xl: '1140px',
}
</script>

<template>
  <Teleport to="body">
    <Transition name="modal">
      <div v-if="show" class="base-modal__backdrop" @click.self="close">
        <div
          class="base-modal__dialog"
          :style="{ maxWidth: sizeMap[size] }"
          role="dialog"
          aria-modal="true"
        >
          <!-- Header: tiêu đề + nút đóng -->
          <div v-if="title || $slots.header" class="base-modal__header">
            <slot name="header">
              <h3 class="base-modal__title">{{ title }}</h3>
            </slot>
            <button class="base-modal__close" @click="close" aria-label="Đóng">
              <svg viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round">
                <path d="M4 4 12 12M12 4 4 12" />
              </svg>
            </button>
          </div>

          <!-- Body: nội dung chính -->
          <div class="base-modal__body">
            <slot />
          </div>

          <!-- Footer: slot tùy chọn cho các nút hành động -->
          <div v-if="$slots.footer" class="base-modal__footer">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
/* === Backdrop phủ toàn màn hình === */
.base-modal__backdrop {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--surface-overlay);
  z-index: var(--z-modal-backdrop);
  padding: var(--space-4);
}

/* === Hộp dialog === */
.base-modal__dialog {
  width: 100%;
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  z-index: var(--z-modal);
  display: flex;
  flex-direction: column;
  max-height: 90vh;
}

/* === Header === */
.base-modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--border-subtle);
}

.base-modal__title {
  margin: 0;
  font-family: var(--font-family);
  font-size: var(--font-size-lg);
  font-weight: 600;
  color: var(--text-primary);
}

/* === Nút đóng X === */
.base-modal__close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  background: none;
  border: none;
  color: var(--text-muted);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.base-modal__close:hover {
  color: var(--text-primary);
  background: var(--surface-sunken);
}

.base-modal__close svg {
  width: 14px;
  height: 14px;
}

/* === Body === */
.base-modal__body {
  padding: var(--space-5);
  overflow-y: auto;
  font-family: var(--font-family);
  font-size: var(--font-size-base);
  color: var(--text-primary);
}

/* === Footer === */
.base-modal__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-5);
  border-top: 1px solid var(--border-subtle);
}

/* === Transition hiệu ứng mở/đóng === */
.modal-enter-active,
.modal-leave-active {
  transition: opacity var(--transition-normal);
}

.modal-enter-active .base-modal__dialog,
.modal-leave-active .base-modal__dialog {
  transition: transform var(--transition-normal);
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .base-modal__dialog {
  transform: translateY(16px) scale(0.97);
}

.modal-leave-to .base-modal__dialog {
  transform: translateY(8px) scale(0.99);
}
</style>
