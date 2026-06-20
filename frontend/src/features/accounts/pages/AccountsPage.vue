<script setup lang="ts">
// AccountsPage.vue — Trang Accounts chính
// 2 trạng thái chọn giống WeBM:
//   - Highlighted (bôi đen): click/drag row → xanh dương, dùng cho copy/context menu
//   - Checked (chọn/cChose): checkbox/Space/double-click → dùng cho Run/Delete

import { ref, onMounted, onUnmounted, onActivated, onDeactivated, computed, nextTick, watch } from 'vue'
import { Users, FolderOpen, Key, X } from 'lucide-vue-next'
import { getSettingsService, getVerifyRunnerService, getEventBusService, getFileDialogService, getInteractionService } from '@/services/client'
import DataGrid from '@/components/grid/DataGrid.vue'
import ContextMenu from '@/components/grid/ContextMenu.vue'
import AccountsToolbar from '@/features/accounts/components/AccountsToolbar.vue'
import AccountsDetailPanel from '@/features/accounts/components/AccountsDetailPanel.vue'
import GeneralSettingsModal from '@/features/accounts/components/GeneralSettingsModal.vue'
import AccountsImportDialog from '@/features/accounts/components/AccountsImportDialog.vue'
import { useAccountsStore } from '@/features/accounts/store/useAccountsStore'
import { useAppStore } from '@/stores/app.store'
import { usePreferencesStore } from '@/stores/preferences.store'
import { useUploadLogStore } from '@/stores/uploadLog.store'
import { useDataGrid } from '@/composables/useDataGrid'
import { useSelection } from '@/composables/useSelection'
import { useColumnVisibility } from '@/composables/useColumnVisibility'
import { useContextMenu } from '@/composables/useContextMenu'
import { useClipboard } from '@/composables/useClipboard'
import { buildAccountContextMenu } from '@/constants/accountContextMenu'
import { ACCOUNT_COLUMNS } from '@/constants/columns'
import type { Account } from '@/services/contracts'
import type { GeneralConfig, IpConfig } from '@/types/settings.types'
import { DEFAULT_GENERAL_CONFIG, DEFAULT_IP_CONFIG } from '@/types/settings.types'

defineOptions({ name: 'AccountsPage' })

const accountsStore = useAccountsStore()
const appStore = useAppStore()
const uploadLogStore = useUploadLogStore()
const columnVis = useColumnVisibility(ACCOUNT_COLUMNS)
const { visibleColumns } = columnVis
const selection = useSelection<Account>()

// Giới hạn tối đa 300 dòng hiển thị trong bảng — tránh lag khi có hàng nghìn accounts
// Cap render rows ở 2000 — đủ cho file mode load 500-1500 acc + tick chọn.
// Vẫn giữ cap để tránh DOM bloat nếu user load file 100k+ rows.
const MAX_DISPLAY_ROWS = 2000
const displayedAccounts = computed(() =>
  accountsStore.accounts.slice(0, MAX_DISPLAY_ROWS) as unknown as Record<string, unknown>[]
)
const grid = useDataGrid<Record<string, unknown>>({
  items: displayedAccounts,
})
const ctxMenu = useContextMenu()
const clipboard = useClipboard()

// Stats bar
const statusBarSlotReady = ref(false)
const pageStats = computed(() => {
  const accs = accountsStore.accounts
  const live = accs.filter(a => a.status === 'live').length
  const die = accs.filter(a => a.status === 'die' || a.status === 'checkpoint').length
  const unknown = accs.filter(a => !a.status || a.status === 'unknown' || a.status === 'new').length
  return {
    live,
    die,
    unknown,
    total: accs.length,
    highlighted: selection.highlightedIds.value.size,
    selected: selection.checkedCount.value,
  }
})

// Ref cho grid content container — dùng để đọc viewport height khi reactivate
const gridContentRef = ref<HTMLElement | null>(null)

// Drag-select state
const isDragging = ref(false)
const dragStartId = ref<number | null>(null)

// Verify run state — dùng từ store để persist khi chuyển tab
const isVerifyRunning = computed({
  get: () => accountsStore.isVerifyRunning,
  set: (v) => { accountsStore.isVerifyRunning = v },
})
// CloneHV mode — cho phép Run không cần chọn account
const cloneHvMode = ref(false)
// File streaming mode — đọc từ thư mục, cũng không cần chọn account trước
const fileMode = ref(false)

// Column groups — giống ViewSettingsPage
const columnGroups = [
  { label: 'Tài khoản', keys: ['uid', 'fullData', 'password', 'twofa', 'email', 'passMail', 'mailRecovery', 'cookie', 'token'] },
  { label: 'Trạng thái', keys: ['status', 'checkpoint', 'statusAds', 'bm', 'tkqc', 'chatSupport'] },
  { label: 'Chạy & Khác', keys: ['avatar', 'cover', 'phone', 'proxy', 'userAgent', 'note', 'noteRun', 'importTime', 'category', 'lastRun', 'runProxy', 'activity'] },
]

const { isColumnVisible, toggleColumn } = usePreferencesStore()

function getGroupColumns(keys: string[]) {
  return ACCOUNT_COLUMNS.filter(c => keys.includes(c.key))
}
function isGroupAllChecked(keys: string[]) {
  return keys.every(k => isColumnVisible(k))
}
function toggleGroup(keys: string[], checked: boolean) {
  keys.forEach(k => {
    const visible = isColumnVisible(k)
    if (checked && !visible) toggleColumn(k)
    if (!checked && visible) toggleColumn(k)
  })
}
const totalVisibleCols = computed(() => ACCOUNT_COLUMNS.filter(c => isColumnVisible(c.key)).length)

// UI state
const showColumnsDropdown = ref(false)
const showSettings = ref(false)
const generalConfig = ref<GeneralConfig>({ ...DEFAULT_GENERAL_CONFIG })
const ipConfig = ref<IpConfig>({ ...DEFAULT_IP_CONFIG })
const showImportDialog = ref(false)
const importLoading = ref(false)
const resultFolderPath = ref('')
const activeRunOutputPath = ref('') // path thực tế đang được dùng trong run hiện tại (có subfolder timestamp)

// === REGISTER LIVE TABLE ===
interface RegThread {
  index: number
  phone: string
  proxy: string       // real IP (displayProxy)
  proxyServer: string // proxy server string (ip:port:user:pass)
  userAgent: string
  uid: string
  password: string
  cookie: string
  token: string
  activity: string
  status: 'running' | 'success' | 'failed'
  verifyStatus: 'live' | 'die' | ''
  rawVerifyStatus: string    // status thô từ runner ("live","die","unknown","error","")
  verifyMessage: string      // message từ verify để detect add-mail-fail
  verifyEmail: string        // mail đã add khi verify (normal mode) → hiện ở cột EMAIL, KHÔNG bị status ghi đè
  inAddMailRetry: boolean    // đang ở vòng retry add-mail (sau outer attempt 1) → tô cam đậm
  finishedAt?: number // timestamp ms khi status chuyển sang success/failed — dùng cho auto-cleanup
}

interface RegisterLog {
  id: number
  time: string
  index: number
  phone: string
  msg: string
  type: 'info' | 'success' | 'error' | 'step'
}

let logIdCounter = 0
const registerLogs = ref<RegisterLog[]>([])
const registerThreads = ref<Map<number, RegThread>>(new Map())
const showRegisterPanel = ref(true)
const regViewMode = ref<'table' | 'log'>('table')
const isRegisterRunning = ref(false)
const isStopping = ref(false)
// Guard cho onActivated: lưu trạng thái running khi rời tab để tránh fetchAccounts race.
let _deactivatedWhileRunning = false
const logBodyRef = ref<HTMLElement | null>(null)
const isRunSortingLocked = computed(() => isVerifyRunning.value || isRegisterRunning.value)

// Bộ đếm tích lũy — chỉ tăng, không bao giờ giảm
// Dùng thay cho computed từ registerThreads để số không bị dao động khi slot được tái sử dụng
const regTotalProcessed = ref(0)
const regTotalSuccess = ref(0)
const regTotalFail = ref(0)
const regTotalLive = ref(0)
const regTotalDie = ref(0)
const regTotalUnknown = ref(0)
const regTotalCheckpoint = ref(0)
const checkpointLimitEnabled = ref(false)
const checkpointLimitCount = ref(0)

// Verify accumulated counters — tích lũy qua tất cả slots (không bị reset khi slot tái sử dụng)
const verifyTotalLive = ref(0)
const verifyTotalDie = ref(0)
const verifyTotalUnknown = ref(0)
const verifyTotalProcessed = ref(0)

// Tỉ lệ thành công hiển thị trong tag
const regSuccessRate = computed(() =>
  regTotalProcessed.value > 0 ? Math.round(regTotalSuccess.value / regTotalProcessed.value * 100) : 0
)
// Verified totals KẾT HỢP cả 2 nguồn:
//   - Normal mode: verify INLINE → regTotalLive/Die
//   - TRUE SPLIT mode: VER pool → verifyTotalLive/Die (qua verify:account-done)
// Cộng lại (1 trong 2 luôn = 0 tùy mode) → status bar VERIFIED hiện đúng ở cả 2 mode.
const verifiedLiveTotal = computed(() => regTotalLive.value + verifyTotalLive.value)
const verifiedDieTotal = computed(() => regTotalDie.value + verifyTotalDie.value)
const verifiedUnknownTotal = computed(() => regTotalUnknown.value + verifyTotalUnknown.value)
const regVerifyLiveRate = computed(() => {
  const total = verifiedLiveTotal.value + verifiedDieTotal.value
  return total > 0 ? Math.round(verifiedLiveTotal.value / total * 100) : 0
})

// ── Persist counter qua localStorage để survive UI reload (auto 6h hoặc F5) ──
// Giữ nguyên số liệu đã đếm thay vì reset về 0 sau reload.
const RUN_STATS_KEY = 'havu:runStats'

function saveRunStats() {
  try {
    localStorage.setItem(RUN_STATS_KEY, JSON.stringify({
      regProcessed: regTotalProcessed.value,
      regSuccess: regTotalSuccess.value,
      regFail: regTotalFail.value,
      regLive: regTotalLive.value,
      regDie: regTotalDie.value,
      regUnknown: regTotalUnknown.value,
      regCheckpoint: regTotalCheckpoint.value,
      verLive: verifyTotalLive.value,
      verDie: verifyTotalDie.value,
      verUnknown: verifyTotalUnknown.value,
      verProcessed: verifyTotalProcessed.value,
    }))
  } catch { /* ignore */ }
}

function clearRunStats() {
  try { localStorage.removeItem(RUN_STATS_KEY) } catch { /* ignore */ }
}

function restoreRunStats(): boolean {
  try {
    const raw = localStorage.getItem(RUN_STATS_KEY)
    if (!raw) return false
    const s = JSON.parse(raw)
    regTotalProcessed.value = s.regProcessed || 0
    regTotalSuccess.value = s.regSuccess || 0
    regTotalFail.value = s.regFail || 0
    regTotalLive.value = s.regLive || 0
    regTotalDie.value = s.regDie || 0
    regTotalUnknown.value = s.regUnknown || 0
    regTotalCheckpoint.value = s.regCheckpoint || 0
    verifyTotalLive.value = s.verLive || 0
    verifyTotalDie.value = s.verDie || 0
    verifyTotalUnknown.value = s.verUnknown || 0
    verifyTotalProcessed.value = s.verProcessed || 0
    return true
  } catch { return false }
}

// Save debounced (mỗi 1s tối đa) khi bất kỳ counter thay đổi.
let _statsSaveTimer = 0
function scheduleStatsSave() {
  if (_statsSaveTimer) return
  _statsSaveTimer = setTimeout(() => {
    _statsSaveTimer = 0
    saveRunStats()
  }, 1000) as unknown as number
}
watch([
  regTotalProcessed, regTotalSuccess, regTotalFail, regTotalLive, regTotalDie, regTotalUnknown, regTotalCheckpoint,
  verifyTotalLive, verifyTotalDie, verifyTotalUnknown, verifyTotalProcessed,
], scheduleStatsSave)
// Map<accountId, lastStatus> — track status cuối mỗi account để fix double-count khi retry:
// account unknown → retry live → giảm unknown, tăng live (KHÔNG cộng Processed 2 lần).
const _verifyLastStatus = new Map<number, string>()
// Map<UID, lastStatus> — dedupe theo UID thay vì slotId.
// Split mode: Unknown.txt được retry → account same UID chạy lại trên slot khác → slotId khác
// nhưng bản chất vẫn là 1 account → Tổng chỉ đếm 1 lần, Live/Die/Unknown cộng trừ theo status mới.
const _verifyLastStatusByUid = new Map<string, string>()
// Status đã "chốt" — batch-status message tới sau sẽ không ghi đè cột HOẠT ĐỘNG.
// 'new' / 'waiting' / '' vẫn cho phép cập nhật realtime.
const FINAL_STATUSES = new Set(['live', 'die', 'unknown', 'checkpoint'])

function preferUserAccessToken(current = '', incoming = '') {
  const cur = current.trim()
  const next = incoming.trim()
  // Ưu tiên token MỚI hợp lệ (EAAAAU Android HOẶC EAAAAAY iOS) — token iOS không bị loại.
  if (next.startsWith('EAAAAU') || next.startsWith('EAAAAAY')) return next
  if (cur.startsWith('EAAAAU') || cur.startsWith('EAAAAAY')) return cur
  return next || cur
}

// Track ID account đã verify-done trong run hiện tại — dùng để auto-clear khỏi grid khi verify:complete.
// File mode: chỉ xóa các dòng user đã tick (un-ticked rows giữ lại).
// Folder/CloneHV mode: xóa hết slot rows (run done = không còn recycling).
const verifiedRunIds = new Set<number>()

// runId tăng mỗi lần start verify mới — closure capture giá trị này, setTimeout chỉ thực thi
// nếu currentRunId === capturedRunId. Tránh setTimeout cũ xóa nhầm dòng của run mới khi user Stop → Run nhanh.
let currentRunId = 0
// Track tổng số dòng đã clear trong run hiện tại (để toast cuối hiển thị đúng số).
let realtimeClearedCount = 0

// Elapsed time
const regStartTime = ref<number>(0)
const regElapsed = ref('')
let _elapsedTimer = 0
let _stopSafetyTimer = 0
let _cleanupTimer = 0

