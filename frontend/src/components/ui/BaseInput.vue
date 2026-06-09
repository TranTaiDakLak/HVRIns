<!-- BaseInput.vue — Ô nhập liệu cơ bản
     Hỗ trợ v-model, nhiều type, toggle mật khẩu, trạng thái lỗi -->

<script setup lang="ts">
import { ref, computed } from 'vue'

// --- Props ---
const props = withDefaults(defineProps<{
  modelValue?: string | number
  type?: 'text' | 'number' | 'password' | 'search'
  placeholder?: string
  disabled?: boolean
  error?: string
  size?: 'sm' | 'md'
}>(), {
  modelValue: '',
  type: 'text',
  placeholder: '',
  disabled: false,
  error: '',
  size: 'md',
})

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
}>()

// Trạng thái hiện/ẩn mật khẩu
const passwordVisible = ref(false)

// Kiểu input thực tế (đổi password <-> text khi toggle)
const inputType = computed(() => {
  if (props.type === 'password') {
    return passwordVisible.value ? 'text' : 'password'
  }
  return props.type
})

// Xử lý input event
function onInput(e: Event) {
  const target = e.target as HTMLInputElement
  const value = props.type === 'number' ? Number(target.value) : target.value
  emit('update:modelValue', value)
}
</script>

<template>
  <div
    class="base-input"
    :class="[
      `base-input--${size}`,
      { 'base-input--error': !!error, 'base-input--disabled': disabled },
    ]"
  >
    <div class="base-input__wrapper">
      <!-- Icon tìm kiếm cho type=search -->
      <svg
        v-if="type === 'search'"
        class="base-input__icon"
        viewBox="0 0 16 16"
        fill="none"
        stroke="currentColor"
        stroke-width="1.5"
      >
        <circle cx="7" cy="7" r="4.5" />
        <path d="M10.5 10.5 14 14" stroke-linecap="round" />
      </svg>

      <input
        class="base-input__field"
        :class="{ 'base-input__field--has-icon': type === 'search' }"
        :type="inputType"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        @input="onInput"
      />

      <!-- Nút toggle hiện/ẩn mật khẩu -->
      <button
        v-if="type === 'password'"
        type="button"
        class="base-input__toggle"
        tabindex="-1"
        @click="passwordVisible = !passwordVisible"
      >
        <!-- Icon mắt mở -->
        <svg v-if="!passwordVisible" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M1 8s2.5-5 7-5 7 5 7 5-2.5 5-7 5-7-5-7-5Z" />
          <circle cx="8" cy="8" r="2" />
        </svg>
        <!-- Icon mắt đóng -->
        <svg v-else viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M1 8s2.5-5 7-5 7 5 7 5-2.5 5-7 5-7-5-7-5Z" />
          <path d="M2 14 14 2" stroke-linecap="round" />
        </svg>
      </button>
    </div>

    <!-- Thông báo lỗi bên dưới -->
    <p v-if="error" class="base-input__error">{{ error }}</p>
  </div>
</template>

<style scoped>
/* === Container === */
.base-input {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

/* === Wrapper bao quanh input === */
.base-input__wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

/* === Input field === */
.base-input__field {
  width: 100%;
  background: var(--surface-elevated);
  color: var(--text-primary);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  font-family: var(--font-family);
  outline: none;
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
}

.base-input__field::placeholder {
  color: var(--text-muted);
}

.base-input__field:focus {
  border-color: var(--border-focus);
  box-shadow: 0 0 0 2px var(--brand-primary-bg);
}

.base-input__field:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* === Size: sm === */
.base-input--sm .base-input__field {
  height: 28px;
  padding: 0 var(--space-2);
  font-size: var(--font-size-sm);
}

/* === Size: md === */
.base-input--md .base-input__field {
  height: 34px;
  padding: 0 var(--space-3);
  font-size: var(--font-size-base);
}

/* === Có icon tìm kiếm — thêm padding trái === */
.base-input__field--has-icon {
  padding-left: var(--space-8) !important;
}

/* === Icon tìm kiếm === */
.base-input__icon {
  position: absolute;
  left: var(--space-3);
  width: 14px;
  height: 14px;
  color: var(--text-muted);
  pointer-events: none;
}

/* === Nút toggle mật khẩu === */
.base-input__toggle {
  position: absolute;
  right: var(--space-2);
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  background: none;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  border-radius: var(--radius-sm);
  transition: color var(--transition-fast);
}

.base-input__toggle:hover {
  color: var(--text-primary);
}

.base-input__toggle svg {
  width: 14px;
  height: 14px;
}

/* === Trạng thái lỗi — viền đỏ === */
.base-input--error .base-input__field {
  border-color: var(--danger-solid);
}

.base-input--error .base-input__field:focus {
  box-shadow: 0 0 0 2px var(--danger-bg);
}

/* === Dòng thông báo lỗi === */
.base-input__error {
  margin: 0;
  font-size: var(--font-size-xs);
  color: var(--danger-text);
}
</style>
