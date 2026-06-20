// useClipboard.ts — Copy data vào clipboard
// Dùng cho context menu "Sao chép" submenu

import type { Account } from '@/services/contracts'
import { useAppStore } from '@/stores/app.store'

/**
 * Composable cung cấp các hàm copy dữ liệu account vào clipboard.
 * Tự động hiển thị toast thành công/lỗi qua appStore.notify().
 */
export function useClipboard() {
  const appStore = useAppStore()

  /**
   * Copy bất kỳ chuỗi text vào clipboard.
   * @param text  - Nội dung cần copy (có thể nhiều dòng)
   * @param label - Tên hiển thị trong toast thành công, VD: 'uid', 'cookie'
   */
  async function copyText(text: string, label?: string) {
    try {
      await navigator.clipboard.writeText(text)
      appStore.notify('success', `Đã copy ${label || 'dữ liệu'} (${text.split('\n').length} dòng)`)
    } catch {
      appStore.notify('error', 'Không thể copy vào clipboard')
    }
  }

  /**
   * Copy giá trị của 1 field từ các accounts đang được chọn (selectedIds).
   * Mỗi account 1 dòng. Bỏ qua accounts có field rỗng.
   * @param accounts    - Toàn bộ danh sách accounts
   * @param selectedIds - Set ID của các accounts đang được check
   * @param field       - Field name cần copy (VD: 'uid', 'cookie', 'email')
   */
  function copyField(accounts: Account[], selectedIds: Set<number>, field: keyof Account) {
    const lines = accounts
      .filter(a => selectedIds.has(a.id))
      .map(a => String(a[field] ?? ''))
      .filter(v => v)
    if (lines.length === 0) {
      appStore.notify('warning', 'Không có dữ liệu để copy')
      return
    }
    copyText(lines.join('\n'), field)
  }

  /**
   * Copy full import format cho các accounts đang được chọn.
   * Format mỗi dòng: `uid|password|2fa|cookie|token|email|passMail`
   * @param accounts    - Toàn bộ danh sách accounts
   * @param selectedIds - Set ID của các accounts đang được check
   */
  function copyFullImport(accounts: Account[], selectedIds: Set<number>) {
    const lines = accounts
      .filter(a => selectedIds.has(a.id))
      .map(a => [a.uid, a.password, a.twofa, a.cookie, a.token, a.email, a.passMail].join('|'))
    if (lines.length === 0) {
      appStore.notify('warning', 'Không có dữ liệu để copy')
      return
    }
    copyText(lines.join('\n'), 'full import')
  }

  return { copyText, copyField, copyFullImport }
}
