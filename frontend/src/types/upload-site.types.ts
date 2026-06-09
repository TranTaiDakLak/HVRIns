export interface UploadSiteSourceConfig {
  enabled: boolean
}

export interface UploadSiteConfig {
  reg: UploadSiteSourceConfig
  ver: UploadSiteSourceConfig
  code: string              // mã kho hàng (stock code)
  apiKey: string            // API key của admin
  adminUsername: string     // tài khoản đăng nhập banclone.pro
  adminPassword: string     // mật khẩu đăng nhập banclone.pro
  filterDuplicate: boolean  // true = filter=1 (lọc trùng UID), false = filter=0
  delayCheckSec: number
  accPerBatch: number
  delayBetweenBatchSec: number
}

export const DEFAULT_UPLOAD_SITE_CONFIG: UploadSiteConfig = {
  reg: { enabled: false },
  ver: { enabled: false },
  code: '69ea28f9e5e3e',
  apiKey: '6ddcacd6d2b59363401c516292a786aaq2Aa14OynFgKJi5lQY7tcEZhXjIvBPs0',
  adminUsername: '',
  adminPassword: '',
  filterDuplicate: false,
  delayCheckSec: 25,
  accPerBatch: 900,
  delayBetweenBatchSec: 9,
}
