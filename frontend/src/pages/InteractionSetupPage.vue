<script setup lang="ts">
// InteractionSetupPage.vue — Thiết lập chạy (Run Profile)
// Là trung tâm cấu hình theo profile: nguồn TK, verify, register, mạng, kết quả.

import { ref, reactive, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useRouter } from 'vue-router'
import { X, ChevronDown, ChevronUp, ChevronRight, ArrowUpToLine } from 'lucide-vue-next'
import { ROUTE_PATHS } from '@/constants/routes'
import { useAppStore } from '@/stores/app.store'
import { getInteractionService, getFileDialogService, getSettingsService, getVerifyRunnerService } from '@/services/client'
import type { VerifyConfig, MailProviderType, PlatformUAConfig } from '@/types/interaction.types'
import {
  DEFAULT_VERIFY_CONFIG,
  VERIFY_MAIL_PROVIDERS,
  VERIFY_MAIL_PROVIDER_GROUPS,
  ZEUS_X_ACCOUNT_CODES,
  DONGVANFB_ACCOUNT_TYPES,
  STORE1S_PRODUCTS,
  UA_POOLS,
} from '@/types/interaction.types'
import { useMailProviderStock } from '@/composables/useMailProviderStock'
import { useBackendProfiles } from '@/composables/useBackendProfiles'
import { useAutoSave } from '@/composables/useAutoSave'
import { useMarqueeSelect } from '@/composables/useMarqueeSelect'
import FieldHelp from '@/features/settings/components/FieldHelp.vue'
import ProfileManager from '@/features/settings/components/ProfileManager.vue'
import InlineValidation from '@/features/settings/components/InlineValidation.vue'
import SearchableSelect from '@/components/ui/SearchableSelect.vue'

const appStore = useAppStore()
const router = useRouter()

// ─── Verify / interaction form ───────────────────────────────────────────────
const form = ref<VerifyConfig>({ ...DEFAULT_VERIFY_CONFIG })

// ─── Load interaction service khi mount ──────────────────────────────────────
onMounted(async () => {
  try {
    const interactionSvc = await getInteractionService()
    const interactionData = await interactionSvc.load()
    if (interactionData) {
      form.value = { ...DEFAULT_VERIFY_CONFIG, ...interactionData }
      if (!form.value.uaPoolKey) form.value.uaPoolKey = 'android'
      // Migrate: nếu config cũ có tempMailDomain nhưng chưa có tempMailDomains map
      // → seed slot cho provider hiện hành để không mất domain đã cấu hình trước.
      if (!form.value.regPlatformUA) form.value.regPlatformUA = {}
      if (!form.value.verifyPlatformUA) form.value.verifyPlatformUA = {}
      if (!form.value.tempMailDomains) form.value.tempMailDomains = {}
      if (form.value.tempMailDomain && !form.value.tempMailDomains[form.value.mailProvider]) {
        form.value.tempMailDomains[form.value.mailProvider] = form.value.tempMailDomain
      }
      if (!form.value.tempMailTokens) form.value.tempMailTokens = {}
      if (form.value.tempMailToken && !form.value.tempMailTokens[form.value.mailProvider]) {
        form.value.tempMailTokens[form.value.mailProvider] = form.value.tempMailToken
      }
      if (interactionData.outputPath) await validateOutputPath(interactionData.outputPath)
      ensureRegPlatformsForm()
      ensureVerifyPlatformsForm()
    }
    // FORCE: Result folder luôn là default (./result/ cạnh exe) — port C# hardcode path.
    // Bỏ qua value user đã save trong interaction.json để tránh trỏ về path cũ không mong muốn.
    const getDefault = (window as any)?.go?.main?.App?.GetDefaultResultPath
    if (typeof getDefault === 'function') {
      const defaultPath = await getDefault()
      if (defaultPath) form.value.resultFolderPath = defaultPath
    }

    // Cookie Initial: luôn dùng file mặc định cạnh app, không cho chọn file riêng.
    // Backend GetDefaultCookiePaths() auto-tạo file rỗng để user paste datr.
    const getCookiePaths = (window as any)?.go?.main?.App?.GetDefaultCookiePaths
    if (typeof getCookiePaths === 'function') {
      const paths = await getCookiePaths()
      if (paths?.initial) form.value.cookieInitialFile = paths.initial
    }

    // Lead Domain Mail: auto-fill default khi user chưa nhập gì (port C# default).
    // User có thể xoá hoặc thay domain khác — chỉ set 1 lần lúc load form.
    if (!form.value.leadDomainMail || !form.value.leadDomainMail.trim()) {
      form.value.leadDomainMail = '@gmail.com,@yahoo.com'
    }

    // Load proxy lists (TempMail + Gmail) từ Config/Proxy/*.txt.
    const loadProxy = (window as any)?.go?.main?.App?.LoadProxyList
    if (typeof loadProxy === 'function') {
      try {
        proxyTempmailList.value = await loadProxy('tempmail')
        proxyGmailList.value = await loadProxy('gmail')
      } catch { /* ignore */ }
    }

    // Sync VerifySourceFolderPath ↔ General.AccountSourcePath — 2 field là 1 nguồn duy nhất.
    if (!form.value.verifySourceFolderPath || !form.value.verifySourceFolderPath.trim()) {
      const getAccountSource = (window as any)?.go?.main?.App?.GetAccountSourceFolder
      if (typeof getAccountSource === 'function') {
        try {
          const generalPath = await getAccountSource()
          if (generalPath) form.value.verifySourceFolderPath = generalPath
        } catch { /* ignore */ }
      }
    }
  } catch {
    appStore.notify('error', 'Không tải được cấu hình chạy')
  }
  await loadUACounts()
  startCookieInitialStatusPolling()

  // Migration: regThreads vừa chuyển từ GeneralConfig sang InteractionConfig.
  // Nếu user chưa set (=0), copy từ general.threadRequest để không reset về 1.
  if (!form.value.regThreads || form.value.regThreads <= 0) {
    try {
      const svc = await getSettingsService()
      const saved = await svc.load()
      const t = Number(saved?.general?.threadRequest)
      form.value.regThreads = (t && t > 0) ? Math.min(600, Math.floor(t)) : 20
    } catch {
      form.value.regThreads = 20
    }
  }
})


// ─── Proxy TempMail / Gmail lists (lưu vào Config/Proxy/*.txt) ────────────────
// Note: 2 checkbox toggle đã move sang Proxy Settings §4. List content tự load
// để tránh lỗi watch undefined — config list edit cũng ở Proxy Settings tab.
const proxyTempmailList = ref('')
const proxyGmailList = ref('')


// Auto-save proxy list — debounce 1s sau khi ngừng gõ thay vì bấm Lưu thủ công.
const proxySaveStatus = ref('')
let _proxyTempmailSaveTimer: ReturnType<typeof setTimeout> | null = null
let _proxyGmailSaveTimer: ReturnType<typeof setTimeout> | null = null
let _proxyStatusClearTimer: ReturnType<typeof setTimeout> | null = null
let _proxyInitTimer: ReturnType<typeof setTimeout> | null = null
let _cookieInitialStatusTimer: ReturnType<typeof setInterval> | null = null
let _proxySaveInitial = true // skip autosave lần đầu (khi load từ disk)

function scheduleStatusClear() {
  if (_proxyStatusClearTimer) clearTimeout(_proxyStatusClearTimer)
  _proxyStatusClearTimer = setTimeout(() => { proxySaveStatus.value = '' }, 1500)
}

watch(proxyTempmailList, () => {
  if (_proxySaveInitial) return
  proxySaveStatus.value = '• Đang lưu...'
  if (_proxyTempmailSaveTimer) clearTimeout(_proxyTempmailSaveTimer)
  _proxyTempmailSaveTimer = setTimeout(async () => {
    await saveProxyTempmail()
    proxySaveStatus.value = '✓ Đã lưu'
    scheduleStatusClear()
  }, 1000)
})

watch(proxyGmailList, () => {
  if (_proxySaveInitial) return
  proxySaveStatus.value = '• Đang lưu...'
  if (_proxyGmailSaveTimer) clearTimeout(_proxyGmailSaveTimer)
  _proxyGmailSaveTimer = setTimeout(async () => {
    await saveProxyGmail()
    proxySaveStatus.value = '✓ Đã lưu'
    scheduleStatusClear()
  }, 1000)
})

// Đợi 1 tick sau mount để proxyTempmailList/proxyGmailList được load xong từ disk,
// rồi mới enable autosave — tránh save lại data vừa load.
onMounted(() => {
  _proxyInitTimer = setTimeout(() => { _proxySaveInitial = false }, 500)
})

// Cleanup tất cả timer khi unmount — tránh setTimeout fire trên ref đã orphan
// + giải phóng closure khi user rời page (đặc biệt UI auto-reload mỗi 6h).
onBeforeUnmount(() => {
  if (_proxyTempmailSaveTimer) clearTimeout(_proxyTempmailSaveTimer)
  if (_proxyGmailSaveTimer) clearTimeout(_proxyGmailSaveTimer)
  if (_proxyStatusClearTimer) clearTimeout(_proxyStatusClearTimer)
  if (_proxyInitTimer) clearTimeout(_proxyInitTimer)
  if (_cookieInitialStatusTimer) clearInterval(_cookieInitialStatusTimer)
  if (_datrPoolTimer) clearInterval(_datrPoolTimer)
  _proxyTempmailSaveTimer = null
  _proxyGmailSaveTimer = null
  _proxyStatusClearTimer = null
  _proxyInitTimer = null
  _cookieInitialStatusTimer = null
  _datrPoolTimer = null
})
async function saveProxyList(kind: 'tempmail' | 'gmail', content: string) {
  const save = (window as any)?.go?.main?.App?.SaveProxyList
  if (typeof save !== 'function') {
    appStore.notify('error', 'Backend chưa sẵn sàng')
    return
  }
  try {
    const result = await save(kind, content)
    // Autosave silent — không toast success mỗi lần gõ. Chỉ notify error để user biết vấn đề.
    if (result !== 'OK') {
      appStore.notify('error', String(result))
    }
  } catch (err) {
    appStore.notify('error', 'Lỗi lưu: ' + String(err))
  }
}
const saveProxyTempmail = () => saveProxyList('tempmail', proxyTempmailList.value)
const saveProxyGmail = () => saveProxyList('gmail', proxyGmailList.value)

// ─── Summary drawer ───────────────────────────────────────────────────────────
const showSummary = ref(false)

// ─── Section accordion state ──────────────────────────────────────────────────
const sectionCollapsed = reactive({ s1: false, s2: false, s3: false })

// ─── Auth source tab ──────────────────────────────────────────────────────────
const authSourceTab = ref<'mail' | 'phone'>('mail')

// ─── Profiles ────────────────────────────────────────────────────────────────
const {
  profiles, activeProfileId, saveProfile, loadProfile,
  cloneProfile, deleteProfile, renameProfile,
} = useBackendProfiles()

async function reloadForm() {
  try {
    const interactionSvc = await getInteractionService()
    const interactionData = await interactionSvc.load()
    if (interactionData) {
      form.value = { ...DEFAULT_VERIFY_CONFIG, ...interactionData }
      if (!form.value.regPlatformUA) form.value.regPlatformUA = {}
      if (!form.value.verifyPlatformUA) form.value.verifyPlatformUA = {}
      ensureRegPlatformsForm()
      ensureVerifyPlatformsForm()
    }
  } catch { /* ignore on reload */ }
}

async function handleProfileLoad(id: string) {
  await loadProfile(id)
  await reloadForm()
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
    const json = JSON.stringify({ interaction: form.value }, null, 2)
    navigator.clipboard.writeText(json).then(() => appStore.notify('success', 'Đã sao chép cấu hình'))
  } catch { appStore.notify('error', 'Không thể sao chép') }
}

function handleProfileImportFromManager(json: string) {
  try {
    const data = JSON.parse(json)
    if (data.interaction) {
      form.value = { ...DEFAULT_VERIFY_CONFIG, ...data.interaction }
      if (!form.value.regPlatformUA) form.value.regPlatformUA = {}
      if (!form.value.verifyPlatformUA) form.value.verifyPlatformUA = {}
      ensureRegPlatformsForm()
      ensureVerifyPlatformsForm()
    }
    appStore.notify('success', 'Đã import — nhấn Lưu để áp dụng')
  } catch {
    appStore.notify('error', 'Import thất bại — JSON không hợp lệ')
  }
}

// ─── Mail provider stock check ────────────────────────────────────────────────
const {
  zeusXStock, zeusXLoading, zeusXError, selectedZeusXStock, checkZeusXStock,
  mail30sProducts, mail30sLoading, mail30sError, selectedMail30sProduct, checkMail30sStock,
  store1sLoading, store1sError, selectedStore1sStock, checkStore1sStock,
  dvfbStock, dvfbLoading, dvfbError, selectedDvfbStock, checkDvfbStock,
} = useMailProviderStock(form)

// ─── Output path handling ─────────────────────────────────────────────────────
const pathError = ref('')
const pathOk = ref(false)

async function validateOutputPath(path: string) {
  if (!path) { pathError.value = ''; pathOk.value = false; return }
  try {
    const svc = await getFileDialogService()
    const err = await svc.validatePath(path)
    if (err) { pathError.value = err; pathOk.value = false }
    else { pathError.value = ''; pathOk.value = true }
  } catch { pathError.value = ''; pathOk.value = false }
}

async function handleBrowseVerifyFolder() {
  try {
    const svc = await getFileDialogService()
    const path = await svc.openFolder()
    if (path) { form.value.outputPath = path; await validateOutputPath(path) }
  } catch { /* ignore */ }
}

async function handleBrowseVerifySourceFolder() {
  try {
    const svc = await getFileDialogService()
    const path = await svc.openFolder()
    if (!path) return
    form.value.verifySourceFolderPath = path
    // Sync vào Cài đặt chung > Nguồn tài khoản — 2 field này là 1 nguồn duy nhất.
    const setAccountSource = (window as any)?.go?.main?.App?.SetAccountSourceFolder
    if (typeof setAccountSource === 'function') {
      try { await setAccountSource(path) } catch { /* ignore */ }
    }
  } catch { /* ignore */ }
}

async function handleBrowseRegisterFolder() {
  try {
    const svc = await getFileDialogService()
    const path = await svc.openFolder()
    if (path) form.value.createOutputPath = path
  } catch { /* ignore */ }
}


async function openCookieInitialFile() {
  const fn = (window as any)?.go?.main?.App?.OpenCookieInitialFile
  if (typeof fn !== 'function') return
  try {
    const result = await fn('')
    if (result && result !== 'OK') appStore.notify('error', String(result))
    await loadCookieInitialStatus()
  } catch (err) {
    appStore.notify('error', 'Không mở được file datr: ' + String(err))
  }
}

// Thay cho "Browse": nút 📁 giờ mở Explorer vào default folder (giống C# FMain Result button).
// App tự quản lý folder ./result/ nên user không chọn path — chỉ cần mở xem.
async function handleOpenResultFolder() {
  try {
    const svc = await getFileDialogService()
    const path = form.value.resultFolderPath
    if (path) {
      await svc.openFolderInExplorer(path)
    }
  } catch { /* ignore */ }
}

async function handleBrowseAndroidDevicesFile() {
  try {
    const svc = await getFileDialogService()
    const path = await svc.openFilePath()
    if (path) form.value.androidDevicesPath = path
  } catch { /* ignore */ }
}

// ─── Count helpers ────────────────────────────────────────────────────────────
const cookieLines = computed(() => form.value.createCookieList.split('\n').filter(l => l.trim()).length)
const mailCount = computed(() => form.value.mailList.split('\n').filter(l => l.trim()).length)
const cookieInitialFileCount = ref(0)
const cookieInitialFileExists = ref(false)
const cookieInitialFileStatus = ref('')
const cookieInitialResolvedPath = ref('')

async function loadCookieInitialStatus() {
  const fn = (window as any)?.go?.main?.App?.GetCookieInitialStatus
  if (typeof fn !== 'function') return
  try {
    const status = await fn(form.value.cookieInitialFile)
    cookieInitialResolvedPath.value = String(status?.path ?? '')
    cookieInitialFileCount.value = Number(status?.count ?? 0)
    cookieInitialFileExists.value = Boolean(status?.exists)
    cookieInitialFileStatus.value = String(status?.error ?? '')
  } catch (err) {
    cookieInitialFileStatus.value = String(err)
  }
}

const datrPoolCount = ref(0)
const poolFileSaveCount = ref(0)
let _datrPoolTimer: ReturnType<typeof setInterval> | null = null

async function loadDatrPoolCount() {
  const fnPool = (window as any)?.go?.main?.App?.GetDatrPoolSize
  const fnSaved = (window as any)?.go?.main?.App?.GetPoolFileSaveCount
  try {
    if (typeof fnPool === 'function') datrPoolCount.value = Number(await fnPool())
    if (typeof fnSaved === 'function') poolFileSaveCount.value = Number(await fnSaved())
  } catch {
    // ignore — pool may not be initialized
  }
}

function startCookieInitialStatusPolling() {
  if (_cookieInitialStatusTimer) clearInterval(_cookieInitialStatusTimer)
  if (_datrPoolTimer) clearInterval(_datrPoolTimer)
  void loadCookieInitialStatus()
  void loadDatrPoolCount()
  _cookieInitialStatusTimer = setInterval(() => {
    void loadCookieInitialStatus()
  }, 30000)
  _datrPoolTimer = setInterval(() => {
    void loadDatrPoolCount()
  }, 2000)
}

watch(() => form.value.cookieInitialFile, () => {
  void loadCookieInitialStatus()
})

// Active UA pool — show "Đang dùng: ..." hint sau khi user click 1 trong 3 button
const activeUaPoolMeta = computed(() => UA_POOLS.find(p => p.key === form.value.uaPoolKey) ?? null)

// ─── UA file counts ───────────────────────────────────────────────────────────
const uaCounts = ref<Record<string, number>>({ android: 0, iphone: 0, request: 0, webchrome: 0, android_mess: 0, ios_mess: 0 })

async function loadUACounts() {
  const fn = (window as any)?.go?.main?.App?.GetUAPoolsStatus
  if (typeof fn !== 'function') return
  try {
    const statuses: Array<{ kind: string; count: number }> = await fn()
    for (const s of statuses) {
      const key = s.kind === 'ios' ? 'iphone' : s.kind
      uaCounts.value[key] = s.count
    }
  } catch { /* ignore */ }
}

async function openUAFile() {
  const fn = (window as any)?.go?.main?.App?.OpenUAFileInEditor
  if (typeof fn !== 'function') return
  await fn(form.value.uaPoolKey)
}


// ── UA Config per-platform (inline) ─────────────────────────────────────────

