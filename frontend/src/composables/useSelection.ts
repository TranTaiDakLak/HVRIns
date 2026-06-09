// useSelection.ts — Logic chọn rows cho data grid
// 2 trạng thái riêng biệt (giống WeBM DataGridView):
//   - highlighted (bôi đen): click/drag/Shift+Click → dùng cho copy, context menu
//   - checked (chọn/cChose): checkbox toggle → dùng cho Run, Delete

import { ref, computed } from 'vue'

/**
 * Quản lý 2 trạng thái chọn dòng của data grid: highlighted và checked.
 * Generic — T phải có field `id: number` để nhận dạng duy nhất mỗi row.
 */
export function useSelection<T extends { id: number }>() {
  // === HIGHLIGHTED (bôi đen) ===
  // Click row, drag, Shift+Click → highlight xanh
  const highlightedIds = ref<Set<number>>(new Set())
  const lastClickedId = ref<number | null>(null)

  const highlightedCount = computed(() => highlightedIds.value.size)

  /** Chỉ highlight đúng 1 row, bỏ highlight tất cả row khác. */
  function highlightOne(id: number) {
    highlightedIds.value = new Set([id])
    lastClickedId.value = id
  }

  /** Toggle highlight cho 1 row (Ctrl+Click). Không ảnh hưởng các row khác. */
  function highlightToggle(id: number) {
    const next = new Set(highlightedIds.value)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    highlightedIds.value = next
    lastClickedId.value = id
  }

  /**
   * Highlight dải từ lastClickedId đến id (Shift+Click).
   * @param id       - ID của row cuối dải cần highlight
   * @param allItems - Toàn bộ danh sách items theo thứ tự hiển thị (để tính range)
   */
  function highlightRange(id: number, allItems: T[]) {
    if (lastClickedId.value === null) { highlightOne(id); return }
    const ids = allItems.map(item => item.id)
    const startIdx = ids.indexOf(lastClickedId.value)
    const endIdx = ids.indexOf(id)
    if (startIdx === -1 || endIdx === -1) { highlightOne(id); return }
    const from = Math.min(startIdx, endIdx)
    const to = Math.max(startIdx, endIdx)
    const next = new Set(highlightedIds.value)
    for (let i = from; i <= to; i++) next.add(ids[i])
    highlightedIds.value = next
    lastClickedId.value = id
  }

  /**
   * Xử lý click vào row có modifier key:
   *   - Shift+Click → highlight range từ last click đến row này
   *   - Ctrl/Cmd+Click → toggle highlight row này
   *   - Click thường → chỉ highlight mình row này
   * @param id       - ID của row được click
   * @param event    - Mouse hoặc Keyboard event (để đọc shiftKey/ctrlKey)
   * @param allItems - Toàn bộ items hiển thị (dùng cho range selection)
   */
  function handleRowClick(id: number, event: MouseEvent | KeyboardEvent, allItems: T[]) {
    if (event.shiftKey) highlightRange(id, allItems)
    else if (event.ctrlKey || event.metaKey) highlightToggle(id)
    else highlightOne(id)
  }

  /** Highlight tất cả rows trong danh sách. */
  function highlightAll(allItems: T[]) {
    highlightedIds.value = new Set(allItems.map(item => item.id))
  }

  /** Xóa toàn bộ highlight và reset lastClickedId. */
  function highlightClear() {
    highlightedIds.value = new Set()
    lastClickedId.value = null
  }

  /**
   * Highlight những rows thỏa điều kiện lọc.
   * @param allItems  - Danh sách tất cả items
   * @param predicate - Hàm kiểm tra, trả về true nếu row cần được highlight
   */
  function highlightByFilter(allItems: T[], predicate: (item: T) => boolean) {
    highlightedIds.value = new Set(allItems.filter(predicate).map(item => item.id))
  }

  /**
   * Gán trực tiếp Set ID được highlight (dùng cho drag-select).
   * @param ids - Set ID mới thay thế toàn bộ highlighted state
   */
  function setHighlighted(ids: Set<number>) {
    highlightedIds.value = ids
  }

  // === CHECKED (chọn checkbox / cChose) ===
  // Toggle checkbox, Space key, context menu "Chọn" → dùng cho Run/Delete
  const checkedIds = ref<Set<number>>(new Set())
  const checkedCount = computed(() => checkedIds.value.size)

  /** Toggle trạng thái checked của 1 row. */
  function toggleCheck(id: number) {
    const next = new Set(checkedIds.value)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    checkedIds.value = next
  }

  /** Check (tích checkbox) tất cả rows trong danh sách. */
  function checkAll(allItems: T[]) {
    checkedIds.value = new Set(allItems.map(item => item.id))
  }

  /** Bỏ tích tất cả checkboxes. */
  function uncheckAll() {
    checkedIds.value = new Set()
  }

  /**
   * Nếu tất cả đều đã tích → bỏ tích hết. Ngược lại → tích hết.
   * @param allItems - Toàn bộ items để so sánh số lượng
   */
  function toggleCheckAll(allItems: T[]) {
    if (checkedIds.value.size === allItems.length && allItems.length > 0) uncheckAll()
    else checkAll(allItems)
  }

  // "Chọn bôi đen" — check all highlighted rows (WeBM SelectHighline, context menu)
  /** Tích checkbox cho tất cả rows đang được highlight (bôi đen). */
  function checkHighlighted() {
    const next = new Set(checkedIds.value)
    for (const id of highlightedIds.value) next.add(id)
    checkedIds.value = next
  }

  // Toggle check cho highlighted rows (WeBM ToggleCheck, Space key)
  // Dòng đã tích → bỏ tích, dòng chưa tích → tích
  /** Toggle checkbox cho tất cả rows đang highlight: đã tích → bỏ, chưa tích → tích. */
  function toggleCheckHighlighted() {
    const next = new Set(checkedIds.value)
    for (const id of highlightedIds.value) {
      if (next.has(id)) next.delete(id)
      else next.add(id)
    }
    checkedIds.value = next
  }

  /**
   * Tích checkbox cho những rows thỏa điều kiện lọc (WeBM SelectbyStatus).
   * @param allItems  - Danh sách tất cả items
   * @param predicate - Hàm kiểm tra, trả về true nếu row cần được check
   */
  function checkByFilter(allItems: T[], predicate: (item: T) => boolean) {
    const next = new Set(checkedIds.value)
    for (const item of allItems.filter(predicate)) next.add(item.id)
    checkedIds.value = next
  }

  /** Trả về mảng ID của các rows đang được tích checkbox. */
  function getCheckedIds(): number[] {
    return Array.from(checkedIds.value)
  }

  // === COMPAT ===
  // selectedIds = checkedIds for backward compat (toolbar, delete, etc.)
  const selectedIds = checkedIds
  const selectedCount = checkedCount

  return {
    // Highlighted (bôi đen)
    highlightedIds,
    highlightedCount,
    highlightOne,
    highlightToggle,
    highlightRange,
    highlightAll,
    highlightClear,
    highlightByFilter,
    setHighlighted,
    handleRowClick,

    // Checked (chọn)
    checkedIds,
    checkedCount,
    toggleCheck,
    checkAll,
    uncheckAll,
    toggleCheckAll,
    checkHighlighted,
    toggleCheckHighlighted,
    checkByFilter,
    getCheckedIds,

    // Compat aliases
    selectedIds,
    selectedCount,
    selectAll: checkAll,
    deselectAll: uncheckAll,
    toggleSelectAll: toggleCheckAll,
    getSelectedIds: getCheckedIds,
    selectByFilter: checkByFilter,
  }
}
