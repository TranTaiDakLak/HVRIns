<script setup lang="ts">
// ProfileExporter.vue — Import/export profile dưới dạng JSON
// Export: serialize form data → clipboard/download
// Import: paste JSON → merge vào form hiện tại
// Dùng trong GeneralSettingsPage (cho GeneralConfig+IpConfig) và InteractionSetupPage (VerifyConfig).

import { ref } from 'vue'
import { ArrowUpDown, X, Copy, Check, Download } from 'lucide-vue-next'

type AnyConfig = Record<string, unknown>

const props = defineProps<{
  /** Data hiện tại cần export */
  data: AnyConfig
  /** Label hiển thị (e.g. "Cài đặt chung", "Thiết lập chạy") */
  label?: string
}>()

const emit = defineEmits<{
  (e: 'import', data: AnyConfig): void
}>()

const showPanel = ref(false)
const activeTab = ref<'export' | 'import'>('export')
const importText = ref('')
const importError = ref('')
const exportCopied = ref(false)

const exportJson = () => JSON.stringify(props.data, null, 2)

async function handleCopy() {
  try {
    await navigator.clipboard.writeText(exportJson())
    exportCopied.value = true
    setTimeout(() => { exportCopied.value = false }, 2000)
  } catch {
    // fallback: select textarea
  }
}

function handleDownload() {
  const blob = new Blob([exportJson()], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `ncs-${props.label ?? 'profile'}-${Date.now()}.json`
  a.click()
  URL.revokeObjectURL(url)
}

function handleImport() {
  importError.value = ''
  try {
    const parsed = JSON.parse(importText.value)
    if (typeof parsed !== 'object' || Array.isArray(parsed)) {
      importError.value = 'JSON không hợp lệ — phải là object.'
      return
    }
    emit('import', parsed as AnyConfig)
    showPanel.value = false
    importText.value = ''
  } catch {
    importError.value = 'Không parse được JSON. Kiểm tra lại nội dung.'
  }
}
</script>

<template>
  <div class="pe-wrap">
    <button type="button" class="pe-trigger" @click="showPanel = !showPanel" :title="`Export/Import ${label ?? ''}`">
      <ArrowUpDown :size="14" />
      <span class="pe-trigger__label">Profile</span>
    </button>

    <div v-if="showPanel" class="pe-panel">
      <div class="pe-header">
        <div class="pe-tabs">
          <button :class="['pe-tab', { active: activeTab === 'export' }]" @click="activeTab = 'export'">Export</button>
          <button :class="['pe-tab', { active: activeTab === 'import' }]" @click="activeTab = 'import'">Import</button>
        </div>
        <button class="pe-close" @click="showPanel = false"><X :size="14" /></button>
      </div>

      <!-- Export -->
      <div v-if="activeTab === 'export'" class="pe-body">
        <p class="pe-hint">Sao chép JSON cấu hình hiện tại để lưu hoặc chia sẻ.</p>
        <textarea class="pe-textarea" readonly :value="exportJson()" rows="8" />
        <div class="pe-actions">
          <button class="pe-btn pe-btn--primary" @click="handleCopy">
            <Check v-if="exportCopied" :size="13" /> <Copy v-else :size="13" />
            {{ exportCopied ? 'Đã sao chép' : 'Sao chép' }}
          </button>
          <button class="pe-btn" @click="handleDownload"><Download :size="13" /> Tải file .json</button>
        </div>
      </div>

      <!-- Import -->
      <div v-else class="pe-body">
        <p class="pe-hint">
          Dán JSON cấu hình vào đây. Các field trùng sẽ được ghi đè;
          field không có trong JSON sẽ giữ nguyên.
        </p>
        <textarea
          class="pe-textarea"
          v-model="importText"
          placeholder='Paste JSON ở đây...'
          rows="8"
          spellcheck="false"
        />
        <div v-if="importError" class="pe-error">{{ importError }}</div>
        <div class="pe-actions">
          <button class="pe-btn pe-btn--primary" :disabled="!importText.trim()" @click="handleImport">
            <Check :size="13" /> Áp dụng
          </button>
          <button class="pe-btn" @click="importText = ''; importError = ''">Xóa</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.pe-wrap {
  position: relative;
}

.pe-trigger {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 10px;
  border-radius: 6px;
  border: 1px solid var(--border-default, #333);
  background: transparent;
  color: var(--text-secondary, #bbb);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}

.pe-trigger:hover {
  border-color: var(--accent, #4fc3f7);
  color: var(--accent, #4fc3f7);
}

.pe-trigger__label {
  font-size: 12px;
}

.pe-panel {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  width: 380px;
  background: var(--bg-secondary, #1e1e1e);
  border: 1px solid var(--border-color, #333);
  border-radius: 10px;
  box-shadow: 0 8px 24px rgba(0,0,0,0.5);
  z-index: 200;
  overflow: hidden;
}

.pe-header {
  display: flex;
  align-items: center;
  padding: 10px 12px 0;
  gap: 8px;
}

.pe-tabs {
  display: flex;
  gap: 2px;
  flex: 1;
}

.pe-tab {
  padding: 5px 12px;
  border-radius: 6px 6px 0 0;
  border: none;
  background: transparent;
  color: var(--text-muted, #888);
  font-size: 13px;
  cursor: pointer;
}

.pe-tab.active {
  background: var(--accent, #4fc3f7);
  color: #000;
  font-weight: 600;
}

.pe-close {
  border: none;
  background: none;
  color: var(--text-muted, #888);
  cursor: pointer;
  font-size: 14px;
  padding: 4px;
  border-radius: 4px;
}
.pe-close:hover { color: var(--text-primary, #e0e0e0); }

.pe-body {
  padding: 12px;
}

.pe-hint {
  font-size: 12px;
  color: var(--text-muted, #888);
  margin: 0 0 8px;
  line-height: 1.5;
}

.pe-textarea {
  width: 100%;
  box-sizing: border-box;
  background: var(--bg-primary, #121212);
  border: 1px solid var(--border-color, #333);
  border-radius: 6px;
  color: var(--text-primary, #e0e0e0);
  font-family: monospace;
  font-size: 11px;
  padding: 8px;
  resize: vertical;
  outline: none;
  line-height: 1.5;
}

.pe-textarea:focus { border-color: var(--accent, #4fc3f7); }

.pe-error {
  background: rgba(239,83,80,0.1);
  border: 1px solid rgba(239,83,80,0.4);
  border-radius: 6px;
  padding: 6px 10px;
  font-size: 12px;
  color: #ef5350;
  margin: 8px 0 0;
}

.pe-actions {
  display: flex;
  gap: 8px;
  margin-top: 10px;
  justify-content: flex-end;
}

.pe-btn {
  padding: 6px 14px;
  border-radius: 6px;
  border: 1px solid var(--border-color, #444);
  background: transparent;
  color: var(--text-primary, #e0e0e0);
  font-size: 12px;
  cursor: pointer;
}
.pe-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.pe-btn--primary {
  background: var(--accent, #4fc3f7);
  border-color: var(--accent, #4fc3f7);
  color: #000;
  font-weight: 600;
}
</style>
