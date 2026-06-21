import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface UploadLogEntry {
  time: string
  msg: string
  uploaded: number
  type: 'ok' | 'err' | 'warn' | 'info'
}

export interface UploadStats {
  totalUploaded: number
  totalFailed: number
  pendingCount: number
  consecutiveFailures: number
  duplicateSkipped: number
  lastUploadAt: string
  lastErrorAt: string
  lastError: string
  lastRotateAt: string
  startedAt: string
}

const EMPTY_STATS: UploadStats = {
  totalUploaded: 0,
  totalFailed: 0,
  pendingCount: 0,
  consecutiveFailures: 0,
  duplicateSkipped: 0,
  lastUploadAt: '',
  lastErrorAt: '',
  lastError: '',
  lastRotateAt: '',
  startedAt: '',
}

const MAX_LOG_LINES = 200 // UI hard cap (giảm từ 500 → 200 để tiết kiệm DOM khi chạy 12h+)

// Map level từ backend → UI type
function levelToType(level: string | undefined, msg: string): UploadLogEntry['type'] {
  if (level === 'ok') return 'ok'
  if (level === 'error') return 'err'
  if (level === 'warn') return 'warn'
  if (level === 'info') return 'info'
  // fallback dựa icon
  if (msg.startsWith('❌')) return 'err'
  if (msg.startsWith('⚠')) return 'warn'
  if (msg.startsWith('✅')) return 'ok'
  return 'info'
}

export const useUploadLogStore = defineStore('uploadLog', () => {
  const logs = ref<UploadLogEntry[]>([])
  const stats = ref<UploadStats>({ ...EMPTY_STATS })
  const isRunning = ref(false)
  let initialized = false
  let statsTimer: ReturnType<typeof setTimeout> | null = null
  let runningTimer: ReturnType<typeof setTimeout> | null = null
  // unsubscribers — Wails EventsOn trả về () => void; lưu lại để dispose() gọi
  // hủy listener tránh leak khi store bị recreate (UI reload mỗi 3h, hot reload).
  let unsubscribers: Array<() => void> = []

  // Total ưu tiên từ backend stats (tích luỹ); fallback sum logs khi backend chưa load.
  const totalUploaded = computed(() =>
    stats.value.totalUploaded > 0
      ? stats.value.totalUploaded
      : logs.value.reduce((sum, e) => sum + e.uploaded, 0)
  )

  function addLog(msg: string, uploaded: number, type: UploadLogEntry['type']) {
    const now = new Date()
    const hh = String(now.getHours()).padStart(2, '0')
    const mm = String(now.getMinutes()).padStart(2, '0')
    const ss = String(now.getSeconds()).padStart(2, '0')
    logs.value.unshift({ time: `${hh}:${mm}:${ss}`, msg, uploaded, type })
    if (logs.value.length > MAX_LOG_LINES) {
      logs.value.splice(MAX_LOG_LINES)
    }
  }

  function clearLogs() {
    logs.value = []
    const w = window as any
    // Gọi backend để clear file luôn (sync UI ↔ disk)
    if (typeof w?.go?.app?.App?.ClearUploadLog === 'function') {
      try { w.go.app.App.ClearUploadLog() } catch { /* ignore */ }
    }
  }

  async function refreshStats() {
    const w = window as any
    if (typeof w?.go?.app?.App?.GetUploadStats !== 'function') return
    try {
      const s = await w.go.app.App.GetUploadStats()
      if (s) stats.value = { ...EMPTY_STATS, ...s }
    } catch { /* ignore */ }
  }

  // Adaptive polling: lần cuối có upload event → polling nhanh 5s; idle quá lâu → 30s
  let lastActivityAt = 0
  function nextPollInterval(): number {
    const idleMs = Date.now() - lastActivityAt
    if (idleMs < 60_000) return 5_000   // active <1min: poll 5s
    if (idleMs < 600_000) return 15_000 // idle <10min: poll 15s
    return 30_000                       // idle lâu: poll 30s
  }
  function scheduleNextPoll() {
    if (statsTimer) clearTimeout(statsTimer)
    statsTimer = setTimeout(async () => {
      await refreshStats()
      scheduleNextPoll()
    }, nextPollInterval())
  }

  function init() {
    if (initialized) return
    initialized = true
    const w = window as any

    // Load stats lần đầu + start adaptive poll
    lastActivityAt = Date.now()
    refreshStats()
    scheduleNextPoll()

    if (!w?.runtime?.EventsOn) return

    const u1 = w.runtime.EventsOn('upload-site:log', (data: { msg: string; uploaded: number; level?: string }) => {
      addLog(data.msg, data.uploaded ?? 0, levelToType(data.level, data.msg))
      lastActivityAt = Date.now()
      isRunning.value = true
      if (runningTimer) clearTimeout(runningTimer)
      runningTimer = setTimeout(() => { isRunning.value = false }, 10_000)
      if ((data.uploaded ?? 0) > 0) refreshStats()
    })

    const u2 = w.runtime.EventsOn('upload-site:stopped', () => {
      addLog('— Đã dừng goroutine upload —', 0, 'info')
      isRunning.value = false
      if (runningTimer) clearTimeout(runningTimer)
      refreshStats()
    })

    // Backend rotate 2h → clear UI log để khớp với file
    const u3 = w.runtime.EventsOn('upload-site:log-cleared', () => {
      logs.value = []
      addLog('🧹 Log tự động dọn (mỗi 2h)', 0, 'info')
      refreshStats()
    })

    // Wails EventsOn trả về unsubscribe function — lưu để dispose() hủy.
    if (typeof u1 === 'function') unsubscribers.push(u1)
    if (typeof u2 === 'function') unsubscribers.push(u2)
    if (typeof u3 === 'function') unsubscribers.push(u3)
  }

  function dispose() {
    if (statsTimer) { clearTimeout(statsTimer); statsTimer = null }
    if (runningTimer) { clearTimeout(runningTimer); runningTimer = null }
    for (const fn of unsubscribers) {
      try { fn() } catch { /* ignore */ }
    }
    unsubscribers = []
    initialized = false
  }

  return { logs, stats, isRunning, totalUploaded, addLog, clearLogs, refreshStats, init, dispose }
})