// Task 6: lưu unsub fns từ bus.on() để cleanup chính xác per-listener.
// Trước đây gọi bus.off(eventName) → xóa MỌI listener cho event đó (kể cả của
// component/store khác cùng nghe). Giữ unsub fns → xóa đúng listener của AccountsPage.
let _verifyUnsubs: Array<() => void> = []
let _registerUnsubs: Array<() => void> = []
let _alwaysOnUnsubs: Array<() => void> = []
// Throttle toast "hết mail" theo provider — tránh spam khi nhiều account cùng fail.
const _poolExhaustedAt = new Map<string, number>()
function clearUnsubs(unsubs: Array<() => void>) {
  for (const fn of unsubs) {
    try { fn() } catch { /* ignore */ }
  }
  unsubs.length = 0
}

// Auto-cleanup timer: mỗi 1 phút. Khi reg ĐANG chạy → evict done threads > 90s tuổi
// (giảm DOM cost với reg hot 200+/min). Khi đã DỪNG → KHÔNG evict gì cả, giữ nguyên
// bảng + footer cho user xem lại kết quả (đến khi bấm "Xóa bảng" hoặc Run mới).
function startAutoCleanup() {
  clearInterval(_cleanupTimer)
  _cleanupTimer = setInterval(() => {
    // Log array cleanup luôn chạy (giữ ≤ 200 entries mới nhất) — kể cả khi đã dừng.
    if (registerLogs.value.length > 200) {
      registerLogs.value.splice(200)
    }
    // Thread eviction CHỈ khi đang chạy reg — dừng rồi thì không xóa bảng.
    if (!isRegisterRunning.value) return
    const now = Date.now()
    const ttl = 90 * 1000 // 90s — thread done quá 90s thì user đã thấy, drop khỏi grid
    let dropped = 0
    for (const [key, t] of registerThreads.value) {
      if ((t.status === 'success' || t.status === 'failed') &&
          t.finishedAt && (now - t.finishedAt > ttl)) {
        registerThreads.value.delete(key)
        dropped++
      }
    }
    if (dropped > 0) {
      // eslint-disable-next-line no-console
      console.debug(`[cleanup] dropped ${dropped} expired done threads, remaining ${registerThreads.value.size}`)
    }
  }, 60 * 1000) as unknown as number
}

function formatElapsed(ms: number): string {
  const s = Math.floor(ms / 1000)
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  return `${d}D ${h}H ${m}M ${sec}S`
}

const RUN_START_KEY = 'havu:runStartTime'

function startElapsedTimer() {
  regStartTime.value = Date.now()
  regElapsed.value = '0D 0H 0M 0S'
  try { localStorage.setItem(RUN_START_KEY, String(regStartTime.value)) } catch { /* ignore */ }
  clearInterval(_elapsedTimer)
  _elapsedTimer = setInterval(() => {
    regElapsed.value = formatElapsed(Date.now() - regStartTime.value)
  }, 1000) as unknown as number
}

function stopElapsedTimer() {
  clearInterval(_elapsedTimer)
  try { localStorage.removeItem(RUN_START_KEY) } catch { /* ignore */ }
}

// resumeElapsedTimer — gọi sau UI reload nếu run vẫn đang chạy ngầm.
// Đọc start time từ localStorage để timer tiếp tục đếm đúng từ lúc bắt đầu run.
function resumeElapsedTimer() {
  let saved = 0
  try {
    const raw = localStorage.getItem(RUN_START_KEY)
    if (raw) saved = parseInt(raw, 10) || 0
  } catch { /* ignore */ }
  if (saved <= 0) saved = Date.now()
  regStartTime.value = saved
  regElapsed.value = formatElapsed(Date.now() - regStartTime.value)
  clearInterval(_elapsedTimer)
  _elapsedTimer = setInterval(() => {
    regElapsed.value = formatElapsed(Date.now() - regStartTime.value)
  }, 1000) as unknown as number
}

// RAF batch state — component-level để tránh leak khi setupVerifyListeners tái đăng ký
const _activityBuffer = new Map<number, string>()
let _rafId = 0

const regSuccessCount = computed(() => {
  let n = 0
  for (const t of registerThreads.value.values()) { if (t.status === 'success') n++ }
  return n
})
const regDoneCount = computed(() => {
  let n = 0
  for (const t of registerThreads.value.values()) { if (t.status !== 'running') n++ }
  return n
})
const regLiveCount = computed(() => {
  let n = 0
  for (const t of registerThreads.value.values()) { if (t.verifyStatus === 'live') n++ }
  return n
})
const regDieCount = computed(() => {
  let n = 0
  for (const t of registerThreads.value.values()) { if (t.verifyStatus === 'die') n++ }
  return n
})

// Split mode active — khi true, grid dùng accountsStore (verify slots) thay vì regGridRows.
// Persist qua localStorage để sống sót khi UI auto-reload (mỗi 6h) trong lúc register run.
// Authoritative source: backend IsRegisterRunning() + config splitMode — restore từ backend ở onMounted.
const SPLIT_STATE_KEY = 'havu:splitModeActive'
const splitModeActive = ref<boolean>((() => {
  try { return localStorage.getItem(SPLIT_STATE_KEY) === '1' } catch { return false }
})())
watch(splitModeActive, (v) => {
  try {
    if (v) localStorage.setItem(SPLIT_STATE_KEY, '1')
    else localStorage.removeItem(SPLIT_STATE_KEY)
  } catch { /* ignore */ }
})

// Chuyển registerThreads → dạng { item, index }[] để dùng với DataGrid cũ
// Split mode: luôn dùng accountsStore grid (verify slots hiển thị, reg panel vẫn chạy riêng)
const isRegMode = computed(() => !splitModeActive.value && (isRegisterRunning.value || registerThreads.value.size > 0))

// Data source cho copy: reg mode dùng regGridRows, không thì dùng accountsStore
const displayAccounts = computed<Account[]>(() => {
  if (isRegMode.value) {
    return regGridRows.value.map(r => r.item as unknown as Account)
  }
  return accountsStore.accounts
})

const regGridRows = computed(() => {
  // Sort by goroutine index so visual row positions are stable even when Map insertion order
  // changes after cleanup evicts done entries and goroutines later re-insert at the end.
  const sorted = Array.from(registerThreads.value.values()).sort((a, b) => a.index - b.index)
  const rows: { item: Record<string, unknown>; index: number }[] = []
  let i = 0
  for (const t of sorted) {
    rows.push({
      index: i++,
      item: {
        id:           t.index,
        uid:          t.uid,
        password:     t.password,
        cookie:       t.cookie,
        token:        t.token,
        // Cột EMAIL: ưu tiên mail đang verify (verifyEmail) khi đã tới bước verify + add mail;
        // chưa verify thì hiện login reg (phone/email reg). verifyEmail KHÔNG bị status verify ghi đè.
        phone:        '',
        email:        t.verifyEmail || t.phone || '',
        proxy:        t.proxyServer, // proxy server (ip:port:user:pass) → cột PROXY
        runProxy:     t.proxy,       // real IP → cột IP CHẠY
        userAgent:    t.userAgent,
        activity:     t.activity,
        status:       regRowStatus(t),
      },
    })
  }
  // Cap 250 (giảm từ 600) — DataGrid render full DOM không virtual scroll cho regGrid;
  // 600 rows × 30 cột = 18K cells gây WebView2 layout cost cao khi reg hot.
  // 250 đủ hiển thị active workers (thường ≤ 200) + một số done gần đây.
  return rows.slice(0, 250)
})

// regRowStatus — map RegThread → status string để tô màu row.
// Mapping:
//   new       = xanh nhạt  → REG đang chạy
//   nvr       = vàng       → REG thành công, chưa verify (hoặc chưa chạy ver)
//   die       = đỏ         → REG thất bại
//   live      = xanh lá    → VER thành công
//   verfail   = xám        → VER xác định chết (verifyStatus = 'die')
//   addmail   = cam đậm    → đang ở pha retry add-mail / unknown do add-mail fail
//   unknown   = trong suốt → unknown sau khi đã retry hết
function isAddMailRetryMessage(message = ''): boolean {
  const msg = message.toLowerCase()
  return msg.includes('add email') ||
    msg.includes('addmail') ||
    msg.includes('add mail') ||
    msg.includes('setemail') ||
    msg.includes('create email') ||
    msg.includes('email service') ||
    msg.includes('otp timeout')
}

function regRowStatus(t: RegThread): string {
  if (t.verifyStatus === 'live') return 'live'
  if (t.verifyStatus === 'die') return 'verfail' // ver xác định chết → xám
  // Đang trong vòng retry add-mail (outer attempt > 1) → cam đậm REAL-TIME
  if (t.status === 'running' && t.inAddMailRetry) return 'addmail'
  if (t.status === 'running' && (t.uid || t.cookie || t.token)) return 'nvr'
  if (t.status === 'running') return 'new'
  if (t.status === 'success') {
    if (t.rawVerifyStatus && t.rawVerifyStatus !== 'live' && t.rawVerifyStatus !== 'die') {
      // Verify đã chạy nhưng unknown — phân biệt nguyên nhân
      if (isAddMailRetryMessage(t.verifyMessage)) {
        return 'addmail' // cam đậm — addmail fail sau khi retry hết
      }
      return 'unknown' // unknown không rõ nguyên nhân
    }
    return 'nvr' // vàng — reg xong nhưng chưa verify
  }
  return 'die' // reg fail → đỏ
}

function classifyLog(msg: string): RegisterLog['type'] {
  const m = msg.toLowerCase()
  if (m.includes('thành công') || m.includes('ok') || m.includes('http 200')) return 'success'
  if (m.includes('lỗi') || m.includes('error') || m.includes('thất bại') || m.includes('fail') || m.includes('http 4') || m.includes('http 5')) return 'error'
  if (m.startsWith('[b') || m.startsWith('[screen') || m.startsWith('[init')) return 'step'
  return 'info'
}

function pushLog(index: number, phone: string, msg: string) {
  const now = new Date()
  const time = now.toTimeString().slice(0, 8)
  registerLogs.value.unshift({ id: ++logIdCounter, time, index, phone, msg, type: classifyLog(msg) })
  // Cap log array 250 cho 24/7 — drop entries cũ nhất (cuối mảng) khi vượt.
  if (registerLogs.value.length > 250) registerLogs.value.splice(250)
  if (regViewMode.value === 'log') {
    nextTick(() => {
      if (logBodyRef.value) logBodyRef.value.scrollTop = 0
    })
  }
}

// Row class — coloring by status
function getRowClass(item: Record<string, unknown>): string {
  const acc = item as unknown as Account
  if (acc.status !== 'live' && acc.status !== 'die' && isAddMailRetryMessage(acc.activity || '')) {
    return 'data-grid__row--status-addmail'
  }
  return `data-grid__row--status-${acc.status}`
}

