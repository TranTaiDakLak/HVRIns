<script setup lang="ts">
// AccountsImportDialog.vue — Import accounts: thư mục nguồn (persistent) + paste/file (one-time)

import { ref, computed, onMounted } from 'vue'
import BaseModal from '../../../components/feedback/BaseModal.vue'

const props = defineProps<{
  show: boolean
  loading?: boolean
}>()

const emit = defineEmits<{
  close: []
  import: [data: string]
  folderImported: [count: number]
}>()

// === FOLDER SOURCE ===
const sourcePath = ref('')
const folderStatus = ref<{ type: 'success' | 'error' | ''; msg: string }>({ type: '', msg: '' })
const folderLoading = ref(false)

onMounted(async () => {
  try {
    const { getBridgeMode } = await import('../../../bridge/client')
    if (getBridgeMode() === 'wails') {
      const { GetAccountSourceFolder, ValidatePath } = await import('../../../../wailsjs/go/main/App')
      const path = await GetAccountSourceFolder()
      sourcePath.value = path
      if (path) {
        const errMsg = await ValidatePath(path)
        if (errMsg) {
          folderStatus.value = { type: 'error', msg: '⚠ Thư mục không còn tồn tại: ' + path }
        }
      }
    }
  } catch { /* ignore */ }
})

async function handleChooseFolder() {
  folderLoading.value = true
  folderStatus.value = { type: '', msg: '' }
  try {
    const { getBridgeMode } = await import('../../../bridge/client')
    if (getBridgeMode() === 'wails') {
      const { OpenFolderDialog, SetAccountSourceFolder } = await import('../../../../wailsjs/go/main/App')
      const path = await OpenFolderDialog()
      if (!path) { folderLoading.value = false; return }
      sourcePath.value = path
      const result = await SetAccountSourceFolder(path)
      folderStatus.value = {
        type: 'success',
        msg: `Đã thêm ${result.imported} account từ ${path}`,
      }
      emit('folderImported', result.imported)
    }
  } catch (e: unknown) {
    folderStatus.value = { type: 'error', msg: String(e) }
  } finally {
    folderLoading.value = false
  }
}

async function handleRefreshFolder() {
  folderLoading.value = true
  folderStatus.value = { type: '', msg: '' }
  try {
    const { getBridgeMode } = await import('../../../bridge/client')
    if (getBridgeMode() === 'wails') {
      const { RefreshAccountSource } = await import('../../../../wailsjs/go/main/App')
      const result = await RefreshAccountSource()
      folderStatus.value = {
        type: result.imported > 0 ? 'success' : 'error',
        msg: result.imported > 0
          ? `Đã thêm ${result.imported} account mới`
          : (result.errors?.[0] ?? 'Không có account mới'),
      }
      if (result.imported > 0) emit('folderImported', result.imported)
    }
  } catch (e: unknown) {
    folderStatus.value = { type: 'error', msg: String(e) }
  } finally {
    folderLoading.value = false
  }
}

// === TỪ 1 FILE (persistent — LoadAccountsFromFile) ===
// Tương ứng với mode "Từ 1 file" ở Settings: pick file → load toàn bộ vào grid,
// backend nhớ path để verify xong xóa dòng khỏi file gốc + auto-reload khi restart.
const fileSourcePath = ref('')
const fileStatus = ref<{ type: 'success' | 'error' | ''; msg: string }>({ type: '', msg: '' })
const fileLoading = ref(false)

async function handleChooseFileSource() {
  fileLoading.value = true
  fileStatus.value = { type: '', msg: '' }
  try {
    const { getBridgeMode } = await import('../../../bridge/client')
    if (getBridgeMode() === 'wails') {
      const { OpenFileDialogPath, LoadAccountsFromFile } = await import('../../../../wailsjs/go/main/App')
      const path = await OpenFileDialogPath()
      if (!path) { fileLoading.value = false; return }
      fileSourcePath.value = path
      const result = await LoadAccountsFromFile(path)
      if (result.errors && result.errors.length > 0) {
        fileStatus.value = { type: 'error', msg: result.errors[0] }
      } else {
        fileStatus.value = {
          type: 'success',
          msg: `Đã load ${result.imported} account từ ${path}`,
        }
        emit('folderImported', result.imported)
      }
    }
  } catch (e: unknown) {
    fileStatus.value = { type: 'error', msg: String(e) }
  } finally {
    fileLoading.value = false
  }
}

// === PASTE / FILE (one-time ad-hoc, không lưu source) ===
const textData = ref('')
const loadingFile = ref(false)

