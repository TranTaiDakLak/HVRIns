<script setup lang="ts">
// GeneralSettingsPage.vue — Cài đặt chung
// Layout 1 cột: global settings only (threads, login, captcha, hành vi)
// Proxy/IP config → ProxySettings. Nguồn TK config → Thiết lập chạy.

import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { X, AlertTriangle, Save, LogIn, FileText } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app.store'
import { getSettingsService, getFileDialogService, getCloneHVService } from '@/bridge/client'
import type { GeneralConfig, AccountSource } from '@/types/settings.types'
import { ROUTE_NAMES } from '@/constants/routes'
import {
  DEFAULT_GENERAL_CONFIG,
  getLoginMethodsByPlatform,
  CAPTCHA_PROVIDERS,
  IP_PROVIDERS,
} from '@/types/settings.types'
import { useBackendProfiles } from '@/composables/useBackendProfiles'
import { useAutoSave } from '@/composables/useAutoSave'
import FieldHelp from '@/components/settings/FieldHelp.vue'
import ProfileManager from '@/components/settings/ProfileManager.vue'
import InlineValidation from '@/components/settings/InlineValidation.vue'

const router = useRouter()
const appStore = useAppStore()
const form = ref<GeneralConfig>({ ...DEFAULT_GENERAL_CONFIG, captchaKeys: { ...DEFAULT_GENERAL_CONFIG.captchaKeys } })

// ─── Account source form ──────────────────────────────────────────────────────
const accountForm = ref({
  accountSource: 'folder' as AccountSource,
  accountSourcePath: '',
  cloneHvUsername: '',
  cloneHvPassword: '',
  cloneHvProductId: '',
  cloneHvAmount: 10,
})
const cloneHvStockInfo = ref<{ name: string; amount: string; price: number } | null>(null)
const cloneHvStockError = ref('')
const loading = ref(false)

// Profiles — backed by Go/Wails profile API
const {
  profiles, activeProfileId, saveProfile, loadProfile,
  cloneProfile, deleteProfile, renameProfile,
} = useBackendProfiles()

async function reloadForm() {
  const settingsSvc = await getSettingsService()
  const data = await settingsSvc.load()
  if (data.general) {
    form.value = { ...DEFAULT_GENERAL_CONFIG, ...data.general, captchaKeys: { ...DEFAULT_GENERAL_CONFIG.captchaKeys, ...data.general.captchaKeys } }
    accountForm.value = {
      accountSource: data.general.accountSource ?? 'folder',
      accountSourcePath: data.general.accountSourcePath ?? '',
      cloneHvUsername: data.general.cloneHvUsername ?? '',
      cloneHvPassword: data.general.cloneHvPassword ?? '',
      cloneHvProductId: data.general.cloneHvProductId ?? '',
      cloneHvAmount: data.general.cloneHvAmount ?? 10,
    }
  }
  // UA pool config (uaPools/uaPoolKey/uaPoolFiles + useRawUa) đã chuyển hẳn sang
  // tab "Thiết lập chạy" → section "User Agent". GeneralSettingsPage không load nữa
  // để tránh 2 nguồn ghi đè nhau.
}

async function handleProfileLoad(id: string) {
  await loadProfile(id)
  try { await reloadForm() } catch { /* ignore */ }
}

async function handleProfileSave(name: string) {
  const id = await saveProfile(name)
  if (!id.startsWith('Lỗi')) {
    await handleSave()
    appStore.notify('success', `Đã tạo profile "${name}"`)
  } else {
    appStore.notify('error', id)
  }
}

function handleProfileExport(_id: string) {
  try {
    const json = JSON.stringify({ general: form.value }, null, 2)
    navigator.clipboard.writeText(json).then(() => appStore.notify('success', 'Đã sao chép cài đặt'))
  } catch { appStore.notify('error', 'Không thể sao chép') }
}

function handleProfileImport(json: string) {
  try {
    const data = JSON.parse(json)
    if (data.general) form.value = { ...DEFAULT_GENERAL_CONFIG, ...data.general, captchaKeys: { ...DEFAULT_GENERAL_CONFIG.captchaKeys, ...data.general.captchaKeys } }
    appStore.notify('success', 'Đã import — nhấn Lưu để áp dụng')
  } catch {
    appStore.notify('error', 'Import thất bại — JSON không hợp lệ')
  }
}

