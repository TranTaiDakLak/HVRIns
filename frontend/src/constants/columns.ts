// columns.ts — Định nghĩa columns cho Accounts grid
// Mapping đầy đủ từ WeBM frmFacebook.Core DataGridView

import type { ColumnDef } from '../composables/useColumnVisibility'

export const ACCOUNT_COLUMNS: ColumnDef[] = [
  // --- Cột cơ bản ---
  { key: 'uid', label: 'UID', width: '130px', sortable: true, defaultVisible: true },
  { key: 'fullData', label: 'Dữ liệu gốc', width: '160px', defaultVisible: false },
  { key: 'password', label: 'Mật khẩu', width: '100px', defaultVisible: true },
  { key: 'twofa', label: '2FA', width: '70px', align: 'center', defaultVisible: false },
  // Cột EMAIL: hiển thị email HOẶC phone (tùy Mode reg) — không chia 2 cột nữa.
  { key: 'email', label: 'Email', width: '130px', sortable: true, defaultVisible: true },
  { key: 'passMail', label: 'Pass Email', width: '120px', defaultVisible: false },
  { key: 'mailRecovery', label: 'Mail khôi phục', width: '160px', defaultVisible: false },
  { key: 'cookie', label: 'Cookie', width: '90px', defaultVisible: false },
  { key: 'token', label: 'Token', width: '120px', defaultVisible: false },
  { key: 'status', label: 'Trạng thái', width: '90px', align: 'center', sortable: true, defaultVisible: true },
  { key: 'checkpoint', label: 'Checkpoint', width: '100px', align: 'center', defaultVisible: false },
  { key: 'statusAds', label: 'Quảng cáo', width: '100px', defaultVisible: false },
  { key: 'bm', label: 'BM', width: '80px', defaultVisible: false },
  { key: 'tkqc', label: 'TKQC', width: '80px', defaultVisible: false },
  { key: 'chatSupport', label: 'Chat Support', width: '100px', defaultVisible: false },

  // --- Khác ---
  { key: 'avatar', label: 'Avatar', width: '80px', defaultVisible: false },
  { key: 'cover', label: 'Cover', width: '80px', defaultVisible: false },
  { key: 'phone', label: 'Phone', width: '120px', defaultVisible: false }, // deprecated — gộp vào cột EMAIL
  { key: 'proxy', label: 'Proxy', width: '120px', defaultVisible: true },
  { key: 'userAgent', label: 'UA', width: '300px', defaultVisible: false },
  { key: 'note', label: 'Ghi chú', width: '150px', defaultVisible: false },
  { key: 'noteRun', label: 'Ghi chú chạy', width: '150px', defaultVisible: false },
  { key: 'importTime', label: 'Ngày nhập', width: '150px', sortable: true, defaultVisible: false },
  { key: 'category', label: 'Thư mục', width: '100px', sortable: true, defaultVisible: false },
  { key: 'lastRun', label: 'Chạy lần cuối', width: '140px', sortable: true, defaultVisible: false },
  { key: 'runProxy', label: 'IP chạy', width: '150px', defaultVisible: true },
  { key: 'activity', label: 'Hoạt động', width: 'auto', defaultVisible: true },
]

// Columns mặc định hiển thị (dùng cho reset)
export const DEFAULT_VISIBLE_COLUMN_KEYS = ACCOUNT_COLUMNS
  .filter(c => c.defaultVisible !== false)
  .map(c => c.key)
