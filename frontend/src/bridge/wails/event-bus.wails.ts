// event-bus.wails.ts — Wails bridge cho EventsOn / EventsOff (Wails runtime events)
import type { IEventBusService } from '@/bridge/contracts'
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime'

export const eventBusWails: IEventBusService = {
  // Task 6: trả về unsub fn từ Wails EventsOn để caller cleanup chính xác
  // (selective per-listener) thay vì gọi off(eventName) global xóa nhầm listener khác.
  on(event: string, callback: (...args: any[]) => void): () => void {
    const unsub = EventsOn(event, callback)
    if (typeof unsub === 'function') return unsub
    // Fallback nếu Wails phiên bản cũ không trả unsub: cleanup theo name (best-effort).
    return () => EventsOff(event)
  },
  off(...events: string[]): void {
    if (events.length > 0) EventsOff(events[0], ...events.slice(1))
  },
}
