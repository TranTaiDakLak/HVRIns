// app.store.ts — Global app runtime state
// Sidebar, notifications, connection status

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

/** Một thông báo toast hiển thị trên UI. */
export interface AppNotification {
  /** ID tự tăng — dùng để remove đúng notification. */
  id: number
  /** Loại: success (xanh), error (đỏ), warning (vàng), info (xanh dương). */
  type: 'success' | 'error' | 'warning' | 'info'
  /** Nội dung thông báo hiển thị cho người dùng. */
  message: string
  /** Thời gian tự động ẩn (ms). 0 = không tự ẩn. Mặc định: 4000. */
  duration?: number
}

let notifId = 0

/**
 * Store global quản lý runtime state của app:
 * - Trạng thái sidebar (collapsed/expanded)
 * - Danh sách toast notifications
 * - Trạng thái kết nối với Go backend (connected / mock)
 */
export const useAppStore = defineStore('app', () => {
  // === State ===
  const sidebarCollapsed = ref(localStorage.getItem('sidebar_collapsed') === 'true')
  const notifications = ref<AppNotification[]>([])
  const connectionStatus = ref<'connected' | 'disconnected' | 'mock'>('mock')

  // === Computed ===
  const hasNotifications = computed(() => notifications.value.length > 0)

  // === Actions ===

  /** Đóng/mở sidebar. Trạng thái được persist vào localStorage. */
  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
    localStorage.setItem('sidebar_collapsed', String(sidebarCollapsed.value))
  }

  /**
   * Hiển thị toast notification. Tự động ẩn sau `duration` ms.
   * @param type     - Loại thông báo (success / error / warning / info)
   * @param message  - Nội dung hiển thị
   * @param duration - Thời gian tự ẩn (ms). Mặc định 4000. Truyền 0 để không tự ẩn.
   */
  function notify(type: AppNotification['type'], message: string, duration = 4000) {
    const id = ++notifId
    notifications.value.push({ id, type, message, duration })
    if (duration > 0) {
      setTimeout(() => removeNotification(id), duration)
    }
  }

  /**
   * Xóa notification khỏi danh sách (dùng khi user bấm nút đóng hoặc hết timeout).
   * @param id - ID của notification cần xóa
   */
  function removeNotification(id: number) {
    notifications.value = notifications.value.filter(n => n.id !== id)
  }

  /**
   * Cập nhật trạng thái kết nối hiển thị ở status bar.
   * @param status - 'connected' (Wails live), 'disconnected' (mất kết nối), 'mock' (dev mode)
   */
  function setConnectionStatus(status: typeof connectionStatus.value) {
    connectionStatus.value = status
  }

  return {
    sidebarCollapsed,
    notifications,
    connectionStatus,
    hasNotifications,
    toggleSidebar,
    notify,
    removeNotification,
    setConnectionStatus,
  }
})
