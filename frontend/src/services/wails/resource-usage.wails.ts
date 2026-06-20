// resource-usage.wails.ts — Wails bridge cho GetResourceUsage
import type { IResourceUsageService, AppResourceUsage } from '@/services/contracts'
import { GetResourceUsage } from '../../../wailsjs/go/app/App'

export const resourceUsageWails: IResourceUsageService = {
  async get(): Promise<AppResourceUsage> {
    const r = await GetResourceUsage()
    return { ramMb: r.ramMb, cpuPct: r.cpuPct }
  },
}
