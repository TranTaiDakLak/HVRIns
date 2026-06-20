<script setup lang="ts">
// InlineValidation.vue — Inline validation indicator for form fields.
// Shows error/warning below a field based on schema registry constraints.
// Usage: <InlineValidation field="threadRequest" :value="form.threadRequest" />

import { computed } from 'vue'
import { validateFieldValue, getFieldMetaByFormKey } from '@/schema/settings-registry'

const props = defineProps<{
  /** Form key matching settings-registry.ts formKey */
  field: string
  /** Current field value */
  value: unknown
  /** Override error message (bypasses registry validation) */
  error?: string
}>()

const meta = computed(() => getFieldMetaByFormKey(props.field))

const validationError = computed(() => {
  if (props.error) return props.error
  return validateFieldValue(props.field, props.value)
})

const isWarning = computed(() => {
  // Warnings: value is near boundary but still valid
  if (!meta.value || meta.value.type !== 'int' || typeof props.value !== 'number') return false
  if (meta.value.max && props.value > meta.value.max * 0.9) return true
  return false
})
</script>

<template>
  <span v-if="validationError" class="iv" :class="{ 'iv--warn': isWarning, 'iv--error': !isWarning }">
    {{ validationError }}
  </span>
</template>

<style scoped>
.iv {
  display: block;
  font-size: 11px;
  padding: 2px 0;
  line-height: 1.4;
}
.iv--error { color: var(--danger-text, #ef5350); }
.iv--warn { color: var(--warning-text, #ffb74d); }
</style>
