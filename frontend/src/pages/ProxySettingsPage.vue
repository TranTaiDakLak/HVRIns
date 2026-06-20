<script setup lang="ts">
// ProxySettingsPage.vue — Kho proxy & provider đổi IP
// §1+§2 có tab Verify / Reg để cấu hình proxy riêng cho từng loại chạy.

import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { X, Globe, Lock } from 'lucide-vue-next'
import { useAppStore } from '@/stores/app.store'
import { getSettingsService, getInteractionService } from '@/bridge/client'
import type { GeneralConfig, IpConfig } from '@/types/settings.types'
import { IP_PROVIDERS, API_CHECK_IP_PROVIDERS, DEFAULT_IP_CONFIG, DEFAULT_GENERAL_CONFIG } from '@/types/settings.types'
import { ROUTE_NAMES } from '@/constants/routes'
import FieldHelp from '@/components/settings/FieldHelp.vue'
import { useAutoSave } from '@/composables/useAutoSave'

const router = useRouter()
const appStore = useAppStore()
const general = ref<GeneralConfig>({ ...DEFAULT_GENERAL_CONFIG })
const ipForm = ref<IpConfig>({ ...DEFAULT_IP_CONFIG })
const loading = ref(false)

// Tab: 'verify' | 'reg' — chọn context cho §1 + §2

// === Kiểm tra IP hiện tại ===
const currentIp = ref('')
const checkingIp = ref(false)

// === Proxy Mail ===
const proxyTempmailList = ref('')
const proxyTempmailCount = computed(() =>
  proxyTempmailList.value.split('\n').filter(l => l.trim() && !l.trim().startsWith('#')).length
)
const proxyMailSaveStatus = ref('')

const useProxyTempmail = ref(false)
const useProxyRentmail = ref(false)
let _interactionInitial = true
let _interactionSaveTimer: ReturnType<typeof setTimeout> | null = null

async function saveInteractionToggle() {
  try {
    const svc = await getInteractionService()
    const cfg = await svc.load()
    await svc.save({
      ...(cfg as any),
      useProxyTempmail: useProxyTempmail.value,
      useProxyGmail: useProxyRentmail.value,
    } as any)
  } catch {
    appStore.notify('error', 'Lỗi lưu toggle proxy mail')
  }
}

watch([useProxyTempmail, useProxyRentmail], () => {
  if (_interactionInitial) return
  if (_interactionSaveTimer) clearTimeout(_interactionSaveTimer)
  _interactionSaveTimer = setTimeout(saveInteractionToggle, 500)
})

async function loadProxyMailList(kind: 'tempmail' | 'gmail'): Promise<string> {
  try {
    const load = (window as any)?.go?.main?.App?.LoadProxyList
    if (typeof load === 'function') return await load(kind)
  } catch { /* ignore */ }
  return ''
}
async function saveProxyMailList(kind: 'tempmail' | 'gmail', content: string) {
  const save = (window as any)?.go?.main?.App?.SaveProxyList
  if (typeof save !== 'function') return
  try {
    const result = await save(kind, content)
    if (result !== 'OK') appStore.notify('error', String(result))
  } catch (err) {
    appStore.notify('error', 'Lỗi lưu: ' + String(err))
  }
}

let _proxyMailInitial = true
let _proxyTempmailSaveTimer: ReturnType<typeof setTimeout> | null = null

watch(proxyTempmailList, (val) => {
  if (_proxyMailInitial) return
  proxyMailSaveStatus.value = '• Đang lưu...'
  if (_proxyTempmailSaveTimer) clearTimeout(_proxyTempmailSaveTimer)
  _proxyTempmailSaveTimer = setTimeout(async () => {
    await saveProxyMailList('tempmail', val)
    proxyMailSaveStatus.value = '✓ Đã lưu'
    setTimeout(() => { proxyMailSaveStatus.value = '' }, 1500)
  }, 1000)
})

async function checkCurrentIp() {
  checkingIp.value = true
  currentIp.value = ''
  try {
    const checkViaProxy = (window as any)?.go?.main?.App?.CheckCurrentIPViaProxy
    if (typeof checkViaProxy === 'function') {
      const result = await checkViaProxy()
      currentIp.value = result || 'Không xác định'
      return
    }
    const provider = general.value.apiCheckIp ?? 0
    let ip = ''
    if (provider === 2) {
      const res = await fetch('https://ipinfo.io/json')
      const data = await res.json()
      ip = data.ip ?? ''
    } else if (provider === 3) {
      const res = await fetch('https://nordvpn.com/wp-admin/admin-ajax.php?action=get_user_info_data')
      const data = await res.json()
      ip = data.ip ?? ''
    } else {
      const res = await fetch('https://api.ipify.org?format=json')
      const data = await res.json()
      ip = data.ip ?? ''
    }
    currentIp.value = ip || 'Không xác định'
  } catch {
    currentIp.value = 'Lỗi kết nối'
  } finally {
    checkingIp.value = false
  }
}

