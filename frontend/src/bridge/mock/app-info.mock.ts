import type { IAppInfoService } from '@/bridge/contracts'

export const appInfoMock: IAppInfoService = {
  async getVersion() {
    return 'dev'
  },
}
