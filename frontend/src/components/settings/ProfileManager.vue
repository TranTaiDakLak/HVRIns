<script setup lang="ts">
// ProfileManager.vue — Profile switcher: save/load/clone/delete/import/export
// Replaces ProfileExporter for full profile lifecycle management.

import { ref, computed } from 'vue'
import { X } from 'lucide-vue-next'
import type { SavedProfile } from '../../composables/useSettingsProfiles'

const props = defineProps<{
  profiles: SavedProfile[]
  activeProfileId: string | null
  label?: string
}>()

const emit = defineEmits<{
  (e: 'load', id: string): void
  (e: 'save', name: string): void
  (e: 'update', id: string): void
  (e: 'clone', id: string, name: string): void
  (e: 'delete', id: string): void
  (e: 'rename', id: string, name: string): void
  (e: 'import', json: string): void
  (e: 'export', id: string): void
  (e: 'reset'): void
}>()

const showPanel = ref(false)
const activeTab = ref<'profiles' | 'import'>('profiles')
const newName = ref('')
const importText = ref('')
const importError = ref('')
const renamingId = ref<string | null>(null)
const renameText = ref('')
const confirmDeleteId = ref<string | null>(null)

const activeProfile = computed(() =>
  props.profiles.find(p => p.id === props.activeProfileId) ?? null,
)

function handleSave() {
  const name = newName.value.trim()
  if (!name) return
  emit('save', name)
  newName.value = ''
}

function handleImport() {
  importError.value = ''
  if (!importText.value.trim()) return
  try {
    JSON.parse(importText.value) // validate JSON
    emit('import', importText.value)
    importText.value = ''
    activeTab.value = 'profiles'
  } catch {
    importError.value = 'JSON không hợp lệ. Kiểm tra lại.'
  }
}

function startRename(profile: SavedProfile) {
  renamingId.value = profile.id
  renameText.value = profile.name
}

function confirmRename() {
  if (!renamingId.value || !renameText.value.trim()) return
  emit('rename', renamingId.value, renameText.value.trim())
  renamingId.value = null
}

function handleDelete(id: string) {
  if (confirmDeleteId.value === id) {
    emit('delete', id)
    confirmDeleteId.value = null
  } else {
    confirmDeleteId.value = id
    setTimeout(() => { confirmDeleteId.value = null }, 3000)
  }
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString('vi-VN', {
      day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit',
    })
  } catch { return iso }
}
</script>