onMounted(async () => {
  loading.value = true
  try {
    const svc = await getSettingsService()
    const data = await svc.load()
    if (data?.general) general.value = { ...DEFAULT_GENERAL_CONFIG, ...data.general }
    if (data?.ip) {
      const ip = { ...DEFAULT_IP_CONFIG, ...data.ip }
      // Migrate: gộp proxyStickyList vào proxyList nếu proxyList rỗng
      if (!ip.proxyList.trim() && ip.proxyStickyList?.trim()) {
        ip.proxyList = ip.proxyStickyList
      }
      if (!ip.regProxyList.trim() && ip.regProxyStickyList?.trim()) {
        ip.regProxyList = ip.regProxyStickyList
      }
      ip.proxyStickyList = ''
      ip.regProxyStickyList = ''
      // Normalize: proxyType rỗng → mặc định HTTP (tránh radio không được tích)
      if (!ip.proxyType) ip.proxyType = 'http'
      if (!ip.xproxyType) ip.xproxyType = 'http'
      if (!ip.regProxyType) ip.regProxyType = 'http'
      // Force reg luôn dùng list riêng (không fallback sang verify)
      ip.useVerifyProxyForReg = false
      ip.regIpProvider = general.value.ipProvider || ip.regIpProvider
      ipForm.value = ip
    }
    proxyTempmailList.value = await loadProxyMailList('tempmail')
    // Migrate: nếu tempmail rỗng nhưng rentmail có data → gộp vào tempmail
    if (!proxyTempmailList.value.trim()) {
      const rentData = await loadProxyMailList('gmail')
      if (rentData.trim()) proxyTempmailList.value = rentData
    }
    try {
      const isvc = await getInteractionService()
      const icfg = await isvc.load() as any
      useProxyTempmail.value = !!icfg?.useProxyTempmail
      useProxyRentmail.value = !!icfg?.useProxyGmail
    } catch { /* ignore */ }
  } catch {
    appStore.notify('error', 'Không tải được cài đặt proxy')
  } finally {
    loading.value = false
    setTimeout(() => { _proxyMailInitial = false; _interactionInitial = false }, 500)
  }
})

// ── Verify proxy computeds ──────────────────────────────────────────────────
const proxyCount  = computed(() => ipForm.value.proxyList.split('\n').filter(l => l.trim()).length)
const fptKeyCount = computed(() => ipForm.value.fptKeys.split('\n').filter(l => l.trim()).length)
const xproxyCount = computed(() => ipForm.value.xproxyList.split('\n').filter(l => l.trim()).length)
function countLines(t: string) { return t.split('\n').filter(l => l.trim()).length }

// ── Reg proxy computeds ─────────────────────────────────────────────────────
const regProxyCount = computed(() => ipForm.value.regProxyList.split('\n').filter(l => l.trim()).length)

// Provider label hiển thị trong badge §1
const verifyProviderLabel = computed(() =>
  IP_PROVIDERS.find(p => p.value === general.value.ipProvider)?.label ?? general.value.ipProvider
)

// Sync regIpProvider với ipProvider khi user đổi provider
watch(() => general.value.ipProvider, (v) => {
  ipForm.value.regIpProvider = v
  ipForm.value.useVerifyProxyForReg = false
})

// ── Auto-save ───────────────────────────────────────────────────────────────
const autoSavePayload = computed(() => ({
  ipProvider: general.value.ipProvider,
  apiCheckIp: general.value.apiCheckIp,
  ip: ipForm.value,
}))

const { status: saveStatus } = useAutoSave(autoSavePayload, async () => {
  await handleSave()
})

async function handleSave() {
  try {
    const svc = await getSettingsService()
    const current = await svc.load()
    const result = await svc.save({
      ...current,
      general: {
        ...(current.general ?? DEFAULT_GENERAL_CONFIG),
        ipProvider: general.value.ipProvider,
        apiCheckIp: general.value.apiCheckIp,
      },
      ip: ipForm.value,
    })
    if (result !== 'OK') throw new Error(result)
  } catch (err) {
    appStore.notify('error', 'Lỗi lưu cài đặt proxy')
    throw err
  }
}
</script>