onMounted(async () => {
  await nextTick()
  statusBarSlotReady.value = !!document.getElementById('status-bar-page-slot')
  // Start periodic cleanup cho 24/7 operation — drop done threads > 5 phút, cap logs 300.
  startAutoCleanup()
  // Listen soft cleanup từ App.vue/status bar một lần cho vòng đời component.
  // KeepAlive giữ component sống khi đổi tab, nên listener vẫn trim buffer dù page đang inactive.
  window.addEventListener('app:soft-cleanup', handleSoftCleanup)
  await accountsStore.fetchAccounts()
  // Load settings đã lưu từ Go backend
  try {
    const svc = await getSettingsService()
    const saved = await svc.load()
    if (saved?.general) generalConfig.value = saved.general
    if (saved?.ip) ipConfig.value = saved.ip
  } catch { /* ignore nếu chưa có */ }
  // Result folder: LUÔN dùng default ./result/ cạnh exe (port C# hardcode path).
  // Ignore value save trong interaction.json — backend cũng force dùng default.
  // User không cần chọn thư mục — app tự quản lý như C# VerifyCloneVIP.
  try {
    const getDefault = (window as any)?.go?.main?.App?.GetDefaultResultPath
    if (typeof getDefault === 'function') {
      resultFolderPath.value = await getDefault() ?? ''
    }
  } catch { /* ignore */ }
  // CloneHV mode từ accountSource (đã load ở trên)
  cloneHvMode.value = generalConfig.value.accountSource === 'api'
  fileMode.value = generalConfig.value.accountSource !== 'api'

  // Always-on listener: accounts:folder-updated emit khi user load file/folder
  // mà không cần Verify chạy. Đảm bảo grid auto-refresh ngay sau khi pick file
  // ở Cài đặt chung (không cần switch tab).
  // Task 6: lưu unsub fn để cleanup chính xác trong onUnmounted; KHÔNG bus.off
  // global vì verifyListeners có thể đăng ký lại 'accounts:folder-updated'.
  try {
    const bus = await getEventBusService()
    clearUnsubs(_alwaysOnUnsubs)
    _alwaysOnUnsubs.push(bus.on('accounts:folder-updated', async (data: { imported?: number; source?: string }) => {
      accountsStore.fetchAccounts()
      // Khi source='file' → switch sang verify view + clear reg data cũ.
      if (data?.source === 'file') {
        // Clear bảng REG cũ để grid switch sang VERIFY view (isRegMode = false khi Map rỗng).
        registerThreads.value = new Map()
        registerLogs.value = []
        regTotalSuccess.value = 0
        regTotalFail.value = 0
        regTotalProcessed.value = 0
        regTotalLive.value = 0
        regTotalDie.value = 0
        regTotalUnknown.value = 0
        regTotalCheckpoint.value = 0
        // Clear split mode để full grid hiện accounts file
        splitModeActive.value = false
        // Re-sync settings từ disk (backend vừa SaveSettings).
        try {
          const svc = await getSettingsService()
          const saved = await svc.load()
          if (saved?.general) {
            generalConfig.value = saved.general
            cloneHvMode.value = saved.general.accountSource === 'api'
            fileMode.value = saved.general.accountSource !== 'api'
          }
        } catch { /* ignore */ }
        appStore.notify('info', `Đã clear bảng cũ + load ${data.imported || 0} accounts từ file để verify`)
      }
    }))
    // email:pool-exhausted — pool mail HẾT HÀNG hoặc HẾT TIỀN → toast cảnh báo nổi bật.
    // Đặt ở _alwaysOnUnsubs để fire cho MỌI chế độ chạy (normal reg+ver inline, split, ver riêng).
    // Throttle 30s/provider để tránh spam toast khi nhiều account cùng fail.
    _alwaysOnUnsubs.push(bus.on('email:pool-exhausted', (data: { provider: string; error: string }) => {
      const now = Date.now()
      const last = _poolExhaustedAt.get(data.provider) || 0
      if (now - last < 30000) return // đã báo trong 30s → bỏ qua
      _poolExhaustedAt.set(data.provider, now)
      const reason = /balance|số dư|insufficient/i.test(data.error)
        ? 'HẾT TIỀN — nạp thêm điểm'
        : 'HẾT HÀNG — chờ có hàng / đổi loại mail'
      appStore.notify('error', `⚠️ Mail [${data.provider}]: ${reason}`)
    }))
    // verify:email — mail đã add khi verify. Đặt ở _alwaysOnUnsubs để fire cho MỌI chế độ:
    //  - split / ver riêng → cập nhật cột EMAIL của VER panel (accountsStore)
    //  - normal reg+ver inline → cập nhật verifyEmail của REG panel row (registerThreads)
    _alwaysOnUnsubs.push(bus.on('verify:email', (data: { accountId: number; email: string }) => {
      if (!data.email) return
      const acc = accountsStore.accountsIndex.get(data.accountId)
      if (acc) acc.email = data.email
      const t = registerThreads.value.get(data.accountId)
      if (t) t.verifyEmail = data.email
    }))
  } catch { /* ignore */ }

  // Nếu đang verify thì re-register Wails events (component vừa remount)
  if (accountsStore.isVerifyRunning) {
    await setupVerifyListeners()
  }

  // Restore running state authoritatively từ backend sau UI reload (auto 6h hoặc user F5).
  // localStorage flag dùng cho instant restore; backend là single source of truth.
  try {
    const w = window as any
    // Prefer GetRunStatus() bundle để giảm IPC + đồng bộ stopping flag
    let isRegRunning = false
    let isVerRunning = false
    let isRegStopping = false
    let isVerStopping = false
    if (typeof w?.go?.main?.App?.GetRunStatus === 'function') {
      const s = await w.go.main.App.GetRunStatus()
      isRegRunning = !!s?.registerRunning
      isVerRunning = !!s?.verifyRunning
      isRegStopping = !!s?.registerStopping
      isVerStopping = !!s?.verifyStopping
    } else {
      // Fallback (Wails binding chưa regen)
      if (typeof w?.go?.main?.App?.IsRegisterRunning === 'function') {
        isRegRunning = Boolean(await w.go.main.App.IsRegisterRunning())
      }
      if (typeof w?.go?.main?.App?.IsVerifyRunning === 'function') {
        isVerRunning = Boolean(await w.go.main.App.IsVerifyRunning())
      }
    }
    // Reflect stopping state vào UI ngay (toolbar disable Start, hiện "Đang dừng...")
    isStopping.value = isRegStopping || isVerStopping

    if (isRegRunning || isVerRunning) {
      // Sync refs để toolbar hiện nút "Dừng", status bar hiện "Đang chạy"
      if (isRegRunning) {
        isRegisterRunning.value = true
        await setupRegisterListeners()
      }
      if (isVerRunning && !accountsStore.isVerifyRunning) {
        isVerifyRunning.value = true
        await setupVerifyListeners()
      }

      // Restore counters đã đếm trước reload (tránh reset về 0)
      restoreRunStats()

      // Resume timer từ thời điểm run bắt đầu (đã lưu localStorage)
      resumeElapsedTimer()

      // Restore split layout nếu register + cfg.splitMode + verifyEnabled
      if (isRegRunning) {
        const runner = await getVerifyRunnerService()
        const cfg = await runner.loadInteractionConfig() as any
        const shouldSplit = cfg?.splitMode === true && cfg?.verifyEnabled === true
        if (shouldSplit) {
          splitModeActive.value = true
          if (!isVerRunning) {
            await setupVerifyListeners()
          }
        } else {
          splitModeActive.value = false
        }
      }
    } else {
      // Không có run nào active → clear tất cả flag stale
      isRegisterRunning.value = false
      splitModeActive.value = false
      stopElapsedTimer()
    }
  } catch { /* ignore — giữ nguyên giá trị từ localStorage */ }
})

// KeepAlive: thêm/xóa window listeners khi page activate/deactivate
onActivated(async () => {
  window.addEventListener('keydown', handleKeyboard)
  window.addEventListener('mouseup', handleGlobalMouseUp)
  // Lấy guard trước khi await — tránh race condition với events đang chạy.
  const wasRunning = _deactivatedWhileRunning
  _deactivatedWhileRunning = false
  // Reload settings từ backend để đồng bộ với GeneralSettingsPage
  try {
    const svc = await getSettingsService()
    const saved = await svc.load()
    if (saved?.general) {
      generalConfig.value = saved.general
      cloneHvMode.value = saved.general.accountSource === 'api'
      fileMode.value = saved.general.accountSource !== 'api'
    }
    if (saved?.ip) ipConfig.value = saved.ip
  } catch { /* ignore */ }
  // Refetch accounts để đảm bảo data mới nhất (activity, status có thể thay đổi khi ở tab khác).
  // SKIP khi đang chạy reg/verify — in-memory data từ events realtime mới hơn backend.
  // SKIP thêm khi wasRunning — đề phòng race condition: event có thể đã set flag=false
  // trong khoảng thời gian await svc.load() ở trên, khiến fetchAccounts bị gọi nhầm.
  if (!isRegisterRunning.value && !isVerifyRunning.value && !wasRunning) {
    await accountsStore.fetchAccounts()
  }
  // Fix virtual scroll sau khi tab được reactivate:
  // nextTick + double rAF: frame đầu layout, frame hai đọc height sau khi split pane settle.
  nextTick(() => {
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        function syncViewport() {
          let h = 0
          if (splitModeActive.value) {
            const verBody = document.querySelector(
              '.accounts-page__content--split .split-pane:last-child .data-grid__body'
            ) as HTMLElement | null
            h = verBody?.clientHeight ?? 0
          }
          if (h <= 0 && gridContentRef.value) {
            h = gridContentRef.value.clientHeight
          }
          grid.setViewportHeight(h > 0 ? h : 600)

          const scrollEl = document.querySelector(
            splitModeActive.value
              ? '.accounts-page__content--split .split-pane:last-child .data-grid__body'
              : '.data-grid__body'
          ) as HTMLElement | null
          if (scrollEl) {
            if (grid.scrollTop.value > grid.totalHeight.value) {
              grid.scrollTop.value = 0
              scrollEl.scrollTop = 0
            } else {
              scrollEl.scrollTop = grid.scrollTop.value
            }
          } else if (grid.scrollTop.value > grid.totalHeight.value) {
            grid.scrollTop.value = 0
          }
        }

        syncViewport()
        // Retry sau 100ms: split pane resize observer có thể chưa chạy xong ở frame 2
        if (splitModeActive.value) {
          setTimeout(syncViewport, 100)
        }
      })
    })
  })
})

onDeactivated(() => {
  // Lưu ngay trước mọi async — dùng để guard fetchAccounts trong onActivated.
  _deactivatedWhileRunning = isRegisterRunning.value || isVerifyRunning.value || splitModeActive.value
  window.removeEventListener('keydown', handleKeyboard)
  window.removeEventListener('mouseup', handleGlobalMouseUp)
  // Đóng tất cả overlay/modal để tránh Teleport backdrop chặn UI ở trang khác
  ctxMenu.hide()
  showColumnsDropdown.value = false
  showSettings.value = false
  showImportDialog.value = false
  isDragging.value = false
  dragStartId.value = null
})

function handleSoftCleanup() {
  // Drop register log buffer (giữ ≤ 50 entries gần nhất để user còn nhìn được context)
  if (registerLogs.value.length > 50) {
    registerLogs.value.splice(50)
  }
}

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyboard)
  window.removeEventListener('mouseup', handleGlobalMouseUp)
  window.removeEventListener('app:soft-cleanup', handleSoftCleanup)
  // Dọn listeners, timers, RAF để tránh leak khi component bị destroy
  clearInterval(_elapsedTimer)
  clearInterval(_cleanupTimer)
  if (_rafId) { cancelAnimationFrame(_rafId); _rafId = 0 }
  if (_stopSafetyTimer) { clearTimeout(_stopSafetyTimer); _stopSafetyTimer = 0 }
  if (_statsSaveTimer) { clearTimeout(_statsSaveTimer); _statsSaveTimer = 0 }
  _activityBuffer.clear()
  // Task 6: cleanup chính xác từng listener qua unsub fn — KHÔNG bus.off global
  // (gây xóa nhầm listener của uploadLog.store hay App.vue cùng nghe).
  clearUnsubs(_verifyUnsubs)
  clearUnsubs(_registerUnsubs)
  clearUnsubs(_alwaysOnUnsubs)
})

// =====================================================
// CELL SELECTION — click/kéo chọn từng ô, Ctrl+C copy (Excel-style)
// Mỗi grid (reg/ver/normal) có selection ĐỘC LẬP — không lan sang nhau.
// =====================================================
type GridKey = 'reg' | 'ver' | 'normal'
const selectedCells = ref(new Set<string>()) // "rowId:colKey"
const cellAnchor = ref<{ rowId: number; colKey: string } | null>(null)
const cellDragging = ref(false)
const activeGridKey = ref<GridKey | ''>('') // grid nào đang có cell select

// Computed cho từng grid — chỉ trả selection nếu grid đang active, else empty.
// → Khi user click sang grid khác, grid cũ sẽ hiện empty.
const regSelectedCells = computed(() => activeGridKey.value === 'reg' ? selectedCells.value : new Set<string>())
const verSelectedCells = computed(() => activeGridKey.value === 'ver' ? selectedCells.value : new Set<string>())
const normalSelectedCells = computed(() => activeGridKey.value === 'normal' ? selectedCells.value : new Set<string>())

// Lấy rows tương ứng với gridKey để tính range select.
// regGridRows wrap trong {item, index} → unwrap về item. Các grid khác đã flat.
function getRowsForGrid(gridKey: GridKey): Account[] {
  if (gridKey === 'reg') return regGridRows.value.map(r => r.item as unknown as Account)
  return grid.sortedItems.value as unknown as Account[] // ver + normal đều dùng grid chính
}

function computeCellRange(gridKey: GridKey, rowId: number, colKey: string): Set<string> | null {
  if (!cellAnchor.value) return null
  const cols = visibleColumns.value.map(c => c.key)
  const rowIds = getRowsForGrid(gridKey).map(r => r.id)
  const aRow = rowIds.indexOf(cellAnchor.value.rowId)
  const aCol = cols.indexOf(cellAnchor.value.colKey)
  const cRow = rowIds.indexOf(rowId)
  const cCol = cols.indexOf(colKey)
  if (aRow < 0 || cRow < 0 || aCol < 0 || cCol < 0) return null
  const rFrom = Math.min(aRow, cRow), rTo = Math.max(aRow, cRow)
  const cFrom = Math.min(aCol, cCol), cTo = Math.max(aCol, cCol)
  const ns = new Set<string>()
  for (let r = rFrom; r <= rTo; r++)
    for (let c = cFrom; c <= cTo; c++)
      ns.add(`${rowIds[r]}:${cols[c]}`)
  return ns
}

// Khi user thao tác grid khác → reset selection cũ sang grid mới.
function switchGridIfNeeded(gridKey: GridKey) {
  if (activeGridKey.value !== gridKey) {
    selectedCells.value = new Set()
    cellAnchor.value = null
    activeGridKey.value = gridKey
  }
}

function handleCellClick(gridKey: GridKey, rowId: number, colKey: string, event: MouseEvent) {
  if (cellDragging.value) return
  switchGridIfNeeded(gridKey)
  const key = `${rowId}:${colKey}`
  if (event.ctrlKey || event.metaKey) {
    const ns = new Set(selectedCells.value)
    if (ns.has(key)) ns.delete(key)
    else ns.add(key)
    selectedCells.value = ns
    cellAnchor.value = { rowId, colKey }
  } else if (event.shiftKey && cellAnchor.value) {
    const range = computeCellRange(gridKey, rowId, colKey)
    if (range) selectedCells.value = range
  } else {
    selectedCells.value = new Set([key])
    cellAnchor.value = { rowId, colKey }
  }
  selection.highlightClear()
}

function handleCellMouseDown(gridKey: GridKey, rowId: number, colKey: string, event: MouseEvent) {
  if ((event.target as HTMLElement).tagName === 'INPUT') return
  switchGridIfNeeded(gridKey)
  cellDragging.value = true
  const key = `${rowId}:${colKey}`
  if (event.ctrlKey || event.metaKey) {
    const ns = new Set(selectedCells.value)
    ns.add(key)
    selectedCells.value = ns
    cellAnchor.value = { rowId, colKey }
  } else if (event.shiftKey && cellAnchor.value) {
    const range = computeCellRange(gridKey, rowId, colKey)
    if (range) selectedCells.value = range
  } else {
    selectedCells.value = new Set([key])
    cellAnchor.value = { rowId, colKey }
  }
  selection.highlightClear()
}

function handleCellMouseEnter(gridKey: GridKey, rowId: number, colKey: string) {
  if (!cellDragging.value || !cellAnchor.value) return
  if (activeGridKey.value !== gridKey) return // drag chỉ trong grid bắt đầu
  const range = computeCellRange(gridKey, rowId, colKey)
  if (range) selectedCells.value = range
}

async function copySelectedCells() {
  if (selectedCells.value.size === 0 || !activeGridKey.value) return false
  const rows = getRowsForGrid(activeGridKey.value as GridKey)
  const cols = visibleColumns.value.map(c => c.key)
  const lines: string[] = []
  for (const row of rows) {
    const rowVals: string[] = []
    let hasAny = false
    for (const col of cols) {
      if (selectedCells.value.has(`${row.id}:${col}`)) {
        rowVals.push(String((row as any)[col] ?? ''))
        hasAny = true
      } else if (hasAny) {
        // Tab giữ cho cột giữa bị skip → align nếu chọn non-contiguous
        rowVals.push('')
      }
    }
    if (hasAny) {
      // Trim trailing empty strings
      while (rowVals.length > 0 && rowVals[rowVals.length - 1] === '') rowVals.pop()
      lines.push(rowVals.join('\t'))
    }
  }
  const text = lines.join('\n')
  if (text) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      return false
    }
  }
  return false
}