<template>
  <div class="pm-wrap">
    <!-- Trigger button showing active profile -->
    <button type="button" class="pm-trigger" @click="showPanel = !showPanel">
      <span class="pm-trigger__icon">&#x1F4CB;</span>
      <span class="pm-trigger__text">
        {{ activeProfile ? activeProfile.name : 'Profiles' }}
      </span>
      <span class="pm-trigger__count" v-if="profiles.length">({{ profiles.length }})</span>
      <span class="pm-trigger__caret">{{ showPanel ? '&#x25B2;' : '&#x25BC;' }}</span>
    </button>

    <!-- Panel -->
    <div v-if="showPanel" class="pm-panel">
      <!-- Tabs -->
      <div class="pm-header">
        <div class="pm-tabs">
          <button :class="['pm-tab', { active: activeTab === 'profiles' }]" @click="activeTab = 'profiles'">
            Profiles
          </button>
          <button :class="['pm-tab', { active: activeTab === 'import' }]" @click="activeTab = 'import'">
            Import
          </button>
        </div>
        <button class="pm-close" @click="showPanel = false">&#x2715;</button>
      </div>

      <!-- Profiles tab -->
      <div v-if="activeTab === 'profiles'" class="pm-body">
        <!-- Save new -->
        <div class="pm-save-row">
          <input
            v-model="newName"
            class="pm-input"
            placeholder="Tên profile mới..."
            @keydown.enter="handleSave"
          />
          <button class="pm-btn pm-btn--save" :disabled="!newName.trim()" @click="handleSave">
            Tạo
          </button>
        </div>
        <div class="pm-hint-row">Profile lưu vĩnh viễn — khác với Preset template (áp dụng tạm thời)</div>

        <!-- Existing profiles list -->
        <div v-if="profiles.length === 0" class="pm-empty">
          Chưa có profile nào. Nhập tên và nhấn "Lưu" để tạo.
        </div>

        <div v-else class="pm-list">
          <div
            v-for="p in profiles"
            :key="p.id"
            class="pm-item"
            :class="{ 'pm-item--active': p.id === activeProfileId }"
          >
            <!-- Rename mode -->
            <div v-if="renamingId === p.id" class="pm-rename-row">
              <input
                v-model="renameText"
                class="pm-input pm-input--rename"
                @keydown.enter="confirmRename"
                @keydown.escape="renamingId = null"
              />
              <button class="pm-action" title="Xác nhận" @click="confirmRename">&#x2714;</button>
              <button class="pm-action" title="Huỷ" @click="renamingId = null"><X :size="12" /></button>
            </div>

            <!-- Normal mode -->
            <template v-else>
              <div class="pm-item__info" @click="emit('load', p.id)">
                <span class="pm-item__name">
                  {{ p.name }}
                  <span v-if="p.id === activeProfileId" class="pm-active-badge">Active</span>
                </span>
                <span class="pm-item__date">{{ formatDate(p.updatedAt) }}</span>
              </div>
              <div class="pm-item__actions">
                <button
                  v-if="p.id === activeProfileId"
                  class="pm-action"
                  :title="`Ghi đè profile '${p.name}' với cấu hình hiện tại`"
                  @click="emit('update', p.id)"
                >&#x1F4BE;</button>
                <button class="pm-action" title="Nhân bản" @click="emit('clone', p.id, p.name + ' (copy)')">&#x1F4CB;</button>
                <button class="pm-action" title="Đổi tên" @click="startRename(p)">&#x270F;</button>
                <button class="pm-action" title="Export JSON" @click="emit('export', p.id)">&#x2B07;</button>
                <button
                  class="pm-action pm-action--delete"
                  :title="confirmDeleteId === p.id ? 'Nhấn lần nữa để xoá' : 'Xoá'"
                  @click="handleDelete(p.id)"
                >
                  {{ confirmDeleteId === p.id ? '&#x2757;' : '&#x1F5D1;' }}
                </button>
              </div>
            </template>
          </div>
        </div>

        <!-- Reset to defaults -->
        <div class="pm-divider" />
        <button class="pm-btn pm-btn--reset" @click="emit('reset'); showPanel = false">
          &#x21BA; Đặt lại về mặc định
        </button>
      </div>

      <!-- Import tab -->
      <div v-else class="pm-body">
        <p class="pm-hint">Dán JSON profile vào đây. Import sẽ điền vào form — nhấn Lưu để áp dụng.</p>
        <textarea
          v-model="importText"
          class="pm-textarea"
          rows="8"
          placeholder='{ "name": "My Profile", "data": { ... } }'
          spellcheck="false"
        />
        <div v-if="importError" class="pm-error">{{ importError }}</div>
        <div class="pm-actions-row">
          <button class="pm-btn pm-btn--save" :disabled="!importText.trim()" @click="handleImport">
            &#x2714; Import
          </button>
          <button class="pm-btn" @click="importText = ''; importError = ''">Xoá</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.pm-wrap { position: relative; }

