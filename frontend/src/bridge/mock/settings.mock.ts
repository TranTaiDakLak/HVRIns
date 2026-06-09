// settings.mock.ts — Mock implementation cho ISettingsService
import type { ISettingsService, SettingsData } from '../contracts'
import { DEFAULT_GENERAL_CONFIG, DEFAULT_IP_CONFIG } from '../../types/settings.types'

const STORAGE_KEY = 'ncs_settings'

export const settingsMock: ISettingsService = {
  async save(data: SettingsData): Promise<string> {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
    return 'OK'
  },

  async load(): Promise<SettingsData> {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      try {
        return JSON.parse(raw)
      } catch { /* fall through */ }
    }
    return {
      general: { ...DEFAULT_GENERAL_CONFIG, captchaKeys: { ...DEFAULT_GENERAL_CONFIG.captchaKeys } },
      ip: { ...DEFAULT_IP_CONFIG },
    }
  },
}
