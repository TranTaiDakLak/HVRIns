// useContextMenu.ts — Composable quản lý context menu state

import { ref, readonly } from 'vue'

/** Định nghĩa 1 item trong context menu. */
export interface MenuItemDef {
  /** Key duy nhất để định danh item (dùng cho xử lý action). */
  key: string
  /** Nhãn hiển thị trong menu. */
  label: string
  /** Icon emoji hoặc ký tự hiển thị trước label. Tùy chọn. */
  icon?: string
  /** Hiển thị nhưng không click được — dùng khi tính năng chưa khả dụng. */
  disabled?: boolean
  /** Nếu true, render thành đường kẻ phân cách thay vì item. */
  separator?: boolean
  /** Submenu — danh sách items con khi hover vào item này. */
  children?: MenuItemDef[]
  /** Callback thực thi khi click item. */
  action?: () => void
}

/**
 * Composable quản lý trạng thái hiển thị context menu:
 * vị trí (x, y), row target, và danh sách items.
 * Tự động clamp vị trí để menu không bị tràn ra ngoài viewport.
 */
export function useContextMenu() {
  const visible = ref(false)
  const x = ref(0)
  const y = ref(0)
  const targetId = ref<number | null>(null)
  const items = ref<MenuItemDef[]>([])

  /**
   * Hiển thị context menu tại vị trí chuột, tự clamp vào viewport.
   * @param event     - MouseEvent từ `@contextmenu` handler để lấy tọa độ
   * @param id        - ID của row/item đang được right-click (dùng để xác định target khi xử lý action)
   * @param menuItems - Danh sách items cần hiển thị trong menu
   */
  function show(event: MouseEvent, id: number, menuItems: MenuItemDef[]) {
    // Viewport clamp
    const menuWidth = 240
    const menuHeight = Math.min(menuItems.length * 32, 500)
    const vw = window.innerWidth
    const vh = window.innerHeight

    x.value = event.clientX + menuWidth > vw ? event.clientX - menuWidth : event.clientX
    y.value = event.clientY + menuHeight > vh ? Math.max(0, vh - menuHeight) : event.clientY

    targetId.value = id
    items.value = menuItems
    visible.value = true
  }

  /** Ẩn context menu và reset targetId. */
  function hide() {
    visible.value = false
    targetId.value = null
  }

  return {
    visible: readonly(visible),
    x: readonly(x),
    y: readonly(y),
    /** ID của row đang được target bởi context menu. null nếu menu đang ẩn. */
    targetId: readonly(targetId),
    items: readonly(items),
    show,
    hide,
  }
}
