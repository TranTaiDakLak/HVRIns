<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { ArrowUpToLine, PenLine, CheckCircle2, Eye, EyeOff, RefreshCw, Trash2 } from 'lucide-vue-next'
import { getUploadSiteService } from '@/services/client'
import type { UploadSiteConfig } from '@/types/upload-site.types'
import { DEFAULT_UPLOAD_SITE_CONFIG } from '@/types/upload-site.types'
import { GetBancloneProducts } from '../../wailsjs/go/app/App'
import { useUploadLogStore } from '@/stores/uploadLog.store'
import { storeToRefs } from 'pinia'

interface BancloneProduct {
  id: string
  code: string   // stock code thật (rỗng nếu chưa có admin cookie)
  name: string
  categoryName: string
  price: string
  amount: number
}

const form = ref<UploadSiteConfig>(JSON.parse(JSON.stringify(DEFAULT_UPLOAD_SITE_CONFIG)))
const saveStatus = ref<'idle' | 'saving' | 'saved' | 'error'>('idle')
const products = ref<BancloneProduct[]>([])
const loadingProducts = ref(false)
const loadError = ref('')
const showAdminPassword = ref(false)
// dùng khi chưa có stock codes thật — chỉ để xem thông tin, không ghi vào form.code
const previewProductId = ref('')

onMounted(async () => {
  try {
    const svc = await getUploadSiteService()
    const data = await svc.load()
    if (data) form.value = { ...DEFAULT_UPLOAD_SITE_CONFIG, ...data }
  } catch { /* fallback to defaults */ }
  const cached = loadProductsFromCache()
  if (cached.length > 0) applyProductList(cached)
  else if (form.value.apiKey) loadProducts(true)
})

let _saveTimer: ReturnType<typeof setTimeout> | null = null
let _statusResetTimer: ReturnType<typeof setTimeout> | null = null
watch(form, () => {
  if (_saveTimer) clearTimeout(_saveTimer)
  _saveTimer = setTimeout(async () => {
    saveStatus.value = 'saving'
    try {
      const svc = await getUploadSiteService()
      const r = await svc.save(form.value)
      saveStatus.value = r === 'OK' ? 'saved' : 'error'
      if (_statusResetTimer) clearTimeout(_statusResetTimer)
      _statusResetTimer = setTimeout(() => { saveStatus.value = 'idle' }, 1500)
    } catch { saveStatus.value = 'error' }
  }, 600)
}, { deep: true })

onBeforeUnmount(() => {
  if (_saveTimer) clearTimeout(_saveTimer)
  if (_statusResetTimer) clearTimeout(_statusResetTimer)
  _saveTimer = null
  _statusResetTimer = null
})


const PRODUCTS_CACHE_KEY = 'banclone_products_list'
const STOCK_CACHE_KEY = 'banclone_stock_map'

function saveProductsCache(list: BancloneProduct[]) {
  try { localStorage.setItem(PRODUCTS_CACHE_KEY, JSON.stringify(list)) } catch { /* ignore */ }
}

function loadProductsFromCache(): BancloneProduct[] {
  try {
    const raw = localStorage.getItem(PRODUCTS_CACHE_KEY)
    return raw ? JSON.parse(raw) : []
  } catch { return [] }
}

function applyStockCache(list: BancloneProduct[]): BancloneProduct[] {
  try {
    const raw = localStorage.getItem(STOCK_CACHE_KEY)
    if (!raw) return list
    const map: Record<string, string> = JSON.parse(raw)
    return list.map(p => ({ ...p, code: p.code || map[p.id] || '' }))
  } catch { return list }
}

function applyProductList(list: BancloneProduct[]) {
  products.value = list
  if (list.some(p => p.code)) {
    const matched = list.some(p => p.code === form.value.code)
    if (!matched && list[0]) form.value.code = list[0].code
  } else {
    if (list[0]) previewProductId.value = list[0].id
  }
}

async function loadProducts(silent = false) {
  if (!form.value.apiKey) {
    if (!silent) loadError.value = 'Nhập API key trước'
    return
  }
  loadingProducts.value = true
  if (!silent) loadError.value = ''
  try {
    const result = await GetBancloneProducts(form.value.apiKey, form.value.adminUsername ?? '', form.value.adminPassword ?? '')
    if (result.startsWith('ERR|')) {
      if (!silent) loadError.value = result.slice(4)
    } else {
      let list = JSON.parse(result) as BancloneProduct[]
      if (!list.some(p => p.code)) list = applyStockCache(list)
      saveProductsCache(list)
      // lưu riêng stock map để backward compat
      const stockMap: Record<string, string> = {}
      list.forEach(p => { if (p.code) stockMap[p.id] = p.code })
      if (Object.keys(stockMap).length) localStorage.setItem(STOCK_CACHE_KEY, JSON.stringify(stockMap))
      applyProductList(list)
    }
  } catch (e) {
    if (!silent) loadError.value = String(e)
  } finally {
    loadingProducts.value = false
  }
}

