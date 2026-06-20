<script setup lang="ts">
// AuthSourcePanel.vue — Panel tái sử dụng cho "Nguồn xác thực" (Mail / Phone).
//
// Trước đây nằm trong rp-shared-footer của InteractionSetupPage. Giờ tách ra
// component riêng để dùng được từ:
//   1. AuthSourcePage (top-level tab — giao diện rộng)
//   2. (legacy) InteractionSetupPage có thể vẫn render — share cùng form ref
//
// Component không sở hữu state — nhận `form` (VerifyConfig) qua prop và `saveNow`
// callback để force-save khi user click chọn provider.

import { ref, computed } from 'vue'
import { FetchWeMakeMailDomains, FetchMailHVDomains, FetchVietXFDomains } from '../../../../wailsjs/go/main/App'
import { BrowserOpenURL } from '../../../../wailsjs/runtime/runtime'
import { ChevronRight } from 'lucide-vue-next'
import type { VerifyConfig, MailProviderType } from '@/types/interaction.types'
import {
  ZEUS_X_ACCOUNT_CODES,
  DONGVANFB_ACCOUNT_TYPES,
  STORE1S_PRODUCTS,
} from '@/types/interaction.types'
import { useMailProviderStock } from '@/composables/useMailProviderStock'
import FieldHelp from '@/components/settings/FieldHelp.vue'
import SearchableSelect from '@/components/ui/SearchableSelect.vue'

const props = defineProps<{
  form: VerifyConfig
  saveNow?: () => void | Promise<void>
}>()

const formRef = computed(() => props.form)

const authSourceTab = ref<'mail' | 'phone'>('mail')

const TEMP_MAIL_PROVIDERS: { value: MailProviderType; label: string }[] = [
  { value: 'moakt',          label: 'Moakt' },
  //{ value: '@i2b.vn',       label: 'Mail1sec' },          // ẩn
  { value: 'mohmal',        label: 'Mohmal' },
  { value: 'tempmail-lol',  label: 'TempMail LOL' },
  //{ value: 'mailtm',        label: 'Mail.tm' },            // ẩn — không đọc được code
  { value: 'tempmail-plus', label: 'TempMail.plus' },
  { value: 'dropmail',      label: 'Dropmail' },
  { value: 'guerrillamail', label: 'GuerrillaMail' },
  { value: 'spam4me',       label: 'Spam4.me' },
  //{ value: 'temp-mail.org', label: 'Temp-Mail.org' },
  //{ value: 'mail.cx',       label: 'Mail.cx (cần API token)' },   // ẩn
  { value: 'mailtd',        label: 'Mail.cx' },
  //{ value: 'inboxes',       label: 'Inboxes.com' },        // ẩn
 // { value: 'dismail',       label: 'Dismail.top' },
  //{ value: 'mailymg',       label: 'Mailymg.com' },
  //{ value: 'altmails',      label: 'AltMails.com' },
  //{ value: 'onesecmail',   label: '1secmail.com' },        // ẩn
  //{ value: 'firetempmail', label: 'FireTempMail.com' },
  //{ value: 'fviainboxes',  label: 'FviaInboxes.com' },     // ẩn
  //{ value: 'byomde',       label: 'Byom.de' },
  //{ value: 'dinlaan',      label: 'Dinlaan.com' },
  //{ value: 'cryptogmail',  label: 'CryptoGmail.com' },
 // { value: 'buslink24',    label: 'Buslink24.com' },
 // { value: 'boxmailstore', label: 'BoxMail.store' },
  { value: 'mailermnx',   label: 'Mailer.mnx-family.com' },
  //{ value: 'tempforward',  label: 'TempForward.com' },
  //{ value: 'tempomintraccoon', label: 'Tempo.Mintraccoon.com' },
  { value: 'tempemail',    label: 'TempEmail.co' },
 // { value: 'tmpinbox',     label: 'TmpInbox.com' },
  //{ value: 'tenminutemail', label: '10MinuteMail.com' },
  { value: 'tempmailto',   label: 'TempMailTo.com' },
  { value: 'onesecemail',  label: '1secemail.com' },
  { value: 'tempmail100',  label: 'TempMail100.com' },
  { value: 'tempmailso',   label: 'TempMail.so' },
  // { value: 'priyoemail',   label: 'Priyo.email (cần API key)' },
  { value: 'tempmailorgpremium', label: 'Temp-Mail.org Premium' },
  { value: 'mailtempcom',  label: 'Mail-Temp.com' },
  //{ value: 'wemakemail',   label: 'WeMakeMail (cần API key)' }, // ẩn
  { value: 'mailhv',       label: 'MailHV (cần API key)' },
  //{ value: 'i2b',          label: 'Mail i2b.vn' },         // ẩn
  //{ value: 'vietxf',       label: 'VietXF' },              // ẩn
]
const RENT_MAIL_PROVIDERS: { value: MailProviderType; label: string; url: string }[] = [
  { value: 'zeus-x',        label: 'ZeusX',            url: 'https://zeus-x.ru' },
  { value: 'dongvanfb',     label: 'DongVanFB',        url: 'https://dongvanfb.net' },
  { value: 'store1s',       label: 'Store1s',          url: 'https://store1s.com' },
  { value: 'mail30s',       label: 'Mail30s',          url: 'https://mail30s.com' },
  { value: 'muamail',       label: 'MuaMail',          url: 'https://muamail.store' },
  { value: 'unlimitmail',   label: 'UnlimitMail',      url: 'https://unlimitmail.com' },
  { value: 'sptmail',       label: 'SPTMail',          url: 'https://sptmail.com' },
  { value: 'emailapiinfo',  label: 'EmailAPI.info',    url: 'https://emailapi.info' },
  { value: 'otpcheap',      label: 'OTP.cheap',        url: 'https://otp.cheap' },
  { value: 'shopgmail9999', label: 'ShopGmail9999',    url: 'https://shopgmail9999.com' },
  { value: 'rentgmail',     label: 'RentGmail.online', url: 'https://rentgmail.online' },
  { value: 'otpcodesms',    label: 'OtpCodesSms.site', url: 'https://otpcodesms.site' },
  { value: 'wmemail',       label: 'Wmemail.com',      url: 'https://wmemail.com' },
]

// HOTMAIL_PROVIDERS — 7 rent providers bán mail Hotmail/Outlook OAuth2.
// Khi user chọn 1 trong các provider này → hiển thị dropdown "Nguồn đọc code Hotmail ưu tiên".
// Backend dispatch theo OTPHotmailPriority qua ReadOTPWithPriority helper.
const HOTMAIL_PROVIDERS: MailProviderType[] = [
  'zeus-x', 'dongvanfb', 'store1s', 'mail30s', 'muamail', 'unlimitmail', 'wmemail',
]

