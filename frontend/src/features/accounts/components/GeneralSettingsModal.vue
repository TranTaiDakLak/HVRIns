<script setup lang="ts">
// GeneralSettingsModal.vue — Modal cài đặt chung (đầy đủ như GeneralSettingsPage)
// Layout 2 cột: trái (Nguồn TK, Mạng IP) | phải (Đăng nhập, Môi trường, Captcha, Sau khi chạy)

import { ref, computed, watch } from 'vue'
import { X, AlertTriangle, FolderOpen } from 'lucide-vue-next'
import BaseModal from '@/components/feedback/BaseModal.vue'
import type { GeneralConfig, IpConfig } from '@/types/settings.types'
import {
  DEFAULT_GENERAL_CONFIG, DEFAULT_IP_CONFIG,
  getLoginMethodsByPlatform,
  CAPTCHA_PROVIDERS,
  IP_PROVIDERS,
} from '@/types/settings.types'
import { UA_POOLS } from '@/types/interaction.types'
import { getFileDialogService, getSettingsService } from '@/services/client'
import { useAppStore } from '@/stores/app.store'

const appStore = useAppStore()

const props = defineProps<{
  show: boolean
  config: GeneralConfig
  ipConfig: IpConfig
}>()

const emit = defineEmits<{
  close: []
  save: [config: GeneralConfig, ipConfig: IpConfig]
}>()

const form = ref<GeneralConfig>({ ...DEFAULT_GENERAL_CONFIG })
const ipForm = ref<IpConfig>({ ...DEFAULT_IP_CONFIG })

// Auto-save state
const autosaveStatus = ref('')
let autosaveTimer: number | undefined
let skipNextAutosave = false // true khi watch fire do load config (không phải user edit)

function triggerAutosave() {
  if (skipNextAutosave) {
    skipNextAutosave = false
    return
  }
  if (autosaveTimer) clearTimeout(autosaveTimer)
  autosaveStatus.value = '• Đang lưu...'
  autosaveTimer = window.setTimeout(() => {
    doAutosave()
  }, 500)
}

async function doAutosave() {
  try {
    emit('save', { ...form.value, captchaKeys: { ...form.value.captchaKeys } }, { ...ipForm.value })
    autosaveStatus.value = '✓ Đã lưu'
    setTimeout(() => { autosaveStatus.value = '' }, 1500)
  } catch {
    autosaveStatus.value = '✗ Lỗi lưu'
  }
}

function handleClose() {
  // Flush any pending autosave trước khi đóng
  if (autosaveTimer) {
    clearTimeout(autosaveTimer)
    autosaveTimer = undefined
    void doAutosave()
  }
  emit('close')
}

// Watch tất cả field để auto-save (deep để bắt nested captchaKeys).
watch([form, ipForm], triggerAutosave, { deep: true })

watch(() => props.show, (visible) => {
  if (visible) {
    skipNextAutosave = true
    form.value = { ...props.config, captchaKeys: { ...props.config.captchaKeys } }
    ipForm.value = { ...props.ipConfig }
    autosaveStatus.value = ''
  }
})

const loginMethods = computed(() => getLoginMethodsByPlatform(form.value.loginPlatform))

watch(() => form.value.loginPlatform, () => {
  const methods = getLoginMethodsByPlatform(form.value.loginPlatform)
  if (form.value.loginMethod >= methods.length) {
    form.value.loginMethod = 0
  }
})

function clampThreadRequest() {
  if (form.value.threadRequest < 1) form.value.threadRequest = 1
  if (form.value.threadRequest > 600) form.value.threadRequest = 600
}

async function browseFolderSource() {
  try {
    const fileSvc = await getFileDialogService()
    const path = await fileSvc.openFolder()
    if (!path) return
    form.value = { ...form.value, accountSourcePath: path }
    await fileSvc.setAccountSourceFolder(path)
  } catch { /* ignore */ }
}

