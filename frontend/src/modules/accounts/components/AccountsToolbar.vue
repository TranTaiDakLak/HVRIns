<script setup lang="ts">
// AccountsToolbar.vue — Toolbar cho Accounts page
// Import, Run, Delete, Columns, Filter, Selection count

import { Play, Square, Loader, SlidersHorizontal, Settings, Plus, Trash2, FolderOpen, RefreshCw, FolderCog } from 'lucide-vue-next'

const props = defineProps<{
  selectedCount: number
  totalCount: number
  filterKeyword: string
  isRunning?: boolean
  isStopping?: boolean
  cloneHvMode?: boolean
  fileMode?: boolean
  resultFolderPath?: string
}>()

const emit = defineEmits<{
  import: []
  delete: []
  run: []
  stop: []
  'update:filterKeyword': [value: string]
  'toggle-columns': []
  settings: []
  'open-result-folder': []
  'open-config-folder': []
}>()

function reloadUI() {
  // Khi đang chạy: reload sẽ làm REG grid trống vài giây (backend vẫn chạy ngầm,
  // accounts vẫn được xử lý + lưu vào file result). Confirm để user biết, tránh
  // tưởng nhầm là đã stop.
  if (props.isRunning) {
    const ok = window.confirm(
      'App đang chạy.\n\n' +
      'Reload chỉ làm mới giao diện — backend KHÔNG dừng, accounts vẫn được xử lý ngầm.\n' +
      'Tuy nhiên REG grid sẽ trống vài giây cho đến khi worker xử lý acc tiếp theo.\n\n' +
      'Bạn vẫn muốn reload?'
    )
    if (!ok) return
  }
  location.reload()
}
</script>

<template>
  <div class="toolbar">
    <h2 class="toolbar__title">Accounts</h2>

    <!-- Nhóm 1: Chạy / Dừng -->
    <div class="toolbar__group">
      <button
        v-if="isStopping"
        class="toolbar__btn toolbar__btn--stopping"
        disabled
      >
        <Loader :size="14" class="toolbar__spin" /> Đang dừng...
      </button>
      <button
        v-else-if="!isRunning"
        class="toolbar__btn toolbar__btn--run"
        :disabled="!cloneHvMode && !fileMode && selectedCount === 0"
        @click="emit('run')"
      >
        <Play :size="14" />  Chạy
        <span v-if="cloneHvMode" class="toolbar__badge">API</span>
        <span v-else-if="fileMode" class="toolbar__badge">File</span>
      </button>
      <button
        v-else
        class="toolbar__btn toolbar__btn--stop"
        @click="emit('stop')"
      >
        <Square :size="14" /> Dừng
      </button>
    </div>

    <div class="toolbar__spacer" />

    <!-- Nút Config Folder: mở thư mục Config trong Explorer -->
    <button
      class="toolbar__btn toolbar__btn--folder"
      title="Mở thư mục Config"
      @click="emit('open-config-folder')"
    >
      <FolderCog :size="14" /> Config
    </button>

    <!-- Nút Result Folder: chọn nếu chưa có, mở Explorer nếu đã có -->
    <button
      class="toolbar__btn toolbar__btn--folder"
      :title="resultFolderPath ? ('Mở: ' + resultFolderPath) : 'Chọn thư mục Result'"
      @click="emit('open-result-folder')"
    >
      <FolderOpen :size="14" /> Result
    </button>

    <!-- Nhóm 2: Hiển thị -->
    <div class="toolbar__group">
      <button class="toolbar__btn toolbar__btn--columns" @click="emit('toggle-columns')">
        <SlidersHorizontal :size="14" /> Columns
      </button>
      <button class="toolbar__btn toolbar__btn--settings" @click="emit('settings')">
        <Settings :size="14" /> Cài đặt
      </button>
    </div>

    <div class="toolbar__separator" />

    <!-- Nhóm 3: Quản lý dữ liệu -->
    <div class="toolbar__group">
      <button class="toolbar__btn toolbar__btn--primary" @click="emit('import')">
        <Plus :size="14" /> Import
      </button>
      <button
        class="toolbar__btn toolbar__btn--danger-outline"
        :disabled="selectedCount === 0"
        @click="emit('delete')"
      >
        <Trash2 :size="14" /> Xóa ({{ selectedCount }})
      </button>
    </div>

    <div class="toolbar__separator" />

    <!-- Filter -->
    <input
      class="toolbar__filter"
      type="text"
      placeholder="Lọc theo UID, Email, Note..."
      :value="filterKeyword"
      @input="emit('update:filterKeyword', ($event.target as HTMLInputElement).value)"
    />

    <!-- Làm mới UI: fix WebView2 rendering glitch sau nhiều giờ chạy -->
    <button
      class="toolbar__btn toolbar__btn--refresh"
      title="Làm mới giao diện (fix nút bị mất sau khi chạy lâu)"
      @click="reloadUI"
    >
      <RefreshCw :size="13" />
    </button>
  </div>
</template>

<style scoped>
.toolbar {
  height: var(--toolbar-height);
  background: var(--surface-elevated);
  border-bottom: 1px solid var(--border-default);
  display: flex;
  align-items: center;
  padding: 0 var(--space-4);
  gap: var(--space-2);
  flex-shrink: 0;
}

