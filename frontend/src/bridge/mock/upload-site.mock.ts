import type { IUploadSiteService } from '../contracts'
import type { UploadSiteConfig } from '../../types/upload-site.types'
import { DEFAULT_UPLOAD_SITE_CONFIG } from '../../types/upload-site.types'

let _store: UploadSiteConfig = JSON.parse(JSON.stringify(DEFAULT_UPLOAD_SITE_CONFIG))

export const uploadSiteMock: IUploadSiteService = {
  async save(data: UploadSiteConfig): Promise<string> {
    _store = JSON.parse(JSON.stringify(data))
    return 'OK'
  },
  async load(): Promise<UploadSiteConfig> {
    return JSON.parse(JSON.stringify(_store))
  },
}