// =====================================================
// DRAG-SELECT (bôi đen) — click + kéo chuột highlight
// =====================================================
function handleRowMouseDown(id: number, event: MouseEvent) {
  if ((event.target as HTMLElement).tagName === 'INPUT') return
  // Nếu click vào cell (data cell) → handleCellClick đã stopPropagation, không vào đây.
  // Còn lại (checkbox/STT) → bắt đầu drag row-highlight.
  dragStartId.value = id
  isDragging.value = false
  if (!event.ctrlKey && !event.shiftKey && !event.metaKey) {
    selection.setHighlighted(new Set([id]))
  }
}

function handleRowMouseEnter(id: number) {
  if (dragStartId.value === null) return
  isDragging.value = true
  const accounts = accountsStore.accounts
  const ids = accounts.map(a => a.id)
  const startIdx = ids.indexOf(dragStartId.value)
  const endIdx = ids.indexOf(id)
  if (startIdx === -1 || endIdx === -1) return
  const from = Math.min(startIdx, endIdx)
  const to = Math.max(startIdx, endIdx)
  const rangeIds = new Set<number>()
  for (let i = from; i <= to; i++) rangeIds.add(ids[i])
  selection.setHighlighted(rangeIds)
}

function handleGlobalMouseUp() {
  isDragging.value = false
  dragStartId.value = null
  // Kết thúc drag cell — không clear selection, giữ kết quả user đã chọn
  cellDragging.value = false
}

// =====================================================
// KEYBOARD
// =====================================================
function handleKeyboard(e: KeyboardEvent) {
  const tag = (e.target as HTMLElement).tagName
  if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return

  // Ctrl+C: nếu có cell đang select → copy TSV các cell đó, else fallback row highlight
  if (e.ctrlKey && (e.key === 'c' || e.key === 'C') && selectedCells.value.size > 0) {
    e.preventDefault()
    copySelectedCells()
    return
  }
  // Ctrl+A: highlight tất cả
  if (e.ctrlKey && e.key === 'a') {
    e.preventDefault()
    selection.highlightAll(accountsStore.accounts)
  }
  // Space: toggle check cho MỌI row đang bôi — bao gồm:
  //   1) Row highlight cũ (drag STT)
  //   2) Row chứa cell selection mới
  // Row đã tích → bỏ tích, row chưa tích → tích.
  if (e.key === ' ') {
    e.preventDefault()
    const rowIdsFromCells = new Set<number>()
    for (const key of selectedCells.value) {
      const [rowIdStr] = key.split(':')
      const rid = Number(rowIdStr)
      if (!Number.isNaN(rid)) rowIdsFromCells.add(rid)
    }
    const combined = new Set<number>([
      ...selection.highlightedIds.value,
      ...rowIdsFromCells,
    ])
    if (combined.size === 0) return
    const next = new Set(selection.checkedIds.value)
    for (const id of combined) {
      if (next.has(id)) next.delete(id)
      else next.add(id)
    }
    selection.checkedIds.value = next
  }
  // Delete: xóa checked accounts
  if (e.key === 'Delete' && selection.checkedCount.value > 0) {
    e.preventDefault()
    handleDelete()
  }
  // Escape
  if (e.key === 'Escape') {
    if (ctxMenu.visible.value) ctxMenu.hide()
    else if (accountsStore.showDetailPanel) accountsStore.closeDetail()
    else if (selectedCells.value.size > 0) {
      selectedCells.value = new Set()
      cellAnchor.value = null
    }
    else selection.highlightClear()
  }
  if (e.ctrlKey && e.key === 'i') {
    e.preventDefault()
    showImportDialog.value = true
  }
  if (e.ctrlKey && e.key === 'f') {
    e.preventDefault()
    const el = document.querySelector('.toolbar__filter') as HTMLInputElement
    el?.focus()
  }
}

// =====================================================
// ROW CLICK — highlight row (bôi đen), NOT check
// =====================================================
function handleRowClick(id: number, event: MouseEvent) {
  // Dùng sortedItems — thứ tự hiển thị thực tế, đúng khi sort/filter
  selection.handleRowClick(id, event, grid.sortedItems.value as unknown as Account[])
}

// Double-click: toggle check (giống WeBM dgvAccount_CellDoubleClick)
function handleRowDblClick(id: number) {
  selection.toggleCheck(id)
}

function clearGridSort() {
  grid.sort.value = { column: '', direction: null }
}

watch(isRunSortingLocked, (locked) => {
  if (locked) clearGridSort()
})

function handleSort(column: string) {
  if (isRunSortingLocked.value) return
  grid.toggleSort(column)
}

// =====================================================
// CHECKBOX — toggle check (cChose)
// =====================================================
function handleToggleAll() {
  // Dùng sortedItems — chỉ toggle accounts đang visible (sau filter/sort)
  selection.toggleCheckAll(grid.sortedItems.value as unknown as Account[])
}

function handleToggleRow(id: number) {
  selection.toggleCheck(id)
}

// =====================================================
// CONTEXT MENU — dùng highlighted rows cho copy
// =====================================================
function handleContextMenu(id: number, event: MouseEvent) {
  // Nếu row chưa highlighted, highlight nó (trừ khi user đang có cell selection — giữ nguyên)
  if (selectedCells.value.size === 0 && !selection.highlightedIds.value.has(id)) {
    selection.highlightOne(id)
  }

  const hasCellSelection = selectedCells.value.size > 0

  // rowIdsFromCells: set các row IDs có ít nhất 1 cell đang select → dùng cho
  // "Chọn bôi đen" + Copy menu khi user dùng cell selection thay vì row highlight.
  const rowIdsFromCells = new Set<number>()
  for (const key of selectedCells.value) {
    const [rowIdStr] = key.split(':')
    const rid = Number(rowIdStr)
    if (!Number.isNaN(rid)) rowIdsFromCells.add(rid)
  }
  // Combined: union giữa row-highlight cũ và rows chứa cell-select mới
  const combinedHighlight = new Set<number>([
    ...selection.highlightedIds.value,
    ...rowIdsFromCells,
  ])

  const menuItems = buildAccountContextMenu({
    // "Chọn tất cả" = check all visible (theo sort/filter hiện tại)
    selectAll: () => selection.checkAll(grid.sortedItems.value as unknown as Account[]),
    // "Chọn bôi đen" = tick checkbox cho MỌI row đang bôi (row highlight HOẶC có cell select)
    selectHighlight: () => {
      if (combinedHighlight.size === 0) return
      const next = new Set(selection.checkedIds.value)
      for (const id of combinedHighlight) next.add(id)
      selection.checkedIds.value = next
    },
    // "Theo trạng thái" = check by status (trong danh sách đang hiển thị)
    selectByStatus: (status) => selection.checkByFilter(grid.sortedItems.value as unknown as Account[], a => (a as unknown as Account).status === status),
    // "Bỏ chọn tất cả" = uncheck all + clear row highlight + clear cell selection
    deselectAll: () => {
      selection.uncheckAll()
      selection.highlightClear()
      selectedCells.value = new Set()
      cellAnchor.value = null
    },
    // Copy theo field — dùng combined (cells rows + highlighted rows)
    copyField: (field) => clipboard.copyField(displayAccounts.value, combinedHighlight, field as keyof Account),
    copyFullImport: () => clipboard.copyFullImport(displayAccounts.value, combinedHighlight),
    // Delete dùng CHECKED rows
    deleteAccounts: handleDelete,
    // Copy các cell đã bôi (Excel-style TSV)
    copyCells: hasCellSelection ? copySelectedCells : undefined,
    cellsCount: selectedCells.value.size,
  })
  ctxMenu.show(event, id, menuItems)
}

// =====================================================
// ACTIONS
// =====================================================
async function handleOpenConfigFolder() {
  const fn = (window as any)?.go?.main?.App?.OpenConfigFolder
  if (!fn) return
  const err = await fn()
  if (err) appStore.notify('error', err)
}

async function handleOpenResultFolder() {
  const svc = await getFileDialogService()

  // Đang chạy (reg HOẶC verify) → mở thư mục run hiện tại (có subfolder timestamp).
  // activeRunOutputPath được set qua register:output-path hoặc verify:output-path.
  if ((isRegisterRunning.value || isVerifyRunning.value) && activeRunOutputPath.value) {
    const err = await svc.openFolderInExplorer(activeRunOutputPath.value)
    if (err) appStore.notify('error', err)
    return
  }

  // Không chạy nhưng vừa có run xong → mở thư mục run cuối
  if (activeRunOutputPath.value) {
    const err = await svc.openFolderInExplorer(activeRunOutputPath.value)
    if (err) appStore.notify('error', err)
    return
  }

  // Có resultFolderPath từ config → mở thư mục gốc (chưa có run nào trong session này)
  if (resultFolderPath.value) {
    const err = await svc.openFolderInExplorer(resultFolderPath.value)
    if (err) appStore.notify('error', err)
    return
  }

  // Chưa có path gì → load từ backend (auto-fallback có thể đã set)
  try {
    const interactionSvc = await getInteractionService()
    const current = await interactionSvc.load()
    if (current.resultFolderPath) {
      resultFolderPath.value = current.resultFolderPath
      await svc.openFolderInExplorer(current.resultFolderPath)
      return
    }
  } catch { /* ignore */ }

  // Vẫn chưa có config → fallback về default path (backend auto-tạo KetQua/ hoặc Documents/KetQua/).
  // Không hiện dialog — user không cần chọn, app luôn biết mở về đâu.
  try {
    const getDefault = (window as any)?.go?.main?.App?.GetDefaultResultPath
    if (typeof getDefault === 'function') {
      const defaultPath = await getDefault()
      if (defaultPath) {
        await svc.openFolderInExplorer(defaultPath)
        return
      }
    }
  } catch { /* ignore */ }

  // Fallback cuối — mở dialog chọn thư mục (chỉ xảy ra nếu Wails binding lỗi)
  const path = await svc.openFolder()
  if (!path) return
  resultFolderPath.value = path
  try {
    const interactionSvc = await getInteractionService()
    const current = await interactionSvc.load()
    await interactionSvc.save({ ...current, resultFolderPath: path })
    appStore.notify('success', 'Đã lưu Result folder')
  } catch { /* ignore */ }
}

async function handleDelete() {
  const ids = selection.getCheckedIds()
  if (ids.length === 0) return
  await accountsStore.deleteAccounts(ids)
  selection.uncheckAll()
}

async function handleImportData(data: string) {
  importLoading.value = true
  try {
    await accountsStore.importAccounts(data)
    showImportDialog.value = false
  } finally {
    importLoading.value = false
  }
}

async function handleFolderImported() {
  await accountsStore.fetchAccounts()
}

async function handleSaveSettings(config: GeneralConfig, ip: IpConfig) {
  generalConfig.value = config
  ipConfig.value = ip
  cloneHvMode.value = config.accountSource === 'api'
  fileMode.value = config.accountSource !== 'api'
  try {
    const svc = await getSettingsService()
    await svc.save({ general: config, ip })
    // KHÔNG notify success — modal đã có label "Tự động lưu" làm indicator,
    // autosave fire mỗi lần thay đổi sẽ flood toast.
  } catch (err) {
    appStore.notify('error', 'Lỗi lưu cài đặt: ' + String(err))
  }
}

