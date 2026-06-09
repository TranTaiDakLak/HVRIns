<!-- FormField.vue — Bao bọc cho input trong form
     Label ở trên, slot cho input ở giữa, error/hint ở dưới -->

<script setup lang="ts">
withDefaults(defineProps<{
  label?: string
  required?: boolean
  error?: string
  hint?: string
}>(), {
  label: '',
  required: false,
  error: '',
  hint: '',
})
</script>

<template>
  <div class="form-field" :class="{ 'form-field--error': !!error }">
    <!-- Label phía trên -->
    <label v-if="label" class="form-field__label">
      {{ label }}
      <span v-if="required" class="form-field__required">*</span>
    </label>

    <!-- Slot cho input component -->
    <div class="form-field__control">
      <slot />
    </div>

    <!-- Dòng lỗi hoặc gợi ý bên dưới -->
    <p v-if="error" class="form-field__error">{{ error }}</p>
    <p v-else-if="hint" class="form-field__hint">{{ hint }}</p>
  </div>
</template>

<style scoped>
/* === Container === */
.form-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

/* === Label === */
.form-field__label {
  font-family: var(--font-family);
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--text-secondary);
}

/* === Dấu * bắt buộc === */
.form-field__required {
  color: var(--danger-text);
  margin-left: 2px;
}

/* === Vùng chứa input === */
.form-field__control {
  display: flex;
  flex-direction: column;
}

/* === Dòng lỗi === */
.form-field__error {
  margin: 0;
  font-family: var(--font-family);
  font-size: var(--font-size-xs);
  color: var(--danger-text);
}

/* === Dòng gợi ý === */
.form-field__hint {
  margin: 0;
  font-family: var(--font-family);
  font-size: var(--font-size-xs);
  color: var(--text-muted);
}
</style>