onMounted(async () => {
  loading.value = true
  try { await reloadForm() } catch { appStore.notify('error', 'Không tải được cài đặt') }
  finally { loading.value = false }
})

const loginMethods = computed(() => getLoginMethodsByPlatform(form.value.loginPlatform))
watch(() => form.value.loginPlatform, () => {
  const methods = getLoginMethodsByPlatform(form.value.loginPlatform)
  if (form.value.loginMethod >= methods.length) form.value.loginMethod = 0
})

async function handleBrowseAccountFolder() {
  try {
    const svc = await getFileDialogService()
    const path = await svc.openFolder()
    if (path) accountForm.value.accountSourcePath = path
  } catch { /* ignore */ }
}

async function handleBrowseAccountFile() {
  try {
    // OpenFileDialogPath — Go binding, mở file picker + trả path (.txt)
    const w = window as any
    const fn = w?.go?.main?.App?.OpenFileDialogPath
    if (typeof fn !== 'function') {
      appStore.notify('error', 'File picker chưa được hỗ trợ bởi backend')
      return
    }
    const path = await fn()
    if (path) {
      // Set BOTH source + path đồng thời để autosave thấy đúng giá trị
      accountForm.value.accountSource = 'file'
      accountForm.value.accountSourcePath = path
      // Auto trigger load accounts từ file vào store + emit event → AccountsPage refresh grid.
      const loadFn = w?.go?.main?.App?.LoadAccountsFromFile
      if (typeof loadFn === 'function') {
        try {
          const result = await loadFn(path)
          const errs = (result?.errors || []).length
          const msg = errs > 0
            ? `Đã load ${result?.imported || 0} accounts (${errs} dòng lỗi, đã bỏ qua)`
            : `Đã load ${result?.imported || 0} accounts — mở tab Accounts để tick chọn`
          appStore.notify('success', msg)
        } catch (err) {
          appStore.notify('error', `Lỗi load file: ${err}`)
        }
      }
      // Force-save explicit để chắc 'file' được persist (race-proof với autosave debounce).
      try {
        const svc = await getSettingsService()
        const current = await svc.load()
        await svc.save({
          ...current,
          general: { ...(current.general || form.value), accountSource: 'file' as any, accountSourcePath: path },
        })
      } catch { /* ignore */ }
    }
  } catch { /* ignore */ }
}

async function handleCheckCloneHVStock() {
  cloneHvStockInfo.value = null
  cloneHvStockError.value = ''
  try {
    const svc = await getCloneHVService()
    const result = await svc.checkStock(
      accountForm.value.cloneHvUsername,
      accountForm.value.cloneHvPassword,
      accountForm.value.cloneHvProductId,
    )
    if (result.error) cloneHvStockError.value = result.error
    else cloneHvStockInfo.value = { name: result.name, amount: result.amount, price: result.price }
  } catch (e) {
    cloneHvStockError.value = 'Lỗi kiểm tra: ' + String(e)
  }
}

function clampThreadRequest() {
  if (form.value.threadRequest < 1) form.value.threadRequest = 1
  if (form.value.threadRequest > 600) form.value.threadRequest = 600
}

function resetForms() {
  form.value = { ...DEFAULT_GENERAL_CONFIG, captchaKeys: { ...DEFAULT_GENERAL_CONFIG.captchaKeys } }
}

// Combine mọi form state vào 1 computed để useAutoSave watch 1 lần.
// UA pool fields đã chuyển sang "Thiết lập chạy" — không watch nữa.
const autoSavePayload = computed(() => ({
  form: form.value,
  account: accountForm.value,
}))

const { status: saveStatus } = useAutoSave(autoSavePayload, async () => {
  // Delegate về handleSave() để reuse logic validate + dual-service save
  await handleSave()
})

