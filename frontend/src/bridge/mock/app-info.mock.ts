import type { IAppInfoService } from '../contracts'

export const appInfoMock: IAppInfoService = {
  async getVersion() {
    return 'dev'
  },
}