// OTP_HOTMAIL_SOURCES — danh sách nguồn đọc OTP cho mail Hotmail.
//   dongvan → tools.dongvanfb.net/api/get_code_oauth2  (~2.6s)
//   unlimit → smail1s.com/get_messages?mode=oauth     (~4.5s)
// Primary fail → tự fallback sang reader còn lại.
const OTP_HOTMAIL_SOURCES: { value: 'dongvan' | 'unlimit'; label: string; desc: string }[] = [
  { value: 'dongvan', label: 'DongVanFB',  desc: 'tools.dongvanfb.net (mặc định, nhanh ~2.6s)' },
  { value: 'unlimit', label: 'UnlimitMail', desc: 'smail1s.com (~4.5s)' },
]

// providerURL — link trang nhà cung cấp hiện đang chọn (để user lấy API key).
const providerURL = computed(() => {
  return RENT_MAIL_PROVIDERS.find(p => p.value === props.form.mailProvider)?.url || ''
})

// openProviderSite — mở trang provider trong browser mặc định OS (Wails runtime).
function openProviderSite() {
  if (providerURL.value) BrowserOpenURL(providerURL.value)
}

const mailCount = computed(() => props.form.mailList.split('\n').filter(l => l.trim()).length)

const mailCategory = computed<'temp' | 'rent'>(() => {
  if (RENT_MAIL_PROVIDERS.some(p => p.value === props.form.mailProvider)) return 'rent'
  return 'temp'
})

// isHotmailProvider — true nếu provider hiện tại bán mail Hotmail OAuth2 (7 providers).
// Khi true, hiện dropdown "Nguồn đọc code Hotmail ưu tiên" cho user chọn DongVan/Unlimit.
const isHotmailProvider = computed<boolean>(() =>
  HOTMAIL_PROVIDERS.includes(props.form.mailProvider as MailProviderType),
)

function selectMailCategory(cat: 'temp' | 'rent') {
  if (cat === 'temp') props.form.mailProvider = 'moakt'
  else if (cat === 'rent') props.form.mailProvider = 'zeus-x'
  void props.saveNow?.()
}

function selectMailProvider(value: string) {
  props.form.mailProvider = value as any
  const dMap = props.form.tempMailDomains || {}
  props.form.tempMailDomain = dMap[value] || ''
  const tMap = props.form.tempMailTokens || {}
  props.form.tempMailToken = tMap[value] || ''
  void props.saveNow?.()
}

const PHONE_SMS_PROVIDERS = [
  { value: 'sms-activate', label: 'SMS-Activate' },
  { value: '5sim',         label: '5SIM.net' },
  { value: 'smshub',       label: 'SMSHub' },
  { value: 'smscodes',     label: 'SMSCodes' },
  { value: 'onlinesim',    label: 'OnlineSim' },
  { value: 'sms-man',      label: 'SMS-Man' },
  { value: 'smsbower',     label: 'SMSBower' },
  { value: 'textverified', label: 'TextVerified' },
]
const PHONE_RENT_PROVIDERS = [
  { value: 'rentsim-vn',   label: 'RentSim.vn' },
  { value: 'simrental',    label: 'SimRental' },
  { value: 'viotp',        label: 'ViOTP' },
  { value: 'otpsim',       label: 'OtpSim' },
]
const phoneCategory = ref<'sms' | 'rent'>('sms')
const phoneProvider = ref<string>('sms-activate')
const phoneApiKey = ref<string>('')
const phoneCountry = ref<string>('VN')
const phoneService = ref<string>('facebook')

const tempMailHasDomain = computed(() =>
  props.form.mailProvider === 'moakt' ||
  props.form.mailProvider === '@i2b.vn' ||
  props.form.mailProvider === 'tempmail-plus' ||
  props.form.mailProvider === 'wemakemail' ||
  props.form.mailProvider === 'mailhv' ||
  props.form.mailProvider === 'mailtd' ||
  props.form.mailProvider === 'vietxf'
)

// Provider yêu cầu token bắt buộc (không có thì không chạy được).
const tempMailTokenRequired = computed(() =>
  props.form.mailProvider === 'wemakemail' ||
  props.form.mailProvider === 'mailhv' ||
  props.form.mailProvider === 'vietxf'
)

// WeMakeMail domain fetch
const wmFetchLoading = ref(false)
const wmFetchError = ref('')
type WmDomains = { plan: string; free: string[]; paid: string[]; all: string[] }
const wmDomains = ref<WmDomains>({ plan: '', free: [], paid: [], all: [] })
const wmActiveTab = ref<'free' | 'paid' | 'all'>('free')

const wmTabDomains = computed(() => wmDomains.value[wmActiveTab.value] ?? [])
const wmHasDomains = computed(() => wmDomains.value.all.length > 0)

async function fetchWmDomains() {
  const apiKey = currentTempMailToken.value
  if (!apiKey) { wmFetchError.value = 'Nhập API key trước'; return }
  wmFetchLoading.value = true
  wmFetchError.value = ''
  wmDomains.value = { plan: '', free: [], paid: [], all: [] }
  try {
    const raw = await FetchWeMakeMailDomains(apiKey)
    if (raw.startsWith('ERROR:')) { wmFetchError.value = raw.slice(7).trim(); return }
    const parsed = JSON.parse(raw) as WmDomains
    wmDomains.value = { plan: parsed.plan ?? '', free: parsed.free ?? [], paid: parsed.paid ?? [], all: parsed.all ?? [] }
    // Auto chọn tab có domain
    wmActiveTab.value = parsed.free?.length ? 'free' : parsed.paid?.length ? 'paid' : 'all'
  } catch (e) {
    wmFetchError.value = String(e)
  } finally {
    wmFetchLoading.value = false
  }
}

// MailHV domain fetch
const vxFetchLoading = ref(false)
const vxFetchError = ref('')
const vxDomains = ref<string[]>([])

const vxHasDomains = computed(() => vxDomains.value.length > 0)

async function fetchVxDomains() {
  const apiKey = currentTempMailToken.value
  if (!apiKey) { vxFetchError.value = 'Nhập API key trước'; return }
  vxFetchLoading.value = true
  vxFetchError.value = ''
  vxDomains.value = []
  try {
    if (props.form.mailProvider === 'vietxf') {
      const raw = await FetchVietXFDomains(apiKey)
      if (raw.startsWith('ERROR:')) { vxFetchError.value = raw.slice(7).trim(); return }
      const parsed = JSON.parse(raw) as { domains: string[] }
      vxDomains.value = parsed.domains ?? []
    } else {
      const raw = await FetchMailHVDomains(apiKey)
      if (raw.startsWith('ERROR:')) { vxFetchError.value = raw.slice(7).trim(); return }
      const parsed = JSON.parse(raw) as { plan: string; free: string[]; paid: string[]; all: string[] }
      vxDomains.value = parsed.all ?? []
    }
  } catch (e) {
    vxFetchError.value = String(e)
  } finally {
    vxFetchLoading.value = false
  }
}