async function handleSave() {
  clampThreadRequest()
  try {
    const svc = await getSettingsService()
    const current = await svc.load()
    const result = await svc.save({
      ...current,
      general: {
        ...(current.general ?? DEFAULT_GENERAL_CONFIG),
        loginPlatform: form.value.loginPlatform,
        loginMethod: form.value.loginMethod,
        threadRequest: form.value.threadRequest,
        delayRequest: form.value.delayRequest,
        delayThread: form.value.delayThread,
        captchaProvider: form.value.captchaProvider,
        captchaKeys: { ...form.value.captchaKeys },
        saveRunColumn: form.value.saveRunColumn,
        backupDB: form.value.backupDB,
        closeAfterDone: form.value.closeAfterDone,
        ipProvider: form.value.ipProvider,
        checkIpBeforeRun: form.value.checkIpBeforeRun,
        delayChangeIp: form.value.delayChangeIp,
        threadCheckInfo: form.value.threadCheckInfo,
        accountSource: accountForm.value.accountSource,
        accountSourcePath: accountForm.value.accountSourcePath,
        cloneHvUsername: accountForm.value.cloneHvUsername,
        cloneHvPassword: accountForm.value.cloneHvPassword,
        cloneHvProductId: accountForm.value.cloneHvProductId,
        cloneHvAmount: accountForm.value.cloneHvAmount,
        localeFake: form.value.localeFake,
        deepFakeInApi: form.value.deepFakeInApi,
        cookieUse: form.value.cookieUse,
        cookieLimit: form.value.cookieLimit,
        cookieLimitCount: form.value.cookieLimitCount,
        cookieMode: form.value.cookieMode,
        uaAddSpecs: form.value.uaAddSpecs,
        uaBuildFile: form.value.uaBuildFile,
        uaCustomType: form.value.uaCustomType,
        simNetworkMode: form.value.simNetworkMode,
        simNetworkType: form.value.simNetworkType,
      },
    })
    if (result !== 'OK') { appStore.notify('error', result); return }

    // UA pool save đã move sang InteractionSetupPage (auto-save trong form chính).
    // Auto-save — indicator trên toolbar đã show, không cần notify popup.
  } catch (err) {
    appStore.notify('error', 'Lỗi lưu cài đặt')
    throw err // rethrow để useAutoSave set status=error
  }
}
</script>

<template>
  <div class="gs-page">
    <div class="gs-page__header">
      <h2>Cài đặt chung</h2>
      <div class="gs-page__header-actions">
        <ProfileManager
          :profiles="profiles"
          :active-profile-id="activeProfileId"
          label="Cài đặt chung"
          @save="handleProfileSave"
          @load="handleProfileLoad"
          @update="handleSave"
          @clone="(id, name) => cloneProfile(id, name)"
          @delete="deleteProfile"
          @rename="renameProfile"
          @export="handleProfileExport"
          @import="handleProfileImport"
          @reset="resetForms"
        />