<template>
  <div class="px-page">

    <!-- Header -->
    <div class="px-page__header">
      <h2>Proxy Settings</h2>
      <div class="px-page__header-actions">
        <span class="px-save-status" :data-status="saveStatus">
          <template v-if="saveStatus === 'saving'">&#x25D0; Đang lưu...</template>
          <template v-else-if="saveStatus === 'saved'">&#x2714; Đã lưu</template>
          <template v-else-if="saveStatus === 'error'">&#x26A0; Lỗi lưu</template>
          <template v-else>&#x2022; Tự động lưu</template>
        </span>
        <button class="px-btn px-btn--danger" @click="$router.back()"><X :size="14" /> Đóng</button>
      </div>
    </div>

    <!-- Body -->
    <div class="px-page__body">

      <div class="px-main-col">

        <!-- §1 NHÀ CUNG CẤP IP — dùng chung cho cả Reg và Verify -->
        <div class="px-section">
          <div class="px-section__header">
            <span class="px-section__num">1</span>
            <span class="px-section__title">Nhà cung cấp IP</span>
            <span class="px-section__badge">{{ verifyProviderLabel }}</span>
          </div>
          <div class="px-section__body">
            <div class="px-field">
              <label>Nhà cung cấp: <FieldHelp field="ipProvider" /></label>
              <select v-model="general.ipProvider" class="px-select">
                <option v-for="p in IP_PROVIDERS" :key="p.value" :value="p.value">{{ p.label }}</option>
              </select>
            </div>
            <div class="px-hint">
              Dùng chung cho cả Đăng ký và Verify. Credentials và danh sách proxy cấu hình ở §2 bên dưới.
              <button class="px-link-btn" @click="router.push({ name: ROUTE_NAMES.INTERACTION_SETUP })">Thiết lập chạy §4</button>
            </div>
          </div>
        </div>

        <!-- §2 CẤU HÌNH PROXY — VER + REG luôn hiển thị cùng lúc -->
        <div class="px-section">
          <div class="px-section__header">
            <span class="px-section__num">2</span>
            <span class="px-section__title">Cấu hình proxy</span>
          </div>
          <div class="px-section__body px-provider-body">

            <!-- ═══ REG (trên) ═══ -->
            <div class="px-dual-op-label px-dual-op-label--reg">✍️ REG — {{ verifyProviderLabel }}</div>

            <div v-if="general.ipProvider === 'none' || general.ipProvider === 'hma'" class="px-empty">
              <span class="px-empty__icon"><Globe :size="20" /></span>
              <div>
                <div class="px-empty__title">Không có proxy list</div>
                <div class="px-empty__sub">Provider hiện tại không dùng danh sách proxy.</div>
              </div>
            </div>

            <template v-else-if="general.ipProvider === 'proxy' || general.ipProvider === 'proxy_fixed'">
              <div class="px-field">
                <label>Loại proxy:</label>
                <label class="px-radio"><input type="radio" v-model="ipForm.regProxyType" value="http" /> HTTP</label>
                <label class="px-radio"><input type="radio" v-model="ipForm.regProxyType" value="socks5" /> SOCKS5</label>
              </div>
              <div class="px-field px-field--col">
                <textarea v-model="ipForm.regProxyList" class="px-textarea px-textarea--lg" rows="6" placeholder="host:port:user:pass&#10;host:port:user_area-XX_session-ID_life-N:pass&#10;Mỗi proxy một dòng..." />
                <div class="px-tab-hint">Tự động nhận diện session proxy (có <code>_session-</code>, <code>-zone-</code>). <span class="px-count">{{ regProxyCount }} proxy</span></div>
              </div>
            </template>

            <template v-else-if="['tinsoft','shoplike','netproxy','minproxy','netproxy_gb','proxy_popular','proxy_farm','fpt','xproxy'].includes(general.ipProvider)">
              <div class="px-hint">
                Provider <strong>{{ verifyProviderLabel }}</strong> — Reg dùng chung credentials với Verify (cấu hình ở §2 bên dưới).
              </div>
            </template>

            <div class="px-dual-sep"></div>

            <!-- ═══ VERIFY (dưới) ═══ -->
            <div class="px-dual-op-label px-dual-op-label--verify">✅ Verify — {{ verifyProviderLabel }}</div>

            <div v-if="general.ipProvider === 'none'" class="px-empty">
              <span class="px-empty__icon"><Globe :size="20" /></span>
              <div>
                <div class="px-empty__title">Không đổi IP</div>
                <div class="px-empty__sub">Request chạy trực tiếp từ IP máy chủ.</div>
              </div>
            </div>

            <div v-else-if="general.ipProvider === 'hma'" class="px-empty">
              <span class="px-empty__icon"><Lock :size="20" /></span>
              <div>
                <div class="px-empty__title">HMA VPN</div>
                <div class="px-empty__sub">Đổi IP qua VPN hệ thống. Cài HMA riêng và bật trước khi chạy.</div>
              </div>
            </div>

            <template v-else-if="general.ipProvider === 'proxy' || general.ipProvider === 'proxy_fixed'">
              <div class="px-field">
                <label>Loại proxy:</label>
                <label class="px-radio"><input type="radio" v-model="ipForm.proxyType" value="http" /> HTTP</label>
                <label class="px-radio"><input type="radio" v-model="ipForm.proxyType" value="socks5" /> SOCKS5</label>
              </div>
              <div class="px-field px-field--col">
                <textarea v-model="ipForm.proxyList" class="px-textarea px-textarea--lg" rows="6" placeholder="host:port:user:pass&#10;host:port:user_area-XX_session-ID_life-N:pass&#10;Mỗi proxy một dòng..." />
                <div class="px-tab-hint">Tự động nhận diện session proxy (có <code>_session-</code>, <code>-zone-</code>). <span class="px-count">{{ proxyCount }} proxy</span></div>
              </div>
              <div v-if="general.ipProvider === 'proxy_fixed'" class="px-hint">
                <strong>Proxy cố định:</strong> mỗi tài khoản dùng proxy gắn trong cột Proxy của danh sách TK.
              </div>
            </template>

            <template v-else-if="general.ipProvider === 'fpt'">
              <div class="px-field px-field--col">
                <label>FPT API Keys <span class="px-count">({{ fptKeyCount }} key)</span>:</label>
                <textarea v-model="ipForm.fptKeys" class="px-textarea" rows="6" placeholder="Mỗi key một dòng..." />
              </div>
            </template>

            <template v-else-if="general.ipProvider === 'xproxy'">
              <div class="px-field">
                <label>Link server:</label>
                <input type="text" v-model="ipForm.xproxyServiceUrl" placeholder="http://xproxy.vn/..." class="px-input" />
              </div>
              <div class="px-field">
                <label>Loại proxy:</label>
                <label class="px-radio"><input type="radio" v-model="ipForm.xproxyType" value="http" /> HTTP</label>
                <label class="px-radio"><input type="radio" v-model="ipForm.xproxyType" value="socks5" /> SOCKS5</label>
              </div>
              <div class="px-field">
                <label>Luồng / IP: <FieldHelp field="xproxyThreadPerIp" /></label>
                <input type="number" v-model.number="ipForm.xproxyThreadPerIp" min="1" class="px-input px-input--sm" />
              </div>
              <div class="px-field">
                <label>Chế độ:</label>
                <label class="px-radio"><input type="radio" v-model="ipForm.xproxyRunType" value="shared" /> Dùng chung proxy</label>
                <label class="px-radio"><input type="radio" v-model="ipForm.xproxyRunType" value="exclusive" /> Mỗi luồng 1 proxy</label>
              </div>
              <div class="px-field px-field--col">
                <label>Danh sách proxy dự phòng <span class="px-count">({{ xproxyCount }} proxy)</span>:</label>
                <textarea v-model="ipForm.xproxyList" class="px-textarea" rows="6" placeholder="Mỗi proxy một dòng..." />
              </div>
            </template>

            <template v-else-if="general.ipProvider === 'tinsoft'">
              <div class="px-field px-field--col">
                <label>Keys <span class="px-count">({{ countLines(ipForm.tinsoftKeys) }} key)</span>:</label>
                <textarea v-model="ipForm.tinsoftKeys" class="px-textarea" rows="6" placeholder="Mỗi key một dòng..." />
              </div>
              <div class="px-field">
                <label>Luồng / IP:</label>
                <input type="number" v-model.number="ipForm.tinsoftThreadPerIp" min="1" class="px-input px-input--sm" />
              </div>
            </template>

            <template v-else-if="general.ipProvider === 'shoplike'">
              <div class="px-field px-field--col">
                <label>Keys <span class="px-count">({{ countLines(ipForm.shoplikeKeys) }} key)</span>:</label>
                <textarea v-model="ipForm.shoplikeKeys" class="px-textarea" rows="6" placeholder="Mỗi key một dòng..." />
              </div>
              <div class="px-field">
                <label>Luồng / IP:</label>
                <input type="number" v-model.number="ipForm.shoplikeThreadPerIp" min="1" class="px-input px-input--sm" />
              </div>
            </template>

            <template v-else-if="general.ipProvider === 'netproxy'">
              <div class="px-field px-field--col">
                <label>Keys <span class="px-count">({{ countLines(ipForm.netproxyKeys) }} key)</span>:</label>
                <textarea v-model="ipForm.netproxyKeys" class="px-textarea" rows="6" placeholder="Mỗi key một dòng..." />
              </div>
              <div class="px-field">
                <label>Luồng / IP:</label>
                <input type="number" v-model.number="ipForm.netproxyThreadPerIp" min="1" class="px-input px-input--sm" />
              </div>
            </template>

            <template v-else-if="general.ipProvider === 'minproxy'">
              <div class="px-field px-field--col">
                <label>Keys <span class="px-count">({{ countLines(ipForm.minproxyKeys) }} key)</span>:</label>
                <textarea v-model="ipForm.minproxyKeys" class="px-textarea" rows="6" placeholder="Mỗi key một dòng..." />
              </div>
              <div class="px-field">
                <label>Luồng / IP:</label>
                <input type="number" v-model.number="ipForm.minproxyThreadPerIp" min="1" class="px-input px-input--sm" />
              </div>
            </template>

            <template v-else-if="general.ipProvider === 'netproxy_gb'">
              <div class="px-field">
                <label>Key dung lượng:</label>
                <input type="text" v-model="ipForm.netproxyGbKey" placeholder="Nhập key NetProxy dung lượng..." class="px-input px-input--mono" />
              </div>
            </template>

            <template v-else-if="general.ipProvider === 'proxy_popular'">
              <div class="px-field">
                <label>Access Token:</label>
                <input type="text" v-model="ipForm.proxyPopularAccessToken" placeholder="Nhập access token..." class="px-input px-input--mono" />
              </div>
              <div class="px-field px-field--col">
                <label>Keys <span class="px-count">({{ countLines(ipForm.proxyPopularKeys) }} key)</span>:</label>
                <textarea v-model="ipForm.proxyPopularKeys" class="px-textarea" rows="5" placeholder="Mỗi key một dòng..." />
              </div>
              <div class="px-field">
                <label>Luồng / IP:</label>
                <input type="number" v-model.number="ipForm.proxyPopularThreadPerIp" min="1" class="px-input px-input--sm" />
              </div>
            </template>

            <template v-else-if="general.ipProvider === 'proxy_farm'">
              <div class="px-field">
                <label>Access Token:</label>
                <input type="text" v-model="ipForm.proxyFarmAccessToken" placeholder="Nhập access token..." class="px-input px-input--mono" />
              </div>
              <div class="px-field px-field--col">
                <label>Keys <span class="px-count">({{ countLines(ipForm.proxyFarmKeys) }} key)</span>:</label>
                <textarea v-model="ipForm.proxyFarmKeys" class="px-textarea" rows="5" placeholder="Mỗi key một dòng..." />
              </div>
              <div class="px-field">
                <label>Luồng / IP:</label>
                <input type="number" v-model.number="ipForm.proxyFarmThreadPerIp" min="1" class="px-input px-input--sm" />
              </div>
            </template>

          </div>
        </div>

      </div><!-- /px-main-col -->

      <div class="px-side-col">

        <!-- §3 KIỂM TRA KẾT NỐI -->
        <div class="px-section">
          <div class="px-section__header">
            <span class="px-section__num">3</span>
            <span class="px-section__title">Kiểm tra kết nối</span>
          </div>
          <div class="px-section__body">
            <div class="px-ip-inline">
              <label class="px-ip-inline__label">API kiểm tra IP:</label>
              <select v-model.number="general.apiCheckIp" class="px-select px-ip-inline__select">
                <option v-for="p in API_CHECK_IP_PROVIDERS" :key="p.value" :value="p.value">{{ p.label }}</option>
              </select>
              <button class="px-btn px-btn--secondary px-ip-inline__btn" :disabled="checkingIp" @click="checkCurrentIp">
                {{ checkingIp ? '⏳ Đang kiểm tra...' : '🔍 Kiểm tra IP hiện tại' }}
              </button>
              <span v-if="currentIp" class="px-ip-result">{{ currentIp }}</span>
            </div>
          </div>
        </div>

        <!-- §4 RETRY & DELAY -->
        <div class="px-section">
          <div class="px-section__header">
            <span class="px-section__num">4</span>
            <span class="px-section__title">Retry & Delay IP</span>
          </div>
          <div class="px-section__body">
            <div class="px-retry-row">
              <div class="px-retry-field">
                <label class="px-retry-label">Số lần retry khi lỗi proxy</label>
                <div class="px-retry-input-wrap">
                  <input
                    v-model.number="ipForm.proxyRetry"
                    type="number" min="0" max="20" step="1"
                    class="px-input px-input--sm"
                  />
                  <span class="px-retry-unit">lần</span>
                </div>
                <span class="px-retry-hint">0 = không retry, dùng proxy tiếp theo ngay</span>
              </div>
              <div class="px-retry-field">
                <label class="px-retry-label">Delay trước khi đổi proxy</label>
                <div class="px-retry-input-wrap">
                  <input
                    v-model.number="ipForm.proxyDelayMs"
                    type="number" min="0" max="60000" step="500"
                    class="px-input px-input--sm"
                  />
                  <span class="px-retry-unit">ms</span>
                </div>
                <span class="px-retry-hint">0 = đổi ngay, 1000 = chờ 1 giây</span>
              </div>
            </div>
          </div>
        </div>

        <!-- §5 PROXY MAIL -->
        <div class="px-section px-section--stretch">
          <div class="px-section__header">
            <span class="px-section__num">5</span>
            <span class="px-section__title">📧 Proxy Mail</span>
            <span class="px-section__hint">{{ proxyMailSaveStatus }}</span>
          </div>
          <div class="px-section__body">
            <div class="px-proxy-mail-header">
              <label class="px-checkbox" title="Bật proxy cho temp mail khi verify">
                <input type="checkbox" v-model="useProxyTempmail" />
                <span>Proxy TempMail</span>
              </label>
              <span class="px-proxy-mail-sep" />
              <label class="px-checkbox" title="Bật proxy cho rent mail">
                <input type="checkbox" v-model="useProxyRentmail" />
                <span>Proxy RentMail</span>
              </label>
              <span class="px-proxy-mail-count">{{ proxyTempmailCount }} proxy</span>
            </div>
            <textarea
              v-model="proxyTempmailList"
              rows="6"
              class="px-proxy-mail-textarea"
              placeholder="Mỗi dòng 1 proxy (host:port:user:pass hoặc http://user:pass@host:port)..."
            />
            <div class="px-hint" style="margin-top: var(--space-2);">
              Tự động lưu khi nhập xong. Nếu danh sách RentMail để trống, tự động dùng chung danh sách TempMail.
            </div>
          </div>
        </div>

      </div><!-- /px-side-col -->

    </div>
  </div>
