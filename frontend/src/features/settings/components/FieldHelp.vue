<script setup lang="ts">
// FieldHelp.vue — Inline help tooltip/hint cho settings fields
// Usage: <FieldHelp field="threadRequest" /> hoặc <FieldHelp hint="Custom hint text" />
// Hover icon → hiện tooltip. Click "?" → expand detail nếu có.

import { ref, computed } from 'vue'
import { getFieldHelp } from '@/features/settings/schema/field-help'

const props = defineProps<{
  /** Key trong FIELD_HELP registry (e.g. 'threadRequest') */
  field?: string
  /** Override hint trực tiếp (không cần registry) */
  hint?: string
  /** Override detail trực tiếp */
  detail?: string
  /** Link tài liệu ngoài */
  docUrl?: string
}>()

const help = computed(() => props.field ? getFieldHelp(props.field) : null)
const resolvedHint = computed(() => props.hint ?? help.value?.hint ?? '')
const resolvedDetail = computed(() => props.detail ?? help.value?.detail ?? '')
const resolvedDocUrl = computed(() => props.docUrl ?? help.value?.docUrl ?? '')

const showTooltip = ref(false)
const expanded = ref(false)
</script>

<template>
  <span v-if="resolvedHint" class="fh-wrap">
    <button
      type="button"
      class="fh-icon"
      :class="{ 'fh-icon--active': showTooltip }"
      @mouseenter="showTooltip = true"
      @mouseleave="showTooltip = false"
      @focus="showTooltip = true"
      @blur="showTooltip = false"
      @click.stop="expanded = !expanded"
      aria-label="Trợ giúp"
    >?</button>

    <!-- Tooltip bubble -->
    <span v-if="showTooltip" class="fh-tooltip" role="tooltip">
      {{ resolvedHint }}
      <span v-if="resolvedDetail || resolvedDocUrl" class="fh-tooltip__more"> (click để xem thêm)</span>
    </span>

    <!-- Expanded detail (inline, dưới field) -->
    <span v-if="expanded" class="fh-detail">
      <span v-if="resolvedDetail">{{ resolvedDetail }}</span>
      <a v-if="resolvedDocUrl" :href="resolvedDocUrl" target="_blank" rel="noopener" class="fh-doc-link">
        📖 Tài liệu
      </a>
    </span>
  </span>
</template>

<style scoped>
.fh-wrap {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 2px;
}

.fh-icon {
  width: 15px;
  height: 15px;
  border-radius: 50%;
  border: 1px solid var(--text-muted, #666);
  background: transparent;
  color: var(--text-muted, #666);
  font-size: 10px;
  font-weight: 700;
  line-height: 1;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  transition: border-color 0.15s, color 0.15s;
  flex-shrink: 0;
}

.fh-icon:hover,
.fh-icon--active {
  border-color: var(--brand-primary);
  color: var(--brand-primary);
}

.fh-tooltip {
  position: absolute;
  bottom: calc(100% + 6px);
  left: 0;
  background: var(--surface-elevated);
  border: 1px solid var(--border-strong);
  border-radius: 6px;
  padding: 6px 10px;
  font-size: 11px;
  color: var(--text-primary);
  white-space: normal;
  width: 260px;
  z-index: 300;
  pointer-events: none;
  box-shadow: var(--shadow-md);
  line-height: 1.5;
}

.fh-tooltip__more {
  display: block;
  color: var(--text-muted);
  font-size: 10px;
  margin-top: 2px;
}

.fh-detail {
  display: block;
  margin-top: 4px;
  font-size: 11px;
  color: var(--text-muted);
  line-height: 1.5;
  background: var(--brand-primary-bg);
  border-left: 2px solid var(--brand-primary);
  padding: 4px 8px;
  border-radius: 0 4px 4px 0;
}

.fh-doc-link {
  display: inline-block;
  margin-top: 4px;
  color: var(--brand-primary);
  text-decoration: none;
  font-size: 11px;
}
.fh-doc-link:hover { text-decoration: underline; }
</style>