async function browseFileSource() {
  try {
    const w = window as any
    const pickFn = w?.go?.app?.App?.OpenFileDialogPath
    if (typeof pickFn !== 'function') return
    const path = await pickFn()
    if (!path) return
    form.value = { ...form.value, accountSourcePath: path, accountSource: 'file' as any }
    // LoadAccountsFromFile — backend clear store + parse file + SaveSettings('file' + path).
    // Backend tự persist accountSource='file' vào general.json trước khi return.
    const loadFn = w?.go?.app?.App?.LoadAccountsFromFile
    if (typeof loadFn === 'function') {
      try {
        const result = await loadFn(path)
        const errs = (result?.errors || []).length
        const msg = errs > 0
          ? `Đã load ${result?.imported || 0} accounts (${errs} dòng lỗi)`
          : `Đã load ${result?.imported || 0} accounts — đóng modal + tick chọn ở bảng để verify`
        appStore.notify('success', msg)
      } catch (e) {
        appStore.notify('error', `Lỗi load file: ${e}`)
      }
    }
    // Force-save lần nữa để chắc chắn 'file' persist (bypass race với autosave debounce).
    // Skip autosave kế tiếp — tránh ghi đè bằng giá trị cũ trong form.
    skipNextAutosave = true
    try {
      const svc = await getSettingsService()
      const current = await svc.load()
      await svc.save({
        ...current,
        general: { ...(current.general || form.value), accountSource: 'file' as any, accountSourcePath: path },
      })
    } catch { /* ignore */ }
  } catch { /* ignore */ }
}

// handleSave removed — replaced by auto-save via watch + triggerAutosave.
// handleClose (defined earlier) flushes pending autosave before emitting close.
</script>