<span class="gs-save-status" :data-status="saveStatus">
          <template v-if="saveStatus === 'saving'">&#x25D0; Đang lưu...</template>
          <template v-else-if="saveStatus === 'saved'">&#x2714; Đã lưu</template>
          <template v-else-if="saveStatus === 'error'">&#x26A0; Lỗi lưu</template>
          <template v-else>&#x2022; Tự động lưu</template>
        </span>
        <button class="gs-btn gs-btn--danger" @click="$router.back()"><X :size="14" /> Đóng</button>
      </div>
    </div>

    <div class="gs-page__body">

      <!-- CỘT TRÁI: Đăng nhập + Nguồn TK + Mạng IP + UA pools -->
      <div class="gs-col">

        <!-- §1+2: Đăng nhập + Môi trường -->
        <fieldset class="gs-fieldset">
          <legend>Đăng nhập &amp; Môi trường</legend>
          <div class="gs-field">
            <label>Dạng đăng nhập:</label>
            <select v-model.number="form.loginMethod" class="gs-select">
              <option v-for="method in loginMethods" :key="method.value" :value="method.value" :disabled="method.disabled">
                {{ method.label }}{{ method.disabled ? ' (N/A)' : '' }}
              </option>
            </select>
          </div>
          <div class="gs-hint" v-if="loginMethods.find(m => m.value === form.loginMethod)?.description">
            {{ loginMethods.find(m => m.value === form.loginMethod)?.description }}
          </div>
          <div class="gs-inline-row">
            <div class="gs-field gs-field--inline">
              <label>Nghỉ: <FieldHelp field="delayRequest" /></label>
              <input type="number" v-model.number="form.delayRequest" min="0" step="100" class="gs-input gs-input--num" />
              <span class="gs-unit">ms</span>
            </div>
            <div class="gs-field gs-field--inline">
              <label>Delay luồng: <FieldHelp field="delayThread" /></label>
              <input type="number" v-model.number="form.delayThread" min="0" step="100" class="gs-input gs-input--num" />
              <span class="gs-unit">ms</span>
            </div>
            <div class="gs-hint" style="grid-column:1/-1;font-size:11px;color:var(--text-muted);">
              Số luồng đã chuyển sang <strong>Thiết lập chạy → Reg account / Verify</strong> để 2 bên tự cài luồng.
            </div>
          </div>
        </fieldset>

        <!-- ── Banner: User Agent đã chuyển sang Thiết lập chạy ───────────── -->
        <div class="gs-ua-moved-banner">
          <span class="gs-ua-moved-banner__icon">🎭</span>
          <div class="gs-ua-moved-banner__text">
            <strong>User Agent pool đã chuyển sang "Thiết lập chạy"</strong>
            <span>Toàn bộ thiết lập UA (3 pool Android/iPhone/Chrome + Dùng UA gốc + addVirtualSpec + buildNumFile) gom vào tab "Thiết lập chạy" → section "User Agent" để chạy 1 chỗ.</span>
          </div>
          <button class="gs-btn gs-btn--primary" @click="$router.push({ name: ROUTE_NAMES.INTERACTION_SETUP })">
            Mở Thiết lập chạy →
          </button>
        </div>

      </div><!-- /gs-col left -->

      <!-- CỘT PHẢI: Nguồn TK + Locale + Sim + Captcha + Sau khi chạy -->
      <div class="gs-col">

        <!-- §0 NGUỒN TÀI KHOẢN -->
        <fieldset class="gs-fieldset">
          <legend>Nguồn tài khoản</legend>
          <div class="gs-field gs-field--row">
            <label class="gs-radio">
              <input type="radio" v-model="accountForm.accountSource" value="folder" />
              <span>Từ thư mục (đọc file .txt)</span>
            </label>
            <label class="gs-radio">
              <input type="radio" v-model="accountForm.accountSource" value="file" />
              <span>Từ 1 file (chọn acc tick)</span>
            </label>
            <label class="gs-radio">
              <input type="radio" v-model="accountForm.accountSource" value="api" />
              <span>Mua từ API (CloneHV)</span>
            </label>
          </div>
          <template v-if="accountForm.accountSource === 'folder'">
            <div class="gs-field">
              <label>Thư mục nguồn:</label>
              <input
                type="text"
                :value="accountForm.accountSourcePath"
                class="gs-input"
                style="flex:1"
                placeholder="Chọn thư mục chứa file .txt tài khoản..."
                readonly
              />
              <button class="gs-btn-icon" title="Chọn thư mục" @click="handleBrowseAccountFolder">
                  <svg width="16" height="14" viewBox="0 0 16 14"><path d="M0 3 L0 13 Q0 14 1 14 L15 14 Q16 14 16 13 L16 5 Q16 4 15 4 L8 4 L6 2 L1 2 Q0 2 0 3 Z" fill="#FFB300"/></svg>
                </button>
            </div>
          </template>
          <template v-else-if="accountForm.accountSource === 'file'">
            <div class="gs-field">
              <label>File nguồn:</label>
              <input
                type="text"
                :value="accountForm.accountSourcePath"
                class="gs-input"
                style="flex:1"
                placeholder="Chọn 1 file .txt chứa accounts..."
                readonly
              />
              <button class="gs-btn-icon" title="Chọn file" @click="handleBrowseAccountFile"><FileText :size="13" style="color:#FFB300" /></button>
            </div>
            <div class="gs-hint">
              Load toàn bộ accounts từ file vào grid. Verify chỉ chạy các account bạn **tick chọn** ở danh sách.
            </div>
          </template>
          <template v-else>
            <div class="gs-hint">Nhập thông tin CloneHV để lấy tài khoản từ kho API.</div>
            <div class="gs-clonehv-grid">
              <div class="gs-field">
                <label>Username:</label>
                <input type="text" v-model="accountForm.cloneHvUsername" class="gs-input" style="flex:1" placeholder="username CloneHV..." />
              </div>
              <div class="gs-field">
                <label>Password:</label>
                <input type="password" v-model="accountForm.cloneHvPassword" class="gs-input" style="flex:1" placeholder="password CloneHV..." />
              </div>
              <div class="gs-field">
                <label>Product ID:</label>
                <input type="text" v-model="accountForm.cloneHvProductId" class="gs-input gs-input--num" placeholder="VD: 18" />
                <button
                  class="gs-btn gs-btn--secondary"
                  :disabled="!accountForm.cloneHvUsername || !accountForm.cloneHvProductId"
                  @click="handleCheckCloneHVStock"
                >Kiểm tra tồn kho</button>
              </div>
              <div class="gs-field">
                <label title="Số tài khoản mua mỗi batch từ CloneHV (match C# FetchBulkAccountsFromBResource amount param)">
                  Số lượng/batch:
                </label>
                <input
                  type="number"
                  v-model.number="accountForm.cloneHvAmount"
                  class="gs-input gs-input--num"
                  min="1" max="500"
                  placeholder="10"
                />
                <span class="gs-hint" style="margin:0 0 0 8px">acc mỗi lần mua (1–500)</span>
              </div>
            </div>
            <div v-if="cloneHvStockInfo" class="gs-hint">
              {{ cloneHvStockInfo.name }} — Tồn kho: <strong>{{ cloneHvStockInfo.amount }}</strong> — {{ cloneHvStockInfo.price.toLocaleString() }}đ/acc
            </div>
            <div v-if="cloneHvStockError" class="gs-hint gs-hint--error">{{ cloneHvStockError }}</div>
          </template>
        </fieldset>

        <!-- §LOCALE + SIM: Locale fake + Sim Network (gộp vì cùng Random/Match by IP) -->
        <fieldset class="gs-fieldset">
          <legend>Locale &amp; Sim Network</legend>
          <div class="gs-two-col">
            <div class="gs-subcol">
              <div class="gs-subcol__label">Locale fake</div>
              <div class="gs-field gs-field--row">
                <label class="gs-radio">
                  <input type="radio" v-model="form.localeFake" value="random" />
                  <span>Random</span>
                </label>
                <label class="gs-radio">
                  <input type="radio" v-model="form.localeFake" value="match-ip" />
                  <span>Match by IP</span>
                </label>
              </div>
              <label class="gs-checkbox">
                <input type="checkbox" v-model="form.deepFakeInApi" />
                <span class="gs-checkbox__content">
                  <span class="gs-checkbox__label">Deep Fake in API</span>
                  <span class="gs-checkbox__desc">Fake locale sâu hơn trong API call</span>
                </span>
              </label>
            </div>
            <div class="gs-subcol">
              <div class="gs-subcol__label">Sim Network</div>
              <div class="gs-field gs-field--row">
                <label class="gs-radio">
                  <input type="radio" v-model="form.simNetworkMode" value="random" />
                  <span>Random</span>
                </label>
                <label class="gs-radio">
                  <input type="radio" v-model="form.simNetworkMode" value="match-ip" />
                  <span>Match by IP</span>
                </label>
              </div>
              <div class="gs-field">
                <label>Loại mạng:</label>
                <select v-model="form.simNetworkType" class="gs-select">
                  <option value="LTE">LTE</option>
                  <option value="4G">4G</option>
                  <option value="3G">3G</option>
                  <option value="WIFI">WIFI</option>
                </select>
              </div>
            </div>
          </div>
        </fieldset>

        <!-- §5 HÀNH VI SAU KHI CHẠY -->
        <fieldset class="gs-fieldset">
          <legend>Sau khi chạy</legend>
          <div class="gs-checks-row">
            <label class="gs-checkbox gs-checkbox--primary">
              <input type="checkbox" v-model="form.saveRunColumn" />
              <span class="gs-checkbox__content">
                <span class="gs-checkbox__label">Lưu cột lần chạy</span>
                <span class="gs-checkbox__desc">Ghi IP chạy, hoạt động vào database</span>
              </span>
            </label>
            <label class="gs-checkbox gs-checkbox--primary">
              <input type="checkbox" v-model="form.backupDB" />
              <span class="gs-checkbox__content">
                <span class="gs-checkbox__label">Sao lưu database</span>
                <span class="gs-checkbox__desc">Tạo file backup sau mỗi lần chạy</span>
              </span>
            </label>
            <label class="gs-checkbox">
              <input type="checkbox" v-model="form.closeAfterDone" />
              <span class="gs-checkbox__content">
                <span class="gs-checkbox__label">Đóng app khi xong</span>
              </span>
            </label>
          </div>
        </fieldset>


      </div><!-- /gs-col right -->

    </div>
  </div>