.toolbar__title {
  font-size: var(--font-size-lg);
  font-weight: 700;
  color: var(--text-primary);
  margin-right: var(--space-4);
}

.toolbar__btn {
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  border: 1px solid var(--border-strong);
  background: var(--surface-elevated);
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: var(--space-2);
  transition: background var(--transition-fast);
  outline: none;
  cursor: pointer;
}

.toolbar__btn:hover:not(:disabled) {
  background: var(--surface-sunken);
}

.toolbar__btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.toolbar__group {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.toolbar__btn--primary {
  background: var(--toolbar-primary-bg, transparent);
  border-color: var(--toolbar-primary-border, var(--brand-primary));
  color: var(--toolbar-primary-color, var(--brand-primary));
  font-weight: 600;
}

.toolbar__btn--primary:hover:not(:disabled) {
  background: var(--toolbar-primary-hover, var(--surface-hover));
}

.toolbar__btn--run {
  background: linear-gradient(135deg, #833AB4, #E1306C, #FD1D1D);
  border-color: #E1306C;
  color: white;
  font-weight: 600;
  min-width: 80px;
}

.toolbar__btn--run:hover:not(:disabled) {
  background: linear-gradient(135deg, #6d2f9a, #c4235c, #e01010);
  opacity: 0.92;
}

.toolbar__btn--run:disabled {
  opacity: 0.35;
}

.toolbar__btn--stop {
  background: linear-gradient(135deg, #E1306C, #FD1D1D);
  border-color: #FD1D1D;
  color: white;
  font-weight: 700;
  min-width: 80px;
  box-shadow: 0 0 0 1px rgba(225, 48, 108, 0.4);
}

.toolbar__btn--stop:hover:not(:disabled) {
  background: linear-gradient(135deg, #c4235c, #e01010);
}

.toolbar__btn--stopping {
  background: linear-gradient(135deg, #405DE6, #833AB4);
  border-color: #833AB4;
  color: white;
  font-weight: 700;
  min-width: 120px;
  cursor: not-allowed;
  opacity: 0.85;
}

.toolbar__btn--danger-outline {
  border-color: var(--toolbar-danger-border, rgba(248, 113, 113, 0.4));
  color: var(--toolbar-danger-color, var(--danger-text));
  background: var(--toolbar-danger-bg, transparent);
}
.toolbar__btn--danger-outline:hover:not(:disabled) {
  background: var(--toolbar-danger-hover, var(--surface-hover));
  border-color: var(--toolbar-danger-color, var(--danger-text));
}

.toolbar__btn--folder {
  color: var(--toolbar-folder-color, #4ade80);
  border-color: var(--toolbar-folder-border, var(--border-strong));
  background: var(--toolbar-folder-bg, transparent);
}
.toolbar__btn--folder:hover:not(:disabled) {
  background: var(--toolbar-folder-hover, var(--surface-hover));
  border-color: var(--toolbar-folder-color, #4ade80);
}

.toolbar__btn--columns {
  color: var(--toolbar-columns-color, #a78bfa);
  border-color: var(--toolbar-columns-border, var(--border-strong));
  background: var(--toolbar-neutral-bg, transparent);
}
.toolbar__btn--columns:hover:not(:disabled) {
  background: var(--toolbar-neutral-hover, var(--surface-hover));
  border-color: var(--toolbar-columns-color, #a78bfa);
}

.toolbar__btn--settings {
  color: var(--toolbar-settings-color, #fbbf24);
  border-color: var(--toolbar-settings-border, var(--border-strong));
  background: var(--toolbar-neutral-bg, transparent);
}
.toolbar__btn--settings:hover:not(:disabled) {
  background: var(--toolbar-neutral-hover, var(--surface-hover));
  border-color: var(--toolbar-settings-color, #fbbf24);
}

.toolbar__separator {
  width: 1px;
  height: 24px;
  background: var(--border-strong);
  margin: 0 var(--space-1);
}

.toolbar__spacer { flex: 1; min-width: 0; }

.toolbar__btn--refresh {
  padding: 5px 7px;
  color: var(--text-muted);
  border-color: transparent;
  background: transparent;
  flex-shrink: 0;
}
.toolbar__btn--refresh:hover:not(:disabled) {
  background: var(--surface-hover);
  color: var(--text-primary);
}

.toolbar__filter {
  background: var(--surface-sunken);
  border: 1px solid var(--border-strong);
  color: var(--text-primary);
  padding: 5px 10px;
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  width: 250px;
  outline: none;
}

.toolbar__filter:focus {
  border-color: var(--border-focus);
}

.toolbar__filter::placeholder {
  color: var(--text-muted);
}

.toolbar__count {
  font-size: var(--font-size-sm);
  color: var(--text-muted);
  white-space: nowrap;
}

.toolbar__spin {
  animation: spin 1s linear infinite;
}
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

.toolbar__badge {
  font-size: 10px;
  font-weight: 700;
  padding: 1px 5px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.25);
  color: white;
  letter-spacing: 0.05em;
}
</style>
