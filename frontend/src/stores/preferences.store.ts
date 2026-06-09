// preferences.store.ts — User preferences (persist vào localStorage)
// Theme, density, column visibility, masking, filter presets

import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export type ThemeMode = 'dark' | 'light'
export type DensityMode = 'compact' | 'default' | 'comfortable'

import { DEFAULT_VISIBLE_COLUMN_KEYS } from '../constants/columns'

// Danh sách columns mặc định hiển thị (từ columns.ts defaultVisible)
const DEFAULT_VISIBLE_COLUMNS = DEFAULT_VISIBLE_COLUMN_KEYS

/**
 * Store lưu preferences của người dùng (persist localStorage).
 * Bao gồm: theme, density, column visibility, data masking, last route.
 * Tất cả thay đổi đều tự động ghi vào localStorage qua watcher.
 */
export const usePreferencesStore = defineStore('preferences', () => {
  // === State (tất cả đều persist) ===
  const theme = ref<ThemeMode>((localStorage.getItem('theme') as ThemeMode) || 'dark')
  const density = ref<DensityMode>((localStorage.getItem('density') as DensityMode) || 'default')
  const dataMasking = ref(localStorage.getItem('data_masking') === 'true')
  // Load cached columns — dùng thẳng cache, không merge với defaults.
  // Merge sẽ thêm lại cột user đã bỏ chọn (vì cột default không có trong cache → bị coi là "mới").
  const cachedCols: string[] | null = JSON.parse(localStorage.getItem('visible_columns_v2') || 'null')
  const visibleColumns = ref<string[]>(cachedCols ?? [...DEFAULT_VISIBLE_COLUMNS])
  const lastRoute = ref(localStorage.getItem('last_route') || '/accounts')

  // === Watch & Persist ===
  watch(theme, (v) => {
    localStorage.setItem('theme', v)
    document.documentElement.setAttribute('data-theme', v)
  }, { immediate: true })

  watch(density, (v) => {
    localStorage.setItem('density', v)
    document.documentElement.setAttribute('data-density', v)
  }, { immediate: true })

  watch(dataMasking, (v) => {
    localStorage.setItem('data_masking', String(v))
  })

  watch(visibleColumns, (v) => {
    try { localStorage.setItem('visible_columns_v2', JSON.stringify(v)) } catch { /* localStorage full */ }
  }, { deep: true })

  watch(lastRoute, (v) => {
    localStorage.setItem('last_route', v)
  })

  // === Actions ===

  /** Chuyển đổi qua lại giữa dark và light theme. Áp dụng ngay lên `data-theme` của html. */
  function toggleTheme() {
    theme.value = theme.value === 'dark' ? 'light' : 'dark'
  }

  /**
   * Đặt mật độ hiển thị grid. Áp dụng ngay lên `data-density` của html.
   * @param mode - 'compact' (28px/row), 'default' (36px/row), 'comfortable' (44px/row)
   */
  function setDensity(mode: DensityMode) {
    density.value = mode
  }

  /**
   * Bật/tắt data masking — che bớt uid/cookie/token trong grid để chụp màn hình an toàn.
   */
  function toggleMasking() {
    dataMasking.value = !dataMasking.value
  }

  /**
   * Bật/tắt hiển thị 1 column trong DataGrid.
   * @param columnKey - Key của column (phải khớp với ColumnDef.key)
   */
  function toggleColumn(columnKey: string) {
    const idx = visibleColumns.value.indexOf(columnKey)
    if (idx >= 0) {
      visibleColumns.value.splice(idx, 1)
    } else {
      visibleColumns.value.push(columnKey)
    }
  }

  /**
   * Kiểm tra column có đang hiển thị không.
   * @param columnKey - Key của column cần kiểm tra
   * @returns true nếu column đang trong danh sách visible
   */
  function isColumnVisible(columnKey: string): boolean {
    return visibleColumns.value.includes(columnKey)
  }

  /** Reset danh sách columns về mặc định (DEFAULT_VISIBLE_COLUMN_KEYS). */
  function resetColumns() {
    visibleColumns.value = [...DEFAULT_VISIBLE_COLUMNS]
  }

  /**
   * Lưu route cuối cùng đã xem để restore khi mở lại app.
   * @param route - Path của route, VD: '/accounts', '/general-settings'
   */
  function setLastRoute(route: string) {
    lastRoute.value = route
  }

  return {
    theme,
    density,
    dataMasking,
    visibleColumns,
    lastRoute,
    toggleTheme,
    setDensity,
    toggleMasking,
    toggleColumn,
    isColumnVisible,
    resetColumns,
    setLastRoute,
  }
})
