import { setActivePinia, createPinia } from 'pinia'
import { useUploadLogStore } from '@/stores/uploadLog.store'

describe('useUploadLogStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  describe('trạng thái khởi tạo', () => {
    it('logs=[], stats zeros, isRunning=false', () => {
      const store = useUploadLogStore()
      expect(store.logs).toHaveLength(0)
      expect(store.isRunning).toBe(false)
      expect(store.stats.totalUploaded).toBe(0)
      expect(store.stats.totalFailed).toBe(0)
    })

    it('totalUploaded=0 khi chưa có log hay stats', () => {
      const store = useUploadLogStore()
      expect(store.totalUploaded).toBe(0)
    })
  })

  describe('addLog', () => {
    it('thêm entry với đúng type và message', () => {
      const store = useUploadLogStore()
      store.addLog('Upload OK', 3, 'ok')
      expect(store.logs).toHaveLength(1)
      expect(store.logs[0].msg).toBe('Upload OK')
      expect(store.logs[0].type).toBe('ok')
      expect(store.logs[0].uploaded).toBe(3)
    })

    it('unshift — newest first', () => {
      const store = useUploadLogStore()
      store.addLog('First', 1, 'ok')
      store.addLog('Second', 2, 'err')
      expect(store.logs[0].msg).toBe('Second')
      expect(store.logs[1].msg).toBe('First')
    })

    it('time format HH:MM:SS', () => {
      const store = useUploadLogStore()
      store.addLog('test', 0, 'info')
      expect(store.logs[0].time).toMatch(/^\d{2}:\d{2}:\d{2}$/)
    })

    it('cap tại 200 dòng (MAX_LOG_LINES)', () => {
      const store = useUploadLogStore()
      for (let i = 0; i < 210; i++) {
        store.addLog(`line ${i}`, 0, 'info')
      }
      expect(store.logs).toHaveLength(200)
      // Dòng cuối cùng phải là dòng thứ 10 bị đẩy ra (oldest)
      // unshift → logs[0] = newest; logs[199] = oldest còn lại (line 10)
      expect(store.logs[0].msg).toBe('line 209')
    })

    it('hỗ trợ tất cả type: ok, err, warn, info', () => {
      const store = useUploadLogStore()
      store.addLog('ok', 0, 'ok')
      store.addLog('err', 0, 'err')
      store.addLog('warn', 0, 'warn')
      store.addLog('info', 0, 'info')
      const types = store.logs.map(l => l.type)
      expect(types).toContain('ok')
      expect(types).toContain('err')
      expect(types).toContain('warn')
      expect(types).toContain('info')
    })
  })

  describe('clearLogs', () => {
    it('xoá toàn bộ logs', () => {
      const store = useUploadLogStore()
      store.addLog('A', 1, 'ok')
      store.addLog('B', 2, 'ok')
      store.clearLogs()
      expect(store.logs).toHaveLength(0)
    })

    it('không throw khi window.go không tồn tại (test env)', () => {
      const store = useUploadLogStore()
      store.addLog('test', 0, 'info')
      expect(() => store.clearLogs()).not.toThrow()
    })
  })

  describe('totalUploaded computed', () => {
    it('fallback: sum logs.uploaded khi stats.totalUploaded=0', () => {
      const store = useUploadLogStore()
      store.addLog('A', 5, 'ok')
      store.addLog('B', 3, 'err')
      expect(store.totalUploaded).toBe(8)
    })

    it('ưu tiên stats.totalUploaded khi > 0', () => {
      const store = useUploadLogStore()
      store.addLog('A', 10, 'ok')
      store.stats.totalUploaded = 999
      expect(store.totalUploaded).toBe(999)
    })

    it('totalUploaded=0 khi logs rỗng và stats=0', () => {
      const store = useUploadLogStore()
      expect(store.totalUploaded).toBe(0)
    })
  })

  describe('dispose', () => {
    it('không throw khi chưa init', () => {
      const store = useUploadLogStore()
      expect(() => store.dispose()).not.toThrow()
    })

    it('có thể gọi nhiều lần liên tiếp', () => {
      const store = useUploadLogStore()
      expect(() => {
        store.dispose()
        store.dispose()
      }).not.toThrow()
    })
  })
})