function vxFillAll() {
  currentTempMailDomain.value = vxDomains.value.join(', ')
}

function vxToggleDomain(d: string) {
  const list = currentTempMailDomain.value
    ? currentTempMailDomain.value.split(',').map(s => s.trim()).filter(Boolean)
    : []
  const idx = list.indexOf(d)
  if (idx >= 0) list.splice(idx, 1)
  else list.push(d)
  currentTempMailDomain.value = list.join(', ')
}

const vxSelectedSet = computed(() => {
  const raw = currentTempMailDomain.value
  return new Set(raw ? raw.split(',').map(s => s.trim()).filter(Boolean) : [])
})

// Điền nhóm domain vào ô nhập
function wmFillGroup(domains: string[]) {
  currentTempMailDomain.value = domains.join(', ')
}

// Toggle 1 domain: nếu đã có trong ô thì bỏ ra, chưa có thì thêm vào
function wmToggleDomain(d: string) {
  const list = currentTempMailDomain.value
    ? currentTempMailDomain.value.split(',').map(s => s.trim()).filter(Boolean)
    : []
  const idx = list.indexOf(d)
  if (idx >= 0) list.splice(idx, 1)
  else list.push(d)
  currentTempMailDomain.value = list.join(', ')
}

const wmSelectedSet = computed(() => {
  const raw = currentTempMailDomain.value
  return new Set(raw ? raw.split(',').map(s => s.trim()).filter(Boolean) : [])
})

function normalizeDomainList(raw: string): string {
  if (!raw) return ''
  return raw.split(/[\r\n,]+/).map(s => s.trim()).filter(Boolean).join(', ')
}
const currentTempMailDomain = computed<string>({
  get() {
    const p = props.form.mailProvider
    const map = props.form.tempMailDomains || {}
    const raw = map[p] !== undefined ? map[p] : (props.form.tempMailDomain || '')
    return normalizeDomainList(raw)
  },
  set(val: string) {
    const p = props.form.mailProvider
    if (!props.form.tempMailDomains) props.form.tempMailDomains = {}
    props.form.tempMailDomains[p] = val
    props.form.tempMailDomain = val
  },
})
const currentDomainPlaceholder = computed(() => {
  const p = props.form.mailProvider
  if (p === '@i2b.vn') return 'i2b.vn, other.net'
  if (p === 'moakt') return 'tmpbox.net, other.net'
  if (p === 'tempmail-plus') return 'mailto.plus, fexpost.com, fexbox.org'
  if (p === 'wemakemail') return 'Để trống = API tự chọn | hoặc: domain1.com, domain2.com'
  if (p === 'mailhv') return 'Để trống = API tự chọn | hoặc: domain1.com, domain2.com'
  if (p === 'mailtd') return 'Để trống = random tất cả | hoặc: sugtbt.com, qabq.com, nqmo.com, end.tw, uuf.me, 6n9.net'
  return 'domain1.com, domain2.com'
})

const currentTempMailToken = computed<string>({
  get() {
    const p = props.form.mailProvider
    const map = props.form.tempMailTokens || {}
    if (map[p] !== undefined) return map[p]
    return props.form.tempMailToken || ''
  },
  set(val: string) {
    const p = props.form.mailProvider
    if (!props.form.tempMailTokens) props.form.tempMailTokens = {}
    props.form.tempMailTokens[p] = val
    props.form.tempMailToken = val
  },
})
const currentProviderLabel = computed(() => {
  const p = props.form.mailProvider
  return TEMP_MAIL_PROVIDERS.find(x => x.value === p)?.label
    || RENT_MAIL_PROVIDERS.find(x => x.value === p)?.label
    || p
})
const isManualMailMode = computed(() => false)

// Mail provider stock check
const {
  zeusXLoading, zeusXError, selectedZeusXStock, checkZeusXStock,
  mail30sProducts, mail30sLoading, mail30sError, selectedMail30sProduct, checkMail30sStock,
  store1sLoading, store1sError, selectedStore1sStock, checkStore1sStock,
  dvfbLoading, dvfbError, selectedDvfbStock, checkDvfbStock,
} = useMailProviderStock(formRef)
</script>

