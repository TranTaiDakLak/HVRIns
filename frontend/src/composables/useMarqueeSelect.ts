// useMarqueeSelect.ts — Composable: kéo chuột tạo khung chọn (marquee / rubber-band)
// để quét chọn nhiều phần tử cùng lúc, giống thao tác chọn file trong File Explorer.
//
// Cách dùng:
//   const m = useMarqueeSelect({ onCommit: (keys, mode) => ... })
//   <div :ref="m.setContainerEl" @mousedown="m.onMouseDown">
//     <button v-for="p in items" :data-pkey="p.key"
//             :class="{ 'is-preview': m.isPreviewed(p.key) }">...</button>
//     <div v-if="m.state.box.visible" class="marquee" :style="boxStyle" />
//   </div>
//
// - Kéo chuột trái: quét chọn (mode 'add'). Alt + kéo: bỏ chọn (mode 'remove').
// - Click đơn (không kéo quá threshold) không kích hoạt marquee — nút giữ nguyên @click.
// - Chuột phải không bị ảnh hưởng (dành cho @contextmenu).

import { reactive, onBeforeUnmount } from 'vue'
import type { ComponentPublicInstance } from 'vue'

export type MarqueeMode = 'add' | 'remove'

export interface MarqueeSelectOptions {
  /** Gọi khi kết thúc kéo (mouseup). `keys` = các phần tử nằm trong khung. */
  onCommit: (keys: string[], mode: MarqueeMode) => void
  /** CSS selector cho phần tử chọn được bên trong container. Mặc định `[data-pkey]`. */
  itemSelector?: string
  /** Tên attribute chứa key của phần tử. Mặc định `data-pkey`. */
  keyAttr?: string
  /** Số px tối thiểu phải kéo mới tính là drag (phân biệt với click). Mặc định 6. */
  threshold?: number
}

interface MarqueeState {
  /** Đang trong thao tác kéo (đã vượt threshold). */
  dragging: boolean
  /** Chế độ hiện tại: thêm hay bỏ chọn. */
  mode: MarqueeMode
  /** Hình chữ nhật overlay, toạ độ tương đối với container (px). */
  box: { visible: boolean; x: number; y: number; w: number; h: number }
  /** Các key đang nằm trong khung (preview realtime khi kéo). */
  previewKeys: Set<string>
}

export interface MarqueeSelectApi {
  state: MarqueeState
  setContainerEl: (el: Element | ComponentPublicInstance | null) => void
  onMouseDown: (e: MouseEvent) => void
  isPreviewed: (key: string) => boolean
}

export function useMarqueeSelect(options: MarqueeSelectOptions): MarqueeSelectApi {
  const itemSelector = options.itemSelector ?? '[data-pkey]'
  const keyAttr = options.keyAttr ?? 'data-pkey'
  const threshold = options.threshold ?? 6

  const state = reactive<MarqueeState>({
    dragging: false,
    mode: 'add',
    box: { visible: false, x: 0, y: 0, w: 0, h: 0 },
    previewKeys: new Set<string>(),
  })

  let container: HTMLElement | null = null
  let startX = 0
  let startY = 0
  let started = false // đã vượt threshold chưa

  function setContainerEl(el: Element | ComponentPublicInstance | null): void {
    // Chỉ nhận native element; nếu lỡ gắn ref vào component thì bỏ qua (tránh crash querySelectorAll).
    container = el instanceof HTMLElement ? el : null
  }

  /** Các phần tử (bỏ qua disabled) giao với hình chữ nhật viewport [left,top,right,bottom]. */
  function hitKeys(left: number, top: number, right: number, bottom: number): string[] {
    if (!container) return []
    const keys: string[] = []
    container.querySelectorAll<HTMLElement>(itemSelector).forEach((item) => {
      if ((item as HTMLButtonElement).disabled) return
      const r = item.getBoundingClientRect()
      // giao nhau khi 2 hình chữ nhật chồng lấn (kể cả chạm mép)
      const intersects = r.left < right && r.right > left && r.top < bottom && r.bottom > top
      if (!intersects) return
      const k = item.getAttribute(keyAttr)
      if (k) keys.push(k)
    })
    return keys
  }

  function updatePreview(keys: string[]): void {
    state.previewKeys.clear()
    for (const k of keys) state.previewKeys.add(k)
  }

  function rectFrom(e: MouseEvent): [number, number, number, number] {
    return [
      Math.min(startX, e.clientX),
      Math.min(startY, e.clientY),
      Math.max(startX, e.clientX),
      Math.max(startY, e.clientY),
    ]
  }

  function onMouseDown(e: MouseEvent): void {
    if (e.button !== 0 || !container) return // chỉ chuột trái; chuột phải = contextmenu
    startX = e.clientX
    startY = e.clientY
    started = false
    state.mode = e.altKey ? 'remove' : 'add'
    window.addEventListener('mousemove', onMouseMove)
    window.addEventListener('mouseup', onMouseUp)
  }

  function onMouseMove(e: MouseEvent): void {
    if (!container) return
    if (!started) {
      const dx = Math.abs(e.clientX - startX)
      const dy = Math.abs(e.clientY - startY)
      if (dx < threshold && dy < threshold) return
      started = true
      state.dragging = true
      state.box.visible = true
    }
    const [left, top, right, bottom] = rectFrom(e)
    const cr = container.getBoundingClientRect()
    state.box.x = left - cr.left
    state.box.y = top - cr.top
    state.box.w = right - left
    state.box.h = bottom - top
    updatePreview(hitKeys(left, top, right, bottom))
    e.preventDefault() // chặn bôi đen text khi kéo
  }

  function onMouseUp(e: MouseEvent): void {
    window.removeEventListener('mousemove', onMouseMove)
    window.removeEventListener('mouseup', onMouseUp)
    const wasDrag = started
    const mode = state.mode
    const keys = wasDrag ? hitKeys(...rectFrom(e)) : []
    // reset trạng thái hiển thị
    started = false
    state.dragging = false
    state.box.visible = false
    updatePreview([])
    if (wasDrag) {
      suppressNextClick()
      options.onCommit(keys, mode)
    }
  }

  /** Sau khi kéo xong, chặn đúng 1 click kế tiếp để không toggle nhầm nút dưới con trỏ. */
  function suppressNextClick(): void {
    const el = container
    if (!el) return
    const handler = (ev: MouseEvent): void => {
      ev.stopImmediatePropagation()
      ev.preventDefault()
      el.removeEventListener('click', handler, true)
    }
    el.addEventListener('click', handler, true)
    // Dọn bẫy nếu không có click nào theo sau (vd nhả chuột ngoài container). Double rAF
    // đảm bảo click (nếu có) luôn được dispatch xong trước bước dọn, kể cả khi onCommit
    // gây re-render xen vào giữa.
    window.requestAnimationFrame(() =>
      window.requestAnimationFrame(() => el.removeEventListener('click', handler, true)),
    )
  }

  function isPreviewed(key: string): boolean {
    return state.previewKeys.has(key)
  }

  onBeforeUnmount(() => {
    window.removeEventListener('mousemove', onMouseMove)
    window.removeEventListener('mouseup', onMouseUp)
  })

  return { state, setContainerEl, onMouseDown, isPreviewed }
}
