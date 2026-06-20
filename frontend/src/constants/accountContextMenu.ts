// accountContextMenu.ts — Cấu trúc context menu cho Accounts grid
// Mapping đầy đủ từ WeBM frmFacebook.Core cmsDGV

import type { MenuItemDef } from '@/composables/useContextMenu'

export interface AccountMenuHandlers {
  selectAll: () => void
  selectHighlight: () => void
  selectByStatus: (status: string) => void
  deselectAll: () => void
  copyField: (field: string) => void
  copyFullImport: () => void
  deleteAccounts: () => void
  // Optional: copy các cell đã bôi (Excel-style) — chỉ hiển thị menu item nếu có.
  copyCells?: () => void
  cellsCount?: number
}

export function buildAccountContextMenu(h: AccountMenuHandlers): MenuItemDef[] {
  const items: MenuItemDef[] = []
  // Item đầu menu khi có cell đang select: "Sao chép ô đã bôi (Ctrl+C)"
  if (h.copyCells && (h.cellsCount ?? 0) > 0) {
    items.push({
      key: 'copy-cells',
      label: `Sao chép ô đã bôi (${h.cellsCount}) — Ctrl+C`,
      icon: '📋',
      action: h.copyCells,
    })
    items.push({ key: 'sep-cells', label: '', separator: true })
  }
  items.push(...[
    {
      key: 'select', label: 'Chọn', icon: '✓',
      children: [
        { key: 'select-all', label: 'Chọn tất cả', action: h.selectAll },
        { key: 'select-highlight', label: 'Chọn bôi đen', action: h.selectHighlight },
        { key: 'sep-sel', label: '', separator: true },
        {
          key: 'select-by-status', label: 'Theo trạng thái',
          children: [
            { key: 'sel-live', label: 'Live', action: () => h.selectByStatus('live') },
            { key: 'sel-die', label: 'Die', action: () => h.selectByStatus('die') },
            { key: 'sel-new', label: 'New', action: () => h.selectByStatus('new') },
            { key: 'sel-unknown', label: 'Unknown', action: () => h.selectByStatus('unknown') },
          ],
        },
      ],
    },
    { key: 'deselect-all', label: 'Bỏ chọn tất cả', icon: '☐', action: h.deselectAll },
    { key: 'sep1', label: '', separator: true },
    {
      key: 'copy', label: 'Sao chép', icon: '📋',
      children: [
        { key: 'copy-uid', label: 'UID', action: () => h.copyField('uid') },
        { key: 'copy-pass', label: 'Mật khẩu', action: () => h.copyField('password') },
        { key: 'copy-2fa', label: '2FA', action: () => h.copyField('twofa') },
        { key: 'copy-full', label: 'Full Import', action: h.copyFullImport },
        { key: 'copy-email', label: 'Email | Pass Email', action: () => h.copyField('email') },
        { key: 'copy-cookie', label: 'Cookie', action: () => h.copyField('cookie') },
        { key: 'copy-token', label: 'Token', action: () => h.copyField('token') },
        { key: 'sep-copy', label: '', separator: true },
        { key: 'copy-activity', label: 'Hoạt động', action: () => h.copyField('activity') },
        { key: 'copy-status', label: 'Trạng thái', action: () => h.copyField('status') },
        { key: 'copy-cp', label: 'Checkpoint', action: () => h.copyField('checkpoint') },
      ],
    },
    { key: 'delete', label: 'Xóa tài khoản', icon: '✕', action: h.deleteAccounts },
    {
      key: 'check', label: 'Kiểm tra tài khoản', icon: '✔',
      children: [
        { key: 'check-info', label: 'Kiểm tra thông tin', disabled: true },
        { key: 'check-live', label: 'Kiểm tra Live/Die', disabled: true },
        { key: 'check-wall', label: 'Kiểm tra tường', disabled: true },
      ],
    },
  ])
  return items
}