<template>
  <div class="auth-source-panel">
    <div class="rp-sf-panel rp-sf-panel--auth">
      <div class="rp-sf-panel__header">
        <div class="rp-sf-panel__label">Nguồn xác thực</div>
        <div class="rp-auth-tabs">
          <button :class="['rp-auth-tab', { 'rp-auth-tab--active': authSourceTab === 'mail' }]"
            @click="authSourceTab = 'mail'">Mail</button>
          <button :class="['rp-auth-tab', { 'rp-auth-tab--active': authSourceTab === 'phone' }]"
            @click="authSourceTab = 'phone'">Phone</button>
        </div>
      </div>

      <div v-if="authSourceTab === 'mail'" class="rp-sf-auth-body">
        <div class="rp-mail-cats">
          <div class="rp-mail-cats__group">
            <button :class="['rp-mail-cat', { 'rp-mail-cat--active': mailCategory === 'temp' }]"
              @click="selectMailCategory('temp')">Temp Mail</button>
            <button :class="['rp-mail-cat', { 'rp-mail-cat--active': mailCategory === 'rent' }]"
              @click="selectMailCategory('rent')">Rent Mail</button>
          </div>
          <!-- Checkbox "Kiểm tra Live / Die" đã chuyển sang panel Verify trong InteractionSetupPage
               (gần "Tự gửi lại OTP khi timeout") — đặt cùng các cài đặt verify khác để dễ tìm. -->
        </div>

        <template v-if="mailCategory === 'temp'">
          <div class="rp-auth-row">
            <div class="rp-field rp-field--provider">
              <label>Temp Mail Provider: <span class="rp-field-hint">({{ TEMP_MAIL_PROVIDERS.length }} nhà cung cấp — gõ để tìm)</span></label>
              <SearchableSelect
                :model-value="form.mailProvider"
                :options="TEMP_MAIL_PROVIDERS"
                search-placeholder="Tìm provider..."
                @update:model-value="v => selectMailProvider(v as MailProviderType)"
              />
            </div>
          </div>
          <div class="rp-tempmail-config">
            <div v-if="tempMailHasDomain" class="rp-field rp-field--col rp-tempmail-cfg-item">
              <label>
                Domain cho <b>{{ currentProviderLabel }}</b>
                <span class="rp-field-hint">(cách nhau bằng dấu phẩy)</span>
              </label>
              <input type="text" v-model="currentTempMailDomain" class="vr-input"
                :placeholder="currentDomainPlaceholder" />
            </div>
            <div class="rp-field rp-field--col rp-tempmail-cfg-item">
              <label>
                Token / API key cho <b>{{ currentProviderLabel }}</b>
                <span class="rp-field-hint">{{ tempMailTokenRequired ? '(bắt buộc)' : '(tuỳ chọn)' }}</span>
              </label>
              <input type="text" v-model="currentTempMailToken" class="vr-input"
                :placeholder="tempMailTokenRequired ? 'API key bắt buộc (wm_live_...)' : 'Để trống nếu không cần'" />
            </div>
          </div>

          <!-- WeMakeMail: fetch + chọn domain theo tier -->
          <template v-if="form.mailProvider === 'wemakemail'">
            <div class="wm-domain-fetch">
              <button type="button" class="vr-btn vr-btn--check wm-fetch-btn"
                :disabled="wmFetchLoading || !currentTempMailToken"
                @click="fetchWmDomains">
                {{ wmFetchLoading ? 'Đang tải...' : '🔍 Tải domain từ API' }}
              </button>
              <span v-if="wmFetchError" class="vr-stock-badge vr-stock-badge--empty">{{ wmFetchError }}</span>
              <span v-if="!currentTempMailToken" class="wm-domain-note">← Nhập API key trước</span>
            </div>

            <div v-if="wmHasDomains" class="wm-domain-results">
              <!-- Tab: Free / Trả phí / Tất cả + gói tài khoản -->
              <div class="wm-tier-tabs">
                <button type="button"
                  v-for="tab in ([
                    { key: 'free', label: 'Free',     count: wmDomains.free.length },
                    { key: 'paid', label: 'Trả phí',  count: wmDomains.paid.length },
                    { key: 'all',  label: 'Tất cả',   count: wmDomains.all.length },
                  ] as const)"
                  :key="tab.key"
                  :class="['wm-tier-tab', `wm-tier-tab--${tab.key}`, { 'wm-tier-tab--active': wmActiveTab === tab.key }]"
                  :disabled="tab.count === 0"
                  @click="wmActiveTab = tab.key">
                  {{ tab.label }}
                  <span class="wm-tier-tab__cnt">{{ tab.count }}</span>
                </button>
                <span v-if="wmDomains.plan" class="wm-plan-badge">{{ wmDomains.plan }}</span>
                <span class="wm-tier-sep" />
                <button type="button" class="wm-domain-chip wm-domain-chip--action"
                  :disabled="wmTabDomains.length === 0"
                  @click="wmFillGroup(wmTabDomains)">
                  Dùng {{ wmActiveTab === 'free' ? 'free' : wmActiveTab === 'paid' ? 'trả phí' : 'tất cả' }}
                </button>
                <button type="button" class="wm-domain-chip wm-domain-chip--clear"
                  :disabled="wmSelectedSet.size === 0"
                  @click="currentTempMailDomain = ''">
                  Bỏ chọn tất cả
                </button>
                <span class="wm-domain-note">· click domain để chọn / bỏ chọn</span>
              </div>

              <!-- Danh sách domain chips -->
              <div class="wm-domain-chips">
                <button
                  v-for="d in wmTabDomains" :key="d"
                  type="button"
                  :class="['wm-domain-chip', { 'wm-domain-chip--active': wmSelectedSet.has(d) }]"
                  @click="wmToggleDomain(d)"
                  :title="wmSelectedSet.has(d) ? 'Bỏ ' + d : 'Thêm ' + d">
                  <span class="wm-domain-chip__check" v-if="wmSelectedSet.has(d)">✓</span>
                  {{ d }}
                </button>
              </div>
            </div>
          </template>

          <!-- MailHV / VietXF: fetch + chọn domain -->
          <template v-if="form.mailProvider === 'mailhv' || form.mailProvider === 'vietxf'">
            <div class="wm-domain-fetch">
              <button type="button" class="vr-btn vr-btn--check wm-fetch-btn"
                :disabled="vxFetchLoading || !currentTempMailToken"
                @click="fetchVxDomains">
                {{ vxFetchLoading ? 'Đang tải...' : '🔍 Tải domain từ API' }}
              </button>
              <span v-if="vxFetchError" class="vr-stock-badge vr-stock-badge--empty">{{ vxFetchError }}</span>
              <span v-if="!currentTempMailToken" class="wm-domain-note">← Nhập API key trước</span>
            </div>

            <div v-if="vxHasDomains" class="wm-domain-results">
              <div class="wm-tier-tabs">
                <span class="wm-plan-badge">{{ vxDomains.length }} domain</span>
                <span class="wm-tier-sep" />
                <button type="button" class="wm-domain-chip wm-domain-chip--action"
                  @click="vxFillAll">
                  Dùng tất cả
                </button>
                <button type="button" class="wm-domain-chip wm-domain-chip--clear"
                  :disabled="vxSelectedSet.size === 0"
                  @click="currentTempMailDomain = ''">
                  Bỏ chọn tất cả
                </button>
                <span class="wm-domain-note">· click domain để chọn / bỏ chọn</span>
              </div>
              <div class="wm-domain-chips">
                <button
                  v-for="d in vxDomains" :key="d"
                  type="button"
                  :class="['wm-domain-chip', { 'wm-domain-chip--active': vxSelectedSet.has(d) }]"
                  @click="vxToggleDomain(d)"
                  :title="vxSelectedSet.has(d) ? 'Bỏ ' + d : 'Thêm ' + d">
                  <span class="wm-domain-chip__check" v-if="vxSelectedSet.has(d)">✓</span>
                  {{ d }}
                </button>
              </div>
            </div>
          </template>
        </template>

        <template v-else-if="mailCategory === 'rent'">
          <div class="rp-auth-row">
            <div class="rp-field rp-field--provider">
              <label>
                Rent Mail Provider: <span class="rp-field-hint">({{ RENT_MAIL_PROVIDERS.length }} nhà cung cấp — gõ để tìm)</span>
                <a v-if="providerURL" href="#" class="rp-provider-link" @click.prevent="openProviderSite">🔗 Lấy API key tại {{ providerURL.replace('https://', '') }}</a>
              </label>
              <SearchableSelect
                :model-value="form.mailProvider"
                :options="RENT_MAIL_PROVIDERS"
                search-placeholder="Tìm provider..."
                @update:model-value="v => selectMailProvider(v as MailProviderType)"
              />
            </div>
            <div class="rp-field" style="max-width:160px">
              <label>Số mua batch đầu: <span class="rp-field-hint">(mặc định 50)</span></label>
              <input
                type="number"
                v-model.number="form.mailPoolBatch"
                class="vr-input"
                min="1"
                max="500"
                placeholder="50"
              />
            </div>
          </div>

          <!-- Nguồn đọc OTP ưu tiên — chỉ hiện cho 7 providers Hotmail OAuth2.
               Primary fail → tự fallback sang reader còn lại. -->
          <div v-if="isHotmailProvider" class="rp-auth-row">
            <div class="rp-field" style="max-width:520px">
              <label>
                Nguồn đọc code Hotmail ưu tiên:
                <span class="rp-field-hint">(primary fail → tự fallback)</span>
              </label>
              <div class="rp-otp-source-options">
                <label
                  v-for="src in OTP_HOTMAIL_SOURCES"
                  :key="src.value"
                  class="rp-otp-source-card"
                  :class="{ 'rp-otp-source-card--active': (form.otpHotmailPriority || 'dongvan') === src.value }"
                >
                  <input
                    type="radio"
                    :value="src.value"
                    v-model="form.otpHotmailPriority"
                    class="rp-otp-source-radio"
                  />
                  <div class="rp-otp-source-body">
                    <div class="rp-otp-source-label">{{ src.label }}</div>
                    <div class="rp-otp-source-desc">{{ src.desc }}</div>
                  </div>
                </label>
              </div>
            </div>
          </div>
        </template>

        <template v-if="form.mailProvider === 'tempmail-lol'">
          <div class="rp-auth-row">
            <div class="rp-field">
              <label>API Key <span style="font-size:11px;color:var(--text-muted)">(tuỳ chọn)</span>:</label>
              <input type="text" v-model="form.tempMailLolApiKey" class="vr-input" placeholder="Bearer token — để trống nếu dùng free tier" />
            </div>
          </div>
        </template>
        <template v-else-if="form.mailProvider === 'priyoemail'">
          <div class="rp-auth-row">
            <div class="rp-field">
              <label>API Key <span style="font-size:11px;color:var(--text-muted)">(bắt buộc)</span>:</label>
              <input type="text" v-model="form.priyoEmailApiKey" class="vr-input" placeholder="API key từ v3.priyo.email (free tier 100k req/tháng)" />
            </div>
          </div>
        </template>
        <template v-else-if="form.mailProvider === 'zeus-x'">
          <div class="rp-auth-row">
            <div class="rp-field">
              <label>API Key:</label>
              <input type="text" v-model="form.zeusXApiKey" class="vr-input" placeholder="API key zeus-x.ru..." />
            </div>
            <div class="rp-field">
              <label>Loại mail:</label>
              <select v-model="form.zeusXAccountCode" class="vr-select">
                <option v-for="ac in ZEUS_X_ACCOUNT_CODES" :key="ac.value" :value="ac.value">{{ ac.label }} — {{ ac.price }}</option>
              </select>
            </div>
            <div class="rp-stock-row">
              <button class="vr-btn vr-btn--check" :disabled="zeusXLoading" @click="checkZeusXStock">{{ zeusXLoading ? '...' : 'Kiểm tra tồn kho' }}</button>
              <span v-if="zeusXError" class="vr-stock-badge vr-stock-badge--empty">{{ zeusXError }}</span>
              <span v-else-if="selectedZeusXStock" :class="['vr-stock-badge', selectedZeusXStock.Instock > 0 ? 'vr-stock-badge--ok' : 'vr-stock-badge--empty']">{{ selectedZeusXStock.Instock > 0 ? `Tồn kho: ${selectedZeusXStock.Instock}` : 'Hết hàng' }}</span>
            </div>
          </div>
        </template>
        <template v-else-if="form.mailProvider === 'dongvanfb'">
          <div class="rp-auth-row">
            <div class="rp-field">
              <label>API Key:</label>
              <input type="text" v-model="form.dvfbApiKey" class="vr-input" placeholder="API key dongvanfb.net..." />
            </div>
            <div class="rp-field">
              <label>Loại mail:</label>
              <select v-model="form.dvfbAccountType" class="vr-select">
                <option v-for="ac in DONGVANFB_ACCOUNT_TYPES" :key="ac.value" :value="ac.value">{{ ac.label }} — {{ ac.price }}</option>
              </select>
            </div>
            <div class="rp-stock-row">
              <button class="vr-btn vr-btn--check" :disabled="dvfbLoading || !form.dvfbApiKey" @click="checkDvfbStock">{{ dvfbLoading ? '...' : 'Kiểm tra tồn kho' }}</button>
              <span v-if="dvfbError" class="vr-stock-badge vr-stock-badge--empty">{{ dvfbError }}</span>
              <span v-else-if="selectedDvfbStock" :class="['vr-stock-badge', selectedDvfbStock.quality > 0 ? 'vr-stock-badge--ok' : 'vr-stock-badge--empty']">{{ selectedDvfbStock.quality > 0 ? `Tồn kho: ${selectedDvfbStock.quality}` : 'Hết hàng' }}</span>
            </div>
          </div>
        </template>
        <template v-else-if="form.mailProvider === 'store1s'">
          <div class="rp-auth-row">
            <div class="rp-field">
              <label>API Key:</label>
              <input type="text" v-model="form.store1sApiKey" class="vr-input" placeholder="API key store1s.com..." />
            </div>
            <div class="rp-field">
              <label>Product ID:</label>
              <select v-model="form.store1sProductId" class="vr-select">
                <option v-for="p in STORE1S_PRODUCTS" :key="p.value" :value="p.value">{{ p.label }}</option>
              </select>
            </div>
            <div class="rp-stock-row">
              <button class="vr-btn vr-btn--check" :disabled="store1sLoading || !form.store1sApiKey" @click="checkStore1sStock">{{ store1sLoading ? '...' : 'Kiểm tra tồn kho' }}</button>
              <span v-if="store1sError" class="vr-stock-badge vr-stock-badge--empty">{{ store1sError }}</span>
              <span v-else-if="selectedStore1sStock !== null" :class="['vr-stock-badge', selectedStore1sStock > 0 ? 'vr-stock-badge--ok' : 'vr-stock-badge--empty']">{{ selectedStore1sStock > 0 ? `Tồn kho: ${selectedStore1sStock}` : 'Hết hàng' }}</span>
            </div>
          </div>
        </template>
        <template v-else-if="form.mailProvider === 'mail30s'">
          <div class="rp-auth-row">
            <div class="rp-field">
              <label>API Key:</label>
              <input type="text" v-model="form.mail30sApiKey" class="vr-input" placeholder="API key mailotp.com..." />
            </div>
            <div class="rp-field">
              <label>Sản phẩm:</label>
              <select v-model="form.mail30sProductSlug" class="vr-select">
                <option value="">-- Tải danh sách trước --</option>
                <!-- Fallback: giữ hiển thị slug đã lưu khi list chưa load (mở lại trang) -->
                <option
                  v-if="form.mail30sProductSlug && !mail30sProducts.some(p => p.slug === form.mail30sProductSlug)"
                  :value="form.mail30sProductSlug"
                >{{ form.mail30sProductSlug }} (đã lưu — bấm Tải sản phẩm để xem tồn kho)</option>
                <option v-for="p in mail30sProducts" :key="p.slug" :value="p.slug">{{ p.name }} — {{ p.price_display }} (tồn: {{ p.stock }})</option>
              </select>
            </div>
            <div class="rp-stock-row">
              <button class="vr-btn vr-btn--check" :disabled="mail30sLoading || !form.mail30sApiKey" @click="checkMail30sStock">{{ mail30sLoading ? '...' : 'Tải sản phẩm' }}</button>
              <span v-if="mail30sError" class="vr-stock-badge vr-stock-badge--empty">{{ mail30sError }}</span>
              <span v-else-if="selectedMail30sProduct !== null" :class="['vr-stock-badge', selectedMail30sProduct.stock > 0 ? 'vr-stock-badge--ok' : 'vr-stock-badge--empty']">{{ selectedMail30sProduct.stock > 0 ? `Tồn: ${selectedMail30sProduct.stock}` : 'Hết hàng' }}</span>
            </div>
          </div>
        </template>
        <template v-else-if="form.mailProvider === 'muamail'">
          <div class="rp-auth-row">
            <div class="rp-field">
              <label>API Key:</label>
              <input type="text" v-model="form.muaMailApiKey" class="vr-input" placeholder="API key muamail.store..." />
            </div>
            <div class="rp-field">
              <label>Product ID:</label>
              <input type="text" v-model="form.muaMailProductId" class="vr-input" placeholder="Product ID từ muamail.store..." />
            </div>
          </div>
        </template>
        <template v-else-if="form.mailProvider === 'unlimitmail'">
          <div class="rp-auth-row">
            <div class="rp-field">
              <label>API Key (Token):</label>
              <input type="text" v-model="form.unlimitMailApiKey" class="vr-input" placeholder="Token từ unlimitmail.com..." />
            </div>
            <div class="rp-field">
              <label>Product ID:</label>
              <input type="text" v-model="form.unlimitMailProductId" class="vr-input" placeholder="Product ID từ unlimitmail.com..." />
            </div>
          </div>
        </template>
        <template v-else-if="form.mailProvider === 'sptmail'">
          <div class="rp-auth-row">
            <div class="rp-field">
              <label>API Key:</label>
              <input type="text" v-model="form.sptMailApiKey" class="vr-input" placeholder="API key sptmail.com..." />
            </div>
            <div class="rp-field">
              <label>Service Code:</label>
              <input type="text" v-model="form.sptMailServiceCode" class="vr-input" placeholder="otpServiceCode từ sptmail.com..." />
            </div>
          </div>
        </template>
        <template v-else-if="form.mailProvider === 'emailapiinfo'">
          <div class="rp-auth-row">
            <div class="rp-field">
              <label>API Key:</label>
              <input type="text" v-model="form.emailAPIInfoApiKey" class="vr-input" placeholder="API key emailapi.info..." />
            </div>
            <div class="rp-field">
              <label>Product Code:</label>
              <input type="text" v-model="form.emailAPIInfoProductCode" class="vr-input" placeholder="gmail (mặc định)" />
            </div>
          </div>
        </template>
        <template v-else-if="form.mailProvider === 'otpcheap'">
          <div class="rp-auth-row">
            <div class="rp-field">
              <label>API Key:</label>
              <input type="text" v-model="form.otpCheapApiKey" class="vr-input" placeholder="API key otp.cheap..." />
            </div>
            <div class="rp-field">
              <label>Service ID:</label>
              <input type="text" v-model="form.otpCheapServiceId" class="vr-input" placeholder="8 (Facebook)" />
            </div>
          </div>
        </template>
        <template v-else-if="form.mailProvider === 'shopgmail9999'">
          <div class="rp-auth-row">
            <div class="rp-field">
              <label>API Key:</label>
              <input type="text" v-model="form.shopGmail9999ApiKey" class="vr-input" placeholder="API key shopgmail9999.com..." />
            </div>
            <div class="rp-field">
              <label>Service:</label>
              <input type="text" v-model="form.shopGmail9999Service" class="vr-input" placeholder="facebook (mặc định)" />
            </div>
          </div>
        </template>
        <template v-else-if="form.mailProvider === 'rentgmail'">
          <div class="rp-auth-row">
            <div class="rp-field">
              <label>Token:</label>
              <input type="text" v-model="form.rentGmailApiKey" class="vr-input" placeholder="Token rentgmail.online..." />
            </div>
            <div class="rp-field">
              <label>Platform:</label>
              <input type="text" v-model="form.rentGmailPlatform" class="vr-input" placeholder="facebook (mặc định)" />
            </div>
          </div>
        </template>
        <template v-else-if="form.mailProvider === 'otpcodesms'">
          <div class="rp-auth-row">
            <div class="rp-field">
              <label>API Key:</label>
              <input type="text" v-model="form.otpCodesSmsApiKey" class="vr-input" placeholder="API key otpcodesms.site..." />
            </div>
            <div class="rp-field">
              <label>Service ID:</label>
              <input type="text" v-model="form.otpCodesSmsServiceId" class="vr-input" placeholder="ID dịch vụ..." />
            </div>
          </div>
        </template>
        <template v-else-if="form.mailProvider === 'wmemail'">
          <div class="rp-auth-row">
            <div class="rp-field">
              <label>Token (API Key):</label>
              <input type="text" v-model="form.wmemailApiKey" class="vr-input" placeholder="Token wmemail.com..." />
            </div>
            <div class="rp-field">
              <label>Commodity ID:</label>
              <input type="text" v-model="form.wmemailCommodity" class="vr-input" placeholder="commodity_id (vd: gói Hotmail OAuth2)" />
            </div>
          </div>
        </template>

        <details v-if="isManualMailMode" class="rp-details rp-details--inset">
          <summary class="rp-details__summary">
            Danh sách mail <span class="rp-count">({{ mailCount }})</span>
            <ChevronRight :size="13" class="rp-details__caret" />
          </summary>
          <div class="rp-details__body">
            <textarea v-model="form.mailList" class="vr-textarea" rows="4" placeholder="Mỗi dòng một mail..." />
          </div>
        </details>
      </div>

      <div v-else-if="authSourceTab === 'phone'" class="rp-sf-auth-body">
        <div class="rp-mail-cats">
          <div class="rp-mail-cats__group">
            <button :class="['rp-mail-cat', { 'rp-mail-cat--active': phoneCategory === 'sms' }]"
              @click="phoneCategory = 'sms'">SMS OTP</button>
            <button :class="['rp-mail-cat', { 'rp-mail-cat--active': phoneCategory === 'rent' }]"
              @click="phoneCategory = 'rent'">Rent Phone</button>
          </div>
          <span class="rp-coming-soon-badge">🚧 Preview</span>
        </div>

        <template v-if="phoneCategory === 'sms'">
          <div class="rp-mail-providers">
            <button v-for="p in PHONE_SMS_PROVIDERS" :key="p.value"
              :class="['rp-mail-provider', { 'rp-mail-provider--active': phoneProvider === p.value }]"
              @click="phoneProvider = p.value">{{ p.label }}</button>
          </div>
        </template>

        <template v-else-if="phoneCategory === 'rent'">
          <div class="rp-mail-providers">
            <button v-for="p in PHONE_RENT_PROVIDERS" :key="p.value"
              :class="['rp-mail-provider', { 'rp-mail-provider--active': phoneProvider === p.value }]"
              @click="phoneProvider = p.value">{{ p.label }}</button>
          </div>
        </template>

        <div class="rp-auth-row">
          <div class="rp-field">
            <label>API Key:</label>
            <input type="text" v-model="phoneApiKey" class="vr-input" placeholder="API key provider đã chọn..." disabled />
          </div>
          <div class="rp-field">
            <label>Country:</label>
            <select v-model="phoneCountry" class="vr-select" disabled>
              <option value="VN">🇻🇳 Vietnam (+84)</option>
              <option value="ID">🇮🇩 Indonesia (+62)</option>
              <option value="PH">🇵🇭 Philippines (+63)</option>
              <option value="IN">🇮🇳 India (+91)</option>
              <option value="US">🇺🇸 United States (+1)</option>
            </select>
          </div>
          <div class="rp-field">
            <label>Service:</label>
            <select v-model="phoneService" class="vr-select" disabled>
              <option value="facebook">Facebook</option>
              <option value="instagram">Instagram</option>
              <option value="gmail">Gmail</option>
            </select>
          </div>
        </div>

        <div class="rp-phone-notice">
          🚧 <strong>Tính năng đang phát triển</strong> — khung UI đã sẵn sàng, backend chưa wire.
          Sẽ hỗ trợ: nhận OTP → auto-fill vào reg flow thay vì dùng email.
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* Lưu ý: phần lớn class (.rp-sf-panel, .rp-auth-tab, .rp-mail-cat, .rp-mail-provider,
   .rp-auth-row, .rp-field, .vr-input, .vr-select, .vr-btn, .vr-stock-badge, ...)
   được style global ở InteractionSetupPage.vue. Khi page InteractionSetupPage còn render
   panel này, CSS cũ vẫn áp dụng. Khi tạo AuthSourcePage standalone, page đó cần
   import css cần thiết hoặc duplicate style cần thiết.
   Để gọn — copy bản style đầy đủ từ InteractionSetupPage. */