<template>
  <BaseModal :show="show" title="Cài đặt chung" size="lg" @close="emit('close')">
    <div class="sm-body">

      <!-- CỘT TRÁI -->
      <div class="sm-col">

        <!-- NỀN TẢNG ĐĂNG NHẬP -->
        <fieldset class="sm-fieldset">
          <legend>Nền tảng đăng nhập</legend>
          <div class="sm-field sm-field--row">
            <label class="sm-radio">
              <input type="radio" value="facebook" v-model="form.loginPlatform" />
              <span>Facebook</span>
            </label>
          </div>
          <div class="sm-field">
            <label>Dạng đăng nhập:</label>
            <select v-model.number="form.loginMethod" class="sm-select">
              <option v-for="method in loginMethods" :key="method.value" :value="method.value" :disabled="method.disabled">
                {{ method.label }}{{ method.disabled ? ' (KHÔNG HOẠT ĐỘNG)' : '' }}
              </option>
            </select>
          </div>
          <div class="sm-hint" v-if="loginMethods[form.loginMethod]?.description">
            {{ loginMethods[form.loginMethod].description }}
          </div>
        </fieldset>

        <!-- MÔI TRƯỜNG CHẠY -->
        <fieldset class="sm-fieldset">
          <legend>Môi trường chạy</legend>
          <div class="sm-field">
            <label>Số luồng Request:</label>
            <input type="number" v-model.number="form.threadRequest" min="1" max="600" class="sm-input sm-input--sm" @blur="clampThreadRequest" @change="clampThreadRequest" />
            <span class="sm-unit">luồng</span>
          </div>
          <div class="sm-field">
            <label>Nghỉ giữa các request:</label>
            <input type="number" v-model.number="form.delayRequest" min="0" step="100" class="sm-input sm-input--sm" />
            <span class="sm-unit">ms</span>
          </div>
          <div class="sm-field">
            <label>Delay luồng:</label>
            <input type="number" v-model.number="form.delayThread" min="0" step="100" class="sm-input sm-input--sm" />
            <span class="sm-unit">ms</span>
          </div>
        </fieldset>

        <!-- USER AGENT INFO -->
        <fieldset class="sm-fieldset">
          <legend>User Agent</legend>
          <div class="sm-hint">UA được đọc từ <code>Config/UserAgent/</code> — chỉnh sửa file trực tiếp bằng Notepad. Pool và modifier cài ở tab Thiết lập chạy.</div>
          <div class="sm-ua-files">
            <div v-for="pool in UA_POOLS" :key="pool.key" class="sm-ua-file-item">
              <span class="sm-ua-file-label">{{ pool.label }}:</span>
              <code class="sm-ua-file-code">Config/UserAgent/{{ pool.file }}</code>
            </div>
          </div>
        </fieldset>

      </div><!-- /sm-col left -->

      <!-- CỘT PHẢI -->
      <div class="sm-col">

        <!-- NGUỒN TÀI KHOẢN -->
        <fieldset class="sm-fieldset">
          <legend>Nguồn tài khoản</legend>
          <div class="sm-field sm-field--row">
            <label class="sm-radio">
              <input type="radio" v-model="form.accountSource" value="folder" />
              <span>Từ thư mục (đọc file .txt)</span>
            </label>
            <label class="sm-radio">
              <input type="radio" v-model="form.accountSource" value="file" />
              <span>Từ 1 file (chọn acc tick)</span>
            </label>
            <label class="sm-radio">
              <input type="radio" v-model="form.accountSource" value="api" />
              <span>Mua từ API (CloneHV)</span>
            </label>
          </div>
          <template v-if="form.accountSource === 'folder'">
            <div class="sm-field">
              <label>Thư mục nguồn:</label>
              <input type="text" :value="form.accountSourcePath" class="sm-input" style="flex:1" placeholder="Chọn thư mục..." readonly />
              <button type="button" class="sm-btn-browse" @click="browseFolderSource" title="Chọn thư mục">
                <FolderOpen :size="14" />
              </button>
            </div>
          </template>
          <template v-else-if="form.accountSource === 'file'">
            <div class="sm-field">
              <label>File nguồn:</label>
              <input type="text" :value="form.accountSourcePath" class="sm-input" style="flex:1" placeholder="Chọn 1 file .txt..." readonly />
              <button type="button" class="sm-btn-browse" @click="browseFileSource" title="Chọn file">
                <FolderOpen :size="14" />
              </button>
            </div>
            <div class="sm-hint">Load toàn bộ accounts từ file vào grid. Verify chỉ chạy các account bạn tick chọn.</div>
          </template>
          <template v-else>
            <div class="sm-hint">Nhập thông tin CloneHV để lấy tài khoản từ kho API.</div>
            <div class="sm-field">
              <label>Username:</label>
              <input type="text" v-model="form.cloneHvUsername" class="sm-input" style="flex:1" placeholder="username CloneHV..." />
            </div>
            <div class="sm-field">
              <label>Password:</label>
              <input type="password" v-model="form.cloneHvPassword" class="sm-input" style="flex:1" placeholder="password CloneHV..." />
            </div>
            <div class="sm-field">
              <label>Product ID:</label>
              <input type="text" v-model="form.cloneHvProductId" class="sm-input sm-input--sm" placeholder="VD: 18" />
            </div>
            <div class="sm-field">
              <label>Số lượng/batch:</label>
              <input type="number" v-model.number="form.cloneHvAmount" class="sm-input sm-input--sm" min="1" max="500" placeholder="10" />
            </div>
          </template>
        </fieldset>

        <!-- LOCALE FAKE + SIM NETWORK -->
        <fieldset class="sm-fieldset">
          <legend>Locale &amp; Sim</legend>
          <div class="sm-subsection-label">Locale fake</div>
          <div class="sm-field sm-field--row">
            <label class="sm-radio">
              <input type="radio" v-model="form.localeFake" value="random" />
              <span>Random</span>
            </label>
            <label class="sm-radio">
              <input type="radio" v-model="form.localeFake" value="match-ip" />
              <span>Match by IP</span>
            </label>
          </div>
          <label class="sm-checkbox">
            <input type="checkbox" v-model="form.deepFakeInApi" />
            <span class="sm-checkbox__content">
              <span>Deep Fake in API</span>
              <span class="sm-desc">Fake locale sâu hơn trong các API call</span>
            </span>
          </label>
          <div class="sm-subsection-divider">Sim Network</div>
          <div class="sm-field sm-field--row">
            <label class="sm-radio">
              <input type="radio" v-model="form.simNetworkMode" value="random" />
              <span>Random</span>
            </label>
            <label class="sm-radio">
              <input type="radio" v-model="form.simNetworkMode" value="match-ip" />
              <span>Match by IP</span>
            </label>
          </div>
          <div class="sm-field">
            <label>Loại mạng:</label>
            <select v-model="form.simNetworkType" class="sm-select">
              <option value="LTE">LTE</option>
              <option value="4G">4G</option>
              <option value="3G">3G</option>
              <option value="WIFI">WIFI</option>
            </select>
          </div>
        </fieldset>

        <!-- SAU KHI CHẠY -->
        <fieldset class="sm-fieldset">
          <legend>Sau khi chạy</legend>
          <label class="sm-checkbox">
            <input type="checkbox" v-model="form.saveRunColumn" />
            <span class="sm-checkbox__content">
              <span>Lưu cột lần chạy</span>
              <span class="sm-desc">Ghi IP chạy, hoạt động vào database</span>
            </span>
          </label>
          <label class="sm-checkbox">
            <input type="checkbox" v-model="form.backupDB" />
            <span class="sm-checkbox__content">
              <span>Sao lưu database</span>
              <span class="sm-desc">Tạo file backup sau mỗi lần chạy</span>
            </span>
          </label>
          <label class="sm-checkbox">
            <input type="checkbox" v-model="form.closeAfterDone" />
            <span>Đóng app khi xong</span>
          </label>
        </fieldset>


      </div><!-- /sm-col right -->

    </div>

    <template #footer>
      <div class="sm-footer">
        <span class="sm-footer__autosave">{{ autosaveStatus }}</span>
        <div class="sm-footer__spacer" />
        <button class="sm-btn sm-btn--danger" @click="handleClose"><X :size="14" /> Đóng</button>
      </div>
    </template>
  </BaseModal>
