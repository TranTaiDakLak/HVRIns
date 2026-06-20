import { ref } from 'vue'
import { useDataGrid } from '@/composables/useDataGrid'

type Row = { id: number; name: string; status: string }

function makeGrid(rows: Row[], opts?: { rowHeight?: number; bufferSize?: number }) {
  return useDataGrid<Row>({ items: ref(rows), ...opts })
}

describe('useDataGrid — sort', () => {
  it('initial: no sort', () => {
    const g = makeGrid([])
    expect(g.sort.value).toEqual({ column: '', direction: null })
  })

  it('toggleSort: null → asc → desc → null cycle', () => {
    const g = makeGrid([])
    g.toggleSort('name')
    expect(g.sort.value).toEqual({ column: 'name', direction: 'asc' })
    g.toggleSort('name')
    expect(g.sort.value).toEqual({ column: 'name', direction: 'desc' })
    g.toggleSort('name')
    expect(g.sort.value).toEqual({ column: '', direction: null })
  })

  it('toggleSort different column resets direction to asc', () => {
    const g = makeGrid([])
    g.toggleSort('name')
    g.toggleSort('name') // → desc
    g.toggleSort('status') // new column → asc
    expect(g.sort.value).toEqual({ column: 'status', direction: 'asc' })
  })

  it('sortedItems: asc by name', () => {
    const rows: Row[] = [
      { id: 2, name: 'charlie', status: 'live' },
      { id: 1, name: 'alice', status: 'die' },
      { id: 3, name: 'bob', status: 'live' },
    ]
    const g = makeGrid(rows)
    g.toggleSort('name')
    expect(g.sortedItems.value.map(r => r.name)).toEqual(['alice', 'bob', 'charlie'])
  })

  it('sortedItems: desc by name', () => {
    const rows: Row[] = [
      { id: 1, name: 'alice', status: 'live' },
      { id: 2, name: 'charlie', status: 'die' },
    ]
    const g = makeGrid(rows)
    g.toggleSort('name')
    g.toggleSort('name') // → desc
    expect(g.sortedItems.value.map(r => r.name)).toEqual(['charlie', 'alice'])
  })
})

describe('useDataGrid — filter', () => {
  const rows: Row[] = [
    { id: 1, name: 'alice', status: 'live' },
    { id: 2, name: 'bob', status: 'die' },
    { id: 3, name: 'carol', status: 'live' },
  ]

  it('no filter: all rows visible', () => {
    const g = makeGrid(rows)
    expect(g.filteredItems.value).toHaveLength(3)
  })

  it('general filter: substring match across fields', () => {
    const g = makeGrid(rows)
    g.filterKeyword.value = 'ali'
    expect(g.filteredItems.value).toHaveLength(1)
    expect(g.filteredItems.value[0].name).toBe('alice')
  })

  it('status keyword: exact match on status field only', () => {
    const g = makeGrid(rows)
    g.filterKeyword.value = 'live'
    expect(g.filteredItems.value).toHaveLength(2)
    expect(g.filteredItems.value.every(r => r.status === 'live')).toBe(true)
  })

  it('clearFilter: resets keyword và hiện all rows', () => {
    const g = makeGrid(rows)
    g.filterKeyword.value = 'bob'
    g.clearFilter()
    expect(g.filterKeyword.value).toBe('')
    expect(g.filteredItems.value).toHaveLength(3)
  })
})

describe('useDataGrid — virtual scroll', () => {
  it('totalHeight = items.length × rowHeight', () => {
    const rows = Array.from({ length: 100 }, (_, i) => ({ id: i, name: String(i), status: 'live' }))
    const g = makeGrid(rows, { rowHeight: 36 })
    expect(g.totalHeight.value).toBe(3600)
  })

  it('visibleItems: chỉ rows trong viewport (không buffer)', () => {
    const rows = Array.from({ length: 100 }, (_, i) => ({ id: i, name: String(i), status: 'live' }))
    const g = makeGrid(rows, { rowHeight: 36, bufferSize: 0 })
    g.setViewportHeight(360) // 10 rows × 36px
    expect(g.visibleItems.value).toHaveLength(10)
    expect(g.visibleItems.value[0].item.id).toBe(0)
  })

  it('offsetTop = startIdx × rowHeight sau khi scroll', () => {
    const rows = Array.from({ length: 100 }, (_, i) => ({ id: i, name: String(i), status: 'live' }))
    const g = makeGrid(rows, { rowHeight: 36, bufferSize: 0 })
    g.setViewportHeight(360)
    g.scrollTop.value = 720 // row 20 đầu tiên
    expect(g.offsetTop.value).toBe(720) // startIdx=20, 20×36=720
  })

  it('onScroll: cập nhật scrollTop từ event', () => {
    const g = makeGrid([])
    const fakeEl = { scrollTop: 500 } as HTMLElement
    g.onScroll({ target: fakeEl } as Event)
    expect(g.scrollTop.value).toBe(500)
  })

  it('setViewportHeight: viewport lớn hơn → nhiều rows hơn', () => {
    const rows = Array.from({ length: 50 }, (_, i) => ({ id: i, name: String(i), status: 'live' }))
    const g = makeGrid(rows, { rowHeight: 36, bufferSize: 0 })
    g.setViewportHeight(108) // 3 rows
    const small = g.visibleItems.value.length
    g.setViewportHeight(360) // 10 rows
    expect(g.visibleItems.value.length).toBeGreaterThan(small)
  })
})

describe('useDataGrid — reset', () => {
  it('reset: xoá sort, filter, scroll về 0', () => {
    const g = makeGrid([{ id: 1, name: 'a', status: 'live' }])
    g.toggleSort('name')
    g.filterKeyword.value = 'test'
    g.scrollTop.value = 200
    g.reset()
    expect(g.sort.value).toEqual({ column: '', direction: null })
    expect(g.filterKeyword.value).toBe('')
    expect(g.scrollTop.value).toBe(0)
  })
})
