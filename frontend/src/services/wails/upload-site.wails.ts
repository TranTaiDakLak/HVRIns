import type { IUploadSiteService } from '@/services/contracts'
import type { UploadSiteConfig } from '@/types/upload-site.types'
import { SaveUploadSiteConfig, LoadUploadSiteConfig } from '../../../wailsjs/go/app/App'

export const uploadSiteWails: IUploadSiteService = {
  async save(data: UploadSiteConfig): Promise<string> {
    return await SaveUploadSiteConfig(data as any)
  },
  async load(): Promise<UploadSiteConfig> {
    return await LoadUploadSiteConfig() as unknown as UploadSiteConfig
  },
}
