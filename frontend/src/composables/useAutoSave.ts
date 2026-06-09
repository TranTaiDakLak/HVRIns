// useAutoSave — Composable tự động save form khi value thay đổi, debounce.
//
// Mục tiêu UX: user không cần bấm nút "Lưu" — mỗi edit đều persist.
// Settings thay đổi tại runtime sẽ được picked up bởi các worker goroutine đang chạy
// thông qua LoadInteractionConfig realtime ở backend (app.go RunVerify GetVerifyConfig).
//
// Cơ chế:
//   - Watch target ref deep
//   - Khi thay đổi → debounce delay (default 500ms)
//   - Gọi save callback — nếu đang save dở thì queue lại lần cuối
//   - Emit status: "idle" | "saving" | "saved" | "error" cho UI hiển thị nhẹ
//
// KHÔNG auto-save trong `skipFirst` ms đầu (default 300ms) — tránh save value mặc
// định khi form vừa load từ server xong, watch immediate trigger.

import { ref, watch, onUnmounted, type WatchSource } from 'vue'

export type AutoSaveStatus = 'idle' | 'saving' | 'saved' | 'error'

export interface UseAutoSaveOptions {
  /** Delay ms sau thay đổi cuối mới save. Default 500. */
  debounceMs?: number
  /** Bỏ qua save trong ms đầu sau mount để tránh save lúc load form. Default 300. */
  skipFirstMs?: number
  /** true = watch deep (cho object/array). Default true. */
  deep?: boolean
  /** Callback khi save xong. */
  onSaved?: () => void
  /** Callback khi save lỗi — nhận error message. */
  onError?: (err: unknown) => void
}

/**
 * Tự động save form khi value thay đổi.
 *
 * @param source - ref form hoặc getter trả về value cần watch
 * @param save - async function gọi API save
 * @param opts - tùy chọn debounce / skipFirst
 *
 * Trả về object với:
 *   status: ref trạng thái ('idle' | 'saving' | 'saved' | 'error')
 *   saveNow: force save ngay (bỏ debounce)
 *   lastError: ref error nếu có
 */
export function useAutoSave<T>(
  source: WatchSource<T>,
  save: (value: T) => Promise<void>,
  opts: UseAutoSaveOptions = {},
) {
  const debounceMs = opts.debounceMs ?? 500
  const skipFirstMs = opts.skipFirstMs ?? 300
  const deep = opts.deep ?? true

  const status = ref<AutoSaveStatus>('idle')
  const lastError = ref<string | null>(null)

  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  let pending = false
  let saving = false
  let mountedAt = Date.now()

  const flush = async (value: T) => {
    if (saving) {
      pending = true // đánh dấu còn 1 lần save nữa cần chạy sau khi xong
      return
    }
    saving = true
    status.value = 'saving'
    try {
      await save(value)
      status.value = 'saved'
      lastError.value = null
      opts.onSaved?.()
      // Reset về idle sau 1.5s để user thấy indicator nháy rồi biến mất
      setTimeout(() => {
        if (status.value === 'saved') status.value = 'idle'
      }, 1500)
    } catch (err) {
      status.value = 'error'
      lastError.value = err instanceof Error ? err.message : String(err)
      opts.onError?.(err)
    } finally {
      saving = false
      // Nếu có thay đổi trong lúc save → chạy lần cuối
      if (pending) {
        pending = false
        // get latest value by triggering watcher path — lấy từ source
        const v = typeof source === 'function' ? source() : (source as any).value
        await flush(v as T)
      }
    }
  }

  const stop = watch(
    source,
    (value) => {
      // Skip save trong skipFirstMs đầu (tránh trigger khi load form xong).
      if (Date.now() - mountedAt < skipFirstMs) return

      if (debounceTimer) clearTimeout(debounceTimer)
      debounceTimer = setTimeout(() => {
        void flush(value as T)
      }, debounceMs)
    },
    { deep },
  )

  /** Force save ngay, bỏ debounce. Dùng khi cần chắc chắn đã persist (VD trước khi navigate). */
  const saveNow = async () => {
    if (debounceTimer) {
      clearTimeout(debounceTimer)
      debounceTimer = null
    }
    const v = typeof source === 'function' ? source() : (source as any).value
    await flush(v as T)
  }

  /** Đặt lại mốc "mountedAt" để cho phép save bắt đầu (hữu ích sau khi load data xong). */
  const markReady = () => {
    mountedAt = Date.now() - skipFirstMs - 1
  }

  onUnmounted(() => {
    stop()
    // Nếu có debounce timer chưa chạy → force save ngay trước khi unmount.
    // Tránh user đổi setting rồi chuyển page trước 500ms, thay đổi bị mất.
    if (debounceTimer) {
      clearTimeout(debounceTimer)
      debounceTimer = null
      const v = typeof source === 'function' ? source() : (source as any).value
      // Fire-and-forget (component đang unmount, không còn UI để await).
      void flush(v as T)
    }
  })

  return { status, lastError, saveNow, markReady }
}