// === RUN / STOP VERIFY ===
async function setupVerifyListeners() {
  const bus = await getEventBusService()

  // Task 6: cleanup listeners cũ qua unsub fns đã lưu — KHÔNG dùng bus.off(name)
  // global vì sẽ xóa nhầm listener của uploadLog.store hay component khác cùng
  // nghe các event này.
  clearUnsubs(_verifyUnsubs)

  // verify:output-path — Go emit ngay sau khi xác định thư mục verify folder.
  // Sync activeRunOutputPath để Result button mở đúng verify run.
  _verifyUnsubs.push(bus.on('verify:output-path', (data: { path: string }) => {
    if (data.path) activeRunOutputPath.value = data.path
  }))

  // Cancel RAF cũ nếu còn pending từ lần đăng ký trước
  if (_rafId) { cancelAnimationFrame(_rafId); _rafId = 0 }
  _activityBuffer.clear()

  function flushActivityBuffer() {
    _rafId = 0
    for (const [id, msg] of _activityBuffer) {
      // ensureSlot tạo placeholder nếu slot chưa vào index (race với fetchAccounts đang chạy).
      // Đảm bảo không rơi mất message hoạt động mặc cho timing không đoán trước giữa FE fetch
      // và BE emit batch-status đầu tiên.
      const acc = accountsStore.ensureSlot(id)
      if (!FINAL_STATUSES.has(acc.status)) acc.activity = msg
    }
    _activityBuffer.clear()
  }

  // Batch event: Go gom tất cả updates vào 1 event/100ms — tránh flood WebView IPC
  _verifyUnsubs.push(bus.on('verify:batch-status', (updates: Array<{ accountId: number; message: string }>) => {
    for (const item of updates) {
      const acc = accountsStore.accountsIndex.get(item.accountId)
      // Chỉ bỏ qua khi status đã CUỐI (live/die/unknown/checkpoint) — tránh ghi đè kết quả verify:account-done.
      // Trước đây check `!== 'new'` làm rơi mất các message phát sớm khi slot còn ở status mặc định
      // "waiting" (split mode vừa khởi tạo) → cột HOẠT ĐỘNG trong pane VER trắng trơn suốt run.
      if (acc && FINAL_STATUSES.has(acc.status)) continue
      // Task 6 dedup: nếu message giống cái đã buffer cho slot này → skip để tránh
      // ghi đè reactive state với cùng giá trị (Vue vẫn invalidate watcher).
      const prev = _activityBuffer.get(item.accountId)
      if (prev === item.message) continue
      _activityBuffer.set(item.accountId, item.message)
    }
    if (!_rafId && _activityBuffer.size > 0) _rafId = requestAnimationFrame(flushActivityBuffer)
  }))
  _verifyUnsubs.push(bus.on('verify:raw-proxy', (data: { accountId: number; proxy: string }) => {
    accountsStore.setDisplayProxy(data.accountId, data.proxy)
  }))
  _verifyUnsubs.push(bus.on('verify:proxy', (data: { accountId: number; proxy: string }) => {
    accountsStore.setRunProxy(data.accountId, data.proxy)
  }))
  // (verify:email handler đã chuyển sang _alwaysOnUnsubs để fire cả normal mode — xem onMounted)
  // Account hoàn thành: O(1) lookup + update status + tích lũy counter
  _verifyUnsubs.push(bus.on('verify:account-done', async (data: { accountId: number; uid?: string; status: string; message: string; token?: string; cookie?: string }) => {
    selection.checkedIds.value.delete(data.accountId)
    selection.checkedIds.value = new Set(selection.checkedIds.value)
    const acc = accountsStore.accountsIndex.get(data.accountId)
    if (acc) {
      if (data.status) acc.status = data.status.toLowerCase() as import('@/services/contracts').AccountStatus
      if (data.message) acc.activity = data.message
      acc.token = preferUserAccessToken(acc.token, data.token || '')
      if (data.cookie) acc.cookie = data.cookie // cookie MỚI sau login verify → cập nhật cột COOKIE
    }
    // Tích lũy — ưu tiên dedupe theo UID (unique per account) thay vì slotId (bị recycle).
    // Split mode: unknown account được pop lại từ Unknown.txt chạy trên slot khác → cùng UID,
    // khác slotId. Không dedupe theo UID → Tổng cộng dồn gấp đôi (REG Live=937 vs VER Tổng=962).
    const s = (data.status || '').toLowerCase()
    const uid = (data.uid || '').trim()
    const prev = uid
      ? _verifyLastStatusByUid.get(uid)
      : _verifyLastStatus.get(data.accountId)
    if (prev) {
      if (prev === 'live') verifyTotalLive.value = Math.max(0, verifyTotalLive.value - 1)
      else if (prev === 'die') verifyTotalDie.value = Math.max(0, verifyTotalDie.value - 1)
      else verifyTotalUnknown.value = Math.max(0, verifyTotalUnknown.value - 1)
    } else {
      verifyTotalProcessed.value++
    }
    if (uid) _verifyLastStatusByUid.set(uid, s)
    _verifyLastStatus.set(data.accountId, s)
    if (s === 'live') verifyTotalLive.value++
    else if (s === 'die') verifyTotalDie.value++
    else verifyTotalUnknown.value++

    // Auto-clear LOGIC — quyết định có xóa dòng khi account verify xong không:
    //   - File-mode standalone (pick file → tick → Run verify): XÓA dòng (user muốn grid sạch dần).
    //   - Split mode (reg+ver chạy cùng): KHÔNG xóa — slots recycle, reg push acc mới vào slot cũ.
    //   - CloneHV pool / folder streaming: KHÔNG xóa — slots recycle tương tự.
    // Chỉ file-mode pure standalone mới qualify cho real-time clear.
    const isFileMode = generalConfig.value.accountSource === 'file'
    const isSlotRecyclingMode = splitModeActive.value ||
                                 generalConfig.value.accountSource === 'api' ||
                                 generalConfig.value.accountSource === 'folder'
    if (isFileMode && !isSlotRecyclingMode) {
      // Track ID → verify:complete dọn stragglers nếu setTimeout chưa fire
      verifiedRunIds.add(data.accountId)
      const idToRemove = data.accountId
      const capturedRunId = currentRunId
      setTimeout(() => {
        // Bail out nếu user đã Stop/Run mới — tránh xóa nhầm dòng của run mới có cùng ID.
        if (capturedRunId !== currentRunId) return
        verifiedRunIds.delete(idToRemove)
        accountsStore.deleteAccounts([idToRemove], true).then(() => {
          realtimeClearedCount++
        }).catch(err => {
          console.error('Real-time clear row failed:', err)
        })
      }, 1500)
    }
    // Slot recycling modes: KHÔNG add vào verifiedRunIds → verify:complete không xóa gì.
    // Slot rows giữ nguyên, backend sẽ update in-place khi reg/dispatch push acc mới vào slot.
  }))
  _verifyUnsubs.push(bus.on('verify:complete', async () => {
    isVerifyRunning.value = false
    isStopping.value = false
    stopElapsedTimer()
    // Auto-clear các dòng đã verify khỏi grid — cho phép user chuẩn bị batch tiếp theo gọn gàng.
    // Live/Die counters vẫn còn trên header; kết quả chi tiết nằm ở Live.txt/Die.txt qua nút Result.
    const idsToRemove = [...verifiedRunIds]
    verifiedRunIds.clear()
    if (idsToRemove.length > 0) {
      try {
        await accountsStore.deleteAccounts(idsToRemove, true)
      } catch (err) {
        console.error('Auto-clear verified rows failed:', err)
      }
    }
    // Tổng = real-time clear (file mode, đã xóa từng dòng) + batch clear (slot modes hoặc stragglers).
    const totalCleared = realtimeClearedCount + idsToRemove.length
    realtimeClearedCount = 0
    // Giải phóng tracking maps — không cần giữ qua run tiếp theo (handleRun cũng clear, nhưng
    // clear ở đây tránh giữ dữ liệu cũ trong bộ nhớ khi app idle nhiều giờ giữa các run).
    _verifyLastStatus.clear()
    _verifyLastStatusByUid.clear()
    appStore.notify('success', `Verify hoàn thành — đã clear ${totalCleared} dòng đã verify (xem kết quả ở Live.txt / Die.txt)`)
    accountsStore.fetchAccounts()
  }))
  // CloneHV / startup: slot init lần đầu → full refresh (chỉ vài lần, không hot path)
  _verifyUnsubs.push(bus.on('verify:accounts-updated', () => {
    accountsStore.fetchAccounts()
  }))
  // Hot path: account mới assign vào slot → update in-place, không IPC round-trip.
  // Backend emit sau mỗi slot assign trong streaming verify (thay thế verify:accounts-updated cũ).
  _verifyUnsubs.push(bus.on('verify:slot-assigned', (data: {
    slotId: number; uid: string; password: string; phone: string; status: string; userAgent?: string; token?: string; cookie?: string
  }) => {
    // Slot recycling: slot cũ giờ thuộc account mới → xoá _verifyLastStatus để lần verify:account-done
    // kế tiếp được tính là lần Processed mới (không bị "double-count" logic bóp thành 0 delta).
    // Trước fix: Tổng verify dừng ở #slots (vd 100) vì mỗi slotId chỉ incr Processed 1 lần duy nhất.
    _verifyLastStatus.delete(data.slotId)
    accountsStore.applySlotAssigned(data)
  }))
  // Folder watcher: file nguồn có account mới → refresh grid
  _verifyUnsubs.push(bus.on('accounts:folder-updated', (data: { imported: number }) => {
    accountsStore.fetchAccounts()
    appStore.notify('info', `Đã thêm ${data.imported} account mới từ thư mục nguồn`)
  }))
}

async function handleRun() {
  const ids = selection.getCheckedIds()

  // Reset split layout khi bắt đầu run mới
  splitModeActive.value = false
  clearGridSort()

  // Đọc config 1 lần để check createEnabled + splitMode
  // cloneHvMode.value dùng trực tiếp — đã được sync từ generalConfig.accountSource === 'api'
  let createEnabled = false
  let splitMode = false
  try {
    const runner = await getVerifyRunnerService()
    const cfg = await runner.loadInteractionConfig()
    createEnabled = (cfg as any).createEnabled === true
    // splitMode chỉ active khi bật cả createEnabled + verifyEnabled — tránh
    // hiển thị panel VERIFY rỗng khi user chỉ chạy register.
    splitMode = (cfg as any).splitMode === true && (cfg as any).verifyEnabled === true
  } catch { /* ignore */ }

  // createEnabled → chạy đăng ký tài khoản thay vì verify
  if (createEnabled) {
    if (splitMode) {
      // Split mode: grid hiển thị verify slots (accountsStore), register panel hiển thị reg threads
      splitModeActive.value = true
      await setupVerifyListeners()
    }
    await handleRunRegister()
    return
  }

  // Register đang chạy → không thể chạy Verify cùng lúc
  if (isRegisterRunning.value) {
    appStore.notify('warning', 'Register đang chạy — vui lòng dừng Register trước khi chạy Verify')
    return
  }

  if (!cloneHvMode.value && !fileMode.value && ids.length === 0) {
    appStore.notify('warning', 'Chưa chọn tài khoản nào để verify')
    return
  }

  // Xác nhận backend đã thật sự dừng (tránh mismatch giữa frontend state và goroutine thực tế)
  try {
    const runner = await getVerifyRunnerService()
    const stillRunning = await runner.isRunning()
    if (stillRunning) {
      appStore.notify('warning', 'Backend vẫn đang dừng, vui lòng chờ vài giây rồi thử lại')
      return
    }
  } catch { /* ignore nếu bridge chưa sẵn sàng */ }

  isVerifyRunning.value = true

  // Bump runId → setTimeout cũ từ run trước (nếu còn pending) sẽ bail out, không xóa nhầm dòng run mới.
  currentRunId++
  realtimeClearedCount = 0

  // Reset bộ track ID đã verify cho run mới (auto-clear khi verify:complete)
  verifiedRunIds.clear()

  // Reset accumulated verify counters khi bắt đầu run mới
  verifyTotalLive.value = 0
  verifyTotalDie.value = 0
  verifyTotalUnknown.value = 0
  verifyTotalProcessed.value = 0
  _verifyLastStatus.clear()
  _verifyLastStatusByUid.clear()
  // Clear path cũ — verify:output-path event sẽ set path mới ngay khi backend tạo folder.
  activeRunOutputPath.value = ''
  // CHỈ clear grid khi CloneHV/folder mode — backend sẽ tự repopulate slot rows.
  // Trong file mode (accountSource='file'), accounts đã load sẵn vào backend a.accounts —
  // clear sẽ làm grid trống và verify chạy "vô hình".
  const isSelectedFileMode = generalConfig.value.accountSource === 'file'
  if (!isSelectedFileMode) {
    accountsStore.clearAccounts()
  }
  startElapsedTimer()

  // Đăng ký listeners TRƯỚC khi gọi RunVerify — await đảm bảo sẵn sàng
  await setupVerifyListeners()

  try {
    const runner = await getVerifyRunnerService()

    // Lấy proxy từ settings
    let proxyStr = ''
    if (ipConfig.value.proxyList) {
      const lines = ipConfig.value.proxyList.split('\n').filter((l: string) => l.trim())
      if (lines.length > 0) proxyStr = lines[0].trim()
    }

    // Backend tự load interaction.json/general.json qua LoadInteractionConfig().
    // verifyConfig ở đây là placeholder — backend ignore, chỉ dùng accountIds + maxThreads + outputPath.
    const config = {
      accountIds: ids,
      maxThreads: generalConfig.value.threadRequest || 1,
      verifyConfig: {} as any,
      outputPath: '',
      proxy: proxyStr,
    }

    const result = await runner.run(config)

    // Lỗi trả về ngay (path sai, đang chạy, không có account...) → reset state
    // Chú ý: "Đang chạy verify..." là message thành công — chỉ check "vui lòng dừng"
    const isError = result.includes('không hợp lệ') || result.includes('Không có account') ||
                    result.includes('vui lòng dừng') || result.includes('Lỗi') ||
                    result.includes('Thiếu cấu hình') || result.includes('không tìm thấy')
    if (isError) {
      appStore.notify('error', result)
      isVerifyRunning.value = false
    }
    // Nếu không phải lỗi → giữ nguyên isVerifyRunning = true, chờ verify:complete
  } catch (err) {
    console.error('RunVerify error:', err)
    isVerifyRunning.value = false
    appStore.notify('error', 'Lỗi khi chạy verify: ' + String(err))
  }
}

// Auto-stop khi số checkpoint vượt ngưỡng (chỉ fire 1 lần khi đang chạy reg)
watch(regTotalCheckpoint, (count) => {
  if (!checkpointLimitEnabled.value) return
  if (!isRegisterRunning.value) return
  if (count < checkpointLimitCount.value) return
  pushLog(0, 'system', `⛔ Vượt ngưỡng checkpoint (${count}/${checkpointLimitCount.value}) — tự dừng reg`)
  appStore.notify('warning', `Vượt ngưỡng ${checkpointLimitCount.value} checkpoint — đã tự dừng reg`)
  checkpointLimitEnabled.value = false // tránh fire nhiều lần
  handleStop()
})

async function handleStop() {
  try {
    const runner = await getVerifyRunnerService()
    if (isVerifyRunning.value || isRegisterRunning.value) {
      isStopping.value = true
    }
    await runner.stop()
    await runner.stopRegister()
    pushLog(0, 'system', '⏹ Đã gửi lệnh dừng — chờ các luồng hoàn tất...')
    // Safety timeout: reset sau 60s nếu register:complete không fire
    if (_stopSafetyTimer) clearTimeout(_stopSafetyTimer)
    _stopSafetyTimer = setTimeout(() => {
      _stopSafetyTimer = 0
      if (isStopping.value) {
        isStopping.value = false
        isVerifyRunning.value = false
        isRegisterRunning.value = false
      }
    }, 60000) as unknown as number
  } catch { /* ignore */ }
}

// === REGISTER FLOW ===