</template>

<style scoped>
.gs-page { display: flex; flex-direction: column; height: 100%; overflow: hidden; }

.gs-page__header {
  height: var(--toolbar-height);
  background: var(--surface-elevated);
  border-bottom: 1px solid var(--border-default);
  display: flex; align-items: center; padding: 0 var(--space-4); flex-shrink: 0;
}
.gs-page__header h2 { font-size: var(--font-size-lg); font-weight: 700; flex: 1; }
.gs-page__header-actions { display: flex; gap: var(--space-2); }

.gs-page__body {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-4);
  padding: var(--space-3) var(--space-4);
  overflow-y: auto;
  align-content: start;
}

.gs-col { display: flex; flex-direction: column; gap: var(--space-3); align-self: start; }

/* Fieldset */
.gs-fieldset {
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  padding: var(--space-2) var(--space-3) var(--space-3);
  display: flex; flex-direction: column; gap: var(--space-2);
}
.gs-fieldset legend {
  font-size: var(--font-size-sm); font-weight: 700; color: var(--brand-primary);
  padding: 0 var(--space-2);
}
.gs-fieldset--readonly {
  border-color: var(--border-subtle, #2a2a2a);
  opacity: 0.85;
}
.gs-fieldset--readonly legend { color: var(--text-muted); }

/* Fields */
.gs-field { display: flex; align-items: center; gap: var(--space-3); }
.gs-field > label:first-child { font-size: var(--font-size-sm); color: var(--text-secondary); white-space: nowrap; min-width: 150px; }
.gs-field--row { gap: var(--space-4); }

/* CloneHV: 2 cột luôn — 4 field xếp 2x2 dù giao diện rộng hay hẹp */
.gs-clonehv-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3) var(--space-4);
  margin-top: var(--space-2);
}
.gs-clonehv-grid .gs-field > label:first-child { min-width: 90px; }
.gs-clonehv-grid .gs-input { min-width: 0; }
/* Nút "Kiểm tra tồn kho" luôn 1 dòng, không wrap dù ô cha hẹp */
.gs-clonehv-grid .gs-btn { white-space: nowrap; flex-shrink: 0; padding: var(--space-2) var(--space-3); }
.gs-clonehv-grid .gs-hint { white-space: nowrap; }

