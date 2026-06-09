import type { IAppInfoService } from '../contracts'
import { GetAppVersion } from '../../../wailsjs/go/main/App'

export const appInfoWails: IAppInfoService = {
  async getVersion() {
    return GetAppVersion()
  },
}
