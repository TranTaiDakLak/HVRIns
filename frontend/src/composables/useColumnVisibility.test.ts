import { setActivePinia, createPinia } from 'pinia'
import { useColumnVisibility, type ColumnDef } from '@/composables/useColumnVisibility'

// Tập columns thu gọn cho test
const COLS: ColumnDef[] = [
  { key: 'uid', label: 'UID', defaultVisible: true },
  { key: 'status', label: 'Trạng thái', defaultVisible: true },
  { key: 'cookie', label: 'Cookie', defaultVisible: false },
]

describe('useColumnVisibility', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  it('isVisible: true cho cột defaultVisible=true', () => {
    const cv = useColumnVisibility(COLS)
    expect(cv.isVisible('uid')).toBe(true)
    expect(cv.isVisible('status')).toBe(true)
  })

  it('isVisible: false cho cột defaultVisible=false', () => {
    const cv = useColumnVisibility(COLS)
    expect(cv.isVisible('cookie')).toBe(false)
  })

  it('visibleColumns: chỉ trả về cột đang hiển thị', () => {
    const cv = useColumnVisibility(COLS)
    const keys = cv.visibleColumns.value.map(c => c.key)
    expect(keys).toContain('uid')
    expect(keys).toContain('status')
    expect(keys).not.toContain('cookie')
  })

  it('toggle: ẩn cột đang visible', () => {
    const cv = useColumnVisibility(COLS)
    cv.toggle('uid')
    expect(cv.isVisible('uid')).toBe(false)
    expect(cv.visibleColumns.value.map(c => c.key)).not.toContain('uid')
  })

  it('toggle: hiện cột đang ẩn', () => {
    const cv = useColumnVisibility(COLS)
    cv.toggle('cookie')
    expect(cv.isVisible('cookie')).toBe(true)
    expect(cv.visibleColumns.value.map(c => c.key)).toContain('cookie')
  })

  it('toggle 2 lần: về lại trạng thái gốc', () => {
    const cv = useColumnVisibility(COLS)
    cv.toggle('uid')
    cv.toggle('uid')
    expect(cv.isVisible('uid')).toBe(true)
  })

  it('resetToDefault: khôi phục cột về mặc định', () => {
    const cv = useColumnVisibility(COLS)
    cv.toggle('uid')    // ẩn uid
    cv.toggle('cookie') // hiện cookie
    cv.resetToDefault()
    // uid là defaultVisible=true → phải visible lại
    expect(cv.isVisible('uid')).toBe(true)
    // cookie là defaultVisible=false và không có trong DEFAULT_VISIBLE_COLUMN_KEYS → ẩn lại
    expect(cv.isVisible('cookie')).toBe(false)
  })

  it('allColumns: trả về toàn bộ columns được truyền vào', () => {
    const cv = useColumnVisibility(COLS)
    expect(cv.allColumns).toHaveLength(COLS.length)
    expect(cv.allColumns[0].key).toBe('uid')
  })
})