async function setupRegisterListeners() {
  const bus = await getEventBusService()
  // Task 6: cleanup listeners cũ qua unsub fns đã lưu (không bus.off global).
  clearUnsubs(_registerUnsubs)

  // === RAF batching — gộp nhiều events trong 1 animation frame (~16ms) thành 1 lần Vue re-render ===
  // Quan trọng: nếu không batch, 100 goroutine fail nhanh (1-2s) → 100 events/s → JS engine bận → click không được xử lý
  type StatusEvent = { index: number; phone: string; proxy: string; proxyServer?: string; userAgent?: string; msg: string; reset?: boolean }
  type DoneEvent = { index: number; phone: string; proxy: string; proxyServer?: string; userAgent?: string; success: boolean; uid: string; cookie: string; password: string; token?: string; message: string; verifyStatus?: string; verifyMessage?: string; verifyEmail?: string; checkpoint?: boolean }
  type TokenEvent = { index: number; uid: string; token: string; cookie: string }

  let pendingStatus: StatusEvent[] = []
  let pendingDone: DoneEvent[] = []
  let pendingTokens: TokenEvent[] = []
  let rafId = 0

  function flushRegisterEvents() {
    rafId = 0
    // Flush register:status — update existing slot HOẶC tạo mới.
    // FIX: trước đây dùng set() thay thế cả entry → token/uid/password/cookie
    // (vừa fill từ register:token) bị wipe khi verify steps emit status liên tục.
    // Giờ giữ nguyên các field đã fill, chỉ ghi đè activity/userAgent/proxy.
    for (const data of pendingStatus) {
      const existing = registerThreads.value.get(data.index)
      // Slot đã done (success/failed) → account mới bắt đầu trên cùng slot ID → reset entry
      // để token/uid/cookie cũ không leak sang account mới.
      if (!data.reset && existing && (existing.status === 'success' || existing.status === 'failed')) {
        continue
      }
      if (!data.reset && existing && existing.status !== 'success' && existing.status !== 'failed') {
        existing.activity = data.msg
        if (data.userAgent) existing.userAgent = data.userAgent
        if (data.proxy) existing.proxy = data.proxy
        if (data.proxyServer) existing.proxyServer = data.proxyServer
        if (data.phone) existing.phone = data.phone
        // Detect "[Retry lần X/Y]" → row sang cam đậm ngay lập tức
        if (/\[Retry lần \d+\/\d+\]/.test(data.msg)) {
          existing.inAddMailRetry = true
        }
      } else {
        registerThreads.value.set(data.index, {
          index: data.index, phone: data.phone, proxy: data.proxy || '', proxyServer: data.proxyServer || '',
          userAgent: data.userAgent || '', uid: '', password: '', cookie: '', token: '', activity: data.msg,
          status: 'running', verifyStatus: '', rawVerifyStatus: '', verifyMessage: '', verifyEmail: '', inAddMailRetry: /\[Retry lần \d+\/\d+\]/.test(data.msg),
        })
      }
    }
    pendingStatus = []
    // Flush register:token — apply uid/token/cookie + đếm REG success ngay khi reg xong.
    // Fire ngay sau reg success (trước khi verify chạy) → counter tăng NGAY, không cần
    // đợi verify xong (register:account-done chỉ đến sau vài phút verify).
    // Phải chạy SAU pendingStatus (entry đã được tạo bởi status flush). Nếu race:
    // status fire chậm → token đến trước → tạo entry tạm để không mất uid/token.
    for (const data of pendingTokens) {
      let t = registerThreads.value.get(data.index)
      if (!t) {
        // Entry chưa có (race với status flush) → tạo placeholder để giữ uid/token
        t = {
          index: data.index, phone: '', proxy: '', proxyServer: '',
          userAgent: '', uid: '', password: '', cookie: '', token: '',
          activity: '', status: 'running' as const, verifyStatus: '', rawVerifyStatus: '', verifyMessage: '', verifyEmail: '', inAddMailRetry: false,
        }
        registerThreads.value.set(data.index, t)
      }
      if (data.uid) t.uid = data.uid
      if (data.token) t.token = data.token
      if (data.cookie) t.cookie = data.cookie
      // Đếm REG success ngay khi token nhận được — không đợi verify:
      regTotalSuccess.value++
      regTotalProcessed.value++
    }
    pendingTokens = []
    // Flush register:account-done — cập nhật slot display + đếm FAIL + verify live/die.
    // regTotalSuccess đã được đếm trong register:token handler (ở trên).
    // Với success=true: chỉ cập nhật verifyStatus/Live/Die, không double-count success.
    // Với success=false: đây là điểm duy nhất đếm fail (register:token không fire cho fail).
    for (const data of pendingDone) {
      let t = registerThreads.value.get(data.index)
      if (!t) {
        t = {
          index: data.index, phone: data.phone || '', proxy: data.proxy || '', proxyServer: data.proxyServer || '',
          userAgent: data.userAgent || '', uid: '', password: '', cookie: '', token: '',
          activity: '', status: 'running' as const, verifyStatus: '', rawVerifyStatus: '', verifyMessage: '', verifyEmail: '', inAddMailRetry: false,
        }
        registerThreads.value.set(data.index, t)
      }
      if (t) {
        t.uid = data.uid || ''
        t.password = data.password || ''
        t.cookie = data.cookie || ''
        if (data.token) t.token = data.token
        if (data.userAgent) t.userAgent = data.userAgent
        if (data.proxy) t.proxy = data.proxy
        if (data.proxyServer) t.proxyServer = data.proxyServer
        if (data.phone) t.phone = data.phone
        if (data.verifyEmail) t.phone = data.verifyEmail
        t.status = data.success ? 'success' : 'failed'
        t.verifyStatus = (data.verifyStatus === 'live' || data.verifyStatus === 'die') ? data.verifyStatus : ''
        t.rawVerifyStatus = data.verifyStatus || ''
        t.verifyMessage = data.verifyMessage || ''
        t.inAddMailRetry = false // reset flag — màu cuối do verifyStatus + verifyMessage quyết
        t.activity = data.success ? 'Thành công' : (data.message || 'Thất bại')
        t.finishedAt = Date.now() // Track timestamp để auto-cleanup sau N phút
      }
      if (data.success) {
        // regTotalSuccess đã đếm ở register:token. Chỉ cập nhật verify outcome.
        if (data.verifyStatus === 'live') regTotalLive.value++
        else if (data.verifyStatus === 'die') regTotalDie.value++
        else if (data.verifyStatus) regTotalUnknown.value++
        pushLog(data.index, data.phone, `✅ THÀNH CÔNG — UID: ${data.uid || '(chờ OTP)'}`)
      } else {
        // Fail: register:token không fire → đây là nơi duy nhất đếm fail + processed.
        regTotalProcessed.value++
        regTotalFail.value++
        if (data.checkpoint) regTotalCheckpoint.value++
        pushLog(data.index, data.phone, `❌ THẤT BẠI — ${data.message}`)
      }
    }
    pendingDone = []
    // Auto-cleanup cho chạy tuần/tháng: cap 1200 threads, keep active; evict done.
    // Active threads (running) ALWAYS kept; done (success/failed) evicted khi vượt cap.
    if (registerThreads.value.size > 1200) {
      for (const [key, t] of registerThreads.value) {
        if (t.status === 'success' || t.status === 'failed') {
          registerThreads.value.delete(key)
          if (registerThreads.value.size <= 600) break
        }
      }
    }
  }

  function scheduleFlush() {
    if (!rafId) rafId = requestAnimationFrame(flushRegisterEvents)
  }

  // register:status — slot reset: queue vào RAF batch
  _registerUnsubs.push(bus.on('register:status', (data: StatusEvent) => {
    if (data.phone === 'system') { pushLog(data.index, data.phone, data.msg); return }
    // Overwrite entry trong queue nếu cùng slot (chỉ giữ cái mới nhất)
    const existing = pendingStatus.findIndex(e => e.index === data.index)
    if (existing >= 0) pendingStatus[existing] = { ...pendingStatus[existing], ...data, reset: pendingStatus[existing].reset || data.reset }
    else pendingStatus.push(data)
    scheduleFlush()
  }))

  // register:batch-status — batch updates mỗi 500ms từ Go
  _registerUnsubs.push(bus.on('register:batch-status', (updates: Array<{ index: number; phone: string; proxy: string; proxyServer?: string; userAgent?: string; msg: string }>) => {
    for (const data of updates) {
      const existing = registerThreads.value.get(data.index)
      if (existing) {
        // Không ghi đè nếu thread đã xong (tránh batch ghi đè message lỗi cuối).
        // Task 6 dedup: skip nếu activity giống — tránh ghi reactive state cùng giá trị.
        if (existing.status === 'success' || existing.status === 'failed') continue
        if (data.userAgent) existing.userAgent = data.userAgent
        if (data.proxy) existing.proxy = data.proxy
        if (data.proxyServer) existing.proxyServer = data.proxyServer
        if (data.phone) existing.phone = data.phone
        if (existing.activity === data.msg) continue
        existing.activity = data.msg
        if (/\[Retry lần \d+\/\d+\]/.test(data.msg)) existing.inAddMailRetry = true
      } else {
        registerThreads.value.set(data.index, {
          index: data.index, phone: data.phone, proxy: data.proxy || '', proxyServer: data.proxyServer || '',
          userAgent: data.userAgent || '', uid: '', password: '', cookie: '', token: '', activity: data.msg,
          status: 'running', verifyStatus: '', rawVerifyStatus: '', verifyMessage: '', verifyEmail: '', inAddMailRetry: /\[Retry lần \d+\/\d+\]/.test(data.msg),
        })
      }
    }
  }))

  // register:account-done — queue vào RAF batch
  _registerUnsubs.push(bus.on('register:account-done', (data: DoneEvent) => {
    pendingDone.push(data)
    scheduleFlush()
  }))

  // register:token — emit ngay sau khi reg success (sau pre-fetch EAA),
  // cập nhật cột TOKEN/COOKIE realtime trên bảng register live, không phải chờ
  // verify xong mới có register:account-done.
  _registerUnsubs.push(bus.on('register:token', (data: TokenEvent) => {
    // Queue thay vì xử lý ngay → tránh race khi register:status chưa flush (entry chưa tồn tại)
    // mà register:token đã đến → handler không thấy entry → uid/token bị mất.
    pendingTokens.push(data)
    scheduleFlush()
  }))

  // register:output-path — Go emit ngay sau khi xác định thư mục lưu kết quả
  _registerUnsubs.push(bus.on('register:output-path', (data: { path: string }) => {
    if (data.path) activeRunOutputPath.value = data.path
  }))

  _registerUnsubs.push(bus.on('register:complete', (data: { total: number; error?: string }) => {
    isVerifyRunning.value = false
    isRegisterRunning.value = false
    isStopping.value = false
    stopElapsedTimer()
    // KHÔNG clear splitModeActive ở đây — user cần xem lại stats + grid của cả pane
    // REG và VER sau khi Stop. splitModeActive sẽ được reset khi:
    //   1. User bấm "Xóa bảng" (clearRegisterTable)
    //   2. User bấm Run mới (handleRun reset trước khi setup lại)
    //   3. Run mới start ở mode khác split (verify only / register only không-split)
    //
    // Counter localStorage CŨNG giữ — restore sau UI reload thấy lại stats final.
    // Error vẫn notify để user biết khi fail (vd proxy health check).
    // Success: KHÔNG toast — avoid flood khi register:complete fire nhiều lần từ pool kết thúc batch.
    // User vẫn thấy "🏁 Hoàn thành" trong log panel.
    if (data.error) {
      pushLog(0, 'system', `❌ Dừng vì: ${data.error}`)
      appStore.notify('error', data.error)
      // Lỗi cứng (proxy fail v.v.) → reset split để user không bị stuck UI cũ.
      splitModeActive.value = false
      clearRunStats()
    } else {
      pushLog(0, 'system', `🏁 Hoàn thành — ${data.total} tài khoản`)
    }
    // Auto-restart pending → KHÔNG fetchAccounts (giữ table sạch). Bình thường thì fetch.
    if (!_autoRestartPending) {
      accountsStore.fetchAccounts()
    }

    // AUTO-RESTART (2026-05-15): nếu trigger được fire trước đó, chạy lại từ đầu sau 2s.
    // Delay nhỏ để backend hoàn tất defer cleanup (state → idle) + UI render Done.
    // Retry tối đa 5 lần (mỗi 2s) nếu backend chưa idle, tránh fail im lặng.
    if (_autoRestartPending) {
      _autoRestartPending = false
      pushLog(0, 'system', '🔁 Auto-restart: chạy lại từ đầu sau 2s...')
      let attempt = 0
      const tryRestart = () => {
        attempt++
        if (isRegisterRunning.value || isVerifyRunning.value) {
          // State chưa idle, retry
          if (attempt < 5) {
            pushLog(0, 'system', `⏳ Auto-restart: chờ state idle (lần ${attempt}/5)...`)
            setTimeout(tryRestart, 2000)
          } else {
            pushLog(0, 'system', '❌ Auto-restart: timeout chờ state idle — hủy.')
            appStore.notify('error', 'Auto-restart thất bại: workers chưa cleanup xong')
          }
          return
        }
        appStore.notify('info', `Auto-restart: bắt đầu run mới (lần ${attempt})`)
        handleRunRegister()
      }
      setTimeout(tryRestart, 2000)
    }
  }))

  // AUTO-RESTART TRIGGER: backend timer fired → mark pending + RESET UI ngay lập tức
  // để user thấy phản hồi tức thì. Workers thực tế vẫn cần wg.Wait drain (~30-60s) nhưng
  // UI đã sạch counters/table. Khi register:complete tới → handler ở trên sẽ re-run.
  _registerUnsubs.push(bus.on('register:auto-restart-trigger', (data: { minutes: number }) => {
    _autoRestartPending = true
    pushLog(0, 'system', `⏰ Auto-restart triggered (sau ${data.minutes} phút) — reset UI và chờ workers drain...`)
    appStore.notify('warning', `Auto-restart sau ${data.minutes} phút — đang reset...`)
    // RESET UI ngay: counters về 0, table sạch, threads sạch — UX phản hồi tức thì.
    regTotalProcessed.value = 0
    regTotalSuccess.value = 0
    regTotalFail.value = 0
    regTotalLive.value = 0
    regTotalDie.value = 0
    regTotalUnknown.value = 0
    regTotalCheckpoint.value = 0
    verifyTotalLive.value = 0
    verifyTotalDie.value = 0
    verifyTotalUnknown.value = 0
    verifyTotalProcessed.value = 0
    _verifyLastStatus.clear()
    _verifyLastStatusByUid.clear()
    verifiedRunIds.clear()
    registerThreads.value = new Map()
    accountsStore.clearAccounts()
    clearRunStats()
    // Reset elapsed timer về 0 — đếm lại từ thời điểm trigger
    startElapsedTimer()
  }))
}

// Flag cờ kích hoạt auto-restart sau khi register:complete tới.
let _autoRestartPending = false

function clearRegisterTable() {
  registerThreads.value = new Map()
  registerLogs.value = []
  // Task fix: bấm "Xóa bảng" cũng dọn split layout + counter để về trạng thái sạch.
  splitModeActive.value = false
  // Reset accumulated counters
  regTotalProcessed.value = 0
  regTotalSuccess.value = 0
  regTotalFail.value = 0
  regTotalLive.value = 0
  regTotalDie.value = 0
  regTotalUnknown.value = 0
  regTotalCheckpoint.value = 0
  verifyTotalLive.value = 0
  verifyTotalDie.value = 0
  verifyTotalUnknown.value = 0
  verifyTotalProcessed.value = 0
  clearRunStats()
}