</template>

<style scoped>
.sm-body {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-4);
  align-items: start;
  max-height: calc(75vh - 120px);
  overflow-y: auto;
  overflow-x: hidden;
  padding-right: var(--space-1);
}

.sm-col { display: flex; flex-direction: column; gap: var(--space-3); }

.sm-fieldset {
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  padding: var(--space-3) var(--space-4) var(--space-4);
  display: flex; flex-direction: column; gap: var(--space-3);
}
.sm-fieldset legend {
  font-size: var(--font-size-sm); font-weight: 700; color: var(--brand-primary);
  padding: 0 var(--space-2);
}

.sm-field { display: flex; align-items: center; gap: var(--space-3); }
.sm-field > label:first-child { font-size: var(--font-size-sm); color: var(--text-secondary); white-space: nowrap; min-width: 140px; }
.sm-field--row { gap: var(--space-4); flex-wrap: wrap; }

.sm-unit { font-size: var(--font-size-sm); color: var(--text-muted); white-space: nowrap; }

.sm-hint {
  font-size: var(--font-size-xs); color: var(--text-muted);
  padding: var(--space-1) var(--space-2);
  background: var(--surface-sunken); border-radius: var(--radius-sm);
  border-left: 2px solid var(--brand-primary);
}
.sm-hint--warning {
  border-left-color: var(--warning-solid);
  color: var(--warning-text);
  background: var(--warning-bg);
  display: flex; align-items: center; gap: var(--space-1);
}

.sm-input {
  flex: 1; background: var(--surface-sunken); border: 1px solid var(--border-default);
  color: var(--text-primary); padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm); font-size: var(--font-size-sm); font-family: inherit; outline: none;
}
.sm-input:focus { border-color: var(--border-focus); }
.sm-input--sm { width: 80px; flex: none; text-align: right; }
.sm-input--mono { font-family: var(--font-mono); font-size: var(--font-size-xs); }

.sm-select {
  flex: 1; background: var(--surface-sunken); border: 1px solid var(--border-default);
  color: var(--text-primary); padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm); font-size: var(--font-size-sm); font-family: inherit; outline: none; cursor: pointer;
}
.sm-select:focus { border-color: var(--border-focus); }
.sm-select option:disabled { color: var(--text-disabled); font-style: italic; }

.sm-radio { display: flex; align-items: center; gap: var(--space-2); font-size: var(--font-size-sm); cursor: pointer; }
.sm-radio input { accent-color: var(--brand-primary); }

