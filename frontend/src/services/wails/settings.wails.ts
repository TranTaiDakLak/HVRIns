// settings.wails.ts — Wails implementation cho ISettingsService
import type { ISettingsService, SettingsData } from '@/services/contracts'
import { SaveSettings, LoadSettings } from '../../../wailsjs/go/app/App'

export const settingsWails: ISettingsService = {
  async save(data: SettingsData): Promise<string> {
    return await SaveSettings(data as any)
  },

  async load(): Promise<SettingsData> {
    return await LoadSettings() as SettingsData
  },
}