// UA gốc cố định theo platform (copy từ internal/facebook/register/s5xx/body.go).
const ORIGINAL_UA_STRINGS: Record<string, string> = {
  // Generic Android FB app (SM-G991B — Galaxy S21)
  android: '[FBAN/FB4A;FBAV/560.0.0.26.63;FBBV/959741200;FBDM/{density=2.625,width=1080,height=2400};FBLC/en_US;FBRV/0;FBCR/;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-G991B;FBSV/14;FBOP/1;FBCA/arm64-v8a:;]',
  // Web Android — mobile Chrome browser
  webandroid: 'Mozilla/5.0 (Linux; Android 15; SM-S931B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/141.0.7390.107 Mobile Safari/537.36',
  // iOS Facebook app (iPhone 14 Pro — iPhone15,2)
  ios: '[FBAN/FBIOS;FBAV/410.0.0.36.108;FBBV/502718897;FBDM/{density=3.0,width=1170,height=2532};FBLC/en_GB;FBSV/17.4;FBOP/5;FBCA/iPhone15,2;]',
  // iOS Native App versions (FBIOS — iPhone 14 Pro, iOS 16.7)
  ios520: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/520.0.0.38.101;FBBV/756351453;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios530: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/530.0.0.59.75;FBBV/790686474;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios540: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/540.0.0.44.68;FBBV/828638047;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios550: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/550.0.0.34.65;FBBV/890804754;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios560: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/555.0.0.36.63;FBBV/923840166;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios555: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/555.0.0.36.63;FBBV/923840166;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios510: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/510.0.0.38.93;FBBV/724276253;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios500: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/500.0.0.52.98;FBBV/696635672;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios490: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/490.1.0.49.107;FBBV/663124541;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios480: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/480.0.0.32.109;FBBV/638556369;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios470: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/470.1.0.43.103;FBBV/617058003;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios460: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/460.0.0.31.103;FBBV/588708950;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios450: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/450.0.0.38.108;FBBV/564431005;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios421: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/421.0.0.24.58;FBBV/489261892;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios422: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/422.0.0.24.58;FBBV/491634260;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios423: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/423.0.0.24.58;FBBV/494006628;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios424: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/424.0.0.24.58;FBBV/496378996;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios425: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/425.0.0.24.58;FBBV/498751364;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios426: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/426.0.0.24.58;FBBV/501123732;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios427: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/427.0.0.24.58;FBBV/503496100;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios428: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/428.0.0.24.58;FBBV/505868468;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios429: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/429.0.0.24.58;FBBV/508240836;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios431: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/431.0.0.33.114;FBBV/513304094;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios432: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/432.0.0.33.114;FBBV/515994984;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios433: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/433.0.0.33.114;FBBV/518685874;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios434: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/434.0.0.33.114;FBBV/521376764;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios435: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/435.0.0.33.114;FBBV/524067654;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios436: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/436.0.0.33.114;FBBV/526758544;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios437: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/437.0.0.33.114;FBBV/529449434;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios438: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/438.0.0.33.114;FBBV/532140324;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios439: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/439.0.0.33.114;FBBV/534831214;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios441: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/441.0.0.38.108;FBBV/540212995;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios442: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/442.0.0.38.108;FBBV/542903885;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios443: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/443.0.0.38.108;FBBV/545594775;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios444: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/444.0.0.38.108;FBBV/548285665;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios445: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/445.0.0.38.108;FBBV/550976555;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios446: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/446.0.0.38.108;FBBV/553667445;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios447: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/447.0.0.38.108;FBBV/556358335;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios448: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/448.0.0.38.108;FBBV/559049225;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios449: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/449.0.0.38.108;FBBV/561740115;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios451: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/451.0.0.38.108;FBBV/566858800;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios452: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/452.0.0.38.108;FBBV/569286594;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios453: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/453.0.0.38.108;FBBV/571714389;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios454: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/454.0.0.38.108;FBBV/574142183;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios455: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/455.0.0.38.108;FBBV/576569978;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios456: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/456.0.0.38.108;FBBV/578997772;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios457: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/457.0.0.38.108;FBBV/581425567;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios458: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/458.0.0.38.108;FBBV/583853361;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios459: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/459.0.0.38.108;FBBV/586281156;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios461: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/461.0.0.31.103;FBBV/591543855;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios462: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/462.0.0.31.103;FBBV/594378761;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios463: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/463.0.0.31.103;FBBV/597213666;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios464: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/464.0.0.31.103;FBBV/600048571;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios465: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/465.0.0.31.103;FBBV/602883477;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios466: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/466.0.0.31.103;FBBV/605718382;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios467: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/467.0.0.31.103;FBBV/608553287;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios468: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/468.0.0.31.103;FBBV/611388192;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios469: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/469.0.0.31.103;FBBV/614223098;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios471: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/471.1.0.43.103;FBBV/619207840;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios472: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/472.1.0.43.103;FBBV/621357676;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios473: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/473.1.0.43.103;FBBV/623507513;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios474: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/474.1.0.43.103;FBBV/625657349;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios475: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/475.1.0.43.103;FBBV/627807186;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios476: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/476.1.0.43.103;FBBV/629957023;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios477: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/477.1.0.43.103;FBBV/632106859;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios478: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/478.1.0.43.103;FBBV/634256696;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios479: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/479.1.0.43.103;FBBV/636406532;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios481: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/481.0.0.32.109;FBBV/641013186;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios482: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/482.0.0.32.109;FBBV/643470003;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios483: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/483.0.0.32.109;FBBV/645926821;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios484: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/484.0.0.32.109;FBBV/648383638;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios485: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/485.0.0.32.109;FBBV/650840455;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios486: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/486.0.0.32.109;FBBV/653297272;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios487: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/487.0.0.32.109;FBBV/655754089;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios488: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/488.0.0.32.109;FBBV/658210907;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios489: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/489.0.0.32.109;FBBV/660667724;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios491: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/491.1.0.49.107;FBBV/666475654;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios492: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/492.1.0.49.107;FBBV/669826767;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios493: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/493.1.0.49.107;FBBV/673177880;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios494: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/494.1.0.49.107;FBBV/676528993;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios495: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/495.1.0.49.107;FBBV/679880107;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios496: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/496.1.0.49.107;FBBV/683231220;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios497: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/497.1.0.49.107;FBBV/686582333;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios498: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/498.1.0.49.107;FBBV/689933446;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios499: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/499.1.0.49.107;FBBV/693284559;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios501: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/501.0.0.52.98;FBBV/699399730;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios502: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/502.0.0.52.98;FBBV/702163788;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios503: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/503.0.0.52.98;FBBV/704927846;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios504: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/504.0.0.52.98;FBBV/707691904;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios505: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/505.0.0.52.98;FBBV/710455963;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios506: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/506.0.0.52.98;FBBV/713220021;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios507: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/507.0.0.52.98;FBBV/715984079;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios508: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/508.0.0.52.98;FBBV/718748137;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios509: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/509.0.0.52.98;FBBV/721512195;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios511: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/511.0.0.38.93;FBBV/727483773;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios512: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/512.0.0.38.93;FBBV/730691293;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios513: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/513.0.0.38.93;FBBV/733898813;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios514: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/514.0.0.38.93;FBBV/737106333;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios515: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/515.0.0.38.93;FBBV/740313853;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios516: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/516.0.0.38.93;FBBV/743521373;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios517: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/517.0.0.38.93;FBBV/746728893;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios518: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/518.0.0.38.93;FBBV/749936413;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios519: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/519.0.0.38.93;FBBV/753143933;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios521: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/521.0.0.38.101;FBBV/759784955;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios522: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/522.0.0.38.101;FBBV/763218457;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios523: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/523.0.0.38.101;FBBV/766651959;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios524: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/524.0.0.38.101;FBBV/770085461;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios525: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/525.0.0.38.101;FBBV/773518964;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios526: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/526.0.0.38.101;FBBV/776952466;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios527: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/527.0.0.38.101;FBBV/780385968;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios528: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/528.0.0.38.101;FBBV/783819470;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios529: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/529.0.0.38.101;FBBV/787252972;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios531: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/531.0.0.59.75;FBBV/794481631;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios532: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/532.0.0.59.75;FBBV/798276789;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios533: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/533.0.0.59.75;FBBV/802071946;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios534: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/534.0.0.59.75;FBBV/805867103;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios535: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/535.0.0.59.75;FBBV/809662261;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios536: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/536.0.0.59.75;FBBV/813457418;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios537: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/537.0.0.59.75;FBBV/817252575;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios538: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/538.0.0.59.75;FBBV/821047732;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios539: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/539.0.0.59.75;FBBV/824842890;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios541: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/541.0.0.44.68;FBBV/834854718;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios542: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/542.0.0.44.68;FBBV/841071388;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios543: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/543.0.0.44.68;FBBV/847288059;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios544: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/544.0.0.44.68;FBBV/853504730;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios545: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/545.0.0.44.68;FBBV/859721401;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios546: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/546.0.0.44.68;FBBV/865938071;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios547: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/547.0.0.44.68;FBBV/872154742;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios548: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/548.0.0.44.68;FBBV/878371413;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios549: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/549.0.0.44.68;FBBV/884588083;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios551: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/551.0.0.34.65;FBBV/897411836;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios552: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/552.0.0.34.65;FBBV/904018919;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios553: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/553.0.0.34.65;FBBV/910626001;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios554: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/554.0.0.34.65;FBBV/917233084;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios556: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/556.0.0.36.63;FBBV/930684417;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios557: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/557.0.0.36.63;FBBV/937528668;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios558: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/558.0.0.36.63;FBBV/944372920;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios559: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/559.0.0.36.63;FBBV/951217171;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios561: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/561.0.0.36.63;FBBV/964905673;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios440: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/450.0.0.38.108;FBBV/564431005;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios430: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/430.0.0.33.114;FBBV/510613204;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios420: 'Mozilla/5.0 (iPhone; CPU iPhone OS 15_8_4 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/19H390 [FBAN/FBIOS;FBAV/420.0.0.24.58;FBBV/486889524;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/15.8.4;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios564: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/564.0.0.57.71;FBBV/985438427;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  ios562: 'Mozilla/5.0 (iPhone; CPU iPhone OS 16_7 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/20H115 [FBAN/FBIOS;FBAV/563.0.0.67.72;FBBV/980285082;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBID/phone;FBLC/en_US;FBOP/5;FBRV/0]',
  // Samsung Galaxy S22 (SM-S901B)
  s22: '[FBAN/FB4A;FBAV/560.0.0.26.63;FBBV/959741200;FBDM/{density=2.625,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S901B;FBSV/13;FBOP/1;FBCA/arm64-v8a:;]',
  // Samsung Galaxy S23 (SM-S911B) — latest FB build
  s23: '[FBAN/FB4A;FBAV/560.0.0.26.63;FBBV/959741200;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  // Samsung Galaxy S24 (SM-S921B)
  s24: '[FBAN/FB4A;FBAV/560.0.0.26.63;FBBV/959741200;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S921B;FBSV/14;FBOP/1;FBCA/arm64-v8a:;]',
  // Samsung Galaxy S25 (SM-S931B)
  s25: '[FBAN/FB4A;FBAV/560.0.0.26.63;FBBV/959741200;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S931B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  // Samsung Galaxy S26 (SM-S941B)
  s26: '[FBAN/FB4A;FBAV/560.0.0.26.63;FBBV/959741200;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S941B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  // Versioned S23 builds (FBAV pinned)
  s415: '[FBAN/FB4A;FBAV/415.0.0.34.107;FBBV/475722615;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s425: '[FBAN/FB4A;FBAV/425.0.0.22.49;FBBV/498189258;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s435: '[FBAN/FB4A;FBAV/435.0.0.42.112;FBBV/523162189;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s445: '[FBAN/FB4A;FBAV/445.0.0.34.118;FBBV/548452792;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s446: '[FBAN/FB4A;FBAV/446.0.0.0.46;FBBV/551427493;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s447: '[FBAN/FB4A;FBAV/447.0.0.0.47;FBBV/554402194;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s448: '[FBAN/FB4A;FBAV/448.0.0.0.48;FBBV/557376894;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s449: '[FBAN/FB4A;FBAV/449.0.0.0.49;FBBV/560351595;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s450: '[FBAN/FB4A;FBAV/450.0.0.0.50;FBBV/563326296;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s451: '[FBAN/FB4A;FBAV/451.0.0.0.51;FBBV/566300997;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s452: '[FBAN/FB4A;FBAV/452.0.0.0.52;FBBV/569275698;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s453: '[FBAN/FB4A;FBAV/453.0.0.0.53;FBBV/572250398;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s454: '[FBAN/FB4A;FBAV/454.0.0.0.54;FBBV/575225099;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s455: '[FBAN/FB4A;FBAV/455.0.0.44.88;FBBV/578199800;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s456: '[FBAN/FB4A;FBAV/456.0.0.0.56;FBBV/580927357;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s457: '[FBAN/FB4A;FBAV/457.0.0.0.57;FBBV/583654914;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s458: '[FBAN/FB4A;FBAV/458.0.0.0.58;FBBV/586382471;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s459: '[FBAN/FB4A;FBAV/459.0.0.0.59;FBBV/589110028;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s460: '[FBAN/FB4A;FBAV/460.0.0.0.60;FBBV/591837585;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s461: '[FBAN/FB4A;FBAV/461.0.0.0.61;FBBV/594565142;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s462: '[FBAN/FB4A;FBAV/462.0.0.0.62;FBBV/597292699;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s463: '[FBAN/FB4A;FBAV/463.0.0.0.63;FBBV/600020256;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s464: '[FBAN/FB4A;FBAV/464.0.0.0.64;FBBV/602747813;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s465: '[FBAN/FB4A;FBAV/465.0.0.63.83;FBBV/605475370;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s466: '[FBAN/FB4A;FBAV/466.0.0.0.66;FBBV/607953641;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s467: '[FBAN/FB4A;FBAV/467.0.0.0.67;FBBV/610431912;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s468: '[FBAN/FB4A;FBAV/468.0.0.0.68;FBBV/612910183;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s469: '[FBAN/FB4A;FBAV/469.0.0.0.69;FBBV/615388454;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s470: '[FBAN/FB4A;FBAV/470.0.0.0.70;FBBV/617866725;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s471: '[FBAN/FB4A;FBAV/471.0.0.0.71;FBBV/620344996;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s472: '[FBAN/FB4A;FBAV/472.0.0.0.72;FBBV/622823267;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s473: '[FBAN/FB4A;FBAV/473.0.0.0.73;FBBV/625301538;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s474: '[FBAN/FB4A;FBAV/474.0.0.0.74;FBBV/627779809;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s475: '[FBAN/FB4A;FBAV/475.1.0.46.82;FBBV/630258084;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s476: '[FBAN/FB4A;FBAV/476.0.0.0.76;FBBV/632538884;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s477: '[FBAN/FB4A;FBAV/477.0.0.0.77;FBBV/634819684;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s478: '[FBAN/FB4A;FBAV/478.0.0.0.78;FBBV/637100484;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s479: '[FBAN/FB4A;FBAV/479.0.0.0.79;FBBV/639381284;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s480: '[FBAN/FB4A;FBAV/480.0.0.0.80;FBBV/641662084;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s481: '[FBAN/FB4A;FBAV/481.0.0.0.81;FBBV/643942884;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s482: '[FBAN/FB4A;FBAV/482.0.0.0.82;FBBV/646223684;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s483: '[FBAN/FB4A;FBAV/483.0.0.0.83;FBBV/648504484;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s484: '[FBAN/FB4A;FBAV/484.0.0.0.84;FBBV/650785284;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s485: '[FBAN/FB4A;FBAV/485.0.0.70.77;FBBV/653066074;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s486: '[FBAN/FB4A;FBAV/486.0.0.0.86;FBBV/656211818;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s487: '[FBAN/FB4A;FBAV/487.0.0.0.87;FBBV/659357562;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s488: '[FBAN/FB4A;FBAV/488.0.0.0.88;FBBV/662503306;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s489: '[FBAN/FB4A;FBAV/489.0.0.0.89;FBBV/665649050;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s490: '[FBAN/FB4A;FBAV/490.0.0.0.90;FBBV/668794794;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s491: '[FBAN/FB4A;FBAV/491.0.0.0.91;FBBV/671940538;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s492: '[FBAN/FB4A;FBAV/492.0.0.0.92;FBBV/675086282;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s493: '[FBAN/FB4A;FBAV/493.0.0.0.93;FBBV/678232026;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s494: '[FBAN/FB4A;FBAV/494.0.0.0.94;FBBV/681377770;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s495: '[FBAN/FB4A;FBAV/495.0.0.45.201;FBBV/684523515;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s496: '[FBAN/FB4A;FBAV/496.0.0.0.96;FBBV/687863501;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s497: '[FBAN/FB4A;FBAV/497.0.0.0.97;FBBV/691203487;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s498: '[FBAN/FB4A;FBAV/498.0.0.0.98;FBBV/694543473;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s499: '[FBAN/FB4A;FBAV/499.0.0.0.99;FBBV/697883459;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s416: '[FBAN/FB4A;FBAV/416.0.0.0.16;FBBV/477969279;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s417: '[FBAN/FB4A;FBAV/417.0.0.0.17;FBBV/480215944;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s418: '[FBAN/FB4A;FBAV/418.0.0.0.18;FBBV/482462608;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s419: '[FBAN/FB4A;FBAV/419.0.0.0.19;FBBV/484709272;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s420: '[FBAN/FB4A;FBAV/420.0.0.0.20;FBBV/486955936;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s421: '[FBAN/FB4A;FBAV/421.0.0.0.21;FBBV/489202601;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s422: '[FBAN/FB4A;FBAV/422.0.0.0.22;FBBV/491449265;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s423: '[FBAN/FB4A;FBAV/423.0.0.0.23;FBBV/493695929;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s424: '[FBAN/FB4A;FBAV/424.0.0.0.24;FBBV/495942594;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s426: '[FBAN/FB4A;FBAV/426.0.0.0.26;FBBV/500686551;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s427: '[FBAN/FB4A;FBAV/427.0.0.0.27;FBBV/503183844;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s428: '[FBAN/FB4A;FBAV/428.0.0.0.28;FBBV/505681137;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s429: '[FBAN/FB4A;FBAV/429.0.0.0.29;FBBV/508178430;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s430: '[FBAN/FB4A;FBAV/430.0.0.0.30;FBBV/510675724;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s431: '[FBAN/FB4A;FBAV/431.0.0.0.31;FBBV/513173017;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s432: '[FBAN/FB4A;FBAV/432.0.0.0.32;FBBV/515670310;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s433: '[FBAN/FB4A;FBAV/433.0.0.0.33;FBBV/518167603;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s434: '[FBAN/FB4A;FBAV/434.0.0.0.34;FBBV/520664896;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s436: '[FBAN/FB4A;FBAV/436.0.0.0.36;FBBV/525691249;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s437: '[FBAN/FB4A;FBAV/437.0.0.0.37;FBBV/528220310;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s438: '[FBAN/FB4A;FBAV/438.0.0.0.38;FBBV/530749370;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s439: '[FBAN/FB4A;FBAV/439.0.0.0.39;FBBV/533278430;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s440: '[FBAN/FB4A;FBAV/440.0.0.0.40;FBBV/535807490;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s441: '[FBAN/FB4A;FBAV/441.0.0.0.41;FBBV/538336551;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s442: '[FBAN/FB4A;FBAV/442.0.0.0.42;FBBV/540865611;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s443: '[FBAN/FB4A;FBAV/443.0.0.0.43;FBBV/543394671;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s444: '[FBAN/FB4A;FBAV/444.0.0.0.44;FBBV/545923732;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s500: '[FBAN/FB4A;FBAV/500.0.0.41.110;FBBV/701223445;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s501: '[FBAN/FB4A;FBAV/501.0.0.39.109;FBBV/705334221;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s502: '[FBAN/FB4A;FBAV/502.0.0.44.108;FBBV/709552334;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s503: '[FBAN/FB4A;FBAV/503.0.0.46.107;FBBV/713889221;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s504: '[FBAN/FB4A;FBAV/504.0.0.48.106;FBBV/718223990;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s505: '[FBAN/FB4A;FBAV/505.0.0.50.105;FBBV/722889334;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s506: '[FBAN/FB4A;FBAV/506.0.0.52.104;FBBV/727114556;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s507: '[FBAN/FB4A;FBAV/507.0.0.54.103;FBBV/731998112;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s508: '[FBAN/FB4A;FBAV/508.0.0.56.102;FBBV/736445778;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s509: '[FBAN/FB4A;FBAV/509.0.0.58.101;FBBV/740998334;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s510: '[FBAN/FB4A;FBAV/510.0.0.60.100;FBBV/745334998;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s511: '[FBAN/FB4A;FBAV/511.0.0.61.99;FBBV/749998221;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s512: '[FBAN/FB4A;FBAV/512.0.0.62.98;FBBV/754334667;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s513: '[FBAN/FB4A;FBAV/513.0.0.63.97;FBBV/758889112;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s514: '[FBAN/FB4A;FBAV/514.0.0.64.96;FBBV/763334556;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s515: '[FBAN/FB4A;FBAV/515.0.0.65.95;FBBV/767889998;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s516: '[FBAN/FB4A;FBAV/516.0.0.66.94;FBBV/772445334;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s517: '[FBAN/FB4A;FBAV/517.0.0.67.93;FBBV/776998112;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s518: '[FBAN/FB4A;FBAV/518.0.0.63.86;FBBV/750617326;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s519: '[FBAN/FB4A;FBAV/519.0.0.68.92;FBBV/781552998;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s520: '[FBAN/FB4A;FBAV/520.0.0.69.91;FBBV/786115334;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s521: '[FBAN/FB4A;FBAV/521.0.0.70.90;FBBV/790667889;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s522: '[FBAN/FB4A;FBAV/522.0.0.71.89;FBBV/795334112;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s523: '[FBAN/FB4A;FBAV/523.0.0.72.88;FBBV/799889556;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s524: '[FBAN/FB4A;FBAV/524.0.0.73.87;FBBV/804445998;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s525: '[FBAN/FB4A;FBAV/525.0.0.53.51;FBBV/773514916;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s526: '[FBAN/FB4A;FBAV/526.0.0.74.86;FBBV/808998334;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s527: '[FBAN/FB4A;FBAV/527.0.0.75.85;FBBV/813556778;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s528: '[FBAN/FB4A;FBAV/528.0.0.76.84;FBBV/818114221;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s529: '[FBAN/FB4A;FBAV/529.0.0.77.83;FBBV/822667889;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s530: '[FBAN/FB4A;FBAV/530.0.0.48.74;FBBV/465017152;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s531: '[FBAN/FB4A;FBAV/531.0.0.47.70;FBBV/792873355;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s532: '[FBAN/FB4A;FBAV/532.0.0.44.92;FBBV/798341220;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s533: '[FBAN/FB4A;FBAV/533.0.0.50.84;FBBV/801500000;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s534: '[FBAN/FB4A;FBAV/534.0.0.56.76;FBBV/804773118;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s535: '[FBAN/FB4A;FBAV/535.0.0.49.72;FBBV/808115902;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s536: '[FBAN/FB4A;FBAV/536.0.0.46.77;FBBV/812334776;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s537: '[FBAN/FB4A;FBAV/537.0.0.47.77;FBBV/816559441;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s538: '[FBAN/FB4A;FBAV/538.0.0.53.70;FBBV/824992114;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s539: '[FBAN/FB4A;FBAV/539.0.0.54.69;FBBV/825520343;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s540: '[FBAN/FB4A;FBAV/540.0.0.50.82;FBBV/834118774;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s541: '[FBAN/FB4A;FBAV/541.0.0.85.79;FBBV/840338122;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s542: '[FBAN/FB4A;FBAV/542.0.0.60.83;FBBV/839882771;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s543: '[FBAN/FB4A;FBAV/543.0.0.55.73;FBBV/846638228;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s544: '[FBAN/FB4A;FBAV/544.0.0.60.112;FBBV/849103284;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s545: '[FBAN/FB4A;FBAV/545.0.0.43.63;FBBV/871838024;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s546: '[FBAN/FB4A;FBAV/546.0.0.42.66;FBBV/876010203;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s547: '[FBAN/FB4A;FBAV/547.0.0.41.68;FBBV/882675876;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s548: '[FBAN/FB4A;FBAV/548.1.0.51.64;FBBV/891619493;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s549: '[FBAN/FB4A;FBAV/549.0.0.61.62;FBBV/891620534;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s550: '[FBAN/FB4A;FBAV/550.0.0.40.60;FBBV/900039717;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s550v2: '[FBAN/FB4A;FBAV/550.0.0.40.60;FBBV/900039717;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s551: '[FBAN/FB4A;FBAV/551.1.0.58.63;FBBV/906186219;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s551v2: '[FBAN/FB4A;FBAV/551.1.0.58.63;FBBV/906186219;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s552: '[FBAN/FB4A;FBAV/552.1.0.45.68;FBBV/911260592;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s552v2: '[FBAN/FB4A;FBAV/552.1.0.45.68;FBBV/911260592;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s553: '[FBAN/FB4A;FBAV/553.0.0.56.58;FBBV/918989583;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s553v2: '[FBAN/FB4A;FBAV/553.0.0.56.58;FBBV/918989583;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s554: '[FBAN/FB4A;FBAV/554.0.0.57.70;FBBV/926292396;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s554v2: '[FBAN/FB4A;FBAV/554.0.0.57.70;FBBV/926292396;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s555: '[FBAN/FB4A;FBAV/555.0.0.49.59;FBBV/926292570;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s555v2: '[FBAN/FB4A;FBAV/555.0.0.49.59;FBBV/926293010;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s556: '[FBAN/FB4A;FBAV/556.1.0.63.64;FBBV/942217461;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s556v2: '[FBAN/FB4A;FBAV/556.1.0.63.64;FBBV/942217461;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s557: '[FBAN/FB4A;FBAV/557.0.0.59.72;FBBV/942218792;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s557v2: '[FBAN/FB4A;FBAV/557.0.0.59.72;FBBV/953308969;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s558: '[FBAN/FB4A;FBAV/558.0.0.70.72;FBBV/953309385;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s558v2: '[FBAN/FB4A;FBAV/558.0.0.70.72;FBBV/959738023;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s559: '[FBAN/FB4A;FBAV/559.1.0.52.72;FBBV/959738728;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s559v2: '[FBAN/FB4A;FBAV/559.1.0.52.72;FBBV/959738221;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s560: '[FBAN/FB4A;FBAV/560.0.0.26.63;FBBV/959741200;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s560v2: '[FBAN/FB4A;FBAV/560.0.0.26.63;FBBV/959741200;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s560v3: '[FBAN/FB4A;FBAV/560.0.0.57.63;FBBV/963497253;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s561: '[FBAN/FB4A;FBAV/561.0.0.3.67;FBBV/964730465;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s561v2: '[FBAN/FB4A;FBAV/561.0.0.42.67;FBBV/968460367;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s561v3: '[FBAN/FB4A;FBAV/561.0.0.42.67;FBBV/968460367;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s561v99: '[FBAN/FB4A;FBAV/561.0.0.42.67;FBBV/968460367;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s562: '[FBAN/FB4A;FBAV/562.0.0.0.17;FBBV/966019818;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s562v3: '[FBAN/FB4A;FBAV/562.0.0.0.28;FBBV/966787663;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s563: '[FBAN/FB4A;FBAV/563.0.0.0.14;FBBV/972124634;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s563v2: '[FBAN/FB4A;FBAV/563.0.0.0.26;FBBV/972941018;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s563s21: '[FBAN/FB4A;FBAV/563.0.0.0.26;FBBV/972941018;FBDM/{density=2.8125,width=1080,height=2400};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-G996B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s563v3s21: '[FBAN/FB4A;FBAV/563.0.0.0.48;FBBV/974373688;FBDM/{density=2.8125,width=1080,height=2400};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-G996B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s563v4s21: '[FBAN/FB4A;FBAV/563.0.0.23.73;FBBV/978036554;FBDM/{density=2.8125,width=1080,height=2400};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-G996B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s564v1s21: '[FBAN/FB4A;FBAV/564.0.0.0.17;FBBV/977893103;FBDM/{density=2.8125,width=1080,height=2400};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-G996B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s563v4s23: '[FBAN/FB4A;FBAV/563.0.0.23.73;FBBV/978036554;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s564v1s23: '[FBAN/FB4A;FBAV/564.0.0.0.17;FBBV/977893103;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s563v5s21: '[FBAN/FB4A;FBAV/563.0.0.23.73;FBBV/980389559;FBDM/{density=2.8125,width=1080,height=2400};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-G996B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s563v5s23: '[FBAN/FB4A;FBAV/563.0.0.23.73;FBBV/980389559;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s563v6s21: '[FBAN/FB4A;FBAV/563.1.0.50.73;FBBV/986611012;FBDM/{density=2.8125,width=1080,height=2400};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-G996B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s563v6s23: '[FBAN/FB4A;FBAV/563.1.0.50.73;FBBV/986611012;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s564v2s21: '[FBAN/FB4A;FBAV/564.0.0.0.61;FBBV/980390555;FBDM/{density=2.8125,width=1080,height=2400};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-G996B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s564v2s23: '[FBAN/FB4A;FBAV/564.0.0.0.61;FBBV/980390555;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s564v3s21: '[FBAN/FB4A;FBAV/564.0.0.48.74;FBBV/986612294;FBDM/{density=2.8125,width=1080,height=2400};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-G996B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s564v3s23: '[FBAN/FB4A;FBAV/564.0.0.48.74;FBBV/986612294;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s565s21: '[FBAN/FB4A;FBAV/565.0.0.0.28;FBBV/984080529;FBDM/{density=2.8125,width=1080,height=2400};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-G996B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s565s23: '[FBAN/FB4A;FBAV/565.0.0.0.28;FBBV/984080529;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s565v2s21: '[FBAN/FB4A;FBAV/565.0.0.0.58;FBBV/986097483;FBDM/{density=2.8125,width=1080,height=2400};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-G996B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s565v2s23: '[FBAN/FB4A;FBAV/565.0.0.0.58;FBBV/986097483;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s561v4s21: '[FBAN/FB4A;FBAV/561.0.0.42.67;FBBV/976056141;FBDM/{density=2.8125,width=1080,height=2400};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-G996B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s561v4s23: '[FBAN/FB4A;FBAV/561.0.0.42.67;FBBV/976056141;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s562v4s21: '[FBAN/FB4A;FBAV/562.0.0.51.73;FBBV/976057955;FBDM/{density=2.8125,width=1080,height=2400};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-G996B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s562v4s23: '[FBAN/FB4A;FBAV/562.0.0.51.73;FBBV/976057955;FBDM/{density=3.0,width=1080,height=2340};FBLC/en_GB;FBRV/0;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBPN/com.facebook.katana;FBDV/SM-S911B;FBSV/15;FBOP/1;FBCA/arm64-v8a:;]',
  s399: 'Dalvik/2.1.0 (Linux; U; Android 15; SM-S911B Build/AP3A.240905.015.A2) [FBAN/FB4A;FBAV/399.0.0.24.93;FBPN/com.facebook.katana;FBLC/en_GB;FBBV/440587081;FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBDV/SM-S911B;FBSV/15;FBLC/en_GB;FBOP/1;FBCA/arm64-v8a:armeabi-v7a;]',
  s273: 'Dalvik/2.1.0 (Linux; U; Android 9; V2242A Build/PQ3A.190705.05150936) [FBAN/FB4A;FBAV/273.0.0.39.123;FBPN/com.facebook.katana;FBLC/vi_VN;FBBV/218047977;FBCR/MobiFone;FBMF/vivo;FBBD/vivo;FBDV/V2242A;FBSV/9;FBCA/x86:armeabi-v7a;FBDM/{density=1.5,width=900,height=1600};FB_FW/1;FBRV/0;]',
}

const REG_PLATFORM_LABELS: Record<string, string> = {
  android: 'Android API', webandroid: 'Web Android', ios: 'iOS HTTP',
  s22: 'Samsung S22', s23: 'Samsung S23', s24: 'Samsung S24',
  s25: 'Samsung S25', s26: 'Samsung S26',
  s415: 'S23 (Fb_415)', s425: 'S23 (Fb_425)', s435: 'S23 (Fb_435)', s445: 'S23 (Fb_445)',
  s446: 'S23 (Fb_446)', s447: 'S23 (Fb_447)', s448: 'S23 (Fb_448)', s449: 'S23 (Fb_449)', s450: 'S23 (Fb_450)',
  s451: 'S23 (Fb_451)', s452: 'S23 (Fb_452)', s453: 'S23 (Fb_453)', s454: 'S23 (Fb_454)', s455: 'S23 (Fb_455)',
  s456: 'S23 (Fb_456)', s457: 'S23 (Fb_457)', s458: 'S23 (Fb_458)', s459: 'S23 (Fb_459)', s460: 'S23 (Fb_460)',
  s461: 'S23 (Fb_461)', s462: 'S23 (Fb_462)', s463: 'S23 (Fb_463)', s464: 'S23 (Fb_464)', s465: 'S23 (Fb_465)',
  s466: 'S23 (Fb_466)', s467: 'S23 (Fb_467)', s468: 'S23 (Fb_468)', s469: 'S23 (Fb_469)', s470: 'S23 (Fb_470)',
  s471: 'S23 (Fb_471)', s472: 'S23 (Fb_472)', s473: 'S23 (Fb_473)', s474: 'S23 (Fb_474)', s475: 'S23 (Fb_475)',
  s476: 'S23 (Fb_476)', s477: 'S23 (Fb_477)', s478: 'S23 (Fb_478)', s479: 'S23 (Fb_479)', s480: 'S23 (Fb_480)',
  s481: 'S23 (Fb_481)', s482: 'S23 (Fb_482)', s483: 'S23 (Fb_483)', s484: 'S23 (Fb_484)', s485: 'S23 (Fb_485)',
  s486: 'S23 (Fb_486)', s487: 'S23 (Fb_487)', s488: 'S23 (Fb_488)', s489: 'S23 (Fb_489)', s490: 'S23 (Fb_490)',
  s491: 'S23 (Fb_491)', s492: 'S23 (Fb_492)', s493: 'S23 (Fb_493)', s494: 'S23 (Fb_494)', s495: 'S23 (Fb_495)',
  s496: 'S23 (Fb_496)', s497: 'S23 (Fb_497)', s498: 'S23 (Fb_498)', s499: 'S23 (Fb_499)',
  s416: 'S23 (Fb_416)', s417: 'S23 (Fb_417)', s418: 'S23 (Fb_418)', s419: 'S23 (Fb_419)', s420: 'S23 (Fb_420)',
  s421: 'S23 (Fb_421)', s422: 'S23 (Fb_422)', s423: 'S23 (Fb_423)', s424: 'S23 (Fb_424)', s426: 'S23 (Fb_426)',
  s427: 'S23 (Fb_427)', s428: 'S23 (Fb_428)', s429: 'S23 (Fb_429)', s430: 'S23 (Fb_430)', s431: 'S23 (Fb_431)',
  s432: 'S23 (Fb_432)', s433: 'S23 (Fb_433)', s434: 'S23 (Fb_434)', s436: 'S23 (Fb_436)', s437: 'S23 (Fb_437)',
  s438: 'S23 (Fb_438)', s439: 'S23 (Fb_439)', s440: 'S23 (Fb_440)', s441: 'S23 (Fb_441)', s442: 'S23 (Fb_442)',
  s443: 'S23 (Fb_443)', s444: 'S23 (Fb_444)',
  s500: 'S23 (Fb_500)',
  s501: 'S23 (Fb_501)',
  s502: 'S23 (Fb_502)',
  s503: 'S23 (Fb_503)',
  s504: 'S23 (Fb_504)',
  s505: 'S23 (Fb_505)',
  s506: 'S23 (Fb_506)',
  s507: 'S23 (Fb_507)',
  s508: 'S23 (Fb_508)',
  s509: 'S23 (Fb_509)',
  s510: 'S23 (Fb_510)',
  s511: 'S23 (Fb_511)',
  s512: 'S23 (Fb_512)',
  s513: 'S23 (Fb_513)',
  s514: 'S23 (Fb_514)',
  s515: 'S23 (Fb_515)',
  s516: 'S23 (Fb_516)',
  s517: 'S23 (Fb_517)',
  s518: 'S23 (Fb_518)',
  s519: 'S23 (Fb_519)',
  s520: 'S23 (Fb_520)',
  s521: 'S23 (Fb_521)',
  s522: 'S23 (Fb_522)',
  s523: 'S23 (Fb_523)',
  s524: 'S23 (Fb_524)',
  s525: 'S23 (Fb_525)',
  s526: 'S23 (Fb_526)',
  s527: 'S23 (Fb_527)',
  s528: 'S23 (Fb_528)',
  s529: 'S23 (Fb_529)',
  s530: 'S23 (Fb_530)',
  s531: 'S23 (Fb_531)',
  s532: 'S23 (Fb_532)',
  s533: 'S23 (Fb_533)',
  s534: 'S23 (Fb_534)',
  s535: 'S23 (Fb_535)',
  s536: 'S23 (Fb_536)',
  s537: 'S23 (Fb_537)',
  s538: 'S23 (Fb_538)',
  s539: 'S23 (Fb_539)',
  s540: 'S23 (Fb_540)',
  s541: 'S23 (Fb_541)',
  s542: 'S23 (Fb_542)',
  s543: 'S23 (Fb_543)',
  s544: 'S23 (Fb_544)',
  s545: 'S23 (Fb_545)', s546: 'S23 (Fb_546)', s547: 'S23 (Fb_547)', s548: 'S23 (Fb_548)', s549: 'S23 (Fb_549)',
  s550: 'S23 (Fb_550)', s551: 'S23 (Fb_551)', s552: 'S23 (Fb_552)', s553: 'S23 (Fb_553)', s554: 'S23 (Fb_554)',
  s555: 'S23 (Fb_555)', s555v2: 'S23 (Fb_555v2)', s556: 'S23 (Fb_556)', s557: 'S23 (Fb_557)',
  s550v2: 'S23 (Fb_550v2)', s551v2: 'S23 (Fb_551v2)', s552v2: 'S23 (Fb_552v2)', s553v2: 'S23 (Fb_553v2)', s554v2: 'S23 (Fb_554v2)',
  s556v2: 'S23 (Fb_556v2)', s557v2: 'S23 (Fb_557v2)',
  s558: 'S23 (Fb_558)', s558v2: 'S23 (Fb_558v2)', s559: 'S23 (Fb_559)', s559v2: 'S23 (Fb_559v2)', s560: 'S23 (Fb_560)', s560v2: 'S23 (Fb_560v2)', s560v3: 'S23 (Fb_560v3)', s561: 'S23 (Fb_561)', s561v2: 'S23 (Fb_561v2)', s561v3: 'S23 (Fb_561v3)', s561v99: 'S23 (Fb_561v99)', s561v4s21: 'S21+ (Fb_561v4)', s561v4s23: 'S23 (Fb_561v4)', s562: 'S23 (Fb_562)', s562v3: 'S23 (Fb_562v3)', s562v4s21: 'S21+ (Fb_562v4)', s562v4s23: 'S23 (Fb_562v4)', s563: 'S23 (Fb_563)', s563s21: 'S21+ (Fb_563)', s563v3s21: 'S21+ (Fb_563v3)', s563v4s21: 'S21+ (Fb_563v4)', s563v4s23: 'S23 (Fb_563v4)', s564v1s21: 'S21+ (Fb_564v1)', s564v1s23: 'S23 (Fb_564v1)', s563v5s21: 'S21+ (Fb_563v5)', s563v5s23: 'S23 (Fb_563v5)', s563v6s21: 'S21+ (Fb_563v6)', s563v6s23: 'S23 (Fb_563v6)', s564v2s21: 'S21+ (Fb_564v2)', s564v2s23: 'S23 (Fb_564v2)', s564v3s21: 'S21+ (Fb_564v3)', s564v3s23: 'S23 (Fb_564v3)', s565: 'S21+ (Fb_565_S21)', s565s23: 'S23 (Fb_565_S23)', s565v2s21: 'S21+ (Fb_565v2_S21)', s565v2s23: 'S23 (Fb_565v2_S23)', s399: 'S23 (Fb_399)',
  // iOS Native App (FBIOS) group
  ios562: 'iOS App (FBIOS 562)',
ios564: 'iOS App (FBIOS 564)',
ios560: 'iOS App (FBIOS 560)',
ios555: 'iOS App (FBIOS 555)',
ios550: 'iOS App (FBIOS 550)',
ios540: 'iOS App (FBIOS 540)',
ios530: 'iOS App (FBIOS 530)',  ios520: 'iOS App (FBIOS 520)',
ios510: 'iOS App (FBIOS 510)',  ios500: 'iOS App (FBIOS 500)',
ios490: 'iOS App (FBIOS 490)',  ios480: 'iOS App (FBIOS 480)',
ios470: 'iOS App (FBIOS 470)',  ios460: 'iOS App (FBIOS 460)',
ios450: 'iOS App (FBIOS 450)',
ios421: 'iOS App (FBIOS 421)',
ios422: 'iOS App (FBIOS 422)',
ios423: 'iOS App (FBIOS 423)',
ios424: 'iOS App (FBIOS 424)',
ios425: 'iOS App (FBIOS 425)',
ios426: 'iOS App (FBIOS 426)',
ios427: 'iOS App (FBIOS 427)',
ios428: 'iOS App (FBIOS 428)',
ios429: 'iOS App (FBIOS 429)',
ios431: 'iOS App (FBIOS 431)',
ios432: 'iOS App (FBIOS 432)',
ios433: 'iOS App (FBIOS 433)',
ios434: 'iOS App (FBIOS 434)',
ios435: 'iOS App (FBIOS 435)',
ios436: 'iOS App (FBIOS 436)',
ios437: 'iOS App (FBIOS 437)',
ios438: 'iOS App (FBIOS 438)',
ios439: 'iOS App (FBIOS 439)',
ios441: 'iOS App (FBIOS 441)',
ios442: 'iOS App (FBIOS 442)',
ios443: 'iOS App (FBIOS 443)',
ios444: 'iOS App (FBIOS 444)',
ios445: 'iOS App (FBIOS 445)',
ios446: 'iOS App (FBIOS 446)',
ios447: 'iOS App (FBIOS 447)',
ios448: 'iOS App (FBIOS 448)',
ios449: 'iOS App (FBIOS 449)',
ios451: 'iOS App (FBIOS 451)',
ios452: 'iOS App (FBIOS 452)',
ios453: 'iOS App (FBIOS 453)',
ios454: 'iOS App (FBIOS 454)',
ios455: 'iOS App (FBIOS 455)',
ios456: 'iOS App (FBIOS 456)',
ios457: 'iOS App (FBIOS 457)',
ios458: 'iOS App (FBIOS 458)',
ios459: 'iOS App (FBIOS 459)',
ios461: 'iOS App (FBIOS 461)',
ios462: 'iOS App (FBIOS 462)',
ios463: 'iOS App (FBIOS 463)',
ios464: 'iOS App (FBIOS 464)',
ios465: 'iOS App (FBIOS 465)',
ios466: 'iOS App (FBIOS 466)',
ios467: 'iOS App (FBIOS 467)',
ios468: 'iOS App (FBIOS 468)',
ios469: 'iOS App (FBIOS 469)',
ios471: 'iOS App (FBIOS 471)',
ios472: 'iOS App (FBIOS 472)',
ios473: 'iOS App (FBIOS 473)',
ios474: 'iOS App (FBIOS 474)',
ios475: 'iOS App (FBIOS 475)',
ios476: 'iOS App (FBIOS 476)',
ios477: 'iOS App (FBIOS 477)',
ios478: 'iOS App (FBIOS 478)',
ios479: 'iOS App (FBIOS 479)',
ios481: 'iOS App (FBIOS 481)',
ios482: 'iOS App (FBIOS 482)',
ios483: 'iOS App (FBIOS 483)',
ios484: 'iOS App (FBIOS 484)',
ios485: 'iOS App (FBIOS 485)',
ios486: 'iOS App (FBIOS 486)',
ios487: 'iOS App (FBIOS 487)',
ios488: 'iOS App (FBIOS 488)',
ios489: 'iOS App (FBIOS 489)',
ios491: 'iOS App (FBIOS 491)',
ios492: 'iOS App (FBIOS 492)',
ios493: 'iOS App (FBIOS 493)',
ios494: 'iOS App (FBIOS 494)',
ios495: 'iOS App (FBIOS 495)',
ios496: 'iOS App (FBIOS 496)',
ios497: 'iOS App (FBIOS 497)',
ios498: 'iOS App (FBIOS 498)',
ios499: 'iOS App (FBIOS 499)',
ios501: 'iOS App (FBIOS 501)',
ios502: 'iOS App (FBIOS 502)',
ios503: 'iOS App (FBIOS 503)',
ios504: 'iOS App (FBIOS 504)',
ios505: 'iOS App (FBIOS 505)',
ios506: 'iOS App (FBIOS 506)',
ios507: 'iOS App (FBIOS 507)',
ios508: 'iOS App (FBIOS 508)',
ios509: 'iOS App (FBIOS 509)',
ios511: 'iOS App (FBIOS 511)',
ios512: 'iOS App (FBIOS 512)',
ios513: 'iOS App (FBIOS 513)',
ios514: 'iOS App (FBIOS 514)',
ios515: 'iOS App (FBIOS 515)',
ios516: 'iOS App (FBIOS 516)',
ios517: 'iOS App (FBIOS 517)',
ios518: 'iOS App (FBIOS 518)',
ios519: 'iOS App (FBIOS 519)',
ios521: 'iOS App (FBIOS 521)',
ios522: 'iOS App (FBIOS 522)',
ios523: 'iOS App (FBIOS 523)',
ios524: 'iOS App (FBIOS 524)',
ios525: 'iOS App (FBIOS 525)',
ios526: 'iOS App (FBIOS 526)',
ios527: 'iOS App (FBIOS 527)',
ios528: 'iOS App (FBIOS 528)',
ios529: 'iOS App (FBIOS 529)',
ios531: 'iOS App (FBIOS 531)',
ios532: 'iOS App (FBIOS 532)',
ios533: 'iOS App (FBIOS 533)',
ios534: 'iOS App (FBIOS 534)',
ios535: 'iOS App (FBIOS 535)',
ios536: 'iOS App (FBIOS 536)',
ios537: 'iOS App (FBIOS 537)',
ios538: 'iOS App (FBIOS 538)',
ios539: 'iOS App (FBIOS 539)',
ios541: 'iOS App (FBIOS 541)',
ios542: 'iOS App (FBIOS 542)',
ios543: 'iOS App (FBIOS 543)',
ios544: 'iOS App (FBIOS 544)',
ios545: 'iOS App (FBIOS 545)',
ios546: 'iOS App (FBIOS 546)',
ios547: 'iOS App (FBIOS 547)',
ios548: 'iOS App (FBIOS 548)',
ios549: 'iOS App (FBIOS 549)',
ios551: 'iOS App (FBIOS 551)',
ios552: 'iOS App (FBIOS 552)',
ios553: 'iOS App (FBIOS 553)',
ios554: 'iOS App (FBIOS 554)',
ios556: 'iOS App (FBIOS 556)',
ios557: 'iOS App (FBIOS 557)',
ios558: 'iOS App (FBIOS 558)',
ios559: 'iOS App (FBIOS 559)',
ios561: 'iOS App (FBIOS 561)',
ios440: 'iOS App (FBIOS 440)',
ios430: 'iOS App (FBIOS 430)',  ios420: 'iOS App (FBIOS 420)',
}
const VER_PLATFORM_LABELS: Record<string, string> = {
  s273: 'Vivo V2242A (Fb_273)',
  'api android': 'api android', 'api mfb': 'api mfb',
  'api token': 'api token', 'api web andr': 'api web andr',
  s415: 'S23 (Fb_415)', s425: 'S23 (Fb_425)', s435: 'S23 (Fb_435)', s445: 'S23 (Fb_445)', s455: 'S23 (Fb_455)',
  s550v2: 'S23 (Fb_550v2)', s551v2: 'S23 (Fb_551v2)', s552v2: 'S23 (Fb_552v2)', s553v2: 'S23 (Fb_553v2)', s554v2: 'S23 (Fb_554v2)', s555: 'S23 (Fb_555)', s555v2: 'S23 (Fb_555v2)', s556: 'S23 (Fb_556)', s556v2: 'S23 (Fb_556v2)', s557: 'S23 (Fb_557)', s557v2: 'S23 (Fb_557v2)',
  s558: 'S23 (Fb_558)', s558v2: 'S23 (Fb_558v2)', s559: 'S23 (Fb_559)', s559v2: 'S23 (Fb_559v2)', s560: 'S23 (Fb_560)', s560v2: 'S23 (Fb_560v2)', s560v3: 'S23 (Fb_560v3)', s561: 'S23 (Fb_561)', s561v2: 'S23 (Fb_561v2)', s561v3: 'S23 (Fb_561v3)', s561v99: 'S23 (Fb_561v99)', s561v4s21: 'S21+ (Fb_561v4)', s561v4s23: 'S23 (Fb_561v4)', s562: 'S23 (Fb_562)', s562v3: 'S23 (Fb_562v3)', s562v4s21: 'S21+ (Fb_562v4)', s562v4s23: 'S23 (Fb_562v4)', s563: 'S23 (Fb_563)', s563v2: 'S23 (Fb_563v2)', s563s21: 'S21+ (Fb_563)', s563v3s21: 'S21+ (Fb_563v3)', s563v4s21: 'S21+ (Fb_563v4)', s563v4s23: 'S23 (Fb_563v4)', s564v1s21: 'S21+ (Fb_564v1)', s564v1s23: 'S23 (Fb_564v1)', s563v5s21: 'S21+ (Fb_563v5)', s563v5s23: 'S23 (Fb_563v5)', s563v6s21: 'S21+ (Fb_563v6)', s563v6s23: 'S23 (Fb_563v6)', s564v2s21: 'S21+ (Fb_564v2)', s564v2s23: 'S23 (Fb_564v2)', s564v3s21: 'S21+ (Fb_564v3)', s564v3s23: 'S23 (Fb_564v3)', s565: 'S21+ (Fb_565_S21)', s565s23: 'S23 (Fb_565_S23)', s565v2s21: 'S21+ (Fb_565v2_S21)', s565v2s23: 'S23 (Fb_565v2_S23)',
  s399: 'S23 (Fb_399 — 2-step)',
  // iOS Native App (FBIOS) verify group
  ios562: 'iOS App (FBIOS 562)',
ios564: 'iOS App (FBIOS 564)',
ios560: 'iOS App (FBIOS 560)',
ios555: 'iOS App (FBIOS 555)',
ios550: 'iOS App (FBIOS 550)',
ios540: 'iOS App (FBIOS 540)',
ios530: 'iOS App (FBIOS 530)',  ios520: 'iOS App (FBIOS 520)',
ios510: 'iOS App (FBIOS 510)',  ios500: 'iOS App (FBIOS 500)',
ios490: 'iOS App (FBIOS 490)',  ios480: 'iOS App (FBIOS 480)',
ios470: 'iOS App (FBIOS 470)',  ios460: 'iOS App (FBIOS 460)',
ios450: 'iOS App (FBIOS 450)',
ios421: 'iOS App (FBIOS 421)',
ios422: 'iOS App (FBIOS 422)',
ios423: 'iOS App (FBIOS 423)',
ios424: 'iOS App (FBIOS 424)',
ios425: 'iOS App (FBIOS 425)',
ios426: 'iOS App (FBIOS 426)',
ios427: 'iOS App (FBIOS 427)',
ios428: 'iOS App (FBIOS 428)',
ios429: 'iOS App (FBIOS 429)',
ios431: 'iOS App (FBIOS 431)',
ios432: 'iOS App (FBIOS 432)',
ios433: 'iOS App (FBIOS 433)',
ios434: 'iOS App (FBIOS 434)',
ios435: 'iOS App (FBIOS 435)',
ios436: 'iOS App (FBIOS 436)',
ios437: 'iOS App (FBIOS 437)',
ios438: 'iOS App (FBIOS 438)',
ios439: 'iOS App (FBIOS 439)',
ios441: 'iOS App (FBIOS 441)',
ios442: 'iOS App (FBIOS 442)',
ios443: 'iOS App (FBIOS 443)',
ios444: 'iOS App (FBIOS 444)',
ios445: 'iOS App (FBIOS 445)',
ios446: 'iOS App (FBIOS 446)',
ios447: 'iOS App (FBIOS 447)',
ios448: 'iOS App (FBIOS 448)',
ios449: 'iOS App (FBIOS 449)',
ios451: 'iOS App (FBIOS 451)',
ios452: 'iOS App (FBIOS 452)',
ios453: 'iOS App (FBIOS 453)',
ios454: 'iOS App (FBIOS 454)',
ios455: 'iOS App (FBIOS 455)',
ios456: 'iOS App (FBIOS 456)',
ios457: 'iOS App (FBIOS 457)',
ios458: 'iOS App (FBIOS 458)',
ios459: 'iOS App (FBIOS 459)',
ios461: 'iOS App (FBIOS 461)',
ios462: 'iOS App (FBIOS 462)',
ios463: 'iOS App (FBIOS 463)',
ios464: 'iOS App (FBIOS 464)',
ios465: 'iOS App (FBIOS 465)',
ios466: 'iOS App (FBIOS 466)',
ios467: 'iOS App (FBIOS 467)',
ios468: 'iOS App (FBIOS 468)',
ios469: 'iOS App (FBIOS 469)',
ios471: 'iOS App (FBIOS 471)',
ios472: 'iOS App (FBIOS 472)',
ios473: 'iOS App (FBIOS 473)',
ios474: 'iOS App (FBIOS 474)',
ios475: 'iOS App (FBIOS 475)',
ios476: 'iOS App (FBIOS 476)',
ios477: 'iOS App (FBIOS 477)',
ios478: 'iOS App (FBIOS 478)',
ios479: 'iOS App (FBIOS 479)',
ios481: 'iOS App (FBIOS 481)',
ios482: 'iOS App (FBIOS 482)',
ios483: 'iOS App (FBIOS 483)',
ios484: 'iOS App (FBIOS 484)',
ios485: 'iOS App (FBIOS 485)',
ios486: 'iOS App (FBIOS 486)',
ios487: 'iOS App (FBIOS 487)',
ios488: 'iOS App (FBIOS 488)',
ios489: 'iOS App (FBIOS 489)',
ios491: 'iOS App (FBIOS 491)',
ios492: 'iOS App (FBIOS 492)',
ios493: 'iOS App (FBIOS 493)',
ios494: 'iOS App (FBIOS 494)',
ios495: 'iOS App (FBIOS 495)',
ios496: 'iOS App (FBIOS 496)',
ios497: 'iOS App (FBIOS 497)',
ios498: 'iOS App (FBIOS 498)',
ios499: 'iOS App (FBIOS 499)',
ios501: 'iOS App (FBIOS 501)',
ios502: 'iOS App (FBIOS 502)',
ios503: 'iOS App (FBIOS 503)',
ios504: 'iOS App (FBIOS 504)',
ios505: 'iOS App (FBIOS 505)',
ios506: 'iOS App (FBIOS 506)',
ios507: 'iOS App (FBIOS 507)',
ios508: 'iOS App (FBIOS 508)',
ios509: 'iOS App (FBIOS 509)',
ios511: 'iOS App (FBIOS 511)',
ios512: 'iOS App (FBIOS 512)',
ios513: 'iOS App (FBIOS 513)',
ios514: 'iOS App (FBIOS 514)',
ios515: 'iOS App (FBIOS 515)',
ios516: 'iOS App (FBIOS 516)',
ios517: 'iOS App (FBIOS 517)',
ios518: 'iOS App (FBIOS 518)',
ios519: 'iOS App (FBIOS 519)',
ios521: 'iOS App (FBIOS 521)',
ios522: 'iOS App (FBIOS 522)',
ios523: 'iOS App (FBIOS 523)',
ios524: 'iOS App (FBIOS 524)',
ios525: 'iOS App (FBIOS 525)',
ios526: 'iOS App (FBIOS 526)',
ios527: 'iOS App (FBIOS 527)',
ios528: 'iOS App (FBIOS 528)',
ios529: 'iOS App (FBIOS 529)',
ios531: 'iOS App (FBIOS 531)',
ios532: 'iOS App (FBIOS 532)',
ios533: 'iOS App (FBIOS 533)',
ios534: 'iOS App (FBIOS 534)',
ios535: 'iOS App (FBIOS 535)',
ios536: 'iOS App (FBIOS 536)',
ios537: 'iOS App (FBIOS 537)',
ios538: 'iOS App (FBIOS 538)',
ios539: 'iOS App (FBIOS 539)',
ios541: 'iOS App (FBIOS 541)',
ios542: 'iOS App (FBIOS 542)',
ios543: 'iOS App (FBIOS 543)',
ios544: 'iOS App (FBIOS 544)',
ios545: 'iOS App (FBIOS 545)',
ios546: 'iOS App (FBIOS 546)',
ios547: 'iOS App (FBIOS 547)',
ios548: 'iOS App (FBIOS 548)',
ios549: 'iOS App (FBIOS 549)',
ios551: 'iOS App (FBIOS 551)',
ios552: 'iOS App (FBIOS 552)',
ios553: 'iOS App (FBIOS 553)',
ios554: 'iOS App (FBIOS 554)',
ios556: 'iOS App (FBIOS 556)',
ios557: 'iOS App (FBIOS 557)',
ios558: 'iOS App (FBIOS 558)',
ios559: 'iOS App (FBIOS 559)',
ios561: 'iOS App (FBIOS 561)',
ios440: 'iOS App (FBIOS 440)',
ios430: 'iOS App (FBIOS 430)',  ios420: 'iOS App (FBIOS 420)',
}

const REG_PLATFORMS_STD = [
  { key: 'android', label: 'Android' }, { key: 'webandroid', label: 'Web Android' },
  { key: 'ios', label: 'iOS' }, { key: 's22', label: 'S22' }, { key: 's23', label: 'S23' },
  { key: 's24', label: 'S24' }, { key: 's25', label: 'S25' }, { key: 's26', label: 'S26' },
]
const REG_PLATFORMS_VER = [
  { key: 's399', label: 'Fb_399' },
  { key: 's415', label: 'Fb_415' }, { key: 's416', label: 'Fb_416' },
  { key: 's417', label: 'Fb_417' }, { key: 's418', label: 'Fb_418' },
  { key: 's419', label: 'Fb_419' }, { key: 's420', label: 'Fb_420' },
  { key: 's421', label: 'Fb_421' }, { key: 's422', label: 'Fb_422' },
  { key: 's423', label: 'Fb_423' }, { key: 's424', label: 'Fb_424' },
  { key: 's425', label: 'Fb_425' }, { key: 's426', label: 'Fb_426' },
  { key: 's427', label: 'Fb_427' }, { key: 's428', label: 'Fb_428' },
  { key: 's429', label: 'Fb_429' }, { key: 's430', label: 'Fb_430' },
  { key: 's431', label: 'Fb_431' }, { key: 's432', label: 'Fb_432' },
  { key: 's433', label: 'Fb_433' }, { key: 's434', label: 'Fb_434' },
  { key: 's435', label: 'Fb_435' }, { key: 's436', label: 'Fb_436' },
  { key: 's437', label: 'Fb_437' }, { key: 's438', label: 'Fb_438' },
  { key: 's439', label: 'Fb_439' }, { key: 's440', label: 'Fb_440' },
  { key: 's441', label: 'Fb_441' }, { key: 's442', label: 'Fb_442' },
  { key: 's443', label: 'Fb_443' }, { key: 's444', label: 'Fb_444' },
  { key: 's445', label: 'Fb_445' }, { key: 's446', label: 'Fb_446' },
  { key: 's447', label: 'Fb_447' }, { key: 's448', label: 'Fb_448' },
  { key: 's449', label: 'Fb_449' }, { key: 's450', label: 'Fb_450' },
  { key: 's451', label: 'Fb_451' }, { key: 's452', label: 'Fb_452' },
  { key: 's453', label: 'Fb_453' }, { key: 's454', label: 'Fb_454' },
  { key: 's455', label: 'Fb_455' },
  { key: 's456', label: 'Fb_456' }, { key: 's457', label: 'Fb_457' },
  { key: 's458', label: 'Fb_458' }, { key: 's459', label: 'Fb_459' }, { key: 's460', label: 'Fb_460' },
  { key: 's461', label: 'Fb_461' }, { key: 's462', label: 'Fb_462' },
  { key: 's463', label: 'Fb_463' }, { key: 's464', label: 'Fb_464' }, { key: 's465', label: 'Fb_465' },
  { key: 's466', label: 'Fb_466' }, { key: 's467', label: 'Fb_467' },
  { key: 's468', label: 'Fb_468' }, { key: 's469', label: 'Fb_469' }, { key: 's470', label: 'Fb_470' },
  { key: 's471', label: 'Fb_471' }, { key: 's472', label: 'Fb_472' },
  { key: 's473', label: 'Fb_473' }, { key: 's474', label: 'Fb_474' }, { key: 's475', label: 'Fb_475' },
  { key: 's476', label: 'Fb_476' }, { key: 's477', label: 'Fb_477' },
  { key: 's478', label: 'Fb_478' }, { key: 's479', label: 'Fb_479' }, { key: 's480', label: 'Fb_480' },
  { key: 's481', label: 'Fb_481' }, { key: 's482', label: 'Fb_482' },
  { key: 's483', label: 'Fb_483' }, { key: 's484', label: 'Fb_484' }, { key: 's485', label: 'Fb_485' },
  { key: 's486', label: 'Fb_486' }, { key: 's487', label: 'Fb_487' },
  { key: 's488', label: 'Fb_488' }, { key: 's489', label: 'Fb_489' }, { key: 's490', label: 'Fb_490' },
  { key: 's491', label: 'Fb_491' }, { key: 's492', label: 'Fb_492' },
  { key: 's493', label: 'Fb_493' }, { key: 's494', label: 'Fb_494' }, { key: 's495', label: 'Fb_495' },
  { key: 's496', label: 'Fb_496' }, { key: 's497', label: 'Fb_497' },
  { key: 's498', label: 'Fb_498' }, { key: 's499', label: 'Fb_499' },
  { key: 's500', label: 'Fb_500' }, { key: 's501', label: 'Fb_501' },
  { key: 's502', label: 'Fb_502' }, { key: 's503', label: 'Fb_503' },
  { key: 's504', label: 'Fb_504' }, { key: 's505', label: 'Fb_505' },
  { key: 's506', label: 'Fb_506' }, { key: 's507', label: 'Fb_507' },
  { key: 's508', label: 'Fb_508' }, { key: 's509', label: 'Fb_509' },
  { key: 's510', label: 'Fb_510' }, { key: 's511', label: 'Fb_511' },
  { key: 's512', label: 'Fb_512' }, { key: 's513', label: 'Fb_513' },
  { key: 's514', label: 'Fb_514' }, { key: 's515', label: 'Fb_515' },
  { key: 's516', label: 'Fb_516' }, { key: 's517', label: 'Fb_517' },
  { key: 's518', label: 'Fb_518' }, { key: 's519', label: 'Fb_519' },
  { key: 's520', label: 'Fb_520' }, { key: 's521', label: 'Fb_521' },
  { key: 's522', label: 'Fb_522' }, { key: 's523', label: 'Fb_523' },
  { key: 's524', label: 'Fb_524' }, { key: 's525', label: 'Fb_525' },
  { key: 's526', label: 'Fb_526' }, { key: 's527', label: 'Fb_527' },
  { key: 's528', label: 'Fb_528' }, { key: 's529', label: 'Fb_529' },
  { key: 's530', label: 'Fb_530' }, { key: 's531', label: 'Fb_531' },
  { key: 's532', label: 'Fb_532' }, { key: 's533', label: 'Fb_533' }, { key: 's534', label: 'Fb_534' },
  { key: 's535', label: 'Fb_535' }, { key: 's536', label: 'Fb_536' },
  { key: 's537', label: 'Fb_537' }, { key: 's538', label: 'Fb_538' },
  { key: 's539', label: 'Fb_539' }, { key: 's540', label: 'Fb_540' },
  { key: 's541', label: 'Fb_541' }, { key: 's542', label: 'Fb_542' },
  { key: 's543', label: 'Fb_543' }, { key: 's544', label: 'Fb_544' },
  { key: 's545', label: 'Fb_545' }, { key: 's546', label: 'Fb_546' },
  { key: 's547', label: 'Fb_547' }, { key: 's548', label: 'Fb_548' },
  { key: 's549', label: 'Fb_549' }, { key: 's550', label: 'Fb_550' },
  { key: 's551', label: 'Fb_551' }, { key: 's552', label: 'Fb_552' },
  { key: 's553', label: 'Fb_553' }, { key: 's554', label: 'Fb_554' },
  { key: 's555', label: 'Fb_555' }, { key: 's555v2', label: 'Fb_555v2' }, { key: 's556', label: 'Fb_556' },
  { key: 's557', label: 'Fb_557' }, { key: 's558', label: 'Fb_558' }, { key: 's558v2', label: 'Fb_558v2' },
  { key: 's559', label: 'Fb_559' }, { key: 's559v2', label: 'Fb_559v2' },
  { key: 's560', label: 'Fb_560' }, { key: 's560v2', label: 'Fb_560v2' },
  { key: 's561', label: 'Fb_561' }, { key: 's561v2', label: 'Fb_561v2' }, { key: 's561v3', label: 'Fb_561v3' },{ key: 's561v99', label: 'Fb_561v99' },  { key: 's561v4s21', label: 'Fb_561v4_S21' }, { key: 's561v4s23', label: 'Fb_561v4_S23' },  { key: 's562', label: 'Fb_562' }, { key: 's562v3', label: 'Fb_562v3' }, { key: 's562v4s21', label: 'Fb_562v4_S21' }, { key: 's562v4s23', label: 'Fb_562v4_S23' }, { key: 's563', label: 'Fb_563' }, { key: 's563s21', label: 'Fb_563_S21' }, { key: 's563v3s21', label: 'Fb_563v3_S21' }, { key: 's563v4s21', label: 'Fb_563v4_S21' }, { key: 's563v4s23', label: 'Fb_563v4_S23' }, { key: 's563v5s21', label: 'Fb_563v5_S21' }, { key: 's563v5s23', label: 'Fb_563v5_S23' }, { key: 's563v6s21', label: 'Fb_563v6_S21' }, { key: 's563v6s23', label: 'Fb_563v6_S23' }, { key: 's564v1s21', label: 'Fb_564v1_S21' }, { key: 's564v1s23', label: 'Fb_564v1_S23' }, { key: 's564v2s21', label: 'Fb_564v2_S21' }, { key: 's564v2s23', label: 'Fb_564v2_S23' }, { key: 's564v3s21', label: 'Fb_564v3_S21' }, { key: 's564v3s23', label: 'Fb_564v3_S23' }, { key: 's565s21', label: 'Fb_565_S21' }, { key: 's565s23', label: 'Fb_565_S23' }, { key: 's565v2s21', label: 'Fb_565v2_S21' }, { key: 's565v2s23', label: 'Fb_565v2_S23' },
]

// iOS Native App (FBIOS) — KHÁC Android, dùng graph.facebook.com + OAuth 6628568379.
// Sẽ mở rộng khi có capture iOS versions mới (iosapp563, iosapp564, ...).
const REG_PLATFORMS_IOS = [
  { key: 'ios562', label: 'iOS_562' },
{ key: 'ios563', label: 'iOS_563' },
{ key: 'ios564', label: 'iOS_564' },
{ key: 'ios563', label: 'iOS_563' },
{ key: 'ios562', label: 'iOS_562' },
{ key: 'ios561', label: 'iOS_561' },
{ key: 'ios560', label: 'iOS_560' },
{ key: 'ios559', label: 'iOS_559' },
{ key: 'ios558', label: 'iOS_558' },
{ key: 'ios557', label: 'iOS_557' },
{ key: 'ios556', label: 'iOS_556' },
{ key: 'ios555', label: 'iOS_555' },
{ key: 'ios554', label: 'iOS_554' },
{ key: 'ios553', label: 'iOS_553' },
{ key: 'ios552', label: 'iOS_552' },
{ key: 'ios551', label: 'iOS_551' },
{ key: 'ios550', label: 'iOS_550' },
{ key: 'ios549', label: 'iOS_549' },
{ key: 'ios548', label: 'iOS_548' },
{ key: 'ios547', label: 'iOS_547' },
{ key: 'ios546', label: 'iOS_546' },
{ key: 'ios545', label: 'iOS_545' },
{ key: 'ios544', label: 'iOS_544' },
{ key: 'ios543', label: 'iOS_543' },
{ key: 'ios542', label: 'iOS_542' },
{ key: 'ios541', label: 'iOS_541' },
{ key: 'ios540', label: 'iOS_540' },
{ key: 'ios539', label: 'iOS_539' },
{ key: 'ios538', label: 'iOS_538' },
{ key: 'ios537', label: 'iOS_537' },
{ key: 'ios536', label: 'iOS_536' },
{ key: 'ios535', label: 'iOS_535' },
{ key: 'ios534', label: 'iOS_534' },
{ key: 'ios533', label: 'iOS_533' },
{ key: 'ios532', label: 'iOS_532' },
{ key: 'ios531', label: 'iOS_531' },
{ key: 'ios530', label: 'iOS_530' },
{ key: 'ios529', label: 'iOS_529' },
{ key: 'ios528', label: 'iOS_528' },
{ key: 'ios527', label: 'iOS_527' },
{ key: 'ios526', label: 'iOS_526' },
{ key: 'ios525', label: 'iOS_525' },
{ key: 'ios524', label: 'iOS_524' },
{ key: 'ios523', label: 'iOS_523' },
{ key: 'ios522', label: 'iOS_522' },
{ key: 'ios521', label: 'iOS_521' },
{ key: 'ios520', label: 'iOS_520' },
{ key: 'ios519', label: 'iOS_519' },
{ key: 'ios518', label: 'iOS_518' },
{ key: 'ios517', label: 'iOS_517' },
{ key: 'ios516', label: 'iOS_516' },
{ key: 'ios515', label: 'iOS_515' },
{ key: 'ios514', label: 'iOS_514' },
{ key: 'ios513', label: 'iOS_513' },
{ key: 'ios512', label: 'iOS_512' },
{ key: 'ios511', label: 'iOS_511' },
{ key: 'ios510', label: 'iOS_510' },
{ key: 'ios509', label: 'iOS_509' },
{ key: 'ios508', label: 'iOS_508' },
{ key: 'ios507', label: 'iOS_507' },
{ key: 'ios506', label: 'iOS_506' },
{ key: 'ios505', label: 'iOS_505' },
{ key: 'ios504', label: 'iOS_504' },
{ key: 'ios503', label: 'iOS_503' },
{ key: 'ios502', label: 'iOS_502' },
{ key: 'ios501', label: 'iOS_501' },
{ key: 'ios500', label: 'iOS_500' },
{ key: 'ios499', label: 'iOS_499' },
{ key: 'ios498', label: 'iOS_498' },
{ key: 'ios497', label: 'iOS_497' },
{ key: 'ios496', label: 'iOS_496' },
{ key: 'ios495', label: 'iOS_495' },
{ key: 'ios494', label: 'iOS_494' },
{ key: 'ios493', label: 'iOS_493' },
{ key: 'ios492', label: 'iOS_492' },
{ key: 'ios491', label: 'iOS_491' },
{ key: 'ios490', label: 'iOS_490' },
{ key: 'ios489', label: 'iOS_489' },
{ key: 'ios488', label: 'iOS_488' },
{ key: 'ios487', label: 'iOS_487' },
{ key: 'ios486', label: 'iOS_486' },
{ key: 'ios485', label: 'iOS_485' },
{ key: 'ios484', label: 'iOS_484' },
{ key: 'ios483', label: 'iOS_483' },
{ key: 'ios482', label: 'iOS_482' },
{ key: 'ios481', label: 'iOS_481' },
{ key: 'ios480', label: 'iOS_480' },
{ key: 'ios479', label: 'iOS_479' },
{ key: 'ios478', label: 'iOS_478' },
{ key: 'ios477', label: 'iOS_477' },
{ key: 'ios476', label: 'iOS_476' },
{ key: 'ios475', label: 'iOS_475' },
{ key: 'ios474', label: 'iOS_474' },
{ key: 'ios473', label: 'iOS_473' },
{ key: 'ios472', label: 'iOS_472' },
{ key: 'ios471', label: 'iOS_471' },
{ key: 'ios470', label: 'iOS_470' },
{ key: 'ios469', label: 'iOS_469' },
{ key: 'ios468', label: 'iOS_468' },
{ key: 'ios467', label: 'iOS_467' },
{ key: 'ios466', label: 'iOS_466' },
{ key: 'ios465', label: 'iOS_465' },
{ key: 'ios464', label: 'iOS_464' },
{ key: 'ios463', label: 'iOS_463' },
{ key: 'ios462', label: 'iOS_462' },
{ key: 'ios461', label: 'iOS_461' },
{ key: 'ios460', label: 'iOS_460' },
{ key: 'ios459', label: 'iOS_459' },
{ key: 'ios458', label: 'iOS_458' },
{ key: 'ios457', label: 'iOS_457' },
{ key: 'ios456', label: 'iOS_456' },
{ key: 'ios455', label: 'iOS_455' },
{ key: 'ios454', label: 'iOS_454' },
{ key: 'ios453', label: 'iOS_453' },
{ key: 'ios452', label: 'iOS_452' },
{ key: 'ios451', label: 'iOS_451' },
{ key: 'ios450', label: 'iOS_450' },
{ key: 'ios449', label: 'iOS_449' },
{ key: 'ios448', label: 'iOS_448' },
{ key: 'ios447', label: 'iOS_447' },
{ key: 'ios446', label: 'iOS_446' },
{ key: 'ios445', label: 'iOS_445' },
{ key: 'ios444', label: 'iOS_444' },
{ key: 'ios443', label: 'iOS_443' },
{ key: 'ios442', label: 'iOS_442' },
{ key: 'ios441', label: 'iOS_441' },
{ key: 'ios440', label: 'iOS_440' },
{ key: 'ios439', label: 'iOS_439' },
{ key: 'ios438', label: 'iOS_438' },
{ key: 'ios437', label: 'iOS_437' },
{ key: 'ios436', label: 'iOS_436' },
{ key: 'ios435', label: 'iOS_435' },
{ key: 'ios434', label: 'iOS_434' },
{ key: 'ios433', label: 'iOS_433' },
{ key: 'ios432', label: 'iOS_432' },
{ key: 'ios431', label: 'iOS_431' },
{ key: 'ios430', label: 'iOS_430' },
{ key: 'ios429', label: 'iOS_429' },
{ key: 'ios428', label: 'iOS_428' },
{ key: 'ios427', label: 'iOS_427' },
{ key: 'ios426', label: 'iOS_426' },
{ key: 'ios425', label: 'iOS_425' },
{ key: 'ios424', label: 'iOS_424' },
{ key: 'ios423', label: 'iOS_423' },
{ key: 'ios422', label: 'iOS_422' },
{ key: 'ios421', label: 'iOS_421' },
{ key: 'ios420', label: 'iOS_420' },
]

const VER_PLATFORMS_ANDR = [
  { key: 'api android', label: 'android' }, { key: 'api token', label: 'token' },
]
const VER_PLATFORMS_MFB = [
  { key: 'api mfb', label: 'mfb' }, { key: 'api web andr', label: 'web andr' },
]
const VER_PLATFORMS_VER = [
  { key: 's273', label: 'Fb_273' },
  { key: 's399', label: 'Fb_399' },
  { key: 's415', label: 'Fb_415' }, { key: 's425', label: 'Fb_425' },
  { key: 's435', label: 'Fb_435' }, { key: 's445', label: 'Fb_445' }, { key: 's455', label: 'Fb_455' },
  { key: 's550v2', label: 'Fb_550v2' }, { key: 's551v2', label: 'Fb_551v2' }, { key: 's552v2', label: 'Fb_552v2' }, { key: 's553v2', label: 'Fb_553v2' }, { key: 's554v2', label: 'Fb_554v2' }, { key: 's555', label: 'Fb_555' }, { key: 's555v2', label: 'Fb_555v2' }, { key: 's556', label: 'Fb_556' }, { key: 's556v2', label: 'Fb_556v2' },
  { key: 's557', label: 'Fb_557' }, { key: 's557v2', label: 'Fb_557v2' }, { key: 's558', label: 'Fb_558' }, { key: 's558v2', label: 'Fb_558v2' },
  { key: 's559', label: 'Fb_559' }, { key: 's559v2', label: 'Fb_559v2' }, { key: 's560', label: 'Fb_560' },
  { key: 's560v2', label: 'Fb_560v2' }, { key: 's560v3', label: 'Fb_560v3' },
  { key: 's561', label: 'Fb_561' }, { key: 's561v2', label: 'Fb_561v2' }, { key: 's561v3', label: 'Fb_561v3' }, { key: 's561v99', label: 'Fb_561v99' }, { key: 's561v4s21', label: 'Fb_561v4_S21' }, { key: 's561v4s23', label: 'Fb_561v4_S23' }, { key: 's562', label: 'Fb_562' }, { key: 's562v3', label: 'Fb_562v3' }, { key: 's562v4s21', label: 'Fb_562v4_S21' }, { key: 's562v4s23', label: 'Fb_562v4_S23' }, { key: 's563', label: 'Fb_563' }, { key: 's563v2', label: 'Fb_563v2' }, { key: 's563s21', label: 'Fb_563_S21' }, { key: 's563v3s21', label: 'Fb_563v3_S21' }, { key: 's563v4s21', label: 'Fb_563v4_S21' }, { key: 's563v4s23', label: 'Fb_563v4_S23' }, { key: 's563v5s21', label: 'Fb_563v5_S21' }, { key: 's563v5s23', label: 'Fb_563v5_S23' }, { key: 's563v6s21', label: 'Fb_563v6_S21' }, { key: 's563v6s23', label: 'Fb_563v6_S23' }, { key: 's564v1s21', label: 'Fb_564v1_S21' }, { key: 's564v1s23', label: 'Fb_564v1_S23' }, { key: 's564v2s21', label: 'Fb_564v2_S21' }, { key: 's564v2s23', label: 'Fb_564v2_S23' }, { key: 's564v3s21', label: 'Fb_564v3_S21' }, { key: 's564v3s23', label: 'Fb_564v3_S23' }, { key: 's565s21', label: 'Fb_565_S21' }, { key: 's565s23', label: 'Fb_565_S23' }, { key: 's565v2s21', label: 'Fb_565v2_S21' }, { key: 's565v2s23', label: 'Fb_565v2_S23' },
]

// iOS Native App (FBIOS) verify group.
// iOS_562 verify đã dùng bộ constant v563 (doc_id/bloks/FBAV 563) → verify được account iOS.
// KHÔNG có nút iOS_563 ở verify: chưa có verifier ios563 riêng (sẽ "verifier not registered").
// Khi nào có capture verify iOS563 thật + verify/ios563/ thì mới thêm lại.
const VER_PLATFORMS_IOS = [
  { key: 'ios562', label: 'iOS_562' },
{ key: 'ios563', label: 'iOS_563' },
{ key: 'ios564', label: 'iOS_564' },
{ key: 'ios563', label: 'iOS_563' },
{ key: 'ios562', label: 'iOS_562' },
{ key: 'ios561', label: 'iOS_561' },
{ key: 'ios560', label: 'iOS_560' },
{ key: 'ios559', label: 'iOS_559' },
{ key: 'ios558', label: 'iOS_558' },
{ key: 'ios557', label: 'iOS_557' },
{ key: 'ios556', label: 'iOS_556' },
{ key: 'ios555', label: 'iOS_555' },
{ key: 'ios554', label: 'iOS_554' },
{ key: 'ios553', label: 'iOS_553' },
{ key: 'ios552', label: 'iOS_552' },
{ key: 'ios551', label: 'iOS_551' },
{ key: 'ios550', label: 'iOS_550' },
{ key: 'ios549', label: 'iOS_549' },
{ key: 'ios548', label: 'iOS_548' },
{ key: 'ios547', label: 'iOS_547' },
{ key: 'ios546', label: 'iOS_546' },
{ key: 'ios545', label: 'iOS_545' },
{ key: 'ios544', label: 'iOS_544' },
{ key: 'ios543', label: 'iOS_543' },
{ key: 'ios542', label: 'iOS_542' },
{ key: 'ios541', label: 'iOS_541' },
{ key: 'ios540', label: 'iOS_540' },
{ key: 'ios539', label: 'iOS_539' },
{ key: 'ios538', label: 'iOS_538' },
{ key: 'ios537', label: 'iOS_537' },
{ key: 'ios536', label: 'iOS_536' },
{ key: 'ios535', label: 'iOS_535' },
{ key: 'ios534', label: 'iOS_534' },
{ key: 'ios533', label: 'iOS_533' },
{ key: 'ios532', label: 'iOS_532' },
{ key: 'ios531', label: 'iOS_531' },
{ key: 'ios530', label: 'iOS_530' },
{ key: 'ios529', label: 'iOS_529' },
{ key: 'ios528', label: 'iOS_528' },
{ key: 'ios527', label: 'iOS_527' },
{ key: 'ios526', label: 'iOS_526' },
{ key: 'ios525', label: 'iOS_525' },
{ key: 'ios524', label: 'iOS_524' },
{ key: 'ios523', label: 'iOS_523' },
{ key: 'ios522', label: 'iOS_522' },
{ key: 'ios521', label: 'iOS_521' },
{ key: 'ios520', label: 'iOS_520' },
{ key: 'ios519', label: 'iOS_519' },
{ key: 'ios518', label: 'iOS_518' },
{ key: 'ios517', label: 'iOS_517' },
{ key: 'ios516', label: 'iOS_516' },
{ key: 'ios515', label: 'iOS_515' },
{ key: 'ios514', label: 'iOS_514' },
{ key: 'ios513', label: 'iOS_513' },
{ key: 'ios512', label: 'iOS_512' },
{ key: 'ios511', label: 'iOS_511' },
{ key: 'ios510', label: 'iOS_510' },
{ key: 'ios509', label: 'iOS_509' },
{ key: 'ios508', label: 'iOS_508' },
{ key: 'ios507', label: 'iOS_507' },
{ key: 'ios506', label: 'iOS_506' },
{ key: 'ios505', label: 'iOS_505' },
{ key: 'ios504', label: 'iOS_504' },
{ key: 'ios503', label: 'iOS_503' },
{ key: 'ios502', label: 'iOS_502' },
{ key: 'ios501', label: 'iOS_501' },
{ key: 'ios500', label: 'iOS_500' },
{ key: 'ios499', label: 'iOS_499' },
{ key: 'ios498', label: 'iOS_498' },
{ key: 'ios497', label: 'iOS_497' },
{ key: 'ios496', label: 'iOS_496' },
{ key: 'ios495', label: 'iOS_495' },
{ key: 'ios494', label: 'iOS_494' },
{ key: 'ios493', label: 'iOS_493' },
{ key: 'ios492', label: 'iOS_492' },
{ key: 'ios491', label: 'iOS_491' },
{ key: 'ios490', label: 'iOS_490' },
{ key: 'ios489', label: 'iOS_489' },
{ key: 'ios488', label: 'iOS_488' },
{ key: 'ios487', label: 'iOS_487' },
{ key: 'ios486', label: 'iOS_486' },
{ key: 'ios485', label: 'iOS_485' },
{ key: 'ios484', label: 'iOS_484' },
{ key: 'ios483', label: 'iOS_483' },
{ key: 'ios482', label: 'iOS_482' },
{ key: 'ios481', label: 'iOS_481' },
{ key: 'ios480', label: 'iOS_480' },
{ key: 'ios479', label: 'iOS_479' },
{ key: 'ios478', label: 'iOS_478' },
{ key: 'ios477', label: 'iOS_477' },
{ key: 'ios476', label: 'iOS_476' },
{ key: 'ios475', label: 'iOS_475' },
{ key: 'ios474', label: 'iOS_474' },
{ key: 'ios473', label: 'iOS_473' },
{ key: 'ios472', label: 'iOS_472' },
{ key: 'ios471', label: 'iOS_471' },
{ key: 'ios470', label: 'iOS_470' },
{ key: 'ios469', label: 'iOS_469' },
{ key: 'ios468', label: 'iOS_468' },
{ key: 'ios467', label: 'iOS_467' },
{ key: 'ios466', label: 'iOS_466' },
{ key: 'ios465', label: 'iOS_465' },
{ key: 'ios464', label: 'iOS_464' },
{ key: 'ios463', label: 'iOS_463' },
{ key: 'ios462', label: 'iOS_462' },
{ key: 'ios461', label: 'iOS_461' },
{ key: 'ios460', label: 'iOS_460' },
{ key: 'ios459', label: 'iOS_459' },
{ key: 'ios458', label: 'iOS_458' },
{ key: 'ios457', label: 'iOS_457' },
{ key: 'ios456', label: 'iOS_456' },
{ key: 'ios455', label: 'iOS_455' },
{ key: 'ios454', label: 'iOS_454' },
{ key: 'ios453', label: 'iOS_453' },
{ key: 'ios452', label: 'iOS_452' },
{ key: 'ios451', label: 'iOS_451' },
{ key: 'ios450', label: 'iOS_450' },
{ key: 'ios449', label: 'iOS_449' },
{ key: 'ios448', label: 'iOS_448' },
{ key: 'ios447', label: 'iOS_447' },
{ key: 'ios446', label: 'iOS_446' },
{ key: 'ios445', label: 'iOS_445' },
{ key: 'ios444', label: 'iOS_444' },
{ key: 'ios443', label: 'iOS_443' },
{ key: 'ios442', label: 'iOS_442' },
{ key: 'ios441', label: 'iOS_441' },
{ key: 'ios440', label: 'iOS_440' },
{ key: 'ios439', label: 'iOS_439' },
{ key: 'ios438', label: 'iOS_438' },
{ key: 'ios437', label: 'iOS_437' },
{ key: 'ios436', label: 'iOS_436' },
{ key: 'ios435', label: 'iOS_435' },
{ key: 'ios434', label: 'iOS_434' },
{ key: 'ios433', label: 'iOS_433' },
{ key: 'ios432', label: 'iOS_432' },
{ key: 'ios431', label: 'iOS_431' },
{ key: 'ios430', label: 'iOS_430' },
{ key: 'ios429', label: 'iOS_429' },
{ key: 'ios428', label: 'iOS_428' },
{ key: 'ios427', label: 'iOS_427' },
{ key: 'ios426', label: 'iOS_426' },
{ key: 'ios425', label: 'iOS_425' },
{ key: 'ios424', label: 'iOS_424' },
{ key: 'ios423', label: 'iOS_423' },
{ key: 'ios422', label: 'iOS_422' },
{ key: 'ios421', label: 'iOS_421' },
{ key: 'ios420', label: 'iOS_420' },
]

type PlatformOption = { key: string; label: string }
const DISABLED_PLATFORM_KEYS = new Set(['s399'])

// Messenger groups — tách riêng khỏi lưới version để không lẫn vào các bản version.
// Android Mess (Reg Mess / AppMess V3) đứng cạnh cụm Version; iOS Mess (Reg/Ver Mess iOS) đứng trong cụm iOS.
const REG_PLATFORMS_MESS_ANDR: PlatformOption[] = [
  { key: 'appmv3reg', label: 'Reg Mess' },
  { key: 'appmv3reg535', label: 'Reg Mess 535' },
  { key: 'appmv3reg545', label: 'Reg Mess 545' },
  { key: 'appmv3reg555', label: 'Reg Mess 555' },
  { key: 'appmv3reg563', label: 'Reg Mess 563' },
  { key: 'appmv3reg564', label: 'Reg Mess 564' },
  { key: 'appmv3reg565', label: 'Reg Mess 565' },
  { key: 'appmv3reg525', label: 'Reg Mess 525' },
  { key: 'appmv3reg515', label: 'Reg Mess 515' },
  { key: 'appmv3reg505', label: 'Reg Mess 505' },
  { key: 'appmv3reg490', label: 'Reg Mess 490' },
]
const REG_PLATFORMS_MESS_IOS: PlatformOption[] = [{ key: 'iosmessreg', label: 'Reg Mess iOS' }]
const VER_PLATFORMS_MESS_ANDR: PlatformOption[] = [
  { key: 'appmessv3', label: 'AppMess V3' },
  { key: 'appmessv3_535', label: 'AppMess 535' },
  { key: 'appmessv3_545', label: 'AppMess 545' },
  { key: 'appmessv3_555', label: 'AppMess 555' },
  { key: 'appmessv3_563', label: 'AppMess 563' },
  { key: 'appmessv3_564', label: 'AppMess 564' },
  { key: 'appmessv3_565', label: 'AppMess 565' },
  { key: 'appmessv3_525', label: 'AppMess 525' },
  { key: 'appmessv3_515', label: 'AppMess 515' },
  { key: 'appmessv3_505', label: 'AppMess 505' },
  { key: 'appmessv3_490', label: 'AppMess 490' },
]
const VER_PLATFORMS_MESS_IOS: PlatformOption[] = [{ key: 'iosmess', label: 'Ver Mess iOS' }]

const IOS_PLATFORM_KEY_SET = new Set([
  ...REG_PLATFORMS_IOS.map(p => p.key),
  ...VER_PLATFORMS_IOS.map(p => p.key),
  // Mess iOS (Reg/Ver Mess iOS) chạy MessengerLite iOS → pool UA = iPhone, KHÔNG cho chọn Android.
  ...REG_PLATFORMS_MESS_IOS.map(p => p.key),
  ...VER_PLATFORMS_MESS_IOS.map(p => p.key),
  'ios',
])

// Messenger pool riêng: Orca Android → Android_Mess.txt, MessengerLite iOS → iOS_Mess.txt.
const MESS_ANDR_KEY_SET = new Set<string>([
  ...REG_PLATFORMS_MESS_ANDR.map(p => p.key),
  ...VER_PLATFORMS_MESS_ANDR.map(p => p.key),
])
const MESS_IOS_KEY_SET = new Set<string>([
  ...REG_PLATFORMS_MESS_IOS.map(p => p.key),
  ...VER_PLATFORMS_MESS_IOS.map(p => p.key),
])

function getAllowedPoolForPlatform(platform: string): string {
  // Messenger check TRƯỚC iOS (Mess iOS cũng nằm trong IOS_PLATFORM_KEY_SET).
  if (MESS_ANDR_KEY_SET.has(platform)) return 'android_mess' // Mess Android → Android_Mess.txt
  if (MESS_IOS_KEY_SET.has(platform)) return 'ios_mess'       // Mess iOS → iOS_Mess.txt
  if (IOS_PLATFORM_KEY_SET.has(platform)) return 'iphone'
  if (platform === 'webandroid' || platform === 'api web andr') return 'webchrome'
  if (platform === 'api mfb') return 'request'
  return 'android'
}

function isPlatformSelectable(platform: string): boolean {
  return !DISABLED_PLATFORM_KEYS.has(platform)
}

function chunkPlatformOptions(items: PlatformOption[], size = 10): PlatformOption[][] {
  const chunks: PlatformOption[][] = []
  for (let i = 0; i < items.length; i += size) chunks.push(items.slice(i, i + size))
  return chunks
}

function platformRangeLabel(items: PlatformOption[]): string {
  const first = items[0]?.label ?? ''
  const last = items[items.length - 1]?.label ?? first
  return first === last ? first : `${first} - ${last}`
}

const regVersionBatches = computed(() => chunkPlatformOptions(REG_PLATFORMS_VER))
const verifyVersionBatches = computed(() => chunkPlatformOptions(VER_PLATFORMS_VER))
const regIosBatches = computed(() => chunkPlatformOptions(REG_PLATFORMS_IOS, 20))
const verifyIosBatches = computed(() => chunkPlatformOptions(VER_PLATFORMS_IOS, 20))
const regPlatformKeys = computed(() => new Set([...REG_PLATFORMS_STD, ...REG_PLATFORMS_VER, ...REG_PLATFORMS_MESS_ANDR, ...REG_PLATFORMS_IOS, ...REG_PLATFORMS_MESS_IOS].map(p => p.key)))
const verifyPlatformKeys = computed(() => new Set([...VER_PLATFORMS_ANDR, ...VER_PLATFORMS_MFB, ...VER_PLATFORMS_VER, ...VER_PLATFORMS_MESS_ANDR, ...VER_PLATFORMS_IOS, ...VER_PLATFORMS_MESS_IOS].map(p => p.key)))
const allRegPlatformOptions = computed(() => [...REG_PLATFORMS_STD, ...REG_PLATFORMS_VER, ...REG_PLATFORMS_MESS_ANDR, ...REG_PLATFORMS_IOS, ...REG_PLATFORMS_MESS_IOS])
const allVerifyPlatformOptions = computed(() => [...VER_PLATFORMS_ANDR, ...VER_PLATFORMS_MFB, ...VER_PLATFORMS_VER, ...VER_PLATFORMS_MESS_ANDR, ...VER_PLATFORMS_IOS, ...VER_PLATFORMS_MESS_IOS])

const PLATFORM_DISPLAY_KEYS = {
  reg: 'hvr:setupVisibleRegPlatforms',
  verify: 'hvr:setupVisibleVerifyPlatforms',
} as const

function loadVisiblePlatformKeys(storageKey: string, fallback: PlatformOption[]): string[] {
  const fallbackKeys = fallback.map(p => p.key)
  try {
    const raw = localStorage.getItem(storageKey)
    if (!raw) return fallbackKeys
    const parsed = JSON.parse(raw)
    const allowed = new Set(fallbackKeys)
    if (Array.isArray(parsed)) {
      const list = [...new Set(parsed.filter((p): p is string => typeof p === 'string' && allowed.has(p)))]
      return list.length > 0 ? list : fallbackKeys
    }
    if (parsed && typeof parsed === 'object') {
      const data = parsed as { visible?: unknown; known?: unknown }
      const visible = Array.isArray(data.visible)
        ? [...new Set(data.visible.filter((p): p is string => typeof p === 'string' && allowed.has(p)))]
        : []
      const known = Array.isArray(data.known)
        ? new Set(data.known.filter((p): p is string => typeof p === 'string'))
        : new Set<string>()
      const newKeys = fallbackKeys.filter(p => !known.has(p))
      const list = [...new Set([...visible, ...newKeys])]
      return list.length > 0 ? list : fallbackKeys
    }
    return fallbackKeys
  } catch {
    return fallbackKeys
  }
}

const visibleRegPlatformKeys = ref<string[]>(loadVisiblePlatformKeys(PLATFORM_DISPLAY_KEYS.reg, allRegPlatformOptions.value))
const visibleVerifyPlatformKeys = ref<string[]>(loadVisiblePlatformKeys(PLATFORM_DISPLAY_KEYS.verify, allVerifyPlatformOptions.value))
const platformDisplayMenu = ref<'reg' | 'verify' | null>(null)

watch(visibleRegPlatformKeys, (v) => {
  localStorage.setItem(PLATFORM_DISPLAY_KEYS.reg, JSON.stringify({
    visible: v,
    known: allRegPlatformOptions.value.map(p => p.key),
  }))
}, { deep: true })
watch(visibleVerifyPlatformKeys, (v) => {
  localStorage.setItem(PLATFORM_DISPLAY_KEYS.verify, JSON.stringify({
    visible: v,
    known: allVerifyPlatformOptions.value.map(p => p.key),
  }))
}, { deep: true })

// Filter 2 tầng: (1) user-toggled visibility, (2) globally hidden (DISABLED_PLATFORM_KEYS).
// Platform trong DISABLED_PLATFORM_KEYS bị ẨN HOÀN TOÀN — không hiển thị button, không
// trong popup config visibility. Để kích hoạt lại 1 platform: xoá khỏi DISABLED_PLATFORM_KEYS.
const visibleRegPlatformsStd = computed(() => REG_PLATFORMS_STD.filter(p => visibleRegPlatformKeys.value.includes(p.key) && !DISABLED_PLATFORM_KEYS.has(p.key)))
const visibleRegPlatformsVer = computed(() => REG_PLATFORMS_VER.filter(p => visibleRegPlatformKeys.value.includes(p.key) && !DISABLED_PLATFORM_KEYS.has(p.key)).slice(0, 1))
const visibleRegPlatformsIos = computed(() => REG_PLATFORMS_IOS.filter(p => visibleRegPlatformKeys.value.includes(p.key) && !DISABLED_PLATFORM_KEYS.has(p.key)).slice(0, 1))
const visibleVerifyPlatformsAndr = computed(() => VER_PLATFORMS_ANDR.filter(p => visibleVerifyPlatformKeys.value.includes(p.key) && !DISABLED_PLATFORM_KEYS.has(p.key)))
const visibleVerifyPlatformsMfb = computed(() => VER_PLATFORMS_MFB.filter(p => visibleVerifyPlatformKeys.value.includes(p.key) && !DISABLED_PLATFORM_KEYS.has(p.key)))
const visibleVerifyPlatformsVer = computed(() => VER_PLATFORMS_VER.filter(p => visibleVerifyPlatformKeys.value.includes(p.key) && !DISABLED_PLATFORM_KEYS.has(p.key)).slice(0, 1))
const visibleVerifyPlatformsIos = computed(() => VER_PLATFORMS_IOS.filter(p => visibleVerifyPlatformKeys.value.includes(p.key) && !DISABLED_PLATFORM_KEYS.has(p.key)).slice(0, 1))
const visibleRegPlatformsMessAndr = computed(() => REG_PLATFORMS_MESS_ANDR.filter(p => visibleRegPlatformKeys.value.includes(p.key) && !DISABLED_PLATFORM_KEYS.has(p.key)).slice(0, 1))
const visibleRegPlatformsMessIos = computed(() => REG_PLATFORMS_MESS_IOS.filter(p => visibleRegPlatformKeys.value.includes(p.key) && !DISABLED_PLATFORM_KEYS.has(p.key)).slice(0, 1))
const visibleVerifyPlatformsMessAndr = computed(() => VER_PLATFORMS_MESS_ANDR.filter(p => visibleVerifyPlatformKeys.value.includes(p.key) && !DISABLED_PLATFORM_KEYS.has(p.key)).slice(0, 1))
const visibleVerifyPlatformsMessIos = computed(() => VER_PLATFORMS_MESS_IOS.filter(p => visibleVerifyPlatformKeys.value.includes(p.key) && !DISABLED_PLATFORM_KEYS.has(p.key)).slice(0, 1))

function togglePlatformDisplayMenu(kind: 'reg' | 'verify') {
  platformDisplayMenu.value = platformDisplayMenu.value === kind ? null : kind
}

function closePlatformDisplayMenu() {
  platformDisplayMenu.value = null
}

function toggleVisiblePlatform(kind: 'reg' | 'verify', platform: string) {
  const target = kind === 'reg' ? visibleRegPlatformKeys : visibleVerifyPlatformKeys
  const cur = new Set(target.value)
  if (cur.has(platform)) cur.delete(platform)
  else cur.add(platform)
  target.value = [...cur]
}

function selectAllVisiblePlatforms(kind: 'reg' | 'verify') {
  if (kind === 'reg') visibleRegPlatformKeys.value = allRegPlatformOptions.value.map(p => p.key)
  else visibleVerifyPlatformKeys.value = allVerifyPlatformOptions.value.map(p => p.key)
}

function clearVisiblePlatforms(kind: 'reg' | 'verify') {
  if (kind === 'reg') visibleRegPlatformKeys.value = []
  else visibleVerifyPlatformKeys.value = []
}

function addVisiblePlatforms(kind: 'reg' | 'verify', platforms: string[]) {
  const target = kind === 'reg' ? visibleRegPlatformKeys : visibleVerifyPlatformKeys
  target.value = [...new Set([...target.value, ...platforms])]
}

function toggleVisiblePlatforms(kind: 'reg' | 'verify', options: PlatformOption[], on: boolean) {
  const target = kind === 'reg' ? visibleRegPlatformKeys : visibleVerifyPlatformKeys
  const next = new Set(target.value)
  for (const p of options) {
    if (on) next.add(p.key)
    else next.delete(p.key)
  }
  target.value = [...next]
}

function visiblePlatformCount(kind: 'reg' | 'verify') {
  return kind === 'reg' ? visibleRegPlatformKeys.value.length : visibleVerifyPlatformKeys.value.length
}

type PlatformSelectionClipboard = {
  kind?: string
  version?: number
  reg?: string[]
  verify?: string[]
  apiRegPlatforms?: string[]
  apiVerifyPlatforms?: string[]
  platforms?: string[]
}

const verifyClipboardAliases: Record<string, string> = {
  s23: 'api android',
  web: 'api mfb',
  webandroid: 'api web andr',
  android: 'api token',
}

function normalizePlatformList(value: unknown, allowed: Set<string>, aliases: Record<string, string> = {}): string[] {
  if (!Array.isArray(value)) return []
  return [...new Set(value
    .filter((p): p is string => typeof p === 'string')
    .map(p => p.trim())
    .map(p => aliases[p] ?? p)
    .filter(p => allowed.has(p) && isPlatformSelectable(p)))]
}

function readPlatformClipboardList(payload: unknown, target: 'reg' | 'verify'): string[] {
  const allowed = target === 'reg' ? regPlatformKeys.value : verifyPlatformKeys.value
  const aliases = target === 'verify' ? verifyClipboardAliases : {}
  if (Array.isArray(payload)) return normalizePlatformList(payload, allowed, aliases)
  if (!payload || typeof payload !== 'object') return []
  const data = payload as PlatformSelectionClipboard
  const candidates = target === 'reg'
    ? [data.reg, data.apiRegPlatforms, data.platforms]
    : [data.verify, data.apiVerifyPlatforms, data.platforms]
  for (const candidate of candidates) {
    const list = normalizePlatformList(candidate, allowed, aliases)
    if (list.length > 0) return list
  }
  return []
}

async function readPlatformSelectionFromClipboard(target: 'reg' | 'verify'): Promise<string[]> {
  try {
    const raw = await navigator.clipboard.readText()
    const payload = JSON.parse(raw)
    return readPlatformClipboardList(payload, target)
  } catch {
    appStore.notify('error', 'Clipboard không phải JSON chọn version hợp lệ')
    return []
  }
}

// ── UA Config inline per-platform (direct binding, no popup) ─────────────────

// Computed ref trỏ thẳng vào PlatformUAConfig của platform đang chọn.
// Mutations trên object này tự động ghi vào form.regPlatformUA / verifyPlatformUA.
const regPlatformCfg = computed<PlatformUAConfig>(() => {
  const p = form.value.apiRegPlatform
  // Init platform config nếu chưa có — đảm bảo v-model mutation PERSIST vào form state.
  if (!form.value.regPlatformUA[p]) {
    form.value.regPlatformUA[p] = { useOriginalUA: false, addVirtualSpecAndroid: false, buildUA: true, replaceCarrier: true, trackingID: false, uaPoolKey: getAllowedPoolForPlatform(p) }
  }
  return form.value.regPlatformUA[p]
})

const verPlatformCfg = computed<PlatformUAConfig>(() => {
  const p = form.value.apiVerifyPlatform
  // Init platform config nếu chưa có — đảm bảo v-model mutation (checkbox tick/untick)
  // PERSIST vào form state thay vì set vào object tạm thời rồi mất khi computed re-evaluate.
  if (!form.value.verifyPlatformUA[p]) {
    form.value.verifyPlatformUA[p] = { useOriginalUA: false, addVirtualSpecAndroid: false, buildUA: true, replaceCarrier: true, trackingID: false, uaPoolKey: getAllowedPoolForPlatform(p) }
  }
  return form.value.verifyPlatformUA[p]
})

// UA Gốc Messenger: build theo version (FBAV/FBBV từ capture), device cố định SM-G996B / iPhone15,2.
const MESS_ANDR_VER_FBAV: Record<string, string> = {
  '530': '530.1.0.67.107|814020040', '535': '535.0.0.101.107|840054075',
  '545': '545.0.0.27.62|870175947', '555': '555.0.0.56.66|930834402',
  '563': '563.0.0.47.86|979328543', '564': '564.0.0.42.89|984961990',
  '565': '565.0.0.0.2|981799924', '525': '525.0.0.44.108|792260954',
  '515': '515.0.0.51.108|763707183', '505': '505.0.0.62.82|730961636',
  '490': '490.0.0.42.108|684080902',
}
const MESS_IOS_UAGOC = 'LightSpeed [FBAN/MessengerLiteForiOS;FBAV/563.0.0.27.106;FBBV/980221516;FBDV/iPhone15,2;FBMD/iPhone;FBSN/iOS;FBSV/16.7;FBSS/3;FBCR/;FBID/phone;FBLC/en_US;FBOP/0]'
function messOriginalUA(platform: string): string {
  if (MESS_IOS_KEY_SET.has(platform)) return MESS_IOS_UAGOC
  if (!MESS_ANDR_KEY_SET.has(platform)) return ''
  const m = platform.match(/(\d{3})$/)
  const ver = m ? m[1] : '530' // appmv3reg / appmessv3 (không số) = 530
  const fb = MESS_ANDR_VER_FBAV[ver]
  if (!fb) return ''
  const [fbav, fbbv] = fb.split('|')
  return `Dalvik/2.1.0 (Linux; U; Android 15; SM-G996B Build/AP3A.240905.015.A2) [FBAN/Orca-Android;FBAV/${fbav};FBPN/com.facebook.orca;FBLC/en_GB;FBBV/${fbbv};FBCR/Viettel;FBMF/samsung;FBBD/samsung;FBDV/SM-G996B;FBSV/15;FBCA/arm64-v8a:null;FBDM/{density=2.8125,width=1080,height=2400};FB_FW/1;]`
}
const regOriginalUA = computed(() => ORIGINAL_UA_STRINGS[form.value.apiRegPlatform] ?? messOriginalUA(form.value.apiRegPlatform))
const verOriginalUA = computed(() => ORIGINAL_UA_STRINGS[form.value.apiVerifyPlatform] ?? messOriginalUA(form.value.apiVerifyPlatform))

// Virtual spec chỉ hiện với non-versioned Android platforms hoặc Fb_XXX có số < 400.
// iOS platforms không dùng Virtual spec (FBIOS không có Dalvik prefix).
function platformShowsVirtualSpec(platform: string): boolean {
  if (IOS_PLATFORM_KEY_SET.has(platform)) return false
  const m = platform.match(/^s(\d+)$/)
  if (!m) return true
  return parseInt(m[1]) < 400
}
const regShowVirtualSpec = computed(() => platformShowsVirtualSpec(form.value.apiRegPlatform))
const verShowVirtualSpec = computed(() => platformShowsVirtualSpec(form.value.apiVerifyPlatform))

const regAllowedPool = computed(() => getAllowedPoolForPlatform(form.value.apiRegPlatform))
const verAllowedPool = computed(() => getAllowedPoolForPlatform(form.value.apiVerifyPlatform))


// Test UA simulation
const regTestLoading = ref(false)
const regTestNote = ref('')
const regTestUA = ref('')
const verTestLoading = ref(false)
const verTestNote = ref('')
const verTestUA = ref('')

watch(() => form.value.apiRegPlatform, (p) => {
  regTestNote.value = ''; regTestUA.value = ''
  const cfg = form.value.regPlatformUA[p]
  if (cfg) cfg.uaPoolKey = getAllowedPoolForPlatform(p)
})
watch(() => form.value.apiVerifyPlatform, (p) => {
  verTestNote.value = ''; verTestUA.value = ''
  const cfg = form.value.verifyPlatformUA[p]
  if (cfg) cfg.uaPoolKey = getAllowedPoolForPlatform(p)
})

function applyBuildUADefault(map: Record<string, PlatformUAConfig>, platform: string) {
  // iOS Native App — default sang iOS pool + KHÔNG bật BuildUA mặc định.
  // User mong muốn iOS không tick gì → đọc từ iOS_UG.txt pool (UAKindIOS).
  const isIOS = IOS_PLATFORM_KEY_SET.has(platform)

  if (!map[platform]) {
    if (isIOS) {
      map[platform] = {
        useOriginalUA: false,
        addVirtualSpecAndroid: false,
        buildUA: false,            // ← KHÁC Android: không auto BuildUA, fall through pool path
        replaceCarrier: true,
        trackingID: false,
        uaPoolKey: 'iphone',       // ← KHÁC Android: pool iOS
      }
    } else {
      map[platform] = {
        useOriginalUA: false,
        addVirtualSpecAndroid: false,
        buildUA: true,
        replaceCarrier: true,
        trackingID: false,
        uaPoolKey: getAllowedPoolForPlatform(platform),
      }
    }
  } else if (!map[platform].useOriginalUA && !map[platform].buildUA) {
    // Existing entry — nếu cả 2 đều off (= dùng pool), giữ nguyên.
    // Chỉ auto-bật BuildUA cho non-iOS để tránh accident "all off" cho Android (sẽ broken).
    if (!isIOS) {
      map[platform].buildUA = true
    }
  }
}

function isRegPlatformOn(platform: string): boolean {
  return (form.value.apiRegPlatforms ?? []).includes(platform)
}

// Toggle 1 version vào/ra danh sách multi-reg. Cho phép bỏ hết (rỗng) — khi rỗng
// backend tự fallback dùng apiRegPlatform. Version vừa bật → "focus" (UA panel áp cho nó).
function toggleRegPlatform(platform: string) {
  if (!isPlatformSelectable(platform)) return
  const cur = form.value.apiRegPlatforms ?? []
  if (cur.includes(platform)) {
    const next = cur.filter(p => p !== platform)
    if (form.value.apiRegPlatform === platform && next.length > 0) form.value.apiRegPlatform = next[0]
    form.value.apiRegPlatforms = next
  } else {
    form.value.apiRegPlatforms = [...cur, platform]
    form.value.apiRegPlatform = platform
    applyBuildUADefault(form.value.regPlatformUA, platform)
    // iOS migration: nếu entry đã tồn tại từ config cũ với uaPoolKey='android',
    // auto-switch sang 'iphone' vì iOS không hợp lý dùng pool Android.
    // User vẫn có thể override sau bằng cách click pool button.
    const allowedPool = getAllowedPoolForPlatform(platform)
    if (form.value.regPlatformUA?.[platform]?.uaPoolKey !== allowedPool) {
      form.value.regPlatformUA[platform].uaPoolKey = allowedPool
    }
  }
  void saveNow()
}

// ── Context menu chuột phải trên vùng nút API REG ───────────────────────────
const regPlatformMenu = ref<{ x: number; y: number } | null>(null)
function openRegPlatformMenu(e: MouseEvent) {
  e.preventDefault()
  regPlatformMenu.value = { x: e.clientX, y: e.clientY }
}
function closeRegPlatformMenu() { regPlatformMenu.value = null }
function selectRegPlatforms(platforms: string[]) {
  const next = [...new Set(platforms.filter(p => p && isPlatformSelectable(p)))]
  form.value.apiRegPlatforms = next
  if (next.length > 0) {
    form.value.apiRegPlatform = next[0]
    for (const p of next) applyBuildUADefault(form.value.regPlatformUA, p)
  }
  closeRegPlatformMenu()
  void saveNow()
}
function addRegPlatforms(platforms: string[]) {
  selectRegPlatforms([...(form.value.apiRegPlatforms ?? []), ...platforms.filter(isPlatformSelectable)])
}
function selectAllRegPlatforms() {
  selectRegPlatforms([...visibleRegPlatformsStd.value, ...visibleRegPlatformsVer.value, ...visibleRegPlatformsMessAndr.value, ...visibleRegPlatformsIos.value, ...visibleRegPlatformsMessIos.value].map(p => p.key))
}
function selectRegVersionBatch(batch: PlatformOption[]) {
  addRegPlatforms(batch.map(p => p.key))
}
async function pasteRegPlatforms() {
  const list = await readPlatformSelectionFromClipboard('reg')
  if (list.length === 0) return
  addVisiblePlatforms('reg', list)
  selectRegPlatforms(list)
  appStore.notify('success', `Đã chọn lại ${list.length} version REG từ JSON`)
}
function clearRegPlatforms() {
  form.value.apiRegPlatforms = []
  closeRegPlatformMenu()
  void saveNow()
}

// Đồng bộ apiRegPlatforms sau mỗi lần load config (field mới — config/profile cũ chưa có).
function ensureRegPlatformsForm() {
  let list = Array.isArray(form.value.apiRegPlatforms)
    ? form.value.apiRegPlatforms.filter(p => typeof p === 'string' && p.trim())
    : []
  list = list.filter(isPlatformSelectable)
  // Filter ra các key không còn tồn tại trong UI (vd platform đã xóa/rename).
  const knownKeys = regPlatformKeys.value
  list = list.filter(p => knownKeys.has(p))
  if (list.length === 0 && form.value.apiRegPlatform && knownKeys.has(form.value.apiRegPlatform)) {
    list = [form.value.apiRegPlatform]
  }
  list = [...new Set(list)]
  form.value.apiRegPlatforms = list
  if (list.length > 0 && !list.includes(form.value.apiRegPlatform)) {
    form.value.apiRegPlatform = list[0]
  }
}

function onRegUAOptionChanged() {
  if (regPlatformCfg.value.useOriginalUA) {
    regPlatformCfg.value.addVirtualSpecAndroid = false
    regPlatformCfg.value.buildUA = false
  }
  void saveNow()
}

function onRegVirtualSpecChanged() {
  if (regPlatformCfg.value.addVirtualSpecAndroid) regPlatformCfg.value.useOriginalUA = false
  void saveNow()
}

function onRegBuildUAChanged() {
  if (regPlatformCfg.value.buildUA) regPlatformCfg.value.useOriginalUA = false
  void saveNow()
}

function isVerifyPlatformOn(platform: string): boolean {
  return (form.value.apiVerifyPlatforms ?? []).includes(platform)
}

// Toggle 1 version vào/ra danh sách multi-verify. Cho phép bỏ hết (rỗng) — khi rỗng
// backend tự fallback dùng apiVerifyPlatform. Version vừa bật → "focus" (UA panel áp cho nó).
function toggleVerifyPlatform(platform: string) {
  if (!isPlatformSelectable(platform)) return
  const cur = form.value.apiVerifyPlatforms ?? []
  if (cur.includes(platform)) {
    const next = cur.filter(p => p !== platform)
    if (form.value.apiVerifyPlatform === platform && next.length > 0) form.value.apiVerifyPlatform = next[0]
    form.value.apiVerifyPlatforms = next
  } else {
    form.value.apiVerifyPlatforms = [...cur, platform]
    form.value.apiVerifyPlatform = platform
    applyBuildUADefault(form.value.verifyPlatformUA, platform)
    // iOS migration: entry cũ uaPoolKey='android' → auto sang allowed pool (iphone cho Mess iOS).
    const allowedPool = getAllowedPoolForPlatform(platform)
    if (form.value.verifyPlatformUA?.[platform]?.uaPoolKey !== allowedPool) {
      form.value.verifyPlatformUA[platform].uaPoolKey = allowedPool
    }
  }
  void saveNow()
}

// Context menu chuột phải trên vùng nút API VERIFY.
const verifyPlatformMenu = ref<{ x: number; y: number } | null>(null)
function openVerifyPlatformMenu(e: MouseEvent) {
  e.preventDefault()
  verifyPlatformMenu.value = { x: e.clientX, y: e.clientY }
}
function closeVerifyPlatformMenu() { verifyPlatformMenu.value = null }
function selectVerifyPlatforms(platforms: string[]) {
  const next = [...new Set(platforms.filter(p => p && isPlatformSelectable(p)))]
  form.value.apiVerifyPlatforms = next
  if (next.length > 0) {
    form.value.apiVerifyPlatform = next[0]
    for (const p of next) applyBuildUADefault(form.value.verifyPlatformUA, p)
  }
  closeVerifyPlatformMenu()
  void saveNow()
}
function addVerifyPlatforms(platforms: string[]) {
  selectVerifyPlatforms([...(form.value.apiVerifyPlatforms ?? []), ...platforms.filter(isPlatformSelectable)])
}
function selectAllVerifyPlatforms() {
  selectVerifyPlatforms([...visibleVerifyPlatformsAndr.value, ...visibleVerifyPlatformsMfb.value, ...visibleVerifyPlatformsVer.value, ...visibleVerifyPlatformsMessAndr.value, ...visibleVerifyPlatformsIos.value, ...visibleVerifyPlatformsMessIos.value].map(p => p.key))
}
function selectVerifyVersionBatch(batch: PlatformOption[]) {
  addVerifyPlatforms(batch.map(p => p.key))
}
async function pasteVerifyPlatforms() {
  const list = await readPlatformSelectionFromClipboard('verify')
  if (list.length === 0) return
  addVisiblePlatforms('verify', list)
  selectVerifyPlatforms(list)
  appStore.notify('success', `Đã chọn lại ${list.length} version VERIFY từ JSON`)
}
function clearVerifyPlatforms() {
  form.value.apiVerifyPlatforms = []
  closeVerifyPlatformMenu()
  void saveNow()
}

// ── Marquee: kéo chuột quét chọn nhiều version (giống chọn file Explorer) ────
// Kéo = thêm vào lựa chọn; Alt + kéo = bỏ chọn các version trong khung.
function onRegMarqueeCommit(keys: string[], mode: 'add' | 'remove') {
  if (keys.length === 0) return
  if (mode === 'remove') {
    const rm = new Set(keys)
    selectRegPlatforms((form.value.apiRegPlatforms ?? []).filter(k => !rm.has(k)))
  } else {
    addRegPlatforms(keys)
  }
}
function onVerMarqueeCommit(keys: string[], mode: 'add' | 'remove') {
  if (keys.length === 0) return
  if (mode === 'remove') {
    const rm = new Set(keys)
    selectVerifyPlatforms((form.value.apiVerifyPlatforms ?? []).filter(k => !rm.has(k)))
  } else {
    addVerifyPlatforms(keys)
  }
}
const regMarquee = useMarqueeSelect({ onCommit: onRegMarqueeCommit })
const verMarquee = useMarqueeSelect({ onCommit: onVerMarqueeCommit })

// Đồng bộ apiVerifyPlatforms sau mỗi lần load config (field mới — config/profile cũ chưa có).
function ensureVerifyPlatformsForm() {
  let list = Array.isArray(form.value.apiVerifyPlatforms)
    ? form.value.apiVerifyPlatforms.filter(p => typeof p === 'string' && p.trim())
    : []
  list = list.filter(isPlatformSelectable)
  // Filter ra các key không còn tồn tại trong UI (vd platform đã xóa "api mfb cũ").
  // Nếu không filter, counter sẽ đếm lệch (3 thay vì 2) vì stale config saved cũ.
  const knownKeys = verifyPlatformKeys.value
  list = list.filter(p => knownKeys.has(p))
  if (list.length === 0 && form.value.apiVerifyPlatform && knownKeys.has(form.value.apiVerifyPlatform)) {
    list = [form.value.apiVerifyPlatform]
  }
  list = [...new Set(list)]
  form.value.apiVerifyPlatforms = list
  if (list.length > 0 && !list.includes(form.value.apiVerifyPlatform)) {
    form.value.apiVerifyPlatform = list[0]
  }
}

function onVerUAOptionChanged() {
  if (verPlatformCfg.value.useOriginalUA) {
    verPlatformCfg.value.addVirtualSpecAndroid = false
    verPlatformCfg.value.buildUA = false
  }
  void saveNow()
}

function onVerVirtualSpecChanged() {
  if (verPlatformCfg.value.addVirtualSpecAndroid) verPlatformCfg.value.useOriginalUA = false
  void saveNow()
}

function onVerBuildUAChanged() {
  if (verPlatformCfg.value.buildUA) verPlatformCfg.value.useOriginalUA = false
  void saveNow()
}

async function runRegUATest() {
  regTestLoading.value = true; regTestNote.value = ''; regTestUA.value = ''
  try {
    const svc = await getVerifyRunnerService()
    const result: string = await svc.simulatePlatformUA(form.value.apiRegPlatform, { ...regPlatformCfg.value, trackingID: form.value.trackingIDReg, kind: 'reg' })
    const nl = result.indexOf('\n')
    if (nl !== -1) { regTestNote.value = result.slice(0, nl); regTestUA.value = result.slice(nl + 1) }
    else { regTestUA.value = result }
  } catch (e) { regTestUA.value = String(e) }
  finally { regTestLoading.value = false }
}

async function runVerUATest() {
  verTestLoading.value = true; verTestNote.value = ''; verTestUA.value = ''
  try {
    const svc = await getVerifyRunnerService()
    const result: string = await svc.simulatePlatformUA(form.value.apiVerifyPlatform, { ...verPlatformCfg.value, trackingID: form.value.trackingIDVer, kind: 'ver' })
    const nl = result.indexOf('\n')
    if (nl !== -1) { verTestNote.value = result.slice(0, nl); verTestUA.value = result.slice(nl + 1) }
    else { verTestUA.value = result }
  } catch (e) { verTestUA.value = String(e) }
  finally { verTestLoading.value = false }
}

// Mở file build version trong text editor mặc định của OS.
// iOS platform → Config/DeviceInfoIOS/ios_app_builds.txt
// Android      → Config/DeviceInfo/versions_and_builds_<kind>.txt
async function openVersionsFile(kind: 'reg' | 'ver', platform: string) {
  try {
    const { OpenVersionsAndBuildsFile } = await import('../../wailsjs/go/app/App')
    const fileKind = IOS_PLATFORM_KEY_SET.has(platform) ? `${kind}-ios` : kind
    const result: string = await OpenVersionsAndBuildsFile(fileKind)
    if (result.startsWith('ERR|')) {
      alert('Không mở được file: ' + result.slice(4))
    }
  } catch (e) {
    alert('Lỗi mở file: ' + String(e))
  }
}

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
  { value: 'temp-mail.org', label: 'Temp-Mail.org' },
  //{ value: 'mail.cx',       label: 'Mail.cx' },   // ẩn
  { value: 'mailtd',        label: 'Mail.cx' },
  //{ value: 'inboxes',       label: 'Inboxes.com' },        // ẩn
  { value: 'dismail',       label: 'Dismail.top' },
  { value: 'mailymg',       label: 'Mailymg.com' },
  { value: 'altmails',      label: 'AltMails.com' },
  //{ value: 'onesecmail',   label: '1secmail.com' },        // ẩn
  { value: 'firetempmail', label: 'FireTempMail.com' },
  //{ value: 'fviainboxes',  label: 'FviaInboxes.com' },     // ẩn
  { value: 'byomde',       label: 'Byom.de' },
  { value: 'dinlaan',      label: 'Dinlaan.com' },
  { value: 'cryptogmail',  label: 'CryptoGmail.com' },
  { value: 'buslink24',    label: 'Buslink24.com' },
  { value: 'boxmailstore', label: 'BoxMail.store' },
  { value: 'mailermnx',   label: 'Mailer.mnx-family.com' },
  { value: 'tempforward',  label: 'TempForward.com' },
  { value: 'tempomintraccoon', label: 'Tempo.Mintraccoon.com' },
  { value: 'tempemail',    label: 'TempEmail.co' },
  { value: 'tmpinbox',     label: 'TmpInbox.com' },
  { value: 'tenminutemail', label: '10MinuteMail.com' },
  { value: 'tempmailto',   label: 'TempMailTo.com' },
  { value: 'onesecemail',  label: '1secemail.com' },
  { value: 'tempmail100',  label: 'TempMail100.com' },
  { value: 'tempmailso',   label: 'TempMail.so' },
  { value: 'priyoemail',   label: 'Priyo.email (cần API key)' },
  { value: 'tempmailorgpremium', label: 'Temp-Mail.org Premium' },
  { value: 'mailtempcom',  label: 'Mail-Temp.com' },
  //{ value: 'wemakemail',   label: 'WeMakeMail (cần API key)' }, // ẩn
  { value: 'mailhv',       label: 'MailHV (cần API key)' },
]
const RENT_MAIL_PROVIDERS: { value: MailProviderType; label: string }[] = [
  { value: 'zeus-x',        label: 'ZeusX' },
  { value: 'dongvanfb',     label: 'DongVanFB' },
  { value: 'store1s',       label: 'Store1s' },
  { value: 'mail30s',       label: 'Mail30s' },
  { value: 'muamail',       label: 'MuaMail' },
  { value: 'unlimitmail',   label: 'UnlimitMail' },
  { value: 'sptmail',       label: 'SPTMail' },
  { value: 'emailapiinfo',  label: 'EmailAPI.info' },
  { value: 'otpcheap',      label: 'OTP.cheap' },
  { value: 'shopgmail9999', label: 'ShopGmail9999' },
  { value: 'rentgmail',     label: 'RentGmail.online' },
  { value: 'otpcodesms',    label: 'OtpCodesSms.site' },
  { value: 'wmemail',       label: 'Wmemail.com' },
]

const mailCategory = computed<'temp' | 'rent'>(() => {
  if (RENT_MAIL_PROVIDERS.some(p => p.value === form.value.mailProvider)) return 'rent'
  return 'temp'
})

function selectMailCategory(cat: 'temp' | 'rent') {
  if (cat === 'temp') form.value.mailProvider = 'moakt'
  else if (cat === 'rent') form.value.mailProvider = 'zeus-x'
  // Force save ngay — user đổi category cần propagate tức thì để worker đang chạy picks up.
  void saveNow()
}

// Force save ngay khi user click provider mail — bypass debounce 500ms.
// Worker verify sẽ reload config ở account KẾ TIẾP, đổi provider có hiệu lực trong vài giây.
function selectMailProvider(value: string) {
  form.value.mailProvider = value as any
  // Sync tempMailDomain + tempMailToken về slot của provider vừa chọn — backend đọc field này cho mỗi call.
  const dMap = form.value.tempMailDomains || {}
  form.value.tempMailDomain = dMap[value] || ''
  const tMap = form.value.tempMailTokens || {}
  form.value.tempMailToken = tMap[value] || ''
  void saveNow()
}

// ─── Phone Auth (preview — UI-only, chưa wire backend) ───────────────────────
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

// Temp mail providers that support custom domain input (moakt, mail1sec, tempmail-plus)
const tempMailHasDomain = computed(() =>
  form.value.mailProvider === 'moakt' ||
  form.value.mailProvider === '@i2b.vn' ||
  form.value.mailProvider === 'tempmail-plus' ||
  form.value.mailProvider === 'wemakemail' ||
  form.value.mailProvider === 'mailhv' ||
  form.value.mailProvider === 'mailtd'
)

// Per-provider domain: mỗi provider giữ domain riêng trong tempMailDomains map.
// Khi user gõ → ghi vào slot của provider hiện hành + đồng bộ tempMailDomain (backend đọc field này).
// Khi đổi provider → input tự reflect domain của provider mới.
// Format: dấu phẩy ngăn cách. Migration: nếu raw value có newline → auto convert sang "a, b, c".
function normalizeDomainList(raw: string): string {
  if (!raw) return ''
  // Split theo cả newline và dấu phẩy → rejoin bằng ", ".
  return raw.split(/[\r\n,]+/).map(s => s.trim()).filter(Boolean).join(', ')
}
const currentTempMailDomain = computed<string>({
  get() {
    const p = form.value.mailProvider
    const map = form.value.tempMailDomains || {}
    const raw = map[p] !== undefined ? map[p] : (form.value.tempMailDomain || '')
    return normalizeDomainList(raw)
  },
  set(val: string) {
    const p = form.value.mailProvider
    if (!form.value.tempMailDomains) form.value.tempMailDomains = {}
    form.value.tempMailDomains[p] = val
    form.value.tempMailDomain = val
  },
})
const currentDomainPlaceholder = computed(() => {
  const p = form.value.mailProvider
  if (p === '@i2b.vn') return 'i2b.vn, other.net'
  if (p === 'moakt') return 'tmpbox.net, other.net'
  if (p === 'tempmail-plus') return 'mailto.plus, fexpost.com, fexbox.org'
  if (p === 'wemakemail') return 'Để trống = API tự chọn | hoặc: domain1.com, domain2.com'
  if (p === 'mailhv') return 'Để trống = API tự chọn | hoặc: domain1.com, domain2.com'
  if (p === 'mailtd') return 'Để trống = random tất cả | hoặc: sugtbt.com, qabq.com, nqmo.com, end.tw, uuf.me, 6n9.net'
  return 'domain1.com, domain2.com'
})

// Per-provider token — giống domain map. Dùng cho provider cần token tự nhập
// (tempmail-lol, priyo.email, hoặc provider tương lai không auto-fetch token).
const currentTempMailToken = computed<string>({
  get() {
    const p = form.value.mailProvider
    const map = form.value.tempMailTokens || {}
    if (map[p] !== undefined) return map[p]
    return form.value.tempMailToken || ''
  },
  set(val: string) {
    const p = form.value.mailProvider
    if (!form.value.tempMailTokens) form.value.tempMailTokens = {}
    form.value.tempMailTokens[p] = val
    form.value.tempMailToken = val
  },
})
const currentProviderLabel = computed(() => {
  const p = form.value.mailProvider
  return TEMP_MAIL_PROVIDERS.find(x => x.value === p)?.label
    || RENT_MAIL_PROVIDERS.find(x => x.value === p)?.label
    || p
})
const isManualMailMode = computed(() => false)

// ─── Auto-save (port C#: không có nút Lưu, mọi thay đổi tự persist) ──────────
// Watch form deep → debounce 500ms → call save API. Worker đang chạy reload config
// qua LoadInteractionConfig realtime ở backend — user đổi thông số giữa chừng có
// hiệu lực ngay cho account kế tiếp.
const { status: saveStatus, saveNow } = useAutoSave(form, async (value) => {
  if (value.timeDelayCheck < 1) value.timeDelayCheck = 1
  if (value.timeDelaySendCode < 1) value.timeDelaySendCode = 1
  // splitMode chỉ có nghĩa khi bật cả Register + Verify.
  // Nếu user tắt Verify → force splitMode=false, tránh AccountsPage vẫn hiển thị
  // layout split + panel VERIFY rỗng khi chỉ chạy reg.
  if (!value.createEnabled || !value.verifyEnabled) value.splitMode = false
  // splitVerifyThreads: 0 = bằng số luồng reg. Khác 0 = clamp [1, 600].
  if (value.splitVerifyThreads < 0) value.splitVerifyThreads = 0
  if (value.splitVerifyThreads > 600) value.splitVerifyThreads = 600
  const interactionSvc = await getInteractionService()
  const r = await interactionSvc.save(value)
  if (r !== 'OK') throw new Error(r)
})

// Clamp helper cho input verify threads — gọi từ @blur/@change của input.
function clampSplitVerifyThreads() {
  const v = Number(form.value.splitVerifyThreads)
  if (isNaN(v) || v < 0) form.value.splitVerifyThreads = 0
  else if (v > 600) form.value.splitVerifyThreads = 600
  else form.value.splitVerifyThreads = Math.floor(v)
}

// Clamp helper cho input reg threads — gọi từ @blur/@change của input.
function clampRegThreads() {
  const v = Number(form.value.regThreads)
  if (isNaN(v) || v < 1) form.value.regThreads = 1
  else if (v > 600) form.value.regThreads = 600
  else form.value.regThreads = Math.floor(v)
}

// Backward-compat shim: các chỗ trong file (ProfileManager @update, handleImport) vẫn gọi
// handleSave() — giữ hàm để không phá flow cũ, chỉ delegate sang autoSave.saveNow.
async function handleSave() {
  // Validate output path (chỉ khi user pick manual) — không block auto-save,
  // chỉ show notify nếu sai.
  await validateOutputPath(form.value.outputPath)
  if (pathError.value) {
    appStore.notify('error', 'Thư mục verify: ' + pathError.value)
  }
  // Trigger save ngay bằng cách edit form → watch fires, debounce chạy.
  // Không cần gọi API trực tiếp vì watch sẽ handle.
}

function resetForms() {
  form.value = { ...DEFAULT_VERIFY_CONFIG }
  reloadForm()
}
</script>

<template>
  <div class="rp-page">

    <!-- ── Toolbar ──────────────────────────────────────────────────────────── -->
    <div class="rp-toolbar">
      <h2 class="rp-toolbar__title">Thiết lập chạy</h2>
      <div class="rp-mode-checks rp-mode-checks--toolbar">
        <label class="rp-mode-check">
          <input type="checkbox" v-model="form.createEnabled" />
          <span>Register</span>
        </label>
        <label class="rp-mode-check">
          <input type="checkbox" v-model="form.verifyEnabled" />
          <span>Verify</span>
        </label>
        <label v-if="form.createEnabled && form.verifyEnabled" class="rp-mode-check rp-mode-check--split">
          <input type="checkbox" v-model="form.splitMode" />
          <span title="Reg và Verify chạy độc lập: reg ghi file → verify tự đọc file đó">Split</span>
        </label>
        <label class="rp-mode-check">
          <input type="checkbox" v-model="form.keepIpSuccess" />
          <span title="Sau khi 1 account reg/verify Live, giữ nguyên IP của slot cho account kế. Fail → IP mới.">Keep IP</span>
        </label>
        <label class="rp-mode-check">
          <input type="checkbox" v-model="form.keepUaSuccess" />
          <span title="Sau khi reg Live, giữ nguyên User-Agent cho slot đó để reg tiếp. Fail → UA mới.">Keep UA</span>
        </label>
        <label class="rp-mode-check">
          <input type="checkbox" v-model="form.keepDatrSuccess" />
          <span title="Sau khi reg ra cookie thành công, dùng datr mới của slot đó để reg tiếp. Fail thì lấy datr khác trong pool.">Keep datr</span>
        </label>
        <label v-if="form.regMode !== 'TempMail'" class="rp-mode-check">
          <input type="checkbox" v-model="form.keepInitialSuccess" />
          <span title="Sau khi reg Live, giữ lại email/phone contact của slot để tiếp tục reg acc kế tiếp. Áp dụng với mode Mail, Phone, Random. Không áp dụng TempMail.">Keep Contact </span>
        </label>
      </div>
      <div class="rp-toolbar__actions">
        <ProfileManager
          :profiles="profiles"
          :active-profile-id="activeProfileId"
          label="Thiết lập chạy"
          @save="handleProfileSave"
          @load="handleProfileLoad"
          @update="handleSave"
          @clone="(id, name) => cloneProfile(id, name)"
          @delete="deleteProfile"
          @rename="renameProfile"
          @export="handleProfileExport"
          @import="handleProfileImportFromManager"
          @reset="resetForms"
        />
        <span class="rp-save-status" :data-status="saveStatus">
          <template v-if="saveStatus === 'saving'">&#x25D0; Đang lưu...</template>
          <template v-else-if="saveStatus === 'saved'">&#x2714; Đã lưu</template>
          <template v-else-if="saveStatus === 'error'">&#x26A0; Lỗi lưu</template>
          <template v-else>&#x2022; Tự động lưu</template>
        </span>
        <button class="rp-btn rp-btn--danger" @click="$router.back()"><X :size="14" /> Đóng</button>
      </div>
    </div>

    <!-- ── Control bar ────────────────────────────────────────────────────── -->
    <div class="rp-controlbar">
      <!-- Mode checkboxes -->
      <div class="rp-mode-checks">
        <label class="rp-mode-check">
          <input type="checkbox" v-model="form.createEnabled" />
          <span>Register</span>
        </label>
        <label class="rp-mode-check">
          <input type="checkbox" v-model="form.verifyEnabled" />
          <span>Verify</span>
        </label>
        <!-- Split mode: chỉ hiện khi bật cả 2 -->
        <label v-if="form.createEnabled && form.verifyEnabled" class="rp-mode-check rp-mode-check--split">
          <input type="checkbox" v-model="form.splitMode" />
          <span title="Reg và Verify chạy độc lập: reg ghi file → verify tự đọc file đó">Split</span>
        </label>
        <!-- Keep IP: giữ proxy cho slot sau reg/verify thành công (port C# KeepIPSuccess) -->
        <label class="rp-mode-check">
          <input type="checkbox" v-model="form.keepIpSuccess" />
          <span title="Sau khi 1 account reg/verify Live, giữ nguyên IP của slot cho account kế. Fail → IP mới.">Keep IP</span>
        </label>
        <!-- Keep UA: giữ User-Agent của slot sau reg thành công -->
        <label class="rp-mode-check">
          <input type="checkbox" v-model="form.keepUaSuccess" />
          <span title="Sau khi reg Live, giữ nguyên User-Agent cho slot đó để reg tiếp. Fail → UA mới.">Keep UA</span>
        </label>
        <label class="rp-mode-check">
          <input type="checkbox" v-model="form.keepDatrSuccess" />
          <span title="Sau khi reg ra cookie thành công, dùng datr mới của slot đó để reg tiếp. Fail thì lấy datr khác trong pool.">Keep datr</span>
        </label>
        <label v-if="form.regMode !== 'TempMail'" class="rp-mode-check">
          <input type="checkbox" v-model="form.keepInitialSuccess" />
          <span title="Sau khi reg Live, giữ lại email/phone contact của slot để tiếp tục reg acc kế tiếp. Áp dụng với mode Mail, Phone, Random. Không áp dụng TempMail.">Keep Contact</span>
        </label>
      </div>

      <div class="rp-controlbar__spacer" />
      <!-- UA Pool selector đã được chuyển xuống section "Reg account" → subsection "User Agent" -->
    </div>

    <Teleport to="body">
      <div v-if="platformDisplayMenu" class="rp-display-modal">
        <div class="rp-display-modal__backdrop" @click="closePlatformDisplayMenu"></div>
        <section class="rp-display-modal__panel" :class="`rp-display-modal__panel--${platformDisplayMenu}`">
          <header class="rp-display-modal__head">
            <div>
              <h3>Hiển thị API {{ platformDisplayMenu === 'reg' ? 'REG' : 'VERIFY' }}</h3>
              <p>Chỉ phiên bản được tick mới hiện thành nút thao tác ở phần thiết lập chạy.</p>
            </div>
            <button type="button" class="rp-display-modal__close" @click="closePlatformDisplayMenu">Đóng</button>
          </header>
          <div class="rp-display-modal__toolbar">
            <span>{{ visiblePlatformCount(platformDisplayMenu) }} đang hiển thị</span>
            <button type="button" @click="selectAllVisiblePlatforms(platformDisplayMenu)">Chọn tất cả</button>
            <button type="button" @click="clearVisiblePlatforms(platformDisplayMenu)">Bỏ chọn tất cả</button>
          </div>

          <div v-if="platformDisplayMenu === 'verify'" class="rp-display-modal__body">
            <section class="rp-display-group rp-display-group--small">
              <div class="rp-display-group__head">
                <strong>API cơ bản</strong>
                <span>
                  <button type="button" @click="toggleVisiblePlatforms('verify', [...VER_PLATFORMS_ANDR, ...VER_PLATFORMS_MFB], true)">Chọn</button>
                  <button type="button" @click="toggleVisiblePlatforms('verify', [...VER_PLATFORMS_ANDR, ...VER_PLATFORMS_MFB], false)">Bỏ</button>
                </span>
              </div>
              <div class="rp-display-grid rp-display-grid--basic">
                <label v-for="p in [...VER_PLATFORMS_ANDR, ...VER_PLATFORMS_MFB]" :key="p.key" class="rp-display-card">
                  <input type="checkbox" :checked="visibleVerifyPlatformKeys.includes(p.key)" @change="toggleVisiblePlatform('verify', p.key)" />
                  <span>{{ p.label }}</span>
                </label>
              </div>
            </section>
            <section class="rp-display-group">
              <div class="rp-display-group__head">
                <strong>Version</strong>
                <span>
                  <button type="button" @click="toggleVisiblePlatforms('verify', VER_PLATFORMS_VER, true)">Chọn</button>
                  <button type="button" @click="toggleVisiblePlatforms('verify', VER_PLATFORMS_VER, false)">Bỏ</button>
                </span>
              </div>
              <div class="rp-display-grid">
                <template v-for="p in VER_PLATFORMS_VER" :key="p.key">
                  <label v-if="!DISABLED_PLATFORM_KEYS.has(p.key)" class="rp-display-card">
                    <input type="checkbox" :checked="visibleVerifyPlatformKeys.includes(p.key)" @change="toggleVisiblePlatform('verify', p.key)" />
                    <span>{{ p.label }}</span>
                  </label>
                </template>
              </div>
            </section>
            <section class="rp-display-group rp-display-group--ios">
              <div class="rp-display-group__head">
                <strong>iOS Native</strong>
                <span>
                  <button type="button" @click="toggleVisiblePlatforms('verify', VER_PLATFORMS_IOS, true)">Chọn</button>
                  <button type="button" @click="toggleVisiblePlatforms('verify', VER_PLATFORMS_IOS, false)">Bỏ</button>
                </span>
              </div>
              <div class="rp-display-grid">
                <label v-for="p in VER_PLATFORMS_IOS" :key="p.key" class="rp-display-card rp-display-card--ios">
                  <input type="checkbox" :checked="visibleVerifyPlatformKeys.includes(p.key)" @change="toggleVisiblePlatform('verify', p.key)" />
                  <span>{{ p.label }}</span>
                </label>
              </div>
            </section>
            <section class="rp-display-group">
              <div class="rp-display-group__head">
                <strong>Messenger</strong>
                <span>
                  <button type="button" @click="toggleVisiblePlatforms('verify', [...VER_PLATFORMS_MESS_ANDR, ...VER_PLATFORMS_MESS_IOS], true)">Chọn</button>
                  <button type="button" @click="toggleVisiblePlatforms('verify', [...VER_PLATFORMS_MESS_ANDR, ...VER_PLATFORMS_MESS_IOS], false)">Bỏ</button>
                </span>
              </div>
              <div class="rp-display-grid">
                <label v-for="p in [...VER_PLATFORMS_MESS_ANDR, ...VER_PLATFORMS_MESS_IOS]" :key="p.key" class="rp-display-card">
                  <input type="checkbox" :checked="visibleVerifyPlatformKeys.includes(p.key)" @change="toggleVisiblePlatform('verify', p.key)" />
                  <span>{{ p.label }}</span>
                </label>
              </div>
            </section>
          </div>

          <div v-else class="rp-display-modal__body">
            <section class="rp-display-group rp-display-group--small">
              <div class="rp-display-group__head">
                <strong>API cơ bản</strong>
                <span>
                  <button type="button" @click="toggleVisiblePlatforms('reg', REG_PLATFORMS_STD, true)">Chọn</button>
                  <button type="button" @click="toggleVisiblePlatforms('reg', REG_PLATFORMS_STD, false)">Bỏ</button>
                </span>
              </div>
              <div class="rp-display-grid rp-display-grid--basic">
                <label v-for="p in REG_PLATFORMS_STD" :key="p.key" class="rp-display-card">
                  <input type="checkbox" :checked="visibleRegPlatformKeys.includes(p.key)" @change="toggleVisiblePlatform('reg', p.key)" />
                  <span>{{ p.label }}</span>
                </label>
              </div>
            </section>
            <section class="rp-display-group">
              <div class="rp-display-group__head">
                <strong>Version</strong>
                <span>
                  <button type="button" @click="toggleVisiblePlatforms('reg', REG_PLATFORMS_VER, true)">Chọn</button>
                  <button type="button" @click="toggleVisiblePlatforms('reg', REG_PLATFORMS_VER, false)">Bỏ</button>
                </span>
              </div>
              <div class="rp-display-grid">
                <template v-for="p in REG_PLATFORMS_VER" :key="p.key">
                  <label v-if="!DISABLED_PLATFORM_KEYS.has(p.key)" class="rp-display-card">
                    <input type="checkbox" :checked="visibleRegPlatformKeys.includes(p.key)" @change="toggleVisiblePlatform('reg', p.key)" />
                    <span>{{ p.label }}</span>
                  </label>
                </template>
              </div>
            </section>
            <section class="rp-display-group rp-display-group--ios">
              <div class="rp-display-group__head">
                <strong>iOS Native</strong>
                <span>
                  <button type="button" @click="toggleVisiblePlatforms('reg', REG_PLATFORMS_IOS, true)">Chọn</button>
                  <button type="button" @click="toggleVisiblePlatforms('reg', REG_PLATFORMS_IOS, false)">Bỏ</button>
                </span>
              </div>
              <div class="rp-display-grid">
                <label v-for="p in REG_PLATFORMS_IOS" :key="p.key" class="rp-display-card rp-display-card--ios">
                  <input type="checkbox" :checked="visibleRegPlatformKeys.includes(p.key)" @change="toggleVisiblePlatform('reg', p.key)" />
                  <span>{{ p.label }}</span>
                </label>
              </div>
            </section>
            <section class="rp-display-group">
              <div class="rp-display-group__head">
                <strong>Messenger</strong>
                <span>
                  <button type="button" @click="toggleVisiblePlatforms('reg', [...REG_PLATFORMS_MESS_ANDR, ...REG_PLATFORMS_MESS_IOS], true)">Chọn</button>
                  <button type="button" @click="toggleVisiblePlatforms('reg', [...REG_PLATFORMS_MESS_ANDR, ...REG_PLATFORMS_MESS_IOS], false)">Bỏ</button>
                </span>
              </div>
              <div class="rp-display-grid">
                <label v-for="p in [...REG_PLATFORMS_MESS_ANDR, ...REG_PLATFORMS_MESS_IOS]" :key="p.key" class="rp-display-card">
                  <input type="checkbox" :checked="visibleRegPlatformKeys.includes(p.key)" @change="toggleVisiblePlatform('reg', p.key)" />
                  <span>{{ p.label }}</span>
                </label>
              </div>
            </section>
          </div>
        </section>
      </div>
    </Teleport>

    <!-- ── Body ───────────────────────────────────────────────────────────── -->
    <div class="rp-page__body">

      <div class="rp-main-col">

      <!-- ══════════════════════════════════════════════════════════
           SECTION 2 — VERIFY / XÁC THỰC
           ══════════════════════════════════════════════════════════ -->
      <div class="rp-section rp-section--verify" style="order: 2">
        <div
          class="rp-section__header rp-section__header--toggle"
          :class="{ 'rp-section__header--no-body': sectionCollapsed.s1 }"
          @click="sectionCollapsed.s1 = !sectionCollapsed.s1"
        >
          <span class="rp-section__num">2</span>
          <span class="rp-section__title">Verify / Xác thực</span>
          <span v-if="form.verifyEnabled" class="rp-section__badge badge--on">BẬT</span>
          <span v-else class="rp-section__badge badge--off">TẮT</span>
          <ChevronDown v-if="sectionCollapsed.s1" :size="14" class="rp-section__caret" />
          <ChevronUp v-else :size="14" class="rp-section__caret" />
        </div>

        <div v-if="!sectionCollapsed.s1" class="rp-section__body">
        <fieldset :disabled="!form.verifyEnabled" class="rp-section__disable-wrap">

          <!-- ── Sub: API & Logic ────────────────────────────────────── -->
          <div class="rp-subsection">
            <div class="rp-subsection__label rp-subsection__label--tools">
              <span>API VERIFY <span class="rp-subsection__hint">— click chọn · kéo chuột để quét chọn (Alt+kéo = bỏ) · chuột phải = menu</span></span>
              <span class="rp-display-filter">
                <button type="button" class="rp-display-filter__btn" @click="togglePlatformDisplayMenu('verify')">Hiển thị</button>
              </span>
            </div>
            <div class="api-dev-notice">⚠️ Bản Instagram đang phát triển — các bản API hiện đã tạm khóa. Chọn/Chạy sẽ trả về "unsupported platform".</div>
            <!-- Platform selector (click bật/tắt 1 version; kéo chuột = quét chọn; chuột phải = menu) -->
            <div class="rp-platform-btns" :class="{ 'rp-platform-btns--dragging': verMarquee.state.dragging, 'rp-platform-btns--removing': verMarquee.state.dragging && verMarquee.state.mode === 'remove' }"
              :ref="verMarquee.setContainerEl" @mousedown="verMarquee.onMouseDown" @contextmenu="openVerifyPlatformMenu">
              <div class="rp-platform-btns__group">
                <button v-for="p in visibleVerifyPlatformsAndr" :key="p.key" type="button" :data-pkey="p.key"
                  :class="['rp-pbtn', { 'rp-pbtn--active': isVerifyPlatformOn(p.key), 'rp-pbtn--focus rp-pbtn--focus-ver': isVerifyPlatformOn(p.key) && form.apiVerifyPlatform === p.key, 'rp-pbtn--marquee': verMarquee.isPreviewed(p.key) }]"
                  :disabled="!isPlatformSelectable(p.key)"
                  @click="toggleVerifyPlatform(p.key)">api {{ p.label }}</button>
                <span v-if="visibleVerifyPlatformsAndr.length && visibleVerifyPlatformsMfb.length" class="rp-pbtn-sep">·</span>
                <button v-for="p in visibleVerifyPlatformsMfb" :key="p.key" type="button" :data-pkey="p.key"
                  :class="['rp-pbtn', { 'rp-pbtn--active': isVerifyPlatformOn(p.key), 'rp-pbtn--focus rp-pbtn--focus-ver': isVerifyPlatformOn(p.key) && form.apiVerifyPlatform === p.key, 'rp-pbtn--marquee': verMarquee.isPreviewed(p.key) }]"
                  :disabled="!isPlatformSelectable(p.key)"
                  @click="toggleVerifyPlatform(p.key)">api {{ p.label }}</button>
              </div>
              <div v-if="visibleVerifyPlatformsVer.length" class="rp-platform-btns__group">
                <button v-for="p in visibleVerifyPlatformsVer" :key="p.key" type="button" :data-pkey="p.key"
                  :class="['rp-pbtn rp-pbtn--versioned', { 'rp-pbtn--active': isVerifyPlatformOn(p.key), 'rp-pbtn--focus rp-pbtn--focus-ver': isVerifyPlatformOn(p.key) && form.apiVerifyPlatform === p.key, 'rp-pbtn--marquee': verMarquee.isPreviewed(p.key) }]"
                  :disabled="!isPlatformSelectable(p.key)"
                  @click="toggleVerifyPlatform(p.key)">{{ p.label }}</button>
              </div>
              <!-- Messenger (Android) — tách riêng khỏi lưới version -->
              <div v-if="visibleVerifyPlatformsMessAndr.length" class="rp-platform-btns__group rp-platform-btns__group--mess">
                <span class="rp-pbtn-grouplbl">Mess</span>
                <button v-for="p in visibleVerifyPlatformsMessAndr" :key="p.key" type="button" :data-pkey="p.key"
                  :class="['rp-pbtn rp-pbtn--versioned', { 'rp-pbtn--active': isVerifyPlatformOn(p.key), 'rp-pbtn--focus rp-pbtn--focus-ver': isVerifyPlatformOn(p.key) && form.apiVerifyPlatform === p.key, 'rp-pbtn--marquee': verMarquee.isPreviewed(p.key) }]"
                  :disabled="!isPlatformSelectable(p.key)"
                  @click="toggleVerifyPlatform(p.key)">{{ p.label }}</button>
              </div>
              <!-- iOS Native App group — màu cyan để phân biệt với Android (verify backend chưa làm) -->
              <div v-if="visibleVerifyPlatformsIos.length" class="rp-platform-btns__group rp-platform-btns__group--ios">
                <span class="rp-pbtn-grouplbl">iOS</span>
                <button v-for="p in visibleVerifyPlatformsIos" :key="p.key" type="button" :data-pkey="p.key"
                  :class="['rp-pbtn rp-pbtn--ios rp-pbtn--versioned', { 'rp-pbtn--active': isVerifyPlatformOn(p.key), 'rp-pbtn--focus rp-pbtn--focus-ver': isVerifyPlatformOn(p.key) && form.apiVerifyPlatform === p.key, 'rp-pbtn--marquee': verMarquee.isPreviewed(p.key) }]"
                  :disabled="!isPlatformSelectable(p.key)"
                  @click="toggleVerifyPlatform(p.key)">{{ p.label }}</button>
              </div>
              <!-- Messenger (iOS) — trong cụm iOS nhưng tách khỏi các version iOS -->
              <div v-if="visibleVerifyPlatformsMessIos.length" class="rp-platform-btns__group rp-platform-btns__group--ios">
                <span class="rp-pbtn-grouplbl">Mess iOS</span>
                <button v-for="p in visibleVerifyPlatformsMessIos" :key="p.key" type="button" :data-pkey="p.key"
                  :class="['rp-pbtn rp-pbtn--ios rp-pbtn--versioned', { 'rp-pbtn--active': isVerifyPlatformOn(p.key), 'rp-pbtn--focus rp-pbtn--focus-ver': isVerifyPlatformOn(p.key) && form.apiVerifyPlatform === p.key, 'rp-pbtn--marquee': verMarquee.isPreviewed(p.key) }]"
                  :disabled="!isPlatformSelectable(p.key)"
                  @click="toggleVerifyPlatform(p.key)">{{ p.label }}</button>
              </div>
              <div v-if="verMarquee.state.box.visible" class="rp-marquee"
                :class="{ 'rp-marquee--remove': verMarquee.state.mode === 'remove' }"
                :style="{ left: verMarquee.state.box.x + 'px', top: verMarquee.state.box.y + 'px', width: verMarquee.state.box.w + 'px', height: verMarquee.state.box.h + 'px' }"></div>
            </div>
            <div v-if="(form.apiVerifyPlatforms?.length ?? 0) > 1" class="rp-multireg-hint rp-multireg-hint--ver">
              Đang chọn <b>{{ form.apiVerifyPlatforms?.length }}</b> version — mỗi account verify dùng 1 version (xoay vòng theo lượt).
              Cấu hình UA bên dưới áp cho version đang focus (viền sáng): <b>{{ form.apiVerifyPlatform }}</b>.
            </div>
            <div v-else-if="(form.apiVerifyPlatforms?.length ?? 0) === 0" class="rp-multireg-hint rp-multireg-hint--empty">
              Chưa chọn version nào — bấm vào nút để chọn. Để trống thì verify dùng mặc định: <b>{{ form.apiVerifyPlatform || 'api android' }}</b>.
            </div>
            <template v-if="verifyPlatformMenu">
              <div class="rp-ctxmenu-overlay" @click="closeVerifyPlatformMenu" @contextmenu.prevent="closeVerifyPlatformMenu"></div>
              <div class="rp-ctxmenu" :style="{ left: verifyPlatformMenu.x + 'px', top: verifyPlatformMenu.y + 'px' }">
                <div class="rp-ctxmenu__submenu">
                  <button type="button" class="rp-ctxmenu__item rp-ctxmenu__item--submenu">Chọn <span>›</span></button>
                  <div class="rp-ctxmenu__flyout">
                    <button type="button" class="rp-ctxmenu__item" @click="selectAllVerifyPlatforms">Tất cả</button>
                    <div class="rp-ctxmenu__sep">── Android ──</div>
                    <button
                      v-for="batch in verifyVersionBatches"
                      :key="batch.map(p => p.key).join('-')"
                      type="button"
                      class="rp-ctxmenu__item"
                      @click="selectVerifyVersionBatch(batch)"
                    >
                      {{ platformRangeLabel(batch) }}
                    </button>
                    <div class="rp-ctxmenu__sep">── iOS ──</div>
                    <button
                      v-for="batch in verifyIosBatches"
                      :key="'ios-'+batch.map(p => p.key).join('-')"
                      type="button"
                      class="rp-ctxmenu__item"
                      @click="selectVerifyVersionBatch(batch)"
                    >
                      {{ platformRangeLabel(batch) }}
                    </button>
                  </div>
                </div>
                <button type="button" class="rp-ctxmenu__item" @click="pasteVerifyPlatforms">Dán từ JSON</button>
                <button type="button" class="rp-ctxmenu__item" @click="clearVerifyPlatforms">Bỏ chọn tất cả</button>
              </div>
            </template>
            <!-- UA inline config (VER) -->
            <div class="rp-uai">
              <div class="rp-uai__row">
                <span class="rp-uai__lbl">Pool:</span>
                <div class="rp-uai__pools">
                  <button v-for="pool in UA_POOLS" :key="pool.key" type="button"
                    :class="['rp-uai__pool', { 'rp-uai__pool--on': verPlatformCfg.uaPoolKey === pool.key }]"
                    :disabled="verPlatformCfg.useOriginalUA || pool.key !== verAllowedPool"
                    @click="verPlatformCfg.uaPoolKey = pool.key">
                    {{ pool.label }}<span class="rp-uai__cnt">{{ uaCounts[pool.key] ?? 0 }}</span>
                  </button>
                </div>
                <label class="rp-uai__check" title="Thêm XID/random16; vào cuối UA — áp dụng tất cả platform">
                  <input type="checkbox" v-model="form.trackingIDVer" @change="saveNow" />
                  <span>Tracking ID</span>
                </label>
                <button type="button" class="rp-uai__test-btn" :disabled="verTestLoading" @click="runVerUATest">
                  {{ verTestLoading ? '...' : verPlatformCfg.useOriginalUA ? '👁 Xem' : '▶ Test' }}
                </button>
              </div>
              <div class="rp-uai__row rp-uai__row--opts">
                <label class="rp-uai__check" :class="{ 'rp-uai__check--dim': !verOriginalUA }" :title="verOriginalUA ? '' : 'Platform này không có UA gốc'">
                  <input type="checkbox" v-model="verPlatformCfg.useOriginalUA" :disabled="!verOriginalUA"
                    @change="onVerUAOptionChanged" />
                  <span>UA Gốc</span>
                </label>
                <label v-if="verShowVirtualSpec" class="rp-uai__check" :class="{ 'rp-uai__check--dim': verPlatformCfg.useOriginalUA }">
                  <input type="checkbox" v-model="verPlatformCfg.addVirtualSpecAndroid" :disabled="verPlatformCfg.useOriginalUA"
                    @change="onVerVirtualSpecChanged" />
                  <span>Virtual spec</span>
                </label>
                <label class="rp-uai__check">
                  <input type="checkbox" v-model="verPlatformCfg.buildUA"
                    @change="onVerBuildUAChanged" />
                  <span>Build UA</span>
                </label>
                <button type="button" v-if="verPlatformCfg.buildUA || IOS_PLATFORM_KEY_SET.has(form.apiVerifyPlatform)" class="rp-uai__open-file"
                  :title="IOS_PLATFORM_KEY_SET.has(form.apiVerifyPlatform) ? 'Mở ios_app_builds.txt — FBIOS build pool cho iOS verify' : 'Mở versions_and_builds_ver.txt — FBAV pool cho verify'"
                  @click="openVersionsFile('ver', form.apiVerifyPlatform)">
                  📝 File version
                </button>
                <label v-if="verPlatformCfg.useOriginalUA && verOriginalUA" class="rp-uai__check rp-uai__check--carrier">
                  <input type="checkbox" v-model="verPlatformCfg.replaceCarrier" @change="saveNow" />
                  <span>Thay nhà mạng</span>
                </label>
              </div>
              <!-- Inline UA Gốc preview cho VERIFY — local, không cần backend. -->
              <div v-if="verPlatformCfg.useOriginalUA && verOriginalUA && !verTestUA" class="rp-uai__codes rp-uai__codes--preview">
                <span class="rp-uai__test-note">UA Gốc ({{ form.apiVerifyPlatform }}) · preview tĩnh — bấm Xem để áp FBCR theo country</span>
                <code class="rp-ua-code">{{ verOriginalUA }}</code>
              </div>
              <div v-if="verTestUA" :class="['rp-uai__codes', { 'rp-uai__codes--ios': IOS_PLATFORM_KEY_SET.has(form.apiVerifyPlatform) }]">
                <span v-if="verTestNote" class="rp-uai__test-note">{{ verTestNote }}</span>
                <code class="rp-ua-code">{{ verTestUA }}</code>
              </div>
            </div>
            <!-- Verify luồng -->
            <div class="rp-grid-2 rp-grid-2--paired">
              <div
                v-if="!(form.createEnabled && form.verifyEnabled && !form.splitMode)"
                class="rp-field"
                title="Số luồng verify chạy song song. 0 = bằng Reg luồng. Áp dụng khi chạy Verify riêng hoặc Reg + Verify + Split mode."
              >
                <label>Verify luồng:</label>
                <input
                  type="number"
                  v-model.number="form.splitVerifyThreads"
                  min="0"
                  max="600"
                  class="vr-input vr-input--num"
                  @blur="clampSplitVerifyThreads"
                  @change="clampSplitVerifyThreads"
                />
              </div>
              <div
                v-else
                class="rp-field rp-field--hint"
                title="Reg + Verify (no Split): verify chạy ngay trên luồng reg. Bật Split mode để cài luồng verify riêng."
              >
                <label>Verify luồng:</label>
                <span class="rp-hint">= Reg luồng (chạy chung)</span>
              </div>
            </div>
          </div>

          <!-- ── Sub: Timing & Delay ────────────────────────────────── -->
          <div class="rp-subsection">
            <div class="rp-subsection__label">Timing &amp; Delay</div>
            <div class="rp-timing-grid rp-timing-grid--dense">
              <div class="rp-field rp-field--compact" title="Tổng thời gian tối đa chờ OTP về mail. Hết giờ này mà không thấy → timeout.">
                <label>Chờ OTP tối đa (s):</label>
                <input type="number" v-model.number="form.timeDelaySendCode" min="1" class="vr-input vr-input--num" />
              </div>
              <div class="rp-field rp-field--compact" title="Khoảng cách giữa 2 lần check mail. 3-5s là chuẩn — quá thấp spam API, quá cao bỏ lỡ OTP.">
                <label>Check mail mỗi (s):</label>
                <input type="number" v-model.number="form.waitMail" min="0" class="vr-input vr-input--num" />
              </div>
              <div class="rp-field rp-field--compact" title="Số lần resend OTP nếu timeout. Tối đa 2 lần (FB block nếu nhiều hơn).">
                <label>Resend OTP (lần):</label>
                <input type="number" v-model.number="form.trySendCode" min="1" max="2" class="vr-input vr-input--num" />
              </div>
              <div class="rp-field rp-field--compact" title="Số lần thử lại add mail khi bị lỗi (ngoài 2 lần mặc định). Mỗi lần retry đổi email mới. 0 = mặc định.">
                <label>Retry add mail (lần):</label>
                <input type="number" v-model.number="form.addMailRetry" min="0" max="5" class="vr-input vr-input--num" />
              </div>
              <div class="rp-field rp-field--compact" title="Đợi N giây sau khi nhận OTP rồi mới submit (giả lập người gõ).">
                <label>Trước submit OTP (s):</label>
                <input type="number" v-model.number="form.delayConfirmEmail" min="0" class="vr-input vr-input--num" />
              </div>
              <div class="rp-field rp-field--compact" title="Đợi N giây sau khi submit confirm OTP rồi mới đọc response Facebook.">
                <label>Sau submit OTP (s):</label>
                <input type="number" v-model.number="form.timeDelayCheck" min="1" class="vr-input vr-input--num" />
              </div>
              <div class="rp-field rp-field--compact" title="Đợi N giây sau bước register → trước khi bắt đầu verify.">
                <label>Reg → verify (s):</label>
                <input type="number" v-model.number="form.delayVeriReg" min="0" class="vr-input vr-input--num" />
              </div>
              <div class="rp-field rp-field--compact" title="Đợi N giây sau verify thành công → trước khi check live/die.">
                <label>Trước check live (s):</label>
                <input type="number" v-model.number="form.delayCheckLive" min="0" class="vr-input vr-input--num" />
              </div>
              <div class="rp-field rp-field--compact" title="Giữ kết quả account cuối trên UI bao lâu trước khi slot chuyển sang account mới.">
                <label>Giữ kết quả UI (s):</label>
                <input type="number" v-model.number="form.delayDisplayResult" min="0" max="60" class="vr-input vr-input--num" />
              </div>
              <div v-show="form.reUseEmail" class="rp-field rp-field--compact" title="Số lần dùng lại 1 mail trước khi lấy mail mới (tiết kiệm chi phí mail rent).">
                <label>Dùng lại mail (lần):</label>
                <input type="number" v-model.number="form.useMailTimes" min="1" class="vr-input vr-input--num" />
              </div>
            </div>
            <div class="rp-checks-row rp-checks-row--merged">
              <label class="rp-checkbox" title="Khi bật, tool sẽ tự gửi lại OTP nếu hết timeout chưa thấy mã.">
                <input type="checkbox" v-model="form.sendAgainCode" />
                <span>Tự gửi lại OTP khi timeout</span>
              </label>
              <span class="rp-checks-sep">·</span>
              <label class="rp-checkbox" title="Sau khi confirm OTP, gọi check Live/Die (picture endpoint / pending redirect) để xác minh acc còn sống → chống ghi nhầm acc chết vào SuccessVerify_No2FA.txt. Vẫn chờ đủ Trước check live (s) trước khi báo Live cuối cùng.">
                <input type="checkbox" v-model="form.checkLiveDieEnabled" />
                <span>Kiểm tra sau reg</span>
              </label>
              <span class="rp-checks-sep">·</span>
              <label class="rp-checkbox" title="Sau pass 1, tự verify lại acc Unknown/Error 1 lần nữa — Pass 2 dùng PROXY MỚI (offset worker slot trong sticky manager) để né proxy đã fail. Acc nào vẫn Unknown sau pass 2 thì giữ Unknown. Đỡ phí acc do lỗi mạng/proxy tạm thời.">
                <input type="checkbox" v-model="form.retryUnknownNow" />
                <span>Verify lại Unknown ngay</span>
              </label>
              <span class="rp-checks-sep">·</span>
              <label class="rp-checkbox">
                <input type="checkbox" v-model="form.reUseEmail" />
                <span>Re-use email</span>
              </label>
              <label class="rp-checkbox">
                <input type="checkbox" v-model="form.fmUserTmpMail" />
                <span>Fm User TmpMail</span>
              </label>
              <span class="rp-checks-sep">·</span>
              <label class="rp-checkbox" title="Sau khi verify Live, dùng chính token + cookie + UA của account đó gọi GraphQL lấy datr mới → thêm vào pool. Yêu cầu: account có token EAAAA.">
                <input type="checkbox" v-model="form.getNewDatrOnLive" />
                <span>Get datr on Live</span>
              </label>
            </div>
          </div>

          <!-- "Thư mục tài khoản cần verify" đã bỏ — nguồn acc configure ở Cài đặt chung (3 modes: folder/file/API).
               Backend đọc settings.General.AccountSource + AccountSourcePath, không cần field riêng. -->


        </fieldset>
        </div>
      </div>

      <!-- ══════════════════════════════════════════════════════════
           SECTION 1 — REGISTER / TẠO TÀI KHOẢN
           ══════════════════════════════════════════════════════════ -->
      <div class="rp-section rp-section--reg" style="order: 1">
        <div
          class="rp-section__header rp-section__header--toggle"
          :class="{ 'rp-section__header--no-body': sectionCollapsed.s2 }"
          @click="sectionCollapsed.s2 = !sectionCollapsed.s2"
        >
          <span class="rp-section__num">1</span>
          <span class="rp-section__title">Reg account</span>
          <span v-if="form.createEnabled" class="rp-section__badge badge--on">BẬT</span>
          <span v-else class="rp-section__badge badge--off">TẮT</span>
          <ChevronDown v-if="sectionCollapsed.s2" :size="14" class="rp-section__caret" />
          <ChevronUp v-else :size="14" class="rp-section__caret" />
        </div>

        <div v-if="!sectionCollapsed.s2" class="rp-section__body">
        <fieldset :disabled="!form.createEnabled" class="rp-section__disable-wrap">

          <!-- API & Setup ──────────────────────────────────────────────────────── -->
          <div class="rp-subsection">
            <div class="rp-subsection__label rp-subsection__label--tools">
              <span>API REG <span class="rp-subsection__hint">— click chọn · kéo chuột để quét chọn (Alt+kéo = bỏ) · chuột phải = menu</span></span>
              <span class="rp-display-filter">
                <button type="button" class="rp-display-filter__btn" @click="togglePlatformDisplayMenu('reg')">Hiển thị</button>
              </span>
            </div>
            <div class="api-dev-notice">⚠️ Bản Instagram đang phát triển — các bản API hiện đã tạm khóa. Chọn/Chạy sẽ trả về "unsupported platform".</div>
            <!-- Platform selector (click bật/tắt 1 version; kéo chuột = quét chọn; chuột phải = menu) -->
            <div class="rp-platform-btns" :class="{ 'rp-platform-btns--dragging': regMarquee.state.dragging, 'rp-platform-btns--removing': regMarquee.state.dragging && regMarquee.state.mode === 'remove' }"
              :ref="regMarquee.setContainerEl" @mousedown="regMarquee.onMouseDown" @contextmenu="openRegPlatformMenu">
              <div class="rp-platform-btns__group">
                <button v-for="p in visibleRegPlatformsStd" :key="p.key" type="button" :data-pkey="p.key"
                  :class="['rp-pbtn', { 'rp-pbtn--active': isRegPlatformOn(p.key), 'rp-pbtn--focus': isRegPlatformOn(p.key) && form.apiRegPlatform === p.key, 'rp-pbtn--marquee': regMarquee.isPreviewed(p.key) }]"
                  :disabled="!isPlatformSelectable(p.key)"
                  @click="toggleRegPlatform(p.key)">{{ p.label }}</button>
              </div>
              <div v-if="visibleRegPlatformsVer.length" class="rp-platform-btns__group">
                <button v-for="p in visibleRegPlatformsVer" :key="p.key" type="button" :data-pkey="p.key"
                  :class="['rp-pbtn rp-pbtn--versioned', { 'rp-pbtn--active': isRegPlatformOn(p.key), 'rp-pbtn--focus': isRegPlatformOn(p.key) && form.apiRegPlatform === p.key, 'rp-pbtn--marquee': regMarquee.isPreviewed(p.key) }]"
                  :disabled="!isPlatformSelectable(p.key)"
                  @click="toggleRegPlatform(p.key)">{{ p.label }}</button>
              </div>
              <!-- Messenger (Android) — tách riêng khỏi lưới version -->
              <div v-if="visibleRegPlatformsMessAndr.length" class="rp-platform-btns__group rp-platform-btns__group--mess">
                <span class="rp-pbtn-grouplbl">Mess</span>
                <button v-for="p in visibleRegPlatformsMessAndr" :key="p.key" type="button" :data-pkey="p.key"
                  :class="['rp-pbtn rp-pbtn--versioned', { 'rp-pbtn--active': isRegPlatformOn(p.key), 'rp-pbtn--focus': isRegPlatformOn(p.key) && form.apiRegPlatform === p.key, 'rp-pbtn--marquee': regMarquee.isPreviewed(p.key) }]"
                  :disabled="!isPlatformSelectable(p.key)"
                  @click="toggleRegPlatform(p.key)">{{ p.label }}</button>
              </div>
              <!-- iOS Native App group — màu cyan để phân biệt với Android orange -->
              <div v-if="visibleRegPlatformsIos.length" class="rp-platform-btns__group rp-platform-btns__group--ios">
                <span class="rp-pbtn-grouplbl">iOS</span>
                <button v-for="p in visibleRegPlatformsIos" :key="p.key" type="button" :data-pkey="p.key"
                  :class="['rp-pbtn rp-pbtn--ios rp-pbtn--versioned', { 'rp-pbtn--active': isRegPlatformOn(p.key), 'rp-pbtn--focus': isRegPlatformOn(p.key) && form.apiRegPlatform === p.key, 'rp-pbtn--marquee': regMarquee.isPreviewed(p.key) }]"
                  :disabled="!isPlatformSelectable(p.key)"
                  @click="toggleRegPlatform(p.key)">{{ p.label }}</button>
              </div>
              <!-- Messenger (iOS) — trong cụm iOS nhưng tách khỏi các version iOS -->
              <div v-if="visibleRegPlatformsMessIos.length" class="rp-platform-btns__group rp-platform-btns__group--ios">
                <span class="rp-pbtn-grouplbl">Mess iOS</span>
                <button v-for="p in visibleRegPlatformsMessIos" :key="p.key" type="button" :data-pkey="p.key"
                  :class="['rp-pbtn rp-pbtn--ios rp-pbtn--versioned', { 'rp-pbtn--active': isRegPlatformOn(p.key), 'rp-pbtn--focus': isRegPlatformOn(p.key) && form.apiRegPlatform === p.key, 'rp-pbtn--marquee': regMarquee.isPreviewed(p.key) }]"
                  :disabled="!isPlatformSelectable(p.key)"
                  @click="toggleRegPlatform(p.key)">{{ p.label }}</button>
              </div>
              <div v-if="regMarquee.state.box.visible" class="rp-marquee"
                :class="{ 'rp-marquee--remove': regMarquee.state.mode === 'remove' }"
                :style="{ left: regMarquee.state.box.x + 'px', top: regMarquee.state.box.y + 'px', width: regMarquee.state.box.w + 'px', height: regMarquee.state.box.h + 'px' }"></div>
            </div>
            <div v-if="(form.apiRegPlatforms?.length ?? 0) > 1" class="rp-multireg-hint">
              Đang chọn <b>{{ form.apiRegPlatforms?.length }}</b> version — mỗi luồng reg cố định 1 version (chia đều theo slot).
              Cấu hình UA bên dưới áp cho version đang focus (viền sáng): <b>{{ form.apiRegPlatform }}</b>.
              <span v-if="(form.apiRegPlatforms?.length ?? 0) > (form.regThreads || 1)" class="rp-multireg-hint__warn">
                ⚠️ Số luồng ({{ form.regThreads || 1 }}) ít hơn số version — chỉ {{ form.regThreads || 1 }} version đầu được chạy.
              </span>
            </div>
            <div v-else-if="(form.apiRegPlatforms?.length ?? 0) === 0" class="rp-multireg-hint rp-multireg-hint--empty">
              Chưa chọn version nào — bấm vào nút để chọn. Để trống thì reg dùng mặc định: <b>{{ form.apiRegPlatform || 's23' }}</b>.
            </div>
            <!-- Context menu chuột phải -->
            <template v-if="regPlatformMenu">
              <div class="rp-ctxmenu-overlay" @click="closeRegPlatformMenu" @contextmenu.prevent="closeRegPlatformMenu"></div>
              <div class="rp-ctxmenu" :style="{ left: regPlatformMenu.x + 'px', top: regPlatformMenu.y + 'px' }">
                <div class="rp-ctxmenu__submenu">
                  <button type="button" class="rp-ctxmenu__item rp-ctxmenu__item--submenu">Chọn <span>›</span></button>
                  <div class="rp-ctxmenu__flyout">
                    <button type="button" class="rp-ctxmenu__item" @click="selectAllRegPlatforms">Tất cả</button>
                    <div class="rp-ctxmenu__sep">── Android ──</div>
                    <button
                      v-for="batch in regVersionBatches"
                      :key="batch.map(p => p.key).join('-')"
                      type="button"
                      class="rp-ctxmenu__item"
                      @click="selectRegVersionBatch(batch)"
                    >
                      {{ platformRangeLabel(batch) }}
                    </button>
                    <div class="rp-ctxmenu__sep">── iOS ──</div>
                    <button
                      v-for="batch in regIosBatches"
                      :key="'ios-'+batch.map(p => p.key).join('-')"
                      type="button"
                      class="rp-ctxmenu__item"
                      @click="selectRegVersionBatch(batch)"
                    >
                      {{ platformRangeLabel(batch) }}
                    </button>
                  </div>
                </div>
                <button type="button" class="rp-ctxmenu__item" @click="pasteRegPlatforms">Dán từ JSON</button>
                <button type="button" class="rp-ctxmenu__item" @click="clearRegPlatforms">Bỏ chọn tất cả</button>
              </div>
            </template>
            <!-- UA inline config (REG) -->
            <div class="rp-uai">
              <div class="rp-uai__row">
                <span class="rp-uai__lbl">Pool:</span>
                <div class="rp-uai__pools">
                  <button v-for="pool in UA_POOLS" :key="pool.key" type="button"
                    :class="['rp-uai__pool', { 'rp-uai__pool--on': regPlatformCfg.uaPoolKey === pool.key }]"
                    :disabled="regPlatformCfg.useOriginalUA || pool.key !== regAllowedPool"
                    @click="regPlatformCfg.uaPoolKey = pool.key">
                    {{ pool.label }}<span class="rp-uai__cnt">{{ uaCounts[pool.key] ?? 0 }}</span>
                  </button>
                </div>
                <label class="rp-uai__check" title="Thêm XID/random16; vào cuối UA — áp dụng tất cả platform">
                  <input type="checkbox" v-model="form.trackingIDReg" @change="saveNow" />
                  <span>Tracking ID</span>
                </label>
                <button type="button" class="rp-uai__test-btn" :disabled="regTestLoading" @click="runRegUATest">
                  {{ regTestLoading ? '...' : regPlatformCfg.useOriginalUA ? '👁 Xem' : '▶ Test' }}
                </button>
              </div>
              <div class="rp-uai__row rp-uai__row--opts">
                <label class="rp-uai__check" :class="{ 'rp-uai__check--dim': !regOriginalUA }" :title="regOriginalUA ? '' : 'Platform này không có UA gốc'">
                  <input type="checkbox" v-model="regPlatformCfg.useOriginalUA" :disabled="!regOriginalUA"
                    @change="onRegUAOptionChanged" />
                  <span>UA Gốc</span>
                </label>
                <label v-if="regShowVirtualSpec" class="rp-uai__check" :class="{ 'rp-uai__check--dim': regPlatformCfg.useOriginalUA }">
                  <input type="checkbox" v-model="regPlatformCfg.addVirtualSpecAndroid" :disabled="regPlatformCfg.useOriginalUA"
                    @change="onRegVirtualSpecChanged" />
                  <span>Virtual spec</span>
                </label>
                <label class="rp-uai__check">
                  <input type="checkbox" v-model="regPlatformCfg.buildUA"
                    @change="onRegBuildUAChanged" />
                  <span>Build UA</span>
                </label>
                <button type="button" v-if="regPlatformCfg.buildUA || IOS_PLATFORM_KEY_SET.has(form.apiRegPlatform)" class="rp-uai__open-file"
                  :title="IOS_PLATFORM_KEY_SET.has(form.apiRegPlatform) ? 'Mở ios_app_builds.txt — FBIOS build pool cho iOS reg' : 'Mở versions_and_builds_reg.txt — FBAV pool cho register'"
                  @click="openVersionsFile('reg', form.apiRegPlatform)">
                  📝 File version
                </button>
                <label v-if="regPlatformCfg.useOriginalUA && regOriginalUA" class="rp-uai__check rp-uai__check--carrier">
                  <input type="checkbox" v-model="regPlatformCfg.replaceCarrier" @change="saveNow" />
                  <span>Thay nhà mạng</span>
                </label>
              </div>
              <!-- Inline UA Gốc preview — hiện ngay khi tick UA Gốc, không cần bấm Xem.
                   Dùng dữ liệu local ORIGINAL_UA_STRINGS[apiRegPlatform], không gọi backend. -->
              <div v-if="regPlatformCfg.useOriginalUA && regOriginalUA && !regTestUA" class="rp-uai__codes rp-uai__codes--preview">
                <span class="rp-uai__test-note">UA Gốc ({{ form.apiRegPlatform }}) · preview tĩnh — bấm Xem để áp FBCR theo country</span>
                <code class="rp-ua-code">{{ regOriginalUA }}</code>
              </div>
              <div v-if="regTestUA" :class="['rp-uai__codes', { 'rp-uai__codes--ios': IOS_PLATFORM_KEY_SET.has(form.apiRegPlatform) }]">
                <span v-if="regTestNote" class="rp-uai__test-note">{{ regTestNote }}</span>
                <code class="rp-ua-code">{{ regTestUA }}</code>
              </div>
            </div>
            <!-- Reg luồng + Delay -->
            <div class="rp-reg-runtime">
              <div class="rp-field rp-field--runtime" title="Số luồng register chạy song song. Nếu Verify bật mà tắt Split mode, verify sẽ chạy chung trên luồng này.">
                <label>Reg luồng:</label>
                <input
                  type="number"
                  v-model.number="form.regThreads"
                  min="1"
                  max="600"
                  class="vr-input vr-input--num"
                  @blur="clampRegThreads"
                  @change="clampRegThreads"
                />
              </div>
              <div class="rp-field rp-field--runtime">
                <label>Delay reg (s):</label>
                <input type="number" v-model.number="form.delayReg" min="0" class="vr-input vr-input--num" />
              </div>
              <div class="rp-field rp-field--runtime">
                <label>Delay step (ms):</label>
                <input type="number" v-model.number="form.delayStep" min="0" class="vr-input vr-input--num" title="Delay giữa các bước (s561v99)" />
              </div>
              <div class="rp-field rp-field--runtime" title="Tự động dừng + chạy lại từ đầu sau N phút (rotate proxy/datr pool tránh burn dài)">
                <label>
                  <input type="checkbox" v-model="form.autoRestartEnabled" />
                  Auto-restart sau (phút):
                </label>
                <input
                  type="number"
                  v-model.number="form.autoRestartMinutes"
                  min="1"
                  max="999"
                  class="vr-input vr-input--num"
                  :disabled="!form.autoRestartEnabled"
                  placeholder="60"
                />
              </div>
            </div>
          </div>

          <!-- Reg Settings -->
          <div class="rp-subsection">
            <div class="rp-subsection__label">Reg Settings</div>
            <div class="rp-grid-2 rp-grid-2--paired rp-reg-settings-grid">
              <div class="rp-field rp-field--lead-domain">
                <label>Lead Domain Mail:</label>
                <input type="text" v-model="form.leadDomainMail" class="vr-input" placeholder="@gmail.com,@yahoo.com" />
              </div>
              <div class="rp-field rp-field--password-reg">
                <label>Password:</label>
                <input type="text" v-model="form.passwordReg" class="vr-input" placeholder="Mẫu password..." />
              </div>
              <div class="rp-field rp-field--name-locale">
                <label>Name:</label>
                <select v-model="form.nameRegLocale" class="vr-select">
                  <option value="US">US</option>
                  <option value="VN">VN</option>
                  <option value="random">Random</option>
                </select>
              </div>
              <div class="rp-field">
                <label>Mode:</label>
                <select v-model="form.regMode" class="vr-select" :disabled="form.regModeRotate">
                  <option value="Mail">Mail (lead domain)</option>
                  <option value="Phone">Phone</option>
                  <option value="TempMail">Temp Mail (reuse cho verify)</option>
                  <option value="MailTemp">Mail-Temp.com</option>
                  <option value="Random">Random</option>
                </select>
              </div>
              <div class="rp-rotate-inline">
                <label class="rp-mode-check rp-mode-check--rotate" :title="(form.regMode === 'TempMail' || form.regMode === 'MailTemp') ? 'Không áp dụng cho TempMail/MailTemp' : 'Tự động xoay giữa Mail và Phone theo chu kỳ thời gian'">
                  <input type="checkbox" v-model="form.regModeRotate" :disabled="form.regMode === 'TempMail' || form.regMode === 'MailTemp'" />
                  <span>Xoay Mail ↔ Phone</span>
                </label>
                <template v-if="form.regModeRotate">
                  <span class="rp-label">Mail:</span>
                  <input
                    type="number" v-model.number="form.regModeRotateMailMinutes"
                    class="vr-input vr-input--num rp-rotate-num" min="1"
                    title="Số phút chạy Mode=Mail trước khi chuyển sang Phone"
                  />
                  <span class="vr-unit">phút</span>
                  <span class="rp-label">Phone:</span>
                  <input
                    type="number" v-model.number="form.regModeRotatePhoneMinutes"
                    class="vr-input vr-input--num rp-rotate-num" min="1"
                    title="Số phút chạy Mode=Phone trước khi chuyển sang Mail"
                  />
                  <span class="vr-unit">phút</span>
                </template>
              </div>
            </div>
            <div class="rp-checks-row">
              <span class="rp-label">Phone × Mail:</span>
              <label class="rp-radio">
                <input type="radio" name="phoneMailMode" value="random-normal" v-model="form.phoneMailMode" />
                <span>Random Normal</span>
              </label>
              <label class="rp-radio">
                <input type="radio" name="phoneMailMode" value="random-file" v-model="form.phoneMailMode" />
                <span>Random File</span>
              </label>
              <label class="rp-radio">
                <input type="radio" name="phoneMailMode" value="fm-phone" v-model="form.phoneMailMode" />
                <span>Fm Phone Code</span>
              </label>
              <!-- 2 toggle UA (buildUA + addVirtualSpecAndroid) đã chuyển vào subsection User Agent ở trên -->
            </div>
          </div>

          <!-- Cookie Initial — đã được chuyển vào trong cụm Reg account -->
          <div class="rp-subsection rp-subsection--ci">
            <div class="rp-ci-head">
              <div class="rp-subsection__label">Cookie Initial</div>
              <span class="rp-ci-desc">Nguồn datr đầu vào cho pool register</span>
            </div>
            <div class="rp-sf-ci">
              <div class="rp-ci-group">
                <span class="rp-ci-group-label">Nguồn</span>
                <div class="rp-ci-method">
                  <label class="rp-radio">
                    <input type="radio" name="cookieInitialMethod" value="file" v-model="form.cookieInitialMethod" />
                    <span>Từ file</span>
                  </label>
                  <label class="rp-radio" title="Sinh datr ngẫu nhiên 24 ký tự theo cấu trúc Facebook web (signature 'Ga'). Không cần file cookie_initial.txt.">
                    <input type="radio" name="cookieInitialMethod" value="new" v-model="form.cookieInitialMethod" />
                    <span>Tạo mới</span>
                  </label>
                </div>
                <label class="rp-ci-limit" title="Khi tích: datr mới thu được từ cookie reg sẽ được ghi vào cookie_initial.txt để tái dùng cho lần reg sau">
                  <input type="checkbox" v-model="form.saveNewDatr" />
                  <span>Add new pool</span>
                </label>
                <div v-if="form.cookieInitialMethod === 'file'" class="rp-ci-filebar">
                  <button type="button" class="rp-mini-btn" :title="cookieInitialResolvedPath || form.cookieInitialFile"
                    @click="openCookieInitialFile">Mở file datr</button>
                  <span class="rp-ci-count" :class="{ 'rp-ci-count--error': !!cookieInitialFileStatus }">
                    {{ cookieInitialFileStatus ? 'Không đọc được file' : `${cookieInitialFileCount.toLocaleString()} datr${datrPoolCount > 0 ? ` · pool: ${datrPoolCount.toLocaleString()}` : ''}${poolFileSaveCount > 0 ? ` · +${poolFileSaveCount.toLocaleString()} saved` : ''}` }}
                  </span>
                </div>
              </div>
              <div v-if="form.cookieInitialMethod === 'file' && cookieInitialFileStatus" class="rp-ci-error"
                :title="cookieInitialFileStatus">
                {{ cookieInitialFileStatus }}
              </div>
              <div class="rp-ci-group">
                <span class="rp-ci-group-label">Giới hạn</span>
                <div class="rp-ci-row rp-ci-row--compact">
                <label class="rp-ci-limit">
                  <input type="checkbox" v-model="form.limitCookieInitial" />
                  <span>Lượt dùng</span>
                  <input type="number" v-model.number="form.limitCookieInitialCount" min="1"
                    class="vr-input vr-input--num rp-ci-limit-num" :disabled="!form.limitCookieInitial" />
                </label>
                <label class="rp-ci-limit" title="Dừng reg tự động khi số checkpoint vượt ngưỡng — tránh đốt proxy">
                  <input type="checkbox" v-model="form.limitCheckpoint" />
                  <span>Checkpoint</span>
                  <input type="number" v-model.number="form.limitCheckpointCount" min="1"
                    class="vr-input vr-input--num rp-ci-limit-num" :disabled="!form.limitCheckpoint" />
                </label>
                <label class="rp-ci-limit" title="Khi tích: datr bị xóa khỏi pool và cookie_initial.txt ngay khi đạt giới hạn ở 1 trong 2 ô bên trái (giới hạn usage hoặc giới hạn checkpoint). Datr reg thành công sẽ reset bộ đếm checkpoint.">
                  <input type="checkbox" v-model="form.deleteDatrCheckpoint" />
                  <span>Xóa khi đạt giới hạn</span>
                </label>
                <label class="rp-ci-limit" title="Xóa datr khỏi pool sau N phút kể từ lúc nạp/sinh. Dùng để tránh dùng datr quá cũ.">
                  <input type="checkbox" v-model="form.limitDatrAge" />
                  <span>Xóa sau</span>
                  <input type="number" v-model.number="form.limitDatrAgeMinutes" min="1"
                    class="vr-input vr-input--num rp-ci-limit-num" :disabled="!form.limitDatrAge" />
                  <span>phút</span>
                </label>
                </div>
              </div>
            </div>
          </div>


        </fieldset>
        </div>
      </div>


      </div><!-- /rp-main-col -->

      <!-- ══════════════════════════════════════════════════════════
           SECTION 3 — USER AGENT
           Đặt dưới Reg/Verify để dùng chung cho cả 2 luồng.
           Nguồn UA + 2 modifier (addVirtualSpec + useBuildNumFile) + Dùng UA gốc.
           Trước đây panel UA pool nằm ở Cài đặt chung — đã chuyển hẳn sang đây
           để gom toàn bộ thiết lập chạy 1 chỗ.
           ══════════════════════════════════════════════════════════ -->
      <div class="rp-section rp-section--ua">
        <div
          class="rp-section__header rp-section__header--toggle"
          :class="{ 'rp-section__header--no-body': sectionCollapsed.s3 }"
          @click="sectionCollapsed.s3 = !sectionCollapsed.s3"
        >
          <span class="rp-section__num">3</span>
          <span class="rp-section__title">User Agent</span>
          <span class="rp-section__badge badge--ua">{{ activeUaPoolMeta?.label || form.uaPoolKey }}</span>
          <ChevronDown v-if="sectionCollapsed.s3" :size="14" class="rp-section__caret" />
          <ChevronUp v-else :size="14" class="rp-section__caret" />
        </div>

        <div v-if="!sectionCollapsed.s3" class="rp-section__body">

          <!-- Chọn loại UA — map sang file Config/UserAgent/{file} -->
          <div class="rp-ua-tabs">
            <button
              v-for="pool in UA_POOLS"
              :key="pool.key"
              type="button"
              :class="['rp-ua-tab', { 'rp-ua-tab--active': form.uaPoolKey === pool.key }]"
              @click="form.uaPoolKey = pool.key"
            >
              {{ pool.label }}
              <span class="rp-ua-tab__count">{{ uaCounts[pool.key] ?? 0 }}</span>
            </button>
          </div>

          <!-- Đường dẫn file nguồn UA + nút mở file -->
          <div class="rp-ua-source-row">
            <span class="rp-ua-source-label">Nguồn:</span>
            <code class="rp-ua-source-path">Config/UserAgent/{{ activeUaPoolMeta?.file }}</code>
            <button type="button" class="rp-btn-icon rp-ua-open-btn" @click="openUAFile" title="Mở file trong Notepad">Mở file</button>
          </div>

        </div>
      </div>

    </div><!-- /rp-page__body -->

    <!-- Result folder: ẩn khỏi UI — app tự tạo ./result/ cạnh exe, user không cần biết path.
         Truy cập qua nút "Result" ở tab Accounts → mở trực tiếp folder. -->
    <!-- <div class="rp-result-folder-bar">...</div> -->

  </div>


</template>

<style scoped>
/* ══ Page shell ══════════════════════════════════════════════════════════════ */
.rp-page {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.rp-toolbar {
  min-height: var(--toolbar-height);
  height: auto;
  background: var(--surface-elevated);
  border-bottom: 1px solid var(--border-default);
  display: flex;
  align-items: center;
  padding: 6px var(--space-4);
  gap: 10px 12px;
  flex-shrink: 0;
  flex-wrap: wrap;
}
.rp-toolbar__title {
  font-size: var(--font-size-lg);
  font-weight: 700;
  flex: 0 0 auto;
  margin-right: var(--space-1);
  white-space: nowrap;
}
.rp-toolbar__actions {
  display: flex;
  gap: var(--space-2);
  align-items: center;
  margin-left: auto;
  flex-shrink: 0;
}

/* ══ Control bar ════════════════════════════════════════════════════════════ */
.rp-controlbar {
  display: none;
  align-items: center;
  gap: var(--space-2);
  padding: 6px var(--space-4);
  background: var(--surface-elevated);
  border-bottom: 1px solid var(--border-default);
  flex-shrink: 0;
  flex-wrap: wrap;
}
.rp-controlbar__sep {
  width: 1px; height: 16px;
  background: var(--border-default);
  flex-shrink: 0;
}
.rp-controlbar__spacer { flex: 1; min-width: var(--space-2); }

/* ══ Mode checkboxes ════════════════════════════════════════════════════════ */
.rp-mode-checks {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.rp-mode-checks--toolbar {
  flex: 1 1 300px;
  min-width: 260px;
}
.rp-mode-check {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  color: var(--text-secondary);
}
.rp-mode-check input[type="checkbox"] {
  width: 14px;
  height: 14px;
  accent-color: var(--brand-primary);
  cursor: pointer;
}

/* ══ UA pool selector ═══════════════════════════════════════════════════════ */
.rp-ua-pool-select {
  display: flex;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  overflow: hidden;
  flex-shrink: 0;
}
.rp-ua-pool-btn {
  padding: 4px 9px;
  background: transparent;
  border: none;
  border-right: 1px solid var(--border-default);
  color: var(--text-secondary);
  font-size: var(--font-size-xs);
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.15s;
}
.rp-ua-pool-btn:last-child { border-right: none; }
.rp-ua-pool-btn:hover { background: var(--surface-hover); color: var(--text-primary); }
.rp-ua-pool-btn--active { background: rgba(225,48,108,0.15); color: var(--accent); font-weight: 700; }

/* ══ User Agent subsection (in Reg account section) ════════════════════════ */
.rp-subsection--ua {
  background: rgba(225,48,108,0.04);
  border: 1px dashed rgba(225,48,108,0.25);
}
.rp-ua-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding: 4px 0;
}
.rp-ua-row--toggles {
  gap: 18px;
  margin-top: 4px;
  padding-top: 6px;
  border-top: 1px dashed var(--border-default);
}
.rp-ua-row__label {
  font-size: var(--font-size-xs);
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}
.rp-ua-pool-select--lg { box-shadow: 0 0 0 1px rgba(225,48,108,0.15) inset; }
.rp-ua-pool-btn--lg {
  padding: 8px 18px;
  font-size: var(--font-size-sm);
  font-weight: 600;
  min-width: 120px;
  text-align: center;
}
.rp-ua-pool-btn--lg.rp-ua-pool-btn--active {
  background: var(--accent);
  color: #000;
  font-weight: 800;
  box-shadow: 0 0 0 2px rgba(225,48,108,0.45);
  text-shadow: 0 0 1px rgba(0,0,0,0.2);
}
.rp-ua-pool-btn--lg.rp-ua-pool-btn--active:hover { background: var(--accent); color: #000; }
.rp-ua-active-hint {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}
.rp-ua-active-hint strong {
  color: var(--accent);
  font-weight: 700;
}

/* ══ Section "User Agent" (top-level, sau Reg/Verify) ══════════════════════
   Layout: sibling của rp-main-col → gap thừa kế từ rp-page__body (space-3).
   Border + radius giống Reg/Verify để 3 section đồng bộ.
   ══════════════════════════════════════════════════════════════════════ */
.rp-section--ua {
  border-left: 3px solid var(--accent);
}
.rp-section--ua .rp-section__num { background: var(--accent); color: #000; }
.rp-section--ua .rp-section__title { color: var(--accent); }
.rp-section--ua .rp-section__body {
  padding: 10px 14px 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.rp-section__badge.badge--ua {
  background: rgba(225,48,108,0.18);
  color: var(--accent);
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

/* Tabs Android / iPhone / Chrome */
.rp-ua-tabs {
  display: flex;
  gap: 3px;
  border-bottom: 1px solid var(--border-default);
  padding-bottom: 4px;
}
.rp-ua-source-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 0;
  font-size: var(--font-size-xs);
  color: var(--text-muted);
}
.rp-ua-source-label { flex-shrink: 0; }
.rp-ua-source-path { color: var(--text-secondary); font-family: var(--font-mono, monospace); }
.rp-ua-tab {
  padding: 4px 12px;
  border-radius: var(--radius-sm) var(--radius-sm) 0 0;
  border: 1px solid var(--border-default);
  border-bottom: none;
  background: transparent;
  color: var(--text-secondary);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 5px;
  transition: all 0.15s;
}
.rp-ua-tab:hover { color: var(--text-primary); background: var(--surface-hover); }
.rp-ua-tab--active {
  background: var(--accent);
  color: #000;
  border-color: var(--accent);
  font-weight: 800;
  text-shadow: 0 0 1px rgba(0,0,0,0.2);
}
.rp-ua-tab__count {
  font-size: 10px;
  font-weight: 700;
  padding: 1px 6px;
  border-radius: 8px;
  background: var(--surface-hover-strong);
}
.rp-ua-tab--active .rp-ua-tab__count {
  background: rgba(0,0,0,0.18);
  color: #000;
}

/* Textarea + file mode */
.vr-textarea--mono {
  width: 100%;
  box-sizing: border-box;
  padding: 4px 8px;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  color: var(--text-primary);
  border-radius: var(--radius-sm);
  font-family: var(--font-mono, monospace);
  font-size: 10px;
  line-height: 1.4;
  resize: vertical;
  outline: none;
  min-height: 36px;
  max-height: 80px;
}
.vr-textarea--mono:focus { border-color: var(--accent); }

.rp-ua-file-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
}
.rp-ua-file-path {
  flex: 1;
  font-size: var(--font-size-xs);
  font-family: var(--font-mono, monospace);
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rp-btn-icon {
  background: transparent;
  border: 1px solid var(--border-default);
  color: var(--text-secondary);
  border-radius: var(--radius-sm);
  padding: 2px 8px;
  font-size: 12px;
  cursor: pointer;
  flex-shrink: 0;
  transition: all 0.15s;
}
.rp-btn-icon:hover { border-color: var(--accent); color: var(--accent); }
.rp-btn-icon--danger:hover { border-color: #f44336; color: #f44336; }
.rp-ua-open-btn { font-size: 11px; padding: 2px 10px; margin-left: auto; flex-shrink: 0; }

.rp-ua-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.rp-ua-count {
  font-size: var(--font-size-xs);
  color: var(--text-muted);
}
.rp-ua-file-badge {
  font-size: 10px;
  font-weight: 700;
  padding: 1px 7px;
  border-radius: 8px;
  background: rgba(225,48,108,0.15);
  color: var(--accent);
  margin-left: 4px;
}
.rp-btn--xs {
  padding: 4px 10px;
  font-size: var(--font-size-xs);
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  color: var(--text-secondary);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: background var(--transition-fast);
}
.rp-btn--xs:hover { background: var(--surface-hover); color: var(--text-primary); }

/* Toggles row — 3 cột đều, gọn */
.rp-ua-toggles-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px 16px;
  padding-top: 6px;
  border-top: 1px dashed var(--border-default);
}
.rp-ua-toggles-row .rp-checkbox {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  cursor: pointer;
  min-width: 0;
}
.rp-ua-toggles-row .rp-checkbox input { accent-color: var(--accent); margin-top: 2px; flex-shrink: 0; }
.rp-checkbox__content { display: flex; flex-direction: column; gap: 0; min-width: 0; }
.rp-checkbox__label {
  font-size: 12px;
  color: var(--text-primary);
  font-weight: 600;
  line-height: 1.3;
}
.rp-checkbox__desc {
  font-size: 10px;
  color: var(--text-muted);
  line-height: 1.3;
}
.rp-checkbox--primary .rp-checkbox__label { color: var(--accent); }
.rp-checkbox--muted { opacity: 0.45; pointer-events: none; }

/* UA Gốc row */
.rp-ua-original-row {
  padding: 4px 0 6px;
}
.rp-ua-original-row .rp-checkbox--original {
  display: flex;
  align-items: flex-start;
  gap: 6px;
  cursor: pointer;
}
.rp-ua-original-row .rp-checkbox--original input { accent-color: #81c784; margin-top: 2px; flex-shrink: 0; }
.rp-ua-original-row .rp-checkbox--original .rp-checkbox__label { color: #81c784; }

/* Disabled state for toggles row when UA Gốc is active */
.rp-ua-toggles-row--disabled {
  opacity: 0.45;
}

/* Hint styles */
.rp-ua-hint--original {
  color: #81c784;
  font-size: 11px;
}

/* ══ Per-platform UA summary (trong UA section) ═════════════════════════════ */
.rp-ua-platform-summary {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  background: var(--surface-sunken);
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle, var(--border-default));
}
.rp-ua-ps-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--font-size-xs);
}
.rp-ua-ps-badge {
  font-size: 10px;
  font-weight: 700;
  padding: 1px 5px;
  border-radius: var(--radius-sm);
  flex-shrink: 0;
}
.rp-ua-ps-badge--reg { background: rgba(251,140,0,0.15); color: #fb8c00; border: 1px solid rgba(251,140,0,0.3); }
.rp-ua-ps-badge--ver { background: rgba(225,48,108,0.12); color: var(--accent); border: 1px solid rgba(225,48,108,0.3); }
.rp-ua-ps-name { color: var(--text-primary); font-weight: 500; flex-shrink: 0; }
.rp-ua-ps-desc {
  color: var(--text-muted);
  font-style: italic;
}
.rp-ua-ps-desc::before { content: '→ '; }

/* ══ Inline UA config panel (rp-uai) ════════════════════════════════════════ */
.rp-uai {
  display: flex;
  flex-direction: column;
  gap: 5px;
  padding: 7px 10px;
  background: var(--surface-sunken);
  border: 1px solid var(--border-subtle, var(--border-default));
  border-radius: var(--radius-md);
}
.rp-uai__row {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.rp-uai__lbl {
  font-size: 10px;
  font-weight: 700;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  flex-shrink: 0;
}
.rp-uai__pools { display: flex; gap: 3px; flex-wrap: wrap; }
.rp-uai__pool {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 500;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.1s;
}
.rp-uai__pool:hover:not(:disabled) { background: var(--surface-hover); color: var(--text-primary); }
.rp-uai__pool--on {
  background: rgba(225,48,108,0.12);
  color: var(--accent);
  border-color: rgba(225,48,108,0.4);
  font-weight: 700;
}
.rp-uai__pool:disabled { opacity: 0.38; cursor: not-allowed; }
.rp-uai__cnt {
  font-size: 9px;
  font-weight: 700;
  padding: 0 4px;
  border-radius: 5px;
  background: rgba(0,0,0,0.22);
  color: inherit;
  line-height: 1.6;
}
/* Options row */
.rp-uai__row--opts { gap: 10px; }
.rp-uai__check {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-weight: 500;
  color: var(--text-secondary);
  cursor: pointer;
  white-space: nowrap;
}
.rp-uai__check input[type="checkbox"] { accent-color: var(--accent); flex-shrink: 0; }
.rp-uai__check--dim { opacity: 0.38; pointer-events: none; }
.rp-uai__check--carrier {
  color: #8ba3c7;
  padding-left: 6px;
  border-left: 2px solid rgba(139,163,199,0.3);
}
/* Test button */
.rp-uai__test-btn {
  margin-left: auto;
  padding: 2px 9px;
  font-size: 11px;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  cursor: pointer;
  flex-shrink: 0;
  transition: all 0.1s;
}
.rp-uai__test-btn:hover:not(:disabled) { border-color: var(--accent); color: var(--accent); }
.rp-uai__test-btn:disabled { opacity: 0.45; cursor: not-allowed; }

.rp-uai__open-file {
  padding: 2px 9px;
  font-size: 11px;
  background: var(--surface-base);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  cursor: pointer;
  flex-shrink: 0;
  transition: all 0.1s;
}
.rp-uai__open-file:hover { border-color: var(--accent); color: var(--accent); }

.rp-reg-runtime {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  padding: 2px 0 4px;
}
.rp-field--runtime {
  flex: none;
}
.rp-field--runtime > label:first-child {
  min-width: auto;
}
/* Code blocks */
.rp-uai__codes {
  display: flex;
  flex-direction: column;
  gap: 2px;
  background: rgba(180,30,30,0.25);
  border-radius: var(--radius-sm);
  padding: 4px 8px;
  overflow-x: auto;
}
.rp-uai__test-note {
  font-size: 10px;
  color: var(--text-muted);
  font-style: italic;
}
/* Preview inline UA Gốc (chưa qua backend) — background xanh dương nhạt để
 * phân biệt với rp-uai__codes thường (đỏ nhạt khi có error/result thật). */
.rp-uai__codes--preview {
  background: rgba(6,182,212,0.12) !important;
  border: 1px solid rgba(6,182,212,0.3);
}
.rp-uai__codes--preview .rp-uai__test-note {
  color: #0891b2;
  font-weight: 500;
}
.rp-uai__codes--ios {
  background: rgba(2,132,199,0.42) !important;
  border: 1px solid rgba(6,182,212,0.6);
}
.rp-uai__codes--ios .rp-uai__test-note {
  color: #bae6fd;
}
.rp-uai__codes--ios .rp-ua-code {
  color: #fff;
}

/* ══ Platform selector buttons ═══════════════════════════════════════════════ */
.rp-platform-btns {
  display: flex;
  flex-direction: column;
  gap: 4px;
  position: relative; /* mốc định vị cho overlay marquee */
  user-select: none;  /* nhãn nút không cần bôi đen — tránh select text khi click/kéo */
}
/* Khi đang kéo quét chọn: đổi con trỏ thành crosshair. */
.rp-platform-btns--dragging {
  cursor: crosshair;
}
/* Khung marquee (rubber-band) khi kéo chuột — toạ độ absolute theo container. */
.rp-marquee {
  position: absolute;
  z-index: 6;
  pointer-events: none;
  border: 1px solid var(--accent);
  background: rgba(59, 130, 246, 0.12);
  border-radius: 2px;
}
.rp-marquee--remove {
  border-color: #ef4444;
  background: rgba(239, 68, 68, 0.12);
}
/* Nút đang nằm trong khung marquee (preview realtime). */
.rp-pbtn--marquee {
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.75) inset;
}
.rp-platform-btns--removing .rp-pbtn--marquee {
  box-shadow: 0 0 0 2px rgba(239, 68, 68, 0.75) inset;
}
.rp-platform-btns__group {
  display: flex;
  flex-wrap: wrap;
  gap: 3px;
}
.rp-pbtn {
  padding: 3px 8px;
  background: var(--surface-sunken);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: 11px;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.12s;
}
.rp-pbtn:hover { background: var(--surface-hover); color: var(--text-primary); }
.rp-pbtn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
  background: var(--surface-sunken);
  color: var(--text-muted);
  border-color: var(--border-subtle);
}
.rp-section--reg .rp-pbtn--active {
  background: rgba(249,115,22,0.15);
  color: #f97316;
  border-color: rgba(249,115,22,0.4);
  font-weight: 700;
}
.rp-section--verify .rp-pbtn--active {
  background: rgba(59,130,246,0.15);
  color: var(--accent);
  border-color: rgba(59,130,246,0.4);
  font-weight: 700;
}
.rp-pbtn--versioned { font-family: var(--font-mono, monospace); font-size: 10.5px; }
.rp-pbtn-sep { color: var(--border-default); font-size: 12px; align-self: center; user-select: none; }

/* iOS Native App group — màu cyan (#06b6d4) để phân biệt rõ với Android orange.
 * Apple-themed cyan vẫn fall trong palette REG section nhưng đủ tương phản. */
.rp-platform-btns__group--ios {
  border-top: 2px solid rgba(6,182,212,0.55);
  padding-top: 8px;
  margin-top: 7px;
  align-items: center;
}
/* Messenger (Android) group — thanh ngăn xanh lá (#22c55e) phân tách cụm Mess Android
 * khỏi lưới version phía trên, tương tự thanh ngăn cyan của nhóm iOS. */
.rp-platform-btns__group--mess {
  border-top: 2px solid rgba(34,197,94,0.55);
  padding-top: 8px;
  margin-top: 7px;
  align-items: center;
}
.rp-platform-btns__group--mess .rp-pbtn-grouplbl {
  color: #22c55e;
  background: rgba(34,197,94,0.1);
  border-color: rgba(34,197,94,0.3);
}
.rp-pbtn-grouplbl {
  font-size: 10px;
  font-weight: 700;
  color: #06b6d4;
  background: rgba(6,182,212,0.1);
  border: 1px solid rgba(6,182,212,0.3);
  border-radius: var(--radius-sm);
  padding: 2px 6px;
  letter-spacing: 0.5px;
  user-select: none;
}
/* iOS CHƯA CHỌN: nền trắng/neutral GIỐNG Android để dễ phân biệt với trạng thái đã chọn.
 * Việc đây là nhóm iOS được nhận biết qua nhãn "iOS" + đường kẻ phân cách phía trên. */
.rp-pbtn--ios {
  border-color: var(--border-default);
  color: var(--text-secondary);
  background: var(--surface-sunken);
}
.rp-pbtn--ios:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}
/* iOS ĐÃ CHỌN — cyan đậm, nổi bật rõ trên nền trắng (override orange REG / blue VERIFY). */
.rp-section--reg .rp-pbtn--ios.rp-pbtn--active,
.rp-section--verify .rp-pbtn--ios.rp-pbtn--active {
  background: rgba(6,182,212,0.28);
  color: #0e7490;
  border-color: rgba(6,182,212,0.75);
  font-weight: 700;
}
.rp-section--reg .rp-pbtn--ios.rp-pbtn--focus,
.rp-section--verify .rp-pbtn--ios.rp-pbtn--focus {
  box-shadow: 0 0 0 1px #06b6d4 inset;
  border-color: #06b6d4;
}

/* Version đang "focus" trong multi-select (UA panel áp cho nó) — viền sáng đậm hơn. */
.rp-section--reg .rp-pbtn--focus {
  box-shadow: 0 0 0 1px #f97316 inset;
  border-color: #f97316;
}
.rp-section--verify .rp-pbtn--focus {
  box-shadow: 0 0 0 1px var(--accent) inset;
  border-color: var(--accent);
}
.rp-multireg-hint {
  margin-top: 5px;
  font-size: 10.5px;
  line-height: 1.4;
  color: var(--text-secondary);
  background: rgba(249,115,22,0.07);
  border: 1px solid rgba(249,115,22,0.25);
  border-radius: var(--radius-sm);
  padding: 4px 8px;
}
.rp-multireg-hint b { color: #f97316; }
.rp-multireg-hint__warn { display: block; margin-top: 2px; color: #d97706; }
.rp-multireg-hint--ver { background: rgba(59,130,246,0.07); border-color: rgba(59,130,246,0.25); }
.rp-multireg-hint--ver b { color: var(--accent); }
.rp-multireg-hint--empty {
  background: var(--surface-sunken);
  border-color: var(--border-default);
  color: var(--text-muted);
}
.rp-multireg-hint--empty b { color: var(--text-secondary); }
.rp-subsection__hint { font-weight: 400; font-size: 10px; color: var(--text-muted); }

.rp-subsection__label--tools {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.rp-display-filter { flex-shrink: 0; }
.rp-display-filter__btn {
  padding: 2px 8px;
  font-size: 10.5px;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  background: var(--surface-sunken);
  color: var(--text-secondary);
  cursor: pointer;
}
.rp-display-filter__btn:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}

.rp-display-modal {
  position: fixed;
  inset: 0;
  z-index: 2000;
  display: grid;
  place-items: center;
  padding: 24px;
}
.rp-display-modal__backdrop {
  position: absolute;
  inset: 0;
  background: rgba(0,0,0,0.48);
}
.rp-display-modal__panel {
  position: relative;
  width: min(920px, calc(100vw - 48px));
  max-height: min(760px, calc(100vh - 48px));
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  box-shadow: 0 18px 64px rgba(0,0,0,0.42);
  --display-accent: #f97316;
}
.rp-display-modal__panel--verify { --display-accent: var(--accent); }
.rp-display-modal__head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 18px;
  padding: 16px 18px;
  border-left: 4px solid var(--display-accent);
  border-bottom: 1px solid var(--border-default);
  background: var(--surface-sunken);
}
.rp-display-modal__head h3 {
  margin: 0;
  font-size: 16px;
  color: var(--text-primary);
}
.rp-display-modal__head p {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--text-secondary);
}
.rp-display-modal__close,
.rp-display-modal__toolbar button,
.rp-display-group__head button {
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  background: var(--surface-elevated);
  color: var(--text-secondary);
  cursor: pointer;
}
.rp-display-modal__close {
  padding: 5px 10px;
  font-size: 12px;
}
.rp-display-modal__toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  border-bottom: 1px solid var(--border-subtle);
  color: var(--text-secondary);
  font-size: 12px;
}
.rp-display-modal__toolbar span { margin-right: auto; }
.rp-display-modal__toolbar button,
.rp-display-group__head button {
  padding: 4px 9px;
  font-size: 11.5px;
}
.rp-display-modal__close:hover,
.rp-display-modal__toolbar button:hover,
.rp-display-group__head button:hover {
  background: var(--surface-hover);
  color: var(--text-primary);
}
.rp-display-modal__body {
  overflow: auto;
  padding: 16px 18px 18px;
}
.rp-display-group + .rp-display-group { margin-top: 16px; }
.rp-display-group__head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}
.rp-display-group__head strong {
  color: var(--text-primary);
  font-size: 13px;
}
.rp-display-group__head span {
  display: flex;
  gap: 6px;
}
.rp-display-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(104px, 1fr));
  gap: 7px;
}
.rp-display-grid--basic {
  grid-template-columns: repeat(auto-fill, minmax(132px, 1fr));
}
.rp-display-card {
  display: flex;
  align-items: center;
  gap: 7px;
  min-height: 34px;
  padding: 7px 9px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-sm);
  background: var(--surface-sunken);
  color: var(--text-primary);
  font-size: 11.5px;
  cursor: pointer;
}
.rp-display-card:hover {
  border-color: color-mix(in srgb, var(--display-accent) 55%, var(--border-default));
  background: var(--surface-hover);
}
.rp-display-card input {
  accent-color: var(--display-accent);
  flex-shrink: 0;
}
.rp-display-card em {
  margin-left: auto;
  font-style: normal;
  font-size: 10px;
  color: var(--text-muted);
}

/* iOS group ở modal display filter — cyan accent giống button ngoài. */
.rp-display-group--ios .rp-display-group__head strong {
  color: #06b6d4;
}
.rp-display-card--ios {
  border-color: rgba(6,182,212,0.3);
  background: rgba(6,182,212,0.05);
}
.rp-display-card--ios:hover {
  border-color: rgba(6,182,212,0.6);
  background: rgba(6,182,212,0.1);
}
.rp-display-card--ios input {
  accent-color: #06b6d4;
}

/* Banner cảnh báo: bản Instagram đang phát triển, API tạm khóa */
.api-dev-notice {
  background: var(--accent-bg);
  border-left: 3px solid var(--accent);
  color: var(--text-secondary);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
  margin-bottom: var(--space-2);
}

/* Context menu chuột phải vùng API REG */
.rp-ctxmenu-overlay { position: fixed; inset: 0; z-index: 998; }
.rp-ctxmenu {
  position: fixed; z-index: 999; min-width: 150px;
  background: var(--surface-elevated); border: 1px solid var(--border-default);
  border-radius: var(--radius-sm); box-shadow: 0 6px 24px rgba(0,0,0,0.25);
  padding: 4px;
}
.rp-ctxmenu__submenu { position: relative; }
.rp-ctxmenu__sep {
  padding: 3px 10px 2px; font-size: 10px; color: var(--text-secondary);
  opacity: 0.7; pointer-events: none; user-select: none;
}
.rp-ctxmenu__item {
  display: block; width: 100%; text-align: left; padding: 6px 10px;
  font-size: 12px; color: var(--text-primary); background: transparent;
  border: none; border-radius: 4px; cursor: pointer;
}
.rp-ctxmenu__item--submenu {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.rp-ctxmenu__item:hover { background: var(--surface-hover); }
.rp-ctxmenu__flyout {
  display: none;
  position: absolute;
  left: calc(100% + 4px);
  top: -4px;
  min-width: 180px;
  max-height: min(420px, calc(100vh - 24px));
  overflow-y: auto;
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  box-shadow: 0 6px 24px rgba(0,0,0,0.25);
  padding: 4px;
}
.rp-ctxmenu__submenu:hover .rp-ctxmenu__flyout,
.rp-ctxmenu__submenu:focus-within .rp-ctxmenu__flyout {
  display: block;
}

.rp-ua-code {
  font-family: 'Consolas', 'Fira Code', monospace;
  font-size: 11.5px;
  color: #fff;
  white-space: pre-wrap;
  word-break: break-all;
  line-height: 1.4;
  display: block;
}

/* ══ Token type row ═══════════════════════════════════════════════════════════ */
.rp-token-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}
.rp-token-row label { color: var(--text-secondary); font-weight: 500; flex-shrink: 0; }
.vr-select--sm { padding: 2px 6px; font-size: 11px; height: auto; }
.rp-token-hint { font-size: 10px; color: var(--text-muted); font-style: italic; }

/* ══ Banner thông báo Nguồn xác thực đã chuyển sang tab riêng ════════════ */
.rp-auth-moved-banner {
  display: flex;
  align-items: center;
  gap: 14px;
  margin: 12px 18px 0;
  padding: 10px 14px;
  background: rgba(225,48,108,0.06);
  border: 1px dashed rgba(225,48,108,0.35);
  border-radius: var(--radius-lg);
}
.rp-auth-moved-banner__icon { font-size: 22px; }
.rp-auth-moved-banner__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1 1 auto;
  min-width: 0;
}
.rp-auth-moved-banner__text strong {
  font-size: var(--font-size-sm);
  color: var(--text-primary);
  font-weight: 700;
}
.rp-auth-moved-banner__text span {
  font-size: var(--font-size-xs);
  color: var(--text-secondary);
}
.rp-auth-moved-banner__btn {
  padding: 6px 14px;
  background: var(--brand-primary);
  color: #000;
  border: none;
  border-radius: var(--radius-md);
  font-size: var(--font-size-xs);
  font-weight: 700;
  cursor: pointer;
  transition: filter var(--transition-fast);
}
.rp-auth-moved-banner__btn:hover { filter: brightness(1.1); }

/* ══ UA label ═══════════════════════════════════════════════════════════════ */
.rp-presets__label {
  font-size: 10px;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  white-space: nowrap;
  align-self: center;
}

/* ══ Summary toggle & drawer ════════════════════════════════════════════════ */
.rp-summary-toggle {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-default);
  background: transparent;
  color: var(--text-muted);
  font-size: var(--font-size-xs);
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.15s;
  flex-shrink: 0;
}
.rp-summary-toggle:hover { border-color: var(--brand-primary); color: var(--brand-primary); }
.rp-summary-toggle--open {
  border-color: var(--brand-primary);
  color: var(--brand-primary);
  background: rgba(225,48,108,0.08);
}
.rp-summary-drawer {
  flex-shrink: 0;
  border-bottom: 1px solid var(--border-default);
}

.rp-page__body {
  flex: 1;
  overflow-y: auto;
  /* bottom padding lớn hơn để section UA cuối không bị status bar che */
  padding: 8px 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-content: start;
}

.rp-main-col { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; align-items: start; }
.rp-main-col--single { grid-template-columns: 1fr; }

/* ══ Section cards ═══════════════════════════════════════════════════════════ */
.rp-section {
  border: 1px solid var(--border-default);
  border-radius: var(--radius-md);
  overflow: hidden;
  min-width: 0;
}

.rp-section__header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 12px;
  min-height: 34px;
  background: var(--surface-elevated);
  border-bottom: 1px solid var(--border-default);
}
.rp-section__num {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--brand-primary);
  color: white;
  font-size: var(--font-size-xs);
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
/* Section color theming */
.rp-section--verify { border-left: 3px solid var(--accent); }
.rp-section--verify .rp-section__num { background: var(--accent); }
.rp-section--verify .rp-section__title { color: var(--accent); }
.rp-section--reg { border-left: 3px solid #f97316; }
.rp-section--reg .rp-section__num { background: #f97316; }
.rp-section--reg .rp-section__title { color: #f97316; }
.rp-section__title {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}
.rp-section__toggle {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
  font-size: var(--font-size-sm);
  font-weight: 600;
}
.rp-section__toggle input { accent-color: var(--brand-primary); }
.rp-section__badge {
  margin-left: auto;
  flex-shrink: 0;
  font-size: var(--font-size-xs);
  font-weight: 600;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  white-space: nowrap;
}
.badge--on    { background: var(--success-bg); color: var(--success-text); }
.badge--off   { background: var(--surface-sunken); color: var(--text-muted); }
.badge--api   { background: var(--info-bg); color: var(--info-text); }
.badge--folder { background: var(--surface-sunken); color: var(--text-muted); }
.badge--neutral { background: var(--surface-sunken); color: var(--text-secondary); }

.rp-section__body {
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

/* ══ Subsections (inside a section) ═════════════════════════════════════════ */
.rp-subsection {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-subtle, var(--border-default));
}
.rp-subsection:last-child { border-bottom: none; padding-bottom: 0; }
.rp-subsection__header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}
.rp-subsection--last { border-bottom: none; padding-bottom: 0; }
.rp-source-folder-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.rp-source-folder-row .vr-input { flex: 1; min-width: 0; }
.rp-avatar-folder-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-top: var(--space-1);
}
.rp-avatar-folder-row .vr-input { flex: 1; min-width: 0; }
.rp-subsection__label {
  font-size: var(--font-size-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
}
/* Inline variant: label + checkboxes cùng 1 hàng (dùng cho Tùy chọn nâng cao) */
.rp-subsection--inline {
  flex-direction: row;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-3);
}
.rp-subsection__label--inline {
  text-transform: none;
  letter-spacing: 0;
  white-space: nowrap;
  color: var(--text-secondary);
}

/* ══ Grid layouts ═══════════════════════════════════════════════════════════ */
.rp-timing-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px 10px;
  margin-bottom: 6px;
}
.rp-timing-grid--dense {
  grid-template-columns: repeat(5, 1fr);
  gap: 6px 8px;
}
.rp-timing-grid--3 {
  grid-template-columns: repeat(3, 1fr);
}
.rp-field--compact { gap: var(--space-2); }
.rp-field--compact label { min-width: 110px; font-size: var(--font-size-xs); }
.rp-checks-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 12px;
  padding-top: 6px;
  border-top: 1px solid var(--border-subtle);
}
.rp-checks-row--merged { align-items: center; }
.rp-checks-sep {
  color: var(--border-strong);
  font-size: 14px;
  line-height: 1;
  flex-shrink: 0;
}
.rp-checks-group-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted);
  white-space: nowrap;
  flex-shrink: 0;
}

/* ══ Proxy tabs panel (TempMail + Gmail gộp) ══ */
.rp-proxy-tabs {
  margin-top: var(--space-2);
  padding: var(--space-2);
  background: var(--surface-subtle);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
}
.rp-proxy-tabs__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-1);
}
.rp-proxy-tabs__buttons {
  display: flex;
  gap: var(--space-1);
}
.rp-proxy-tab {
  padding: 4px 10px;
  font-size: var(--font-size-xs);
  font-weight: 500;
  color: var(--text-muted);
  background: transparent;
  border: 1px solid var(--border-default);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.15s;
}
.rp-proxy-tab:hover {
  color: var(--text-primary);
  background: var(--surface-hover);
}
.rp-proxy-tab--active {
  color: var(--primary-text);
  background: var(--primary-subtle);
  border-color: var(--primary-solid);
}
.rp-proxy-tabs__hint {
  font-size: 11px;
  color: var(--text-muted);
  margin-bottom: var(--space-1);
  font-style: italic;
}
.rp-proxy-textarea {
  font-family: 'Consolas', 'Courier New', monospace;
  font-size: 12px;
  resize: vertical;
  min-height: 120px;
  width: 100%;
  display: block;
  box-sizing: border-box;
  white-space: pre;
  overflow-x: auto;
}
.rp-proxy-tabs__save-status {
  font-size: 11px;
  color: var(--text-muted);
  font-style: italic;
  min-width: 80px;
  text-align: right;
}

.rp-grid-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px 12px;
  align-items: start;
}
.rp-grid-2 .rp-field--col,
.rp-grid-2 .rp-field--span,
.rp-grid-2 .rp-stock-row,
.rp-grid-2 .rp-stock-info,
.rp-grid-2 .rp-hint,
.rp-grid-2 .rp-field-group,
.rp-grid-2 > details {
  grid-column: 1 / -1;
}
/* grid-hint sits in col 2, not full-span — override the .rp-hint rule above */
.rp-grid-2 .rp-grid-hint {
  grid-column: auto;
}
/* Sections đang ở half-width (2-col outer grid) → inner grid về 1 cột để input có đủ chỗ */
.rp-section .rp-grid-2 {
  grid-template-columns: 1fr;
}
/* Ngoại lệ: các cặp field ngắn vẫn cho đứng cạnh nhau */
.rp-section .rp-grid-2--paired {
  grid-template-columns: 1fr 1fr;
}
/* 3-col variant: API selector rộng hơn, 2 number input thu gọn */
.rp-section .rp-grid-2--triple {
  grid-template-columns: 2fr 1fr 1fr;
}
.rp-grid-2--triple .rp-field > label:first-child { min-width: 0; flex-shrink: 0; }
.rp-grid-2--triple > .rp-field { min-width: 0; gap: var(--space-2); }
/* Timing grid trong section: force 3 cột (user request).
   Mỗi ô = 1 cột grid riêng để label-input KHÔNG kéo nhau lệch.
   Label dùng grid sub-column auto-fit (1fr label + 70px input fixed). */
.rp-section .rp-timing-grid {
  grid-template-columns: repeat(3, 1fr);
  align-items: center;
}
.rp-section .rp-timing-grid .rp-field--compact {
  display: grid;
  grid-template-columns: 1fr 64px;
  align-items: center;
  gap: 5px;
  min-width: 0;
}
.rp-section .rp-timing-grid--dense .rp-field--compact {
  grid-template-columns: 1fr 56px;
  gap: 4px;
}
.rp-section .rp-timing-grid--dense .rp-field--compact label {
  font-size: 10.5px;
}
.rp-section .rp-timing-grid .rp-field--compact label {
  min-width: 0;
  font-size: 11px;
  text-align: right;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.rp-section .rp-timing-grid .rp-field--compact .vr-input--num {
  width: 100%;
}
/* Reg settings grid:
   section > ~586px → 50%-8px > 285px → 2 cột; còn lại → 1 cột (tránh cramped) */
.rp-section .rp-grid-2--adapt {
  grid-template-columns: repeat(auto-fill, minmax(max(calc(50% - 8px), 285px), 1fr));
}

/* ══ Fields ═════════════════════════════════════════════════════════════════ */
.rp-field {
  display: flex;
  align-items: center;
  gap: 8px;
}
.rp-field > label:first-child {
  font-size: 12px;
  color: var(--text-secondary);
  white-space: nowrap;
  min-width: 120px;
  flex-shrink: 0;
}
.rp-field--col {
  flex-direction: column;
  align-items: stretch;
}
.rp-field--col > label { margin-bottom: var(--space-1); font-size: var(--font-size-sm); color: var(--text-secondary); }

.rp-field-group { display: flex; flex-direction: column; gap: var(--space-2); }
.rp-label { font-size: var(--font-size-sm); color: var(--text-secondary); }

.rp-output-row { display: flex; align-items: center; gap: var(--space-3); }
.rp-output-label { font-size: var(--font-size-sm); color: var(--text-secondary); white-space: nowrap; min-width: 140px; flex-shrink: 0; }

.rp-count { font-size: var(--font-size-xs); color: var(--text-muted); font-weight: normal; }

/* ══ Radio / checkbox ═══════════════════════════════════════════════════════ */
.rp-radio { display: flex; align-items: center; gap: 6px; font-size: 12px; cursor: pointer; }
.rp-radio input { accent-color: var(--brand-primary); }
.rp-checkbox { display: flex; align-items: center; gap: 6px; font-size: 12px; cursor: pointer; }
.rp-checkbox input { accent-color: var(--brand-primary); }

/* ══ Hints ══════════════════════════════════════════════════════════════════ */
.rp-hint {
  font-size: var(--font-size-xs);
  color: var(--text-muted);
  padding: var(--space-2) var(--space-3);
  background: var(--surface-sunken);
  border-radius: var(--radius-sm);
  border-left: 2px solid var(--border-default);
}
.rp-hint--ok { border-color: var(--success-text); color: var(--success-text); background: var(--success-bg); }
.rp-hint--error { border-color: var(--danger-solid); color: var(--danger-text); background: var(--danger-bg); }

/* ══ Stock info ═════════════════════════════════════════════════════════════ */
.rp-stock-info {
  display: flex;
  gap: var(--space-4);
  align-items: center;
  padding: var(--space-2) var(--space-3);
  background: var(--success-bg);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
  color: var(--success-text);
}
.rp-stock-row { display: flex; align-items: center; gap: var(--space-3); flex-wrap: wrap; }

/* ══ Details/Summary (UA list) ══════════════════════════════════════════════ */
.rp-details { border: 1px solid var(--border-default); border-radius: var(--radius-sm); }
.rp-details__summary {
  padding: var(--space-2) var(--space-3);
  font-size: var(--font-size-sm);
  font-weight: 500;
  color: var(--text-secondary);
  background: var(--surface-elevated);
  border-radius: var(--radius-sm);
  cursor: pointer;
  user-select: none;
  list-style: none;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  transition: background 0.15s, color 0.15s;
}
.rp-details__summary::-webkit-details-marker { display: none; }
.rp-details__caret {
  margin-left: auto;
  flex-shrink: 0;
  color: var(--text-muted);
  transition: transform 0.2s ease;
}
.rp-details[open] > .rp-details__summary {
  border-bottom-left-radius: 0;
  border-bottom-right-radius: 0;
  border-bottom: 1px solid var(--border-subtle);
  color: var(--text-primary);
}
.rp-details[open] > .rp-details__summary .rp-details__caret {
  transform: rotate(90deg);
}
.rp-details__summary:hover {
  background: var(--surface-hover-subtle);
  color: var(--text-primary);
}
.rp-details__body { padding: var(--space-3); }

.rp-shared-badge {
  font-size: 10px; font-weight: 600;
  padding: 1px 6px; border-radius: 8px;
  background: rgba(171,71,188,0.12); color: #ce93d8;
  white-space: nowrap; flex-shrink: 0;
}
.rp-details--shared { grid-column: 1 / -1; }

/* ══ Advanced section (details/summary) ════════════════════════════════════ */
.rp-section--advanced {
  opacity: 0.85;
  border-style: dashed;
}
.rp-section--advanced[open] { opacity: 1; }

.rp-section__header--clickable {
  cursor: pointer;
  user-select: none;
  list-style: none;
}
.rp-section__header--clickable::-webkit-details-marker { display: none; }
.rp-section__num--muted {
  background: var(--surface-sunken);
  color: var(--text-muted);
}
.rp-grid-hint {
  align-self: center;
}

/* ══ Link button ════════════════════════════════════════════════════════════ */
.rp-link-btn {
  background: none;
  border: none;
  color: var(--brand-primary);
  font-size: var(--font-size-xs);
  cursor: pointer;
  text-decoration: underline;
  padding: 0;
}

/* ══ Reuse shared form control styles ═══════════════════════════════════════ */
.vr-input {
  flex: 1;
  min-width: 0; /* cho phép input co khi grid hẹp — tránh overflow cắt cuối */
  background: var(--surface-sunken);
  border: 1px solid var(--border-default);
  color: var(--text-primary);
  padding: 5px 8px;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
  font-family: inherit;
  outline: none;
}
.vr-select {
  min-width: 0; /* đồng bộ với vr-input — tránh select tràn khỏi grid */
}
/* Trong grid 2 cột, giảm label width để input có đủ chỗ thở */
.rp-grid-2--paired .rp-field > label:first-child { min-width: 0; flex-shrink: 0; }
/* Grid cell phải được phép shrink dưới content-width để input không tràn panel */
.rp-grid-2--paired > .rp-field {
  min-width: 0;
  gap: var(--space-2); /* label ↔ input gap nhỏ lại — tránh padding thừa giữa cột */
}
.rp-reg-settings-grid {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;
}
.rp-reg-settings-grid .rp-field--lead-domain,
.rp-reg-settings-grid .rp-field--password-reg {
  flex: 1 1 calc(50% - 8px);
  min-width: 0;
}
.rp-grid-2--paired > .rp-field--name-locale {
  flex: 0 0 auto;
}
.rp-field--name-locale .vr-select {
  width: 128px;
  flex: none;
}
.rp-rotate-inline {
  display: flex;
  align-items: center;
  gap: 8px 10px;
  flex-wrap: wrap;
  flex: 1 1 260px;
  min-width: 0;
  padding-top: 2px;
}
.rp-rotate-num {
  width: 58px !important;
  text-align: center;
}
.vr-input:focus { border-color: var(--border-focus); }
.vr-input:disabled { opacity: 0.5; cursor: not-allowed; }
.vr-input--num { width: 64px; flex: none; text-align: right; }
.vr-input--sm { width: 90px; flex: none; }
.vr-input--error { border-color: var(--danger-solid) !important; }

.vr-select {
  flex: 1;
  background: var(--surface-sunken);
  border: 1px solid var(--border-default);
  color: var(--text-primary);
  padding: 5px 8px;
  border-radius: var(--radius-sm);
  font-size: var(--font-size-sm);
  font-family: inherit;
  outline: none;
  cursor: pointer;
}
.vr-select:focus { border-color: var(--border-focus); }

.vr-textarea {
  width: 100%;
  max-width: 100%;
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
  min-height: 80px;
}
.vr-textarea:focus { border-color: var(--border-focus); }
.vr-textarea--mono { font-family: var(--font-mono); font-size: var(--font-size-xs); }

.vr-btn {
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  font-weight: 600;
  border: 1px solid var(--border-default);
  cursor: pointer;
  transition: opacity var(--transition-fast);
}
.vr-btn--browse { background: var(--brand-primary); border-color: var(--brand-primary); color: white; flex: none; padding: var(--space-2) var(--space-3); }
.vr-btn--check {
  background: var(--info-bg);
  border: 1px solid var(--info-text);
  color: var(--info-text);
  padding: var(--space-1) var(--space-3);
  border-radius: var(--radius-sm);
  font-size: var(--font-size-xs);
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
}
.vr-btn--check:disabled { opacity: 0.4; cursor: not-allowed; }

.rp-btn {
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-md);
  font-size: var(--font-size-sm);
  font-weight: 600;
  border: 1px solid var(--border-default);
  cursor: pointer;
  min-width: 80px;
}
.rp-btn--primary { background: var(--brand-primary); border-color: var(--brand-primary); color: white; }
.rp-btn--primary:hover { opacity: 0.9; }
.rp-btn--danger { background: var(--danger-solid); border-color: var(--danger-solid); color: white; }
.rp-btn--danger:hover { opacity: 0.9; }

/* Save-status indicator cho auto-save */
.rp-save-status {
  display: inline-flex;
  align-items: center;
  padding: var(--space-1) var(--space-3);
  font-size: var(--font-size-sm);
  color: var(--text-muted);
  transition: color 0.2s;
}
.rp-save-status[data-status="saving"]  { color: var(--brand-primary); }
.rp-save-status[data-status="saved"]   { color: var(--success-solid, #16a34a); }
.rp-save-status[data-status="error"]   { color: var(--danger-solid); }

.vr-unit { font-size: var(--font-size-sm); color: var(--text-muted); white-space: nowrap; }
.vr-stock-badge { font-size: var(--font-size-xs); font-weight: 600; padding: 3px 10px; border-radius: var(--radius-sm); }
.vr-stock-badge--ok { background: var(--success-bg); color: var(--success-text); }
.vr-stock-badge--empty { background: var(--danger-bg); color: var(--danger-text); }

/* ══ Layout fixes ═══════════════════════════════════════════════════════════ */
/* Full-span checkboxes in 2-col grid (sendAgainCode, checkLiveDieEnabled) */
.rp-grid-2 .rp-checkbox { grid-column: 1 / -1; }

/* No inner border when section is collapsed (no body) */
.rp-section__header--no-body { border-bottom: none; }

/* Disabled fieldset wrapper — strip native fieldset chrome, apply dim effect */
.rp-section__disable-wrap {
  border: none;
  padding: 0;
  margin: 0;
  min-width: 0;
}
.rp-section__disable-wrap:disabled {
  opacity: 0.45;
  pointer-events: none;
  cursor: not-allowed;
}

/* Clickable section header for accordion */
.rp-section__header--toggle {
  cursor: pointer;
  user-select: none;
}
.rp-section__header--toggle:hover {
  background: var(--surface-hover-subtle);
}
.rp-section__caret {
  color: var(--text-muted);
  flex-shrink: 0;
  margin-left: var(--space-1);
}

/* Inline ok indicator after path inputs */
.rp-ok-icon {
  color: var(--success-text);
  font-size: var(--font-size-sm);
  font-weight: 700;
  flex-shrink: 0;
}

/* ══ Mode empty state ═══════════════════════════════════════════════════════ */
.rp-mode-empty {
  grid-column: 1 / -1;
  padding: var(--space-4) var(--space-4);
  text-align: center;
  font-size: var(--font-size-sm);
  color: var(--text-muted);
  background: var(--surface-sunken);
  border: 1px dashed var(--border-default);
  border-radius: var(--radius-md);
}
.rp-mode-empty strong {
  color: var(--text-secondary);
}

/* Compact ok hint (replaces verbose full-path block) */
.rp-hint--compact {
  padding: 2px var(--space-2);
  border-left-width: 2px;
  font-size: 10px;
  line-height: 1.4;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ══ Horizontal radio row (Register createType) ═════════════════════════════ */
.rp-field--radios {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.rp-field--radios .rp-label {
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  white-space: nowrap;
}

/* ══ Details with left-indent (inside subsection, after grid) ═══════════════ */
.rp-details--inset {
  margin-top: var(--space-2);
}

/* Details meta (inline value preview in summary) */
.rp-details__meta {
  margin-left: var(--space-2);
  font-style: italic;
}

/* ══ Shared bottom row ═══════════════════════════════════════════════════════ */
.rp-shared-row {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: var(--space-3);
  align-items: start;
}
.rp-cookie-initial {
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  background: var(--surface-elevated);
  overflow: hidden;
  width: 380px;
  flex-shrink: 0;
}
/* Cookie Initial body */
.rp-ci-body {
  padding: var(--space-2) var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.rp-ci-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}
.rp-subsection--ci {
  gap: 6px;
}
.rp-ci-head {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.rp-ci-head .rp-subsection__label {
  flex: none;
}
.rp-ci-desc {
  font-size: 11px;
  color: var(--text-muted);
}
.rp-ci-group {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.rp-ci-group-label {
  width: 58px;
  flex: none;
  color: var(--text-muted);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
}
.rp-ci-method {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.rp-ci-row {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.rp-ci-row--compact {
  gap: 8px 12px;
}
.rp-ci-filebar {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.rp-mini-btn {
  border: 1px solid var(--border-default);
  background: var(--surface-elevated);
  color: var(--text-primary);
  border-radius: var(--radius-sm);
  padding: 5px 9px;
  font-size: 12px;
  line-height: 1;
  white-space: nowrap;
  cursor: pointer;
}
.rp-mini-btn:hover {
  border-color: var(--border-focus);
}
.rp-ci-count {
  color: var(--success-solid);
  font-size: 12px;
  white-space: nowrap;
}
.rp-ci-count--error {
  color: var(--danger-solid);
}
.rp-ci-error {
  color: var(--danger-solid);
  font-size: 11px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.rp-ci-limit {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-secondary);
  cursor: pointer;
  white-space: nowrap;
}
.rp-ci-limit-num {
  width: 64px !important;
  text-align: center;
}
.rp-ci-file {
  display: flex;
  gap: var(--space-1);
}
.rp-ci-file .vr-input {
  flex: 1;
  min-width: 0;
}
.rp-ci-browse-btn {
  padding: 2px 8px;
  font-size: 13px;
  flex-shrink: 0;
}
.rp-ci-hint {
  font-size: 11px;
  color: var(--text-secondary);
  line-height: 1.4;
  padding: 2px 0;
}
.rp-ci-hint code {
  font-family: var(--font-mono, monospace);
  background: var(--surface-elevated);
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 10px;
}

/* ══ Auth source shared section ══════════════════════════════════════════════ */
.rp-auth-source {
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  background: var(--surface-elevated);
  overflow: hidden;
  flex-shrink: 0;
}
.rp-auth-source__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--border-default);
  background: rgba(255, 255, 255, 0.08);
}
.rp-auth-source__title-row {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
}
.rp-auth-source__title {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}
.rp-auth-source__desc {
  font-size: 11px;
  color: var(--text-muted);
}
.rp-auth-tabs {
  display: flex;
  gap: var(--space-1);
}
.rp-auth-tab {
  padding: 3px 12px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-default);
  background: transparent;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}
.rp-auth-tab:hover:not(:disabled):not(.rp-auth-tab--active) {
  border-color: var(--brand-primary);
  color: var(--brand-primary);
}
.rp-auth-tab--active {
  background: var(--brand-primary);
  border-color: var(--brand-primary);
  color: #fff;
}
.rp-auth-tab--active:hover:not(:disabled) {
  background: var(--brand-primary-hover, var(--brand-primary));
  color: #fff;
}
.rp-auth-tab--soon {
  opacity: 0.45;
  cursor: not-allowed;
}
.rp-auth-source__body {
  padding: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.rp-auth-row {
  display: flex;
  align-items: flex-end;
  gap: var(--space-3);
  flex-wrap: wrap;
}
.rp-field--provider {
  min-width: 220px;
  flex: 1;
}
.rp-auth-live-check {
  white-space: nowrap;
  padding-bottom: 4px;
}

/* Mail category selector */
.rp-mail-cats {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}
.rp-mail-cats__group {
  display: flex;
  gap: var(--space-1);
}
.rp-mail-cat {
  padding: 4px 14px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-default);
  background: transparent;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s;
}
.rp-mail-cat:hover:not(.rp-mail-cat--active) {
  border-color: var(--brand-primary);
  color: var(--brand-primary);
}
.rp-mail-cat--active {
  background: var(--brand-primary);
  border-color: var(--brand-primary);
  color: #fff;
}
.rp-mail-cat--active:hover {
  background: var(--brand-primary-hover, var(--brand-primary));
  color: #fff;
}

/* Provider dropdown — compact select replaces button grid */
.rp-provider-select {
  max-width: 360px;
  width: 100%;
}

/* Temp mail config — Domain + Token side-by-side, label stacked above input */
.rp-tempmail-config {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3);
  margin-top: var(--space-2);
}
.rp-tempmail-cfg-item label {
  display: flex;
  align-items: baseline;
  gap: var(--space-1);
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
}
.rp-tempmail-cfg-item label .rp-field-hint {
  font-size: 11px;
  color: var(--text-muted);
  font-weight: normal;
}
/* Khi không có ô Domain (provider không support) → Token chiếm full width */
.rp-tempmail-config:has(.rp-tempmail-cfg-item:only-child) {
  grid-template-columns: 1fr;
}
@media (max-width: 900px) {
  .rp-tempmail-config {
    grid-template-columns: 1fr;
  }
}

/* Mail provider buttons (legacy — reserved cho grid nếu cần lại) */
.rp-mail-providers {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(110px, 1fr));
  gap: 4px;
  margin-top: 2px;
}
.rp-mail-provider {
  padding: 3px 8px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-default);
  background: var(--surface-elevated);
  color: var(--text-secondary);
  font-size: 11px;
  cursor: pointer;
  transition: all 0.15s;
  text-align: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.rp-mail-provider:hover {
  border-color: var(--brand-primary);
  color: var(--brand-primary);
}
.rp-mail-provider--active {
  background: rgba(34,197,94,0.12);
  border-color: var(--brand-primary);
  color: var(--brand-primary);
  font-weight: 600;
}
.rp-field--domain { flex: 1; }
.rp-field-hint { font-size: 10px; color: var(--text-muted); font-weight: 400; }
.rp-domain-textarea { min-height: 64px; resize: vertical; font-size: 12px; }
.rp-phone-notice {
  margin-top: 8px;
  padding: 8px 12px;
  background: rgba(251, 191, 36, 0.1);
  border: 1px solid rgba(251, 191, 36, 0.3);
  border-radius: var(--radius-sm);
  font-size: 11px;
  color: var(--text-secondary);
  line-height: 1.5;
}
.rp-coming-soon-badge {
  font-size: 10px;
  padding: 2px 8px;
  background: rgba(251, 191, 36, 0.15);
  color: #f59e0b;
  border-radius: 10px;
  font-weight: 600;
}
.rp-auth-source__body--phone {
  align-items: center;
  justify-content: center;
  min-height: 80px;
}
.rp-auth-phone-placeholder {
  color: var(--text-muted);
  font-size: 13px;
  text-align: center;
  padding: var(--space-4);
}

/* ══ Result folder bar (footer cố định, ngoài scroll area) ══════════════════ */
.rp-result-folder-bar {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-4);
  background: var(--surface-elevated);
  border-top: 1px solid var(--border-default);
  border-left: 3px solid var(--brand-primary);
  flex-shrink: 0;
}
.rp-result-folder-bar__label {
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-secondary);
  white-space: nowrap;
  flex-shrink: 0;
}

/* ══ Shared footer (Cookie Initial + Nguồn xác thực) ══════════════════════ */
.rp-shared-footer {
  flex-shrink: 0;
  display: flex;
  align-items: stretch;
  border-top: 1px solid var(--border-default);
  background: var(--surface-base);
}
.rp-sf-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
}
.rp-sf-panel--ci { flex: 0 0 auto; min-width: 320px; align-self: flex-start; }
.rp-sf-panel--auth { flex: 1 1 0; min-width: 0; }
.rp-sf-panel__label {
  font-size: var(--font-size-xs);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--text-muted);
}
.rp-sf-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}
.rp-sf-divider {
  width: 1px;
  background: var(--border-default);
  flex-shrink: 0;
}
.rp-sf-ci {
  display: flex;
  flex-direction: column;
  gap: 7px;
}
.rp-sf-auth-body {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

/* ── Auto-upload row (chung cho reg + verify) ── */
.rp-autoupload-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 7px 10px;
  margin-top: 8px;
  background: var(--brand-primary-bg);
  border: 1px solid var(--brand-primary-border);
  border-radius: var(--radius-md);
  gap: 10px;
}
.rp-autoupload-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-primary);
  cursor: pointer;
  user-select: none;
}
.rp-autoupload-label input[type="checkbox"] { cursor: pointer; }
.rp-autoupload-icon { color: var(--brand-primary); flex-shrink: 0; }
.rp-autoupload-link {
  background: transparent;
  border: none;
  color: var(--brand-primary);
  font-size: 11px;
  font-weight: 600;
  cursor: pointer;
  padding: 2px 4px;
  border-radius: var(--radius-sm);
  white-space: nowrap;
  flex-shrink: 0;
  outline: none;
}
.rp-autoupload-link:hover { text-decoration: underline; }

@media (max-width: 1180px) {
  .rp-save-status {
    display: none;
  }
  .rp-mode-checks--toolbar {
    order: 3;
    flex-basis: 100%;
  }
  .rp-toolbar__actions {
    margin-left: auto;
  }
}

@media (max-width: 900px) {
  .rp-main-col {
    grid-template-columns: 1fr;
  }
  .rp-section .rp-grid-2--paired,
  .rp-section .rp-timing-grid {
    grid-template-columns: 1fr;
  }
  .rp-section .rp-timing-grid .rp-field--compact {
    grid-template-columns: minmax(120px, 1fr) 64px;
  }
}

</style>
