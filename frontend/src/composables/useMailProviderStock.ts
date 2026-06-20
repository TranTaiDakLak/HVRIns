// useMailProviderStock.ts — Composable hợp nhất 4 pattern stock-check lặp lại
// trong InteractionSetupPage.vue (ZeusX, Mail30s, Store1s, DongVanFB).
// Dùng endpoint constants từ api-endpoints.ts để tránh hardcode.

import { ref, computed } from 'vue'
import type { Ref } from 'vue'
import { MAIL_PROVIDER_ENDPOINTS } from '@/constants/api-endpoints'
import { FetchStore1sProducts } from '../../wailsjs/go/app/App'
import type { ZeusXItem, Mail30sProduct, DvfbItem, VerifyConfig } from '@/types/interaction.types'

/**
 * Composable kiểm tra tồn kho của 4 nhà cung cấp mail (ZeusX, Mail30s, Store1s, DongVanFB).
 * Đọc API key và product ID từ `form` reactive ref, gọi API tương ứng khi check.
 *
 * @param form - Reactive ref chứa VerifyConfig — dùng để lấy API keys và product IDs
 *               của từng provider (zeusXApiKey, mail30sApiKey, store1sApiKey, dvfbApiKey, ...)
 */
export function useMailProviderStock(form: Ref<VerifyConfig>) {
  // ── ZeusX ──────────────────────────────────────────────────────────────────
  const zeusXStock = ref<ZeusXItem[]>([])
  const zeusXLoading = ref(false)
  const zeusXError = ref('')

  /** Item ZeusX đang được chọn (match theo form.zeusXAccountCode). null nếu chưa chọn. */
  const selectedZeusXStock = computed(() =>
    zeusXStock.value.find(i => i.AccountCode === form.value.zeusXAccountCode) ?? null
  )

  /**
   * Gọi ZeusX API để lấy danh sách account types còn trong kho.
   * Dùng endpoint từ MAIL_PROVIDER_ENDPOINTS.zeusX (không cần API key).
   */
  async function checkZeusXStock() {
    zeusXLoading.value = true
    zeusXError.value = ''
    zeusXStock.value = []
    try {
      const res = await fetch(MAIL_PROVIDER_ENDPOINTS.zeusX)
      const data = await res.json()
      if (data.Code === 0) zeusXStock.value = data.Data ?? []
      else zeusXError.value = `ZeusX error: ${data.Message ?? 'unknown'}`
    } catch (e) {
      zeusXError.value = 'Không kết nối được ZeusX: ' + String(e)
    } finally {
      zeusXLoading.value = false
    }
  }

  // ── Mail30s ─────────────────────────────────────────────────────────────────
  const mail30sProducts = ref<Mail30sProduct[]>([])
  const mail30sLoading = ref(false)
  const mail30sError = ref('')

  /** Product Mail30s đang được chọn (match theo form.mail30sProductSlug). */
  const selectedMail30sProduct = computed(() =>
    mail30sProducts.value.find(p => p.slug === form.value.mail30sProductSlug) ?? null
  )

  /**
   * Gọi Mail30s API để lấy danh sách sản phẩm còn hàng.
   * Dùng form.mail30sApiKey làm auth header.
   */
  async function checkMail30sStock() {
    mail30sLoading.value = true
    mail30sError.value = ''
    mail30sProducts.value = []
    try {
      const res = await fetch(MAIL_PROVIDER_ENDPOINTS.mail30sProducts(form.value.mail30sApiKey))
      const data = await res.json()
      if (data.success) mail30sProducts.value = data.data ?? []
      else mail30sError.value = data.message || 'Lỗi API Mail30s'
    } catch (e) {
      mail30sError.value = 'Không kết nối được Mail30s: ' + String(e)
    } finally {
      mail30sLoading.value = false
    }
  }

  // ── Store1s ──────────────────────────────────────────────────────────────────
  const store1sStockMap = ref<Record<string, number>>({})
  const store1sLoading = ref(false)
  const store1sError = ref('')

  /** Số lượng tồn kho của product đang chọn (form.store1sProductId). null nếu chưa load. */
  const selectedStore1sStock = computed(() => {
    const id = form.value.store1sProductId
    return id in store1sStockMap.value ? store1sStockMap.value[id] : null
  })

  /**
   * Gọi Store1s API để lấy map productId → số lượng tồn kho.
   * Dùng form.store1sApiKey làm auth. Kết quả flatten từ categories → products.
   */
  async function checkStore1sStock() {
    store1sLoading.value = true
    store1sError.value = ''
    store1sStockMap.value = {}
    try {
      // Gọi QUA BACKEND Go (FetchStore1sProducts) — webview fetch() trực tiếp store1s.com
      // bị CORS block ("Failed to fetch"). Backend không có CORS restriction.
      const raw = await FetchStore1sProducts(form.value.store1sApiKey)
      if (raw.startsWith('ERROR:')) {
        store1sError.value = raw.replace('ERROR:', '').trim()
        return
      }
      const products: { id: string; stock: number }[] = JSON.parse(raw)
      const map: Record<string, number> = {}
      for (const p of products) {
        map[p.id] = p.stock ?? 0
      }
      store1sStockMap.value = map
    } catch (e) {
      store1sError.value = 'Không kết nối được Store1s: ' + String(e)
    } finally {
      store1sLoading.value = false
    }
  }

  // ── DongVanFB ────────────────────────────────────────────────────────────────
  const dvfbStock = ref<DvfbItem[]>([])
  const dvfbLoading = ref(false)
  const dvfbError = ref('')

  /** Item DongVanFB đang được chọn (match theo form.dvfbAccountType). */
  const selectedDvfbStock = computed(() =>
    dvfbStock.value.find(i => String(i.id) === form.value.dvfbAccountType) ?? null
  )

  /**
   * Gọi DongVanFB API để lấy danh sách account types còn hàng.
   * Dùng form.dvfbApiKey làm auth.
   */
  async function checkDvfbStock() {
    dvfbLoading.value = true
    dvfbError.value = ''
    dvfbStock.value = []
    try {
      const res = await fetch(MAIL_PROVIDER_ENDPOINTS.dongvanfbAccountTypes(form.value.dvfbApiKey))
      const data = await res.json()
      if (data.status) dvfbStock.value = data.data ?? []
      else dvfbError.value = data.message || 'Lỗi API DongVanFB'
    } catch (e) {
      dvfbError.value = 'Không kết nối được DongVanFB: ' + String(e)
    } finally {
      dvfbLoading.value = false
    }
  }

  return {
    // ZeusX
    zeusXStock, zeusXLoading, zeusXError, selectedZeusXStock, checkZeusXStock,
    // Mail30s
    mail30sProducts, mail30sLoading, mail30sError, selectedMail30sProduct, checkMail30sStock,
    // Store1s
    store1sStockMap, store1sLoading, store1sError, selectedStore1sStock, checkStore1sStock,
    // DongVanFB
    dvfbStock, dvfbLoading, dvfbError, selectedDvfbStock, checkDvfbStock,
  }
}