async function handleRunRegister() {
  clearGridSort()
  isVerifyRunning.value = true
  isRegisterRunning.value = true
  registerLogs.value = []
  registerThreads.value = new Map()
  activeRunOutputPath.value = ''
  regTotalProcessed.value = 0
  regTotalSuccess.value = 0
  regTotalFail.value = 0
  regTotalLive.value = 0
  regTotalDie.value = 0
  regTotalUnknown.value = 0
  regTotalCheckpoint.value = 0
  // Clear stats cũ — run mới đếm lại từ 0
  clearRunStats()
  verifyTotalLive.value = 0
  verifyTotalDie.value = 0
  verifyTotalUnknown.value = 0
  verifyTotalProcessed.value = 0
  _verifyLastStatus.clear()
  _verifyLastStatusByUid.clear()
  verifiedRunIds.clear()
  accountsStore.clearAccounts()
  showRegisterPanel.value = true
  regViewMode.value = 'table'
  startElapsedTimer()
  await setupRegisterListeners()

  try {
    const runner = await getVerifyRunnerService()
    // Load checkpoint limit config để auto-stop khi vượt ngưỡng
    try {
      const cfg = await runner.loadInteractionConfig() as any
      checkpointLimitEnabled.value = cfg.limitCheckpoint === true
      checkpointLimitCount.value = Number(cfg.limitCheckpointCount) || 50
    } catch { /* ignore — default no auto-stop */ }
    const threads = generalConfig.value.threadRequest || 1
    const result = await runner.runRegister(threads)

    const isError = result.includes('Không có profile') || result.includes('Lỗi') || result.includes('vui lòng dừng')
    if (isError) {
      appStore.notify('error', result)
      isVerifyRunning.value = false
      isRegisterRunning.value = false
      splitModeActive.value = false  // lỗi khởi động → reset ngay (run chưa bắt đầu)
    } else {
      pushLog(0, 'system', result)
    }
  } catch (err) {
    appStore.notify('error', 'Lỗi khi chạy đăng ký: ' + String(err))
    isVerifyRunning.value = false
    isRegisterRunning.value = false
    splitModeActive.value = false  // lỗi khởi động → reset ngay
  }
}
</script>

<template>
  <div class="accounts-page" :class="{ 'accounts-page--dragging': isDragging }">
    <!-- Toolbar: hiển thị checked count -->
    <AccountsToolbar
      :selected-count="selection.checkedCount.value"
      :total-count="accountsStore.total"
      :filter-keyword="grid.filterKeyword.value"
      :is-running="isRegisterRunning || isVerifyRunning"
      :is-stopping="isStopping"
      :clone-hv-mode="cloneHvMode"
      :file-mode="fileMode"
      :result-folder-path="resultFolderPath"
      @update:filter-keyword="grid.filterKeyword.value = $event"
      @import="showImportDialog = true"
      @delete="handleDelete"
      @run="handleRun"
      @stop="handleStop"
      @toggle-columns="showColumnsDropdown = !showColumnsDropdown"
      @settings="showSettings = true"
      @open-result-folder="handleOpenResultFolder"
      @open-config-folder="handleOpenConfigFolder"
    />

    <!-- Columns dropdown panel -->
    <Teleport to="body">
      <div
        v-if="showColumnsDropdown"
        class="columns-overlay"
        @click.self="showColumnsDropdown = false"
      >
        <div class="columns-panel">
          <div class="columns-panel__header">
            <span>Hiển thị cột</span>
            <span class="columns-panel__count">{{ totalVisibleCols }} / {{ columnVis.allColumns.length }} cột</span>
            <button class="columns-panel__reset" @click="columnVis.resetToDefault()">Reset mặc định</button>
            <button class="columns-panel__close" @click="showColumnsDropdown = false"><X :size="14" /></button>
          </div>
          <div class="columns-panel__body">
            <div v-for="group in columnGroups" :key="group.label" class="columns-panel__group">
              <!-- Group header with select-all checkbox -->
              <div class="columns-panel__group-header">
                <label class="columns-panel__group-check">
                  <input
                    type="checkbox"
                    :checked="isGroupAllChecked(group.keys)"
                    @change="(e) => toggleGroup(group.keys, (e.target as HTMLInputElement).checked)"
                  />
                  <span>{{ group.label.toUpperCase() }}</span>
                </label>
              </div>
              <!-- Group items -->
              <div class="columns-panel__group-items">
                <label
                  v-for="col in getGroupColumns(group.keys)"
                  :key="col.key"
                  class="columns-panel__item"
                >
                  <input
                    type="checkbox"
                    :checked="columnVis.isVisible(col.key)"
                    @change="columnVis.toggle(col.key)"
                  />
                  <span>{{ col.label }}</span>
                </label>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <div ref="gridContentRef" class="accounts-page__content" :class="{ 'accounts-page__content--split': splitModeActive }">
      <!-- ===== SPLIT MODE: 2 grids stacked ===== -->
      <template v-if="splitModeActive">
        <!-- REG pool (trên) -->
        <div class="split-pane">
          <div class="split-pane__header">
            <span class="split-pane__label split-pane__label--reg">REG</span>
          </div>
          <DataGrid
            :rows="regGridRows"
            :columns="visibleColumns"
            :total-height="regGridRows.length * 36"
            :offset-top="0"
            :sort-column="grid.sort.value.column"
            :sort-direction="grid.sort.value.direction"
            :sort-disabled="isRunSortingLocked"
            :selected-ids="selection.checkedIds.value"
            :highlighted-ids="selection.highlightedIds.value"
            :selected-cells="regSelectedCells"
            :row-class="getRowClass"
            @sort="handleSort"
            @scroll="() => {}"
            @row-click="handleRowClick"
            @row-dblclick="handleRowDblClick"
            @toggle-all="handleToggleAll"
            @toggle-row="handleToggleRow"
            @contextmenu="handleContextMenu"
            @row-mousedown="handleRowMouseDown"
            @row-mouseenter="handleRowMouseEnter"
            @cell-click="(r, c, e) => handleCellClick('reg', r, c, e)"
            @cell-mousedown="(r, c, e) => handleCellMouseDown('reg', r, c, e)"
            @cell-mouseenter="(r, c) => handleCellMouseEnter('reg', r, c)"
          >
            <template #cell-status="{ value }">
              <span class="status-badge" :class="`status-badge--${value}`">{{ value }}</span>
            </template>
            <template #cell-uid="{ value }">
              <span class="uid-cell">{{ value }}</span>
            </template>
            <template #cell-proxy="{ row }">
              <span v-if="(row as any).proxy" class="proxy-cell" :title="(row as any).proxy">{{ (row as any).proxy }}</span>
            </template>
            <template #cell-runProxy="{ row }">
              <span v-if="(row as any).runProxy" class="proxy-cell" :title="(row as any).runProxy">{{ (row as any).runProxy }}</span>
            </template>
            <template #cell-activity="{ value }">
              <span v-if="value" class="run-cell" :title="String(value)">{{ value }}</span>
            </template>
          </DataGrid>
        </div>

        <!-- VERIFY pool (dưới) -->
        <div class="split-pane">
          <div class="split-pane__header">
            <span class="split-pane__label split-pane__label--verify">VERIFY</span>
          </div>
          <DataGrid
            :rows="grid.visibleItems.value"
            :columns="visibleColumns"
            :total-height="grid.totalHeight.value"
            :offset-top="grid.offsetTop.value"
            :sort-column="grid.sort.value.column"
            :sort-direction="grid.sort.value.direction"
            :sort-disabled="isRunSortingLocked"
            :selected-ids="selection.checkedIds.value"
            :highlighted-ids="selection.highlightedIds.value"
            :selected-cells="verSelectedCells"
            :row-class="getRowClass"
            @sort="handleSort"
            @scroll="grid.onScroll"
            @row-click="handleRowClick"
            @row-dblclick="handleRowDblClick"
            @toggle-all="handleToggleAll"
            @toggle-row="handleToggleRow"
            @contextmenu="handleContextMenu"
            @row-mousedown="handleRowMouseDown"
            @row-mouseenter="handleRowMouseEnter"
            @cell-click="(r, c, e) => handleCellClick('ver', r, c, e)"
            @cell-mousedown="(r, c, e) => handleCellMouseDown('ver', r, c, e)"
            @cell-mouseenter="(r, c) => handleCellMouseEnter('ver', r, c)"
          >
            <template #cell-status="{ value }">
              <span class="status-badge" :class="`status-badge--${value}`">{{ value }}</span>
            </template>
            <template #cell-uid="{ value }">
              <span class="uid-cell">{{ value }}</span>
            </template>
            <template #cell-checkpoint="{ value }">
              <span v-if="value" class="cp-badge">{{ value }}</span>
            </template>
            <template #cell-proxy="{ row }">
              <span v-if="(row as any).proxy" class="proxy-cell" :title="(row as any).proxy">{{ (row as any).proxy }}</span>
            </template>
            <template #cell-runProxy="{ row }">
              <span v-if="(row as any).runProxy" class="proxy-cell" :title="(row as any).runProxy">{{ (row as any).runProxy }}</span>
            </template>
            <template #cell-activity="{ value }">
              <span v-if="value" class="run-cell" :title="String(value)">{{ value }}</span>
            </template>
          </DataGrid>
        </div>
      </template>

      <!-- ===== NORMAL MODE: single grid ===== -->
      <template v-else>
        <!-- Loading / error / empty — chỉ hiện khi không ở chế độ đăng ký -->
        <div v-if="!isRegMode && accountsStore.loading && accountsStore.accounts.length === 0" class="accounts-page__state">Đang tải dữ liệu...</div>
        <div v-else-if="!isRegMode && accountsStore.error && accountsStore.accounts.length === 0" class="accounts-page__state">
          <p>{{ accountsStore.error }}</p>
          <button @click="accountsStore.fetchAccounts()">Thử lại</button>
        </div>
        <div v-else-if="!isRegMode && !accountsStore.loading && accountsStore.accounts.length === 0" class="accounts-page__state">
          <div class="accounts-page__empty-icon"><Users :size="40" /></div>
          <div class="accounts-page__empty-title">Chưa có tài khoản nào</div>
          <div class="accounts-page__empty-desc">
            Import từ file .txt hoặc mua tự động qua CloneHV API
          </div>
          <div class="accounts-page__empty-actions">
            <button class="accounts-page__cta" @click="showImportDialog = true">
              <FolderOpen :size="15" /> Import từ file
            </button>
            <button class="accounts-page__cta accounts-page__cta--secondary"
              @click="$router.push('/general-settings')">
              <Key :size="15" /> Thiết lập CloneHV
            </button>
          </div>
        </div>

        <!-- Data Grid — dùng cho cả accounts thường lẫn chế độ đăng ký -->
        <DataGrid
          v-else
          :rows="isRegMode ? regGridRows : grid.visibleItems.value"
          :columns="visibleColumns"
          :total-height="isRegMode ? regGridRows.length * 36 : grid.totalHeight.value"
          :offset-top="isRegMode ? 0 : grid.offsetTop.value"
          :sort-column="grid.sort.value.column"
          :sort-direction="grid.sort.value.direction"
          :sort-disabled="isRunSortingLocked"
          :selected-ids="selection.checkedIds.value"
          :highlighted-ids="selection.highlightedIds.value"
          :selected-cells="isRegMode ? regSelectedCells : normalSelectedCells"
          :row-class="getRowClass"
          @sort="handleSort"
          @scroll="grid.onScroll"
          @row-click="handleRowClick"
          @row-dblclick="handleRowDblClick"
          @toggle-all="handleToggleAll"
          @toggle-row="handleToggleRow"
          @contextmenu="handleContextMenu"
          @row-mousedown="handleRowMouseDown"
          @row-mouseenter="handleRowMouseEnter"
          @cell-click="(r, c, e) => handleCellClick(isRegMode ? 'reg' : 'normal', r, c, e)"
          @cell-mousedown="(r, c, e) => handleCellMouseDown(isRegMode ? 'reg' : 'normal', r, c, e)"
          @cell-mouseenter="(r, c) => handleCellMouseEnter(isRegMode ? 'reg' : 'normal', r, c)"
        >
          <template #cell-status="{ value }">
            <span class="status-badge" :class="`status-badge--${value}`">{{ value }}</span>
          </template>
          <template #cell-uid="{ value }">
            <span class="uid-cell">{{ value }}</span>
          </template>
          <template #cell-checkpoint="{ value }">
            <span v-if="value" class="cp-badge">{{ value }}</span>
          </template>
          <template #cell-proxy="{ row }">
            <span v-if="(row as any).proxy" class="proxy-cell" :title="(row as any).proxy">
              {{ (row as any).proxy }}
            </span>
          </template>
          <template #cell-runProxy="{ row }">
            <span v-if="(row as any).runProxy" class="proxy-cell" :title="(row as any).runProxy">
              {{ (row as any).runProxy }}
            </span>
          </template>
          <template #cell-activity="{ value }">
            <span v-if="value" class="run-cell" :title="String(value)">{{ value }}</span>
          </template>
        </DataGrid>
      </template>

      <AccountsDetailPanel
        :account="accountsStore.detailAccount"
        :show="accountsStore.showDetailPanel"
        @close="accountsStore.closeDetail()"
      />
    </div>

    <!-- Stats bar -->
    <Teleport v-if="statusBarSlotReady" to="#status-bar-page-slot">
      <div class="accounts-stats-bar">
        <template v-if="isRegisterRunning || registerThreads.size > 0 || regTotalProcessed > 0">
        <!-- REG stats -->
        <span class="stats-item stats-item--label">REGISTER {{ regSuccessRate }}%</span>
        <span class="stats-item stats-item--sep">|</span>
        <span class="stats-item" :class="isRegisterRunning ? 'stats-item--running' : ''">
          {{ isRegisterRunning ? '● Đang chạy' : '■ Đã xong' }}
        </span>
        <span class="stats-item stats-item--sep">|</span>
        <span class="stats-item stats-item--live">Live: {{ regTotalSuccess }}</span>
        <span class="stats-item stats-item--sep">|</span>
        <span class="stats-item stats-item--die">Die: {{ regTotalFail }}</span>
        <template v-if="regTotalCheckpoint > 0">
          <span class="stats-item stats-item--sep">|</span>
          <span class="stats-item stats-item--checkpoint">Checkpoint: {{ regTotalCheckpoint }}</span>
        </template>
        <template v-if="verifiedLiveTotal + verifiedDieTotal + verifiedUnknownTotal > 0">
          <span class="stats-item stats-item--sep stats-item--sep-thick">‖</span>
          <span class="stats-item stats-item--label stats-item--label-ver">VERIFIED {{ regVerifyLiveRate }}%</span>
          <span class="stats-item stats-item--sep">|</span>
          <span class="stats-item stats-item--live">Live: {{ verifiedLiveTotal }}</span>
          <span class="stats-item stats-item--sep">|</span>
          <span class="stats-item stats-item--die">Die: {{ verifiedDieTotal }}</span>
          <template v-if="verifiedUnknownTotal > 0">
            <span class="stats-item stats-item--sep">|</span>
            <span class="stats-item stats-item--unknown">Unknown: {{ verifiedUnknownTotal }}</span>
          </template>
        </template>
        <template v-if="uploadLogStore.stats.totalUploaded > 0 || uploadLogStore.isRunning">
          <span class="stats-item stats-item--sep stats-item--sep-thick">‖</span>
          <span class="stats-item stats-item--upload" :class="{ 'stats-item--upload-active': uploadLogStore.isRunning }">
            {{ uploadLogStore.isRunning ? '↑' : '✓' }} Upload: {{ uploadLogStore.totalUploaded }}
          </span>
        </template>
        <span class="stats-item stats-item--sep">|</span>
        <span class="stats-item stats-item--elapsed">{{ regElapsed }}</span>

        <!-- REMOVED 2026-05-16: VER stats block trùng với VERIFIED ở trên (cùng đếm
             inline verify result). Sau khi refactor Split = pure UI, inline verify
             update vào regTotalLive/Die → VERIFIED block đã hiển thị đầy đủ → block
             này bị duplicate. Verify panel ở dưới có counter riêng nếu cần. -->

        <span class="stats-item stats-item--sep" style="margin-left:auto"></span>
        <button class="stats-clear-btn" @click="clearRegisterTable">Xóa bảng</button>
        </template>
        <template v-else-if="isVerifyRunning || verifyTotalProcessed > 0">
        <!-- Verify đang chạy hoặc vừa xong: hiển thị counter tích lũy (không bị reset theo slot) -->
        <span class="stats-item" :class="isVerifyRunning ? 'stats-item--running' : ''">
          {{ isVerifyRunning ? '● Đang verify' : '■ Đã xong' }}
        </span>
        <span class="stats-item stats-item--sep">|</span>
        <span class="stats-item stats-item--live">Live: {{ verifyTotalLive }}</span>
        <span class="stats-item stats-item--sep">|</span>
        <span class="stats-item stats-item--die">Die: {{ verifyTotalDie }}</span>
        <!-- Unknown bị ẩn theo yêu cầu: counter vẫn track ở backend nhưng KHÔNG hiển thị UI -->
        <span class="stats-item stats-item--sep">|</span>
        <span class="stats-item stats-item--elapsed">Time Elapsed: {{ regElapsed }}</span>
        <span class="stats-item stats-item--sep">|</span>
        <span class="stats-item">Bôi đen: {{ pageStats.highlighted }}</span>
        <span class="stats-item stats-item--sep">|</span>
        <span class="stats-item">Đã chọn: {{ pageStats.selected }}</span>
        </template>
        <template v-else>
        <span class="stats-item stats-item--live">Live: {{ pageStats.live }}</span>
        <span class="stats-item stats-item--sep">|</span>
        <span class="stats-item stats-item--die">Die: {{ pageStats.die }}</span>
        <span class="stats-item stats-item--sep">|</span>
        <span class="stats-item stats-item--unknown">Unknown: {{ pageStats.unknown }}</span>
        <span class="stats-item stats-item--sep">|</span>
        <span v-if="pageStats.total > MAX_DISPLAY_ROWS" class="stats-item stats-item--display-limit">
          Hiển thị: {{ MAX_DISPLAY_ROWS }}/{{ pageStats.total }}
        </span>
        <span v-if="pageStats.total > MAX_DISPLAY_ROWS" class="stats-item stats-item--sep">|</span>
        <span class="stats-item">Bôi đen: {{ pageStats.highlighted }}</span>
        <span class="stats-item stats-item--sep">|</span>
        <span class="stats-item">Đã chọn: {{ pageStats.selected }}</span>
        </template>
      </div>
    </Teleport>

    <ContextMenu
      :visible="ctxMenu.visible.value"
      :x="ctxMenu.x.value"
      :y="ctxMenu.y.value"
      :items="(ctxMenu.items.value as any)"
      @close="ctxMenu.hide()"
    />

    <GeneralSettingsModal
      :show="showSettings"
      :config="generalConfig"
      :ip-config="ipConfig"
      @close="showSettings = false"
      @save="handleSaveSettings"
    />

    <AccountsImportDialog
      :show="showImportDialog"
      :loading="importLoading"
      @close="showImportDialog = false"
      @import="handleImportData"
      @folder-imported="handleFolderImported"
    />

  </div>
