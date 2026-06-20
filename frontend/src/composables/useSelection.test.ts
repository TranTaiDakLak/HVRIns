// useSelection.test.ts — Unit tests cho composable 2-state selection
// Không phụ thuộc Pinia, không phụ thuộc service layer.

import { useSelection } from '@/composables/useSelection'

type Item = { id: number }

const items: Item[] = [{ id: 1 }, { id: 2 }, { id: 3 }]

describe('useSelection — highlighted', () => {
  it('khởi tạo: rỗng', () => {
    const s = useSelection<Item>()
    expect(s.highlightedIds.value.size).toBe(0)
    expect(s.highlightedCount.value).toBe(0)
  })

  it('highlightOne: chỉ highlight 1 row, bỏ row trước', () => {
    const s = useSelection<Item>()
    s.highlightOne(1)
    expect(s.highlightedIds.value.has(1)).toBe(true)
    s.highlightOne(2)
    expect(s.highlightedIds.value.has(1)).toBe(false)
    expect(s.highlightedIds.value.has(2)).toBe(true)
    expect(s.highlightedCount.value).toBe(1)
  })

  it('highlightToggle: thêm rồi bỏ, không ảnh hưởng row khác', () => {
    const s = useSelection<Item>()
    s.highlightOne(1)
    s.highlightToggle(2)
    expect(s.highlightedIds.value.has(1)).toBe(true)
    expect(s.highlightedIds.value.has(2)).toBe(true)
    s.highlightToggle(2)
    expect(s.highlightedIds.value.has(2)).toBe(false)
  })

  it('highlightRange: chọn dải liên tiếp', () => {
    const s = useSelection<Item>()
    s.highlightOne(1)
    s.highlightRange(3, items)
    expect(s.highlightedIds.value.has(1)).toBe(true)
    expect(s.highlightedIds.value.has(2)).toBe(true)
    expect(s.highlightedIds.value.has(3)).toBe(true)
  })

  it('highlightRange ngược: từ id cao về id thấp', () => {
    const s = useSelection<Item>()
    s.highlightOne(3)
    s.highlightRange(1, items)
    expect(s.highlightedIds.value.size).toBe(3)
  })

  it('highlightAll: highlight toàn bộ', () => {
    const s = useSelection<Item>()
    s.highlightAll(items)
    expect(s.highlightedCount.value).toBe(3)
  })

  it('highlightClear: xóa sạch và reset lastClickedId', () => {
    const s = useSelection<Item>()
    s.highlightAll(items)
    s.highlightClear()
    expect(s.highlightedCount.value).toBe(0)
    // Sau clear, highlightRange không có lastClickedId → fallback highlightOne
    s.highlightRange(2, items)
    expect(s.highlightedIds.value.size).toBe(1)
    expect(s.highlightedIds.value.has(2)).toBe(true)
  })

  it('highlightByFilter: chỉ highlight rows thỏa predicate', () => {
    const s = useSelection<Item>()
    s.highlightByFilter(items, item => item.id > 1)
    expect(s.highlightedIds.value.has(1)).toBe(false)
    expect(s.highlightedIds.value.has(2)).toBe(true)
    expect(s.highlightedIds.value.has(3)).toBe(true)
  })
})

describe('useSelection — checked', () => {
  it('khởi tạo: rỗng', () => {
    const s = useSelection<Item>()
    expect(s.checkedCount.value).toBe(0)
  })

  it('toggleCheck: tích rồi bỏ', () => {
    const s = useSelection<Item>()
    s.toggleCheck(1)
    expect(s.checkedIds.value.has(1)).toBe(true)
    s.toggleCheck(1)
    expect(s.checkedIds.value.has(1)).toBe(false)
  })

  it('checkAll / uncheckAll', () => {
    const s = useSelection<Item>()
    s.checkAll(items)
    expect(s.checkedCount.value).toBe(3)
    s.uncheckAll()
    expect(s.checkedCount.value).toBe(0)
  })

  it('toggleCheckAll: tất cả đã tích → bỏ; chưa hết → tích hết', () => {
    const s = useSelection<Item>()
    s.toggleCheckAll(items)
    expect(s.checkedCount.value).toBe(3)
    s.toggleCheckAll(items)
    expect(s.checkedCount.value).toBe(0)
  })

  it('checkHighlighted: chỉ check highlighted rows', () => {
    const s = useSelection<Item>()
    s.highlightOne(2)
    s.checkHighlighted()
    expect(s.checkedIds.value.has(2)).toBe(true)
    expect(s.checkedIds.value.has(1)).toBe(false)
  })

  it('toggleCheckHighlighted: đã check → bỏ; chưa check → check', () => {
    const s = useSelection<Item>()
    s.highlightAll(items)
    s.toggleCheck(1) // pre-check id 1
    s.toggleCheckHighlighted()
    // id 1 đã check → bỏ
    expect(s.checkedIds.value.has(1)).toBe(false)
    // id 2,3 chưa check → check
    expect(s.checkedIds.value.has(2)).toBe(true)
    expect(s.checkedIds.value.has(3)).toBe(true)
  })

  it('checkByFilter: check theo predicate', () => {
    const s = useSelection<Item>()
    s.checkByFilter(items, item => item.id !== 2)
    expect(s.checkedIds.value.has(1)).toBe(true)
    expect(s.checkedIds.value.has(2)).toBe(false)
    expect(s.checkedIds.value.has(3)).toBe(true)
  })

  it('getCheckedIds: trả mảng đúng', () => {
    const s = useSelection<Item>()
    s.toggleCheck(1)
    s.toggleCheck(3)
    const ids = s.getCheckedIds()
    expect(ids).toContain(1)
    expect(ids).toContain(3)
    expect(ids).not.toContain(2)
  })

  it('selectedIds / selectedCount là alias của checkedIds / checkedCount', () => {
    const s = useSelection<Item>()
    s.toggleCheck(2)
    expect(s.selectedIds.value.has(2)).toBe(true)
    expect(s.selectedCount.value).toBe(1)
  })
})
