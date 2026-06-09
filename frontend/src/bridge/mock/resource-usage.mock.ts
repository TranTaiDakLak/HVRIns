// resource-usage.mock.ts — Mock bridge cho GetResourceUsage
import type { IResourceUsageService } from '../contracts'

export const resourceUsageMock: IResourceUsageService = {
  async get() {
    return { ramMb: 0, cpuPct: 0 }
  },
}
