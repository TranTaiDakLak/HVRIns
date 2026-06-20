import type { IAppInfoService } from '@/services/contracts'
import { GetAppVersion } from '../../../wailsjs/go/main/App'

export const appInfoWails: IAppInfoService = {
  async getVersion() {
    return GetAppVersion()
  },
}
