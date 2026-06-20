import { setActivePinia, createPinia } from 'pinia'
import { useAppStore } from '@/stores/app.store'

describe('useAppStore', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
  })

  describe('trạng thái khởi tạo', () => {
    it('sidebarCollapsed=false, notifications=[], connectionStatus=mock', () => {
      const store = useAppStore()
      expect(store.sidebarCollapsed).toBe(false)
      expect(store.notifications).toHaveLength(0)
      expect(store.connectionStatus).toBe('mock')
    })

    it('hasNotifications=false khi không có notification', () => {
      const store = useAppStore()
      expect(store.hasNotifications).toBe(false)
    })
  })

  describe('toggleSidebar', () => {
    it('false → true → false', () => {
      const store = useAppStore()
      store.toggleSidebar()
      expect(store.sidebarCollapsed).toBe(true)
      store.toggleSidebar()
      expect(store.sidebarCollapsed).toBe(false)
    })

    it('persist vào localStorage', () => {
      const store = useAppStore()
      store.toggleSidebar()
      expect(localStorage.getItem('sidebar_collapsed')).toBe('true')
      store.toggleSidebar()
      expect(localStorage.getItem('sidebar_collapsed')).toBe('false')
    })
  })

  describe('notify', () => {
    it('thêm notification đúng type và message', () => {
      const store = useAppStore()
      store.notify('success', 'Thao tác thành công')
      expect(store.notifications).toHaveLength(1)
      expect(store.notifications[0].type).toBe('success')
      expect(store.notifications[0].message).toBe('Thao tác thành công')
      expect(store.hasNotifications).toBe(true)
    })

    it('thêm nhiều notification, giữ thứ tự', () => {
      const store = useAppStore()
      store.notify('info', 'A', 0)
      store.notify('error', 'B', 0)
      expect(store.notifications).toHaveLength(2)
      expect(store.notifications[0].message).toBe('A')
      expect(store.notifications[1].message).toBe('B')
    })

    it('duration=0: không tự xoá sau timeout', () => {
      vi.useFakeTimers()
      const store = useAppStore()
      store.notify('warning', 'Sticky', 0)
      vi.runAllTimers()
      expect(store.notifications).toHaveLength(1)
      vi.useRealTimers()
    })

    it('duration>0: tự xoá sau timeout', () => {
      vi.useFakeTimers()
      const store = useAppStore()
      store.notify('info', 'Auto-remove', 1000)
      expect(store.notifications).toHaveLength(1)
      vi.advanceTimersByTime(1001)
      expect(store.notifications).toHaveLength(0)
      vi.useRealTimers()
    })
  })

  describe('removeNotification', () => {
    it('xoá đúng notification theo ID', () => {
      const store = useAppStore()
      store.notify('error', 'A', 0)
      store.notify('error', 'B', 0)
      const idA = store.notifications[0].id
      store.removeNotification(idA)
      expect(store.notifications).toHaveLength(1)
      expect(store.notifications[0].message).toBe('B')
    })

    it('ID không tồn tại: không có lỗi', () => {
      const store = useAppStore()
      store.notify('info', 'X', 0)
      store.removeNotification(99999)
      expect(store.notifications).toHaveLength(1)
    })
  })

  describe('setConnectionStatus', () => {
    it('cập nhật connection status', () => {
      const store = useAppStore()
      store.setConnectionStatus('connected')
      expect(store.connectionStatus).toBe('connected')
      store.setConnectionStatus('disconnected')
      expect(store.connectionStatus).toBe('disconnected')
      store.setConnectionStatus('mock')
      expect(store.connectionStatus).toBe('mock')
    })
  })
})
