<!-- BaseToggle.vue — Switch bật/tắt (checkbox styled thành switch)
     Hỗ trợ v-model, label, disabled. Port từ ý tưởng BaseToggle (MDR pipeline chips).
     Dùng cho dải pipeline chips Reg/Ver/ReUseEmail (S02-D2-T001). -->

<script setup lang="ts">
const props = withDefaults(defineProps<{
  modelValue?: boolean
  label?: string
  disabled?: boolean
}>(), {
  modelValue: false,
  label: '',
  disabled: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

function toggle() {
  if (!props.disabled) emit('update:modelValue', !props.modelValue)
}
</script>

<template>
  <label
    class="base-toggle"
    :class="{ 'base-toggle--disabled': disabled }"
    @click.prevent="toggle"
  >
    <span
      class="base-toggle__track"
      :class="{ 'base-toggle__track--on': modelValue }"
      role="switch"
      :aria-checked="modelValue"
    >
      <span class="base-toggle__thumb" :class="{ 'base-toggle__thumb--on': modelValue }" />
    </span>
    <span v-if="label" class="base-toggle__label">{{ label }}</span>
  </label>
</template>

<style scoped>
.base-toggle {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
  user-select: none;
}

.base-toggle--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* === Track === */
.base-toggle__track {
  position: relative;
  display: inline-block;
  width: 32px;
  height: 18px;
  border-radius: 9px;
  background: var(--surface-sunken, #2a2a2a);
  border: 1px solid var(--border-strong, #444);
  transition: background var(--transition-fast), border-color var(--transition-fast);
  flex-shrink: 0;
}

.base-toggle__track--on {
  background: var(--brand-primary);
  border-color: var(--brand-primary);
}

/* === Thumb === */
.base-toggle__thumb {
  position: absolute;
  top: 1px;
  left: 1px;
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: #fff;
  transition: transform var(--transition-fast);
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
}

.base-toggle__thumb--on {
  transform: translateX(14px);
}

/* === Label === */
.base-toggle__label {
  font-family: var(--font-family);
  font-size: var(--font-size-base);
  color: var(--text-primary);
}
</style>
