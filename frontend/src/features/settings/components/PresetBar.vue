<script setup lang="ts">
// PresetBar.vue — Thanh chọn preset nhanh cho InteractionSetupPage
// Hiển thị 5 preset buttons; click → apply, highlight active.

import { defineProps, defineEmits } from 'vue'
import type { VerifyPreset } from '@/types/interaction.types'

const props = defineProps<{
  presets: VerifyPreset[]
  activeId: string | null
}>()

const emit = defineEmits<{
  (e: 'apply', presetId: string): void
}>()
</script>

<template>
  <div class="preset-bar">
    <div class="preset-bar__label-group">
      <span class="preset-bar__label">Preset</span>
      <span class="preset-bar__sub">template · áp dụng không lưu vĩnh viễn</span>
    </div>
    <div class="preset-bar__items">
      <button
        v-for="p in presets"
        :key="p.id"
        type="button"
        class="preset-btn"
        :class="{ 'preset-btn--active': activeId === p.id }"
        :title="p.description + '\n\nPreset chỉ thay đổi form hiện tại — nhấn Lưu để lưu vĩnh viễn.'"
        @click="emit('apply', p.id)"
      >
        <span class="preset-btn__icon">{{ p.icon }}</span>
        <span class="preset-btn__name">{{ p.name }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.preset-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: var(--surface-elevated, #1a1a1a);
  border-bottom: 1px solid var(--border-default, #2a2a2a);
  flex-shrink: 0;
}

.preset-bar__label-group {
  display: flex;
  flex-direction: column;
  gap: 1px;
  flex-shrink: 0;
}

.preset-bar__label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted, #888);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  white-space: nowrap;
}

.preset-bar__sub {
  font-size: 9px;
  color: var(--text-disabled, #555);
  white-space: nowrap;
  font-style: italic;
}

.preset-bar__items {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.preset-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 14px;
  border: 1px solid var(--border-default, #2a2a2a);
  background: transparent;
  color: var(--text-secondary, #bbb);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}

.preset-btn:hover {
  border-color: var(--accent, #4fc3f7);
  color: var(--accent, #4fc3f7);
  background: rgba(79,195,247,0.06);
}

.preset-btn--active {
  border-color: var(--accent, #4fc3f7);
  background: rgba(79,195,247,0.12);
  color: var(--accent, #4fc3f7);
  font-weight: 600;
}

.preset-btn__icon {
  font-size: 13px;
}

.preset-btn__name {
  font-size: 12px;
}
</style>
