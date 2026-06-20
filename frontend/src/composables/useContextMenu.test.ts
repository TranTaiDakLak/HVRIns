import { useContextMenu } from '@/composables/useContextMenu'
import type { MenuItemDef } from '@/composables/useContextMenu'

const MENU_ITEMS: MenuItemDef[] = [
  { key: 'delete', label: 'Xoá' },
  { key: 'copy', label: 'Copy' },
  { key: 'export', label: 'Xuất' },
]

function makeEvent(clientX: number, clientY: number): MouseEvent {
  return { clientX, clientY } as MouseEvent
}

describe('useContextMenu — trạng thái khởi tạo', () => {
  it('visible=false, targetId=null, items=[], x=0, y=0', () => {
    const ctx = useContextMenu()
    expect(ctx.visible.value).toBe(false)
    expect(ctx.targetId.value).toBeNull()
    expect(ctx.items.value).toHaveLength(0)
    expect(ctx.x.value).toBe(0)
    expect(ctx.y.value).toBe(0)
  })
})

describe('useContextMenu — show', () => {
  beforeEach(() => {
    vi.stubGlobal('innerWidth', 1024)
    vi.stubGlobal('innerHeight', 768)
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('show: đặt visible=true, targetId, items', () => {
    const ctx = useContextMenu()
    ctx.show(makeEvent(100, 100), 42, MENU_ITEMS)
    expect(ctx.visible.value).toBe(true)
    expect(ctx.targetId.value).toBe(42)
    expect(ctx.items.value).toHaveLength(3)
  })

  it('show: không clamp khi menu vừa viewport', () => {
    const ctx = useContextMenu()
    ctx.show(makeEvent(100, 100), 1, MENU_ITEMS)
    // clientX=100, menuWidth=240 → 100+240=340 < 1024 → x=100 (không clamp)
    expect(ctx.x.value).toBe(100)
  })

  it('show: clamp x khi gần cạnh phải (clientX + menuWidth > vw)', () => {
    const ctx = useContextMenu()
    // clientX=900, menuWidth=240 → 900+240=1140 > 1024 → x=900-240=660
    ctx.show(makeEvent(900, 100), 1, MENU_ITEMS)
    expect(ctx.x.value).toBe(900 - 240)
  })

  it('show: clamp y khi gần cạnh dưới (clientY + menuHeight > vh)', () => {
    const ctx = useContextMenu()
    // 3 items × 32px = 96px; clientY=720, 720+96=816 > 768 → y = max(0, 768-96) = 672
    ctx.show(makeEvent(100, 720), 1, MENU_ITEMS)
    expect(ctx.y.value).toBe(768 - Math.min(MENU_ITEMS.length * 32, 500))
  })

  it('show: cập nhật items mới khi gọi lần 2', () => {
    const ctx = useContextMenu()
    ctx.show(makeEvent(100, 100), 1, MENU_ITEMS)
    const newItems: MenuItemDef[] = [{ key: 'only', label: 'Only' }]
    ctx.show(makeEvent(200, 200), 2, newItems)
    expect(ctx.items.value).toHaveLength(1)
    expect(ctx.targetId.value).toBe(2)
  })
})

describe('useContextMenu — hide', () => {
  beforeEach(() => {
    vi.stubGlobal('innerWidth', 1024)
    vi.stubGlobal('innerHeight', 768)
  })
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('hide: visible=false, targetId=null', () => {
    const ctx = useContextMenu()
    ctx.show(makeEvent(100, 100), 5, MENU_ITEMS)
    ctx.hide()
    expect(ctx.visible.value).toBe(false)
    expect(ctx.targetId.value).toBeNull()
  })

  it('hide: items giữ nguyên (để animate out)', () => {
    const ctx = useContextMenu()
    ctx.show(makeEvent(100, 100), 5, MENU_ITEMS)
    ctx.hide()
    expect(ctx.items.value).toHaveLength(3)
  })
})
