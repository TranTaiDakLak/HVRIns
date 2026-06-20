// useColumnVisibility.ts — Quản lý hiển thị cột cho DataGrid
// Consume preferences store, expose toggle/check helpers

import { computed } from 'vue'
import { usePreferencesStore } from '@/stores/preferences.store'

/** Định nghĩa 1 column trong DataGrid. */
export interface ColumnDef {
  /** Key duy nhất — dùng để map với field trong Account object và lưu vào preferences. */
  key: string
  /** Tên hiển thị trong header. */
  label: string
  /** Chiều rộng cột, VD: '120px', 'auto'. Mặc định: 'auto'. */
  width?: string
  /** Cho phép click header để sort không. Mặc định: true. */
  sortable?: boolean
  /** Căn lề nội dung cell. Mặc định: 'left'. */
  align?: 'left' | 'center' | 'right'
  /** Hiển thị mặc định khi chưa có preferences. Mặc định: true. */
  defaultVisible?: boolean
}

/**
 * Composable quản lý việc ẩn/hiện columns trong DataGrid.
 * Trạng thái lưu vào preferences store (persist localStorage).
 *
 * @param allColumns - Danh sách đầy đủ tất cả columns có thể hiển thị
 * @returns visibleColumns (computed), toggle, isVisible, resetToDefault
 */
export function useColumnVisibility(allColumns: ColumnDef[]) {
  const prefs = usePreferencesStore()

  /** Danh sách columns đang được hiển thị (reactive computed). */
  const visibleColumns = computed(() =>
    allColumns.filter(col => prefs.isColumnVisible(col.key))
  )

  /**
   * Bật/tắt hiển thị 1 column. Trạng thái được persist vào localStorage.
   * @param columnKey - Key của column cần toggle
   */
  function toggle(columnKey: string) {
    prefs.toggleColumn(columnKey)
  }

  /**
   * Kiểm tra column có đang hiển thị không.
   * @param columnKey - Key của column cần kiểm tra
   * @returns true nếu column đang visible
   */
  function isVisible(columnKey: string): boolean {
    return prefs.isColumnVisible(columnKey)
  }

  /** Reset danh sách columns về trạng thái mặc định (theo defaultVisible trong ColumnDef). */
  function resetToDefault() {
    prefs.resetColumns()
  }

  return {
    allColumns,
    visibleColumns,
    toggle,
    isVisible,
    resetToDefault,
  }
}
