<!-- BaseButton.vue — Nút bấm cơ bản
     Hỗ trợ 4 variant: primary, secondary, danger, ghost
     Hỗ trợ 3 size: sm, md, lg
     Có thể hiện spinner khi loading -->

<script setup lang="ts">
// --- Props ---
withDefaults(defineProps<{
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost'
  size?: 'sm' | 'md' | 'lg'
  disabled?: boolean
  loading?: boolean
  block?: boolean
}>(), {
  variant: 'primary',
  size: 'md',
  disabled: false,
  loading: false,
  block: false,
})
</script>

<template>
  <button
    class="base-btn"
    :class="[
      `base-btn--${variant}`,
      `base-btn--${size}`,
      { 'base-btn--block': block, 'base-btn--loading': loading },
    ]"
    :disabled="disabled || loading"
  >
    <!-- Spinner khi đang loading -->
    <span v-if="loading" class="base-btn__spinner" />
    <span class="base-btn__content" :class="{ 'base-btn__content--hidden': loading }">
      <slot />
    </span>
  </button>
</template>

<style scoped>
/* === Base === */
.base-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  position: relative;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  font-family: var(--font-family);
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;
  user-select: none;
  outline: none;
}

.base-btn:focus-visible {
  box-shadow: 0 0 0 2px var(--surface-base), 0 0 0 4px var(--border-focus);
}

.base-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* === Size === */
.base-btn--sm {
  height: 28px;
  padding: 0 var(--space-3);
  font-size: var(--font-size-sm);
}

.base-btn--md {
  height: 34px;
  padding: 0 var(--space-4);
  font-size: var(--font-size-base);
}

.base-btn--lg {
  height: 40px;
  padding: 0 var(--space-6);
  font-size: var(--font-size-md);
}

/* === Variant: Primary — nền gradient Instagram, chữ trắng ===
   Dùng --brand-gradient; nếu biến không resolve thì rơi về literal gradient.
   Viền --brand-primary (#E1306C) đảm bảo nút luôn hiển thị (req 10.7). */
.base-btn--primary {
  background: var(--brand-gradient, linear-gradient(135deg, #833AB4, #E1306C, #FD1D1D));
  color: #fff;
  border-color: var(--brand-primary);
}

.base-btn--primary:hover:not(:disabled) {
  background: linear-gradient(135deg, #6d2f9a, #c4235c, #e01010);
  border-color: var(--brand-primary-hover);
}

.base-btn--primary:active:not(:disabled) {
  background: var(--brand-primary-active);
}

/* === Variant: Secondary — nền surface, viền mặc định === */
.base-btn--secondary {
  background: var(--surface-elevated);
  color: var(--text-primary);
  border-color: var(--border-default);
}

.base-btn--secondary:hover:not(:disabled) {
  background: var(--surface-sunken);
  border-color: var(--border-strong);
}

/* === Variant: Danger — trong suốt, viền + chữ đỏ === */
.base-btn--danger {
  background: transparent;
  color: var(--danger-text);
  border-color: var(--danger-solid);
}

.base-btn--danger:hover:not(:disabled) {
  background: var(--danger-bg);
}

/* === Variant: Ghost — trong suốt, chỉ chữ === */
.base-btn--ghost {
  background: transparent;
  color: var(--text-secondary);
  border-color: transparent;
}

.base-btn--ghost:hover:not(:disabled) {
  background: var(--brand-primary-bg);
  color: var(--text-primary);
}

/* === Block — chiều rộng 100% === */
.base-btn--block {
  width: 100%;
}

/* === Loading spinner === */
.base-btn--loading {
  pointer-events: none;
}

.base-btn__spinner {
  position: absolute;
  width: 16px;
  height: 16px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: var(--radius-full);
  animation: btn-spin 0.6s linear infinite;
}

.base-btn__content--hidden {
  visibility: hidden;
}

@keyframes btn-spin {
  to { transform: rotate(360deg); }
}
</style>
