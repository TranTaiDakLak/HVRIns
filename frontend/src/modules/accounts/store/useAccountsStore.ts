// useAccountsStore.ts — Store quản lý state cho Accounts module
// Rows, selection, filters, sort, loading, error

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getAccountService } from '../../../bridge/client'
import type { Account, AccountFilter, AccountStatus, BridgeError } from '../../../bridge/contracts'
import { useAppStore } from '../../../stores/app.store'

/**
 * Store trung tâm cho Accounts module.
 * Quản lý danh sách accounts, trạng thái loading, filter,
 * detail panel, và cache IP chạy realtime của verify runner.
 */
export const useAccountsStore = defineStore('accounts', () => {
  // === State ===
  const accounts = ref<Account[]>([])
  // Index O(1) — cập nhật mỗi khi accounts thay đổi, dùng cho realtime event handlers
  const accountsIndex = new Map<number, Account>()
  const loading = ref(false)
  const error = ref<string | null>(null)
  const filter = ref<AccountFilter>({})
  const detailAccount = ref<Account | null>(null)
  const showDetailPanel = ref(false)
  // Trạng thái verify đang chạy — persist trong store để không mất khi chuyển tab
  const isVerifyRunning = ref(false)
  // Cache IP chạy realtime — persist trong store, không bị xóa khi fetchAccounts.
  // Cap 2000 entries (LRU): 24/7 chạy nhiều không leak RAM.
  const runProxyCache = new Map<number, string>()
  const displayProxyCache = new Map<number, string>()
  const MAX_PROXY_CACHE = 2000

  // capMap giữ map dưới limit, xoá entries cũ nhất (Map insertion-order = LRU).
  function capMap<K, V>(m: Map<K, V>, limit: number) {
    if (m.size <= limit) return
    const dropCount = m.size - Math.floor(limit * 0.8) // drop 20% → batch cleanup, tránh mỗi set = delete
    let i = 0
    for (const k of m.keys()) {
      if (i >= dropCount) break
      m.delete(k)
      i++
    }
  }

  // === Computed ===
  const total = computed(() => accounts.value.length)

  // Thống kê theo status — single pass O(n) thay vì 4 lần filter
  const stats = computed(() => {
    let live = 0, die = 0, checkpoint = 0, newCount = 0
    for (const a of accounts.value) {
      if (a.status === 'live') live++
      else if (a.status === 'die') die++
      else if (a.status === 'checkpoint') checkpoint++
      else if (a.status === 'new') newCount++
    }
    return { total: accounts.value.length, live, die, checkpoint, new: newCount }
  })

  // === Actions ===

  /**
   * Tải danh sách accounts từ bridge (Go backend hoặc mock).
   * Tự động giữ lại runProxy cache để không xóa IP đang chạy.
   * @param filterOverride - Filter tạm thời cho lần gọi này. Nếu bỏ trống, dùng filter hiện tại.
   */
  async function fetchAccounts(filterOverride?: AccountFilter) {
    loading.value = true
    error.value = null

    try {
      const service = await getAccountService()
      const result = await service.list(filterOverride || filter.value)
      // Restore proxy caches — không để fetchAccounts xóa mất proxy đang chạy
      accounts.value = result.items.map(acc => ({
        ...acc,
        runProxy: runProxyCache.get(acc.id) ?? acc.runProxy ?? '',
        proxy: displayProxyCache.get(acc.id) ?? acc.proxy ?? '',
      }))
      // Rebuild index
      accountsIndex.clear()
      for (const acc of accounts.value) accountsIndex.set(acc.id, acc)
    } catch (err) {
      const bridgeErr = err as BridgeError
      error.value = bridgeErr.message || 'Lỗi khi tải danh sách accounts'
      const appStore = useAppStore()
      appStore.notify('error', error.value)
    } finally {
      loading.value = false
    }
  }

  /**
   * Import accounts từ chuỗi text (paste từ clipboard hoặc đọc từ file).
   * Format mỗi dòng: `uid|password|2fa|cookie|token|email|passMail`.
   * Tự động reload danh sách sau khi import xong.
   * @param data - Chuỗi text chứa accounts, mỗi dòng 1 account
   */
  async function importAccounts(data: string) {
    loading.value = true
    error.value = null

    try {
      const service = await getAccountService()
      const result = await service.import(data)
      const appStore = useAppStore()

      if (result.errors.length > 0) {
        appStore.notify('warning', `Import ${result.imported} accounts, ${result.errors.length} lỗi`)
      } else {
        appStore.notify('success', `Import thành công ${result.imported} accounts`)
      }

      // Reload danh sách sau import
      await fetchAccounts()
      return result
    } catch (err) {
      const bridgeErr = err as BridgeError
      error.value = bridgeErr.message || 'Lỗi khi import accounts'
      const appStore = useAppStore()
      appStore.notify('error', error.value)
      throw err
    } finally {
      loading.value = false
    }
  }

  /**
   * Xóa accounts theo danh sách IDs. Tự động reload danh sách sau khi xóa.
   * @param ids - Mảng ID của các accounts cần xóa
   * @param silent - Bỏ qua toast notify (dùng cho auto-clear trong verify flow).
   */
  async function deleteAccounts(ids: number[], silent = false) {
    if (ids.length === 0) return

    loading.value = true
    error.value = null

    try {
      const service = await getAccountService()
      const result = await service.delete(ids)
      if (!silent) {
        const appStore = useAppStore()
        appStore.notify('success', `Đã xóa ${result.deleted} accounts`)
      }

      // Reload danh sách sau xóa
      await fetchAccounts()
      return result
    } catch (err) {
      const bridgeErr = err as BridgeError
      error.value = bridgeErr.message || 'Lỗi khi xóa accounts'
      const appStore = useAppStore()
      appStore.notify('error', error.value)
      throw err
    } finally {
      loading.value = false
    }
  }

  /**
   * Mở detail panel và load thông tin đầy đủ của 1 account.
   * @param id - ID của account cần xem chi tiết
   */
  async function openDetail(id: number) {
    try {
      const service = await getAccountService()
      detailAccount.value = await service.get(id)
      showDetailPanel.value = true
    } catch (err) {
      const bridgeErr = err as BridgeError
      const appStore = useAppStore()
      appStore.notify('error', bridgeErr.message || 'Lỗi khi tải chi tiết account')
    }
  }

  /** Đóng detail panel và xóa account đang xem. */
  function closeDetail() {
    showDetailPanel.value = false
    detailAccount.value = null
  }

  /**
   * Lưu IP proxy đang chạy cho 1 account vào cache và cập nhật trực tiếp trong store.
   * Dùng bởi verify runner để hiển thị IP realtime trong grid mà không cần fetchAccounts lại.
   * @param id - ID của account
   * @param ip - IP proxy đang dùng (VD: '103.x.x.x/vn')
   */
  function setRunProxy(id: number, ip: string) {
    runProxyCache.set(id, ip)
    capMap(runProxyCache, MAX_PROXY_CACHE)
    const acc = accountsIndex.get(id)
    if (acc) acc.runProxy = ip
  }

  /**
   * Cập nhật cột PROXY (proxy thô) cho account đang chạy — cache + in-memory.
   * Cache đảm bảo proxy không bị fetchAccounts xóa mất (race condition với accounts-updated).
   */
  function setDisplayProxy(id: number, proxyStr: string) {
    displayProxyCache.set(id, proxyStr)
    capMap(displayProxyCache, MAX_PROXY_CACHE)
    const acc = accountsIndex.get(id)
    if (acc) acc.proxy = proxyStr
  }

  /** Xóa toàn bộ cache proxy khi verify hoàn thành, chuẩn bị cho lần chạy tiếp theo. */
  function clearRunProxyCache() {
    runProxyCache.clear()
    displayProxyCache.clear()
  }

  /**
   * Cập nhật in-place 1 slot row khi account mới được assign vào slot đó.
   * Thay thế full fetchAccounts() trong hot path — O(1) index lookup, không IPC round-trip.
   *
   * Reset các field stale từ account trước (email, runProxy, activity),
   * set field mới (uid, password, phone, status).
   */
  function applySlotAssigned(data: {
    slotId: number
    uid: string
    password: string
    phone: string
    status: string
    userAgent?: string
    token?: string
    cookie?: string
  }) {
    let acc = accountsIndex.get(data.slotId)
    if (!acc) {
      // Race với verify:accounts-updated → fetchAccounts (async): backend đã emit slot-assigned
      // trong lúc FE còn đang fetch list slot ban đầu. Thay vì drop event, auto-tạo row placeholder
      // để UA/hoạt động realtime không bị rơi mất chỉ vì timing.
      acc = createEmptyAccount(data.slotId)
      accounts.value.push(acc)
      accountsIndex.set(data.slotId, acc)
    }
    acc.uid = data.uid
    acc.password = data.password
    acc.phone = data.phone
    acc.status = (data.status as AccountStatus) ?? 'new'
    // UA mới được pick khi assign slot — cập nhật để cột UA hiện đúng UA đang chạy.
    // Nếu BE không gửi (legacy event) thì giữ UA cũ — tránh clear bất ngờ.
    if (data.userAgent !== undefined) acc.userAgent = data.userAgent
    acc.token = data.token || ''
    acc.cookie = data.cookie || ''
    // Reset stale fields từ account cũ trong slot này
    acc.email = ''
    acc.runProxy = ''
    acc.activity = ''
    acc.noteRun = ''
    // acc.proxy KHÔNG reset về '' — để tránh sort jump khi user đang sort theo cột PROXY.
    // proxy sẽ được cập nhật ngay sau bởi verify:raw-proxy event. Cache xóa để fetchAccounts
    // không restore giá trị cũ.
    runProxyCache.delete(data.slotId)
    displayProxyCache.delete(data.slotId)
  }

  /**
   * Đảm bảo slot tồn tại trong store — dùng cho batch-status handler khi race với fetch.
   * Trả về acc hiện có hoặc tạo placeholder mới.
   */
  function ensureSlot(slotId: number): Account {
    let acc = accountsIndex.get(slotId)
    if (!acc) {
      acc = createEmptyAccount(slotId)
      accounts.value.push(acc)
      accountsIndex.set(slotId, acc)
    }
    return acc
  }

  function createEmptyAccount(id: number): Account {
    return {
      id,
      uid: '',
      fullData: '',
      password: '',
      twofa: '',
      email: '',
      passMail: '',
      mailRecovery: '',
      cookie: '',
      token: '',
      status: 'new' as AccountStatus,
      checkpoint: '',
      statusAds: '',
      bm: '',
      tkqc: '',
      chatSupport: '',
      fullName: '',
      location: '',
      avatar: '',
      cover: '',
      phone: '',
      proxy: '',
      userAgent: '',
      note: '',
      noteRun: '',
      importTime: '',
      category: '',
      lastRun: '',
      runProxy: '',
      activity: '',
      sourceCode: '',
    }
  }

  /**
   * Xóa 1 account khỏi danh sách hiển thị trong store (không xóa khỏi database).
   * Dùng sau khi verify runner xử lý xong 1 account và cần loại nó khỏi queue.
   * @param id - ID của account cần remove khỏi store
   */
  function removeAccount(id: number) {
    const idx = accounts.value.findIndex(a => a.id === id)
    if (idx !== -1) accounts.value.splice(idx, 1)
    accountsIndex.delete(id)
    runProxyCache.delete(id)
  }

  /**
   * Merge filter mới vào filter hiện tại (partial update).
   * @param newFilter - Các field filter cần cập nhật (chỉ truyền những field thay đổi)
   */
  function setFilter(newFilter: Partial<AccountFilter>) {
    filter.value = { ...filter.value, ...newFilter }
  }

  function clearAccounts() {
    accounts.value = []
    accountsIndex.clear()
    runProxyCache.clear()
    displayProxyCache.clear()
  }

  return {
    accounts,
    accountsIndex,
    loading,
    error,
    filter,
    detailAccount,
    showDetailPanel,
    total,
    stats,
    fetchAccounts,
    importAccounts,
    deleteAccounts,
    openDetail,
    closeDetail,
    setFilter,
    isVerifyRunning,
    setRunProxy,
    setDisplayProxy,
    clearRunProxyCache,
    removeAccount,
    clearAccounts,
    applySlotAssigned,
    ensureSlot,
  }
})