.pm-trigger {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 12px;
  border-radius: 6px;
  border: 1px solid var(--border-default);
  background: transparent;
  color: var(--text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.15s;
}
.pm-trigger:hover {
  border-color: var(--brand-primary);
  color: var(--brand-primary);
}
.pm-trigger__icon { font-size: 13px; }
.pm-trigger__count { color: var(--text-muted); font-size: 11px; }
.pm-trigger__caret { font-size: 8px; color: var(--text-muted); }

.pm-panel {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  width: 400px;
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  border-radius: 10px;
  box-shadow: var(--shadow-lg);
  z-index: 200;
  overflow: hidden;
}

.pm-header {
  display: flex;
  align-items: center;
  padding: 10px 12px 0;
  gap: 8px;
}

.pm-tabs { display: flex; gap: 2px; flex: 1; }

.pm-tab {
  padding: 5px 12px;
  border-radius: 6px 6px 0 0;
  border: none;
  background: transparent;
  color: var(--text-muted);
  font-size: 13px;
  cursor: pointer;
}
.pm-tab.active {
  background: var(--brand-primary);
  color: white;
  font-weight: 600;
}

.pm-close {
  border: none;
  background: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 14px;
  padding: 4px;
  border-radius: 4px;
}
.pm-close:hover { color: var(--text-primary); }

.pm-body { padding: 12px; }

.pm-save-row {
  display: flex;
  gap: 8px;
  margin-bottom: 10px;
}

.pm-input {
  flex: 1;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  color: var(--text-primary);
  font-size: 12px;
  padding: 6px 10px;
  outline: none;
}
.pm-input:focus { border-color: var(--brand-primary); }
.pm-input--rename { font-size: 12px; padding: 3px 8px; }

.pm-btn {
  padding: 6px 14px;
  border-radius: 6px;
  border: 1px solid var(--border-default);
  background: transparent;
  color: var(--text-primary);
  font-size: 12px;
  cursor: pointer;
  white-space: nowrap;
}
.pm-btn:disabled { opacity: 0.4; cursor: not-allowed; }
.pm-btn--save {
  background: var(--brand-primary);
  border-color: var(--brand-primary);
  color: white;
  font-weight: 600;
}
.pm-btn--save:hover:not(:disabled) { background: var(--brand-primary-hover); border-color: var(--brand-primary-hover); }
.pm-btn--reset {
  width: 100%;
  font-size: 11px;
  color: var(--text-muted);
  border-color: var(--border-default);
}
.pm-btn--reset:hover { border-color: var(--danger-solid); color: var(--danger-text); }

.pm-hint-row {
  font-size: 10px;
  color: var(--text-muted);
  margin-bottom: 4px;
  margin-top: -4px;
  font-style: italic;
}

.pm-divider {
  height: 1px;
  background: var(--border-default);
  margin: 8px 0;
}

.pm-empty {
  text-align: center;
  color: var(--text-muted);
  font-size: 12px;
  padding: 16px 0;
}

.pm-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: 280px;
  overflow-y: auto;
}

.pm-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border-radius: 6px;
  border: 1px solid transparent;
  transition: all 0.15s;
}
.pm-item:hover { background: var(--surface-hover-subtle); }
.pm-item--active {
  background: var(--brand-primary-bg);
  border-color: var(--brand-primary-border);
}

.pm-item__info {
  flex: 1;
  cursor: pointer;
  min-width: 0;
}

.pm-item__name {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.pm-active-badge {
  font-size: 9px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: var(--brand-primary);
  color: white;
  padding: 1px 5px;
  border-radius: 4px;
  flex-shrink: 0;
}

.pm-item__date {
  display: block;
  font-size: 10px;
  color: var(--text-muted);
}

.pm-item__actions {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
}

.pm-action {
  border: none;
  background: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 12px;
  padding: 3px 5px;
  border-radius: 4px;
  transition: color 0.15s;
}
.pm-action:hover { color: var(--text-primary); background: var(--surface-hover); }
.pm-action--delete:hover { color: var(--danger-text); }

.pm-rename-row {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 1;
}

.pm-hint {
  font-size: 12px;
  color: var(--text-muted);
  margin: 0 0 8px;
  line-height: 1.5;
}

.pm-textarea {
  width: 100%;
  box-sizing: border-box;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  border-radius: 6px;
  color: var(--text-primary);
  font-family: monospace;
  font-size: 11px;
  padding: 8px;
  resize: vertical;
  outline: none;
  line-height: 1.5;
}
.pm-textarea:focus { border-color: var(--brand-primary); }

.pm-error {
  background: var(--danger-bg);
  border: 1px solid var(--danger-solid);
  border-radius: 6px;
  padding: 6px 10px;
  font-size: 12px;
  color: var(--danger-text);
  margin: 8px 0 0;
}

.pm-actions-row {
  display: flex;
  gap: 8px;
  margin-top: 10px;
  justify-content: flex-end;
}
</style>