async function handleChooseFile() {
  loadingFile.value = true
  try {
    const { getBridgeMode } = await import('../../../bridge/client')
    if (getBridgeMode() === 'wails') {
      const { OpenTextFileDialog } = await import('../../../../wailsjs/go/main/App')
      const content = await OpenTextFileDialog()
      if (content) textData.value = content
    }
  } catch { /* ignore */ }
  finally { loadingFile.value = false }
}

const lineCount = computed(() =>
  textData.value.trim() ? textData.value.split('\n').filter(l => l.trim()).length : 0
)

function handleImport() {
  if (lineCount.value === 0) return
  emit('import', textData.value)
  textData.value = ''
}

function handleClose() {
  textData.value = ''
  folderStatus.value = { type: '', msg: '' }
  emit('close')
}
</script>

<template>
  <BaseModal :show="show" title="Import Accounts" size="md" @close="handleClose">
    <div class="import-dialog">

      <!-- ===== SECTION 1: THƯ MỤC NGUỒN ===== -->
      <div class="import-section">
        <div class="import-section__title">Thư mục nguồn tài khoản</div>
        <p class="import-section__desc">
          Chọn thư mục chứa file .txt tài khoản. App sẽ tự động đọc khi khởi động và thêm account mới khi nhấn <strong>Làm mới</strong>.
        </p>

        <div class="folder-row">
          <div class="folder-path" :class="{ 'folder-path--set': sourcePath }">
            {{ sourcePath || 'Chưa chọn thư mục...' }}
          </div>
          <button class="import-btn import-btn--folder" @click="handleChooseFolder" :disabled="folderLoading">
            {{ folderLoading ? '...' : '📁 Chọn' }}
          </button>
          <button
            class="import-btn import-btn--refresh"
            @click="handleRefreshFolder"
            :disabled="folderLoading || !sourcePath"
            title="Quét lại thư mục, thêm account mới"
          >&#x27F3; Làm mới</button>
        </div>

        <div v-if="folderStatus.msg" class="folder-status" :class="'folder-status--' + folderStatus.type">
          {{ folderStatus.msg }}
        </div>
      </div>

      <!-- Divider -->
      <div class="import-divider"><span>hoặc từ 1 file (verify chỉ acc tick)</span></div>

      <!-- ===== SECTION 2: TỪ 1 FILE (persistent, tương đương Settings "Từ 1 file") ===== -->
      <div class="import-section">
        <div class="import-section__title">Từ 1 file (chọn acc tick)</div>
        <p class="import-section__desc">
          Pick 1 file .txt → load toàn bộ vào grid → tick chọn acc cần verify. Sau verify xong, các dòng đã verify sẽ tự xóa khỏi file gốc. App nhớ path → khởi động lại auto-load.
        </p>

        <div class="folder-row">
          <div class="folder-path" :class="{ 'folder-path--set': fileSourcePath }">
            {{ fileSourcePath || 'Chưa chọn file...' }}
          </div>
          <button class="import-btn import-btn--folder" @click="handleChooseFileSource" :disabled="fileLoading">
            {{ fileLoading ? '...' : '📄 Chọn file' }}
          </button>
        </div>

        <div v-if="fileStatus.msg" class="folder-status" :class="'folder-status--' + fileStatus.type">
          {{ fileStatus.msg }}
        </div>
      </div>

      <!-- Divider -->
      <div class="import-divider"><span>hoặc thêm một lần (paste/ad-hoc)</span></div>

      <!-- ===== SECTION 3: PASTE / FILE (one-time) ===== -->
      <div class="import-section">
        <div class="import-dialog__detect-info">
          <span class="detect-tag">UID (field đầu)</span>
          <span class="detect-tag">Password (sau UID)</span>
          <span class="detect-tag">Cookie (c_user=)</span>
          <span class="detect-tag">Token (EAA...)</span>
          <span class="detect-tag">2FA (32 ký tự)</span>
          <span class="detect-tag">Email (@)</span>
          <span class="detect-tag">Phone (8-15 số)</span>
        </div>

        <div class="import-dialog__file-row">
          <button class="import-btn import-btn--file" @click="handleChooseFile" :disabled="loading || loadingFile">
            {{ loadingFile ? 'Đang đọc...' : '📂 Chọn file .txt' }}
          </button>
          <span v-if="lineCount > 0" class="import-dialog__file-hint">{{ lineCount }} dòng</span>
          <span v-else class="import-dialog__file-hint import-dialog__info--muted">hoặc dán dữ liệu bên dưới</span>
        </div>

        <textarea
          v-model="textData"
          class="import-dialog__textarea"
          rows="8"
          placeholder="Dán dữ liệu tài khoản phân cách bằng | (mỗi dòng 1 account)"
          :disabled="loading"
        />

        <div class="import-dialog__info">
          <span v-if="lineCount > 0">{{ lineCount }} dòng sẽ được import</span>
          <span v-else class="import-dialog__info--muted">Chưa có dữ liệu</span>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="import-dialog__footer">
        <button
          class="import-btn import-btn--primary"
          :disabled="lineCount === 0 || loading"
          @click="handleImport"
        >
          {{ loading ? 'Đang import...' : `Import ${lineCount} accounts` }}
        </button>
        <button class="import-btn import-btn--ghost" @click="handleClose" :disabled="loading">Hủy</button>
      </div>
    </template>
  </BaseModal>
