// event-bus.mock.ts — Mock event bus (no-op; events never fire in mock mode)
import type { IEventBusService } from '@/bridge/contracts'

export const eventBusMock: IEventBusService = {
  // Task 6: contract trả về unsub fn — mock trả no-op fn để caller cleanup logic
  // không bị nil/undefined kiểm tra.
  on(_event: string, _callback: (...args: any[]) => void): () => void {
    return () => { /* no-op unsub */ }
  },
  off(..._events: string[]): void { /* no-op */ },
}
