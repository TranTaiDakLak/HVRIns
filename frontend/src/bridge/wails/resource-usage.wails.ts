// resource-usage.wails.ts — Wails bridge cho GetResourceUsage
import type { IResourceUsageService, AppResourceUsage } from '../contracts'
import { GetResourceUsage } from '../../../wailsjs/go/main/App'

export const resourceUsageWails: IResourceUsageService = {
  async get(): Promise<AppResourceUsage> {
    const r = await GetResourceUsage()
    return { ramMb: r.ramMb, cpuPct: r.cpuPct }
  },
}
