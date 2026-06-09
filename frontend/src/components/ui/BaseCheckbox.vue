<!-- BaseCheckbox.vue — Checkbox tùy chỉnh
     Hỗ trợ v-model, label text, disabled
     Màu xanh khi được chọn -->

<script setup lang="ts">
// --- Props ---
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

// Toggle giá trị khi click
function toggle() {
  if (!props.disabled) {
    emit('update:modelValue', !props.modelValue)
  }
}
</script>

<template>
  <label
    class="base-checkbox"
    :class="{ 'base-checkbox--disabled': disabled }"
    @click.prevent="toggle"
  >
    <!-- Ô checkbox tùy chỉnh -->
    <span
      class="base-checkbox__box"
      :class="{ 'base-checkbox__box--checked': modelValue }"
    >
      <!-- Icon dấu tick khi checked -->
      <svg
        v-if="modelValue"
        class="base-checkbox__icon"
        viewBox="0 0 12 12"
        fill="none"
        stroke="#fff"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      >
        <polyline points="2.5 6 5 8.5 9.5 3.5" />
      </svg>
    </span>

    <!-- Label text bên phải -->
    <span v-if="label" class="base-checkbox__label">{{ label }}</span>
  </label>
</template>

<style scoped>
/* === Container === */
.base-checkbox {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
  user-select: none;
}

.base-checkbox--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* === Ô checkbox === */
.base-checkbox__box {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--surface-elevated);
  transition: all var(--transition-fast);
  flex-shrink: 0;
}

.base-checkbox__box--checked {
  background: var(--brand-primary);
  border-color: var(--brand-primary);
}

/* === Icon tick === */
.base-checkbox__icon {
  width: 10px;
  height: 10px;
}

/* === Label === */
.base-checkbox__label {
  font-family: var(--font-family);
  font-size: var(--font-size-base);
  color: var(--text-primary);
}
</style>