</template>

<style scoped>
.px-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.px-page__header {
  height: var(--toolbar-height);
  background: var(--surface-elevated);
  border-bottom: 1px solid var(--border-default);
  display: flex;
  align-items: center;
  padding: 0 var(--space-4);
  gap: var(--space-3);
}
.px-page__header h2 { flex: 1; font-size: var(--font-size-lg); font-weight: 700; }
.px-page__header-actions { display: flex; gap: var(--space-2); }

.px-page__body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-3) var(--space-4);
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3) var(--space-4);
  align-content: start;
}

.px-main-col { display: flex; flex-direction: column; gap: var(--space-3); }
.px-side-col { display: flex; flex-direction: column; gap: var(--space-3); align-self: start; }

/* Dual op labels — phân tách Verify / Reg trong cùng section */
.px-dual-op-label {
  font-size: var(--font-size-xs);
  font-weight: 700;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-bottom: var(--space-2);
}
.px-dual-op-label--verify {
  color: #16a34a;
  background: rgba(34,197,94,0.10);
  border: 1px solid rgba(34,197,94,0.25);
}
.px-dual-op-label--reg {
  color: #b45309;
  background: rgba(251,146,60,0.10);
  border: 1px solid rgba(251,146,60,0.25);
}
.px-dual-sep {
  border: none;
  border-top: 1px dashed var(--border-default);
  margin: var(--space-4) 0;
}

