// interaction.wails.ts — Wails implementation cho IInteractionService
import type { IInteractionService } from '@/services/contracts'
import type { VerifyConfig } from '@/types/interaction.types'
import { SaveInteractionConfig, LoadInteractionConfig } from '../../../wailsjs/go/main/App'

export const interactionWails: IInteractionService = {
  async save(data: VerifyConfig): Promise<string> {
    return await SaveInteractionConfig(data as any)
  },

  async load(): Promise<VerifyConfig> {
    return await LoadInteractionConfig() as unknown as VerifyConfig
  },
}
