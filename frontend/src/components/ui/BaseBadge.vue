<!-- BaseBadge.vue — Nhãn trạng thái nhỏ dạng pill
     5 variant: success, danger, warning, info, default
     2 size: sm, md -->

<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'success' | 'danger' | 'warning' | 'info' | 'default'
  size?: 'sm' | 'md'
}>(), {
  variant: 'default',
  size: 'md',
})
</script>

<template>
  <span
    class="base-badge"
    :class="[`base-badge--${variant}`, `base-badge--${size}`]"
  >
    <slot />
  </span>
</template>

<style scoped>
/* === Base — hình pill bo tròn === */
.base-badge {
  display: inline-flex;
  align-items: center;
  font-family: var(--font-family);
  font-weight: 500;
  border-radius: var(--radius-full);
  white-space: nowrap;
  line-height: 1;
}

/* === Size === */
.base-badge--sm {
  height: 18px;
  padding: 0 var(--space-2);
  font-size: var(--font-size-xs);
}

.base-badge--md {
  height: 22px;
  padding: 0 var(--space-3);
  font-size: var(--font-size-sm);
}

/* === Variant: Default — nền xám nhạt === */
.base-badge--default {
  background: var(--surface-elevated);
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
}

/* === Variant: Success — nền xanh nhạt === */
.base-badge--success {
  background: var(--success-bg);
  color: var(--success-text);
}

/* === Variant: Danger — nền đỏ nhạt === */
.base-badge--danger {
  background: var(--danger-bg);
  color: var(--danger-text);
}

/* === Variant: Warning — nền vàng nhạt === */
.base-badge--warning {
  background: var(--warning-bg);
  color: var(--warning-text);
}

/* === Variant: Info — nền xanh dương nhạt === */
.base-badge--info {
  background: var(--info-bg);
  color: var(--info-text);
}
</style>