/* Section cards */
.px-section {
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  overflow: hidden;
}
.px-section--stretch { flex: 1; display: flex; flex-direction: column; }
.px-section--stretch .px-section__body { flex: 1; }

.px-section__header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  background: var(--surface-elevated);
  border-bottom: 1px solid var(--border-default);
}
.px-section__num {
  width: 22px; height: 22px;
  border-radius: 50%;
  background: var(--brand-primary); color: white;
  font-size: var(--font-size-xs); font-weight: 700;
  display: flex; align-items: center; justify-content: center;
  flex-shrink: 0;
}
.px-section__title { font-size: var(--font-size-sm); font-weight: 600; flex: 1; }
.px-section__badge {
  font-size: var(--font-size-xs); font-weight: 600;
  padding: 2px 8px; border-radius: var(--radius-sm);
  background: rgba(34,197,94,0.1); color: var(--brand-primary);
}
.px-section__body {
  padding: var(--space-4);
  display: flex; flex-direction: column; gap: var(--space-3);
}

/* Fields */
.px-field { display: flex; align-items: center; gap: var(--space-3); }
.px-field > label:first-child {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  white-space: nowrap;
  min-width: 130px;
}
.px-field--col { flex-direction: column; align-items: stretch; }
.px-field--col > label {
  min-width: auto !important;
  margin-bottom: var(--space-1);
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}
.px-count { font-size: var(--font-size-xs); color: var(--text-muted); font-weight: 400; }

