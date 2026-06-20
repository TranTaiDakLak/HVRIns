import type { IAppInfoService } from '@/services/contracts'

export const appInfoMock: IAppInfoService = {
  async getVersion() {
    return 'dev'
  },
}