.sm-checkbox { display: flex; align-items: flex-start; gap: var(--space-2); font-size: var(--font-size-sm); cursor: pointer; }
.sm-checkbox input { accent-color: var(--brand-primary); margin-top: 2px; }
.sm-checkbox__content { display: flex; flex-direction: column; gap: 1px; }
.sm-desc { font-size: var(--font-size-xs); color: var(--text-muted); }

.sm-footer { display: flex; align-items: center; gap: var(--space-2); }
.sm-footer__spacer { flex: 1; }
.sm-footer__autosave { font-size: 12px; color: var(--text-muted); min-width: 100px; }

.sm-btn {
  padding: var(--space-2) var(--space-4); border-radius: var(--radius-md);
  font-size: var(--font-size-sm); font-weight: 600; border: 1px solid transparent; cursor: pointer;
  display: flex; align-items: center; gap: var(--space-1);
  transition: background var(--transition-fast);
}
.sm-btn--primary { background: var(--brand-primary); border-color: var(--brand-primary); color: white; }
.sm-btn--primary:hover { background: var(--brand-primary-hover); }
.sm-btn--danger { background: var(--danger-solid); border-color: var(--danger-solid); color: white; }
.sm-btn--danger:hover { opacity: 0.9; }
.sm-btn--ghost { background: transparent; color: var(--text-secondary); border-color: var(--border-default); }
.sm-btn--ghost:hover { background: var(--surface-hover); }
.sm-btn--xs { padding: 3px 10px; font-size: var(--font-size-xs); font-weight: 500; }

/* UA pools */
.sm-ua-tabs { display: flex; gap: 4px; flex-wrap: wrap; }
.sm-ua-tab {
  display: flex; align-items: center; gap: 5px;
  padding: 4px 10px; border-radius: var(--radius-sm);
  border: 1px solid var(--border-default);
  background: transparent; color: var(--text-secondary);
  font-size: var(--font-size-xs); cursor: pointer;
  transition: all var(--transition-fast);
}
.sm-ua-tab:hover { border-color: var(--brand-primary); color: var(--brand-primary); }
.sm-ua-tab--active { background: var(--brand-primary); border-color: var(--brand-primary); color: white; }
.sm-ua-count { font-size: 10px; opacity: 0.75; }

.sm-ua-file-row { display: flex; align-items: center; gap: var(--space-2); }
.sm-ua-file-path { flex: 1; font-size: var(--font-size-xs); color: var(--text-muted); font-family: var(--font-mono); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.sm-icon-btn { border: none; background: none; cursor: pointer; padding: 2px 5px; border-radius: var(--radius-sm); font-size: 11px; color: var(--text-muted); }
.sm-icon-btn--danger:hover { color: var(--danger-text); background: var(--danger-bg); }

.sm-textarea {
  width: 100%; box-sizing: border-box;
  background: var(--surface-sunken); border: 1px solid var(--border-default);
  color: var(--text-primary); padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm); font-size: 11px; font-family: var(--font-mono);
  resize: vertical; outline: none; line-height: 1.5;
}
.sm-textarea:focus { border-color: var(--border-focus); }

.sm-ua-footer { display: flex; align-items: center; justify-content: space-between; }
.sm-muted { font-size: var(--font-size-xs); color: var(--text-muted); }

.sm-subsection-divider {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  padding-top: var(--space-1);
  border-top: 1px solid var(--border-default);
  margin-top: var(--space-1);
}

.sm-subsection-label {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.sm-inline-options {
  display: flex;
  flex-direction: row;
  gap: var(--space-4);
  flex-wrap: wrap;
}

.sm-btn-browse {
  display: flex; align-items: center; justify-content: center;
  padding: 5px 8px;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  background: var(--surface-raised);
  color: var(--text-secondary);
  cursor: pointer;
  flex-shrink: 0;
  transition: background 0.15s, border-color 0.15s;
}
.sm-btn-browse:hover { background: var(--surface-hover); border-color: var(--brand-primary); color: var(--brand-primary); }
</style>