const hasStockCodes = computed(() => products.value.some(p => p.code))
const selectedProduct = computed(() => {
  if (hasStockCodes.value) return products.value.find(p => p.code === form.value.code)
  return products.value.find(p => p.id === previewProductId.value)
})

const showApiKey = ref(false)
const canStart = computed(() => form.value.reg.enabled || form.value.ver.enabled)
const saveStatusText = computed(() => {
  if (saveStatus.value === 'saving') return 'Đang lưu...'
  if (saveStatus.value === 'saved') return '✓ Đã lưu'
  if (saveStatus.value === 'error') return '⚠ Lỗi lưu'
  return 'Tự động lưu'
})

// ── Log panel — dùng store để log không mất khi chuyển tab ────────────────
const uploadLogStore = useUploadLogStore()
const { logs, totalUploaded, stats } = storeToRefs(uploadLogStore)
const logEl = ref<HTMLElement | null>(null)

function clearLogs() { uploadLogStore.clearLogs() }

// Tin nhắn mới ở đầu — không cần scroll
</script>

<template>
  <div class="usite-page">

    <!-- Header -->
    <div class="usite-header">
      <ArrowUpToLine :size="16" class="usite-header__icon" />
      <span class="usite-header__title">Đẩy tài khoản</span>
      <span class="usite-header__sub">Đẩy tài khoản lên banclone.pro tự động</span>
      <span class="usite-save-status" :data-status="saveStatus">{{ saveStatusText }}</span>
    </div>

    <!-- Settings + Log -->
    <div class="usite-main">

    <!-- Left: Settings -->
    <div class="usite-body">

      <!-- §1 Nguồn dữ liệu -->
      <div class="usite-section">
        <div class="usite-section__title">Nguồn dữ liệu</div>

        <div class="usite-src" :class="{ 'usite-src--off': !form.reg.enabled }">
          <div class="usite-src__header">
            <span class="usite-src__tag usite-src__tag--reg">
              <PenLine :size="11" /> Reg
            </span>
            <span class="usite-src__desc">Tài khoản reg xong sẽ được đẩy lên</span>
            <label class="usite-toggle">
              <input type="checkbox" v-model="form.reg.enabled" />
              <span class="usite-toggle__track"><span class="usite-toggle__thumb" /></span>
            </label>
          </div>
        </div>

        <div class="usite-src" :class="{ 'usite-src--off': !form.ver.enabled }">
          <div class="usite-src__header">
            <span class="usite-src__tag usite-src__tag--ver">
              <CheckCircle2 :size="11" /> Ver
            </span>
            <span class="usite-src__desc">Tài khoản verify xong sẽ được đẩy lên</span>
            <label class="usite-toggle">
              <input type="checkbox" v-model="form.ver.enabled" />
              <span class="usite-toggle__track"><span class="usite-toggle__thumb" /></span>
            </label>
          </div>
        </div>

        <div v-if="!canStart" class="usite-source-hint">
          Bật ít nhất 1 nguồn để tự động đẩy lên khi chạy
        </div>
      </div>

      <!-- §2 Cài đặt API -->
      <div class="usite-section">
        <div class="usite-section__title">Cài đặt API (banclone.pro)</div>

        <div class="usite-field">
          <label class="usite-label">API Key</label>
          <div class="usite-inline-row">
            <input
              class="usite-input usite-input--grow"
              :type="showApiKey ? 'text' : 'password'"
              v-model="form.apiKey"
              placeholder="API key của bạn"
            />
            <button class="usite-btn usite-btn--ghost usite-btn--icon" @click="showApiKey = !showApiKey" tabindex="-1">
              <Eye v-if="!showApiKey" :size="13" />
              <EyeOff v-else :size="13" />
            </button>
          </div>
        </div>

        <!-- Tài khoản admin banclone.pro -->
        <div class="usite-field">
          <label class="usite-label">Tài khoản banclone.pro <span class="usite-check-hint">(để tự lấy mã kho)</span></label>
          <input class="usite-input" type="text" v-model="form.adminUsername" placeholder="Tên đăng nhập" autocomplete="username" />
        </div>
        <div class="usite-field">
          <label class="usite-label">Mật khẩu</label>
          <div class="usite-inline-row">
            <input
              class="usite-input usite-input--grow"
              :type="showAdminPassword ? 'text' : 'password'"
              v-model="form.adminPassword"
              placeholder="Mật khẩu"
              autocomplete="current-password"
            />
            <button class="usite-btn usite-btn--ghost usite-btn--icon" @click="showAdminPassword = !showAdminPassword" tabindex="-1">
              <Eye v-if="!showAdminPassword" :size="13" />
              <EyeOff v-else :size="13" />
            </button>
            <button
              class="usite-btn usite-btn--primary usite-btn--load"
              :class="{ 'usite-btn--loading': loadingProducts }"
              @click="loadProducts()"
              :disabled="loadingProducts"
              title="Tải danh sách kho hàng"
            >
              <RefreshCw :size="12" :class="{ 'usite-spin': loadingProducts }" />
              <span>{{ loadingProducts ? 'Đang tải...' : 'Tải kho' }}</span>
            </button>
          </div>
          <div v-if="loadError" class="usite-error">{{ loadError }}</div>
        </div>

        <!-- Kho hàng -->
        <div class="usite-field">
          <label class="usite-label">Kho hàng</label>
          <div v-if="products.length > 0">
            <!-- Có stock codes thật → dropdown ghi thẳng vào form.code -->
            <select v-if="hasStockCodes" class="usite-select" v-model="form.code">
              <option v-for="p in products" :key="p.id" :value="p.code">
                {{ p.code }} | {{ p.name }} — {{ p.categoryName }}
              </option>
            </select>
            <!-- Chưa có stock codes → chỉ xem thông tin, không ghi vào form.code -->
            <select v-else class="usite-select" v-model="previewProductId">
              <option v-for="p in products" :key="p.id" :value="p.id">
                {{ p.id }} | {{ p.name }} — {{ p.categoryName }}
              </option>
            </select>
            <div v-if="selectedProduct" class="usite-product-info">
              Giá: {{ selectedProduct.price }} VND &nbsp;|&nbsp;
              Tồn kho: {{ selectedProduct.amount.toLocaleString() }}
            </div>
          </div>
          <div v-else class="usite-product-info" style="margin-top:0">
            Bấm <b>Tải kho</b> để xem danh sách sản phẩm
          </div>
        </div>

        <div class="usite-field">
          <label class="usite-check-row">
            <input type="checkbox" v-model="form.filterDuplicate" class="usite-checkbox" />
            <span class="usite-check-label">Lọc trùng UID <span class="usite-check-hint">(filter=1)</span></span>
          </label>
        </div>
      </div>

      <!-- §3 Timing -->
      <div class="usite-section">
        <div class="usite-section__title">Timing</div>
        <div class="usite-grid-3">
          <div class="usite-field">
            <label class="usite-label">Delay check (giây)</label>
            <input class="usite-input usite-input--num" type="number" v-model.number="form.delayCheckSec" min="1" max="300" />
          </div>
          <div class="usite-field">
            <label class="usite-label">Acc up 1 lần</label>
            <input class="usite-input usite-input--num" type="number" v-model.number="form.accPerBatch" min="1" max="9999" />
          </div>
          <div class="usite-field">
            <label class="usite-label">Delay sau up (giây)</label>
            <input class="usite-input usite-input--num" type="number" v-model.number="form.delayBetweenBatchSec" min="0" max="300" />
          </div>
        </div>
      </div>

    </div><!-- /usite-body -->

    <!-- Right: Log panel -->
    <div class="usite-log">
      <div class="usite-log__header">
        <span class="usite-log__title">Nhật ký đẩy</span>
        <span v-if="totalUploaded > 0" class="usite-log__total" title="Đã upload thành công trong session này">✓ <b>{{ totalUploaded }}</b></span>
        <span v-if="stats.pendingCount > 0" class="usite-log__pending" title="Còn trong hàng đợi (in-memory)">⏳ <b>{{ stats.pendingCount }}</b></span>
        <span v-if="stats.duplicateSkipped > 0" class="usite-log__dup" title="UID trùng đã bị bỏ qua (không push 2 lần cùng UID)">⊘ <b>{{ stats.duplicateSkipped }}</b></span>
        <span v-if="stats.totalFailed > 0" class="usite-log__failed" title="Tổng lượt push lỗi (đã retry)">⚠ <b>{{ stats.totalFailed }}</b></span>
        <span v-if="stats.consecutiveFailures > 0" class="usite-log__retry" :title="stats.lastError">⟲ Retry liên tiếp: {{ stats.consecutiveFailures }}</span>
        <button class="usite-log__clear" @click="clearLogs" title="Xóa nhật ký (file + UI)">
          <Trash2 :size="12" />
        </button>
      </div>
      <div class="usite-log__body" ref="logEl">
        <div v-if="logs.length === 0" class="usite-log__empty">
          Chưa có hoạt động. Bật "Ver/Reg xong tự đẩy lên site" trong Thiết lập chạy để bắt đầu.
        </div>
        <div
          v-for="(entry, i) in logs"
          :key="i"
          class="usite-log__entry"
          :class="`usite-log__entry--${entry.type}`"
        >
          <span class="usite-log__time">{{ entry.time }}</span>
          <span class="usite-log__msg">{{ entry.msg }}</span>
        </div>
      </div>
    </div><!-- /usite-log -->

    </div><!-- /usite-main -->
  </div>
