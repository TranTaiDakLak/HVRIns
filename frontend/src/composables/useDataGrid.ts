// useDataGrid.ts — Orchestrate sort, filter, virtual scroll cho DataGrid
// Bao gồm: sort toggle, keyword filter, visible range tính toán

import { ref, computed, type Ref } from 'vue'

/** Trạng thái sort hiện tại: column nào và chiều nào. */
export interface SortState {
  /** Key của column đang sort. Rỗng = không sort. */
  column: string
  /** Chiều sort. null = không sort. */
  direction: 'asc' | 'desc' | null
}

/** Tùy chọn khởi tạo cho useDataGrid. */
export interface DataGridOptions<T> {
  /** Dữ liệu gốc dạng reactive Ref — grid tự re-compute khi mảng này thay đổi. */
  items: Ref<T[]>
  /** Chiều cao mỗi row tính bằng px. Mặc định: 36. */
  rowHeight?: number
  /** Số row buffer render thêm trên/dưới viewport để tránh flicker khi scroll nhanh. Mặc định: 5. */
  bufferSize?: number
}

/**
 * Composable tổng hợp sort + filter + virtual scroll cho DataGrid.
 * Nhận dữ liệu gốc, trả về:
 *   - `visibleItems` — chỉ những rows cần render trong viewport
 *   - `totalHeight`  — chiều cao tổng để scrollbar đúng tỷ lệ
 *   - `offsetTop`    — khoảng cách từ đầu list đến row đầu tiên được render
 *
 * @param options - Cấu hình gồm dữ liệu, rowHeight, bufferSize
 */
export function useDataGrid<T extends Record<string, unknown>>(options: DataGridOptions<T>) {
  const { items, rowHeight = 36, bufferSize = 5 } = options

  // === Sort state ===
  const sort = ref<SortState>({ column: '', direction: null })

  // === Filter state ===
  const filterKeyword = ref('')

  // === Virtual scroll state ===
  const scrollTop = ref(0)
  const viewportHeight = ref(600)

  /**
   * Cycle sort cho 1 column: không sort → asc → desc → không sort.
   * @param column - Key của column cần toggle sort
   */
  function toggleSort(column: string) {
    if (sort.value.column !== column) {
      sort.value = { column, direction: 'asc' }
    } else if (sort.value.direction === 'asc') {
      sort.value = { column, direction: 'desc' }
    } else {
      sort.value = { column: '', direction: null }
    }
  }

  // Status keywords — khi nhập đúng tên status thì chỉ filter theo field status
  const STATUS_KEYWORDS = new Set(['unknown', 'live', 'die', 'new', 'checkpoint'])

  // Filtered items: nếu keyword là status keyword → filter chính xác theo status field
  // Ngược lại → search across tất cả fields như bình thường
  const filteredItems = computed(() => {
    let result = items.value

    if (filterKeyword.value) {
      const kw = filterKeyword.value.toLowerCase().trim()
      if (STATUS_KEYWORDS.has(kw)) {
        // Filter chính xác theo status — tránh match nhầm với activity/note
        result = result.filter(item =>
          String(item['status'] ?? '').toLowerCase() === kw
        )
      } else {
        result = result.filter(item =>
          Object.values(item).some(val =>
            String(val ?? '').toLowerCase().includes(kw)
          )
        )
      }
    }

    return result
  })

  // Sorted items
  const sortedItems = computed(() => {
    const data = [...filteredItems.value]
    const { column, direction } = sort.value

    if (!column || !direction) return data

    return data.sort((a, b) => {
      const va = String(a[column] ?? '')
      const vb = String(b[column] ?? '')
      const cmp = va.localeCompare(vb, undefined, { numeric: true })
      return direction === 'desc' ? -cmp : cmp
    })
  })

  // Virtual scroll: tính visible range
  const totalHeight = computed(() => sortedItems.value.length * rowHeight)

  const visibleRange = computed(() => {
    const startIdx = Math.max(0, Math.floor(scrollTop.value / rowHeight) - bufferSize)
    const endIdx = Math.min(
      sortedItems.value.length,
      Math.ceil((scrollTop.value + viewportHeight.value) / rowHeight) + bufferSize
    )
    return { startIdx, endIdx }
  })

  // Chỉ render rows trong visible range
  const visibleItems = computed(() => {
    const { startIdx, endIdx } = visibleRange.value
    return sortedItems.value.slice(startIdx, endIdx).map((item, idx) => ({
      item,
      index: startIdx + idx,
    }))
  })

  // Offset cho spacer phía trên (tạo vị trí scroll đúng)
  const offsetTop = computed(() => visibleRange.value.startIdx * rowHeight)

  /**
   * Cập nhật scrollTop từ scroll event của container.
   * Gọi trong `@scroll` handler của element chứa grid body.
   * @param event - Native scroll event
   */
  function onScroll(event: Event) {
    const target = event.target as HTMLElement
    scrollTop.value = target.scrollTop
  }

  /**
   * Cập nhật viewport height khi container resize.
   * Gọi từ ResizeObserver.
   * @param height - Chiều cao mới của viewport tính bằng px
   */
  function setViewportHeight(height: number) {
    viewportHeight.value = height
  }

  /** Xóa filter keyword, hiện lại toàn bộ rows. */
  function clearFilter() {
    filterKeyword.value = ''
  }

  /** Reset toàn bộ state: sort, filter, scroll về ban đầu. */
  function reset() {
    sort.value = { column: '', direction: null }
    filterKeyword.value = ''
    scrollTop.value = 0
  }

  return {
    // Sort
    sort,
    toggleSort,
    // Filter
    filterKeyword,
    filteredItems,
    clearFilter,
    // Sorted data
    sortedItems,
    // Virtual scroll
    totalHeight,
    visibleRange,
    visibleItems,
    offsetTop,
    scrollTop,
    onScroll,
    setViewportHeight,
    // General
    reset,
  }
}