</template>

<style scoped>
.accounts-page { display: flex; flex-direction: column; height: 100%; overflow: hidden; }
.accounts-page--dragging { user-select: none; cursor: default; }
.accounts-page__content { flex: 1; display: flex; overflow: hidden; }
.accounts-page__content--split { flex-direction: column; }

/* Split mode panes */
.split-pane {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}
.split-pane + .split-pane {
  border-top: 2px solid var(--border-strong);
}
.split-pane__header {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: 0 var(--space-3);
  height: 28px;
  background: rgba(255, 255, 255, 0.07);
  border-bottom: 1px solid rgba(255, 255, 255, 0.12);
  font-size: var(--font-size-xs);
  color: var(--text-primary);
  user-select: none;
}
.split-pane__label {
  font-weight: 800;
  font-size: 10px;
  padding: 2px 7px;
  border-radius: var(--radius-sm);
  letter-spacing: 0.08em;
}
.split-pane__label--reg    { background: rgba(99,102,241,0.30); color: #a5b4fc; }
.split-pane__label--verify { background: rgba(34,197,94,0.25);  color: #4ade80; }
.split-pane__count { color: var(--text-secondary); font-weight: 600; }
.split-pane__sep   { color: rgba(255,255,255,0.2); }
.split-pane__stat  { font-weight: 600; color: var(--text-primary); }
.split-pane__stat--running { color: #4ade80; }
.split-pane__stat--live    { color: #4ade80; font-weight: 700; }
.split-pane__stat--die     { color: #f87171; font-weight: 700; }

.accounts-stats-bar {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  font-size: var(--font-size-sm);
  color: var(--text-secondary);
  user-select: none;
}
.stats-item { display: flex; align-items: center; flex-shrink: 0; font-weight: 500; white-space: nowrap; }
.stats-item--live { color: var(--success-text); font-weight: 700; font-size: var(--font-size-base); }
.stats-item--die { color: var(--danger-text); font-weight: 700; font-size: var(--font-size-base); }
.stats-item--checkpoint { color: #f59e0b; font-weight: 700; font-size: var(--font-size-base); }
.stats-item--unknown { color: var(--text-muted); font-weight: 700; font-size: var(--font-size-base); }
.stats-item--elapsed { color: #f97316; font-variant-numeric: tabular-nums; font-weight: 600; }
.stats-item--sep { color: var(--border-strong); }
.stats-item--sep-thick { color: rgba(255,255,255,0.3); font-size: 14px; margin: 0 2px; }
.stats-item--display-limit { color: var(--warning-text, #f59e0b); font-weight: 600; font-size: var(--font-size-sm); }
.stats-item--label { font-size: 10px; font-weight: 800; padding: 1px 6px; border-radius: var(--radius-sm); letter-spacing: 0.06em; background: rgba(59,130,246,0.30); color: #1e293b; }
.stats-item--label-ver { background: rgba(22,163,74,0.70); color: #f0fdf4; }
.stats-item--upload { color: #60a5fa; font-weight: 600; font-size: var(--font-size-base); }
.stats-item--upload-active { color: #3b82f6; animation: upload-pulse 1.2s ease-in-out infinite; }
@keyframes upload-pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.55; } }
.stats-item--running { color: #4ade80; font-weight: 700; }

.accounts-page__state {
  flex: 1; display: flex; flex-direction: column;
  align-items: center; justify-content: center;
  color: var(--text-secondary); gap: var(--space-3);
}
.accounts-page__empty-icon   { font-size: 40px; opacity: 0.3; }
.accounts-page__empty-title  { font-size: var(--font-size-md); font-weight: 600; color: var(--text-primary); }
.accounts-page__empty-desc   { font-size: var(--font-size-sm); color: var(--text-muted); text-align: center; max-width: 300px; }
.accounts-page__empty-actions { display: flex; gap: var(--space-3); flex-wrap: wrap; justify-content: center; }

.accounts-page__cta {
  padding: var(--space-2) var(--space-4); border-radius: var(--radius-md);
  background: var(--brand-gradient, linear-gradient(135deg, #833AB4, #E1306C, #FD1D1D));
  color: white; font-weight: 600;
  border: none;
}
.accounts-page__cta--secondary {
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  color: var(--text-secondary);
}
.accounts-page__cta--secondary:hover {
  border-color: var(--brand-primary);
  color: var(--brand-primary);
  background: rgba(34,197,94,0.06);
}

.status-badge {
  display: inline-block; padding: 2px 8px;
  border-radius: var(--radius-full); font-size: var(--font-size-xs); font-weight: 500;
}
.status-badge--live { background: var(--success-bg); color: var(--success-text); }
.status-badge--die { background: rgba(220, 38, 38, 0.22); color: #ef4444; }
.status-badge--checkpoint { background: var(--warning-bg); color: var(--warning-text); }
.status-badge--new { background: var(--info-bg); color: var(--info-text); }
.status-badge--unknown { background: var(--surface-sunken); color: var(--text-muted); border: 1px solid var(--border-default); }
.status-badge--nvr { background: rgba(234, 179, 8, 0.26); color: #ca8a04; }
.status-badge--addmail { background: rgba(194, 65, 12, 0.28); color: #ea580c; }
.status-badge--verfail { background: rgba(107, 114, 128, 0.18); color: #4b5563; }

.uid-cell { color: var(--text-link); font-weight: 500; }
.cp-badge {
  display: inline-block; padding: 1px 6px; border-radius: var(--radius-sm);
  font-size: var(--font-size-xs); background: var(--warning-bg); color: var(--warning-text);
}
.run-cell { font-size: var(--font-size-sm); color: var(--text-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 100%; }
.proxy-cell { font-size: var(--font-size-xs); color: var(--info-text); font-family: var(--font-mono); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 100%; }

/* Columns panel */
.columns-overlay {
  position: fixed;
  inset: 0;
  z-index: 300;
  background: var(--surface-overlay);
}
.columns-panel {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: min(920px, 92vw);
  max-height: 85vh;
  background: var(--surface-elevated);
  border: 1px solid var(--border-default);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  display: flex;
  flex-direction: column;
  overflow: hidden;
  z-index: 301;
}
.columns-panel__header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--border-default);
  font-size: var(--font-size-sm);
  font-weight: 600;
  color: var(--text-primary);
  flex-shrink: 0;
}
.columns-panel__count {
  font-size: var(--font-size-xs);
  font-weight: 500;
  color: var(--text-muted);
  background: var(--surface-sunken);
  padding: 1px 7px;
  border-radius: 20px;
}
.columns-panel__reset {
  margin-left: auto;
  font-size: var(--font-size-xs);
  color: var(--brand-primary);
  background: none;
  border: none;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
}
.columns-panel__reset:hover { text-decoration: underline; }
.columns-panel__close {
  color: var(--text-muted);
  background: none;
  border: none;
  cursor: pointer;
  padding: 2px 4px;
  display: flex;
  align-items: center;
}
.columns-panel__close:hover { color: var(--text-primary); }

.columns-panel__body {
  overflow-y: auto;
  padding: var(--space-2) var(--space-3);
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

/* Groups */
.columns-panel__group {
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  overflow: hidden;
}
.columns-panel__group-header {
  background: var(--surface-sunken);
  padding: 5px var(--space-3);
  border-bottom: 1px solid var(--border-subtle);
}
.columns-panel__group-check {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
  font-size: 11px;
  font-weight: 700;
  color: var(--text-secondary);
  letter-spacing: 0.05em;
}
.columns-panel__group-check input { accent-color: var(--brand-primary); cursor: pointer; }

.columns-panel__group-items {
  padding: var(--space-2) var(--space-3);
  display: flex;
  flex-wrap: wrap;
  gap: 2px 0;
}

.columns-panel__item {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 4px 8px;
  cursor: pointer;
  font-size: var(--font-size-xs);
  color: var(--text-primary);
  border-radius: var(--radius-sm);
  white-space: nowrap;
  min-width: 100px;
}
.columns-panel__item:hover { background: var(--surface-hover-subtle); }
.columns-panel__item input[type="checkbox"] {
  accent-color: var(--brand-primary);
  width: 13px;
  height: 13px;
  cursor: pointer;
  flex-shrink: 0;
}

.stats-item--running { color: #3fb950; font-weight: 700; animation: pulse-dot 1.4s ease-in-out infinite; }
.stats-clear-btn {
  background: none; border: 1px solid var(--border-default); color: var(--text-muted);
  flex-shrink: 0; white-space: nowrap; font-size: 11px; padding: 2px 8px; border-radius: 4px; cursor: pointer; margin-left: auto;
}
.stats-clear-btn:hover { color: var(--text-primary); border-color: var(--border-strong); }


/* Log line */
.reg-log-line {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 1px 12px;
  white-space: pre-wrap;
  word-break: break-all;
}
.reg-log-line:hover { background: var(--surface-hover-subtle); }
.reg-log-line__time  { color: #484f58; flex-shrink: 0; font-size: 11px; width: 60px; }
.reg-log-line__thread { color: #58a6ff; flex-shrink: 0; font-size: 11px; max-width: 160px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.reg-log-line__msg   { color: #8b949e; }
.reg-log-line--success .reg-log-line__msg { color: #3fb950; }
.reg-log-line--error   .reg-log-line__msg { color: #f85149; }
.reg-log-line--step    .reg-log-line__msg { color: #c9d1d9; }
.reg-log-line--info    .reg-log-line__msg { color: #8b949e; }
</style>