</template>

<style scoped>
.usite-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
  background: var(--surface-base);
}

/* Header */
.usite-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border-default);
  background: var(--surface-elevated);
  flex-shrink: 0;
}
.usite-header__icon { color: var(--brand-primary); }
.usite-header__title { font-size: 14px; font-weight: 700; color: var(--text-primary); }
.usite-header__sub { font-size: 11px; color: var(--text-muted); }
.usite-save-status { font-size: 11px; color: var(--text-muted); margin-left: 4px; }
.usite-save-status[data-status="saved"] { color: var(--success-text); }
.usite-save-status[data-status="error"] { color: var(--danger-text); }

/* Main 2-col layout */
.usite-main {
  flex: 1;
  display: flex;
  overflow: hidden;
  min-height: 0;
}

/* Left: Settings */
.usite-body {
  width: 480px;
  flex-shrink: 0;
  overflow-y: auto;
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  border-right: 1px solid var(--border-default);
}

/* Right: Log panel */
.usite-log {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow: hidden;
}
.usite-log__header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-subtle);
  background: var(--surface-elevated);
  flex-shrink: 0;
}
.usite-log__title {
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--text-muted);
}
.usite-log__total {
  font-size: 11px;
  color: var(--success-text, #22c55e);
  margin-left: 2px;
}
.usite-log__pending {
  font-size: 11px;
  color: var(--warning-text, #f59e0b);
}
.usite-log__dup {
  font-size: 11px;
  color: var(--text-muted);
}
.usite-log__failed {
  font-size: 11px;
  color: var(--danger-text, #ef4444);
  opacity: 0.85;
}
.usite-log__retry {
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 9999px;
  background: rgba(239, 68, 68, 0.12);
  color: var(--danger-text, #ef4444);
  border: 1px solid rgba(239, 68, 68, 0.28);
}
.usite-log__clear {
  margin-left: auto;
  background: none;
  border: none;
  cursor: pointer;
  color: var(--text-disabled);
  padding: 2px 4px;
  border-radius: var(--radius-sm);
  display: flex;
  align-items: center;
}
.usite-log__clear:hover { color: var(--text-muted); background: var(--surface-hover); }
.usite-log__body {
  flex: 1;
  overflow-y: auto;
  padding: 8px 10px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-family: monospace;
  font-size: 11px;
}
.usite-log__empty {
  color: var(--text-disabled);
  font-family: inherit;
  font-size: 11px;
  text-align: center;
  padding: 24px 16px;
  line-height: 1.6;
}
.usite-log__entry {
  display: flex;
  gap: 8px;
  padding: 2px 4px;
  border-radius: 3px;
  line-height: 1.5;
}
.usite-log__entry--ok  { color: var(--success-text, #22c55e); }
.usite-log__entry--err { color: var(--danger-text, #f87171); }
.usite-log__entry--info{ color: var(--text-muted); }
.usite-log__time { flex-shrink: 0; opacity: 0.55; }
.usite-log__msg  { flex: 1; word-break: break-word; }

/* Section */
.usite-section {
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  padding: 11px 13px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.usite-section__title {
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.07em;
  color: var(--text-muted);
  padding-bottom: 5px;
  border-bottom: 1px solid var(--border-subtle);
}

/* Source card */
.usite-src {
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  overflow: hidden;
  transition: opacity var(--transition-fast);
}
.usite-src--off { opacity: 0.48; }

.usite-src__header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 7px 10px;
  background: var(--surface-sunken);
}

.usite-src__tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.03em;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
}
.usite-src__tag--reg { background: rgba(249,115,22,.14); color: #f97316; border: 1px solid rgba(249,115,22,.28); }
.usite-src__tag--ver { background: rgba(34,197,94,.14); color: #22c55e; border: 1px solid rgba(34,197,94,.28); }

.usite-src__desc {
  flex: 1;
  font-size: 11px;
  color: var(--text-muted);
}

/* Toggle */
.usite-toggle { display: inline-flex; align-items: center; cursor: pointer; margin-left: auto; }
.usite-toggle input { display: none; }
.usite-toggle__track {
  width: 32px; height: 17px;
  border-radius: 9999px;
  background: var(--border-strong);
  position: relative;
  transition: background var(--transition-fast);
}
.usite-toggle input:checked + .usite-toggle__track { background: var(--brand-primary); }
.usite-toggle__thumb {
  position: absolute; top: 2px; left: 2px;
  width: 13px; height: 13px;
  border-radius: 50%; background: #fff;
  transition: left var(--transition-fast);
}
.usite-toggle input:checked + .usite-toggle__track .usite-toggle__thumb { left: 17px; }

/* Source hint */
.usite-source-hint {
  font-size: 11px;
  color: var(--text-disabled);
  text-align: center;
}

/* Fields */
.usite-field { display: flex; flex-direction: column; gap: 4px; }
.usite-label { font-size: 11px; font-weight: 600; color: var(--text-secondary); }
.usite-grid-3 { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 9px; }
.usite-inline-row { display: flex; gap: 6px; align-items: center; }

/* Checkbox row */
.usite-check-row {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  cursor: pointer;
  user-select: none;
}
.usite-checkbox {
  width: 14px; height: 14px;
  accent-color: var(--brand-primary);
  cursor: pointer;
  flex-shrink: 0;
}
.usite-check-label { font-size: 12px; font-weight: 600; color: var(--text-primary); }
.usite-check-hint { font-size: 11px; font-weight: 400; color: var(--text-muted); }

/* Inputs */
.usite-input {
  background: var(--surface-sunken);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-size: 12px;
  padding: 5px 8px;
  outline: none;
  width: 100%;
  box-sizing: border-box;
}
.usite-input:focus { border-color: var(--border-focus); box-shadow: 0 0 0 2px var(--brand-primary-bg); }
.usite-input::placeholder { color: var(--text-disabled); }
.usite-input--grow { flex: 1; width: auto; min-width: 0; }
.usite-input--num { text-align: right; }

/* Eye toggle button */
.usite-btn {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 5px 10px; font-size: 12px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-default);
  background: var(--surface-elevated);
  color: var(--text-primary);
  cursor: pointer;
  outline: none; flex-shrink: 0;
}
.usite-btn--ghost {
  background: transparent; border-color: transparent; color: var(--text-muted); padding: 5px 6px;
}
.usite-btn--ghost:hover { background: var(--surface-hover); color: var(--text-primary); }
.usite-btn--icon { padding: 4px 7px; }

.usite-btn--primary {
  background: var(--brand-primary);
  border-color: var(--brand-primary);
  color: #fff;
  padding: 5px 10px;
  font-size: 11px;
  gap: 4px;
}
.usite-btn--primary:hover:not(:disabled) { opacity: 0.88; }
.usite-btn--primary:disabled { opacity: 0.55; cursor: not-allowed; }

@keyframes usite-spin { to { transform: rotate(360deg); } }
.usite-spin { animation: usite-spin 0.8s linear infinite; }

.usite-select {
  background: var(--surface-sunken);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-size: 12px;
  padding: 5px 8px;
  outline: none;
  width: 100%;
  box-sizing: border-box;
  cursor: pointer;
}
.usite-select:focus { border-color: var(--border-focus); }

.usite-product-info {
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 4px;
  padding: 3px 6px;
  background: var(--surface-sunken);
  border-radius: var(--radius-sm);
}

.usite-error {
  font-size: 11px;
  color: var(--danger-text, #e53);
  margin-top: 4px;
}

.usite-code-hint {
  font-size: 10px;
  color: var(--text-disabled);
  margin-top: 3px;
}
.usite-code-hint code {
  font-family: monospace;
  background: var(--surface-sunken);
  padding: 1px 4px;
  border-radius: 3px;
}
.usite-textarea {
  resize: vertical;
  min-height: 56px;
  font-size: 11px;
  font-family: monospace;
  line-height: 1.4;
}
.usite-warn-badge {
  font-size: 10px;
  font-weight: 500;
  color: #f59e0b;
  background: rgba(245,158,11,.12);
  border: 1px solid rgba(245,158,11,.3);
  border-radius: 4px;
  padding: 1px 6px;
  margin-left: 6px;
}
</style>