/* Hint */
.px-hint {
  font-size: var(--font-size-xs); color: var(--text-muted);
  padding: var(--space-1) var(--space-2);
  background: var(--surface-sunken); border-radius: var(--radius-sm);
  border-left: 2px solid var(--brand-primary);
}
.px-hint--italic { font-style: italic; border-left: none; background: transparent; padding: 4px 0; }

/* Shared hint (reg dùng chung verify) */
.px-shared-hint {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
  font-size: var(--font-size-xs);
  color: var(--text-muted);
  padding: var(--space-2) var(--space-3);
  background: rgba(34,197,94,0.06);
  border: 1px solid rgba(34,197,94,0.2);
  border-radius: var(--radius-sm);
}
.px-shared-hint svg { color: var(--brand-primary); flex-shrink: 0; margin-top: 1px; }

/* Toggle (useVerifyProxyForReg) */
.px-toggle-label {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
  min-width: unset !important;
}
.px-toggle-input { display: none; }
.px-toggle-track {
  width: 34px; height: 18px; border-radius: 9px;
  background: var(--border-default);
  position: relative; transition: background 0.2s; flex-shrink: 0;
}
.px-toggle-input:checked ~ .px-toggle-track { background: var(--brand-primary); }
.px-toggle-thumb {
  position: absolute; top: 2px; left: 2px;
  width: 14px; height: 14px; border-radius: 50%;
  background: white; transition: left 0.2s;
  box-shadow: 0 1px 2px rgba(0,0,0,.25);
}
.px-toggle-input:checked ~ .px-toggle-track .px-toggle-thumb { left: 18px; }
.px-toggle-text {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-primary);
}

