// interaction.mock.ts — Mock implementation cho IInteractionService
import type { IInteractionService } from '@/services/contracts'
import type { VerifyConfig } from '@/types/interaction.types'
import { DEFAULT_VERIFY_CONFIG } from '@/types/interaction.types'

const STORAGE_KEY = 'ncs_interaction'

export const interactionMock: IInteractionService = {
  async save(data: VerifyConfig): Promise<string> {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
    return 'OK'
  },

  async load(): Promise<VerifyConfig> {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      try { return JSON.parse(raw) } catch { /* fall through */ }
    }
    return { ...DEFAULT_VERIFY_CONFIG }
  },
}