.auth-source-panel {
  display: flex;
  flex-direction: column;
  width: 100%;
}

/* ── Section panel ─────────────────────────────────────────────────────────── */
.rp-sf-panel {
  display: flex;
  flex-direction: column;
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  padding: 12px 14px;
  gap: 10px;
}
.rp-sf-panel--auth { flex: 1 1 0; min-width: 0; }
.rp-sf-panel__header {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.rp-sf-panel__label {
  font-size: var(--font-size-sm);
  font-weight: 700;
  color: var(--text-primary);
}
.rp-sf-auth-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

/* ── Tabs Mail / Phone ────────────────────────────────────────────────────── */
.rp-auth-tabs {
  display: flex;
  gap: 4px;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  padding: 2px;
}
.rp-auth-tab {
  padding: 5px 14px;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: var(--font-size-xs);
  font-weight: 600;
  border-radius: 4px;
  cursor: pointer;
  transition: background var(--transition-fast), color var(--transition-fast);
}
.rp-auth-tab:hover:not(:disabled):not(.rp-auth-tab--active) {
  background: var(--surface-hover);
  color: var(--text-primary);
}
.rp-auth-tab--active {
  background: var(--brand-primary);
  color: #000;
  font-weight: 700;
}
.rp-auth-tab--active:hover:not(:disabled) {
  background: var(--brand-primary);
}

.rp-auth-live-check {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-left: auto;
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}

/* ── Categories Temp/Rent / SMS/Rent ──────────────────────────────────────── */
.rp-mail-cats {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.rp-mail-cats__group {
  display: flex;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  overflow: hidden;
}
.rp-mail-cat {
  padding: 6px 16px;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: var(--font-size-xs);
  font-weight: 600;
  cursor: pointer;
  transition: background var(--transition-fast), color var(--transition-fast);
}
.rp-mail-cat:hover:not(.rp-mail-cat--active) {
  background: var(--surface-hover);
  color: var(--text-primary);
}
.rp-mail-cat--active {
  background: var(--brand-primary);
  color: #000;
  font-weight: 700;
}

/* ── Providers row (Phone tab) ─────────────────────────────────────────────── */
.rp-mail-providers {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.rp-mail-provider {
  padding: 5px 10px;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.15s;
}
.rp-mail-provider:hover { background: var(--surface-hover); color: var(--text-primary); }
.rp-mail-provider--active {
  background: var(--brand-primary);
  color: #000;
  border-color: var(--brand-primary);
  font-weight: 700;
}

/* ── Generic field row ─────────────────────────────────────────────────────── */
.rp-auth-row {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: flex-end;
}
.rp-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
  flex: 1 1 200px;
  min-width: 200px;
}
.rp-field--col { flex-direction: column; }
.rp-field--provider { flex: 1 1 240px; min-width: 220px; }
.rp-field label {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--text-secondary);
}
.rp-field-hint {
  font-weight: 400;
  font-size: 11px;
  color: var(--text-muted);
  margin-left: 4px;
}
.rp-provider-link {
  font-weight: 500;
  font-size: 11px;
  color: var(--accent, #2563eb);
  margin-left: 8px;
  cursor: pointer;
  text-decoration: none;
}
.rp-provider-link:hover {
  text-decoration: underline;
}
.rp-tempmail-config {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}
.rp-tempmail-cfg-item { flex: 1 1 240px; min-width: 220px; }

.rp-stock-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.rp-checkbox {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: var(--font-size-xs);
  cursor: pointer;
  user-select: none;
}
.rp-checkbox input[type="checkbox"] { accent-color: var(--brand-primary); }

.rp-coming-soon-badge {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 8px;
  background: rgba(255,193,7,0.15);
  color: #ffc107;
  font-weight: 600;
}
.rp-phone-notice {
  padding: 8px 12px;
  border-radius: var(--radius-md);
  background: rgba(225,48,108,0.08);
  border: 1px dashed rgba(225,48,108,0.3);
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
  line-height: 1.6;
}
.rp-phone-notice strong { color: var(--accent); }

/* ── Manual mail textarea ──────────────────────────────────────────────────── */
.rp-details { margin-top: 4px; }
.rp-details__summary {
  cursor: pointer;
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 4px;
}
.rp-details__caret { transition: transform 0.15s; }
details[open] .rp-details__caret { transform: rotate(90deg); }
.rp-details__body { padding-top: 6px; }
.rp-count {
  font-weight: 400;
  color: var(--text-muted);
  margin-left: 4px;
}

/* ── Reusable input/select/btn (đảm bảo standalone đẹp ở AuthSourcePage) ──── */
.vr-input,
.vr-select {
  padding: 6px 10px;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-size: var(--font-size-xs);
  min-height: 32px;
}
.vr-input:focus,
.vr-select:focus {
  outline: none;
  border-color: var(--brand-primary);
  box-shadow: 0 0 0 2px var(--brand-primary-bg);
}
.vr-textarea {
  width: 100%;
  padding: 8px;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-size: var(--font-size-xs);
  font-family: var(--font-mono, monospace);
  resize: vertical;
}
.vr-btn {
  padding: 6px 12px;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  font-size: var(--font-size-xs);
  cursor: pointer;
  transition: background var(--transition-fast);
}
.vr-btn:hover:not(:disabled) { background: var(--surface-hover); }
.vr-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.vr-btn--check {
  background: var(--brand-primary-bg);
  color: var(--brand-primary);
  border-color: var(--brand-primary);
  font-weight: 600;
}
.vr-stock-badge {
  font-size: 11px;
  padding: 3px 8px;
  border-radius: 8px;
  font-weight: 600;
}
.vr-stock-badge--ok { background: rgba(76,175,80,0.15); color: #4caf50; }
.vr-stock-badge--empty { background: rgba(244,67,54,0.15); color: #f44336; }

/* ── WeMakeMail domain fetch panel ─────────────────────────────────────────── */
.wm-domain-fetch {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.wm-fetch-btn { flex-shrink: 0; }

.wm-domain-results {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 10px;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
}
.wm-domain-results__header {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.wm-domain-results__title {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--text-secondary);
}
.wm-domain-note {
  font-size: 11px;
  color: var(--text-muted);
}
.wm-domain-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
}
.wm-domain-chip {
  padding: 3px 10px;
  border-radius: 12px;
  border: 1px solid var(--border-default);
  background: var(--surface-elevated);
  color: var(--text-secondary);
  font-size: 11px;
  cursor: pointer;
  transition: all 0.12s;
  font-family: var(--font-mono, monospace);
}
.wm-domain-chip:hover { background: var(--surface-hover); color: var(--text-primary); border-color: var(--brand-primary); }
.wm-domain-chip--active {
  background: rgba(225,48,108,0.15);
  color: var(--accent);
  border-color: var(--accent);
  font-weight: 600;
}
.wm-domain-chip--action {
  background: var(--brand-primary-bg);
  color: var(--brand-primary);
  border-color: var(--brand-primary);
  font-weight: 600;
  border-radius: var(--radius-md);
}
.wm-domain-chip--action:hover { opacity: 0.85; }
.wm-domain-chip--clear {
  background: transparent;
  color: var(--text-muted, #888);
  border-color: var(--border-color, #555);
  border-radius: var(--radius-md);
}
.wm-domain-chip--clear:not(:disabled):hover { color: var(--danger, #e57373); border-color: var(--danger, #e57373); }
.wm-domain-chip--clear:disabled { opacity: 0.4; cursor: not-allowed; }
.wm-domain-chip__check {
  margin-right: 3px;
  color: var(--accent);
  font-weight: 700;
}

/* ── Tier tabs (Free / Premium / Tất cả) ───────────────────────────────────── */
.wm-tier-tabs {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}
.wm-tier-tab {
  padding: 3px 10px;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.12s;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.wm-tier-tab:hover:not(:disabled):not(.wm-tier-tab--active) {
  background: var(--surface-hover);
  color: var(--text-primary);
}
.wm-tier-tab:disabled { opacity: 0.4; cursor: not-allowed; }
.wm-tier-tab--active {
  background: var(--brand-primary-bg);
  border-color: var(--brand-primary);
  color: var(--brand-primary);
}
.wm-tier-tab--free.wm-tier-tab--active {
  background: rgba(76,175,80,0.12);
  border-color: #4caf50;
  color: #4caf50;
}
.wm-tier-tab--paid.wm-tier-tab--active {
  background: rgba(255,193,7,0.12);
  border-color: #ffc107;
  color: #ffc107;
}
.wm-plan-badge {
  padding: 2px 7px;
  border-radius: 10px;
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  background: rgba(255,193,7,0.15);
  border: 1px solid rgba(255,193,7,0.4);
  color: #ffc107;
  white-space: nowrap;
}
.wm-tier-tab__cnt {
  font-size: 10px;
  font-weight: 700;
  background: var(--surface-elevated);
  border-radius: 8px;
  padding: 0 5px;
  min-width: 16px;
  text-align: center;
  opacity: 0.85;
}
.wm-tier-sep {
  flex: 1;
}

/* ─── OTP Hotmail source priority cards ─────────────────────────────────── */
/* Theme-agnostic: dùng currentColor + transparent bg + accent-color cho radio,
   tránh hardcode màu để tự sync với light/dark theme của app. */
.rp-otp-source-options {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}
.rp-otp-source-card {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 8px 12px;
  border: 1px solid rgba(127, 127, 127, 0.3);
  border-radius: 6px;
  cursor: pointer;
  background: transparent;
  transition: border-color 0.15s, background 0.15s;
  min-width: 220px;
  color: inherit;
}
.rp-otp-source-card:hover {
  border-color: #2563eb;
  background: rgba(37, 99, 235, 0.04);
}
.rp-otp-source-card--active {
  border-color: #2563eb;
  background: rgba(37, 99, 235, 0.08);
}
.rp-otp-source-radio {
  margin-top: 3px;
  cursor: pointer;
  accent-color: #2563eb;
}
.rp-otp-source-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.rp-otp-source-label {
  font-weight: 600;
  font-size: 13px;
  color: inherit;
}
.rp-otp-source-desc {
  font-size: 11px;
  opacity: 0.7;
}
</style>