/* Proxy Mail section */
.px-section__hint {
  margin-left: auto;
  font-size: 11px;
  color: var(--text-muted);
  font-style: italic;
}
.px-proxy-mail-header {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2) var(--space-3);
  padding-bottom: var(--space-2);
  border-bottom: 1px dashed var(--border-default);
  margin-bottom: var(--space-2);
}
.px-proxy-mail-sep { width: 1px; height: 18px; background: var(--border-default); }
.px-checkbox {
  display: flex; align-items: center; gap: var(--space-2);
  font-size: var(--font-size-sm); color: var(--text-secondary); cursor: pointer;
}
.px-checkbox input { accent-color: var(--brand-primary); }
.px-proxy-mail-count {
  margin-left: auto;
  font-size: var(--font-size-xs);
  color: var(--text-muted);
}
.px-proxy-mail-textarea {
  width: 100%; box-sizing: border-box;
  font-family: 'Consolas', 'Courier New', monospace;
  font-size: 12px;
  background: var(--surface-sunken);
  border: 1px solid var(--border-default);
  color: var(--text-primary);
  padding: var(--space-2);
  border-radius: var(--radius-sm);
  outline: none; resize: vertical; min-height: 140px;
  white-space: pre; overflow-x: auto;
}
.px-proxy-mail-textarea:focus { border-color: var(--border-focus); }

/* Inputs */
.px-input {
  flex: 1; background: var(--surface-sunken); border: 1px solid var(--border-default);
  color: var(--text-primary); padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm); font-size: var(--font-size-sm); font-family: inherit; outline: none;
}
.px-input:focus { border-color: var(--border-focus); }
.px-input--sm { width: 70px; flex: none; text-align: right; }
.px-input--mono { font-family: var(--font-mono); font-size: var(--font-size-xs); }

.px-select {
  flex: 1; background: var(--surface-sunken); border: 1px solid var(--border-default);
  color: var(--text-primary); padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm); font-size: var(--font-size-sm); font-family: inherit; outline: none; cursor: pointer;
}
.px-select:focus { border-color: var(--border-focus); }

.px-textarea {
  width: 100%; background: var(--surface-sunken); border: 1px solid var(--border-default);
  color: var(--text-primary); padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm); font-family: var(--font-mono); font-size: var(--font-size-xs);
  resize: vertical; outline: none; line-height: 1.5;
}
.px-textarea:focus { border-color: var(--border-focus); }
.px-textarea::placeholder { color: var(--text-disabled); }
.px-textarea--lg { min-height: 100px; }

.px-radio { display: flex; align-items: center; gap: var(--space-2); font-size: var(--font-size-sm); cursor: pointer; }
.px-radio input { accent-color: var(--brand-primary); }

/* Provider config body */
.px-provider-body { min-height: 80px; }

