<script setup lang="ts">
// AccountsDetailPanel.vue — Side panel chi tiết account
// Mở khi double-click row, quick actions

import type { Account } from '@/bridge/contracts'
import { useAppStore } from '@/stores/app.store'
import { usePreferencesStore } from '@/stores/preferences.store'
import { X, Copy, Key, Cookie, Hash } from 'lucide-vue-next'

defineProps<{
  account: Account | null
  show: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const appStore = useAppStore()
const prefs = usePreferencesStore()

// Copy text vào clipboard
async function copyToClipboard(text: string, label: string) {
  try {
    await navigator.clipboard.writeText(text)
    appStore.notify('success', `Đã copy ${label}`)
  } catch {
    appStore.notify('error', 'Không thể copy vào clipboard')
  }
}

// Hiển thị giá trị dựa theo masking setting
function displayValue(value: string): string {
  if (!value) return '-'
  if (prefs.dataMasking) return '••••••••'
  return value
}
</script>

<template>
  <aside v-if="show && account" class="detail-panel">
    <div class="detail-panel__header">
      <h3>Chi tiết Account</h3>
      <button class="detail-panel__close" @click="emit('close')"><X :size="16" /></button>
    </div>

    <div class="detail-panel__body">
      <div class="detail-field">
        <label>UID</label>
        <div class="detail-field__value detail-field__value--mono selectable">{{ account.uid }}</div>
      </div>

      <div class="detail-field">
        <label>Email</label>
        <div class="detail-field__value selectable">{{ account.email || '-' }}</div>
      </div>

      <div class="detail-field">
        <label>Status</label>
        <div class="detail-field__value">
          <span class="status-badge" :class="`status-badge--${account.status}`">{{ account.status }}</span>
        </div>
      </div>

      <div class="detail-field">
        <label>Password</label>
        <div class="detail-field__value detail-field__value--mono selectable">{{ displayValue(account.password) }}</div>
      </div>

      <div class="detail-field">
        <label>Cookie</label>
        <div class="detail-field__value detail-field__value--mono detail-field__value--wrap selectable">
          {{ displayValue(account.cookie) }}
        </div>
      </div>

      <div class="detail-field">
        <label>Token</label>
        <div class="detail-field__value detail-field__value--mono selectable">{{ displayValue(account.token) }}</div>
      </div>

      <div class="detail-field">
        <label>2FA</label>
        <div class="detail-field__value">{{ account.twofa || '-' }}</div>
      </div>

      <div class="detail-field">
        <label>Pass Mail</label>
        <div class="detail-field__value detail-field__value--mono selectable">{{ displayValue(account.passMail) }}</div>
      </div>

      <div class="detail-field">
        <label>Mail Recovery</label>
        <div class="detail-field__value selectable">{{ account.mailRecovery || '-' }}</div>
      </div>

      <div class="detail-field">
        <label>Source</label>
        <div class="detail-field__value">{{ account.sourceCode || '-' }}</div>
      </div>

      <div class="detail-field">
        <label>Import Time</label>
        <div class="detail-field__value">{{ account.importTime || '-' }}</div>
      </div>

      <div class="detail-field">
        <label>Note</label>
        <div class="detail-field__value selectable">{{ account.note || '-' }}</div>
      </div>
    </div>

    <!-- Quick actions -->
    <div class="detail-panel__actions">
      <button class="action-btn" @click="copyToClipboard(account.uid, 'UID')"><Copy :size="13" /> Copy UID</button>
      <button class="action-btn" @click="copyToClipboard(account.password, 'Password')"><Key :size="13" /> Copy Pass</button>
      <button class="action-btn" @click="copyToClipboard(account.cookie, 'Cookie')"><Cookie :size="13" /> Copy Cookie</button>
      <button class="action-btn" @click="copyToClipboard(account.token, 'Token')"><Hash :size="13" /> Copy Token</button>
    </div>
  </aside>
</template>

<style scoped>
.detail-panel {
  width: var(--detail-panel-width);
  background: var(--surface-elevated);
  border-left: 1px solid var(--border-default);
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  overflow: hidden;
}

.detail-panel__header {
  display: flex;
  align-items: center;
  padding: var(--space-4);
  border-bottom: 1px solid var(--border-default);
}

.detail-panel__header h3 {
  font-size: var(--font-size-md);
  font-weight: 700;
  flex: 1;
}

.detail-panel__close {
  color: var(--text-muted);
  font-size: 16px;
  padding: var(--space-1);
  border-radius: var(--radius-sm);
}

.detail-panel__close:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}

.detail-panel__body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-4);
}

.detail-field {
  margin-bottom: var(--space-3);
}

.detail-field label {
  display: block;
  font-size: var(--font-size-xs);
  text-transform: uppercase;
  color: var(--text-muted);
  margin-bottom: 2px;
  letter-spacing: 0.3px;
}

.detail-field__value {
  font-size: var(--font-size-base);
  color: var(--text-primary);
  background: var(--surface-sunken);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-subtle);
  word-break: break-all;
}

.detail-field__value--mono {
  font-family: var(--font-mono);
  font-size: var(--font-size-sm);
}

.detail-field__value--wrap {
  max-height: 60px;
  overflow-y: auto;
}

.detail-panel__actions {
  padding: var(--space-4);
  border-top: 1px solid var(--border-default);
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.action-btn {
  padding: var(--space-1) var(--space-3);
  border-radius: var(--radius-md);
  font-size: var(--font-size-xs);
  border: 1px solid var(--border-strong);
  background: var(--surface-elevated);
  color: var(--text-primary);
  transition: background var(--transition-fast);
}

.action-btn:hover {
  background: var(--surface-hover);
}

/* Status badges */
.status-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-size: var(--font-size-xs);
  font-weight: 500;
}

.status-badge--live { background: var(--success-bg); color: var(--success-text); }
.status-badge--die { background: rgba(220, 38, 38, 0.22); color: #ef4444; }
.status-badge--checkpoint { background: var(--warning-bg); color: var(--warning-text); }
.status-badge--new { background: var(--info-bg); color: var(--info-text); }
.status-badge--unknown { background: var(--surface-hover-strong); color: var(--text-muted); }
</style>