</template>

<style scoped>
.import-dialog {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

/* Section */
.import-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.import-section__title {
  font-size: var(--font-size-sm);
  font-weight: 700;
  color: var(--text-primary);
}
.import-section__desc {
  font-size: var(--font-size-xs);
  color: var(--text-muted);
  line-height: 1.5;
}

/* Folder row */
.folder-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.folder-path {
  flex: 1;
  background: var(--surface-sunken);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  padding: var(--space-2) var(--space-3);
  font-size: var(--font-size-xs);
  color: var(--text-muted);
  font-family: var(--font-mono);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.folder-path--set {
  color: var(--text-primary);
  border-color: var(--brand-primary-border);
}

/* Folder status */
.folder-status {
  font-size: var(--font-size-xs);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
}
.folder-status--success {
  background: var(--success-bg, rgba(34,197,94,0.1));
  color: var(--success-text, #22c55e);
}
.folder-status--error {
  background: var(--danger-bg, rgba(239,68,68,0.1));
  color: var(--danger-text, #ef4444);
}

/* Divider */
.import-divider {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  color: var(--text-disabled);
  font-size: var(--font-size-xs);
}
.import-divider::before,
.import-divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--border-subtle, var(--border-default));
}

/* Detect tags */
.import-dialog__detect-info {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.detect-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-size: var(--font-size-xs);
  background: var(--brand-primary-bg);
  color: var(--brand-primary);
  border: 1px solid var(--brand-primary-border);
}

/* File row */
.import-dialog__file-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.import-dialog__file-hint {
  font-size: var(--font-size-sm);
  color: var(--brand-primary);
  font-weight: 600;
}
.import-dialog__info--muted {
  color: var(--text-muted);
  font-weight: 400;
}

/* Textarea */
.import-dialog__textarea {
  width: 100%;
  background: var(--surface-sunken);
  border: 1px solid var(--border-default);
  color: var(--text-primary);
  padding: var(--space-3);
  border-radius: var(--radius-md);
  font-family: var(--font-mono);
  font-size: var(--font-size-sm);
  resize: vertical;
  outline: none;
  line-height: 1.6;
}
.import-dialog__textarea:focus { border-color: var(--border-focus); }
.import-dialog__textarea::placeholder { color: var(--text-disabled); }

/* Info */
.import-dialog__info {
  font-size: var(--font-size-sm);
  color: var(--brand-primary);
  font-weight: 600;
}

/* Footer */
.import-dialog__footer {
  display: flex;
  gap: var(--space-2);
  justify-content: flex-end;
}

/* Buttons */
.import-btn {
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  font-weight: 600;
  border: none;
  cursor: pointer;
  white-space: nowrap;
}
.import-btn--primary {
  background: var(--brand-primary);
  color: white;
}
.import-btn--primary:hover:not(:disabled) { background: var(--brand-primary-hover); }
.import-btn--primary:disabled { opacity: 0.4; cursor: not-allowed; }

.import-btn--ghost {
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border-default);
}
.import-btn--ghost:hover:not(:disabled) { background: var(--surface-hover-subtle); }

.import-btn--file {
  background: var(--surface-elevated);
  color: var(--text-primary);
  border: 1px dashed var(--border-default);
  cursor: pointer;
}
.import-btn--file:hover:not(:disabled) {
  border-color: var(--brand-primary);
  color: var(--brand-primary);
}

.import-btn--folder {
  background: var(--brand-primary);
  color: white;
  padding: var(--space-2) var(--space-3);
  flex-shrink: 0;
}
.import-btn--folder:disabled { opacity: 0.5; cursor: not-allowed; }

.import-btn--refresh {
  background: var(--surface-elevated);
  color: var(--text-primary);
  border: 1px solid var(--border-default);
  padding: var(--space-2) var(--space-3);
  flex-shrink: 0;
}
.import-btn--refresh:hover:not(:disabled) { border-color: var(--brand-primary); color: var(--brand-primary); }
.import-btn--refresh:disabled { opacity: 0.4; cursor: not-allowed; }
</style>