/* Proxy type tabs */
.px-proxy-tabs {
  display: flex; gap: var(--space-1);
  border-bottom: 1px solid var(--border-default);
  margin-bottom: var(--space-1);
}
.px-proxy-tab {
  display: flex; align-items: center; gap: var(--space-2);
  padding: 6px 14px;
  font-size: var(--font-size-xs); font-weight: 500;
  color: var(--text-muted);
  background: transparent;
  border: 1px solid transparent;
  border-bottom: 2px solid transparent;
  border-radius: var(--radius-sm) var(--radius-sm) 0 0;
  cursor: pointer; transition: all 0.15s; margin-bottom: -1px;
}
.px-proxy-tab:hover { color: var(--text-primary); background: var(--surface-hover); }
.px-proxy-tab--active {
  color: var(--brand-primary);
  background: var(--surface-default);
  border-color: var(--border-default);
  border-bottom-color: var(--surface-default);
}
.px-proxy-tab__count {
  display: inline-flex; align-items: center; justify-content: center;
  min-width: 18px; height: 18px; padding: 0 5px;
  border-radius: 9px; font-size: 10px; font-weight: 700;
  background: var(--surface-sunken); color: var(--text-muted);
}
.px-proxy-tab--active .px-proxy-tab__count {
  background: rgba(34,197,94,0.15); color: var(--brand-primary);
}
.px-tab-hint { font-size: 11px; color: var(--text-muted); margin-top: var(--space-1); }
.px-tab-hint code {
  font-family: var(--font-mono);
  background: var(--surface-sunken);
  padding: 1px 4px; border-radius: 3px; font-size: 11px;
}

/* Empty state */
.px-empty { display: flex; align-items: flex-start; gap: var(--space-3); padding: var(--space-3) 0; }
.px-empty__icon { font-size: 24px; flex-shrink: 0; }
.px-empty__title { font-size: var(--font-size-sm); font-weight: 600; color: var(--text-secondary); margin-bottom: var(--space-1); }
.px-empty__sub { font-size: var(--font-size-xs); color: var(--text-muted); }

/* IP check inline */
.px-ip-inline { display: flex; align-items: center; gap: var(--space-2); flex-wrap: wrap; }
.px-ip-inline__label { font-size: var(--font-size-sm); color: var(--text-secondary); white-space: nowrap; }
.px-ip-inline__select { flex: 1; min-width: 160px; }
.px-ip-inline__btn { white-space: nowrap; flex-shrink: 0; }
.px-ip-result {
  font-size: var(--font-size-sm); font-family: var(--font-mono);
  color: var(--success-text); background: var(--surface-sunken);
  padding: var(--space-1) var(--space-3); border-radius: var(--radius-sm);
  border: 1px solid var(--border-default);
}

/* Retry & Delay section */
.px-retry-row { display: flex; gap: var(--space-6); flex-wrap: wrap; }
.px-retry-field { display: flex; flex-direction: column; gap: var(--space-1); min-width: 180px; }
.px-retry-label { font-size: var(--font-size-sm); font-weight: 500; color: var(--text-primary); }
.px-retry-input-wrap { display: flex; align-items: center; gap: var(--space-2); }
.px-input--sm { width: 90px; padding: var(--space-1) var(--space-2); font-size: var(--font-size-sm); border: 1px solid var(--border-default); border-radius: var(--radius-sm); background: var(--surface-default); color: var(--text-primary); text-align: right; }
.px-input--sm:focus { outline: none; border-color: var(--accent-primary); }
.px-retry-unit { font-size: var(--font-size-sm); color: var(--text-secondary); white-space: nowrap; }
.px-retry-hint { font-size: 11px; color: var(--text-tertiary); }

/* Buttons */
.px-btn {
  padding: var(--space-2) var(--space-4); border-radius: var(--radius-md);
  font-size: var(--font-size-sm); font-weight: 600;
  border: 1px solid var(--border-default); cursor: pointer;
}
.px-btn--primary { background: var(--brand-primary); border-color: var(--brand-primary); color: white; }
.px-btn--primary:hover:not(:disabled) { background: var(--brand-primary-hover); }
.px-btn--primary:disabled { opacity: 0.5; cursor: not-allowed; }
.px-btn--secondary { background: var(--surface-sunken); border-color: var(--border-strong); color: var(--text-primary); }
.px-btn--secondary:hover:not(:disabled) { background: var(--surface-hover); }
.px-btn--secondary:disabled { opacity: 0.5; cursor: not-allowed; }
.px-btn--danger { background: var(--danger-solid); border-color: var(--danger-solid); color: white; }
.px-btn--danger:hover { opacity: 0.9; }

.px-save-status {
  display: inline-flex; align-items: center;
  padding: var(--space-1) var(--space-3);
  font-size: var(--font-size-sm); color: var(--text-muted);
  transition: color 0.2s;
}
.px-save-status[data-status="saving"] { color: var(--brand-primary); }
.px-save-status[data-status="saved"]  { color: var(--success-solid, #16a34a); }
.px-save-status[data-status="error"]  { color: var(--danger-solid); }

.px-link-btn {
  background: none; border: none; color: var(--brand-primary);
  font-size: var(--font-size-xs); cursor: pointer; text-decoration: underline; padding: 0;
}
</style>