.gs-unit { font-size: var(--font-size-sm); color: var(--text-muted); white-space: nowrap; }

.gs-hint {
  font-size: var(--font-size-xs); color: var(--text-muted);
  padding: var(--space-1) var(--space-2);
  background: var(--surface-sunken); border-radius: var(--radius-sm);
  border-left: 2px solid var(--brand-primary);
}
.gs-hint--warning {
  border-color: var(--warning-solid, #f59e0b);
  color: var(--warning-text, #fbbf24);
  background: var(--warning-bg, rgba(245,158,11,0.08));
}
.gs-hint--error {
  border-color: var(--danger-solid, #ef4444);
  color: var(--danger-text, #fca5a5);
  background: var(--danger-bg, rgba(239,68,68,0.08));
}

/* Inputs */
.gs-input {
  flex: 1; background: var(--surface-sunken); border: 1px solid var(--border-default);
  color: var(--text-primary); padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm); font-size: var(--font-size-sm); font-family: inherit; outline: none;
}
.gs-input:focus { border-color: var(--border-focus); }
.gs-input--num { width: 80px; flex: none; text-align: right; }
.gs-input--mono { font-family: var(--font-mono); font-size: var(--font-size-xs); }

.gs-select {
  flex: 1; background: var(--surface-sunken); border: 1px solid var(--border-default);
  color: var(--text-primary); padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm); font-size: var(--font-size-sm); font-family: inherit; outline: none; cursor: pointer;
}
.gs-select:focus { border-color: var(--border-focus); }
.gs-select option:disabled { color: var(--text-disabled); font-style: italic; }
.gs-select--sm { flex: none; width: 100px; }

.gs-ua-custom-row {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: var(--space-3) var(--space-4);
  padding-top: var(--space-2);
  border-top: 1px solid var(--border-subtle, var(--border-default));
}

.gs-radio, .gs-checkbox { display: flex; align-items: flex-start; gap: var(--space-2); font-size: var(--font-size-sm); cursor: pointer; }
.gs-radio input, .gs-checkbox input { accent-color: var(--brand-primary); margin-top: 2px; flex-shrink: 0; }

.gs-checkbox__content { display: flex; flex-direction: column; gap: 1px; }
.gs-checkbox__label { font-size: var(--font-size-sm); color: var(--text-primary); line-height: 1.4; }
.gs-checkbox__desc { font-size: var(--font-size-xs); color: var(--text-muted); }

.gs-checkbox-divider {
  height: 1px;
  background: var(--border-default);
  margin: var(--space-1) 0;
}

/* Buttons */
.gs-btn {
  padding: var(--space-2) var(--space-4); border-radius: var(--radius-md);
  font-size: var(--font-size-sm); font-weight: 600; border: 1px solid var(--border-default); cursor: pointer;
}
.gs-btn--primary { background: var(--brand-primary); border-color: var(--brand-primary); color: white; }
.gs-btn--primary:hover { background: var(--brand-primary-hover); }
.gs-btn--secondary { background: transparent; border-color: var(--border-color, #555); color: var(--text-secondary, #aaa); }
.gs-btn--secondary:hover { border-color: var(--accent); color: var(--accent); }
.gs-btn--danger { background: var(--danger-solid); border-color: var(--danger-solid); color: white; }
.gs-btn--danger:hover { opacity: 0.9; }

.gs-save-status {
  display: inline-flex;
  align-items: center;
  padding: var(--space-1) var(--space-3);
  font-size: var(--font-size-sm);
  color: var(--text-muted);
  transition: color 0.2s;
}
.gs-save-status[data-status="saving"] { color: var(--brand-primary); }
.gs-save-status[data-status="saved"]  { color: var(--success-solid, #16a34a); }
.gs-save-status[data-status="error"]  { color: var(--danger-solid); }

/* 2-col split inside fieldset */
.gs-two-col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3) var(--space-4);
  align-items: start;
}
.gs-subcol {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.gs-subcol__label {
  font-size: var(--font-size-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
  margin-bottom: var(--space-1);
}

/* Checkbox/field inline trong cùng hàng */
.gs-inline-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-3) var(--space-4);
}
.gs-field--inline { flex: 0 0 auto; }

/* Checkboxes ngang hàng */
.gs-checks-row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3) var(--space-5);
  align-items: flex-start;
}

/* Account source read-only summary */
.gs-source-summary { display: flex; align-items: center; gap: var(--space-3); }
.gs-source-badge { font-size: var(--font-size-xs); font-weight: 600; padding: 2px 8px; border-radius: var(--radius-sm); }
.badge--api { background: var(--info-bg); color: var(--info-text); }
.badge--folder { background: var(--surface-sunken); color: var(--text-muted); }
.gs-source-path { font-size: var(--font-size-xs); color: var(--text-muted); font-family: var(--font-mono); }

/* Proxy summary chips */
.gs-proxy-summary { display: flex; align-items: center; gap: var(--space-2); flex-wrap: wrap; }
.gs-proxy-badge {
  font-size: var(--font-size-xs); font-weight: 600; padding: 2px 10px;
  border-radius: var(--radius-sm); background: rgba(255,183,77,0.12); color: #ffb74d;
}
.gs-proxy-chip {
  font-size: var(--font-size-xs); padding: 2px 8px;
  border-radius: var(--radius-sm); background: var(--surface-sunken); color: var(--text-muted);
}

/* Badge for readonly fieldsets */
.gs-badge { font-weight: 400; font-size: var(--font-size-xs); }
.gs-badge--readonly {
  color: var(--text-muted); background: var(--surface-sunken);
  padding: 1px 6px; border-radius: var(--radius-sm); margin-left: var(--space-2);
}

.gs-link-btn {
  background: none; border: none; color: var(--brand-primary);
  font-size: var(--font-size-xs); cursor: pointer; text-decoration: underline; padding: 0;
}

/* UA pool tabs */
.gs-ua-tabs {
  display: flex;
  gap: 4px;
  border-bottom: 1px solid var(--border-default);
  padding-bottom: var(--space-2);
}
.gs-ua-tab {
  padding: 4px 12px;
  border-radius: var(--radius-sm) var(--radius-sm) 0 0;
  border: 1px solid var(--border-default);
  border-bottom: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: var(--font-size-xs);
  font-weight: 500;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 6px;
  transition: all 0.15s;
}
.gs-ua-tab:hover { color: var(--text-primary); background: var(--surface-hover-subtle); }
.gs-ua-tab--active {
  background: var(--brand-primary);
  border-color: var(--brand-primary);
  color: white;
  font-weight: 600;
}
.gs-ua-tab__count {
  font-size: 10px;
  font-weight: 700;
  padding: 1px 5px;
  border-radius: 8px;
  background: var(--surface-hover-strong);
}
.gs-ua-tab--active .gs-ua-tab__count { background: rgba(0,0,0,0.2); }

.gs-textarea {
  width: 100%;
  box-sizing: border-box;
  background: var(--surface-sunken);
  border: 1px solid var(--border-default);
  color: var(--text-primary);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
  font-family: inherit;
  resize: vertical;
  outline: none;
  min-height: 60px;
}
.gs-textarea:focus { border-color: var(--border-focus); }
.gs-textarea--mono { font-family: var(--font-mono); font-size: var(--font-size-xs); }
.gs-ua-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.gs-ua-count { font-size: var(--font-size-xs); color: var(--text-muted); }
.gs-ua-file-badge {
  font-size: 10px; font-weight: 600;
  padding: 1px 6px; border-radius: 8px;
  background: rgba(225,48,108,0.12); color: var(--accent);
  margin-left: 4px;
}

/* File mode row */
.gs-ua-file-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  background: var(--surface-sunken);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
}
.gs-ua-file-path {
  flex: 1;
  font-size: var(--font-size-xs);
  font-family: var(--font-mono);
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gs-btn-icon {
  background: transparent;
  border: 1px solid var(--border-default);
  color: var(--text-secondary);
  border-radius: var(--radius-sm);
  padding: 2px 7px;
  font-size: 12px;
  cursor: pointer;
  flex-shrink: 0;
  transition: all 0.15s;
}
.gs-btn-icon:hover { border-color: var(--brand-primary); color: var(--brand-primary); }
.gs-btn-icon--danger:hover { border-color: var(--danger-solid); color: var(--danger-solid); }
.gs-btn--xs { font-size: var(--font-size-xs); padding: 3px 10px; min-width: unset; }

.gs-script-box {
  position: relative;
  margin-top: var(--space-2);
  background: var(--surface-sunken);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.gs-script-code {
  margin: 0;
  padding: var(--space-2) var(--space-3);
  padding-right: 80px;
  font-size: 11px;
  line-height: 1.6;
  color: var(--text-secondary);
  white-space: pre;
  overflow: auto;
  max-height: 120px;
}
.gs-script-copy {
  position: absolute;
  top: var(--space-2);
  right: var(--space-2);
  padding: 3px 10px;
  font-size: 11px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-default);
  background: var(--surface-elevated);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}
.gs-script-copy:hover { border-color: var(--brand-primary); color: var(--brand-primary); }
.gs-script-copy--copied { border-color: var(--success-solid, #22c55e); color: var(--success-solid, #22c55e); }

/* Banner thông báo UA panel đã chuyển sang Thiết lập chạy */
.gs-ua-moved-banner {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 16px;
  background: rgba(225,48,108,0.06);
  border: 1px dashed rgba(225,48,108,0.35);
  border-radius: var(--radius-md);
}
.gs-ua-moved-banner__icon { font-size: 22px; flex-shrink: 0; }
.gs-ua-moved-banner__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1 1 auto;
  min-width: 0;
}
.gs-ua-moved-banner__text strong {
  font-size: var(--font-size-sm);
  color: var(--text-primary);
  font-weight: 700;
}
.gs-ua-moved-banner__text span {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}
</style>
